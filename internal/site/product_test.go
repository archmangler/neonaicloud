package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewProductStore(dir)
	if err := store.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	in := Product{
		Title:        "Test Product",
		Slug:         "test-product",
		Status:       StatusPublished,
		Summary:      "A summary",
		Capabilities: []string{"cloud", "embedded-systems"},
		Updated:      "2026-07-04",
		Body:         "# Hello\n\nBody **bold** and [link](/products).",
	}
	if err := store.Save(in); err != nil {
		t.Fatal(err)
	}

	out, err := store.Get("test-product")
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != in.Title || out.Status != in.Status || out.Summary != in.Summary {
		t.Fatalf("mismatch: %+v", out)
	}
	if len(out.Capabilities) != 2 || out.Capabilities[0] != "cloud" {
		t.Fatalf("capabilities: %#v", out.Capabilities)
	}
	if out.Body == "" {
		t.Fatal("empty body")
	}

	published, err := store.ListPublished()
	if err != nil {
		t.Fatal(err)
	}
	if len(published) != 1 {
		t.Fatalf("published=%d", len(published))
	}

	in.Status = StatusDraft
	if err := store.Save(in); err != nil {
		t.Fatal(err)
	}
	published, err = store.ListPublished()
	if err != nil {
		t.Fatal(err)
	}
	if len(published) != 0 {
		t.Fatalf("expected no published, got %d", len(published))
	}

	path := filepath.Join(dir, "products", "test-product.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRenderMarkdown(t *testing.T) {
	html := string(RenderMarkdown("# Title\n\nPara **x** and [a](/b).\n\n- one\n- two"))
	for _, want := range []string{"<h2>", "<strong>x</strong>", `<a href="/b">a</a>`, "<li>"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}
