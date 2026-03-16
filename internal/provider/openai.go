package provider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"flux-ai-gateway/internal/scheduler"
)

// OpenAIProvider implements BackendProvider for OpenAI API
type OpenAIProvider struct {
	ModelName string
	APIKey    string
	BaseURL   string
	Client    *http.Client
}

func NewOpenAIProvider(model, apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		ModelName: model,
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Client:    &http.Client{},
	}
}

func (o *OpenAIProvider) Name() string { return o.ModelName }

func (o *OpenAIProvider) SendRequest(ctx context.Context, body []byte) scheduler.Response {
	convertedBody, err := MaybeConvertGeminiToOpenAI(body, o.ModelName)
	if err != nil {
		return scheduler.Response{Err: fmt.Errorf("failed to convert body for OpenAI: %w", err)}
	}

	url := fmt.Sprintf("%s/chat/completions", o.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(convertedBody))
	if err != nil {
		return scheduler.Response{Err: err}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.Client.Do(req)
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
