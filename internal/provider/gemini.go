package provider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"flux-ai-gateway/internal/scheduler"
)

// GeminiProvider implements BackendProvider for the official Gemini REST API
type GeminiProvider struct {
	ModelName string
	APIKey    string
	Client    *http.Client
}

func NewGeminiProvider(modelName, apiKey string) *GeminiProvider {
	return &GeminiProvider{
		ModelName: modelName,
		APIKey:    apiKey,
		Client:    &http.Client{},
	}
}

func (g *GeminiProvider) Name() string {
	return g.ModelName
}

func (g *GeminiProvider) SendRequest(ctx context.Context, body []byte) scheduler.Response {
	// Using Gemini REST API with Server-Sent Events (SSE) stream
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", g.ModelName, g.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return scheduler.Response{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return scheduler.Response{Err: err}
	}

	return scheduler.Response{
		Body:       resp.Body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Err:        nil,
	}
}
