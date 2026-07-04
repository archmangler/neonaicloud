package site

import (
	"html"
	"html/template"
	"regexp"
	"strings"
)

var (
	reBold = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

// RenderMarkdown converts a small Markdown subset to safe HTML.
// Supports paragraphs, # / ## headings, - lists, **bold**, and [text](url).
func RenderMarkdown(src string) template.HTML {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}

	var b strings.Builder
	lines := strings.Split(src, "\n")
	inList := false
	var para []string

	flushPara := func() {
		if len(para) == 0 {
			return
		}
		text := strings.TrimSpace(strings.Join(para, " "))
		if text != "" {
			b.WriteString("<p>")
			b.WriteString(inlineMarkdown(text))
			b.WriteString("</p>\n")
		}
		para = para[:0]
	}

	closeList := func() {
		if inList {
			b.WriteString("</ul>\n")
			inList = false
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			flushPara()
			closeList()
		case strings.HasPrefix(trimmed, "## "):
			flushPara()
			closeList()
			b.WriteString("<h3>")
			b.WriteString(inlineMarkdown(strings.TrimSpace(trimmed[3:])))
			b.WriteString("</h3>\n")
		case strings.HasPrefix(trimmed, "# "):
			flushPara()
			closeList()
			b.WriteString("<h2>")
			b.WriteString(inlineMarkdown(strings.TrimSpace(trimmed[2:])))
			b.WriteString("</h2>\n")
		case strings.HasPrefix(trimmed, "- "):
			flushPara()
			if !inList {
				b.WriteString("<ul>\n")
				inList = true
			}
			b.WriteString("<li>")
			b.WriteString(inlineMarkdown(strings.TrimSpace(trimmed[2:])))
			b.WriteString("</li>\n")
		default:
			closeList()
			para = append(para, trimmed)
		}
	}
	flushPara()
	closeList()
	return template.HTML(b.String())
}

func inlineMarkdown(text string) string {
	escaped := html.EscapeString(text)
	escaped = reBold.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = reLink.ReplaceAllStringFunc(escaped, func(m string) string {
		parts := reLink.FindStringSubmatch(m)
		if len(parts) != 3 {
			return m
		}
		label := parts[1]
		href := parts[2]
		if !safeLink(href) {
			return label
		}
		return `<a href="` + href + `">` + label + `</a>`
	})
	return escaped
}

func safeLink(href string) bool {
	h := strings.ToLower(strings.TrimSpace(href))
	return strings.HasPrefix(h, "http://") ||
		strings.HasPrefix(h, "https://") ||
		strings.HasPrefix(h, "/") ||
		strings.HasPrefix(h, "#")
}
