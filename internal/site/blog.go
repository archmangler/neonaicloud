package site

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BlogPost is a CMS-managed article under content/blogs.
type BlogPost struct {
	Title   string
	Slug    string
	Status  string
	Summary string
	Updated string
	Body    string
}

// Published reports whether the post is public.
func (b BlogPost) Published() bool {
	return b.Status == StatusPublished
}

func (s *ProductStore) blogsDir() string { return filepath.Join(s.root, "blogs") }

// ListBlogs returns all blog posts, newest updated first.
func (s *ProductStore) ListBlogs() ([]BlogPost, error) {
	entries, err := os.ReadDir(s.blogsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []BlogPost
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := s.loadBlogFile(filepath.Join(s.blogsDir(), e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Updated == out[j].Updated {
			return out[i].Title < out[j].Title
		}
		return out[i].Updated > out[j].Updated
	})
	return out, nil
}

// ListPublishedBlogs returns published posts only.
func (s *ProductStore) ListPublishedBlogs() ([]BlogPost, error) {
	all, err := s.ListBlogs()
	if err != nil {
		return nil, err
	}
	var out []BlogPost
	for _, b := range all {
		if b.Published() {
			out = append(out, b)
		}
	}
	return out, nil
}

// GetBlog loads a post by slug.
func (s *ProductStore) GetBlog(slug string) (BlogPost, error) {
	if !ValidSlug(slug) {
		return BlogPost{}, os.ErrNotExist
	}
	return s.loadBlogFile(s.blogPath(slug))
}

func (s *ProductStore) blogPath(slug string) string {
	return filepath.Join(s.blogsDir(), slug+".md")
}

func (s *ProductStore) loadBlogFile(path string) (BlogPost, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return BlogPost{}, err
	}
	b, err := parseBlog(string(raw))
	if err != nil {
		return BlogPost{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if b.Slug == "" {
		b.Slug = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	return b, nil
}

func parseBlog(raw string) (BlogPost, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return BlogPost{}, fmt.Errorf("missing front matter")
	}
	rest := raw[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			end = len(rest) - 4
		} else {
			return BlogPost{}, fmt.Errorf("unterminated front matter")
		}
	}
	meta := rest[:end]
	body := ""
	if end+5 <= len(rest) {
		body = strings.TrimSpace(rest[end+5:])
	}

	b := BlogPost{Body: body, Status: StatusDraft}
	for _, line := range strings.Split(meta, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "title":
			b.Title = val
		case "slug":
			b.Slug = val
		case "status":
			b.Status = val
		case "summary":
			b.Summary = val
		case "updated":
			b.Updated = val
		}
	}
	if b.Updated == "" {
		b.Updated = time.Now().UTC().Format("2006-01-02")
	}
	return b, nil
}
