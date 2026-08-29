package config

import (
	"testing"
)

func TestLoad_MissingHost(t *testing.T) {
	t.Setenv("XUI_HOST", "")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "admin")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when XUI_HOST is missing, got nil")
	}
	if got := err.Error(); !contains(got, "XUI_HOST") {
		t.Errorf("error should mention XUI_HOST, got: %s", got)
	}
}

func TestLoad_MissingUsername(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "")
	t.Setenv("XUI_PASSWORD", "admin")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when XUI_USERNAME is missing, got nil")
	}
	if got := err.Error(); !contains(got, "XUI_USERNAME") {
		t.Errorf("error should mention XUI_USERNAME, got: %s", got)
	}
}

func TestLoad_MissingPassword(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when XUI_PASSWORD is missing, got nil")
	}
	if got := err.Error(); !contains(got, "XUI_PASSWORD") {
		t.Errorf("error should mention XUI_PASSWORD, got: %s", got)
	}
}

func TestLoad_AllRequiredVarsPresent(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "secret")
	t.Setenv("XUI_BASE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "http://localhost:2053" {
		t.Errorf("Host = %q, want %q", cfg.Host, "http://localhost:2053")
	}
	if cfg.Username != "admin" {
		t.Errorf("Username = %q, want %q", cfg.Username, "admin")
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret")
	}
}

func TestLoad_DefaultBasePath(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "admin")
	t.Setenv("XUI_BASE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BasePath != "/" {
		t.Errorf("BasePath = %q, want %q", cfg.BasePath, "/")
	}
}

func TestLoad_BasePathNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no slashes", "custom", "/custom/"},
		{"leading slash only", "/custom", "/custom/"},
		{"trailing slash only", "custom/", "/custom/"},
		{"both slashes", "/custom/", "/custom/"},
		{"nested path", "panel/sub", "/panel/sub/"},
		{"already normalized", "/panel/sub/", "/panel/sub/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XUI_HOST", "http://localhost:2053")
			t.Setenv("XUI_USERNAME", "admin")
			t.Setenv("XUI_PASSWORD", "admin")
			t.Setenv("XUI_BASE_PATH", tt.input)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.BasePath != tt.expected {
				t.Errorf("BasePath = %q, want %q", cfg.BasePath, tt.expected)
			}
		})
	}
}

func TestLoad_HostTrailingSlashTrimmed(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053/")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "admin")
	t.Setenv("XUI_BASE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "http://localhost:2053" {
		t.Errorf("Host = %q, want trailing slash trimmed to %q", cfg.Host, "http://localhost:2053")
	}
}

func TestConfig_BaseURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		basePath string
		expected string
	}{
		{"default base path", "http://localhost:2053", "/", "http://localhost:2053/"},
		{"custom base path", "http://localhost:2053", "/panel/", "http://localhost:2053/panel/"},
		{"nested base path", "http://example.com", "/a/b/", "http://example.com/a/b/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Host:     tt.host,
				BasePath: tt.basePath,
			}
			if got := cfg.BaseURL(); got != tt.expected {
				t.Errorf("BaseURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoad_MultipleFieldsMissing(t *testing.T) {
	t.Setenv("XUI_HOST", "")
	t.Setenv("XUI_USERNAME", "")
	t.Setenv("XUI_PASSWORD", "")
	t.Setenv("XUI_BASE_PATH", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when multiple fields are missing, got nil")
	}
	got := err.Error()
	if !contains(got, "XUI_HOST") {
		t.Errorf("error should mention XUI_HOST, got: %s", got)
	}
	if !contains(got, "XUI_USERNAME") {
		t.Errorf("error should mention XUI_USERNAME, got: %s", got)
	}
	if !contains(got, "XUI_PASSWORD") {
		t.Errorf("error should mention XUI_PASSWORD, got: %s", got)
	}
}

func TestLoad_TokenOnly_NoCredentials(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "")
	t.Setenv("XUI_PASSWORD", "")
	t.Setenv("XUI_API_TOKEN", "tok-123")
	t.Setenv("XUI_BASE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("token-only config should be valid, got error: %v", err)
	}
	if cfg.APIToken != "tok-123" {
		t.Errorf("APIToken = %q, want %q", cfg.APIToken, "tok-123")
	}
}

func TestLoad_TokenWithCredentials(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "secret")
	t.Setenv("XUI_API_TOKEN", "tok-123")
	t.Setenv("XUI_BASE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIToken != "tok-123" || cfg.Username != "admin" || cfg.Password != "secret" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoad_PartialCredentialsWithToken(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "")
	t.Setenv("XUI_API_TOKEN", "tok-123")
	t.Setenv("XUI_BASE_PATH", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for partial credentials (username without password), got nil")
	}
	if got := err.Error(); !contains(got, "set together") {
		t.Errorf("error should explain credentials must be set together, got: %s", got)
	}
}

// contains is a small helper to avoid importing strings in tests.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLoad_ParsesToolsets(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "admin")
	t.Setenv("XUI_TOOLSETS", " Clients ,, inbounds,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := []string{"clients", "inbounds"}
	if len(cfg.Toolsets) != len(want) {
		t.Fatalf("toolsets = %v, want %v", cfg.Toolsets, want)
	}
	for i, name := range want {
		if cfg.Toolsets[i] != name {
			t.Errorf("toolsets[%d] = %q, want %q", i, cfg.Toolsets[i], name)
		}
	}
}

func TestLoad_NoToolsetsMeansEmpty(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "admin")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Toolsets) != 0 {
		t.Errorf("toolsets = %v, want empty so every group loads", cfg.Toolsets)
	}
}

func TestLoad_ReturnsConfigOnValidationFailure(t *testing.T) {
	t.Setenv("XUI_HOST", "")
	t.Setenv("XUI_USERNAME", "")
	t.Setenv("XUI_PASSWORD", "")
	t.Setenv("XUI_API_TOKEN", "")

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	// The server starts anyway so tools stay listable; Load must hand back a
	// usable value rather than nil, carrying the failure on the Config.
	if cfg == nil {
		t.Fatal("Load returned nil config; an incomplete config must still be usable")
	}
	if cfg.Err() == nil {
		t.Error("cfg.Err() = nil, want the validation failure recorded on the config")
	}
	if cfg.BasePath != "/" {
		t.Errorf("BasePath = %q, want the default %q even on a failed config", cfg.BasePath, "/")
	}
}

func TestLoad_ErrIsNilWhenValid(t *testing.T) {
	t.Setenv("XUI_HOST", "http://localhost:2053")
	t.Setenv("XUI_USERNAME", "admin")
	t.Setenv("XUI_PASSWORD", "admin")
	t.Setenv("XUI_API_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Err() != nil {
		t.Errorf("cfg.Err() = %v, want nil for a complete config", cfg.Err())
	}
}
