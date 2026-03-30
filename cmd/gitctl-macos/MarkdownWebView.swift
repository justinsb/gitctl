import SwiftUI
import WebKit

// MARK: - Full-page Detail Web View

/// A single WKWebView that renders the entire detail page (header, body, comments) as HTML.
/// Handles its own scrolling — no parent ScrollView needed.
struct DetailWebView: NSViewRepresentable {
    let html: String

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeNSView(context: Context) -> WKWebView {
        let webView = WKWebView(frame: .zero)
        webView.navigationDelegate = context.coordinator
        webView.loadHTMLString(html, baseURL: nil)
        context.coordinator.currentHTML = html
        return webView
    }

    func updateNSView(_ webView: WKWebView, context: Context) {
        if context.coordinator.currentHTML != html {
            context.coordinator.currentHTML = html
            webView.loadHTMLString(html, baseURL: nil)
        }
    }

    class Coordinator: NSObject, WKNavigationDelegate {
        var currentHTML: String = ""

        func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
            if navigationAction.navigationType == .linkActivated, let url = navigationAction.request.url {
                NSWorkspace.shared.open(url)
                decisionHandler(.cancel)
            } else {
                decisionHandler(.allow)
            }
        }
    }
}

// MARK: - HTML Page Builder

/// Builds a complete HTML page for a PR or issue detail view.
struct DetailPageBuilder {

    /// Builds HTML for a pull request detail page including header, body, and comments.
    static func buildPRPage(pr: PullRequest, comments: [Comment]?, commentError: String?) -> String {
        var header = ""

        let repo = escapeHTML(pr.status?.repo ?? "")
        let number = pr.status?.number ?? 0
        let title = escapeHTML(pr.spec?.title ?? "untitled")
        let author = escapeHTML(pr.status?.author ?? "")
        let state = pr.status?.state ?? ""
        let stateColor = state == "open" ? "#1a7f37" : "#cf222e"

        header += "<div class=\"meta\">\(repo)#\(number)</div>\n"
        header += "<h1>\(title)</h1>\n"
        header += "<div class=\"meta-row\">"
        header += "<span>👤 \(author)</span>"
        header += "<span style=\"color:\(stateColor)\">● \(escapeHTML(state))</span>"
        if pr.status?.draft == true {
            header += "<span class=\"badge\">Draft</span>"
        }
        if pr.status?.merged == true {
            header += "<span class=\"badge badge-merged\">Merged</span>"
        }
        if let updated = pr.status?.updatedAt, updated.count >= 10 {
            header += "<span>🕐 \(escapeHTML(String(updated.prefix(10))))</span>"
        }
        header += "</div>\n"

        if let labels = pr.status?.labels, !labels.isEmpty {
            header += "<div class=\"labels\">"
            for label in labels {
                header += "<span class=\"label\">\(escapeHTML(label))</span>"
            }
            header += "</div>\n"
        }

        let body = pr.spec?.body ?? ""
        return buildPage(header: header, bodyHTML: body, comments: comments, commentError: commentError)
    }

    /// Builds HTML for an issue detail page including header, body, and comments.
    static func buildIssuePage(issue: Issue, comments: [Comment]?, commentError: String?) -> String {
        var header = ""

        let repo = escapeHTML(issue.status?.repo ?? "")
        let number = issue.status?.number ?? 0
        let title = escapeHTML(issue.spec?.title ?? "untitled")
        let author = escapeHTML(issue.status?.author ?? "")
        let state = issue.status?.state ?? ""
        let stateColor = state == "open" ? "#1a7f37" : "#cf222e"

        header += "<div class=\"meta\">\(repo)#\(number)</div>\n"
        header += "<h1>\(title)</h1>\n"
        header += "<div class=\"meta-row\">"
        header += "<span>👤 \(author)</span>"
        header += "<span style=\"color:\(stateColor)\">● \(escapeHTML(state))</span>"
        if let updated = issue.status?.updatedAt, updated.count >= 10 {
            header += "<span>🕐 \(escapeHTML(String(updated.prefix(10))))</span>"
        }
        header += "</div>\n"

        if let labels = issue.status?.labels, !labels.isEmpty {
            header += "<div class=\"labels\">"
            for label in labels {
                header += "<span class=\"label\">\(escapeHTML(label))</span>"
            }
            header += "</div>\n"
        }

        let body = issue.spec?.body ?? ""
        return buildPage(header: header, bodyHTML: body, comments: comments, commentError: commentError)
    }

    private static func buildPage(header: String, bodyHTML: String, comments: [Comment]?, commentError: String?) -> String {
        var commentsHTML = ""
        if let error = commentError {
            commentsHTML = "<p class=\"error\">\(escapeHTML(error))</p>"
        } else if let comments = comments {
            if comments.isEmpty {
                commentsHTML = "<p class=\"meta\">No comments</p>"
            } else {
                commentsHTML = "<h2>\(comments.count) comment(s)</h2>\n"
                for comment in comments {
                    let author = escapeHTML(comment.status?.author ?? "unknown")
                    let date = comment.status?.createdAt.flatMap { $0.count >= 10 ? String($0.prefix(10)) : nil } ?? ""
                    let body = comment.spec?.body ?? ""
                    commentsHTML += """
                    <div class="comment">
                        <div class="comment-header">
                            <span class="comment-author">👤 \(author)</span>
                            <span class="comment-date">\(escapeHTML(date))</span>
                        </div>
                        <div class="markdown-body">\(body)</div>
                    </div>
                    """
                }
            }
        } else {
            commentsHTML = "<p class=\"loading\">Loading comments...</p>"
        }

        return """
        <!DOCTYPE html>
        <html>
        <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <style>
        \(pageCSS)
        </style>
        </head>
        <body>
        <div class="header">\(header)</div>
        <hr>
        <div class="markdown-body">\(bodyHTML.isEmpty ? "<em class=\"meta\">No description provided.</em>" : bodyHTML)</div>
        <hr>
        <div class="comments">\(commentsHTML)</div>
        </body>
        </html>
        """
    }

