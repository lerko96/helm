package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lerko/helm/internal/httpclient"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// SlugifyPageName returns a URL/ID-safe form of a page name. Exposed so the
// API layer and the proxy handler agree on widget IDs without either owning
// the algorithm.
func SlugifyPageName(name string) string {
	return strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// WidgetID is the stable identifier emitted in /api/config/pages and
// referenced by the proxy endpoint. Keeping it here (instead of in the API
// layer) lets non-API consumers build the same ID without reaching through
// an HTTP round-trip.
func WidgetID(pageName string, colIdx int, widgetType string, widgetIdx int) string {
	return SlugifyPageName(pageName) + "-col-" + strconv.Itoa(colIdx) + "-" + widgetType + "-" + strconv.Itoa(widgetIdx)
}

type Config struct {
	Server             ServerConfig  `yaml:"server"`
	Auth               AuthConfig    `yaml:"auth"`
	Storage            StorageConfig `yaml:"storage"`
	Docker             DockerConfig  `yaml:"docker"`
	IframeAllowedHosts []string      `yaml:"iframe_allowed_hosts"`
	Pages              []Page        `yaml:"pages"`
}

// DockerConfig gates the docker-status widget. Disabled by default — mounting
// /var/run/docker.sock is a real security surface (container on that socket
// can escape), so opt-in.
type DockerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Socket  string `yaml:"socket"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type AuthConfig struct {
	Password string `yaml:"password"`
	Secret   string `yaml:"secret"`
}

type StorageConfig struct {
	DBPath          string `yaml:"db_path"`
	AttachmentsPath string `yaml:"attachments_path"`
}

type Page struct {
	Name    string   `yaml:"name"`
	Columns []Column `yaml:"columns"`
}

type Column struct {
	Size    string   `yaml:"size"` // small, medium, large
	Widgets []Widget `yaml:"widgets"`
}

type Widget struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config,omitempty"`
}

// CustomAPIConfig describes the per-widget config accepted by the `custom-api`
// widget. Parsed from Widget.Config at Load() time so malformed configs fail
// fast rather than at first request.
type CustomAPIConfig struct {
	URL     string
	Refresh time.Duration
	Headers map[string]string
}

// minCustomAPIRefresh caps how often a custom-api widget may hit upstream.
// Prevents accidental config typos from turning a refresh into a DoS.
const minCustomAPIRefresh = 10 * time.Second

// DefaultCustomAPIRefresh applies when a widget omits `refresh`.
const DefaultCustomAPIRefresh = 5 * time.Minute

// ParseCustomAPI extracts + validates the typed config for a custom-api widget.
func ParseCustomAPI(raw map[string]any) (CustomAPIConfig, error) {
	var out CustomAPIConfig

	urlVal, _ := raw["url"].(string)
	if urlVal == "" {
		return out, fmt.Errorf("custom-api: url is required")
	}
	if err := httpclient.Validate(urlVal, httpclient.Options{}); err != nil {
		return out, fmt.Errorf("custom-api url %q: %w", urlVal, err)
	}
	out.URL = urlVal

	out.Refresh = DefaultCustomAPIRefresh
	if refRaw, ok := raw["refresh"]; ok {
		refStr, ok := refRaw.(string)
		if !ok {
			return out, fmt.Errorf("custom-api: refresh must be a duration string")
		}
		d, err := time.ParseDuration(refStr)
		if err != nil {
			return out, fmt.Errorf("custom-api refresh %q: %w", refStr, err)
		}
		if d < minCustomAPIRefresh {
			return out, fmt.Errorf("custom-api refresh %s below minimum %s", d, minCustomAPIRefresh)
		}
		out.Refresh = d
	}

	if hdrRaw, ok := raw["headers"]; ok {
		hdrMap, ok := hdrRaw.(map[string]any)
		if !ok {
			return out, fmt.Errorf("custom-api: headers must be a string map")
		}
		out.Headers = make(map[string]string, len(hdrMap))
		for k, v := range hdrMap {
			s, ok := v.(string)
			if !ok {
				return out, fmt.Errorf("custom-api: header %q must be a string", k)
			}
			out.Headers[k] = s
		}
	}

	return out, nil
}

// IframeConfig describes the per-widget config accepted by the `iframe` widget.
// URL is validated at config load against the operator-declared allowlist
// (Config.IframeAllowedHosts) so a typo or drift fails fast, not in the
// browser where the CSP would silently block the frame.
type IframeConfig struct {
	URL     string
	Height  string
	Sandbox string
}

// DefaultIframeSandbox is what the frontend sets on the iframe when the
// widget config omits `sandbox`. Narrow enough to keep cross-origin frames
// from tampering with Helm's DOM, wide enough for most self-hosted tools.
const DefaultIframeSandbox = "allow-same-origin allow-scripts"

// ParseIframe extracts + validates the typed config for an iframe widget.
// Requires the URL's host be on the operator's iframe_allowed_hosts list —
// the CSP frame-src directive is built from the same list, so a widget
// pointing at an un-allowlisted host would render as a blank frame.
func ParseIframe(raw map[string]any, allowedHosts []string) (IframeConfig, error) {
	var out IframeConfig

	urlVal, _ := raw["url"].(string)
	if urlVal == "" {
		return out, fmt.Errorf("iframe: url is required")
	}

	host, err := extractHost(urlVal)
	if err != nil {
		return out, fmt.Errorf("iframe url %q: %w", urlVal, err)
	}
	if !hostAllowed(host, allowedHosts) {
		return out, fmt.Errorf(
			"iframe url %q: host %q not in iframe_allowed_hosts (add to config.yml to embed)",
			urlVal, host,
		)
	}
	out.URL = urlVal

	if h, ok := raw["height"].(string); ok {
		out.Height = h
	}
	if sb, ok := raw["sandbox"].(string); ok {
		out.Sandbox = sb
	} else {
		out.Sandbox = DefaultIframeSandbox
	}

	return out, nil
}

func extractHost(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("scheme %q not allowed (use https or http)", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("url has no host")
	}
	return u.Hostname(), nil
}

func hostAllowed(host string, allowed []string) bool {
	for _, h := range allowed {
		if strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Storage: StorageConfig{
			DBPath:          "./data/helm.db",
			AttachmentsPath: "./data/attachments",
		},
		Docker: DockerConfig{
			Enabled: false,
			Socket:  "/var/run/docker.sock",
		},
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if cfg.Auth.Password == "" {
		return nil, fmt.Errorf("auth.password must be set in config")
	}
	if len(cfg.Auth.Secret) < 32 {
		return nil, fmt.Errorf("auth.secret must be at least 32 characters")
	}

	// Validate per-widget configs. Fail fast — a typo in a URL should stop
	// startup, not surface on first request.
	for pi, page := range cfg.Pages {
		for ci, col := range page.Columns {
			for wi, w := range col.Widgets {
				if err := validateWidget(w, cfg); err != nil {
					return nil, fmt.Errorf(
						"page[%d] %q column[%d] widget[%d] (%s): %w",
						pi, page.Name, ci, wi, w.Type, err,
					)
				}
			}
		}
	}

	return cfg, nil
}

// validateWidget delegates to per-type validators. Unknown types are allowed
// through — the frontend treats them as empty widgets, matching existing
// behavior where `type` is a loose contract.
func validateWidget(w Widget, cfg *Config) error {
	switch w.Type {
	case "custom-api":
		_, err := ParseCustomAPI(w.Config)
		return err
	case "iframe":
		_, err := ParseIframe(w.Config, cfg.IframeAllowedHosts)
		return err
	}
	return nil
}
