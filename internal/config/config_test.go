package config

import (
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

func TestWidgetID_Stable(t *testing.T) {
	// Must match the ID format emitted by configPagesHandler. If either
	// drifts, the proxy handler stops finding widgets.
	got := WidgetID("My Dashboard", 1, "custom-api", 2)
	want := "my-dashboard-col-1-custom-api-2"
	if got != want {
		t.Errorf("WidgetID = %q, want %q", got, want)
	}
}
