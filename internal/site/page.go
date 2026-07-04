package site

import "html/template"

// NavItem is a primary navigation entry.
type NavItem struct {
	Href  string
	Label string
	Key   string
}

// Page is the shared template context for public and admin pages.
type Page struct {
	Title        string
	Description  string
	ActiveNav    string
	Nav          []NavItem
	Capabilities []Capability
	Capability   *Capability

	Products         []Product
	Product          *Product
	ProductBody      template.HTML
	CapabilityFilter string

	ContactSent  bool
	ContactError string
	FormName     string
	FormEmail    string
	FormOrg      string
	FormMessage  string

	AdminEnabled bool
	LoggedIn     bool
	Flash        string
	FormError    string
	CSRF         string
	Next         string
	MediaFiles   []MediaFile
	IsNewProduct bool

	// SEO
	Path         string
	CanonicalURL string
	SiteName     string
	OGType       string
	OGImage      string
	Robots       string
	NoIndex      bool
}
