// Package storage provides a generic resource store following Kubernetes patterns.
package storage

import (
	"context"
	"fmt"
	"sync"
)

// CRUDStoreIface is the interface implemented by both CRUDStore and PersistentCRUDStore.
type CRUDStoreIface[T any] interface {
	List(ctx context.Context) ([]T, error)
	Get(ctx context.Context, name string) (T, bool, error)
	Create(ctx context.Context, item T) error
	Update(ctx context.Context, item T) error
	Delete(ctx context.Context, name string) error
}

// CRUDStore is a generic, thread-safe store for user-created resources.
// Unlike ResourceStore (designed for controller bulk-sync), CRUDStore supports
// individual Create, Update, and Delete operations.
type CRUDStore[T any] struct {
	mu       sync.RWMutex
	items    map[string]T
	nameFunc func(T) string
}

// NewCRUDStore creates a new empty CRUDStore.
// The nameFunc extracts the unique name (key) from a resource.
func NewCRUDStore[T any](nameFunc func(T) string) *CRUDStore[T] {
	return &CRUDStore[T]{
		items:    make(map[string]T),
		nameFunc: nameFunc,
	}
}

// List returns all items in the store.
func (s *CRUDStore[T]) List(ctx context.Context) ([]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]T, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}

// Get returns a single item by name.
func (s *CRUDStore[T]) Get(ctx context.Context, name string) (T, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[name]
	return item, ok, nil
}

// Create adds a new item. Returns an error if an item with the same name already exists.
func (s *CRUDStore[T]) Create(ctx context.Context, item T) error {
	name := s.nameFunc(item)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[name]; exists {
		return fmt.Errorf("resource %q already exists", name)
	}
	s.items[name] = item
	return nil
}

// Update replaces an existing item. Returns an error if the item does not exist.
func (s *CRUDStore[T]) Update(ctx context.Context, item T) error {
	name := s.nameFunc(item)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[name]; !exists {
		return fmt.Errorf("resource %q not found", name)
	}
	s.items[name] = item
	return nil
}

// Delete removes an item by name. Returns an error if the item does not exist.
func (s *CRUDStore[T]) Delete(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[name]; !exists {
		return fmt.Errorf("resource %q not found", name)
	}
	delete(s.items, name)
	return nil
}
