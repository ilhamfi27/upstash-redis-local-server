package internal

import (
	"sync"
	"sync/atomic"
	"time"
)

const freeTierDailyQuota int64 = 10000

// Metrics tracks request usage for the local dashboard.
type Metrics struct {
	totalRequests atomic.Int64
	startedAt     time.Time
	byCommand     map[string]int64
	mu            sync.RWMutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		startedAt: time.Now(),
		byCommand: make(map[string]int64),
	}
}

func (m *Metrics) Record(command string) {
	m.totalRequests.Add(1)
	name := command
	if name == "" {
		name = "unknown"
	}
	m.mu.Lock()
	m.byCommand[name]++
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	commands := make(map[string]int64, len(m.byCommand))
	for k, v := range m.byCommand {
		commands[k] = v
	}

	total := m.totalRequests.Load()
	remaining := freeTierDailyQuota - total
	if remaining < 0 {
		remaining = 0
	}

	return map[string]interface{}{
		"total_requests":       total,
		"uptime_seconds":       int64(time.Since(m.startedAt).Seconds()),
		"commands":             commands,
		"free_tier_quota":      freeTierDailyQuota,
		"quota_saved":          total,
		"cloud_quota_remaining": remaining,
		"unlimited_local":      true,
	}
}
