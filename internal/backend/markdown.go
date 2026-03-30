package backend

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// mdRenderer is a shared goldmark instance configured for GitHub-flavored markdown.
var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		html.WithUnsafe(), // Allow raw HTML in markdown (matches GitHub behavior)
	),
)

// renderMarkdown converts a markdown string to HTML.
func renderMarkdown(md string) string {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(md), &buf); err != nil {
		return md // Fall back to raw text on error.
	}
	return buf.String()
}
