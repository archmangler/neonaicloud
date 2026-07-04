package site

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// Server is the public Neon AI Cloud website.
type Server struct {
	templates *template.Template
	mux       *http.ServeMux
	static    http.Handler
}

// New constructs a server with routes and embedded assets.
func New() (*Server, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"navClass": navClass,
	}).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}

	s := &Server{
		templates: tmpl,
		mux:       http.NewServeMux(),
		static:    http.FileServer(http.FS(staticRoot)),
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", s.static))
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /capabilities", s.handleCapabilities)
	s.mux.HandleFunc("GET /capabilities/{slug}", s.handleCapability)
	s.mux.HandleFunc("GET /approach", s.handleApproach)
	s.mux.HandleFunc("GET /about", s.handleAbout)
	s.mux.HandleFunc("GET /contact", s.handleContactGet)
	s.mux.HandleFunc("POST /contact", s.handleContactPost)
	s.mux.HandleFunc("GET /contact/digital-twin", s.handleDigitalTwin)
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
	page := basePage(
		"Neon AI Cloud",
		"Premium AI application development, platform engineering, AI infrastructure, embedded systems, and cloud.",
		"home",
	)
	s.render(w, "home.html", page)
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	page := basePage(
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
	page := basePage(
		cap.Name+" — Neon AI Cloud",
		cap.Outcome,
		"capabilities",
	)
	page.Capability = &cap
	s.render(w, "capability.html", page)
}

func (s *Server) handleApproach(w http.ResponseWriter, r *http.Request) {
	page := basePage(
		"Approach — Neon AI Cloud",
		"How Neon AI Cloud designs, implements, and validates infrastructure and platforms.",
		"approach",
	)
	s.render(w, "approach.html", page)
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	page := basePage(
		"About — Neon AI Cloud",
		"An engineering-led company specialising in AI infrastructure and the autonomous enterprise.",
		"about",
	)
	s.render(w, "about.html", page)
}

func (s *Server) handleContactGet(w http.ResponseWriter, r *http.Request) {
	page := basePage(
		"Contact — Neon AI Cloud",
		"Engage Neon AI Cloud on infrastructure, platforms, applications, embedded systems, or cloud.",
		"contact",
	)
	s.render(w, "contact.html", page)
}

func (s *Server) handleDigitalTwin(w http.ResponseWriter, r *http.Request) {
	page := basePage(
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

	page := basePage(
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

	// Phase 1 acknowledges receipt only. Delivery (mailer/webhook) is later.
	log.Printf("contact enquiry from %q <%s> org=%q", page.FormName, page.FormEmail, page.FormOrg)
	page.ContactSent = true
	page.FormName = ""
	page.FormEmail = ""
	page.FormOrg = ""
	page.FormMessage = ""
	s.render(w, "contact.html", page)
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
