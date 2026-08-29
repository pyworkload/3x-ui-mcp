package xui

import (
	"context"
	"fmt"
)

// --- API token API methods (panel v3.7.0+) ---
//
// Tokens are scoped (admin, monitor, node-sync) and may expire. The plaintext
// exists only in the create response — the panel stores a SHA-256 hash — and
// delete/setEnabled fail closed unless the caller states the scope it expects,
// so a token cannot be disabled by guessing its id alone.

const apiTokensBase = "panel/api/setting/apiTokens"

// ListAPITokens returns token metadata. The token values are never returned.
func (c *Client) ListAPITokens(ctx context.Context) (*Response, error) {
	return c.Get(ctx, apiTokensBase)
}

// CreateAPIToken mints a token and returns its plaintext exactly once.
// expiresAt is Unix milliseconds, or 0 for no expiry.
func (c *Client) CreateAPIToken(ctx context.Context, name, scope string, expiresAt int64) (*Response, error) {
	return c.PostJSON(ctx, apiTokensBase+"/create", map[string]any{
		"name":      name,
		"scope":     scope,
		"expiresAt": expiresAt,
	})
}

// DeleteAPIToken permanently removes a token; anything using it stops
// authenticating at once. expectedScope must match the stored scope.
func (c *Client) DeleteAPIToken(ctx context.Context, id int, expectedScope string) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf("%s/delete/%d", apiTokensBase, id), map[string]any{
		"expectedScope": expectedScope,
	})
}

// SetAPITokenEnabled toggles a token without deleting it.
func (c *Client) SetAPITokenEnabled(ctx context.Context, id int, expectedScope string, enabled bool) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf("%s/setEnabled/%d", apiTokensBase, id), map[string]any{
		"enabled":       enabled,
		"expectedScope": expectedScope,
	})
}
