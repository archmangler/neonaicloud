package site

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBlogRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewProductStore(dir)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	in := BlogPost{
		Title:   "Factory Architect",
		Slug:    "neon-ai-factory-architect",
		Status:  StatusPublished,
		Summary: "Public release announcement",
		Updated: "2026-08-30",
		Body:    "# Hello\n\n![diagram](/media/blogs/test.jpg)",
	}
	raw := "---\ntitle: " + in.Title + "\nslug: " + in.Slug + "\nstatus: " + in.Status +
		"\nsummary: " + in.Summary + "\nupdated: " + in.Updated + "\n---\n\n" + in.Body
	path := filepath.Join(dir, "blogs", in.Slug+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := store.GetBlog(in.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != in.Title || !out.Published() {
		t.Fatalf("unexpected: %+v", out)
	}

	published, err := store.ListPublishedBlogs()
	if err != nil {
		t.Fatal(err)
	}
	if len(published) != 1 {
		t.Fatalf("published=%d", len(published))
	}
}

func TestRenderMarkdownImage(t *testing.T) {
	html := string(RenderMarkdown("![Alt text](/media/blogs/example.jpg)"))
	for _, want := range []string{`<figure class="prose-figure">`, `src="/media/blogs/example.jpg"`, `<figcaption>Alt text</figcaption>`} {
		if !contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
