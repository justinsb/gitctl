package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pb "github.com/justinsb/gitctl/proto"
)

// Client is a GitHub API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new GitHub API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
	}
}

// GitHubRepo represents a repository from the GitHub API
type GitHubRepo struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	HTMLURL         string `json:"html_url"`
	Private         bool   `json:"private"`
	Fork            bool   `json:"fork"`
	StargazersCount int32  `json:"stargazers_count"`
	ForksCount      int32  `json:"forks_count"`
	OpenIssuesCount int32  `json:"open_issues_count"`
	Language        string `json:"language"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	PushedAt        string `json:"pushed_at"`
}

// ListRepositories fetches all repositories for a given username
func (c *Client) ListRepositories(ctx context.Context, username string) ([]*pb.Repository, error) {
	url := fmt.Sprintf("%s/users/%s/repos?per_page=100", c.baseURL, username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var githubRepos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&githubRepos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to protobuf format
	repos := make([]*pb.Repository, len(githubRepos))
	for i, gr := range githubRepos {
		repos[i] = &pb.Repository{
			Name:             gr.Name,
			FullName:         gr.FullName,
			Description:      gr.Description,
			HtmlUrl:          gr.HTMLURL,
			Private:          gr.Private,
			Fork:             gr.Fork,
			StargazersCount:  gr.StargazersCount,
			ForksCount:       gr.ForksCount,
			OpenIssuesCount:  gr.OpenIssuesCount,
			Language:         gr.Language,
			CreatedAt:        gr.CreatedAt,
			UpdatedAt:        gr.UpdatedAt,
			PushedAt:         gr.PushedAt,
		}
	}

	return repos, nil
}
