package provider

import (
	"context"
	"fmt"

	"flux-ai-gateway/internal/scheduler"
)

// NewProvider is a factory that instantiates the correct BackendProvider
// based on the provider ID read from the model registry.
func NewProvider(modelID, providerID, apiKey, baseURL string) (scheduler.BackendProvider, error) {
	if apiKey == "" {
		return &MockBackend{BackendName: modelID}, nil
	}

	switch providerID {
	case "google":
		return NewGeminiProvider(modelID, apiKey), nil
	case "openai":
		return NewOpenAIProvider(modelID, apiKey, baseURL), nil
	case "anthropic":
		// Placeholder — implement when needed
		return nil, fmt.Errorf("anthropic provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
}

// --- OpenAI Provider (SSE-compatible) ----------------------------------------

type OpenAIProvider struct {
	ModelName string
	APIKey    string
	BaseURL   string
}

func NewOpenAIProvider(model, apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{ModelName: model, APIKey: apiKey, BaseURL: baseURL}
}

func (o *OpenAIProvider) Name() string { return o.ModelName }

func (o *OpenAIProvider) SendRequest(ctx context.Context, body []byte) scheduler.Response {
	// OpenAI uses the same SSE streaming format
	// POST /v1/chat/completions with stream: true
	// The body forwarded from the client will need to be reformatted from Gemini to OpenAI format
	// For now this is a placeholder — implement actual OpenAI transport when adding full support
	return scheduler.Response{
		Err: fmt.Errorf("OpenAI provider stub — full implementation pending (model: %s)", o.ModelName),
	}
}
