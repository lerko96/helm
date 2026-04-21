package httpclient

import (
	"net"
	"strings"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"10.255.255.254", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true}, // AWS metadata
		{"0.0.0.0", true},
		{"::1", true},
		{"fc00::1", true},
		{"fe80::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"172.32.0.1", false}, // just outside 172.16/12
		{"2606:4700:4700::1111", false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("parse %q", c.ip)
		}
		if got := IsPrivateIP(ip); got != c.private {
			t.Errorf("IsPrivateIP(%s) = %v, want %v", c.ip, got, c.private)
		}
	}
}

func TestValidate_Schemes(t *testing.T) {
	// Scheme check happens before DNS — safe to assert against nonexistent hosts.
	if err := Validate("ftp://example.invalid/x", Options{}); err == nil {
		t.Error("expected scheme rejection for ftp://")
	}
	if err := Validate("http://example.invalid/x", Options{}); err == nil {
		t.Error("expected scheme rejection for http:// with default allowlist")
	}
	if err := Validate("http://example.invalid/x", Options{
		AllowedSchemes: []string{"http", "https"},
	}); err == nil || !strings.Contains(err.Error(), "resolve") {
		// resolution may fail, but scheme check must pass
		if err != nil && strings.Contains(err.Error(), "not in allowlist") {
			t.Errorf("scheme rejected when in allowlist: %v", err)
		}
	}
}

func TestValidate_Malformed(t *testing.T) {
	cases := []string{
		"",
		"://nohost",
		"https://",
	}
	for _, raw := range cases {
		if err := Validate(raw, Options{}); err == nil {
			t.Errorf("Validate(%q) expected error, got nil", raw)
		}
	}
}

func TestValidate_PrivateHost(t *testing.T) {
	// Literal loopback — no DNS needed.
	cases := []string{
		"https://127.0.0.1/x",
		"https://10.0.0.1/x",
		"https://192.168.1.1/x",
		"https://[::1]/x",
		"https://169.254.169.254/latest/meta-data/", // AWS metadata
	}
	for _, raw := range cases {
		err := Validate(raw, Options{})
		if err == nil {
			t.Errorf("Validate(%q) expected private-IP rejection, got nil", raw)
			continue
		}
		if !strings.Contains(err.Error(), "private") {
			t.Errorf("Validate(%q) expected private-address error, got: %v", raw, err)
		}
	}
}

func TestValidate_PrivateHost_Allowed(t *testing.T) {
	// With AllowPrivateIPs, literal loopback should pass validation.
	if err := Validate("https://127.0.0.1/x", Options{AllowPrivateIPs: true}); err != nil {
		t.Errorf("AllowPrivateIPs should permit loopback: %v", err)
	}
}

func TestNew_Defaults(t *testing.T) {
	c := New(Options{})
	if c.Timeout == 0 {
		t.Error("default timeout not applied")
	}
	if c.Transport == nil {
		t.Error("transport nil")
	}
}
