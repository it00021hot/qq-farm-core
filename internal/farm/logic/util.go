package logic

import (
	"sync"
	"time"
)

var (
	serverTimeMu    sync.RWMutex
	serverTimeMs    int64
	localTimeAtSync int64
)

func unixMilli() int64 {
	return time.Now().UnixMilli()
}

// SyncServerTime stores server wall-clock in milliseconds.
func SyncServerTime(ms int64) {
	serverTimeMu.Lock()
	serverTimeMs = ms
	localTimeAtSync = unixMilli()
	serverTimeMu.Unlock()
}

// GetServerTimeSec returns extrapolated server time in seconds.
func GetServerTimeSec() int64 {
	serverTimeMu.RLock()
	st, local := serverTimeMs, localTimeAtSync
	serverTimeMu.RUnlock()
	if st == 0 {
		return unixMilli() / 1000
	}
	elapsed := unixMilli() - local
	return (st + elapsed) / 1000
}

// ToNum coerces numeric-ish values to int64.
func ToNum(v any) int64 {
	switch n := v.(type) {
	case nil:
		return 0
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case uint64:
		return int64(n)
	default:
		return 0
	}
}

// ToTimeSec normalizes a timestamp to seconds (>1e12 treated as ms).
func ToTimeSec(v int64) int64 {
	if v <= 0 {
		return 0
	}
	if v > 1e12 {
		return v / 1000
	}
	return v
}

// MatureInSec returns seconds until the mature phase begins.
// Matches bot getLandsDetail: matureBegin > nowSec ? matureBegin - nowSec : 0
func MatureInSec(matureBeginSec, nowSec int64) int64 {
	if matureBeginSec > nowSec {
		return matureBeginSec - nowSec
	}
	return 0
}

// InQuietHours checks quiet window (HH:MM strings; supports midnight wrap).
// When enabled and start==end, treat as all-day quiet (bot visit-strategy.inFriendQuietHours).
func InQuietHours(enabled bool, start, end, nowHHMM string) bool {
	if !enabled || start == "" || end == "" {
		return false
	}
	if start == end {
		return true
	}
	if start < end {
		return nowHHMM >= start && nowHHMM < end
	}
	return nowHHMM >= start || nowHHMM < end
}
