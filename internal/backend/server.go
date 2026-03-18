package backend

import (
	"context"
	"log"

	"github.com/justinsb/gitctl/internal/github"
	pb "github.com/justinsb/gitctl/proto"
)

// Server implements the GitCtl gRPC service
type Server struct {
	pb.UnimplementedGitCtlServer
	githubClient *github.Client
}

// NewServer creates a new GitCtl server
func NewServer() *Server {
	return &Server{
		githubClient: github.NewClient(),
	}
}

// ListRepositories implements the ListRepositories RPC
func (s *Server) ListRepositories(ctx context.Context, req *pb.ListRepositoriesRequest) (*pb.ListRepositoriesResponse, error) {
	log.Printf("ListRepositories request for username: %s", req.Username)

	repos, err := s.githubClient.ListRepositories(ctx, req.Username)
	if err != nil {
		log.Printf("Error listing repositories: %v", err)
		return nil, err
	}

	log.Printf("Found %d repositories for %s", len(repos), req.Username)
	return &pb.ListRepositoriesResponse{
		Repositories: repos,
	}, nil
}
