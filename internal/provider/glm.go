package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"flux-ai-gateway/internal/scheduler"
)

// GLMProvider implements the BackendProvider for Zhipu AI (GLM)
// It uses an OpenAI-compatible interface.
type GLMProvider struct {
	ModelName string
	APIKey    string
	BaseURL   string
	Client    *http.Client
}

func NewGLMProvider(model, apiKey, baseURL string) *GLMProvider {
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	return &GLMProvider{
		ModelName: model,
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Client:    &http.Client{},
	}
}

func (g *GLMProvider) Name() string { return g.ModelName }

func (g *GLMProvider) SendRequest(ctx context.Context, body []byte) scheduler.Response {
	// 1. Check if the body is in Gemini format. If so, convert it to OpenAI format.
	convertedBody, err := MaybeConvertGeminiToOpenAI(body, g.ModelName)
	if err != nil {
		return scheduler.Response{Err: fmt.Errorf("failed to convert body for GLM: %w", err)}
	}

	// Debug log
	fmt.Printf("[GLM] Sending Request Body: %s\n", string(convertedBody))

	url := fmt.Sprintf("%s/chat/completions", g.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(convertedBody))
	if err != nil {
		return scheduler.Response{Err: err}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+g.APIKey)

	resp, err := g.Client.Do(req)
	if err != nil {
		return scheduler.Response{Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("[GLM] Error Response (%d): %s\n", resp.StatusCode, string(bodyBytes))
		// Re-create reader for the response
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	return scheduler.Response{
		Body:       resp.Body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Err:        nil,
	}
}
