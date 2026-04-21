package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseCustomAPI_Happy(t *testing.T) {
	got, err := ParseCustomAPI(map[string]any{
		"url":     "https://example.com/api",
		"refresh": "30s",
		"headers": map[string]any{
			"X-API-Key": "abc",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.URL != "https://example.com/api" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Refresh != 30*time.Second {
		t.Errorf("Refresh = %s", got.Refresh)
	}
	if got.Headers["X-API-Key"] != "abc" {
		t.Errorf("Headers = %v", got.Headers)
	}
}

func TestParseCustomAPI_DefaultRefresh(t *testing.T) {
	got, err := ParseCustomAPI(map[string]any{"url": "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Refresh != DefaultCustomAPIRefresh {
		t.Errorf("Refresh = %s, want default %s", got.Refresh, DefaultCustomAPIRefresh)
	}
}

func TestParseCustomAPI_RejectsBelowMin(t *testing.T) {
	_, err := ParseCustomAPI(map[string]any{
		"url":     "https://example.com",
		"refresh": "1s",
	})
	if err == nil || !strings.Contains(err.Error(), "minimum") {
		t.Errorf("expected minimum-refresh rejection, got: %v", err)
	}
}

func TestParseCustomAPI_RejectsPrivateIP(t *testing.T) {
	_, err := ParseCustomAPI(map[string]any{"url": "https://127.0.0.1/api"})
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Errorf("expected private-IP rejection, got: %v", err)
	}
}

func TestParseCustomAPI_RejectsHTTP(t *testing.T) {
	_, err := ParseCustomAPI(map[string]any{"url": "http://example.com/api"})
	if err == nil {
		t.Error("expected http:// rejection under https-only default")
	}
}

func TestParseCustomAPI_RequiresURL(t *testing.T) {
	_, err := ParseCustomAPI(map[string]any{})
	if err == nil {
		t.Error("expected url-required error")
	}
}

func TestParseIframe_Happy(t *testing.T) {
	got, err := ParseIframe(
		map[string]any{"url": "https://grafana.example.com/dashboard"},
		[]string{"grafana.example.com"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.URL != "https://grafana.example.com/dashboard" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Sandbox != DefaultIframeSandbox {
		t.Errorf("Sandbox default = %q, want %q", got.Sandbox, DefaultIframeSandbox)
	}
}

func TestParseIframe_CustomSandbox(t *testing.T) {
	got, err := ParseIframe(
		map[string]any{
			"url":     "https://ha.example.com",
			"sandbox": "allow-same-origin allow-scripts allow-forms",
			"height":  "720px",
		},
		[]string{"ha.example.com"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Sandbox != "allow-same-origin allow-scripts allow-forms" {
		t.Errorf("Sandbox = %q", got.Sandbox)
	}
	if got.Height != "720px" {
		t.Errorf("Height = %q", got.Height)
	}
}

func TestParseIframe_RejectsNonAllowlistedHost(t *testing.T) {
	_, err := ParseIframe(
		map[string]any{"url": "https://evil.example.com"},
		[]string{"grafana.example.com"},
	)
	if err == nil || !strings.Contains(err.Error(), "iframe_allowed_hosts") {
		t.Errorf("expected allowlist rejection, got: %v", err)
	}
}

func TestParseIframe_RejectsEmptyAllowlist(t *testing.T) {
	_, err := ParseIframe(
		map[string]any{"url": "https://example.com"},
		nil,
	)
	if err == nil {
		t.Error("expected rejection with empty allowlist")
	}
}

func TestParseIframe_RequiresURL(t *testing.T) {
	_, err := ParseIframe(map[string]any{}, []string{"example.com"})
	if err == nil {
		t.Error("expected url-required error")
	}
}

func TestParseIframe_RejectsBadScheme(t *testing.T) {
	_, err := ParseIframe(
		map[string]any{"url": "javascript:alert(1)"},
		[]string{"alert"},
	)
	if err == nil {
		t.Error("expected non-http(s) scheme rejection")
	}
}

func TestParseIframe_HostCompareIsCaseInsensitive(t *testing.T) {
	_, err := ParseIframe(
		map[string]any{"url": "https://Grafana.Example.COM/x"},
		[]string{"grafana.example.com"},
	)
	if err != nil {
		t.Errorf("expected case-insensitive host match, got: %v", err)
	}
}

func TestWidgetID_Stable(t *testing.T) {
	// Must match the ID format emitted by configPagesHandler. If either
	// drifts, the proxy handler stops finding widgets.
	got := WidgetID("My Dashboard", 1, "custom-api", 2)
	want := "my-dashboard-col-1-custom-api-2"
	if got != want {
		t.Errorf("WidgetID = %q, want %q", got, want)
	}
}

// TestConfigExamples_AllLoad keeps config-examples/ honest — every example
// must pass the same Load() validation as a real config. Catches drift when
// we tighten validation or rename a field.
func TestConfigExamples_AllLoad(t *testing.T) {
	dir := filepath.Join("..", "..", "config-examples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read config-examples dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no examples found in config-examples/")
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			if _, err := Load(path); err != nil {
				t.Errorf("example %s failed to load: %v", path, err)
			}
		})
	}
}
