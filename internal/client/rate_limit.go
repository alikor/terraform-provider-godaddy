package client

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type endpointRateLimiter struct {
	rpm      int
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func newEndpointRateLimiter(rpm int) *endpointRateLimiter {
	if rpm < 1 {
		rpm = 1
	}
	if rpm > 60 {
		rpm = 60
	}

	return &endpointRateLimiter{
		rpm:      rpm,
		limiters: make(map[string]*rate.Limiter),
	}
}

func (l *endpointRateLimiter) Wait(ctx context.Context, template string) error {
	l.mu.Lock()
	limiter, ok := l.limiters[template]
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(float64(l.rpm)/60.0), 1)
		l.limiters[template] = limiter
	}
	l.mu.Unlock()

	return limiter.Wait(ctx)
}
