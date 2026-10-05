package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type clientBucket struct {
	tokens     float64
	lastUpdate time.Time
}

// RateLimiter implements a token-bucket rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64 // bucket capacity
	clients map[string]*clientBucket
	ticker  *time.Ticker
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(rps, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:    float64(rps),
		burst:   float64(burst),
		clients: make(map[string]*clientBucket),
		ticker:  time.NewTicker(5 * time.Minute),
	}

	// Clean up stale clients every 5 minutes
	go func() {
		for range rl.ticker.C {
			rl.cleanup(10 * time.Minute)
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for ip, b := range rl.clients {
		if b.lastUpdate.Before(cutoff) {
			delete(rl.clients, ip)
		}
	}
}

// Stop stops the cleanup timer.
func (rl *RateLimiter) Stop() {
	if rl.ticker != nil {
		rl.ticker.Stop()
	}
}

// Limit returns an HTTP middleware enforcing rate limits.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)

		rl.mu.Lock()
		b, exists := rl.clients[ip]
		now := time.Now()

		if !exists {
			rl.clients[ip] = &clientBucket{
				tokens:     rl.burst - 1,
				lastUpdate: now,
			}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		// Replenish tokens based on elapsed time
		elapsed := now.Sub(b.lastUpdate).Seconds()
		b.tokens += elapsed * rl.rate
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastUpdate = now

		if b.tokens < 1.0 {
			rl.mu.Unlock()
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Rate limit exceeded. Please slow down."}}`))
			return
		}

		b.tokens -= 1.0
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	// If behind Cloudflare or reverse proxy
	cfIP := r.Header.Get("CF-Connecting-IP")
	if cfIP != "" {
		return cfIP
	}
	xForwarded := r.Header.Get("X-Forwarded-For")
	if xForwarded != "" {
		parts := strings.Split(xForwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
