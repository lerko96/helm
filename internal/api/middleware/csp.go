package middleware

import (
	"net/http"
	"strings"
)

// CSP builds a Content-Security-Policy middleware whose `frame-src` directive
// is the operator-declared iframe allowlist plus `'self'`. Everything else is
// locked down to same-origin by default.
//
// Why middleware and not a static string: the allowlist comes from the loaded
// config, so the policy has to be built per-server-start (not per-request —
// the string is closed over once).
func CSP(iframeAllowedHosts []string) func(http.Handler) http.Handler {
	frameSrc := "'self'"
	if len(iframeAllowedHosts) > 0 {
		frameSrc = "'self' " + strings.Join(iframeAllowedHosts, " ")
	}

	policy := strings.Join([]string{
		"default-src 'self'",
		// Vite emits inline style tags for its CSS bundle; frontend uses inline
		// `style={...}` all over. Keeping 'unsafe-inline' here is a deliberate
		// call — revisit if we ever fully migrate to CSS modules.
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"script-src 'self'",
		"img-src 'self' data: https:",
		"connect-src 'self'",
		"frame-src " + frameSrc,
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}, "; ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", policy)
			next.ServeHTTP(w, r)
		})
	}
}
