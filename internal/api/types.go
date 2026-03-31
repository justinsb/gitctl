// Package api defines the Kubernetes CRD-style types for the gitctl API.
// Resources follow the Kubernetes wire protocol (JSON over REST).
package api

const (
	// Group is the API group for gitctl resources.
	Group = "gitctl.justinsb.com"
	// Version is the API version.
	Version = "v1alpha1"
	// GitRepoKind is the Kind name for a single GitRepo resource.
	GitRepoKind = "GitRepo"
	// GitRepoListKind is the Kind name for a list of GitRepo resources.
	GitRepoListKind = "GitRepoList"
	// PullRequestKind is the Kind name for a single PullRequest resource.
	PullRequestKind = "PullRequest"
	// PullRequestListKind is the Kind name for a list of PullRequest resources.
	PullRequestListKind = "PullRequestList"
	// IssueKind is the Kind name for a single Issue resource.
	IssueKind = "Issue"
	// IssueListKind is the Kind name for a list of Issue resources.
	IssueListKind = "IssueList"
	// CommentKind is the Kind name for a single Comment resource.
	CommentKind = "Comment"
	// CommentListKind is the Kind name for a list of Comment resources.
	CommentListKind = "CommentList"
	// PRCommitKind is the Kind name for a single PRCommit resource.
	PRCommitKind = "PRCommit"
	// PRCommitListKind is the Kind name for a list of PRCommit resources.
	PRCommitListKind = "PRCommitList"
	// CheckRunKind is the Kind name for a single CheckRun resource.
	CheckRunKind = "CheckRun"
	// CheckRunListKind is the Kind name for a list of CheckRun resources.
	CheckRunListKind = "CheckRunList"
	// PRFileKind is the Kind name for a single PRFile resource.
	PRFileKind = "PRFile"
	// PRFileListKind is the Kind name for a list of PRFile resources.
	PRFileListKind = "PRFileList"
	// ReviewCommentKind is the Kind name for a single ReviewComment resource.
	ReviewCommentKind = "ReviewComment"
	// ReviewCommentListKind is the Kind name for a list of ReviewComment resources.
	ReviewCommentListKind = "ReviewCommentList"
	// APIVersion is the combined apiVersion field value.
	APIVersion = Group + "/" + Version
)

// ObjectMeta holds metadata common to all resources.
type ObjectMeta struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// ListMeta holds metadata common to list resources.
// Reserved for future use (e.g. resourceVersion, continue token).
type ListMeta struct{}

// GitRepoSpec contains user-specified fields for a GitHub repository.
type GitRepoSpec struct {
	// Description is the human-readable description set by the repository owner.
	Description string `json:"description,omitempty"`
	// Private indicates whether the repository is private.
	Private bool `json:"private,omitempty"`
}

// GitRepoStatus contains GitHub-generated (observed) fields for a repository.
type GitRepoStatus struct {
	// FullName is the "owner/repo" qualified name assigned by GitHub.
	FullName string `json:"fullName,omitempty"`
	// HTMLURL is the browser URL for the repository.
	HTMLURL string `json:"htmlUrl,omitempty"`
	// Fork indicates whether this repository is a fork.
	Fork bool `json:"fork,omitempty"`
	// StargazersCount is the number of stars.
	StargazersCount int `json:"stargazersCount,omitempty"`
	// ForksCount is the number of forks.
	ForksCount int `json:"forksCount,omitempty"`
	// OpenIssuesCount is the number of open issues.
	OpenIssuesCount int `json:"openIssuesCount,omitempty"`
	// Language is the primary programming language detected by GitHub.
	Language string `json:"language,omitempty"`
	// CreatedAt is the RFC 3339 timestamp when the repository was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt is the RFC 3339 timestamp of the last metadata update.
	UpdatedAt string `json:"updatedAt,omitempty"`
	// PushedAt is the RFC 3339 timestamp of the last push.
	PushedAt string `json:"pushedAt,omitempty"`
}

// GitRepo represents a GitHub repository as a Kubernetes-style CRD resource.
// Group: gitctl.justinsb.com, Version: v1alpha1, Kind: GitRepo.
type GitRepo struct {
	APIVersion string     `json:"apiVersion,omitempty"`
	Kind       string     `json:"kind,omitempty"`
	Metadata   ObjectMeta    `json:"metadata,omitempty"`
	Spec       GitRepoSpec   `json:"spec,omitempty"`
	Status     GitRepoStatus `json:"status,omitempty"`
}

// GitRepoList is a list of GitRepo resources, following the Kubernetes list convention.
type GitRepoList struct {
	APIVersion string    `json:"apiVersion,omitempty"`
	Kind       string    `json:"kind,omitempty"`
	Metadata   ListMeta  `json:"metadata,omitempty"`
	Items      []GitRepo `json:"items"`
}

