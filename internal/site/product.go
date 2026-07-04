package site

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Product is a CMS-managed commercial offering.
type Product struct {
	Title        string
	Slug         string
	Status       string
	Summary      string
	Capabilities []string
	Updated      string
	Body         string
}

// Published reports whether the product is public.
func (p Product) Published() bool {
	return p.Status == StatusPublished
}

// ProductStore persists products and media under a content directory.
type ProductStore struct {
	root string
}

// NewProductStore creates a store rooted at contentDir.
func NewProductStore(contentDir string) *ProductStore {
	return &ProductStore{root: contentDir}
}

func (s *ProductStore) productsDir() string { return filepath.Join(s.root, "products") }
func (s *ProductStore) mediaDir() string    { return filepath.Join(s.root, "media") }

// EnsureLayout creates content directories if missing.
func (s *ProductStore) EnsureLayout() error {
	for _, dir := range []string{s.root, s.productsDir(), s.mediaDir(), filepath.Join(s.mediaDir(), "products")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// List returns all products, newest updated first.
func (s *ProductStore) List() ([]Product, error) {
	entries, err := os.ReadDir(s.productsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Product
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p, err := s.loadFile(filepath.Join(s.productsDir(), e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Updated == out[j].Updated {
			return out[i].Title < out[j].Title
		}
		return out[i].Updated > out[j].Updated
	})
	return out, nil
}

// ListPublished returns published products only.
func (s *ProductStore) ListPublished() ([]Product, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	var out []Product
	for _, p := range all {
		if p.Published() {
			out = append(out, p)
		}
	}
	return out, nil
}

// Get loads a product by slug.
func (s *ProductStore) Get(slug string) (Product, error) {
	if !ValidSlug(slug) {
		return Product{}, os.ErrNotExist
	}
	return s.loadFile(s.productPath(slug))
}

// Save writes a product file. Slug is taken from p.Slug.
func (s *ProductStore) Save(p Product) error {
	if err := validateProduct(p); err != nil {
		return err
	}
	if p.Updated == "" {
		p.Updated = time.Now().UTC().Format("2006-01-02")
	}
	if err := s.EnsureLayout(); err != nil {
		return err
	}
	return os.WriteFile(s.productPath(p.Slug), []byte(encodeProduct(p)), 0o644)
}

// Delete removes a product by slug.
func (s *ProductStore) Delete(slug string) error {
	if !ValidSlug(slug) {
		return fmt.Errorf("invalid slug")
	}
	err := os.Remove(s.productPath(slug))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Exists reports whether a product file exists.
func (s *ProductStore) Exists(slug string) bool {
	if !ValidSlug(slug) {
		return false
	}
	_, err := os.Stat(s.productPath(slug))
	return err == nil
}

func (s *ProductStore) productPath(slug string) string {
	return filepath.Join(s.productsDir(), slug+".md")
}

func (s *ProductStore) loadFile(path string) (Product, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Product{}, err
	}
	p, err := parseProduct(string(raw))
	if err != nil {
		return Product{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if p.Slug == "" {
		p.Slug = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	return p, nil
}

// ValidSlug reports whether slug is URL-safe.
func ValidSlug(slug string) bool {
	return slugPattern.MatchString(slug)
}

// Slugify converts a title into a slug candidate.
func slugify(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	lastDash := false
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func validateProduct(p Product) error {
	p.Title = strings.TrimSpace(p.Title)
	p.Slug = strings.TrimSpace(p.Slug)
	p.Status = strings.TrimSpace(p.Status)
	p.Summary = strings.TrimSpace(p.Summary)
	if p.Title == "" {
		return fmt.Errorf("title is required")
	}
	if !ValidSlug(p.Slug) {
		return fmt.Errorf("slug must be lowercase letters, numbers, and hyphens")
	}
	if p.Status != StatusDraft && p.Status != StatusPublished {
		return fmt.Errorf("status must be draft or published")
	}
	return nil
}

func parseProduct(raw string) (Product, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return Product{}, fmt.Errorf("missing front matter")
	}
	rest := raw[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		// allow trailing --- at EOF
		if strings.HasSuffix(rest, "\n---") {
			end = len(rest) - 4
		} else {
			return Product{}, fmt.Errorf("unterminated front matter")
		}
	}
	meta := rest[:end]
	body := ""
	if end+5 <= len(rest) {
		body = strings.TrimSpace(rest[end+5:])
	} else if strings.HasSuffix(rest, "\n---") {
		body = ""
	}

	p := Product{Body: body, Status: StatusDraft}
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
			p.Title = val
		case "slug":
			p.Slug = val
		case "status":
			p.Status = val
		case "summary":
			p.Summary = val
		case "updated":
			p.Updated = val
		case "capabilities":
			p.Capabilities = splitCSV(val)
		}
	}
	return p, nil
}

func encodeProduct(p Product) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: " + p.Title + "\n")
	b.WriteString("slug: " + p.Slug + "\n")
	b.WriteString("status: " + p.Status + "\n")
	b.WriteString("summary: " + p.Summary + "\n")
	b.WriteString("capabilities: " + strings.Join(p.Capabilities, ", ") + "\n")
	b.WriteString("updated: " + p.Updated + "\n")
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(p.Body))
	if p.Body != "" && !strings.HasSuffix(p.Body, "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}

func splitCSV(val string) []string {
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// MediaFile is an uploaded asset under content/media.
type MediaFile struct {
	Name string
	Path string // URL path /media/...
	Size int64
}

// ListMedia returns files under content/media recursively.
func (s *ProductStore) ListMedia() ([]MediaFile, error) {
	root := s.mediaDir()
	var out []MediaFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		urlPath := "/media/" + filepath.ToSlash(rel)
		out = append(out, MediaFile{Name: rel, Path: urlPath, Size: info.Size()})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SaveMedia writes an uploaded file under media/products.
func (s *ProductStore) SaveMedia(filename string, data []byte) (MediaFile, error) {
	name, err := sanitizeFilename(filename)
	if err != nil {
		return MediaFile{}, err
	}
	dir := filepath.Join(s.mediaDir(), "products")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return MediaFile{}, err
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return MediaFile{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return MediaFile{}, err
	}
	return MediaFile{
		Name: filepath.ToSlash(filepath.Join("products", name)),
		Path: "/media/products/" + name,
		Size: info.Size(),
	}, nil
}

// DeleteMedia removes a media file by relative path under media/.
func (s *ProductStore) DeleteMedia(rel string) error {
	rel = filepath.Clean(rel)
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("invalid media path")
	}
	path := filepath.Join(s.mediaDir(), rel)
	// Ensure path stays under media dir.
	mediaAbs, err := filepath.Abs(s.mediaDir())
	if err != nil {
		return err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(pathAbs, mediaAbs+string(os.PathSeparator)) && pathAbs != mediaAbs {
		return fmt.Errorf("invalid media path")
	}
	return os.Remove(path)
}

func sanitizeFilename(name string) (string, error) {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("invalid filename")
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" || out == "." || strings.HasPrefix(out, ".") {
		return "", fmt.Errorf("invalid filename")
	}
	return out, nil
}
