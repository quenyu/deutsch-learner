package httpapi

import (
	"sync"
	"time"
)

type fixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]fixedWindowEntry
}

type fixedWindowEntry struct {
	windowStart time.Time
	count       int
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]fixedWindowEntry),
	}
}

func (l *fixedWindowLimiter) Allow(key string) bool {
	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.evictExpired(now)

	entry, exists := l.entries[key]
	if !exists || now.Sub(entry.windowStart) >= l.window {
		l.entries[key] = fixedWindowEntry{
			windowStart: now,
			count:       1,
		}
		return true
	}

	if entry.count >= l.limit {
		return false
	}

	entry.count++
	l.entries[key] = entry
	return true
}

func (l *fixedWindowLimiter) Window() time.Duration {
	return l.window
}

func (l *fixedWindowLimiter) evictExpired(now time.Time) {
	threshold := now.Add(-2 * l.window)
	for key, entry := range l.entries {
		if entry.windowStart.Before(threshold) {
			delete(l.entries, key)
		}
	}
}
