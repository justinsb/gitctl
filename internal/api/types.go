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
	APIVersion string     `json:"apiVersion,omitempty"`
	Kind       string     `json:"kind,omitempty"`
	Metadata   ListMeta   `json:"metadata,omitempty"`
	Items      []GitRepo  `json:"items"`
}
