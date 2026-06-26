package internal

import (
	"math/rand"
	"time"
)

// ChaosConfig injects artificial latency and errors so apps can test how they
// behave against a slow or flaky Upstash without touching the cloud.
type ChaosConfig struct {
	// Latency is added to every request before it is handled.
	Latency time.Duration
	// ErrorRate is the probability (0.0–1.0) that a request fails with a
	// simulated upstream error.
	ErrorRate float64
}

// Enabled reports whether any chaos behaviour is configured.
func (c *ChaosConfig) Enabled() bool {
	return c != nil && (c.Latency > 0 || c.ErrorRate > 0)
}

func (c *ChaosConfig) applyLatency() {
	if c == nil || c.Latency <= 0 {
		return
	}
	time.Sleep(c.Latency)
}

func (c *ChaosConfig) maybeError() (string, bool) {
	if c == nil || c.ErrorRate <= 0 {
		return "", false
	}
	if rand.Float64() < c.ErrorRate {
		return "ERR simulated upstream failure (chaos injection)", true
	}
	return "", false
}
