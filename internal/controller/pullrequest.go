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

// PullRequestController periodically polls GitHub for pull requests
// and writes them to storage.
type PullRequestController struct {
	githubClient *github.Client
	store        *storage.ResourceStore[api.PullRequest]
	commentStore *storage.ResourceStore[api.Comment]
	username     string
	interval     time.Duration
	readiness    *backend.ReadinessTracker
}

// NewPullRequestController creates a new controller that syncs PRs for the given username.
func NewPullRequestController(client *github.Client, store *storage.ResourceStore[api.PullRequest], commentStore *storage.ResourceStore[api.Comment], username string, interval time.Duration, readiness *backend.ReadinessTracker) *PullRequestController {
	return &PullRequestController{
		githubClient: client,
		store:        store,
		commentStore: commentStore,
		username:     username,
		interval:     interval,
		readiness:    readiness,
	}
}

// Run starts the controller loop. It blocks until ctx is cancelled.
func (c *PullRequestController) Run(ctx context.Context) {
	log.Printf("PullRequestController: starting sync for username %q every %v", c.username, c.interval)

	c.sync(ctx)
	c.readiness.ReportReady()

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
		if err := c.store.ReplaceAll(ctx, "outbound:"+c.username, outbound); err != nil {
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
		if err := c.store.ReplaceAll(ctx, "assigned:"+c.username, assigned); err != nil {
			log.Printf("PullRequestController: error storing assigned PRs: %v", err)
		} else {
			log.Printf("PullRequestController: synced %d assigned PRs for %s", len(assigned), c.username)
		}
	}

	// Pre-fetch comments for all synced PRs.
	allPRs := append(outbound, assigned...)
	seen := make(map[string]bool)
	for _, pr := range allPRs {
		if pr.Status.Repo == "" || pr.Status.Number == 0 {
			continue
		}
		key := fmt.Sprintf("%s#%d", pr.Status.Repo, pr.Status.Number)
		if seen[key] {
			continue
		}
		seen[key] = true
		comments, err := c.githubClient.ListIssueComments(ctx, pr.Status.Repo, pr.Status.Number)
		if err != nil {
			log.Printf("PullRequestController: error fetching comments for %s: %v", key, err)
			continue
		}
		if err := c.commentStore.ReplaceAll(ctx, key, comments); err != nil {
			log.Printf("PullRequestController: error storing comments for %s: %v", key, err)
		}
	}
}
