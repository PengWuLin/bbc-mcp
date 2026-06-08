package ratelimit

import (
	"errors"
	"sync"
)

var (
	ErrRateLimitExceeded = errors.New("服务繁忙，请稍后重试")
)

type RateLimiter struct {
	mu sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

func (r *RateLimiter) TryAcquire() bool {
	return r.mu.TryLock()
}

func (r *RateLimiter) Release() {
	r.mu.Unlock()
}

func (r *RateLimiter) Process(f func()) error {
	if !r.TryAcquire() {
		return ErrRateLimitExceeded
	}
	defer r.Release()
	f()
	return nil
}