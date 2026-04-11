package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds the 3x-ui panel connection settings.
type Config struct {
	Host     string // Panel URL, e.g. "http://localhost:2053"
	BasePath string // Panel base path, e.g. "/" or "/custom/"
	Username string
	Password string
}

// Load reads configuration from environment variables and validates it.
func Load() (*Config, error) {
	cfg := &Config{
		Host:     os.Getenv("XUI_HOST"),
		BasePath: os.Getenv("XUI_BASE_PATH"),
		Username: os.Getenv("XUI_USERNAME"),
		Password: os.Getenv("XUI_PASSWORD"),
	}

	if cfg.BasePath == "" {
		cfg.BasePath = "/"
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	cfg.normalize()
	return cfg, nil
}

func (c *Config) validate() error {
	var errs []string

	if c.Host == "" {
		errs = append(errs, "XUI_HOST is required (e.g. http://localhost:2053)")
	}
	if c.Username == "" {
		errs = append(errs, "XUI_USERNAME is required")
	}
	if c.Password == "" {
		errs = append(errs, "XUI_PASSWORD is required")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (c *Config) normalize() {
	c.Host = strings.TrimRight(c.Host, "/")

	if !strings.HasPrefix(c.BasePath, "/") {
		c.BasePath = "/" + c.BasePath
	}
	if !strings.HasSuffix(c.BasePath, "/") {
		c.BasePath += "/"
	}
}

// BaseURL returns the full base URL for API requests.
func (c *Config) BaseURL() string {
	return c.Host + c.BasePath
}
