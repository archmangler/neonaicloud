package site

import (
	"net/http"
	"strings"
)

func (s *Server) publicBaseURL(r *http.Request) string {
	if base := strings.TrimRight(s.cfg.PublicBaseURL, "/"); base != "" {
		return base
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

func (s *Server) applySEO(r *http.Request, page Page) Page {
	path := page.Path
	if path == "" {
		path = r.URL.Path
	}
	if path == "" {
		path = "/"
	}
	page.Path = path

	base := s.publicBaseURL(r)
	if path == "/" {
		page.CanonicalURL = base + "/"
	} else {
		page.CanonicalURL = base + path
	}

	page.SiteName = "Neon AI Cloud"
	if page.OGType == "" {
		page.OGType = "website"
	}
	if page.OGImage == "" {
		page.OGImage = base + "/static/brand/neoncloudai-logo-simple-hq.png"
	}
	if page.Robots == "" {
		if page.NoIndex {
			page.Robots = "noindex, nofollow"
		} else {
			page.Robots = "index, follow"
		}
	}
	return page
}
