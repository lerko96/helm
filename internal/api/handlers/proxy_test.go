package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lerko/helm/internal/config"
	"github.com/lerko/helm/internal/httpclient"
)

// newTestState bypasses buildWidgetIndex (which would re-validate URLs via
// httpclient.Validate — rejecting the 127.0.0.1 test server). Tests inject
// CustomAPIConfig directly and use an AllowPrivateIPs client.
func newTestState(widgets map[string]config.CustomAPIConfig) *proxyState {
	out := &proxyState{
		widgets: make(map[string]proxyEntry, len(widgets)),
		client:  httpclient.New(httpclient.Options{AllowPrivateIPs: true, Timeout: 5 * time.Second}),
		cache:   make(map[string]proxyCacheEntry),
	}
	for id, cfg := range widgets {
		out.widgets[id] = proxyEntry{cfg: cfg}
	}
	return out
}

func TestProxy_MissingWidgetID(t *testing.T) {
	state := newTestState(nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proxy", nil)
	state.handler()(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestProxy_UnknownWidget(t *testing.T) {
	state := newTestState(nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=nope", nil)
	state.handler()(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestProxy_Passthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	state := newTestState(map[string]config.CustomAPIConfig{
		"w1": {URL: upstream.URL, Refresh: time.Minute},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=w1", nil)
	state.handler()(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	body := w.Body.String()
	if body != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}
}

func TestProxy_Cache(t *testing.T) {
	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("payload"))
	}))
	defer upstream.Close()

	state := newTestState(map[string]config.CustomAPIConfig{
		"w1": {URL: upstream.URL, Refresh: time.Minute},
	})
	handler := state.handler()

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=w1", nil)
		handler(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("iter %d: status = %d", i, w.Code)
		}
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("upstream hit %d times, want 1 (cache miss on first, hit on rest)", got)
	}
}

func TestProxy_CacheExpires(t *testing.T) {
	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("payload"))
	}))
	defer upstream.Close()

	// Refresh shorter than test sleep to force re-fetch.
	state := newTestState(map[string]config.CustomAPIConfig{
		"w1": {URL: upstream.URL, Refresh: 50 * time.Millisecond},
	})
	handler := state.handler()

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=w1", nil)
		handler(w, req)
		if i == 0 {
			time.Sleep(80 * time.Millisecond)
		}
	}
	if got := hits.Load(); got != 2 {
		t.Errorf("upstream hit %d times, want 2 after cache expiry", got)
	}
}

func TestProxy_HeadersForwarded(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	state := newTestState(map[string]config.CustomAPIConfig{
		"w1": {
			URL:     upstream.URL,
			Refresh: time.Minute,
			Headers: map[string]string{"Authorization": "Bearer secret-token"},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=w1", nil)
	state.handler()(w, req)

	if gotAuth != "Bearer secret-token" {
		t.Errorf("upstream Authorization = %q, want secret-token forwarded", gotAuth)
	}
}

func TestProxy_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	state := newTestState(map[string]config.CustomAPIConfig{
		"w1": {URL: upstream.URL, Refresh: time.Minute},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=w1", nil)
	state.handler()(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", w.Code)
	}
}

func TestProxy_BodySizeCap(t *testing.T) {
	// Upstream returns 2 MiB; cap is 1 MiB.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.Copy(w, strings.NewReader(strings.Repeat("x", 2<<20)))
	}))
	defer upstream.Close()

	state := newTestState(map[string]config.CustomAPIConfig{
		"w1": {URL: upstream.URL, Refresh: time.Minute},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proxy?widget_id=w1", nil)
	state.handler()(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Body.Len(); got != maxProxyBodyBytes {
		t.Errorf("body = %d bytes, want cap %d", got, maxProxyBodyBytes)
	}
}
