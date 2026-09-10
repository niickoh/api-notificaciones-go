package services

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// FixedWindowLimiter implementa rate limit en memoria por ventana fija.
type FixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateLimitEntry
}

// NewFixedWindowLimiter crea un rate limiter en memoria.
func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	if limit <= 0 {
		limit = int(^uint(0) >> 1)
	}

	return &FixedWindowLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]rateLimitEntry),
	}
}

// Allow indica si el identificador puede seguir consumiendo requests.
func (l *FixedWindowLimiter) Allow(key string) bool {
	if key == "" {
		key = "anonymous"
	}

	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStart) >= l.window {
		l.entries[key] = rateLimitEntry{count: 1, windowStart: now}
		return true
	}

	if entry.count >= l.limit {
		return false
	}

	entry.count++
	l.entries[key] = entry
	return true
}
