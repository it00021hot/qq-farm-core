package memcache

import (
	"path"
	"strings"
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time // zero means no expiry
}

// Store is a process-local KV with optional TTL.
type Store struct {
	mu   sync.RWMutex
	data map[string]entry
}

func New() *Store {
	s := &Store{data: make(map[string]entry)}
	go s.gcLoop()
	return s
}

func (s *Store) gcLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, e := range s.data {
			if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
				delete(s.data, k)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := entry{value: value}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return "", false
	}
	return e.value, true
}

func (s *Store) Del(keys ...string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for _, k := range keys {
		if _, ok := s.data[k]; ok {
			delete(s.data, k)
			n++
		}
	}
	return n
}

// Keys returns keys matching a redis-style pattern (supports trailing * and prefix*suffix via path.Match).
func (s *Store) Keys(pattern string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	out := make([]string, 0)
	for k, e := range s.data {
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			continue
		}
		matched, _ := path.Match(pattern, k)
		if matched || matchGlob(pattern, k) {
			out = append(out, k)
		}
	}
	return out
}

func matchGlob(pattern, s string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(s, prefix)
	}
	return pattern == s
}

// Default global store used by app cache layer.
var Default = New()
