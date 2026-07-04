package site

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed templates/*.html templates/admin/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// Server is the public Neon AI Cloud website and CMS.
type Server struct {
	templates *template.Template
	mux       *http.ServeMux
	static    http.Handler
	media     http.Handler
	auth      *authenticator
	store     *ProductStore
	cfg       Config
}

// New constructs a server with routes, content store, and embedded assets.
func New(cfg Config) (*Server, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"navClass":   navClass,
		"statusChip": statusChip,
		"hasCap":     hasCap,
		"capName":    capabilityName,
		"join":       strings.Join,
	}).ParseFS(templateFS, "templates/*.html", "templates/admin/*.html")
	if err != nil {
		return nil, err
	}

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}

	store := NewProductStore(cfg.ContentDir)
	if err := store.EnsureLayout(); err != nil {
		return nil, err
	}

	s := &Server{
		templates: tmpl,
		mux:       http.NewServeMux(),
		static:    http.FileServer(http.FS(staticRoot)),
		media:     http.StripPrefix("/media/", http.FileServer(http.Dir(store.mediaDir()))),
		auth:      newAuthenticator(cfg),
		store:     store,
		cfg:       cfg,
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", s.static))
	s.mux.Handle("GET /media/", s.media)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)

	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /capabilities", s.handleCapabilities)
	s.mux.HandleFunc("GET /capabilities/{slug}", s.handleCapability)
	s.mux.HandleFunc("GET /products", s.handleProducts)
	s.mux.HandleFunc("GET /products/{slug}", s.handleProduct)
	s.mux.HandleFunc("GET /approach", s.handleApproach)
	s.mux.HandleFunc("GET /about", s.handleAbout)
	s.mux.HandleFunc("GET /contact", s.handleContactGet)
	s.mux.HandleFunc("POST /contact", s.handleContactPost)
	s.mux.HandleFunc("GET /contact/digital-twin", s.handleDigitalTwin)

	s.mux.HandleFunc("GET /admin/login", s.handleAdminLoginGet)
	s.mux.HandleFunc("POST /admin/login", s.handleAdminLoginPost)
	s.mux.HandleFunc("POST /admin/logout", s.requireAdmin(s.handleAdminLogout))
	s.mux.HandleFunc("GET /admin", s.requireAdmin(s.handleAdminHome))
	s.mux.HandleFunc("GET /admin/{$}", s.requireAdmin(s.handleAdminHome))
	s.mux.HandleFunc("GET /admin/products/new", s.requireAdmin(s.handleAdminProductNew))
	s.mux.HandleFunc("POST /admin/products", s.requireAdmin(s.handleAdminProductCreate))
	s.mux.HandleFunc("GET /admin/products/{slug}/edit", s.requireAdmin(s.handleAdminProductEdit))
	s.mux.HandleFunc("POST /admin/products/{slug}", s.requireAdmin(s.handleAdminProductUpdate))
	s.mux.HandleFunc("POST /admin/products/{slug}/delete", s.requireAdmin(s.handleAdminProductDelete))
	s.mux.HandleFunc("GET /admin/media", s.requireAdmin(s.handleAdminMediaGet))
	s.mux.HandleFunc("POST /admin/media", s.requireAdmin(s.handleAdminMediaUpload))
	s.mux.HandleFunc("POST /admin/media/delete", s.requireAdmin(s.handleAdminMediaDelete))
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	page := s.basePage(
		"Neon AI Cloud",
		"Premium AI application development, platform engineering, AI infrastructure, embedded systems, and cloud.",
		"home",
	)
	products, err := s.store.ListPublished()
	if err != nil {
		log.Printf("list products: %v", err)
	} else if len(products) > 3 {
		page.Products = products[:3]
	} else {
		page.Products = products
	}
	s.render(w, "home.html", page)
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	page := s.basePage(
		"Capabilities — Neon AI Cloud",
		"Service lines spanning AI applications, platforms, infrastructure, embedded systems, and cloud.",
		"capabilities",
	)
	s.render(w, "capabilities.html", page)
}

func (s *Server) handleCapability(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	cap, ok := CapabilityBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	page := s.basePage(cap.Name+" — Neon AI Cloud", cap.Outcome, "capabilities")
	page.Capability = &cap
	products, err := s.store.ListPublished()
	if err != nil {
		log.Printf("list products: %v", err)
	} else {
		var related []Product
		for _, p := range products {
			for _, c := range p.Capabilities {
				if c == slug {
					related = append(related, p)
					break
				}
			}
		}
		page.Products = related
	}
	s.render(w, "capability.html", page)
}

func (s *Server) handleProducts(w http.ResponseWriter, r *http.Request) {
	page := s.basePage(
		"Products — Neon AI Cloud",
		"Commercial offerings across AI applications, platforms, infrastructure, embedded systems, and cloud.",
		"products",
	)
	filter := strings.TrimSpace(r.URL.Query().Get("capability"))
	page.CapabilityFilter = filter
	products, err := s.store.ListPublished()
	if err != nil {
		log.Printf("list products: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if filter != "" {
		var filtered []Product
		for _, p := range products {
			for _, c := range p.Capabilities {
				if c == filter {
					filtered = append(filtered, p)
					break
				}
			}
		}
		products = filtered
	}
	page.Products = products
	s.render(w, "products.html", page)
}

func (s *Server) handleProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	p, err := s.store.Get(slug)
	if err != nil || !p.Published() {
		http.NotFound(w, r)
		return
	}
	page := s.basePage(p.Title+" — Neon AI Cloud", p.Summary, "products")
	page.Product = &p
	page.ProductBody = RenderMarkdown(p.Body)
	s.render(w, "product.html", page)
}

func (s *Server) handleApproach(w http.ResponseWriter, r *http.Request) {
	page := s.basePage(
		"Approach — Neon AI Cloud",
		"How Neon AI Cloud designs, implements, and validates infrastructure and platforms.",
		"approach",
	)
	s.render(w, "approach.html", page)
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	page := s.basePage(
		"About — Neon AI Cloud",
		"An engineering-led company specialising in AI infrastructure and the autonomous enterprise.",
		"about",
	)
	s.render(w, "about.html", page)
}

func (s *Server) handleContactGet(w http.ResponseWriter, r *http.Request) {
	page := s.basePage(
		"Contact — Neon AI Cloud",
		"Engage Neon AI Cloud on infrastructure, platforms, applications, embedded systems, or cloud.",
		"contact",
	)
	s.render(w, "contact.html", page)
}

func (s *Server) handleDigitalTwin(w http.ResponseWriter, r *http.Request) {
	page := s.basePage(
		"C-suite digital twin — Neon AI Cloud",
		"Talk with a digital twin of the Neon AI Cloud C-suite. Agentic chatbot integration forthcoming.",
		"contact",
	)
	s.render(w, "digital-twin.html", page)
}

func (s *Server) handleContactPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	page := s.basePage(
		"Contact — Neon AI Cloud",
		"Engage Neon AI Cloud on infrastructure, platforms, applications, embedded systems, or cloud.",
		"contact",
	)
	page.FormName = strings.TrimSpace(r.FormValue("name"))
	page.FormEmail = strings.TrimSpace(r.FormValue("email"))
	page.FormOrg = strings.TrimSpace(r.FormValue("organisation"))
	page.FormMessage = strings.TrimSpace(r.FormValue("message"))

	if page.FormName == "" || page.FormEmail == "" || page.FormMessage == "" {
		page.ContactError = "Name, email, and message are required."
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, "contact.html", page)
		return
	}

	log.Printf("contact enquiry from %q <%s> org=%q", page.FormName, page.FormEmail, page.FormOrg)
	page.ContactSent = true
	page.FormName = ""
	page.FormEmail = ""
	page.FormOrg = ""
	page.FormMessage = ""
	s.render(w, "contact.html", page)
}

func (s *Server) basePage(title, description, activeNav string) Page {
	return Page{
		Title:       title,
		Description: description,
		ActiveNav:   activeNav,
		Nav: []NavItem{
			{Href: "/capabilities", Label: "Capabilities", Key: "capabilities"},
			{Href: "/products", Label: "Products", Key: "products"},
			{Href: "/approach", Label: "Approach", Key: "approach"},
			{Href: "/about", Label: "About", Key: "about"},
			{Href: "/contact", Label: "Contact", Key: "contact"},
		},
		Capabilities: Capabilities(),
		AdminEnabled: s.auth.Enabled(),
	}
}

func (s *Server) render(w http.ResponseWriter, name string, page Page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, page); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func navClass(active, key string) string {
	if active == key {
		return "nav-link nav-link-active"
	}
	return "nav-link"
}

func statusChip(status string) string {
	if status == StatusPublished {
		return "chip status-chip status-published"
	}
	return "chip status-chip status-draft"
}

func hasCap(caps []string, slug string) bool {
	for _, c := range caps {
		if c == slug {
			return true
		}
	}
	return false
}

func capabilityName(slug string) string {
	if c, ok := CapabilityBySlug(slug); ok {
		return c.Name
	}
	return slug
}

// ContentDir returns the configured content directory (for diagnostics).
func (s *Server) ContentDir() string {
	return filepath.Clean(s.cfg.ContentDir)
}
