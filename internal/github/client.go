package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/justinsb/gitctl/internal/api"
)

// Client is a GitHub API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new GitHub API client.
// If GITHUB_TOKEN is set in the environment, it will be used for authentication.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
		token:   os.Getenv("GITHUB_TOKEN"),
	}
}

// setAuthHeader adds the Authorization header if a token is configured.
func (c *Client) setAuthHeader(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

// githubRepo represents a repository from the GitHub API
type githubRepo struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	HTMLURL         string `json:"html_url"`
	Private         bool   `json:"private"`
	Fork            bool   `json:"fork"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	OpenIssuesCount int    `json:"open_issues_count"`
	Language        string `json:"language"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	PushedAt        string `json:"pushed_at"`
}

// ListRepositories fetches all repositories for a given username and returns
// them as CRD-style GitRepo resources.
func (c *Client) ListRepositories(ctx context.Context, username string) ([]api.GitRepo, error) {
	url := fmt.Sprintf("%s/users/%s/repos?per_page=100", c.baseURL, username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var githubRepos []githubRepo
	if err := json.NewDecoder(resp.Body).Decode(&githubRepos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to CRD-style GitRepo resources.
	// User-specified fields go into Spec; GitHub-generated fields go into Status.
	repos := make([]api.GitRepo, len(githubRepos))
	for i, gr := range githubRepos {
		repos[i] = api.GitRepo{
			APIVersion: api.APIVersion,
			Kind:       api.GitRepoKind,
			Metadata: api.ObjectMeta{
				Name: gr.Name,
			},
			Spec: api.GitRepoSpec{
				Description: gr.Description,
				Private:     gr.Private,
			},
			Status: api.GitRepoStatus{
				FullName:        gr.FullName,
				HTMLURL:         gr.HTMLURL,
				Fork:            gr.Fork,
				StargazersCount: gr.StargazersCount,
				ForksCount:      gr.ForksCount,
				OpenIssuesCount: gr.OpenIssuesCount,
				Language:        gr.Language,
				CreatedAt:       gr.CreatedAt,
				UpdatedAt:       gr.UpdatedAt,
				PushedAt:        gr.PushedAt,
			},
		}
	}

	return repos, nil
}

// githubSearchResult represents the response from the GitHub Search API.
type githubSearchResult struct {
	Items []githubSearchItem `json:"items"`
}

// githubSearchItem represents an item from the GitHub Search API (/search/issues).
type githubSearchItem struct {
	Number        int           `json:"number"`
	Title         string        `json:"title"`
	Body          string        `json:"body"`
	State         string        `json:"state"`
	HTMLURL       string        `json:"html_url"`
	RepositoryURL string        `json:"repository_url"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	Draft         bool          `json:"draft"`
	User          githubUser    `json:"user"`
	Assignees     []githubUser  `json:"assignees"`
	Labels        []githubLabel `json:"labels"`
	PullRequest   *githubPRRef  `json:"pull_request"`
}

type githubUser struct {
	Login string `json:"login"`
}

type githubLabel struct {
	Name string `json:"name"`
}

type githubPRRef struct {
	MergedAt string `json:"merged_at"`
}

// repoFullName extracts "owner/repo" from a repository_url like
// "https://api.github.com/repos/owner/repo".
func repoFullName(repositoryURL string) string {
	const prefix = "/repos/"
	i := strings.LastIndex(repositoryURL, prefix)
	if i == -1 {
		return ""
	}
	return repositoryURL[i+len(prefix):]
}

// searchIssues performs a GitHub search and returns the raw items.
func (c *Client) searchIssues(ctx context.Context, query string) ([]githubSearchItem, error) {
	u := fmt.Sprintf("%s/search/issues?q=%s&sort=updated&order=desc&per_page=100",
		c.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub Search API returned status %d", resp.StatusCode)
	}

	var result githubSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return result.Items, nil
}

// SearchPullRequestsByAuthor returns open PRs authored by the given username.
func (c *Client) SearchPullRequestsByAuthor(ctx context.Context, username string) ([]api.PullRequest, error) {
	query := fmt.Sprintf("type:pr author:%s is:open", username)
	items, err := c.searchIssues(ctx, query)
	if err != nil {
		return nil, err
	}
	return convertToPullRequests(items), nil
}

// SearchAssignedPullRequests returns open PRs assigned to the given username.
func (c *Client) SearchAssignedPullRequests(ctx context.Context, username string) ([]api.PullRequest, error) {
	query := fmt.Sprintf("type:pr assignee:%s is:open", username)
	items, err := c.searchIssues(ctx, query)
	if err != nil {
		return nil, err
	}
	return convertToPullRequests(items), nil
}

// SearchAssignedIssues returns open issues assigned to the given username.
func (c *Client) SearchAssignedIssues(ctx context.Context, username string) ([]api.Issue, error) {
	query := fmt.Sprintf("type:issue assignee:%s is:open", username)
	items, err := c.searchIssues(ctx, query)
	if err != nil {
		return nil, err
	}
	return convertToIssues(items), nil
}

func convertToPullRequests(items []githubSearchItem) []api.PullRequest {
	prs := make([]api.PullRequest, len(items))
	for i, item := range items {
		assignees := make([]string, len(item.Assignees))
		for j, a := range item.Assignees {
			assignees[j] = a.Login
		}
		labels := make([]string, len(item.Labels))
		for j, l := range item.Labels {
			labels[j] = l.Name
		}
		merged := item.PullRequest != nil && item.PullRequest.MergedAt != ""
		prs[i] = api.PullRequest{
			APIVersion: api.APIVersion,
			Kind:       api.PullRequestKind,
			Metadata: api.ObjectMeta{
				Name: fmt.Sprintf("%s#%d", repoFullName(item.RepositoryURL), item.Number),
			},
			Spec: api.PullRequestSpec{
				Title: item.Title,
				Body:  item.Body,
			},
			Status: api.PullRequestStatus{
				Repo:      repoFullName(item.RepositoryURL),
				Number:    item.Number,
				State:     item.State,
				Author:    item.User.Login,
				Assignees: assignees,
				HTMLURL:   item.HTMLURL,
				Draft:     item.Draft,
				Merged:    merged,
				Labels:    labels,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			},
		}
	}
	return prs
}

func convertToIssues(items []githubSearchItem) []api.Issue {
	issues := make([]api.Issue, len(items))
	for i, item := range items {
		assignees := make([]string, len(item.Assignees))
		for j, a := range item.Assignees {
			assignees[j] = a.Login
		}
		labels := make([]string, len(item.Labels))
		for j, l := range item.Labels {
			labels[j] = l.Name
		}
		issues[i] = api.Issue{
			APIVersion: api.APIVersion,
			Kind:       api.IssueKind,
			Metadata: api.ObjectMeta{
				Name: fmt.Sprintf("%s#%d", repoFullName(item.RepositoryURL), item.Number),
			},
			Spec: api.IssueSpec{
				Title: item.Title,
				Body:  item.Body,
			},
			Status: api.IssueStatus{
				Repo:      repoFullName(item.RepositoryURL),
				Number:    item.Number,
				State:     item.State,
				Author:    item.User.Login,
				Assignees: assignees,
				HTMLURL:   item.HTMLURL,
				Labels:    labels,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			},
		}
	}
	return issues
}

type githubComment struct {
	ID        int        `json:"id"`
	Body      string     `json:"body"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	User      githubUser `json:"user"`
}

// ListIssueComments fetches comments for an issue or pull request.
// The GitHub API uses the same endpoint for both issue and PR comments.
func (c *Client) ListIssueComments(ctx context.Context, repo string, number int) ([]api.Comment, error) {
	u := fmt.Sprintf("%s/repos/%s/issues/%d/comments?per_page=100", c.baseURL, repo, number)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch comments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var ghComments []githubComment
	if err := json.NewDecoder(resp.Body).Decode(&ghComments); err != nil {
		return nil, fmt.Errorf("failed to decode comments response: %w", err)
	}

	comments := make([]api.Comment, len(ghComments))
	for i, gc := range ghComments {
		comments[i] = api.Comment{
			APIVersion: api.APIVersion,
			Kind:       api.CommentKind,
			Metadata: api.ObjectMeta{
				Name: fmt.Sprintf("%s#%d-comment-%d", repo, number, gc.ID),
			},
			Spec: api.CommentSpec{
				Body: gc.Body,
			},
			Status: api.CommentStatus{
				Author:    gc.User.Login,
				HTMLURL:   gc.HTMLURL,
				CreatedAt: gc.CreatedAt,
				UpdatedAt: gc.UpdatedAt,
			},
		}
	}

	return comments, nil
}
