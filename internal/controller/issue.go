package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/backend"
	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage"
)

// IssueController periodically polls GitHub for issues
// and writes them to storage.
type IssueController struct {
	githubClient *github.Client
	store        *storage.ResourceStore[api.Issue]
	commentStore *storage.ResourceStore[api.Comment]
	username     string
	interval     time.Duration
	readiness    *backend.ReadinessTracker
}

// NewIssueController creates a new controller that syncs issues for the given username.
func NewIssueController(client *github.Client, store *storage.ResourceStore[api.Issue], commentStore *storage.ResourceStore[api.Comment], username string, interval time.Duration, readiness *backend.ReadinessTracker) *IssueController {
	return &IssueController{
		githubClient: client,
		store:        store,
		commentStore: commentStore,
		username:     username,
		interval:     interval,
		readiness:    readiness,
	}
}

// Run starts the controller loop. It blocks until ctx is cancelled.
func (c *IssueController) Run(ctx context.Context) {
	log.Printf("IssueController: starting sync for username %q every %v", c.username, c.interval)

	c.sync(ctx)
	c.readiness.ReportReady()

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

	if err := c.store.ReplaceAll(ctx, "assigned:"+c.username, issues); err != nil {
		log.Printf("IssueController: error storing assigned issues: %v", err)
		return
	}

	log.Printf("IssueController: synced %d assigned issues for %s", len(issues), c.username)

	// Pre-fetch comments for each issue.
	for _, issue := range issues {
		if issue.Status.Repo == "" || issue.Status.Number == 0 {
			continue
		}
		key := fmt.Sprintf("%s#%d", issue.Status.Repo, issue.Status.Number)
		comments, err := c.githubClient.ListIssueComments(ctx, issue.Status.Repo, issue.Status.Number)
		if err != nil {
			log.Printf("IssueController: error fetching comments for %s: %v", key, err)
			continue
		}
		if err := c.commentStore.ReplaceAll(ctx, key, comments); err != nil {
			log.Printf("IssueController: error storing comments for %s: %v", key, err)
		}
	}
}
