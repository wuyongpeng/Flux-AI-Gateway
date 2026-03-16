package proxy

import (
	"context"
	"io"
	"log"
	"time"
)

// StreamMonitor wraps an io.ReadCloser (the upstream response body)
// to track TTFT and ITL.
type StreamMonitor struct {
	OriginalReader io.ReadCloser
	StartTime      time.Time
	LastTokenTime  time.Time
	FirstTokenRead bool

	// ITLThreshold defines the max allowed time between tokens
	ITLThreshold time.Duration
	// HandleStall is a callback invoked when a stall is detected
	HandleStall func(context.Context, []byte) (io.ReadCloser, error)

	// Buffer to keep track of generated context for failover
	generatedContext []byte
}

// Read implements io.Reader, intercepting standard reads to monitor latency
func (sm *StreamMonitor) Read(p []byte) (n int, err error) {
	// First check if a stall occurred since the last chunk
	if sm.FirstTokenRead && time.Since(sm.LastTokenTime) > sm.ITLThreshold {
		// A stall happened before reading the next chunk.
		// Trigger failover.
		// TODO: Implement the switch
	}

	n, err = sm.OriginalReader.Read(p)
	if n > 0 {
		now := time.Now()
		// Simple heuristic for checking if there's actual data
		// A real implementation might scan for "data:"
		if !sm.FirstTokenRead {
			sm.FirstTokenRead = true
			sm.LastTokenTime = now

			ttft := time.Since(sm.StartTime)
			log.Printf("[Metrics] TTFT (Time To First Token): %v\n", ttft)

			// Record TTFT Metric here:
			// metrics.RecordTTFT(ttft)
		} else {
			// Record ITL
			// itl := now.Sub(sm.LastTokenTime)
			sm.LastTokenTime = now
		}

		// Save generated context to pass to failover model if needed
		sm.generatedContext = append(sm.generatedContext, p[:n]...)
	}

	return n, err
}

// Close closes the underlying reader
func (sm *StreamMonitor) Close() error {
	if sm.OriginalReader != nil {
		return sm.OriginalReader.Close()
	}
	return nil
}
