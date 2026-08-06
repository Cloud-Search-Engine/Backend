package cache

import (
	"context"
	"time"
)

// Cache is a minimal interface for query/result caching (Redis in production).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// Memory is an in-process cache for local development.
type Memory struct {
	data map[string]entry
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemory constructs an empty memory cache.
func NewMemory() *Memory {
	return &Memory{data: map[string]entry{}}
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, bool, error) {
	e, ok := m.data[key]
	if !ok {
		return nil, false, nil
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		delete(m.data, key)
		return nil, false, nil
	}
	return e.value, true, nil
}

func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.data[key] = entry{value: value, expiresAt: exp}
	return nil
}
