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
	APIToken string   // Optional Bearer API token (3x-ui v3.2.8+). When set, /panel/api/* calls bypass CSRF.
	Toolsets []string // Tool groups to expose; empty means every group.
}

// Load reads configuration from environment variables and validates it.
func Load() (*Config, error) {
	cfg := &Config{
		Host:     os.Getenv("XUI_HOST"),
		BasePath: os.Getenv("XUI_BASE_PATH"),
		Username: os.Getenv("XUI_USERNAME"),
		Password: os.Getenv("XUI_PASSWORD"),
		APIToken: os.Getenv("XUI_API_TOKEN"),
		Toolsets: splitList(os.Getenv("XUI_TOOLSETS")),
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

	hasCreds := c.Username != "" && c.Password != ""

	// Either a full credential pair or an API token must be present.
	if !hasCreds && c.APIToken == "" {
		errs = append(errs, "authentication required: set XUI_API_TOKEN, or both XUI_USERNAME and XUI_PASSWORD")
	}
	// Partial credentials are almost certainly a misconfiguration.
	if !hasCreds && (c.Username != "" || c.Password != "") {
		errs = append(errs, "XUI_USERNAME and XUI_PASSWORD must be set together")
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

// splitList parses a comma-separated environment value into trimmed, lowercased
// entries, dropping empties so "clients, ,inbounds," behaves as expected.
func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.ToLower(strings.TrimSpace(part)); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// BaseURL returns the full base URL for API requests.
func (c *Config) BaseURL() string {
	return c.Host + c.BasePath
}
