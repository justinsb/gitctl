package urlparse

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantQuery       string
		wantDisplayName string
		wantOK          bool
	}{
		{
			name:            "pulls with percent-encoded and plus-separated query",
			input:           "https://github.com/kubernetes/kops/pulls?q=is%3Apr+is%3Aopen+-label%3Aarea%3Apython-client",
			wantQuery:       "is:pr is:open -label:area:python-client repo:kubernetes/kops",
			wantDisplayName: "kubernetes/kops - is:pr is:open -label:area:python-client",
			wantOK:          true,
		},
		{
			name:            "pulls with sort:updated-desc qualifier",
			input:           "https://github.com/kubernetes/kops/pulls?q=is%3Apr+is%3Aopen+sort%3Aupdated-desc",
			wantQuery:       "is:pr is:open sort:updated-desc repo:kubernetes/kops",
			wantDisplayName: "kubernetes/kops - is:pr is:open sort:updated-desc",
			wantOK:          true,
		},
		{
			name:            "pulls with no query string",
			input:           "https://github.com/kubernetes/kops/pulls",
			wantQuery:       "repo:kubernetes/kops",
			wantDisplayName: "kubernetes/kops",
			wantOK:          true,
		},
		{
			name:            "issues URL",
			input:           "https://github.com/kubernetes/kops/issues?q=is%3Aissue+is%3Aopen",
			wantQuery:       "is:issue is:open repo:kubernetes/kops",
			wantDisplayName: "kubernetes/kops - is:issue is:open",
			wantOK:          true,
		},
		{
			name:            "query that already contains repo: filter",
			input:           "https://github.com/kubernetes/kops/pulls?q=is%3Apr+repo%3Akubernetes%2Fkops",
			wantQuery:       "is:pr repo:kubernetes/kops",
			wantDisplayName: "kubernetes/kops - is:pr repo:kubernetes/kops",
			wantOK:          true,
		},
		{
			name:   "not a github.com URL",
			input:  "https://gitlab.com/kubernetes/kops/pulls",
			wantOK: false,
		},
		{
			name:   "github.com repo root (no section)",
			input:  "https://github.com/kubernetes/kops",
			wantOK: false,
		},
		{
			name:   "github.com with unsupported section",
			input:  "https://github.com/kubernetes/kops/commits",
			wantOK: false,
		},
		{
			name:   "invalid URL",
			input:  "not a url",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseGitHubURL(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ParseGitHubURL(%q) ok = %v, want %v", tc.input, ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if got.Query != tc.wantQuery {
				t.Errorf("Query = %q, want %q", got.Query, tc.wantQuery)
			}
			if got.DisplayName != tc.wantDisplayName {
				t.Errorf("DisplayName = %q, want %q", got.DisplayName, tc.wantDisplayName)
			}
		})
	}
}
