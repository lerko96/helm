package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// Secrets must never leak into /api/config/pages. Custom-api widgets declare
// `url` and `headers` (for Authorization etc.) which the browser has no need
// to see — the proxy endpoint looks them up server-side by widget_id.
func TestConfigPages_SanitizesCustomAPISecrets(t *testing.T) {
	cfg := &config.Config{
		Pages: []config.Page{
			{
				Name: "Home",
				Columns: []config.Column{
					{
						Size: "small",
						Widgets: []config.Widget{
							{
								Type: "custom-api",
								Config: map[string]any{
									"url":      "https://example.com/api",
									"refresh":  "30s",
									"template": "hello {{.name}}",
									"headers": map[string]any{
										"Authorization": "Bearer super-secret",
										"X-API-Key":     "abc123",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config/pages", nil)
	configPagesHandler(cfg)(w, req)

	body := w.Body.String()
	for _, leaked := range []string{"super-secret", "abc123", "Authorization", "X-API-Key", "example.com"} {
		if strings.Contains(body, leaked) {
			t.Errorf("response leaked %q in %s", leaked, body)
		}
	}

	var pages []apiPage
	if err := json.Unmarshal(w.Body.Bytes(), &pages); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	wc := pages[0].Columns[0].Widgets[0].Config
	if wc["template"] != "hello {{.name}}" {
		t.Errorf("template stripped or altered: %v", wc["template"])
	}
	if wc["refresh"] != "30s" {
		t.Errorf("refresh stripped or altered: %v", wc["refresh"])
	}
	if _, ok := wc["url"]; ok {
		t.Error("url should not reach the browser")
	}
	if _, ok := wc["headers"]; ok {
		t.Error("headers should not reach the browser")
	}
}

// Non-custom-api widgets pass their config through untouched — sanitization
// is keyed on widget type, not a global denylist.
func TestConfigPages_PassesThroughNonCustomAPI(t *testing.T) {
	cfg := &config.Config{
		Pages: []config.Page{
			{
				Name: "Home",
				Columns: []config.Column{
					{
						Size: "small",
						Widgets: []config.Widget{
							{Type: "memos", Config: map[string]any{"limit": 15}},
						},
					},
				},
			},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config/pages", nil)
	configPagesHandler(cfg)(w, req)

	var pages []apiPage
	if err := json.Unmarshal(w.Body.Bytes(), &pages); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	wc := pages[0].Columns[0].Widgets[0].Config
	if v, ok := wc["limit"]; !ok || int(v.(float64)) != 15 {
		t.Errorf("memos config altered: %v", wc)
	}
}
