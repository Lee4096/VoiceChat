package middleware

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)
	ip := "192.168.1.1"

	if !limiter.Allow(ip) {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Error("Second request should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Error("Third request should be allowed")
	}

	if limiter.Allow(ip) {
		t.Error("Fourth request should be denied")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	if !limiter.Allow("192.168.1.1") {
		t.Error("First IP request should be allowed")
	}
	if !limiter.Allow("192.168.1.1") {
		t.Error("First IP second request should be allowed")
	}
	if limiter.Allow("192.168.1.1") {
		t.Error("First IP third request should be denied")
	}

	if !limiter.Allow("192.168.1.2") {
		t.Error("Second IP first request should be allowed")
	}
	if !limiter.Allow("192.168.1.2") {
		t.Error("Second IP second request should be allowed")
	}
	if limiter.Allow("192.168.1.2") {
		t.Error("Second IP third request should be denied")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	limiter := NewRateLimiter(1, 100*time.Millisecond)
	ip := "192.168.1.1"

	if !limiter.Allow(ip) {
		t.Error("First request should be allowed")
	}
	if limiter.Allow(ip) {
		t.Error("Second request should be denied")
	}

	time.Sleep(150 * time.Millisecond)

	if !limiter.Allow(ip) {
		t.Error("Request after window expiry should be allowed")
	}
}