// PullRequestSpec contains user-specified fields for a pull request.
type PullRequestSpec struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// PullRequestStatus contains GitHub-generated (observed) fields for a pull request.
type PullRequestStatus struct {
	// Repo is the "owner/repo" qualified name.
	Repo string `json:"repo,omitempty"`
	// Number is the PR number within the repository.
	Number int `json:"number,omitempty"`
	// State is the PR state (open, closed).
	State string `json:"state,omitempty"`
	// Author is the GitHub username who created the PR.
	Author string `json:"author,omitempty"`
	// Assignees are the GitHub usernames assigned to the PR.
	Assignees []string `json:"assignees,omitempty"`
	// HTMLURL is the browser URL for the pull request.
	HTMLURL string `json:"htmlUrl,omitempty"`
	// Draft indicates whether this is a draft PR.
	Draft bool `json:"draft,omitempty"`
	// Merged indicates whether the PR has been merged.
	Merged bool `json:"merged,omitempty"`
	// Labels are the label names applied to the PR.
	Labels []string `json:"labels,omitempty"`
	// CreatedAt is the RFC 3339 timestamp when the PR was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt is the RFC 3339 timestamp of the last update.
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// PullRequest represents a GitHub pull request as a Kubernetes-style CRD resource.
type PullRequest struct {
	APIVersion string              `json:"apiVersion,omitempty"`
	Kind       string              `json:"kind,omitempty"`
	Metadata   ObjectMeta          `json:"metadata,omitempty"`
	Spec       PullRequestSpec     `json:"spec,omitempty"`
	Status     PullRequestStatus   `json:"status,omitempty"`
}

// PullRequestList is a list of PullRequest resources.
type PullRequestList struct {
	APIVersion string          `json:"apiVersion,omitempty"`
	Kind       string          `json:"kind,omitempty"`
	Metadata   ListMeta        `json:"metadata,omitempty"`
	Items      []PullRequest   `json:"items"`
}

// IssueSpec contains user-specified fields for an issue.
type IssueSpec struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// IssueStatus contains GitHub-generated (observed) fields for an issue.
type IssueStatus struct {
	// Repo is the "owner/repo" qualified name.
	Repo string `json:"repo,omitempty"`
	// Number is the issue number within the repository.
	Number int `json:"number,omitempty"`
	// State is the issue state (open, closed).
	State string `json:"state,omitempty"`
	// Author is the GitHub username who created the issue.
	Author string `json:"author,omitempty"`
	// Assignees are the GitHub usernames assigned to the issue.
	Assignees []string `json:"assignees,omitempty"`
	// HTMLURL is the browser URL for the issue.
	HTMLURL string `json:"htmlUrl,omitempty"`
	// Labels are the label names applied to the issue.
	Labels []string `json:"labels,omitempty"`
	// CreatedAt is the RFC 3339 timestamp when the issue was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt is the RFC 3339 timestamp of the last update.
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// Issue represents a GitHub issue as a Kubernetes-style CRD resource.
type Issue struct {
	APIVersion string      `json:"apiVersion,omitempty"`
	Kind       string      `json:"kind,omitempty"`
	Metadata   ObjectMeta  `json:"metadata,omitempty"`
	Spec       IssueSpec   `json:"spec,omitempty"`
	Status     IssueStatus `json:"status,omitempty"`
}

// IssueList is a list of Issue resources.
type IssueList struct {
	APIVersion string    `json:"apiVersion,omitempty"`
	Kind       string    `json:"kind,omitempty"`
	Metadata   ListMeta  `json:"metadata,omitempty"`
	Items      []Issue   `json:"items"`
}

// CommentSpec contains user-specified fields for a comment.
type CommentSpec struct {
	Body string `json:"body,omitempty"`
}

// CommentStatus contains GitHub-generated (observed) fields for a comment.
type CommentStatus struct {
	// Author is the GitHub username who wrote the comment.
	Author string `json:"author,omitempty"`
	// HTMLURL is the browser URL for the comment.
	HTMLURL string `json:"htmlUrl,omitempty"`
	// CreatedAt is the RFC 3339 timestamp when the comment was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt is the RFC 3339 timestamp of the last update.
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// Comment represents a GitHub issue or PR comment as a Kubernetes-style CRD resource.
type Comment struct {
	APIVersion string        `json:"apiVersion,omitempty"`
	Kind       string        `json:"kind,omitempty"`
	Metadata   ObjectMeta    `json:"metadata,omitempty"`
	Spec       CommentSpec   `json:"spec,omitempty"`
	Status     CommentStatus `json:"status,omitempty"`
}

// CommentList is a list of Comment resources.
type CommentList struct {
	APIVersion string    `json:"apiVersion,omitempty"`
	Kind       string    `json:"kind,omitempty"`
	Metadata   ListMeta  `json:"metadata,omitempty"`
	Items      []Comment `json:"items"`
}

// PRCommitSpec contains user-specified fields for a commit in a pull request.
type PRCommitSpec struct {
	Message string `json:"message,omitempty"`
	Author  string `json:"author,omitempty"`
}

// PRCommitStatus contains GitHub-generated fields for a commit.
type PRCommitStatus struct {
	SHA     string `json:"sha,omitempty"`
	HTMLURL string `json:"htmlUrl,omitempty"`
	Date    string `json:"date,omitempty"`
}

// PRCommit represents a commit within a pull request.
type PRCommit struct {
	APIVersion string         `json:"apiVersion,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	Metadata   ObjectMeta     `json:"metadata,omitempty"`
	Spec       PRCommitSpec   `json:"spec,omitempty"`
	Status     PRCommitStatus `json:"status,omitempty"`
}

// PRCommitList is a list of PRCommit resources.
type PRCommitList struct {
	APIVersion string     `json:"apiVersion,omitempty"`
	Kind       string     `json:"kind,omitempty"`
	Metadata   ListMeta   `json:"metadata,omitempty"`
	Items      []PRCommit `json:"items"`
}

// CheckRunSpec contains user-specified fields for a CI check run.
type CheckRunSpec struct {
	Name string `json:"name,omitempty"`
}

// CheckRunStatus contains GitHub-generated fields for a check run.
type CheckRunStatus struct {
	Status      string `json:"status,omitempty"`
	Conclusion  string `json:"conclusion,omitempty"`
	DetailsURL  string `json:"detailsUrl,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
}

// CheckRun represents a CI status check run.
type CheckRun struct {
	APIVersion string         `json:"apiVersion,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	Metadata   ObjectMeta     `json:"metadata,omitempty"`
	Spec       CheckRunSpec   `json:"spec,omitempty"`
	Status     CheckRunStatus `json:"status,omitempty"`
}

// CheckRunList is a list of CheckRun resources.
type CheckRunList struct {
	APIVersion string     `json:"apiVersion,omitempty"`
	Kind       string     `json:"kind,omitempty"`
	Metadata   ListMeta   `json:"metadata,omitempty"`
	Items      []CheckRun `json:"items"`
}

// PRFileSpec is empty; file metadata is entirely GitHub-generated.
type PRFileSpec struct{}

// PRFileStatus contains GitHub-generated fields for a changed file in a PR.
type PRFileStatus struct {
	Filename   string `json:"filename,omitempty"`
	FileStatus string `json:"fileStatus,omitempty"`
	Additions  int    `json:"additions,omitempty"`
	Deletions  int    `json:"deletions,omitempty"`
	Changes    int    `json:"changes,omitempty"`
	Patch      string `json:"patch,omitempty"`
}

// PRFile represents a changed file in a pull request.
type PRFile struct {
	APIVersion string       `json:"apiVersion,omitempty"`
	Kind       string       `json:"kind,omitempty"`
	Metadata   ObjectMeta   `json:"metadata,omitempty"`
	Spec       PRFileSpec   `json:"spec,omitempty"`
	Status     PRFileStatus `json:"status,omitempty"`
}

// PRFileList is a list of PRFile resources.
type PRFileList struct {
	APIVersion string   `json:"apiVersion,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Metadata   ListMeta `json:"metadata,omitempty"`
	Items      []PRFile `json:"items"`
}

// ReviewCommentSpec contains user-specified fields for a file-level review comment.
type ReviewCommentSpec struct {
	Body string `json:"body,omitempty"`
}

// ReviewCommentStatus contains GitHub-generated fields for a review comment.
type ReviewCommentStatus struct {
	Path      string `json:"path,omitempty"`
	Line      int    `json:"line,omitempty"`
	Side      string `json:"side,omitempty"`
	Author    string `json:"author,omitempty"`
	HTMLURL   string `json:"htmlUrl,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	DiffHunk  string `json:"diffHunk,omitempty"`
	InReplyTo int    `json:"inReplyTo,omitempty"`
}

// ReviewComment represents a file-level review comment on a pull request.
type ReviewComment struct {
	APIVersion string              `json:"apiVersion,omitempty"`
	Kind       string              `json:"kind,omitempty"`
	Metadata   ObjectMeta          `json:"metadata,omitempty"`
	Spec       ReviewCommentSpec   `json:"spec,omitempty"`
	Status     ReviewCommentStatus `json:"status,omitempty"`
}

// ReviewCommentList is a list of ReviewComment resources.
type ReviewCommentList struct {
	APIVersion string          `json:"apiVersion,omitempty"`
	Kind       string          `json:"kind,omitempty"`
	Metadata   ListMeta        `json:"metadata,omitempty"`
	Items      []ReviewComment `json:"items"`
}
