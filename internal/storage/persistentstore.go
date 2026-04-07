// Package storage provides a generic resource store following Kubernetes patterns.
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// persistedData is the on-disk format for a FileStorage.
type persistedData[T any] struct {
	Items []T `json:"items"`
}

// FileStorage wraps CRUDStore with JSON file persistence.
// All items are serialized to a single JSON file after each mutation.
// On creation, the file is loaded if it exists.
type FileStorage[T any] struct {
	inner    *CRUDStore[T]
	filePath string
}

// NewFileStorage creates a FileStorage backed by filePath.
// If filePath exists, items are loaded from it. The directory must exist.
func NewFileStorage[T any](nameFunc func(T) string, filePath string) (*FileStorage[T], error) {
	inner := NewCRUDStore[T](nameFunc)
	s := &FileStorage[T]{inner: inner, filePath: filePath}
	if err := s.load(); err != nil {
		return nil, fmt.Errorf("loading %s: %w", filePath, err)
	}
	return s, nil
}

// load reads items from the JSON file, if it exists.
func (s *FileStorage[T]) load() error {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return nil // nothing persisted yet
	}
	if err != nil {
		return err
	}
	var pd persistedData[T]
	if err := json.Unmarshal(data, &pd); err != nil {
		return err
	}
	for _, item := range pd.Items {
		if err := s.inner.Create(context.Background(), item); err != nil {
			// Log and skip duplicates (shouldn't happen with a clean file).
			log.Printf("FileStorage: skipping duplicate item on load: %v", err)
		}
	}
	return nil
}

// save writes all current items to the JSON file atomically.
func (s *FileStorage[T]) save(ctx context.Context) error {
	items, err := s.inner.List(ctx)
	if err != nil {
		return err
	}
	pd := persistedData[T]{Items: items}
	data, err := json.MarshalIndent(pd, "", "  ")
	if err != nil {
		return err
	}
	// Write to a temp file in the same directory then rename for atomicity.
	dir := filepath.Dir(s.filePath)
	tmp, err := os.CreateTemp(dir, ".gitctl-views-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.filePath)
}

// List returns all items in the store.
func (s *FileStorage[T]) List(ctx context.Context) ([]T, error) {
	return s.inner.List(ctx)
}

// Get returns a single item by name.
func (s *FileStorage[T]) Get(ctx context.Context, name string) (T, bool, error) {
	return s.inner.Get(ctx, name)
}

// Create adds a new item and persists to disk.
func (s *FileStorage[T]) Create(ctx context.Context, item T) error {
	if err := s.inner.Create(ctx, item); err != nil {
		return err
	}
	if err := s.save(ctx); err != nil {
		log.Printf("FileStorage: failed to persist after create: %v", err)
	}
	return nil
}

// Update replaces an existing item and persists to disk.
func (s *FileStorage[T]) Update(ctx context.Context, item T) error {
	if err := s.inner.Update(ctx, item); err != nil {
		return err
	}
	if err := s.save(ctx); err != nil {
		log.Printf("FileStorage: failed to persist after update: %v", err)
	}
	return nil
}

// Delete removes an item and persists to disk.
func (s *FileStorage[T]) Delete(ctx context.Context, name string) error {
	if err := s.inner.Delete(ctx, name); err != nil {
		return err
	}
	if err := s.save(ctx); err != nil {
		log.Printf("FileStorage: failed to persist after delete: %v", err)
	}
	return nil
}
