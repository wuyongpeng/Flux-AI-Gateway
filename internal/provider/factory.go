package provider

import (
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
	case "glm":
		return NewGLMProvider(modelID, apiKey, baseURL), nil
	case "anthropic":
		// Placeholder — implement when needed
		return nil, fmt.Errorf("anthropic provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
}
