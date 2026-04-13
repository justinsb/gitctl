package backend

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/storage"
	"github.com/justinsb/gitctl/klient/meta"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	viewStore := storage.NewCRUDStore[api.View](func(v api.View) string { return v.Metadata.Name })
	return NewServer(
		storage.NewResourceStore[api.GitRepo](),
		storage.NewResourceStore[api.PullRequest](),
		storage.NewResourceStore[api.Issue](),
		storage.NewResourceStore[api.Comment](),
		storage.NewResourceStore[api.PRCommit](),
		storage.NewResourceStore[api.CheckRun](),
		storage.NewResourceStore[api.PRFile](),
		storage.NewResourceStore[api.ReviewComment](),
		viewStore,
		nil,
		NewReadinessTracker(0),
	)
}

// TestParseURLWithSortQualifier tests that the /parseurl endpoint correctly
// handles a GitHub URL containing a sort: qualifier (regression test for #31).
func TestParseURLWithSortQualifier(t *testing.T) {
	s := newTestServer(t)

	// This is the exact URL from the bug report.
	rawURL := "https://github.com/kubernetes/kops/pulls?q=is%3Apr+is%3Aopen+sort%3Aupdated-desc"

	// Use url.QueryEscape to simulate how a well-behaved client would encode
	// the URL as a query parameter value.
	req := httptest.NewRequest(http.MethodGet,
		"/apis/gitctl.justinsb.com/v1alpha1/parseurl?url="+url.QueryEscape(rawURL),
		nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("parseurl: got status %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var result struct {
		Query       string `json:"query"`
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode parseurl response: %v", err)
	}

	wantQuery := "is:pr is:open sort:updated-desc repo:kubernetes/kops"
	if result.Query != wantQuery {
		t.Errorf("query = %q, want %q", result.Query, wantQuery)
	}
	wantDisplayName := "kubernetes/kops - is:pr is:open sort:updated-desc"
	if result.DisplayName != wantDisplayName {
		t.Errorf("displayName = %q, want %q", result.DisplayName, wantDisplayName)
	}
}

// TestCreateViewWithSortQualifierQuery tests the full flow of creating a view
// derived from parsing a GitHub URL with sort: syntax (regression test for #31).
func TestCreateViewWithSortQualifierQuery(t *testing.T) {
	s := newTestServer(t)

	view := api.View{
		KubeObject: meta.KubeObject{
			APIVersion: api.APIVersion,
			Kind:       api.ViewKind,
			Metadata:   meta.ObjectMeta{Name: "kuberneteskops---ispr-isopen-sortupdated-desc"},
		},
		Spec: api.ViewSpec{
			Query:       "is:pr is:open sort:updated-desc repo:kubernetes/kops",
			DisplayName: "kubernetes/kops - is:pr is:open sort:updated-desc",
		},
	}

	body, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("failed to marshal view: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost,
		"/apis/gitctl.justinsb.com/v1alpha1/views",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createView: got status %d, want 201; body: %s", w.Code, w.Body.String())
	}

	var created api.View
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode created view: %v", err)
	}
	if created.Metadata.Name != view.Metadata.Name {
		t.Errorf("name = %q, want %q", created.Metadata.Name, view.Metadata.Name)
	}
	if created.Spec.Query != view.Spec.Query {
		t.Errorf("query = %q, want %q", created.Spec.Query, view.Spec.Query)
	}
}
