package backend

import (
	"html/template"

	"github.com/justinsb/gitctl/internal/api"
)

// prDetailData holds all data needed to render a PR detail page.
type prDetailData struct {
	Owner          string
	Repo           string
	Number         int
	ActiveTab      string
	PR             *api.PullRequest
	Comments       []api.Comment
	Commits        []api.PRCommit
	CheckRuns      []api.CheckRun
	Files          []fileDiffData
	ReviewComments []api.ReviewComment
	Error          string
}

// fileDiffData holds a file's diff parsed into hunks, with associated review comments.
type fileDiffData struct {
	File           api.PRFile
	Hunks          []DiffHunk
	ReviewComments []api.ReviewComment
}

// issueDetailData holds all data needed to render an issue detail page.
type issueDetailData struct {
	Owner    string
	Repo     string
	Number   int
	Issue    *api.Issue
	Comments []api.Comment
	Error    string
}

// funcMap for templates.
var templateFuncMap = template.FuncMap{
	"shortSHA": func(sha string) string {
		if len(sha) > 7 {
			return sha[:7]
		}
		return sha
	},
	"shortDate": func(date string) string {
		if len(date) >= 10 {
			return date[:10]
		}
		return date
	},
	"add": func(a, b int) int {
		return a + b
	},
	"reviewCommentsForLine": func(comments []api.ReviewComment, path string, line int) []api.ReviewComment {
		var result []api.ReviewComment
		for _, c := range comments {
			if c.Status.Path == path && c.Status.Line == line {
				result = append(result, c)
			}
		}
		return result
	},
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"checkIcon": func(status, conclusion string) string {
		if status != "completed" {
			return "⏳"
		}
		switch conclusion {
		case "success":
			return "✅"
		case "failure":
			return "❌"
		case "cancelled":
			return "⏹️"
		case "skipped":
			return "⏭️"
		case "neutral":
			return "◻️"
		default:
			return "❓"
		}
	},
	"fileStatusBadge": func(status string) string {
		switch status {
		case "added":
			return "A"
		case "removed":
			return "D"
		case "modified":
			return "M"
		case "renamed":
			return "R"
		case "copied":
			return "C"
		default:
			return "?"
		}
	},
	"fileStatusClass": func(status string) string {
		switch status {
		case "added":
			return "file-added"
		case "removed":
			return "file-removed"
		default:
			return "file-modified"
		}
	},
}

var prDetailTemplate = template.Must(template.New("pr-detail").Funcs(templateFuncMap).Parse(prDetailTemplateStr))
var issueDetailTemplate = template.Must(template.New("issue-detail").Funcs(templateFuncMap).Parse(issueDetailTemplateStr))

const prDetailTemplateStr = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>` + pageCSS + `</style>
</head>
<body>
{{if .Error}}<div class="error">{{.Error}}</div>{{else}}
<div class="header">
  <div class="meta">{{.Owner}}/{{.Repo}}#{{.Number}}</div>
  <h1>{{.PR.Spec.Title}}</h1>
  <div class="meta-row">
    <span>{{.PR.Status.Author}}</span>
    <span class="state-{{.PR.Status.State}}">{{.PR.Status.State}}</span>
    {{if .PR.Status.Draft}}<span class="badge">Draft</span>{{end}}
    {{if .PR.Status.Merged}}<span class="badge badge-merged">Merged</span>{{end}}
    {{if .PR.Status.UpdatedAt}}<span>{{shortDate .PR.Status.UpdatedAt}}</span>{{end}}
  </div>
  {{if .PR.Status.Labels}}
  <div class="labels">
    {{range .PR.Status.Labels}}<span class="label">{{.}}</span>{{end}}
  </div>
  {{end}}
</div>

