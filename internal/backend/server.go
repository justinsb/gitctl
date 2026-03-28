package backend

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/storage"
)

// Server is the HTTP server exposing the gitctl Kubernetes-style API.
type Server struct {
	repoStore storage.GitRepoStore
	prStore   storage.PullRequestStore
	issueStore storage.IssueStore
	mux       *http.ServeMux
}

// NewServer creates a new HTTP Server and registers its routes.
func NewServer(repoStore storage.GitRepoStore, prStore storage.PullRequestStore, issueStore storage.IssueStore) *Server {
	s := &Server{
		repoStore:  repoStore,
		prStore:    prStore,
		issueStore: issueStore,
		mux:        http.NewServeMux(),
	}

	base := "/apis/" + api.Group + "/" + api.Version
	s.mux.HandleFunc(base+"/gitrepos", s.handleListGitRepos)
	s.mux.HandleFunc(base+"/pullrequests", s.handleListPullRequests)
	s.mux.HandleFunc(base+"/issues", s.handleListIssues)

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

	repos, err := s.repoStore.List(r.Context(), username)
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

	prs, err := s.prStore.ListPullRequests(r.Context(), key)
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

	issues, err := s.issueStore.ListIssues(r.Context(), key)
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
