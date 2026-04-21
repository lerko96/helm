package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lerko/helm/internal/config"
	"github.com/lerko/helm/internal/httpclient"
)

// maxProxyBodyBytes bounds each upstream response. Custom-api widgets are
// dashboards, not file transfers — a run-away upstream shouldn't exhaust
// memory. 1 MiB covers anything reasonable.
const maxProxyBodyBytes = 1 << 20

// proxyEntry is a prebuilt lookup: widget ID → validated config. The map is
// built once at handler construction; config reloads require a process
// restart (matches the rest of the server's config model).
type proxyEntry struct {
	cfg config.CustomAPIConfig
}

// proxyCacheEntry holds the last successful upstream response. `expiresAt`
// tracks when a re-fetch is required; before that we serve the cached bytes
// untouched so a burst of tab reloads doesn't hammer upstream.
type proxyCacheEntry struct {
	body        []byte
	contentType string
	expiresAt   time.Time
}

type proxyState struct {
	widgets map[string]proxyEntry
	client  *http.Client

	mu    sync.Mutex
	cache map[string]proxyCacheEntry
}

// ProxyCustomAPI returns a handler that resolves widget IDs from the loaded
// config and fetches their upstream URLs through the SSRF-hardened client.
// URLs are never taken from the request — the client only names *which*
// widget, the server decides what to fetch. Response body is a pass-through.
func ProxyCustomAPI(cfg *config.Config) http.HandlerFunc {
	return newProxyState(cfg, httpclient.Options{Timeout: 15 * time.Second}).handler()
}

// newProxyState is the seam used by tests to inject an httpclient with
// AllowPrivateIPs (httptest.Server binds to 127.0.0.1, which the production
// config rejects).
func newProxyState(cfg *config.Config, opts httpclient.Options) *proxyState {
	return &proxyState{
		widgets: buildWidgetIndex(cfg),
		client:  httpclient.New(opts),
		cache:   make(map[string]proxyCacheEntry),
	}
}

func (s *proxyState) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		widgetID := r.URL.Query().Get("widget_id")
		if widgetID == "" {
			respondError(w, http.StatusBadRequest, "widget_id required")
			return
		}

		entry, ok := s.widgets[widgetID]
		if !ok {
			respondError(w, http.StatusNotFound, "widget not found or not custom-api")
			return
		}

		body, contentType, err := s.fetch(r.Context(), widgetID, entry)
		if err != nil {
			respondError(w, http.StatusBadGateway, "upstream fetch failed")
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func (s *proxyState) fetch(ctx context.Context, id string, entry proxyEntry) ([]byte, string, error) {
	// Cache hit — serve untouched.
	s.mu.Lock()
	if cached, ok := s.cache[id]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.Unlock()
		return cached.body, cached.contentType, nil
	}
	s.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.cfg.URL, nil)
	if err != nil {
		return nil, "", err
	}
	for k, v := range entry.cfg.Headers {
		req.Header.Set(k, v)
	}
	// Upstreams that sniff Accept often pick JSON when we ask; custom-api
	// widgets are overwhelmingly JSON-templated.
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json, */*;q=0.5")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyBodyBytes))
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	s.mu.Lock()
	s.cache[id] = proxyCacheEntry{
		body:        body,
		contentType: contentType,
		expiresAt:   time.Now().Add(entry.cfg.Refresh),
	}
	s.mu.Unlock()

	return body, contentType, nil
}

// buildWidgetIndex walks the config and resolves each custom-api widget to
// its validated config. IDs come from config.WidgetID so they stay in
// lockstep with /api/config/pages.
func buildWidgetIndex(cfg *config.Config) map[string]proxyEntry {
	out := make(map[string]proxyEntry)
	for _, page := range cfg.Pages {
		for ci, col := range page.Columns {
			for wi, w := range col.Widgets {
				if w.Type != "custom-api" {
					continue
				}
				parsed, err := config.ParseCustomAPI(w.Config)
				if err != nil {
					// Config already passed Load() validation; reaching here means
					// the validator and the parser disagree — a programmer error.
					continue
				}
				out[config.WidgetID(page.Name, ci, w.Type, wi)] = proxyEntry{cfg: parsed}
			}
		}
	}
	return out
}
