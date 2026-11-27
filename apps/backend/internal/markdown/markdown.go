package markdown

import (
	"bytes"
	"sync"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	once     sync.Once
	markdown goldmark.Markdown
	policy   *bluemonday.Policy
)

func initRenderer() {
	markdown = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)
	policy = bluemonday.UGCPolicy()
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("target", "rel").OnElements("a")
}

// Render converts Markdown to sanitized HTML. Raw HTML is escaped by goldmark
// and the result is additionally sanitized by bluemonday.
func Render(source string) string {
	once.Do(initRenderer)
	var buf bytes.Buffer
	if err := markdown.Convert([]byte(source), &buf); err != nil {
		return ""
	}
	return policy.Sanitize(buf.String())
}
