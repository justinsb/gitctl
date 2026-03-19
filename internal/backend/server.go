package backend

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/github"
)

// Server is the HTTP server exposing the gitctl Kubernetes-style API.
type Server struct {
	githubClient *github.Client
	mux          *http.ServeMux
}

// NewServer creates a new HTTP Server and registers its routes.
func NewServer() *Server {
	s := &Server{
		githubClient: github.NewClient(),
		mux:          http.NewServeMux(),
	}

	// LIST /apis/gitctl.justinsb.com/v1alpha1/gitrepos
	s.mux.HandleFunc("/apis/"+api.Group+"/"+api.Version+"/gitrepos", s.handleListGitRepos)

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

	repos, err := s.githubClient.ListRepositories(r.Context(), username)
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
