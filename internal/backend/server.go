package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mime"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage"
	"github.com/justinsb/gitctl/internal/urlparse"
	"github.com/justinsb/gitctl/klient/meta"
)

// Server is the HTTP server exposing the gitctl Kubernetes-style API.
type Server struct {
	repoStore          *storage.ResourceStore[api.GitRepo]
	prStore            *storage.ResourceStore[api.PullRequest]
	issueStore         *storage.ResourceStore[api.Issue]
	commentStore       *storage.ResourceStore[api.Comment]
	commitStore        *storage.ResourceStore[api.PRCommit]
	checkRunStore      *storage.ResourceStore[api.CheckRun]
	prFileStore        *storage.ResourceStore[api.PRFile]
	reviewCommentStore *storage.ResourceStore[api.ReviewComment]
	viewStore          storage.Storage[api.View]
	githubClient       *github.Client
	readiness          *ReadinessTracker
	mux                *http.ServeMux
}

// NewServer creates a new HTTP Server and registers its routes.
func NewServer(
	repoStore *storage.ResourceStore[api.GitRepo],
	prStore *storage.ResourceStore[api.PullRequest],
	issueStore *storage.ResourceStore[api.Issue],
	commentStore *storage.ResourceStore[api.Comment],
	commitStore *storage.ResourceStore[api.PRCommit],
	checkRunStore *storage.ResourceStore[api.CheckRun],
	prFileStore *storage.ResourceStore[api.PRFile],
	reviewCommentStore *storage.ResourceStore[api.ReviewComment],
	viewStore storage.Storage[api.View],
	githubClient *github.Client,
	readiness *ReadinessTracker,
) *Server {
	s := &Server{
		repoStore:          repoStore,
		prStore:            prStore,
		issueStore:         issueStore,
		commentStore:       commentStore,
		commitStore:        commitStore,
		checkRunStore:      checkRunStore,
		prFileStore:        prFileStore,
		reviewCommentStore: reviewCommentStore,
		viewStore:          viewStore,
		githubClient:       githubClient,
		readiness:          readiness,
		mux:                http.NewServeMux(),
	}

	base := "/apis/" + api.Group + "/" + api.Version
	s.mux.HandleFunc(base+"/gitrepos", s.handleListGitRepos)
	s.mux.HandleFunc(base+"/pullrequests", s.handleListPullRequests)
	s.mux.HandleFunc(base+"/issues", s.handleListIssues)
	s.mux.HandleFunc(base+"/comments", s.handleListComments)
	s.mux.HandleFunc(base+"/views", s.handleViews)
	s.mux.HandleFunc(base+"/views/", s.handleViewByName)
	s.mux.HandleFunc(base+"/parseurl", s.handleParseURL)
	s.mux.HandleFunc("/readyz", s.handleReadyz)

	s.registerUIRoutes()

	return s
}

// handleReadyz implements the Kubernetes /readyz convention.
// Returns 200 OK when the backend has completed its initial sync, 503 otherwise.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if s.readiness.IsReady() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	} else {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}
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
		KubeList: meta.KubeList{
			APIVersion: api.APIVersion,
			Kind:       api.GitRepoListKind,
		},
		Items: repos,
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
		KubeList: meta.KubeList{
			APIVersion: api.APIVersion,
			Kind:       api.PullRequestListKind,
		},
		Items: prs,
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
		KubeList: meta.KubeList{
			APIVersion: api.APIVersion,
			Kind:       api.IssueListKind,
		},
		Items: issues,
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
		KubeList: meta.KubeList{
			APIVersion: api.APIVersion,
			Kind:       api.CommentListKind,
		},
		Items: comments,
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

// handleViews handles GET (list) and POST (create) on /apis/.../views.
func (s *Server) handleViews(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		views, err := s.viewStore.List(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list := api.ViewList{
			KubeList: meta.KubeList{
				APIVersion: api.APIVersion,
				Kind:       api.ViewListKind,
			},
			Items: views,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		// TODO: Limit the request body size.
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading request body: %v", err)
			http.Error(w, "error reading request body", http.StatusBadRequest)
			return
		}
		log.Printf("createView request body (%d bytes): %s", len(bodyBytes), string(bodyBytes))
		var view api.View
		if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&view); err != nil {
			log.Printf("createView invalid JSON: %v, body: %q", err, string(bodyBytes))
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if view.Metadata.Name == "" {
			http.Error(w, "metadata.name is required", http.StatusBadRequest)
			return
		}
		view.APIVersion = api.APIVersion
		view.Kind = api.ViewKind
		if err := s.viewStore.Create(r.Context(), view); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(view)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleViewByName handles GET/PUT/DELETE on /apis/.../views/{name}
// and GET on /apis/.../views/{name}/results.
func (s *Server) handleViewByName(w http.ResponseWriter, r *http.Request) {
	base := "/apis/" + api.Group + "/" + api.Version + "/views/"
	remainder := strings.TrimPrefix(r.URL.Path, base)

	// Check for /results suffix
	if name, ok := strings.CutSuffix(remainder, "/results"); ok {
		s.handleViewResults(w, r, name)
		return
	}

	name := remainder

	switch r.Method {
	case http.MethodGet:
		view, found, err := s.viewStore.Get(r.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "view not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(view)

	case http.MethodPut:
		var view api.View
		if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		view.Metadata.Name = name
		view.APIVersion = api.APIVersion
		view.Kind = api.ViewKind
		if err := s.viewStore.Update(r.Context(), view); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(view)

	case http.MethodDelete:
		if err := s.viewStore.Delete(r.Context(), name); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleViewResults executes a view's query against GitHub and returns results.
func (s *Server) handleViewResults(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	view, found, err := s.viewStore.Get(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "view not found", http.StatusNotFound)
		return
	}

	query := view.Spec.Query

	// Resolve @me to the authenticated GitHub user.
	if strings.Contains(query, "@me") {
		username, err := s.githubClient.GetAuthenticatedUser(r.Context())
		if err != nil {
			http.Error(w, "failed to resolve @me: "+err.Error(), http.StatusInternalServerError)
			return
		}
		query = strings.ReplaceAll(query, "@me", username)
	}

	log.Printf("Executing view %q with query: %s", name, query)

	prs, issues, err := s.githubClient.SearchQuery(r.Context(), query)
	if err != nil {
		log.Printf("Error executing view query: %v", err)
		http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("View %q returned %d PRs and %d issues", name, len(prs), len(issues))

	results := api.ViewResults{
		KubeObject: meta.KubeObject{
			APIVersion: api.APIVersion,
			Kind:       "ViewResults",
		},
		PullRequests: prs,
		Issues:       issues,
	}

	if wantsHTML(r) {
		renderPRBodies(results.PullRequests)
		renderIssueBodies(results.Issues)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleParseURL handles GET /apis/.../parseurl?url=<github-url>.
// It parses a GitHub pulls/issues URL and returns the search query and display name.
func (s *Server) handleParseURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "url query parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("parseurl: parsing URL %q", rawURL)
	result, ok := urlparse.ParseGitHubURL(rawURL)
	if !ok {
		log.Printf("parseurl: URL %q is not a supported GitHub URL", rawURL)
		http.Error(w, "not a supported GitHub URL", http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
