// Package storage defines the interfaces for resource storage,
// following the Kubernetes pattern of controllers writing to storage
// and API servers reading from it.
package storage

import (
	"context"

	"github.com/justinsb/gitctl/internal/api"
)

// GitRepoStore provides read and write access to GitRepo resources.
type GitRepoStore interface {
	// List returns all GitRepo resources for the given username.
	List(ctx context.Context, username string) ([]api.GitRepo, error)

	// ReplaceAll atomically replaces all GitRepo resources for a username.
	// This is used by controllers to sync the full state from an external source.
	ReplaceAll(ctx context.Context, username string, repos []api.GitRepo) error
}

// PullRequestStore provides read and write access to PullRequest resources.
// The key parameter is a scoped identifier like "outbound:username" or "assigned:username".
type PullRequestStore interface {
	// ListPullRequests returns all PullRequest resources for the given key.
	ListPullRequests(ctx context.Context, key string) ([]api.PullRequest, error)

	// ReplaceAllPullRequests atomically replaces all PullRequest resources for a key.
	ReplaceAllPullRequests(ctx context.Context, key string, prs []api.PullRequest) error
}

// IssueStore provides read and write access to Issue resources.
// The key parameter is a scoped identifier like "assigned:username".
type IssueStore interface {
	// ListIssues returns all Issue resources for the given key.
	ListIssues(ctx context.Context, key string) ([]api.Issue, error)

	// ReplaceAllIssues atomically replaces all Issue resources for a key.
	ReplaceAllIssues(ctx context.Context, key string, issues []api.Issue) error
}

// CommentStore provides read and write access to Comment resources.
// The key parameter is "repo#number" (e.g. "owner/repo#123").
type CommentStore interface {
	// ListComments returns cached comments for the given key, or nil if not cached.
	ListComments(ctx context.Context, key string) ([]api.Comment, bool, error)

	// ReplaceAllComments atomically replaces all comments for a key.
	ReplaceAllComments(ctx context.Context, key string, comments []api.Comment) error
}
