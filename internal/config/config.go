package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

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

	return cfg, nil
}
