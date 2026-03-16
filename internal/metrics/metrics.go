package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Metric for TTFT
	TTFTSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "flux_ttft_seconds",
			Help:    "Time To First Token in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5},
		},
		[]string{"model"},
	)

	// Metric for 429 Errors
	Error429Total = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flux_error_429_total",
			Help: "Total count of rate limited requests",
		},
		[]string{"user_id"},
	)

	// Metric for Token Usage
	TokenUsage = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flux_token_usage_total",
			Help: "Total tokens consumed",
		},
		[]string{"model", "user_id"},
	)
)

func init() {
	prometheus.MustRegister(TTFTSeconds, Error429Total, TokenUsage)
}
