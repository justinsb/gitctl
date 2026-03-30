package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mime"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage"
)

// Server is the HTTP server exposing the gitctl Kubernetes-style API.
type Server struct {
	repoStore    *storage.ResourceStore[api.GitRepo]
	prStore      *storage.ResourceStore[api.PullRequest]
	issueStore   *storage.ResourceStore[api.Issue]
	commentStore *storage.ResourceStore[api.Comment]
	githubClient *github.Client
	mux          *http.ServeMux
}

// NewServer creates a new HTTP Server and registers its routes.
func NewServer(repoStore *storage.ResourceStore[api.GitRepo], prStore *storage.ResourceStore[api.PullRequest], issueStore *storage.ResourceStore[api.Issue], commentStore *storage.ResourceStore[api.Comment], githubClient *github.Client) *Server {
	s := &Server{
		repoStore:    repoStore,
		prStore:      prStore,
		issueStore:   issueStore,
		commentStore: commentStore,
		githubClient: githubClient,
		mux:          http.NewServeMux(),
	}

	base := "/apis/" + api.Group + "/" + api.Version
	s.mux.HandleFunc(base+"/gitrepos", s.handleListGitRepos)
	s.mux.HandleFunc(base+"/pullrequests", s.handleListPullRequests)
	s.mux.HandleFunc(base+"/issues", s.handleListIssues)
	s.mux.HandleFunc(base+"/comments", s.handleListComments)

	return s
}

// ServeHTTP implements http.Handler so Server can be passed directly to http.Serve.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleListGitRepos handles GET /apis/gitctl.justinsb.com/v1alpha1/gitrepos.
// The username to query is passed as the "username" query parameter.
func (s *Server) handleListGitRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		http.Error(w, "username query parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("Listing repositories for username: %s", username)

	repos, _, err := s.repoStore.List(r.Context(), username)
	if err != nil {
		log.Printf("Error listing repositories: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Found %d repositories for %s", len(repos), username)

	list := api.GitRepoList{
		APIVersion: api.APIVersion,
		Kind:       api.GitRepoListKind,
		Items:      repos,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleListPullRequests handles GET /apis/gitctl.justinsb.com/v1alpha1/pullrequests.
// Query parameters: username (required), scope (required: "outbound" or "assigned").
func (s *Server) handleListPullRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		http.Error(w, "username query parameter is required", http.StatusBadRequest)
		return
	}

	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope == "" {
		http.Error(w, "scope query parameter is required (outbound or assigned)", http.StatusBadRequest)
		return
	}

	key := scope + ":" + username
	log.Printf("Listing pull requests for key: %s", key)

	prs, _, err := s.prStore.List(r.Context(), key)
	if err != nil {
		log.Printf("Error listing pull requests: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Found %d pull requests for %s", len(prs), key)

	list := api.PullRequestList{
		APIVersion: api.APIVersion,
		Kind:       api.PullRequestListKind,
		Items:      prs,
	}

	if wantsHTML(r) {
		renderPRBodies(list.Items)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleListIssues handles GET /apis/gitctl.justinsb.com/v1alpha1/issues.
// Query parameters: username (required), scope (required: "assigned").
func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		http.Error(w, "username query parameter is required", http.StatusBadRequest)
		return
	}

	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope == "" {
		http.Error(w, "scope query parameter is required (assigned)", http.StatusBadRequest)
		return
	}

	key := scope + ":" + username
	log.Printf("Listing issues for key: %s", key)

	issues, _, err := s.issueStore.List(r.Context(), key)
	if err != nil {
		log.Printf("Error listing issues: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Found %d issues for %s", len(issues), key)

	list := api.IssueList{
		APIVersion: api.APIVersion,
		Kind:       api.IssueListKind,
		Items:      issues,
	}

	if wantsHTML(r) {
		renderIssueBodies(list.Items)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleListComments handles GET /apis/gitctl.justinsb.com/v1alpha1/comments.
// Query parameters: repo (required, e.g. "owner/repo"), number (required, issue/PR number).
// Comments are cached in storage after first fetch from GitHub.
func (s *Server) handleListComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repo := strings.TrimSpace(r.URL.Query().Get("repo"))
	if repo == "" {
		http.Error(w, "repo query parameter is required", http.StatusBadRequest)
		return
	}

	numberStr := strings.TrimSpace(r.URL.Query().Get("number"))
	if numberStr == "" {
		http.Error(w, "number query parameter is required", http.StatusBadRequest)
		return
	}

	number, err := strconv.Atoi(numberStr)
	if err != nil {
		http.Error(w, "number must be an integer", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s#%d", repo, number)
	log.Printf("Listing comments for %s", key)

	// Check cache first.
	comments, cached, err := s.commentStore.List(r.Context(), key)
	if err != nil {
		log.Printf("Error reading comments from cache: %v", err)
	}

	if !cached {
		// Cache miss: fetch from GitHub and store.
		comments, err = s.githubClient.ListIssueComments(r.Context(), repo, number)
		if err != nil {
			log.Printf("Error listing comments: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if storeErr := s.commentStore.ReplaceAll(r.Context(), key, comments); storeErr != nil {
			log.Printf("Error caching comments: %v", storeErr)
		}
	} else {
		// Cache hit: refresh in the background for next time.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			fresh, err := s.githubClient.ListIssueComments(ctx, repo, number)
			if err != nil {
				log.Printf("Error refreshing comments for %s: %v", key, err)
				return
			}
			if storeErr := s.commentStore.ReplaceAll(ctx, key, fresh); storeErr != nil {
				log.Printf("Error storing refreshed comments for %s: %v", key, storeErr)
			}
		}()
	}

	log.Printf("Found %d comments for %s (cached=%v)", len(comments), key, cached)

	list := api.CommentList{
		APIVersion: api.APIVersion,
		Kind:       api.CommentListKind,
		Items:      comments,
	}

	if wantsHTML(r) {
		renderCommentBodies(list.Items)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// wantsHTML checks whether the client prefers HTML-rendered markdown bodies.
// Clients signal this by including "text/html" in the Accept header.
func wantsHTML(r *http.Request) bool {
	for _, a := range strings.Split(r.Header.Get("Accept"), ",") {
		mt, _, err := mime.ParseMediaType(strings.TrimSpace(a))
		if err == nil && mt == "text/html" {
			return true
		}
	}
	return false
}

// renderPRBodies converts markdown Body fields to HTML in-place.
func renderPRBodies(prs []api.PullRequest) {
	for i := range prs {
		if prs[i].Spec.Body != "" {
			prs[i].Spec.Body = renderMarkdown(prs[i].Spec.Body)
		}
	}
}

// renderIssueBodies converts markdown Body fields to HTML in-place.
func renderIssueBodies(issues []api.Issue) {
	for i := range issues {
		if issues[i].Spec.Body != "" {
			issues[i].Spec.Body = renderMarkdown(issues[i].Spec.Body)
		}
	}
}

// renderCommentBodies converts markdown Body fields to HTML in-place.
func renderCommentBodies(comments []api.Comment) {
	for i := range comments {
		if comments[i].Spec.Body != "" {
			comments[i].Spec.Body = renderMarkdown(comments[i].Spec.Body)
		}
	}
}
