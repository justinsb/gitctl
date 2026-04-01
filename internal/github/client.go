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
	"github.com/justinsb/gitctl/klient/meta"
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
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.GitRepoKind,
				Metadata: meta.ObjectMeta{
					Name: gr.Name,
				},
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
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.PullRequestKind,
				Metadata: meta.ObjectMeta{
					Name: fmt.Sprintf("%s#%d", repoFullName(item.RepositoryURL), item.Number),
				},
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
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.IssueKind,
				Metadata: meta.ObjectMeta{
					Name: fmt.Sprintf("%s#%d", repoFullName(item.RepositoryURL), item.Number),
				},
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

// githubCommit represents a commit from the GitHub Pull Request Commits API.
type githubCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// githubCheckRunsResponse wraps the check runs array from the GitHub API.
type githubCheckRunsResponse struct {
	CheckRuns []githubCheckRun `json:"check_runs"`
}

type githubCheckRun struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	DetailsURL  string `json:"details_url"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
}

// githubPRFile represents a changed file from the GitHub Pull Request Files API.
type githubPRFile struct {
	SHA       string `json:"sha"`
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Patch     string `json:"patch"`
}

// githubReviewComment represents a review comment from the GitHub Pull Request Review Comments API.
type githubReviewComment struct {
	ID        int        `json:"id"`
	Body      string     `json:"body"`
	Path      string     `json:"path"`
	Position  *int       `json:"position"`
	Line      int        `json:"line"`
	Side      string     `json:"side"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	User      githubUser `json:"user"`
	DiffHunk  string     `json:"diff_hunk"`
	InReplyTo int        `json:"in_reply_to_id"`
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
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.CommentKind,
				Metadata: meta.ObjectMeta{
					Name: fmt.Sprintf("%s#%d-comment-%d", repo, number, gc.ID),
				},
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

// ListPRCommits fetches commits for a pull request.
func (c *Client) ListPRCommits(ctx context.Context, repo string, number int) ([]api.PRCommit, error) {
	u := fmt.Sprintf("%s/repos/%s/pulls/%d/commits?per_page=100", c.baseURL, repo, number)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR commits: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var ghCommits []githubCommit
	if err := json.NewDecoder(resp.Body).Decode(&ghCommits); err != nil {
		return nil, fmt.Errorf("failed to decode commits response: %w", err)
	}

	commits := make([]api.PRCommit, len(ghCommits))
	for i, gc := range ghCommits {
		commits[i] = api.PRCommit{
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.PRCommitKind,
				Metadata: meta.ObjectMeta{
					Name: gc.SHA,
				},
			},
			Spec: api.PRCommitSpec{
				Message: gc.Commit.Message,
				Author:  gc.Commit.Author.Name,
			},
			Status: api.PRCommitStatus{
				SHA:     gc.SHA,
				HTMLURL: gc.HTMLURL,
				Date:    gc.Commit.Author.Date,
			},
		}
	}

	return commits, nil
}

// ListCheckRuns fetches check runs for a given git ref (commit SHA or branch).
func (c *Client) ListCheckRuns(ctx context.Context, repo string, ref string) ([]api.CheckRun, error) {
	u := fmt.Sprintf("%s/repos/%s/commits/%s/check-runs?per_page=100", c.baseURL, repo, ref)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch check runs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var result githubCheckRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode check runs response: %w", err)
	}

	checks := make([]api.CheckRun, len(result.CheckRuns))
	for i, gc := range result.CheckRuns {
		checks[i] = api.CheckRun{
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.CheckRunKind,
				Metadata: meta.ObjectMeta{
					Name: fmt.Sprintf("%d", gc.ID),
				},
			},
			Spec: api.CheckRunSpec{
				Name: gc.Name,
			},
			Status: api.CheckRunStatus{
				Status:      gc.Status,
				Conclusion:  gc.Conclusion,
				DetailsURL:  gc.DetailsURL,
				StartedAt:   gc.StartedAt,
				CompletedAt: gc.CompletedAt,
			},
		}
	}

	return checks, nil
}

// ListPRFiles fetches the list of changed files for a pull request.
func (c *Client) ListPRFiles(ctx context.Context, repo string, number int) ([]api.PRFile, error) {
	u := fmt.Sprintf("%s/repos/%s/pulls/%d/files?per_page=100", c.baseURL, repo, number)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var ghFiles []githubPRFile
	if err := json.NewDecoder(resp.Body).Decode(&ghFiles); err != nil {
		return nil, fmt.Errorf("failed to decode files response: %w", err)
	}

	files := make([]api.PRFile, len(ghFiles))
	for i, gf := range ghFiles {
		files[i] = api.PRFile{
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.PRFileKind,
				Metadata: meta.ObjectMeta{
					Name: gf.Filename,
				},
			},
			Status: api.PRFileStatus{
				Filename:   gf.Filename,
				FileStatus: gf.Status,
				Additions:  gf.Additions,
				Deletions:  gf.Deletions,
				Changes:    gf.Changes,
				Patch:      gf.Patch,
			},
		}
	}

	return files, nil
}

// ListReviewComments fetches file-level review comments for a pull request.
func (c *Client) ListReviewComments(ctx context.Context, repo string, number int) ([]api.ReviewComment, error) {
	u := fmt.Sprintf("%s/repos/%s/pulls/%d/comments?per_page=100", c.baseURL, repo, number)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch review comments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var ghComments []githubReviewComment
	if err := json.NewDecoder(resp.Body).Decode(&ghComments); err != nil {
		return nil, fmt.Errorf("failed to decode review comments response: %w", err)
	}

	comments := make([]api.ReviewComment, len(ghComments))
	for i, gc := range ghComments {
		comments[i] = api.ReviewComment{
			KubeObject: meta.KubeObject{
				APIVersion: api.APIVersion,
				Kind:       api.ReviewCommentKind,
				Metadata: meta.ObjectMeta{
					Name: fmt.Sprintf("%s#%d-review-%d", repo, number, gc.ID),
				},
			},
			Spec: api.ReviewCommentSpec{
				Body: gc.Body,
			},
			Status: api.ReviewCommentStatus{
				Path:      gc.Path,
				Line:      gc.Line,
				Side:      gc.Side,
				Author:    gc.User.Login,
				HTMLURL:   gc.HTMLURL,
				CreatedAt: gc.CreatedAt,
				UpdatedAt: gc.UpdatedAt,
				DiffHunk:  gc.DiffHunk,
				InReplyTo: gc.InReplyTo,
				Outdated:  gc.Position == nil,
			},
		}
	}

	return comments, nil
}

// CreateIssueComment creates a new comment on an issue or pull request.
func (c *Client) CreateIssueComment(ctx context.Context, repo string, number int, body string) error {
	u := fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.baseURL, repo, number)

	payload := struct {
		Body string `json:"body"`
	}{Body: body}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u, strings.NewReader(string(jsonBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateReviewComment creates a new file-level review comment on a pull request.
func (c *Client) CreateReviewComment(ctx context.Context, repo string, number int, body string, commitSHA string, path string, line int) error {
	u := fmt.Sprintf("%s/repos/%s/pulls/%d/comments", c.baseURL, repo, number)

	payload := struct {
		Body     string `json:"body"`
		CommitID string `json:"commit_id"`
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Side     string `json:"side"`
	}{
		Body:     body,
		CommitID: commitSHA,
		Path:     path,
		Line:     line,
		Side:     "RIGHT",
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal review comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u, strings.NewReader(string(jsonBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create review comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return nil
}
