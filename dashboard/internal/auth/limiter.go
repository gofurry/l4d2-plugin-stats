package auth

import (
	"sync"
	"time"
)

type Limit struct {
	Count int
	Reset time.Time
}
type Limiter struct {
	mu       sync.Mutex
	entries  map[string]Limit
	max      int
	window   time.Duration
	capacity int
}

func NewLimiter(max int, window time.Duration, capacity int) *Limiter {
	return &Limiter{entries: make(map[string]Limit), max: max, window: window, capacity: capacity}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	entry, ok := l.entries[key]
	if !ok || now.After(entry.Reset) {
		entry = Limit{Reset: now.Add(l.window)}
	}
	if entry.Count >= l.max {
		l.entries[key] = entry
		return false
	}
	entry.Count++
	l.entries[key] = entry
	if len(l.entries) > l.capacity {
		for k, v := range l.entries {
			if now.After(v.Reset) {
				delete(l.entries, k)
			}
		}
		if len(l.entries) > l.capacity {
			for k := range l.entries {
				delete(l.entries, k)
				break
			}
		}
	}
	return true
}