<div class="tabs">
  <a href="/ui/repos/{{.Owner}}/{{.Repo}}/pulls/{{.Number}}" class="tab{{if eq .ActiveTab "conversation"}} active{{end}}">Conversation</a>
  <a href="/ui/repos/{{.Owner}}/{{.Repo}}/pulls/{{.Number}}?tab=commits" class="tab{{if eq .ActiveTab "commits"}} active{{end}}">Commits</a>
  <a href="/ui/repos/{{.Owner}}/{{.Repo}}/pulls/{{.Number}}?tab=checks" class="tab{{if eq .ActiveTab "checks"}} active{{end}}">Checks</a>
  <a href="/ui/repos/{{.Owner}}/{{.Repo}}/pulls/{{.Number}}?tab=files" class="tab{{if eq .ActiveTab "files"}} active{{end}}">Files changed</a>
</div>

{{if eq .ActiveTab "conversation"}}
<div class="tab-content">
  <div class="markdown-body">{{if .PR.Spec.Body}}{{safeHTML .PR.Spec.Body}}{{else}}<em class="meta">No description provided.</em>{{end}}</div>
  <hr>
  {{if .Comments}}
  <h2>{{len .Comments}} comment(s)</h2>
  {{range .Comments}}
  <div class="comment">
    <div class="comment-header">
      <span class="comment-author">{{.Status.Author}}</span>
      <span class="comment-date">{{shortDate .Status.CreatedAt}}</span>
    </div>
    <div class="markdown-body">{{safeHTML .Spec.Body}}</div>
  </div>
  {{end}}
  {{else}}
  <p class="meta">No comments yet.</p>
  {{end}}

  <div class="comment-form">
    <h3>Add a comment</h3>
    <form method="POST" action="/ui/repos/{{.Owner}}/{{.Repo}}/pulls/{{.Number}}/comments">
      <textarea name="body" rows="4" placeholder="Leave a comment..." required></textarea>
      <button type="submit">Comment</button>
    </form>
  </div>
</div>

{{else if eq .ActiveTab "commits"}}
<div class="tab-content">
  {{if .Commits}}
  <h2>{{len .Commits}} commit(s)</h2>
  <div class="commit-list">
    {{range .Commits}}
    <div class="commit-row">
      <div class="commit-message">{{.Spec.Message}}</div>
      <div class="commit-meta">
        <span>{{.Spec.Author}}</span>
        <span>{{shortDate .Status.Date}}</span>
        <code class="commit-sha">{{shortSHA .Status.SHA}}</code>
      </div>
    </div>
    {{end}}
  </div>
  {{else}}
  <p class="meta">No commits found.</p>
  {{end}}
</div>

{{else if eq .ActiveTab "checks"}}
<div class="tab-content">
  {{if .CheckRuns}}
  <h2>{{len .CheckRuns}} check(s)</h2>
  <div class="check-list">
    {{range .CheckRuns}}
    <div class="check-row">
      <span class="check-icon">{{checkIcon .Status.Status .Status.Conclusion}}</span>
      <span class="check-name">{{.Spec.Name}}</span>
      <span class="check-conclusion">{{if eq .Status.Status "completed"}}{{.Status.Conclusion}}{{else}}{{.Status.Status}}{{end}}</span>
      {{if .Status.DetailsURL}}<a href="{{.Status.DetailsURL}}" class="check-details">Details</a>{{end}}
    </div>
    {{end}}
  </div>
  {{else}}
  <p class="meta">No checks found.</p>
  {{end}}
</div>

