// Package httpclient provides an HTTP client hardened against SSRF for
// outbound requests to operator-declared URLs (CalDAV sources, custom-api
// widgets, etc.). It enforces:
//
//   - scheme allowlist (https-only by default)
//   - DNS resolution-time IP filter (blocks loopback, RFC1918, link-local, ULA)
//   - dial-time IP filter (same check at connect time — defense against DNS
//     rebinding, where a host resolves to a public IP at validation but a
//     private IP at dial)
//   - request-wide timeout
//
// Callers should Validate the URL at config load (fail fast on malformed or
// obviously disallowed URLs) and use the *http.Client returned by New for the
// actual request.
package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"time"
)

// Options configures a SafeClient. Zero value applies secure defaults.
type Options struct {
	// Timeout applied as http.Client.Timeout. Defaults to 30s.
	Timeout time.Duration

	// AllowedSchemes is the set of URL schemes accepted by Validate and the
	// dialer. Defaults to {"https"}.
	AllowedSchemes []string

	// AllowPrivateIPs disables the RFC1918/loopback/link-local/ULA filter.
	// Only set for tests or operator-opt-in to reach internal hosts.
	AllowPrivateIPs bool
}

func (o Options) withDefaults() Options {
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if len(o.AllowedSchemes) == 0 {
		o.AllowedSchemes = []string{"https"}
	}
	return o
}

// privateRanges enumerates CIDR blocks treated as SSRF-sensitive.
var privateRanges = func() []net.IPNet {
	cidrs := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // link-local
		"0.0.0.0/8",      // "this network"
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 ULA
		"fe80::/10",      // IPv6 link-local
	}
	out := make([]net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(fmt.Sprintf("httpclient: invalid CIDR %q: %v", c, err))
		}
		out = append(out, *n)
	}
	return out
}()

// IsPrivateIP reports whether ip sits in an SSRF-sensitive range.
func IsPrivateIP(ip net.IP) bool {
	for _, block := range privateRanges {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// Validate checks rawURL against the scheme allowlist and resolves its host,
// rejecting if any address is private (unless AllowPrivateIPs). Call at
// config-load time for fail-fast behavior.
func Validate(rawURL string, opts Options) error {
	opts = opts.withDefaults()

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("URL missing scheme: %q", rawURL)
	}
	if !slices.Contains(opts.AllowedSchemes, u.Scheme) {
		return fmt.Errorf("scheme %q not in allowlist %v", u.Scheme, opts.AllowedSchemes)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL missing host: %q", rawURL)
	}
	if opts.AllowPrivateIPs {
		return nil
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", host, err)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if IsPrivateIP(ip) {
			return fmt.Errorf("host %q resolves to private address %s", host, addr)
		}
	}
	return nil
}

// New returns an *http.Client whose dialer rejects private-IP destinations
// (unless AllowPrivateIPs). Use Validate in addition for fail-fast config
// validation.
func New(opts Options) *http.Client {
	opts = opts.withDefaults()

	baseDialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if !opts.AllowPrivateIPs {
				for _, ip := range ips {
					if IsPrivateIP(ip.IP) {
						return nil, fmt.Errorf("httpclient: refusing to dial private address %s", ip.IP)
					}
				}
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("httpclient: no addresses for %s", host)
			}
			// Prefer the first resolved address. (Go's default dialer does
			// happy-eyeballs across all returned addrs; we trade that for
			// explicit IP control. All addrs already passed the filter.)
			return baseDialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}

	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}
}
