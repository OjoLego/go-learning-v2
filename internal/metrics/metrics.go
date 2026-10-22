package metrics

import (
	"sync"
)

type Metrics struct {
	mu      sync.RWMutex
	counter map[string]int64
}

func New() *Metrics {
	return &Metrics{counter: make(map[string]int64)}
}

func (m *Metrics) Inc(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter[key]++
}

func (m *Metrics) Get(key string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counter[key]
}

func (m *Metrics) Snapshot() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]int64, len(m.counter))
	for k, v := range m.counter {
		out[k] = v
	}
	return out
}
