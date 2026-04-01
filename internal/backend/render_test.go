package backend

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/justinsb/gitctl/gotesting"
	"github.com/justinsb/gitctl/internal/api"
	"sigs.k8s.io/yaml"
)

// TestGoldenPRDetail walks testdata/ subdirectories and runs golden tests
// for PR detail and issue detail template rendering.
func TestGoldenPRDetail(t *testing.T) {
	testdataDir := "testdata"
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			dir := filepath.Join(testdataDir, entry.Name())
			runGoldenTest(t, dir)
		})
	}
}

func runGoldenTest(t *testing.T, dir string) {
	t.Helper()

	hasPR := fileExists(filepath.Join(dir, "pr.yaml"))
	hasIssue := fileExists(filepath.Join(dir, "issue.yaml"))

	if hasPR {
		runPRGoldenTest(t, dir)
	}

	if hasIssue {
		runIssueGoldenTest(t, dir)
	}

	if !hasPR && !hasIssue {
		t.Fatalf("testdata directory %s has neither pr.yaml nor issue.yaml", dir)
	}
}

func runPRGoldenTest(t *testing.T, dir string) {
	t.Helper()

	var pr api.PullRequest
	mustLoadYAML(t, filepath.Join(dir, "pr.yaml"), &pr)

	// Extract owner/repo/number from the PR status.
	owner, repo := splitRepo(pr.Status.Repo)
	number := pr.Status.Number

	// Render markdown in PR body (same as handler).
	if pr.Spec.Body != "" {
		pr.Spec.Body = renderMarkdown(pr.Spec.Body)
	}

	// Load optional associated resources.
	var comments []api.Comment
	if fileExists(filepath.Join(dir, "comments.yaml")) {
		mustLoadYAML(t, filepath.Join(dir, "comments.yaml"), &comments)
	}

	var commits []api.PRCommit
	if fileExists(filepath.Join(dir, "commits.yaml")) {
		mustLoadYAML(t, filepath.Join(dir, "commits.yaml"), &commits)
	}

	var checkRuns []api.CheckRun
	if fileExists(filepath.Join(dir, "checkruns.yaml")) {
		mustLoadYAML(t, filepath.Join(dir, "checkruns.yaml"), &checkRuns)
	}

	var files []api.PRFile
	if fileExists(filepath.Join(dir, "files.yaml")) {
		mustLoadYAML(t, filepath.Join(dir, "files.yaml"), &files)
	}

	var reviewComments []api.ReviewComment
	if fileExists(filepath.Join(dir, "reviewcomments.yaml")) {
		mustLoadYAML(t, filepath.Join(dir, "reviewcomments.yaml"), &reviewComments)
	}

	// Render each tab view.
	tabs := []struct {
		name       string
		goldenFile string
		buildData  func() prDetailData
	}{
		{
			name:       "conversation",
			goldenFile: "_pr_detail_conversation.html",
			buildData: func() prDetailData {
				c := cloneComments(comments)
				renderCommentBodies(c)
				rc := cloneReviewComments(reviewComments)
				renderReviewCommentBodies(rc)
				return prDetailData{
					Owner:          owner,
					Repo:           repo,
					Number:         number,
					ActiveTab:      "conversation",
					PR:             &pr,
					Comments:       c,
					ReviewComments: rc,
				}
			},
		},
		{
			name:       "commits",
			goldenFile: "_pr_detail_commits.html",
			buildData: func() prDetailData {
				return prDetailData{
					Owner:     owner,
					Repo:      repo,
					Number:    number,
					ActiveTab: "commits",
					PR:        &pr,
					Commits:   commits,
				}
			},
		},
		{
			name:       "checks",
			goldenFile: "_pr_detail_checks.html",
			buildData: func() prDetailData {
				return prDetailData{
					Owner:     owner,
					Repo:      repo,
					Number:    number,
					ActiveTab: "checks",
					PR:        &pr,
					CheckRuns: checkRuns,
				}
			},
		},
		{
			name:       "files",
			goldenFile: "_pr_detail_files.html",
			buildData: func() prDetailData {
				rc := cloneReviewComments(reviewComments)
				renderReviewCommentBodies(rc)

				var fileDiffs []fileDiffData
				for _, f := range files {
					fd := fileDiffData{
						File:           f,
						Hunks:          parsePatch(f.Status.Patch),
						ReviewComments: filterReviewCommentsForFile(rc, f.Status.Filename),
					}
					fileDiffs = append(fileDiffs, fd)
				}

				return prDetailData{
					Owner:          owner,
					Repo:           repo,
					Number:         number,
					ActiveTab:      "files",
					PR:             &pr,
					Files:          fileDiffs,
					ReviewComments: rc,
				}
			},
		},
	}

	for _, tab := range tabs {
		t.Run(tab.name, func(t *testing.T) {
			data := tab.buildData()
			var buf bytes.Buffer
			if err := prDetailTemplate.Execute(&buf, data); err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			goldenPath := filepath.Join(dir, tab.goldenFile)
			gotesting.CheckGoldenOutput(t, goldenPath, buf.String())
		})
	}
}

func runIssueGoldenTest(t *testing.T, dir string) {
	t.Helper()

	var issue api.Issue
	mustLoadYAML(t, filepath.Join(dir, "issue.yaml"), &issue)

	owner, repo := splitRepo(issue.Status.Repo)
	number := issue.Status.Number

	if issue.Spec.Body != "" {
		issue.Spec.Body = renderMarkdown(issue.Spec.Body)
	}

	var comments []api.Comment
	if fileExists(filepath.Join(dir, "comments.yaml")) {
		mustLoadYAML(t, filepath.Join(dir, "comments.yaml"), &comments)
		renderCommentBodies(comments)
	}

	data := issueDetailData{
		Owner:    owner,
		Repo:     repo,
		Number:   number,
		Issue:    &issue,
		Comments: comments,
	}

	var buf bytes.Buffer
	if err := issueDetailTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}

	goldenPath := filepath.Join(dir, "_issue_detail.html")
	gotesting.CheckGoldenOutput(t, goldenPath, buf.String())
}

// --- helpers ---

func mustLoadYAML(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", path, err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func splitRepo(fullRepo string) (owner, repo string) {
	parts := strings.SplitN(fullRepo, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return fullRepo, ""
}

// cloneComments returns a shallow copy so renderCommentBodies doesn't mutate
// the original slice for other subtests.
func cloneComments(comments []api.Comment) []api.Comment {
	out := make([]api.Comment, len(comments))
	copy(out, comments)
	return out
}

func cloneReviewComments(comments []api.ReviewComment) []api.ReviewComment {
	out := make([]api.ReviewComment, len(comments))
	copy(out, comments)
	return out
}

// renderPRBodies is used by the server but we handle it manually in tests
// since we call renderMarkdown directly on the PR body.
var _ = strconv.Itoa // keep strconv import if needed
