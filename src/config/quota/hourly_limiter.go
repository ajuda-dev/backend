package quota

import (
	"sync"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
)

type HourlyLimiter struct {
	mu      sync.Mutex
	buckets map[string]map[string][]time.Time
}

func NewHourlyLimiter() *HourlyLimiter {
	return &HourlyLimiter{
		buckets: make(map[string]map[string][]time.Time),
	}
}

func (l *HourlyLimiter) Check(bucket, key string, limit int, now time.Time, message string) *rest_err.RestErr {
	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.pruned(bucket, key, now)
	if len(times) >= limit {
		l.set(bucket, key, times)
		return rest_err.NewTooManyRequestsError(message)
	}
	l.set(bucket, key, times)
	return nil
}

func (l *HourlyLimiter) Record(bucket, key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.pruned(bucket, key, now)
	l.set(bucket, key, append(times, now))
}

func (l *HourlyLimiter) pruned(bucket, key string, now time.Time) []time.Time {
	hourAgo := now.Add(-time.Hour)
	byKey := l.buckets[bucket]
	if byKey == nil {
		return nil
	}
	return pruneTimes(byKey[key], hourAgo)
}

func (l *HourlyLimiter) set(bucket, key string, times []time.Time) {
	byKey := l.buckets[bucket]
	if byKey == nil {
		byKey = make(map[string][]time.Time)
		l.buckets[bucket] = byKey
	}
	if len(times) == 0 {
		delete(byKey, key)
		return
	}
	byKey[key] = times
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
