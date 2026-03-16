package provider

import (
	"encoding/json"
	"strings"
)

// MaybeConvertGeminiToOpenAI transforms Gemini-formatted input to OpenAI-formatted input.
// This allows the same client request to work across different backends.
func MaybeConvertGeminiToOpenAI(body []byte, modelName string) ([]byte, error) {
	var geminiReq struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"contents"`
	}

	if err := json.Unmarshal(body, &geminiReq); err != nil || len(geminiReq.Contents) == 0 {
		// Possibly already OpenAI format or invalid. Return as is.
		return body, nil
	}

	// Transform to OpenAI format with robust merging of consecutive roles.
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var messages []message

	for _, c := range geminiReq.Contents {
		role := strings.ToLower(c.Role)
		switch role {
		case "model", "assistant":
			role = "assistant"
		case "system":
			role = "system"
		case "user", "":
			role = "user"
		default:
			role = "user"
		}

		// Merge parts into a single content string
		var contentParts []string
		for _, p := range c.Parts {
			if p.Text != "" {
				contentParts = append(contentParts, p.Text)
			}
		}
		content := strings.Join(contentParts, " ")

		if content == "" {
			continue // Zhipu AI and others reject empty messages
		}

		// Merge with previous message if the role is the same.
		// Zhipu AI (GLM) requires alternating roles (user/assistant).
		if len(messages) > 0 && messages[len(messages)-1].Role == role {
			messages[len(messages)-1].Content += "\n" + content
		} else {
			messages = append(messages, message{Role: role, Content: content})
		}
	}

	if len(messages) == 0 {
		return body, nil // Fallback to original if conversion resulted in nothing
	}

	openaiReq := map[string]interface{}{
		"model":    modelName,
		"messages": messages,
		"stream":   true,
	}

	return json.Marshal(openaiReq)
}
