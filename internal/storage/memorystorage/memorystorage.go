// Package memorystorage provides an in-memory implementation of the storage interfaces.
package memorystorage

import (
	"context"
	"sync"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/storage"
)

// ensure MemoryStore implements all store interfaces.
var _ storage.GitRepoStore = &MemoryStore{}
var _ storage.PullRequestStore = &MemoryStore{}
var _ storage.IssueStore = &MemoryStore{}
var _ storage.CommentStore = &MemoryStore{}

// MemoryStore is a thread-safe in-memory store for all resource types.
type MemoryStore struct {
	mu       sync.RWMutex
	repos    map[string][]api.GitRepo      // keyed by username
	prs      map[string][]api.PullRequest  // keyed by scope (e.g. "outbound:user")
	issues   map[string][]api.Issue        // keyed by scope (e.g. "assigned:user")
	comments map[string][]api.Comment      // keyed by "repo#number"
}

// New creates a new MemoryStore.
func New() *MemoryStore {
	return &MemoryStore{
		repos:    make(map[string][]api.GitRepo),
		prs:      make(map[string][]api.PullRequest),
		issues:   make(map[string][]api.Issue),
		comments: make(map[string][]api.Comment),
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

	out := make([]api.GitRepo, len(repos))
	copy(out, repos)
	return out, nil
}

// ReplaceAll atomically replaces all GitRepo resources for a username.
func (s *MemoryStore) ReplaceAll(ctx context.Context, username string, repos []api.GitRepo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]api.GitRepo, len(repos))
	copy(stored, repos)
	s.repos[username] = stored
	return nil
}

// ListPullRequests returns all PullRequest resources for the given key.
func (s *MemoryStore) ListPullRequests(ctx context.Context, key string) ([]api.PullRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prs := s.prs[key]
	if prs == nil {
		return nil, nil
	}

	out := make([]api.PullRequest, len(prs))
	copy(out, prs)
	return out, nil
}

// ReplaceAllPullRequests atomically replaces all PullRequest resources for a key.
func (s *MemoryStore) ReplaceAllPullRequests(ctx context.Context, key string, prs []api.PullRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]api.PullRequest, len(prs))
	copy(stored, prs)
	s.prs[key] = stored
	return nil
}

// ListIssues returns all Issue resources for the given key.
func (s *MemoryStore) ListIssues(ctx context.Context, key string) ([]api.Issue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	issues := s.issues[key]
	if issues == nil {
		return nil, nil
	}

	out := make([]api.Issue, len(issues))
	copy(out, issues)
	return out, nil
}

// ReplaceAllIssues atomically replaces all Issue resources for a key.
func (s *MemoryStore) ReplaceAllIssues(ctx context.Context, key string, issues []api.Issue) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]api.Issue, len(issues))
	copy(stored, issues)
	s.issues[key] = stored
	return nil
}

// ListComments returns cached comments for the given key, or false if not cached.
func (s *MemoryStore) ListComments(ctx context.Context, key string) ([]api.Comment, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	comments, ok := s.comments[key]
	if !ok {
		return nil, false, nil
	}

	out := make([]api.Comment, len(comments))
	copy(out, comments)
	return out, true, nil
}

// ReplaceAllComments atomically replaces all comments for a key.
func (s *MemoryStore) ReplaceAllComments(ctx context.Context, key string, comments []api.Comment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]api.Comment, len(comments))
	copy(stored, comments)
	s.comments[key] = stored
	return nil
}