    private static func escapeHTML(_ s: String) -> String {
        s.replacingOccurrences(of: "&", with: "&amp;")
         .replacingOccurrences(of: "<", with: "&lt;")
         .replacingOccurrences(of: ">", with: "&gt;")
         .replacingOccurrences(of: "\"", with: "&quot;")
    }
}

// MARK: - CSS

extension DetailPageBuilder {

static let pageCSS: String = #"""
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
}
h1 { font-size: 1.5em; font-weight: 600; margin: 8px 0; }
h2 { font-size: 1.2em; font-weight: 600; margin: 16px 0 8px 0; }
hr { height: 1px; background: #d1d9e0; border: 0; margin: 16px 0; }
a { color: #0969da; text-decoration: none; }
a:hover { text-decoration: underline; }
.meta { font-size: 12px; color: #656d76; }
.meta-row { display: flex; gap: 12px; align-items: center; font-size: 13px; color: #656d76; flex-wrap: wrap; }
.badge {
    font-size: 11px;
    padding: 1px 7px;
    border-radius: 12px;
    background: rgba(175,184,193,0.2);
}
.badge-merged { background: rgba(163,113,247,0.15); }
.labels { display: flex; gap: 4px; margin-top: 6px; flex-wrap: wrap; }
.label {
    font-size: 11px;
    padding: 1px 7px;
    border-radius: 12px;
    background: rgba(56,132,244,0.1);
}
.loading { color: #656d76; font-style: italic; }
.error { color: #cf222e; font-size: 12px; }
.comment {
    border: 1px solid #d1d9e0;
    border-radius: 8px;
    margin-bottom: 12px;
    overflow: hidden;
}
.comment-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    font-size: 13px;
    border-bottom: 1px solid #d1d9e0;
}
@media (prefers-color-scheme: dark) {
    .comment-header { border-bottom-color: #30363d; }
    .meta { color: #9198a1; }
    .meta-row { color: #9198a1; }
}
.comment-author { font-weight: 500; }
.comment-date { color: #656d76; font-size: 12px; }
.comment > .markdown-body { padding: 12px; }

/* Markdown body styles */
.markdown-body {
    word-wrap: break-word;
    overflow-wrap: break-word;
}
.markdown-body > *:first-child { margin-top: 0; }
.markdown-body > *:last-child { margin-bottom: 0; }
.markdown-body p,
.markdown-body blockquote,
.markdown-body ul,
.markdown-body ol,
.markdown-body table,
.markdown-body pre {
    margin-top: 0;
    margin-bottom: 16px;
}
.markdown-body h1 { font-size: 1.5em; font-weight: 600; margin-top: 24px; margin-bottom: 16px; padding-bottom: 0.3em; border-bottom: 1px solid #d1d9e0; }
.markdown-body h2 { font-size: 1.3em; font-weight: 600; margin-top: 24px; margin-bottom: 16px; padding-bottom: 0.3em; border-bottom: 1px solid #d1d9e0; }
.markdown-body h3 { font-size: 1.15em; font-weight: 600; margin-top: 24px; margin-bottom: 16px; }
.markdown-body h4, .markdown-body h5, .markdown-body h6 { font-weight: 600; margin-top: 24px; margin-bottom: 16px; }
.markdown-body code {
    padding: 0.2em 0.4em;
    font-size: 85%;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    background: rgba(175,184,193,0.2);
    border-radius: 6px;
    white-space: break-spaces;
}
.markdown-body pre {
    padding: 16px;
    overflow: auto;
    font-size: 85%;
    line-height: 1.45;
    background: #f6f8fa;
    border-radius: 6px;
}
.markdown-body pre code {
    padding: 0;
    background: transparent;
    font-size: 100%;
    white-space: pre;
}
.markdown-body blockquote {
    padding: 0 1em;
    border-left: 0.25em solid #d1d9e0;
    color: #59636e;
}
.markdown-body ul, .markdown-body ol { padding-left: 2em; }
.markdown-body li + li { margin-top: 0.25em; }
.markdown-body img { max-width: 100%; height: auto; border-radius: 6px; }
.markdown-body table {
    border-spacing: 0;
    border-collapse: collapse;
    width: max-content;
    max-width: 100%;
    overflow: auto;
    display: block;
}
.markdown-body table th,
.markdown-body table td {
    padding: 6px 13px;
    border: 1px solid #d1d9e0;
}
.markdown-body table th { font-weight: 600; }
.markdown-body table tr:nth-child(2n) { background: rgba(175,184,193,0.1); }
.markdown-body hr {
    height: 0.25em;
    padding: 0;
    margin: 24px 0;
    background: #d1d9e0;
    border: 0;
}
.markdown-body input[type="checkbox"] { margin-right: 0.4em; }
"""#

}
