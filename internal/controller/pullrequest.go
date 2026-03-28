package controller

import (
	"context"
	"log"
	"time"

	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage"
)

// PullRequestController periodically polls GitHub for pull requests
// and writes them to storage.
type PullRequestController struct {
	githubClient *github.Client
	store        storage.PullRequestStore
	username     string
	interval     time.Duration
}

// NewPullRequestController creates a new controller that syncs PRs for the given username.
func NewPullRequestController(client *github.Client, store storage.PullRequestStore, username string, interval time.Duration) *PullRequestController {
	return &PullRequestController{
		githubClient: client,
		store:        store,
		username:     username,
		interval:     interval,
	}
}

// Run starts the controller loop. It blocks until ctx is cancelled.
func (c *PullRequestController) Run(ctx context.Context) {
	log.Printf("PullRequestController: starting sync for username %q every %v", c.username, c.interval)

	c.sync(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("PullRequestController: stopping")
			return
		case <-ticker.C:
			c.sync(ctx)
		}
	}
}

func (c *PullRequestController) sync(ctx context.Context) {
	log.Printf("PullRequestController: syncing PRs for %s", c.username)

	// Sync outbound PRs (authored by user).
	outbound, err := c.githubClient.SearchPullRequestsByAuthor(ctx, c.username)
	if err != nil {
		log.Printf("PullRequestController: error fetching outbound PRs: %v", err)
	} else {
		if err := c.store.ReplaceAllPullRequests(ctx, "outbound:"+c.username, outbound); err != nil {
			log.Printf("PullRequestController: error storing outbound PRs: %v", err)
		} else {
			log.Printf("PullRequestController: synced %d outbound PRs for %s", len(outbound), c.username)
		}
	}

	// Sync assigned PRs.
	assigned, err := c.githubClient.SearchAssignedPullRequests(ctx, c.username)
	if err != nil {
		log.Printf("PullRequestController: error fetching assigned PRs: %v", err)
	} else {
		if err := c.store.ReplaceAllPullRequests(ctx, "assigned:"+c.username, assigned); err != nil {
			log.Printf("PullRequestController: error storing assigned PRs: %v", err)
		} else {
			log.Printf("PullRequestController: synced %d assigned PRs for %s", len(assigned), c.username)
		}
	}
}
