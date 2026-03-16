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
// Slower or trailing requests are automatically cancelled to save tokens.
func FastResponse(ctx context.Context, backends []BackendProvider, reqBody []byte) Response {
	// Create a Cancellable context that will kill losers
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resChan := make(chan Response, len(backends))

	for _, backend := range backends {
		go func(b BackendProvider) {
			// Sending the request to the backend
			// We clone the body since we're using it in multiple goroutines
			bCopy := make([]byte, len(reqBody))
			copy(bCopy, reqBody)

			resp := b.SendRequest(context.WithoutCancel(ctx), bCopy)

			// Simple check: if not error and OK status, we push it to chan
			if resp.Err == nil && resp.StatusCode == http.StatusOK {
				// We peek the first byte to assure TTFT is there
				buf := make([]byte, 1)
				n, err := resp.Body.Read(buf)
				if err == nil && n > 0 {
					// Prepend the peeked byte back
					combinedReader := io.MultiReader(bytes.NewReader(buf), resp.Body)
					resp.Body = struct {
						io.Reader
						io.Closer
					}{
						Reader: combinedReader,
						Closer: resp.Body,
					}
					select {
					case resChan <- resp:
					case <-raceCtx.Done():
						// We lost the race, or request was cancelled
						if resp.Body != nil {
							resp.Body.Close()
						}
					}
					return
				}
			}

			// In case of error, or not status OK, we still push to unblock
			select {
			case resChan <- resp:
			case <-raceCtx.Done():
				if resp.Body != nil {
					resp.Body.Close()
				}
			}

		}(backend)
	}

	// Wait for the first *successful* response, or collect all errors
	var lastRes Response
	errorsCount := 0

	for range backends {
		res := <-resChan
		if res.Err == nil && res.StatusCode == http.StatusOK {
			// We found a winner. Returning it triggers defers, which cancel other goroutines.
			return res
		}
		lastRes = res
		errorsCount++
		if errorsCount == len(backends) {
			return lastRes // all failed
		}
	}

	return lastRes
}
