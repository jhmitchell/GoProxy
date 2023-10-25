package rproxy

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

var rateLimiters = make(map[string]*rate.Limiter)
var mtx sync.Mutex

// GetRateLimiter retrieves a rate limiter for a given IP address, creating one if it doesn't exist.
func GetRateLimiter(ip string) *rate.Limiter {
	mtx.Lock()
	defer mtx.Unlock()

	limiter, exists := rateLimiters[ip]
	if !exists {
		// Allow 5 requests per second with a burst of 10
		limiter = rate.NewLimiter(5, 10)
		rateLimiters[ip] = limiter
	}
	return limiter
}

// rateLimiterMiddleware checks if the incoming request exceeds the rate limit.
// If it does, the middleware responds with a "Too Many Requests" status and stops further processing.
func RateLimiterMiddleware(p *proxy) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Extract the IP address from the request
		ip := req.RemoteAddr

		// Get the base IP addr, compatible with IPv6
		if index := strings.LastIndex(ip, ":"); index != -1 {
			ip = ip[:index]
		}

		limiter := GetRateLimiter(ip)
		if !limiter.Allow() {
			p.log.Warn("Rate limit exceeded",
				zap.String("Client IP", ip),
				zap.String("Method", req.Method),
				zap.String("URL", req.URL.String()),
				zap.String("User-Agent", req.UserAgent()),
				zap.Time("Timestamp", time.Now()))

			http.Error(rw, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		p.ReverseProxy.ServeHTTP(rw, req)
	})
}
