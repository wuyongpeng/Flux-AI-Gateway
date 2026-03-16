package scheduler

import (
	"context"
	"log"
	"net/http"
)

// FailoverResponse executes requests sequentially.
// If a backend returns a non-200 status (like 429), it immediately tries the next one.
func FailoverResponse(ctx context.Context, backends []BackendProvider, reqBody []byte) Response {
	var lastRes Response

	for _, b := range backends {
		// Individual request context to allow independent cancellation if needed,
		// though in sequential failover we usually just use the parent context.
		resp := b.SendRequest(ctx, reqBody)

		if resp.Err == nil && resp.StatusCode == http.StatusOK {
			resp.WinnerName = b.Name()
			return resp
		}

		log.Printf("[Failover] Backend '%s' failed (Status: %d, Err: %v). Trying next...", b.Name(), resp.StatusCode, resp.Err)
		lastRes = resp

		// If the parent context is cancelled, stop trying
		select {
		case <-ctx.Done():
			return Response{Err: ctx.Err()}
		default:
		}
	}

	return lastRes
}
