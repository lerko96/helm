package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lerko/helm/internal/broker"
	"github.com/lerko/helm/internal/config"
)

// Guards against a regression where the Phase-C proxy handler existed but
// was never mounted on the router — requests fell through to the SPA
// catch-all and returned index.html instead of 401.
func TestRouter_ProxyMountedUnderAuth(t *testing.T) {
	cfg := &config.Config{Auth: config.AuthConfig{Secret: "test-secret-at-least-32-characters-long"}}
	r := NewRouter(cfg, nil, nil, broker.New())

	req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=anything", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/proxy without auth: got %d, want 401 (SPA fallback regression?)", rec.Code)
	}
}
