package config

import (
	"fmt"
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
	Server  ServerConfig  `yaml:"server"`
	Auth    AuthConfig    `yaml:"auth"`
	Storage StorageConfig `yaml:"storage"`
	Pages   []Page        `yaml:"pages"`
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
				if err := validateWidget(w); err != nil {
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
func validateWidget(w Widget) error {
	switch w.Type {
	case "custom-api":
		_, err := ParseCustomAPI(w.Config)
		return err
	}
	return nil
}
