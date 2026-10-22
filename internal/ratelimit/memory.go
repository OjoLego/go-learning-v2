package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

type MemoryLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

func NewMemoryLimiter(rps rate.Limit, burst int) *MemoryLimiter {
	return &MemoryLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
	}
}

func (m *MemoryLimiter) Allow(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	lim, ok := m.limiters[key]
	if !ok {
		lim = rate.NewLimiter(m.rps, m.burst)
		m.limiters[key] = lim
	}
	return lim.Allow()
}