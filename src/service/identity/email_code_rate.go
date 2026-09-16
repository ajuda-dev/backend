package identity

import (
	"sync"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
)

type emailCodeRateLimiter struct {
	mu       sync.Mutex
	cfg      EmailCodeConfig
	sends    map[string][]time.Time
	attempts map[string][]time.Time
}

func newEmailCodeRateLimiter(cfg EmailCodeConfig) *emailCodeRateLimiter {
	return &emailCodeRateLimiter{
		cfg:      cfg,
		sends:    make(map[string][]time.Time),
		attempts: make(map[string][]time.Time),
	}
}

func (r *emailCodeRateLimiter) AllowSend(key string, now time.Time) *rest_err.RestErr {
	return r.allowSend(key, now, "too many password reset requests")
}

func (r *emailCodeRateLimiter) AllowVerificationSend(key string, now time.Time) *rest_err.RestErr {
	return r.allowSend(key, now, "too many verification emails")
}

func (r *emailCodeRateLimiter) allowSend(key string, now time.Time, tooManyMessage string) *rest_err.RestErr {
	r.mu.Lock()
	defer r.mu.Unlock()

	hourAgo := now.Add(-time.Hour)
	interval := time.Duration(r.cfg.ResendIntervalSeconds) * time.Second
	sends := pruneTimes(r.sends[key], hourAgo)

	if len(sends) > 0 {
		last := sends[len(sends)-1]
		if now.Sub(last) < interval {
			r.sends[key] = sends
			return rest_err.NewTooManyRequestsError(tooManyMessage)
		}
	}
	if len(sends) >= r.cfg.MaxSendsPerHour {
		r.sends[key] = sends
		return rest_err.NewTooManyRequestsError(tooManyMessage)
	}

	r.sends[key] = append(sends, now)
	return nil
}

func (r *emailCodeRateLimiter) AllowAttempt(key string, now time.Time) *rest_err.RestErr {
	return r.allowAttempt(key, now, "too many password reset attempts")
}

func (r *emailCodeRateLimiter) AllowVerificationAttempt(key string, now time.Time) *rest_err.RestErr {
	return r.allowAttempt(key, now, "too many verification attempts")
}

func (r *emailCodeRateLimiter) allowAttempt(key string, now time.Time, tooManyMessage string) *rest_err.RestErr {
	r.mu.Lock()
	defer r.mu.Unlock()

	window := time.Duration(r.cfg.TTLMinutes) * time.Minute
	cutoff := now.Add(-window)
	attempts := pruneTimes(r.attempts[key], cutoff)
	r.attempts[key] = attempts
	if len(attempts) >= r.cfg.MaxAttempts {
		return rest_err.NewTooManyRequestsError(tooManyMessage)
	}
	return nil
}

func (r *emailCodeRateLimiter) RecordFailedAttempt(key string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	window := time.Duration(r.cfg.TTLMinutes) * time.Minute
	cutoff := now.Add(-window)
	attempts := pruneTimes(r.attempts[key], cutoff)
	r.attempts[key] = append(attempts, now)
}

func (r *emailCodeRateLimiter) ClearAttempts(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.attempts, key)
}

func pruneTimes(times []time.Time, cutoff time.Time) []time.Time {
	if len(times) == 0 {
		return times
	}
	i := 0
	for i < len(times) && times[i].Before(cutoff) {
		i++
	}
	if i == 0 {
		return times
	}
	return append([]time.Time(nil), times[i:]...)
}