{{else if eq .ActiveTab "files"}}
<div class="tab-content">
  {{if .Files}}
  <div class="files-summary">
    {{len .Files}} file(s) changed
  </div>
  {{range .Files}}
  <div class="file-diff">
    <div class="file-header">
      <span class="file-status-badge {{fileStatusClass .File.Status.FileStatus}}">{{fileStatusBadge .File.Status.FileStatus}}</span>
      <span class="file-name">{{.File.Status.Filename}}</span>
      <span class="file-stats">
        {{if .File.Status.Additions}}<span class="additions">+{{.File.Status.Additions}}</span>{{end}}
        {{if .File.Status.Deletions}}<span class="deletions">-{{.File.Status.Deletions}}</span>{{end}}
      </span>
    </div>
    {{if .Hunks}}
    <table class="diff-table">
    {{range .Hunks}}
      <tr class="diff-hunk-header"><td colspan="3">{{.Header}}</td></tr>
      {{range .Lines}}
      <tr class="diff-line diff-{{.Type}}">
        <td class="diff-line-num">{{if .OldLine}}{{.OldLine}}{{end}}</td>
        <td class="diff-line-num">{{if .NewLine}}{{.NewLine}}{{end}}</td>
        <td class="diff-line-content"><pre>{{.Content}}</pre></td>
      </tr>
      {{$comments := reviewCommentsForLine $.ReviewComments $.File.Status.Filename .NewLine}}
      {{range $comments}}
      <tr class="diff-comment-row">
        <td colspan="3">
          <div class="diff-comment">
            <div class="comment-header">
              <span class="comment-author">{{.Status.Author}}</span>
              <span class="comment-date">{{shortDate .Status.CreatedAt}}</span>
            </div>
            <div class="markdown-body">{{safeHTML .Spec.Body}}</div>
          </div>
        </td>
      </tr>
      {{end}}
      {{end}}
    {{end}}
    </table>
    {{end}}
  </div>
  {{end}}

  <div class="comment-form">
    <h3>Add a review comment</h3>
    <form method="POST" action="/ui/repos/{{.Owner}}/{{.Repo}}/pulls/{{.Number}}/review-comments">
      <label>File path: <input type="text" name="path" required placeholder="e.g. main.go"></label>
      <label>Line number: <input type="number" name="line" required min="1" placeholder="e.g. 42"></label>
      <textarea name="body" rows="4" placeholder="Leave a review comment..." required></textarea>
      <button type="submit">Add review comment</button>
    </form>
  </div>
  {{else}}
  <p class="meta">No files changed.</p>
  {{end}}
</div>
{{end}}
{{end}}
</body>
</html>`

const issueDetailTemplateStr = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>` + pageCSS + `</style>
</head>
<body>
{{if .Error}}<div class="error">{{.Error}}</div>{{else}}
<div class="header">
  <div class="meta">{{.Owner}}/{{.Repo}}#{{.Number}}</div>
  <h1>{{.Issue.Spec.Title}}</h1>
  <div class="meta-row">
    <span>{{.Issue.Status.Author}}</span>
    <span class="state-{{.Issue.Status.State}}">{{.Issue.Status.State}}</span>
    {{if .Issue.Status.UpdatedAt}}<span>{{shortDate .Issue.Status.UpdatedAt}}</span>{{end}}
  </div>
  {{if .Issue.Status.Labels}}
  <div class="labels">
    {{range .Issue.Status.Labels}}<span class="label">{{.}}</span>{{end}}
  </div>
  {{end}}
</div>
<hr>
<div class="markdown-body">{{if .Issue.Spec.Body}}{{safeHTML .Issue.Spec.Body}}{{else}}<em class="meta">No description provided.</em>{{end}}</div>
<hr>
{{if .Comments}}
<h2>{{len .Comments}} comment(s)</h2>
{{range .Comments}}
<div class="comment">
  <div class="comment-header">
    <span class="comment-author">{{.Status.Author}}</span>
    <span class="comment-date">{{shortDate .Status.CreatedAt}}</span>
  </div>
  <div class="markdown-body">{{safeHTML .Spec.Body}}</div>
</div>
{{end}}
{{else}}
<p class="meta">No comments yet.</p>
{{end}}

<div class="comment-form">
  <h3>Add a comment</h3>
  <form method="POST" action="/ui/repos/{{.Owner}}/{{.Repo}}/issues/{{.Number}}/comments">
    <textarea name="body" rows="4" placeholder="Leave a comment..." required></textarea>
    <button type="submit">Comment</button>
  </form>
</div>
{{end}}
</body>
</html>`

