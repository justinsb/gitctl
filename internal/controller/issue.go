package controller

import (
	"context"
	"log"
	"time"

	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage"
)

// IssueController periodically polls GitHub for issues
// and writes them to storage.
type IssueController struct {
	githubClient *github.Client
	store        storage.IssueStore
	username     string
	interval     time.Duration
}

// NewIssueController creates a new controller that syncs issues for the given username.
func NewIssueController(client *github.Client, store storage.IssueStore, username string, interval time.Duration) *IssueController {
	return &IssueController{
		githubClient: client,
		store:        store,
		username:     username,
		interval:     interval,
	}
}

// Run starts the controller loop. It blocks until ctx is cancelled.
func (c *IssueController) Run(ctx context.Context) {
	log.Printf("IssueController: starting sync for username %q every %v", c.username, c.interval)

	c.sync(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("IssueController: stopping")
			return
		case <-ticker.C:
			c.sync(ctx)
		}
	}
}

func (c *IssueController) sync(ctx context.Context) {
	log.Printf("IssueController: syncing assigned issues for %s", c.username)

	issues, err := c.githubClient.SearchAssignedIssues(ctx, c.username)
	if err != nil {
		log.Printf("IssueController: error fetching assigned issues: %v", err)
		return
	}

	if err := c.store.ReplaceAllIssues(ctx, "assigned:"+c.username, issues); err != nil {
		log.Printf("IssueController: error storing assigned issues: %v", err)
		return
	}

	log.Printf("IssueController: synced %d assigned issues for %s", len(issues), c.username)
}
