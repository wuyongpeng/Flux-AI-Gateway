package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"flux-ai-gateway/internal/arbiter"
	"flux-ai-gateway/internal/limiter"
	"flux-ai-gateway/internal/middleware"
	"flux-ai-gateway/internal/proxy"
	"flux-ai-gateway/internal/registry"
)

func main() {
	r := gin.Default()

	// 1. Initialize Registry, Limiter & Arbiter
	reg, err := registry.NewModelRegistry("configs/models.yaml")
	if err != nil {
		log.Fatalf("Failed to initialize model registry: %v", err)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rl := limiter.NewRateLimiter(redisAddr)
	policyArbiter := arbiter.NewPolicyArbiter(reg)

	// 2. Setup Routes

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/v1")
	{
		// Apply Auth and Rate Limiting to all API requests
		v1.Use(middleware.AuthAndLimitMiddleware(rl))

		// The HTTP request acts as our AI entry point taking an OpenAI/Gemini shaped JSON body
		v1.POST("/chat/completions", proxy.HandleGatewayRequest(policyArbiter))
	}

	// 3. Start Server
	log.Println("Starting Flux-AI-Gateway on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
