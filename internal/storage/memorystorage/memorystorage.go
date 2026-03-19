// Package memorystorage provides an in-memory implementation of the storage interfaces.
package memorystorage

import (
	"context"
	"sync"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/storage"
)

// ensure MemoryStore implements GitRepoStore.
var _ storage.GitRepoStore = &MemoryStore{}

// MemoryStore is a thread-safe in-memory store for GitRepo resources.
type MemoryStore struct {
	mu    sync.RWMutex
	repos map[string][]api.GitRepo // keyed by username
}

// New creates a new MemoryStore.
func New() *MemoryStore {
	return &MemoryStore{
		repos: make(map[string][]api.GitRepo),
	}
}

// List returns all GitRepo resources for the given username.
func (s *MemoryStore) List(ctx context.Context, username string) ([]api.GitRepo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repos := s.repos[username]
	if repos == nil {
		return nil, nil
	}

	// Return a copy to avoid data races.
	out := make([]api.GitRepo, len(repos))
	copy(out, repos)
	return out, nil
}

// ReplaceAll atomically replaces all GitRepo resources for a username.
func (s *MemoryStore) ReplaceAll(ctx context.Context, username string, repos []api.GitRepo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy.
	stored := make([]api.GitRepo, len(repos))
	copy(stored, repos)
	s.repos[username] = stored
	return nil
}
