package site

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	pushoverEndpoint    = "https://api.pushover.net/1/messages.json"
	pushoverMaxMessage  = 900 // leave headroom under Pushover's 1024 limit
	pushoverHTTPTimeout = 12 * time.Second
)

// NotifyContactEnquiry delivers a contact-form enquiry via Pushover.
func (s *Server) NotifyContactEnquiry(name, email, org, message string) error {
	token := strings.TrimSpace(s.cfg.PushoverToken)
	user := strings.TrimSpace(s.cfg.PushoverUser)
	if token == "" || user == "" {
		return fmt.Errorf("contact notifications are not configured (set PUSHOVER_TOKEN and PUSHOVER_USER)")
	}

	body := formatContactPushoverMessage(name, email, org, message)
	form := url.Values{}
	form.Set("token", token)
	form.Set("user", user)
	form.Set("title", "Neon AI Cloud enquiry")
	form.Set("message", body)
	form.Set("priority", "0")

	client := &http.Client{Timeout: pushoverHTTPTimeout}
	resp, err := client.PostForm(pushoverEndpoint, form)
	if err != nil {
		return fmt.Errorf("pushover request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushover returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func formatContactPushoverMessage(name, email, org, message string) string {
	var b strings.Builder
	b.WriteString("New website enquiry\n")
	b.WriteString("Name: ")
	b.WriteString(name)
	b.WriteString("\nEmail: ")
	b.WriteString(email)
	if strings.TrimSpace(org) != "" {
		b.WriteString("\nOrganisation: ")
		b.WriteString(org)
	}
	b.WriteString("\n\n")
	b.WriteString(strings.TrimSpace(message))

	out := b.String()
	if len(out) <= pushoverMaxMessage {
		return out
	}
	const marker = "..."
	keep := pushoverMaxMessage - len(marker)
	if keep < 1 {
		keep = 1
	}
	return out[:keep] + marker
}

func (s *Server) handleContactPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	page := s.twinPage(
		"Contact — Neon AI Cloud",
		"Engage Neon AI Cloud on infrastructure, platforms, applications, embedded systems, or cloud.",
		"contact",
		r.URL.Query().Get("persona"),
		true,
	)
	page.FormName = strings.TrimSpace(r.FormValue("name"))
	page.FormEmail = strings.TrimSpace(r.FormValue("email"))
	page.FormOrg = strings.TrimSpace(r.FormValue("organisation"))
	page.FormMessage = strings.TrimSpace(r.FormValue("message"))

	if page.FormName == "" || page.FormEmail == "" || page.FormMessage == "" {
		page.ContactError = "Name, email, and message are required."
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, r, "contact.html", page)
		return
	}

	log.Printf("contact enquiry from %q <%s> org=%q", page.FormName, page.FormEmail, page.FormOrg)
	if err := s.NotifyContactEnquiry(page.FormName, page.FormEmail, page.FormOrg, page.FormMessage); err != nil {
		log.Printf("contact notify: %v", err)
		page.ContactError = "Unable to deliver your enquiry right now. Please try again shortly."
		w.WriteHeader(http.StatusBadGateway)
		s.render(w, r, "contact.html", page)
		return
	}

	page.ContactSent = true
	page.FormName = ""
	page.FormEmail = ""
	page.FormOrg = ""
	page.FormMessage = ""
	s.render(w, r, "contact.html", page)
}
