package backend

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/storage"
)

// registerUIRoutes registers the /ui/ HTML page routes.
func (s *Server) registerUIRoutes() {
	s.mux.HandleFunc("/ui/repos/", s.handleUIRouter)
}

// handleUIRouter routes /ui/repos/{owner}/{repo}/pulls/{number}... requests.
func (s *Server) handleUIRouter(w http.ResponseWriter, r *http.Request) {
	// Parse path: /ui/repos/{owner}/{repo}/pulls/{number}[/suffix]
	// or:         /ui/repos/{owner}/{repo}/issues/{number}[/suffix]
	path := strings.TrimPrefix(r.URL.Path, "/ui/repos/")
	parts := strings.Split(path, "/")

	if len(parts) < 4 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	owner := parts[0]
	repo := parts[1]
	kind := parts[2] // "pulls" or "issues"
	number, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "invalid number", http.StatusBadRequest)
		return
	}

	suffix := ""
	if len(parts) > 4 {
		suffix = parts[4]
	}

	switch kind {
	case "pulls":
		switch {
		case r.Method == http.MethodPost && suffix == "comments":
			s.handleCreateIssueComment(w, r, owner, repo, number, "conversation")
		case r.Method == http.MethodPost && suffix == "review-comments":
			s.handleCreateReviewComment(w, r, owner, repo, number)
		case r.Method == http.MethodGet:
			s.handlePRDetail(w, r, owner, repo, number)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "issues":
		switch {
		case r.Method == http.MethodPost && suffix == "comments":
			s.handleCreateIssueComment(w, r, owner, repo, number, "")
		case r.Method == http.MethodGet:
			s.handleIssueDetail(w, r, owner, repo, number)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// handlePRDetail renders the PR detail HTML page with tabs.
func (s *Server) handlePRDetail(w http.ResponseWriter, r *http.Request, owner, repo string, number int) {
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "conversation"
	}

	fullRepo := owner + "/" + repo
	key := fmt.Sprintf("%s#%d", fullRepo, number)

	// Find the PR in storage (check both outbound and assigned scopes).
	pr := s.findPR(r.Context(), fullRepo, number)

	data := prDetailData{
		Owner:     owner,
		Repo:      repo,
		Number:    number,
		ActiveTab: tab,
		PR:        pr,
	}

	if pr == nil {
		data.Error = fmt.Sprintf("Pull request %s#%d not found. Make sure it appears in your Feed or Assigned list.", fullRepo, number)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		prDetailTemplate.Execute(w, data)
		return
	}

	// Render PR body markdown to HTML.
	if pr.Spec.Body != "" {
		pr.Spec.Body = renderMarkdown(pr.Spec.Body)
	}

	switch tab {
	case "conversation":
		comments := fetchCached(r.Context(), s.commentStore, key, func(ctx context.Context) ([]api.Comment, error) {
			return s.githubClient.ListIssueComments(ctx, fullRepo, number)
		})
		renderCommentBodies(comments)
		data.Comments = comments

	case "commits":
		data.Commits = fetchCached(r.Context(), s.commitStore, key, func(ctx context.Context) ([]api.PRCommit, error) {
			return s.githubClient.ListPRCommits(ctx, fullRepo, number)
		})

	case "checks":
		// Need head commit SHA for checks lookup.
		commits := fetchCached(r.Context(), s.commitStore, key, func(ctx context.Context) ([]api.PRCommit, error) {
			return s.githubClient.ListPRCommits(ctx, fullRepo, number)
		})
		if len(commits) > 0 {
			headSHA := commits[len(commits)-1].Status.SHA
			checkKey := fmt.Sprintf("%s#%d@%s", fullRepo, number, headSHA)
			data.CheckRuns = fetchCached(r.Context(), s.checkRunStore, checkKey, func(ctx context.Context) ([]api.CheckRun, error) {
				return s.githubClient.ListCheckRuns(ctx, fullRepo, headSHA)
			})
		}

	case "files":
		files := fetchCached(r.Context(), s.prFileStore, key, func(ctx context.Context) ([]api.PRFile, error) {
			return s.githubClient.ListPRFiles(ctx, fullRepo, number)
		})
		reviewComments := fetchCached(r.Context(), s.reviewCommentStore, key, func(ctx context.Context) ([]api.ReviewComment, error) {
			return s.githubClient.ListReviewComments(ctx, fullRepo, number)
		})
		renderReviewCommentBodies(reviewComments)
		data.ReviewComments = reviewComments

		for _, f := range files {
			fd := fileDiffData{
				File:           f,
				Hunks:          parsePatch(f.Status.Patch),
				ReviewComments: filterReviewCommentsForFile(reviewComments, f.Status.Filename),
			}
			data.Files = append(data.Files, fd)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := prDetailTemplate.Execute(w, data); err != nil {
		log.Printf("Error executing PR template: %v", err)
	}
}

// handleIssueDetail renders the issue detail HTML page.
func (s *Server) handleIssueDetail(w http.ResponseWriter, r *http.Request, owner, repo string, number int) {
	fullRepo := owner + "/" + repo
	key := fmt.Sprintf("%s#%d", fullRepo, number)

	issue := s.findIssue(r.Context(), fullRepo, number)

	data := issueDetailData{
		Owner:  owner,
		Repo:   repo,
		Number: number,
		Issue:  issue,
	}

	if issue == nil {
		data.Error = fmt.Sprintf("Issue %s#%d not found.", fullRepo, number)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		issueDetailTemplate.Execute(w, data)
		return
	}

	if issue.Spec.Body != "" {
		issue.Spec.Body = renderMarkdown(issue.Spec.Body)
	}

	comments := fetchCached(r.Context(), s.commentStore, key, func(ctx context.Context) ([]api.Comment, error) {
		return s.githubClient.ListIssueComments(ctx, fullRepo, number)
	})
	renderCommentBodies(comments)
	data.Comments = comments

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := issueDetailTemplate.Execute(w, data); err != nil {
		log.Printf("Error executing issue template: %v", err)
	}
}

// handleCreateIssueComment handles POST to create a new issue/PR comment.
func (s *Server) handleCreateIssueComment(w http.ResponseWriter, r *http.Request, owner, repo string, number int, redirectTab string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "comment body is required", http.StatusBadRequest)
		return
	}

	fullRepo := owner + "/" + repo
	key := fmt.Sprintf("%s#%d", fullRepo, number)

	if err := s.githubClient.CreateIssueComment(r.Context(), fullRepo, number, body); err != nil {
		log.Printf("Error creating comment on %s: %v", key, err)
		http.Error(w, fmt.Sprintf("Failed to create comment: %v", err), http.StatusInternalServerError)
		return
	}

	// Invalidate cache so next load fetches fresh comments.
	s.commentStore.Invalidate(r.Context(), key)

	// Redirect back to the detail page.
	redirectURL := fmt.Sprintf("/ui/repos/%s/%s/pulls/%d", owner, repo, number)
	if redirectTab != "" {
		redirectURL += "?tab=" + redirectTab
	}
	// Check if this is an issue comment (no tab param means issue).
	if redirectTab == "" {
		redirectURL = fmt.Sprintf("/ui/repos/%s/%s/issues/%d", owner, repo, number)
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// handleCreateReviewComment handles POST to create a new file-level review comment.
func (s *Server) handleCreateReviewComment(w http.ResponseWriter, r *http.Request, owner, repo string, number int) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	path := strings.TrimSpace(r.FormValue("path"))
	lineStr := strings.TrimSpace(r.FormValue("line"))

	if body == "" || path == "" || lineStr == "" {
		http.Error(w, "body, path, and line are required", http.StatusBadRequest)
		return
	}

	line, err := strconv.Atoi(lineStr)
	if err != nil {
		http.Error(w, "line must be an integer", http.StatusBadRequest)
		return
	}

	fullRepo := owner + "/" + repo
	key := fmt.Sprintf("%s#%d", fullRepo, number)

	// Get head commit SHA for the review comment.
	commits := fetchCached(r.Context(), s.commitStore, key, func(ctx context.Context) ([]api.PRCommit, error) {
		return s.githubClient.ListPRCommits(ctx, fullRepo, number)
	})
	if len(commits) == 0 {
		http.Error(w, "no commits found for this PR", http.StatusInternalServerError)
		return
	}
	headSHA := commits[len(commits)-1].Status.SHA

	if err := s.githubClient.CreateReviewComment(r.Context(), fullRepo, number, body, headSHA, path, line); err != nil {
		log.Printf("Error creating review comment on %s: %v", key, err)
		http.Error(w, fmt.Sprintf("Failed to create review comment: %v", err), http.StatusInternalServerError)
		return
	}

	// Invalidate review comment cache.
	s.reviewCommentStore.Invalidate(r.Context(), key)

	http.Redirect(w, r, fmt.Sprintf("/ui/repos/%s/%s/pulls/%d?tab=files", owner, repo, number), http.StatusSeeOther)
}

// fetchCached fetches items from a store, falling back to GitHub on cache miss.
// On cache hit, triggers a background refresh for next request.
func fetchCached[T any](ctx context.Context, store *storage.ResourceStore[T], key string, fetch func(context.Context) ([]T, error)) []T {
	items, cached, err := store.List(ctx, key)
	if err != nil {
		log.Printf("Error reading from cache for %s: %v", key, err)
	}

	if !cached {
		items, err = fetch(ctx)
		if err != nil {
			log.Printf("Error fetching %s: %v", key, err)
			return nil
		}
		if storeErr := store.ReplaceAll(ctx, key, items); storeErr != nil {
			log.Printf("Error caching %s: %v", key, storeErr)
		}
	} else {
		// Background refresh.
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			fresh, err := fetch(bgCtx)
			if err != nil {
				log.Printf("Error refreshing %s: %v", key, err)
				return
			}
			if storeErr := store.ReplaceAll(bgCtx, key, fresh); storeErr != nil {
				log.Printf("Error storing refreshed %s: %v", key, storeErr)
			}
		}()
	}

	return items
}

// findPR looks up a PR by repo and number across all stored scopes.
func (s *Server) findPR(ctx context.Context, repo string, number int) *api.PullRequest {
	allPRs, _, _ := s.prStore.ListAll(ctx)
	for _, prList := range allPRs {
		for i := range prList {
			if prList[i].Status.Repo == repo && prList[i].Status.Number == number {
				pr := prList[i]
				return &pr
			}
		}
	}
	return nil
}

// findIssue looks up an issue by repo and number across all stored scopes.
func (s *Server) findIssue(ctx context.Context, repo string, number int) *api.Issue {
	issues, _, _ := s.issueStore.ListAll(ctx)
	for _, issueList := range issues {
		for i := range issueList {
			if issueList[i].Status.Repo == repo && issueList[i].Status.Number == number {
				issue := issueList[i]
				return &issue
			}
		}
	}
	return nil
}

// renderReviewCommentBodies converts markdown bodies in review comments to HTML.
func renderReviewCommentBodies(comments []api.ReviewComment) {
	for i := range comments {
		if comments[i].Spec.Body != "" {
			comments[i].Spec.Body = renderMarkdown(comments[i].Spec.Body)
		}
	}
}

// filterReviewCommentsForFile returns review comments that belong to the given file path.
func filterReviewCommentsForFile(comments []api.ReviewComment, path string) []api.ReviewComment {
	var result []api.ReviewComment
	for _, c := range comments {
		if c.Status.Path == path {
			result = append(result, c)
		}
	}
	return result
}
