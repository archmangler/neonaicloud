package site

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxMediaBytes = 10 << 20 // 10 MiB

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.Enabled() {
			http.Error(w, "admin is not configured (set ADMIN_USER and ADMIN_PASSWORD)", http.StatusServiceUnavailable)
			return
		}
		if !s.auth.Authenticated(r) {
			nextURL := r.URL.RequestURI()
			http.Redirect(w, r, "/admin/login?next="+url.QueryEscape(nextURL), http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodPost {
			ct := r.Header.Get("Content-Type")
			if strings.HasPrefix(ct, "multipart/form-data") {
				if err := r.ParseMultipartForm(maxMediaBytes); err != nil {
					http.Error(w, "invalid multipart form", http.StatusBadRequest)
					return
				}
			} else if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			if !s.auth.ValidCSRF(r, r.FormValue("csrf")) {
				http.Error(w, "invalid csrf token", http.StatusForbidden)
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) withAdmin(page Page, r *http.Request) Page {
	page.AdminEnabled = s.auth.Enabled()
	page.LoggedIn = s.auth.Authenticated(r)
	page.CSRF = s.auth.CSRFToken(r)
	page.NoIndex = true
	return page
}

func (s *Server) handleAdminLoginGet(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Enabled() {
		http.Error(w, "admin is not configured (set ADMIN_USER and ADMIN_PASSWORD)", http.StatusServiceUnavailable)
		return
	}
	if s.auth.Authenticated(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	page := s.basePage("Admin login — Neon AI Cloud", "CMS login", "")
	page.Next = safeNext(r.URL.Query().Get("next"))
	page.NoIndex = true
	page = s.withAdmin(page, r)
	s.render(w, r, "admin_login.html", page)
}

func (s *Server) handleAdminLoginPost(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Enabled() {
		http.Error(w, "admin is not configured", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	user := strings.TrimSpace(r.FormValue("username"))
	pass := r.FormValue("password")
	next := safeNext(r.FormValue("next"))
	if !s.auth.CheckPassword(user, pass) {
		page := s.basePage("Admin login — Neon AI Cloud", "CMS login", "")
		page.FormError = "Invalid username or password."
		page.Next = next
		page = s.withAdmin(page, r)
		w.WriteHeader(http.StatusUnauthorized)
		s.render(w, r, "admin_login.html", page)
		return
	}
	s.auth.StartSession(w, user)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	s.auth.ClearSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (s *Server) handleAdminHome(w http.ResponseWriter, r *http.Request) {
	page := s.basePage("Admin — Neon AI Cloud", "Product CMS", "")
	page = s.withAdmin(page, r)
	page.Flash = r.URL.Query().Get("flash")
	products, err := s.store.List()
	if err != nil {
		log.Printf("admin list products: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	page.Products = products
	s.render(w, r, "admin_home.html", page)
}

func (s *Server) handleAdminProductNew(w http.ResponseWriter, r *http.Request) {
	page := s.basePage("New product — Admin", "Create product", "")
	page = s.withAdmin(page, r)
	page.IsNewProduct = true
	page.Product = &Product{Status: StatusDraft, Updated: time.Now().UTC().Format("2006-01-02")}
	s.render(w, r, "admin_product_form.html", page)
}

func (s *Server) handleAdminProductCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	p := productFromForm(r)
	if p.Slug == "" {
		p.Slug = slugify(p.Title)
	}
	page := s.basePage("New product — Admin", "Create product", "")
	page = s.withAdmin(page, r)
	page.IsNewProduct = true
	page.Product = &p

	if err := validateProduct(p); err != nil {
		page.FormError = err.Error()
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, r, "admin_product_form.html", page)
		return
	}
	if s.store.Exists(p.Slug) {
		page.FormError = "A product with this slug already exists."
		w.WriteHeader(http.StatusConflict)
		s.render(w, r, "admin_product_form.html", page)
		return
	}
	if err := s.store.Save(p); err != nil {
		page.FormError = err.Error()
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, r, "admin_product_form.html", page)
		return
	}
	http.Redirect(w, r, "/admin?flash="+url.QueryEscape("Created "+p.Slug), http.StatusSeeOther)
}

func (s *Server) handleAdminProductEdit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	p, err := s.store.Get(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	page := s.basePage("Edit "+p.Title+" — Admin", "Edit product", "")
	page = s.withAdmin(page, r)
	page.Product = &p
	page.Flash = r.URL.Query().Get("flash")
	s.render(w, r, "admin_product_form.html", page)
}

func (s *Server) handleAdminProductUpdate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	existing, err := s.store.Get(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p := productFromForm(r)
	p.Slug = existing.Slug // slug is immutable after create
	page := s.basePage("Edit "+existing.Title+" — Admin", "Edit product", "")
	page = s.withAdmin(page, r)
	page.Product = &p

	if err := validateProduct(p); err != nil {
		page.FormError = err.Error()
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, r, "admin_product_form.html", page)
		return
	}
	if err := s.store.Save(p); err != nil {
		page.FormError = err.Error()
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, r, "admin_product_form.html", page)
		return
	}
	http.Redirect(w, r, "/admin/products/"+p.Slug+"/edit?flash="+url.QueryEscape("Saved"), http.StatusSeeOther)
}

func (s *Server) handleAdminProductDelete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := s.store.Delete(slug); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin?flash="+url.QueryEscape("Deleted "+slug), http.StatusSeeOther)
}

func (s *Server) handleAdminMediaGet(w http.ResponseWriter, r *http.Request) {
	page := s.basePage("Media — Admin", "Media library", "")
	page = s.withAdmin(page, r)
	page.Flash = r.URL.Query().Get("flash")
	files, err := s.store.ListMedia()
	if err != nil {
		log.Printf("list media: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	page.MediaFiles = files
	s.render(w, r, "admin_media.html", page)
}

func (s *Server) handleAdminMediaUpload(w http.ResponseWriter, r *http.Request) {
	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, "/admin/media?flash="+url.QueryEscape("No file selected"), http.StatusSeeOther)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxMediaBytes+1))
	if err != nil {
		http.Error(w, "failed to read upload", http.StatusBadRequest)
		return
	}
	if len(data) > maxMediaBytes {
		http.Error(w, "file exceeds 10MB limit", http.StatusRequestEntityTooLarge)
		return
	}
	saved, err := s.store.SaveMedia(hdr.Filename, data)
	if err != nil {
		http.Redirect(w, r, "/admin/media?flash="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/media?flash="+url.QueryEscape("Uploaded "+saved.Path), http.StatusSeeOther)
}

func (s *Server) handleAdminMediaDelete(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimSpace(r.FormValue("path"))
	if err := s.store.DeleteMedia(rel); err != nil {
		http.Redirect(w, r, "/admin/media?flash="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/media?flash="+url.QueryEscape("Deleted "+rel), http.StatusSeeOther)
}

func productFromForm(r *http.Request) Product {
	caps := r.Form["capabilities"]
	if len(caps) == 0 {
		// support comma-separated single field
		caps = splitCSV(r.FormValue("capabilities"))
	}
	status := strings.TrimSpace(r.FormValue("status"))
	if status == "" {
		status = StatusDraft
	}
	return Product{
		Title:        strings.TrimSpace(r.FormValue("title")),
		Slug:         strings.TrimSpace(r.FormValue("slug")),
		Status:       status,
		Summary:      strings.TrimSpace(r.FormValue("summary")),
		Capabilities: caps,
		Updated:      strings.TrimSpace(r.FormValue("updated")),
		Body:         strings.TrimSpace(r.FormValue("body")),
	}
}

func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/admin") || strings.HasPrefix(next, "//") {
		return "/admin"
	}
	return next
}
