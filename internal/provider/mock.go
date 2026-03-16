package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	"flux-ai-gateway/internal/scheduler"
)

// MockBackend simulates an upstream API with a specific latency and token speed.
// In a real implementation this would adapt Google Gemini or OpenAI HTTP protocols.
type MockBackend struct {
	BackendName  string
	InitialDelay time.Duration
	TokenDelay   time.Duration
}

func (m *MockBackend) Name() string {
	return m.BackendName
}

func (m *MockBackend) SendRequest(ctx context.Context, body []byte) scheduler.Response {
	// Simulate the network call
	select {
	case <-time.After(m.InitialDelay):
	case <-ctx.Done():
		return scheduler.Response{Err: ctx.Err()}
	}

	// We return a mock stream that sends `data: token` periodically
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()
		tokens := []string{"data: Hello\n\n", "data: world\n\n", "data: this\n\n", "data: is\n\n", "data: flux\n\n", "data: gateway.\n\n"}
		for _, token := range tokens {
			select {
			case <-time.After(m.TokenDelay):
				writer.Write([]byte(token))
			case <-ctx.Done():
				return
			}
		}
	}()

	headers := make(http.Header)
	headers.Set("Content-Type", "text/event-stream")

	return scheduler.Response{
		Body:       reader,
		StatusCode: http.StatusOK,
		Header:     headers,
		Err:        nil,
	}
}

// Ensure MockBackend implements BackendProvider
var _ scheduler.BackendProvider = (*MockBackend)(nil)
