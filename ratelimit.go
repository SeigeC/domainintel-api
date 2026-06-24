package main

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client         *redis.Client
	requestsPerMin int
	burst          int
}

func NewRateLimiter(url string, perMin, burst int) *RateLimiter {
	if url == "" {
		return nil
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil
	}
	return &RateLimiter{
		client:         redis.NewClient(opts),
		requestsPerMin: perMin,
		burst:          burst,
	}
}

// Allow uses a fixed-window counter per apiKey per minute (INCR + EXPIRE).
func (r *RateLimiter) Allow(apiKey string) (bool, error) {
	if r == nil {
		return true, nil
	}
	now := time.Now().UTC()
	key := "rl:" + apiKey + ":" + now.Format("200601021504")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		// fail-open: don't block traffic on cache failure
		return true, nil
	}
	if count == 1 {
		r.client.Expire(ctx, key, 60*time.Second)
	}
	return count <= int64(r.requestsPerMin), nil
}

func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientKey(r)
			allowed, err := limiter.Allow(key)
			if err != nil || !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":429,"message":"rate limit exceeded","type":"rate_limited"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientKey(r *http.Request) string {
	if k := r.Header.Get("X-RapidAPI-Proxy-Secret"); k != "" {
		return "proxy:" + k
	}
	if k := r.Header.Get("X-API-Key"); k != "" {
		return "key:" + k
	}
	return "ip:" + clientIP(r)
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		for i := 0; i < len(ip); i++ {
			if ip[i] == ',' {
				return ip[:i]
			}
		}
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host := r.RemoteAddr
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}
