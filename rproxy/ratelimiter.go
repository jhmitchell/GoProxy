package rproxy

import (
	"sync"

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
		// Example: Allow 5 requests per second with a burst of 10
		limiter = rate.NewLimiter(5, 10)
		rateLimiters[ip] = limiter
	}
	return limiter
}
