package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSP_DefaultPolicy(t *testing.T) {
	h := CSP(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	policy := w.Header().Get("Content-Security-Policy")
	if policy == "" {
		t.Fatal("Content-Security-Policy header missing")
	}
	if !strings.Contains(policy, "frame-src 'self'") {
		t.Errorf("expected frame-src 'self' with empty allowlist, got: %s", policy)
	}
	if strings.Contains(policy, "frame-src 'self' ") {
		// Trailing space after 'self' means an allowlisted host was appended;
		// with nil allowlist there shouldn't be any.
		t.Errorf("unexpected allowlist tokens with nil config: %s", policy)
	}
}

func TestCSP_AppendsAllowedHosts(t *testing.T) {
	h := CSP([]string{"grafana.example.com", "ha.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	policy := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "frame-src 'self' grafana.example.com ha.example.com") {
		t.Errorf("frame-src missing declared hosts: %s", policy)
	}
	if !strings.Contains(policy, "frame-ancestors 'none'") {
		t.Errorf("frame-ancestors should pin to 'none': %s", policy)
	}
}

func TestCSP_HeaderPresentOnAllResponses(t *testing.T) {
	// Even 4xx/5xx responses carry the CSP header — the middleware writes it
	// before the handler runs, so errors can't accidentally disable the policy.
	h := CSP([]string{"x.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Header().Get("Content-Security-Policy") == "" {
		t.Error("CSP header dropped on error response")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
