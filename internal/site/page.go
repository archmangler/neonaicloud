package site

// NavItem is a primary navigation entry.
type NavItem struct {
	Href  string
	Label string
	Key   string
}

// Page is the shared template context for every public page.
type Page struct {
	Title       string
	Description string
	ActiveNav   string
	Nav         []NavItem
	Capabilities []Capability
	Capability  *Capability
	ContactSent bool
	ContactError string
	FormName    string
	FormEmail   string
	FormOrg     string
	FormMessage string
}

func basePage(title, description, activeNav string) Page {
	return Page{
		Title:       title,
		Description: description,
		ActiveNav:   activeNav,
		Nav: []NavItem{
			{Href: "/capabilities", Label: "Capabilities", Key: "capabilities"},
			{Href: "/approach", Label: "Approach", Key: "approach"},
			{Href: "/about", Label: "About", Key: "about"},
			{Href: "/contact", Label: "Contact", Key: "contact"},
		},
		Capabilities: Capabilities(),
	}
}
