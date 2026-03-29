// Package storage provides a generic resource store following Kubernetes patterns.
package storage

import (
	"context"
	"sync"
)

// ResourceStore is a generic, thread-safe store for any resource type.
// Each instance handles one resource kind, keyed by a string identifier.
// This mirrors how Kubernetes API server storage works per-GVR.
type ResourceStore[T any] struct {
	mu    sync.RWMutex
	items map[string][]T
}

// NewResourceStore creates a new empty ResourceStore.
func NewResourceStore[T any]() *ResourceStore[T] {
	return &ResourceStore[T]{
		items: make(map[string][]T),
	}
}

// List returns all items for the given key.
// The found return value indicates whether the key exists in the store,
// allowing callers to distinguish "not cached" from "cached but empty".
func (s *ResourceStore[T]) List(ctx context.Context, key string) ([]T, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.items[key]
	if !ok {
		return nil, false, nil
	}

	out := make([]T, len(items))
	copy(out, items)
	return out, true, nil
}

// ReplaceAll atomically replaces all items for a key.
// This is used by controllers to sync the full state from an external source.
func (s *ResourceStore[T]) ReplaceAll(ctx context.Context, key string, items []T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]T, len(items))
	copy(stored, items)
	s.items[key] = stored
	return nil
}
