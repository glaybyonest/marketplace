package security

import (
	"net/http"
	"strings"
	"time"
)

const (
	DefaultAccessCookieName  = "mp_access_token"
	DefaultRefreshCookieName = "mp_refresh_token"
	DefaultCSRFCookieName    = "mp_csrf_token"
)

type CookieAuthConfig struct {
	Enabled       bool
	Secure        bool
	Domain        string
	SameSite      http.SameSite
	AccessCookie  string
	RefreshCookie string
	CSRFCookie    string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func NewCookieAuthConfig(enabled, secure bool, domain, sameSite string, accessTTL, refreshTTL time.Duration) CookieAuthConfig {
	return CookieAuthConfig{
		Enabled:       enabled,
		Secure:        secure,
		Domain:        strings.TrimSpace(domain),
		SameSite:      parseSameSite(sameSite),
		AccessCookie:  DefaultAccessCookieName,
		RefreshCookie: DefaultRefreshCookieName,
		CSRFCookie:    DefaultCSRFCookieName,
		AccessTTL:     accessTTL,
		RefreshTTL:    refreshTTL,
	}
}

func (c CookieAuthConfig) AccessToken(r *http.Request) string {
	return c.readCookie(r, c.AccessCookie)
}

func (c CookieAuthConfig) RefreshToken(r *http.Request) string {
	return c.readCookie(r, c.RefreshCookie)
}

func (c CookieAuthConfig) CSRFToken(r *http.Request) string {
	return c.readCookie(r, c.CSRFCookie)
}

func (c CookieAuthConfig) SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken, csrfToken string, now time.Time) {
	if !c.Enabled {
		return
	}

	http.SetCookie(w, c.newCookie(c.AccessCookie, accessToken, now.Add(c.AccessTTL), true))
	http.SetCookie(w, c.newCookie(c.RefreshCookie, refreshToken, now.Add(c.RefreshTTL), true))
	http.SetCookie(w, c.newCookie(c.CSRFCookie, csrfToken, now.Add(c.RefreshTTL), false))
}

func (c CookieAuthConfig) ClearAuthCookies(w http.ResponseWriter) {
	if !c.Enabled {
		return
	}

	http.SetCookie(w, c.expiredCookie(c.AccessCookie, true))
	http.SetCookie(w, c.expiredCookie(c.RefreshCookie, true))
	http.SetCookie(w, c.expiredCookie(c.CSRFCookie, false))
}

func (c CookieAuthConfig) newCookie(name, value string, expiresAt time.Time, httpOnly bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   c.Domain,
		Expires:  expiresAt.UTC(),
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: httpOnly,
		Secure:   c.Secure,
		SameSite: c.SameSite,
	}
}

func (c CookieAuthConfig) expiredCookie(name string, httpOnly bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   c.Domain,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: httpOnly,
		Secure:   c.Secure,
		SameSite: c.SameSite,
	}
}

func (c CookieAuthConfig) readCookie(r *http.Request, name string) string {
	if !c.Enabled || r == nil || strings.TrimSpace(name) == "" {
		return ""
	}
	cookie, err := r.Cookie(name)
	if err != nil || cookie == nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
