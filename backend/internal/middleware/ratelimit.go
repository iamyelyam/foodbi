package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter is a simple sliding-window per-key limiter.
// Each key gets its own bucket of timestamps; on each request we drop
// timestamps older than `window` and reject if the bucket is full.
//
// Memory-bounded by periodic GC (sweeps expired entries every window).
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string][]time.Time
	limit    int
	window   time.Duration
	stopGC   chan struct{}
}

// NewRateLimiter creates a limiter that allows `limit` requests per `window`
// per unique key.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
		stopGC:  make(chan struct{}),
	}
	go rl.gcLoop()
	return rl
}

// Allow returns true if the caller is under the limit, false otherwise.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)
	bucket := rl.buckets[key]

	// Drop expired timestamps
	filtered := bucket[:0]
	for _, t := range bucket {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= rl.limit {
		rl.buckets[key] = filtered
		return false
	}

	filtered = append(filtered, now)
	rl.buckets[key] = filtered
	return true
}

// Close stops the GC goroutine.
func (rl *RateLimiter) Close() { close(rl.stopGC) }

func (rl *RateLimiter) gcLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stopGC:
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.window)
			for k, bucket := range rl.buckets {
				keep := bucket[:0]
				for _, t := range bucket {
					if t.After(cutoff) {
						keep = append(keep, t)
					}
				}
				if len(keep) == 0 {
					delete(rl.buckets, k)
				} else {
					rl.buckets[k] = keep
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Middleware returns an HTTP middleware that rate-limits by client IP.
// On rejection, responds with 429 Too Many Requests.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.Allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "too many requests, please slow down"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the client IP. Respects X-Forwarded-For (Railway sets this)
// and falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For: client, proxy1, proxy2 → take first (real client)
		if idx := strings.IndexByte(xff, ','); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
