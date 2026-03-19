// Package controller implements Kubernetes-style controllers that
// watch external sources and reconcile state into storage.
package controller

import (
	"context"
	"log"
	"time"

	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage"
)

// GitRepoController periodically polls GitHub for repositories
// and writes them to storage.
type GitRepoController struct {
	githubClient *github.Client
	store        storage.GitRepoStore
	username     string
	interval     time.Duration
}

// NewGitRepoController creates a new controller that syncs repos for the given username.
func NewGitRepoController(client *github.Client, store storage.GitRepoStore, username string, interval time.Duration) *GitRepoController {
	return &GitRepoController{
		githubClient: client,
		store:        store,
		username:     username,
		interval:     interval,
	}
}

// Run starts the controller loop. It blocks until ctx is cancelled.
// It performs an initial sync immediately, then polls at the configured interval.
func (c *GitRepoController) Run(ctx context.Context) {
	log.Printf("GitRepoController: starting sync for username %q every %v", c.username, c.interval)

	// Sync immediately on startup.
	c.sync(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("GitRepoController: stopping")
			return
		case <-ticker.C:
			c.sync(ctx)
		}
	}
}

func (c *GitRepoController) sync(ctx context.Context) {
	log.Printf("GitRepoController: syncing repositories for %s", c.username)

	repos, err := c.githubClient.ListRepositories(ctx, c.username)
	if err != nil {
		log.Printf("GitRepoController: error fetching repositories: %v", err)
		return
	}

	if err := c.store.ReplaceAll(ctx, c.username, repos); err != nil {
		log.Printf("GitRepoController: error storing repositories: %v", err)
		return
	}

	log.Printf("GitRepoController: synced %d repositories for %s", len(repos), c.username)
}
