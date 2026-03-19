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
