package httpapi

import (
	"testing"
	"time"
)

func TestFixedWindowLimiter(t *testing.T) {
	limiter := newFixedWindowLimiter(2, 25*time.Millisecond)
	key := "127.0.0.1"

	if !limiter.Allow(key) {
		t.Fatalf("expected first request to be allowed")
	}
	if !limiter.Allow(key) {
		t.Fatalf("expected second request to be allowed")
	}
	if limiter.Allow(key) {
		t.Fatalf("expected third request to be blocked in the same window")
	}

	time.Sleep(30 * time.Millisecond)

	if !limiter.Allow(key) {
		t.Fatalf("expected request to be allowed after window reset")
	}
}