const pageCSS = `
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif;
    font-size: 14px;
    line-height: 1.6;
    color: #1f2328;
    background: #ffffff;
    padding: 20px;
}
@media (prefers-color-scheme: dark) {
    body { color: #e6edf3; background: #1e1e1e; }
    a { color: #58a6ff; }
    .comment { background: rgba(110,118,129,0.15); border-color: #30363d; }
    hr { background: #30363d; }
    .badge { background: rgba(110,118,129,0.3); }
    .badge-merged { background: rgba(163,113,247,0.2); }
    .label { background: rgba(56,132,244,0.2); }
    .markdown-body code { background: rgba(110,118,129,0.4); }
    .markdown-body pre { background: #161b22; }
    .markdown-body blockquote { border-left-color: #3b434b; color: #9198a1; }
    .markdown-body table th, .markdown-body table td { border-color: #30363d; }
    .markdown-body table tr:nth-child(2n) { background: rgba(110,118,129,0.1); }
    .markdown-body h1, .markdown-body h2 { border-bottom-color: #30363d; }
    .comment-header { border-bottom-color: #30363d; }
    .meta { color: #9198a1; }
    .meta-row { color: #9198a1; }
    .tabs { border-bottom-color: #30363d; }
    .tab { color: #9198a1; }
    .tab:hover { color: #e6edf3; }
    .tab.active { color: #e6edf3; border-bottom-color: #f78166; }
    .commit-row { border-bottom-color: #30363d; }
    .check-row { border-bottom-color: #30363d; }
    .file-header { background: #161b22; border-color: #30363d; }
    .diff-table { border-color: #30363d; }
    .diff-line-num { color: #9198a1; border-color: #30363d; }
    .diff-line-content { border-color: #30363d; }
    .diff-add { background: rgba(63,185,80,0.15); }
    .diff-add .diff-line-content { background: rgba(63,185,80,0.1); }
    .diff-remove { background: rgba(248,81,73,0.15); }
    .diff-remove .diff-line-content { background: rgba(248,81,73,0.1); }
    .diff-hunk-header td { background: rgba(56,132,244,0.15); color: #9198a1; }
    .diff-comment { background: rgba(110,118,129,0.15); border-color: #30363d; }
    textarea { background: #161b22; color: #e6edf3; border-color: #30363d; }
    input[type="text"], input[type="number"] { background: #161b22; color: #e6edf3; border-color: #30363d; }
    .file-diff { border-color: #30363d; }
    .files-summary { border-color: #30363d; }
}
h1 { font-size: 1.5em; font-weight: 600; margin: 8px 0; }
h2 { font-size: 1.2em; font-weight: 600; margin: 16px 0 8px 0; }
h3 { font-size: 1.1em; font-weight: 600; margin: 16px 0 8px 0; }
hr { height: 1px; background: #d1d9e0; border: 0; margin: 16px 0; }
a { color: #0969da; text-decoration: none; }
a:hover { text-decoration: underline; }
.meta { font-size: 12px; color: #656d76; }
.meta-row { display: flex; gap: 12px; align-items: center; font-size: 13px; color: #656d76; flex-wrap: wrap; }
.state-open { color: #1a7f37; }
.state-closed { color: #cf222e; }
.badge {
    font-size: 11px; padding: 1px 7px; border-radius: 12px;
    background: rgba(175,184,193,0.2);
}
.badge-merged { background: rgba(163,113,247,0.15); }
.labels { display: flex; gap: 4px; margin-top: 6px; flex-wrap: wrap; }
.label {
    font-size: 11px; padding: 1px 7px; border-radius: 12px;
    background: rgba(56,132,244,0.1);
}
.loading { color: #656d76; font-style: italic; }
.error { color: #cf222e; font-size: 14px; padding: 20px; }

/* Tabs */
.tabs {
    display: flex; gap: 0; margin: 16px 0 0 0;
    border-bottom: 1px solid #d1d9e0;
}
.tab {
    padding: 8px 16px; font-size: 14px; font-weight: 500;
    color: #656d76; text-decoration: none;
    border-bottom: 2px solid transparent; margin-bottom: -1px;
}
.tab:hover { color: #1f2328; text-decoration: none; }
.tab.active { color: #1f2328; border-bottom-color: #f78166; }
.tab-content { padding-top: 16px; }

/* Comments */
.comment {
    border: 1px solid #d1d9e0; border-radius: 8px;
    margin-bottom: 12px; overflow: hidden;
}
.comment-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 8px 12px; font-size: 13px;
    border-bottom: 1px solid #d1d9e0;
}
.comment-author { font-weight: 500; }
.comment-date { color: #656d76; font-size: 12px; }
.comment > .markdown-body { padding: 12px; }

/* Comment form */
.comment-form { margin-top: 24px; }
.comment-form textarea {
    width: 100%; padding: 12px; font-family: inherit; font-size: 14px;
    border: 1px solid #d1d9e0; border-radius: 6px;
    resize: vertical; margin-bottom: 8px;
}
.comment-form input[type="text"],
.comment-form input[type="number"] {
    padding: 6px 10px; font-family: inherit; font-size: 14px;
    border: 1px solid #d1d9e0; border-radius: 6px;
    margin-bottom: 8px;
}
.comment-form label {
    display: block; margin-bottom: 8px; font-size: 13px;
}
.comment-form button {
    padding: 8px 16px; font-size: 14px; font-weight: 500;
    background: #1a7f37; color: white; border: none; border-radius: 6px;
    cursor: pointer;
}
.comment-form button:hover { background: #168030; }

/* Commits */
.commit-list { margin-top: 8px; }
.commit-row {
    padding: 12px 0; border-bottom: 1px solid #d1d9e0;
}
.commit-message { font-weight: 500; margin-bottom: 4px; white-space: pre-line; }
.commit-meta { font-size: 12px; color: #656d76; display: flex; gap: 12px; align-items: center; }
.commit-sha {
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 12px; padding: 1px 6px; border-radius: 4px;
    background: rgba(175,184,193,0.2);
}

/* Checks */
.check-list { margin-top: 8px; }
.check-row {
    padding: 10px 0; border-bottom: 1px solid #d1d9e0;
    display: flex; align-items: center; gap: 10px;
}
.check-icon { font-size: 16px; }
.check-name { font-weight: 500; flex: 1; }
.check-conclusion { font-size: 12px; color: #656d76; }
.check-details { font-size: 12px; }

/* Files changed */
.files-summary {
    padding: 12px 0; margin-bottom: 16px; font-weight: 500;
    border-bottom: 1px solid #d1d9e0;
}
.file-diff { margin-bottom: 24px; border: 1px solid #d1d9e0; border-radius: 6px; overflow: hidden; }
.file-header {
    padding: 8px 12px; display: flex; align-items: center; gap: 8px;
    font-size: 13px; background: #f6f8fa; border-bottom: 1px solid #d1d9e0;
}
.file-status-badge {
    font-family: ui-monospace, SFMono-Regular, monospace;
    font-size: 11px; font-weight: 600; padding: 1px 5px; border-radius: 4px;
}
.file-added { color: #1a7f37; background: rgba(26,127,55,0.1); }
.file-removed { color: #cf222e; background: rgba(207,34,46,0.1); }
.file-modified { color: #9a6700; background: rgba(154,103,0,0.1); }
.file-name { font-family: ui-monospace, SFMono-Regular, monospace; font-weight: 500; }
.file-stats { margin-left: auto; font-size: 12px; }
.additions { color: #1a7f37; margin-right: 4px; }
.deletions { color: #cf222e; }

/* Diff table */
.diff-table {
    width: 100%; border-collapse: collapse;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 12px; line-height: 20px;
}
.diff-line-num {
    width: 50px; min-width: 50px; padding: 0 8px;
    text-align: right; color: #656d76; user-select: none;
    border-right: 1px solid #d1d9e0; vertical-align: top;
}
.diff-line-content {
    padding: 0 12px; white-space: pre-wrap; word-wrap: break-word;
}
.diff-line-content pre {
    margin: 0; font: inherit; white-space: pre-wrap; word-wrap: break-word;
}
.diff-add { background: rgba(63,185,80,0.1); }
.diff-add .diff-line-content { background: rgba(63,185,80,0.15); }
.diff-remove { background: rgba(248,81,73,0.1); }
.diff-remove .diff-line-content { background: rgba(248,81,73,0.15); }
.diff-hunk-header td {
    padding: 4px 12px; background: rgba(56,132,244,0.1);
    color: #656d76; font-size: 12px;
}
.diff-comment-row td { padding: 0; }
.diff-comment {
    margin: 4px 12px; border: 1px solid #d1d9e0; border-radius: 6px;
    overflow: hidden;
}
.diff-comment .comment-header { padding: 6px 10px; font-size: 12px; }
.diff-comment .markdown-body { padding: 8px 10px; font-family: -apple-system, BlinkMacSystemFont, sans-serif; font-size: 13px; }

/* Markdown body styles */
.markdown-body { word-wrap: break-word; overflow-wrap: break-word; }
.markdown-body > *:first-child { margin-top: 0; }
.markdown-body > *:last-child { margin-bottom: 0; }
.markdown-body p, .markdown-body blockquote, .markdown-body ul,
.markdown-body ol, .markdown-body table, .markdown-body pre {
    margin-top: 0; margin-bottom: 16px;
}
.markdown-body h1 { font-size: 1.5em; font-weight: 600; margin-top: 24px; margin-bottom: 16px; padding-bottom: 0.3em; border-bottom: 1px solid #d1d9e0; }
.markdown-body h2 { font-size: 1.3em; font-weight: 600; margin-top: 24px; margin-bottom: 16px; padding-bottom: 0.3em; border-bottom: 1px solid #d1d9e0; }
.markdown-body h3 { font-size: 1.15em; font-weight: 600; margin-top: 24px; margin-bottom: 16px; }
.markdown-body h4, .markdown-body h5, .markdown-body h6 { font-weight: 600; margin-top: 24px; margin-bottom: 16px; }
.markdown-body code {
    padding: 0.2em 0.4em; font-size: 85%;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    background: rgba(175,184,193,0.2); border-radius: 6px; white-space: break-spaces;
}
.markdown-body pre {
    padding: 16px; overflow: auto; font-size: 85%; line-height: 1.45;
    background: #f6f8fa; border-radius: 6px;
}
.markdown-body pre code { padding: 0; background: transparent; font-size: 100%; white-space: pre; }
.markdown-body blockquote { padding: 0 1em; border-left: 0.25em solid #d1d9e0; color: #59636e; }
.markdown-body ul, .markdown-body ol { padding-left: 2em; }
.markdown-body li + li { margin-top: 0.25em; }
.markdown-body img { max-width: 100%; height: auto; border-radius: 6px; }
.markdown-body table { border-spacing: 0; border-collapse: collapse; width: max-content; max-width: 100%; overflow: auto; display: block; }
.markdown-body table th, .markdown-body table td { padding: 6px 13px; border: 1px solid #d1d9e0; }
.markdown-body table th { font-weight: 600; }
.markdown-body table tr:nth-child(2n) { background: rgba(175,184,193,0.1); }
.markdown-body hr { height: 0.25em; padding: 0; margin: 24px 0; background: #d1d9e0; border: 0; }
.markdown-body input[type="checkbox"] { margin-right: 0.4em; }
`
