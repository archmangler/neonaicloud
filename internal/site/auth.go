package site

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	sessionCookie = "neon_admin_session"
	csrfCookie    = "neon_admin_csrf"
	sessionTTL    = 12 * time.Hour
)

type authenticator struct {
	user     string
	password string
	secret   []byte
	enabled  bool
}

func newAuthenticator(cfg Config) *authenticator {
	if !cfg.AdminEnabled() {
		return &authenticator{enabled: false}
	}
	return &authenticator{
		user:     cfg.AdminUser,
		password: cfg.AdminPassword,
		secret:   []byte(cfg.SessionSecret),
		enabled:  true,
	}
}

func (a *authenticator) Enabled() bool { return a.enabled }

func (a *authenticator) CheckPassword(user, pass string) bool {
	if !a.enabled {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(a.user)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(a.password)) == 1
	return userOK && passOK
}

func (a *authenticator) Authenticated(r *http.Request) bool {
	if !a.enabled {
		return false
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return false
	}
	user, exp, err := a.parseSession(c.Value)
	if err != nil {
		return false
	}
	if time.Now().After(exp) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(user), []byte(a.user)) == 1
}

func (a *authenticator) StartSession(w http.ResponseWriter, user string) {
	exp := time.Now().Add(sessionTTL)
	token := a.signSession(user, exp)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	csrf := a.signCSRF(user, exp)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    csrf,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func (a *authenticator) ClearSession(w http.ResponseWriter) {
	clear := func(name string) {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}
	clear(sessionCookie)
	clear(csrfCookie)
}

func (a *authenticator) CSRFToken(r *http.Request) string {
	c, err := r.Cookie(csrfCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func (a *authenticator) ValidCSRF(r *http.Request, token string) bool {
	if !a.enabled || token == "" {
		return false
	}
	c, err := r.Cookie(csrfCookie)
	if err != nil {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(c.Value)) != 1 {
		return false
	}
	// Token must still match an active session user.
	sc, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	user, exp, err := a.parseSession(sc.Value)
	if err != nil || time.Now().After(exp) {
		return false
	}
	expected := a.signCSRF(user, exp)
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func (a *authenticator) signSession(user string, exp time.Time) string {
	payload := user + "|" + strconv.FormatInt(exp.Unix(), 10)
	sig := a.mac(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
}

func (a *authenticator) parseSession(token string) (string, time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", time.Time{}, errAuth
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", time.Time{}, errAuth
	}
	payload := string(raw)
	if subtle.ConstantTimeCompare([]byte(a.mac(payload)), []byte(parts[1])) != 1 {
		return "", time.Time{}, errAuth
	}
	user, expStr, ok := strings.Cut(payload, "|")
	if !ok {
		return "", time.Time{}, errAuth
	}
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return "", time.Time{}, errAuth
	}
	return user, time.Unix(expUnix, 0), nil
}

func (a *authenticator) signCSRF(user string, exp time.Time) string {
	payload := "csrf|" + user + "|" + strconv.FormatInt(exp.Unix(), 10)
	return a.mac(payload)
}

func (a *authenticator) mac(payload string) string {
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

type authError struct{}

func (authError) Error() string { return "invalid session" }

var errAuth authError
