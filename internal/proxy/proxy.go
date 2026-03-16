package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"time"

	"flux-ai-gateway/internal/arbiter"
	"flux-ai-gateway/internal/scheduler"

	"github.com/gin-gonic/gin"
)

// HandleGatewayRequest is the main HTTP handler that acts as the Proxy.
// It reads the user request, triggers the Hedged Scheduler, and pipes the result back
// to the user via SSE, tracking TTFT and ITL.
func HandleGatewayRequest(arbiter *arbiter.PolicyArbiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record the true request start time at the very beginning.
		// This is what will be used for real user-perceived TTFT.
		requestStart := time.Now()

		// Read abstract scenario from header, default to "speed_racing"
		scenario := c.GetHeader("X-Flux-Scenario")
		if scenario == "" {
			scenario = "speed_racing"
		}

		backends, fallbackBackend, strategy, err := arbiter.GetBackends(scenario)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// 立即打印请求到达提示
		log.Printf("[Gateway] ✅ 请求已到达 Gateway — scenario: %s", scenario)
		if strategy == "hedged" {
			log.Printf("[Gateway] 🏁 正在选择最优模型 (hedged race: %d backends)...", len(backends))
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "bad request"})
			return
		}
		c.Request.Body.Close()

		// 1. Race / Hedged Requests
		bestResponse := scheduler.FastResponse(c.Request.Context(), backends, body)
		if bestResponse.Err != nil {
			c.JSON(500, gin.H{"error": bestResponse.Err.Error()})
			return
		}

		// 2. Wrap body with StreamMonitor
		sm := &StreamMonitor{
			OriginalReader: bestResponse.Body,
			StartTime:      requestStart, // true user-perceived TTFT from request arrival
			ITLThreshold:   2 * time.Second,
			// Handling Stall / Breakpoint failover
			HandleStall: func(ctx context.Context, generated []byte) (io.ReadCloser, error) {
				log.Println("Stall detected (ITL > 2s). Executing Breakpoint Failover...")
				// We inject the previously generated tokens into the context
				// and fire a new request to the fallback provider.
				newReqBody := append(body, generated...)
				res := fallbackBackend.SendRequest(ctx, newReqBody)
				return res.Body, res.Err
			},
		}
		defer sm.Close()

		// Copy headers to response
		for k, v := range bestResponse.Header {
			if k != "Content-Length" {
				c.Writer.Header()[k] = v
			}
		}
		c.Writer.WriteHeader(bestResponse.StatusCode)

		// 3. Pipe the Stream Monitor to the Gin Response

		buf := make([]byte, 4096)
		ttftRecorded := false

		// Regex to parse the usageMetadata from Gemini response
		tokenRegex := regexp.MustCompile(`"totalTokenCount":\s*(\d+)`)
		var lastTokenCount string

		for {
			n, err := sm.Read(buf)
			if n > 0 {
				chunk := buf[:n]

				if !ttftRecorded {
					ttft := time.Since(sm.StartTime)
					winnerName := bestResponse.WinnerName
					ttftMsg := fmt.Sprintf("data: [Metrics] TTFT: %v (winner: %s)\n\n", ttft, winnerName)
					c.Writer.Write([]byte(ttftMsg))
					log.Printf("[Gateway] 🏆 最优模型: %s | TTFT: %v", winnerName, ttft)
					ttftRecorded = true
				}

				// Look for token count in this chunk
				matches := tokenRegex.FindSubmatch(chunk)
				if len(matches) > 1 {
					lastTokenCount = string(matches[1])
				}

				_, writeErr := c.Writer.Write(chunk)
				if writeErr != nil {
					log.Printf("Write err: %v\n", writeErr)
					break
				}
				c.Writer.Flush()
			}
			if err != nil {
				if err != io.EOF && err != context.Canceled {
					log.Printf("Read err: %v\n", err)
				}
				break
			}

			if sm.FirstTokenRead && time.Since(sm.LastTokenTime) > sm.ITLThreshold {
				// Try fetching fallback stream
				newReader, stErr := sm.HandleStall(c.Request.Context(), sm.generatedContext)
				if stErr == nil && newReader != nil {
					sm.OriginalReader.Close()
					sm.OriginalReader = newReader // Swap the reader!
					sm.LastTokenTime = time.Now() // reset timer so we don't immediately stall again
				}
			}
		}

		totalTime := time.Since(sm.StartTime)
		c.Writer.Write([]byte(fmt.Sprintf("\ndata: [Metrics] Total Answer Generation Latency: %v\n\n", totalTime)))
		if lastTokenCount != "" {
			c.Writer.Write([]byte(fmt.Sprintf("data: [Metrics] Total Token Usage: %s\n\n", lastTokenCount)))
		}
		c.Writer.Flush()
	}
}
