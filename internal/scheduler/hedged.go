package scheduler

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

// Response struct holds the upstream response and any error
type Response struct {
	Body       io.ReadCloser
	StatusCode int
	Header     http.Header
	Err        error
	WinnerName string // name of the winning backend provider
}

// BackendProvider represents an upstream API (e.g. Gemini, OpenAI)
// that we can send our hedged requests to.
type BackendProvider interface {
	Name() string
	SendRequest(ctx context.Context, body []byte) Response
}

// FastResponse executes a Hedged Request across multiple backends.
// It fires requests to all backends concurrently and returns the first stream
// that successfully starts returning data (the "winner").
// Losers are cancelled immediately by calling their individual context cancel funcs.
func FastResponse(ctx context.Context, backends []BackendProvider, reqBody []byte) Response {
	type entry struct {
		resp Response
		idx  int // which backend index sent this
	}

	n := len(backends)

	// One cancel func per goroutine. Stored before goroutines launch so the
	// main loop can cancel individual losers by index.
	cancelFns := make([]context.CancelFunc, n)
	resChan := make(chan entry, n)

	for i, backend := range backends {
		goroutineCtx, cancelI := context.WithCancel(ctx)
		cancelFns[i] = cancelI

		go func(b BackendProvider, myCtx context.Context, myIdx int) {
			bCopy := make([]byte, len(reqBody))
			copy(bCopy, reqBody)

			// HTTP call uses this goroutine's own context.
			// When we call cancelFns[myIdx]() from the main loop, the HTTP
			// connection is aborted immediately.
			resp := b.SendRequest(myCtx, bCopy)

			if resp.Err == nil && resp.StatusCode == http.StatusOK {
				// Peek one byte to confirm data is actually flowing (true TTFT check).
				buf := make([]byte, 1)
				nRead, err := resp.Body.Read(buf)
				if err == nil && nRead > 0 {
					combined := io.MultiReader(bytes.NewReader(buf), resp.Body)
					resp.Body = struct {
						io.Reader
						io.Closer
					}{Reader: combined, Closer: resp.Body}

					resChan <- entry{resp: resp, idx: myIdx}
					return
				}
			}

			// Error / non-200 / no data — send so the main loop is unblocked.
			resChan <- entry{resp: resp, idx: myIdx}
		}(backend, goroutineCtx, i)
	}

	// Collect results. Cancel every loser individually.
	// This means Pro's ongoing HTTP request is immediately killed by the OS
	// the moment we call cancelFns[proIdx]().
	var lastRes Response
	errorsCount := 0

	for range backends {
		e := <-resChan

		if e.resp.Err == nil && e.resp.StatusCode == http.StatusOK {
			// Winner found. Cancel ALL other goroutines' HTTP connections NOW.
			for i, cancel := range cancelFns {
				if i != e.idx {
					cancel() // aborts the loser's HTTP connection immediately
				}
			}
			// Tag the winning model name onto the response
			e.resp.WinnerName = backends[e.idx].Name()
			return e.resp
		}

		// This candidate failed — cancel its context and mark it done.
		cancelFns[e.idx]()
		lastRes = e.resp
		errorsCount++
		if errorsCount == n {
			return lastRes // all failed
		}
	}

	return lastRes
}
