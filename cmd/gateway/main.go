package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"flux-ai-gateway/internal/arbiter"
	"flux-ai-gateway/internal/proxy"
	"flux-ai-gateway/internal/registry"
)

func main() {
	r := gin.Default()

	// 1. Initialize Registry & Arbiter
	reg, err := registry.NewModelRegistry("configs/models.yaml")
	if err != nil {
		log.Fatalf("Failed to initialize model registry: %v", err)
	}

	policyArbiter := arbiter.NewPolicyArbiter(reg)

	// 2. Setup Route
	// The HTTP request acts as our AI entry point taking an OpenAI/Gemini shaped JSON body
	r.POST("/v1/chat/completions", proxy.HandleGatewayRequest(policyArbiter))

	// 3. Start Server
	log.Println("Starting Flux-AI-Gateway on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
