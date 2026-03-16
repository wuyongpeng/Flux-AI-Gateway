package main

import (
	"flux-ai-gateway/internal/provider"
	"fmt"
)

func main() {
	geminiBody := []byte(`{"contents": [{"parts": [{"text": "Hello, how are you? Explain quantum physics in 33 words."}]}]}`)

	converted, err := provider.MaybeConvertGeminiToOpenAI(geminiBody, "glm-4-flash")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Converted: %s\n", string(converted))
}
