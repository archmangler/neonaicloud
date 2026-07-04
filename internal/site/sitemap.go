package site

import (
	"encoding/xml"
	"log"
	"net/http"
	"time"
)

type urlSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	base := s.publicBaseURL(r)
	today := time.Now().UTC().Format("2006-01-02")

	entries := []urlEntry{
		{Loc: base + "/", ChangeFreq: "weekly", Priority: "1.0", LastMod: today},
		{Loc: base + "/capabilities", ChangeFreq: "monthly", Priority: "0.9", LastMod: today},
		{Loc: base + "/products", ChangeFreq: "weekly", Priority: "0.9", LastMod: today},
		{Loc: base + "/approach", ChangeFreq: "monthly", Priority: "0.8", LastMod: today},
		{Loc: base + "/blogs", ChangeFreq: "weekly", Priority: "0.7", LastMod: today},
		{Loc: base + "/about", ChangeFreq: "monthly", Priority: "0.7", LastMod: today},
		{Loc: base + "/contact", ChangeFreq: "monthly", Priority: "0.7", LastMod: today},
	}

	for _, c := range Capabilities() {
		entries = append(entries, urlEntry{
			Loc:        base + "/capabilities/" + c.Slug,
			ChangeFreq: "monthly",
			Priority:   "0.8",
			LastMod:    today,
		})
	}

	products, err := s.store.ListPublished()
	if err != nil {
		log.Printf("sitemap products: %v", err)
	} else {
		for _, p := range products {
			lastMod := p.Updated
			if lastMod == "" {
				lastMod = today
			}
			entries = append(entries, urlEntry{
				Loc:        base + "/products/" + p.Slug,
				ChangeFreq: "weekly",
				Priority:   "0.8",
				LastMod:    lastMod,
			})
		}
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(urlSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  entries,
	}); err != nil {
		log.Printf("sitemap encode: %v", err)
	}
}

func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	base := s.publicBaseURL(r)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(
		"User-agent: *\n" +
			"Allow: /\n" +
			"Disallow: /admin\n" +
			"Disallow: /admin/\n" +
			"\n" +
			"Sitemap: " + base + "/sitemap.xml\n",
	))
}
