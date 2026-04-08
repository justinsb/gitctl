// Package urlparse provides GitHub URL parsing for creating Views.
package urlparse

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseResult holds the result of parsing a GitHub URL into a search query.
type ParseResult struct {
	// Query is the GitHub search syntax query string
	// (e.g. "is:pr is:open repo:kubernetes/kops").
	Query string `json:"query"`
	// DisplayName is a human-friendly label for the view.
	DisplayName string `json:"displayName"`
}

// ParseGitHubURL parses a GitHub pulls or issues URL into a search query
// and display name suitable for creating a View.
//
// Example input:
//
//	https://github.com/kubernetes/kops/pulls?q=is%3Apr+is%3Aopen+-label%3Aarea%3Apython-client
//
// Example output:
//
//	Query:       "is:pr is:open -label:area:python-client repo:kubernetes/kops"
//	DisplayName: "kubernetes/kops - is:pr is:open -label:area:python-client"
//
// Returns (ParseResult{}, false) if the URL is not a supported GitHub URL.
func ParseGitHubURL(rawURL string) (ParseResult, bool) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "github.com" {
		return ParseResult{}, false
	}

	// Split path into components, ignoring empty segments.
	pathComponents := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	if len(pathComponents) < 3 {
		return ParseResult{}, false
	}

	owner := pathComponents[0]
	repo := pathComponents[1]
	section := pathComponents[2]

	if section != "pulls" && section != "issues" {
		return ParseResult{}, false
	}

	repoFullName := fmt.Sprintf("%s/%s", owner, repo)

	// url.ParseQuery correctly decodes both %XX escapes and '+' as space,
	// matching how GitHub encodes its search query parameters.
	queryParams, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ParseResult{}, false
	}

	qParam := queryParams.Get("q")

	// Append repo: filter if the query does not already have one.
	searchQuery := qParam
	if !strings.Contains(searchQuery, "repo:") {
		if searchQuery == "" {
			searchQuery = "repo:" + repoFullName
		} else {
			searchQuery = searchQuery + " repo:" + repoFullName
		}
	}

	displayName := repoFullName
	if qParam != "" {
		displayName = repoFullName + " - " + qParam
	}

	return ParseResult{Query: searchQuery, DisplayName: displayName}, true
}
