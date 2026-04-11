package xui

import (
	"context"
	"fmt"
	"net/url"
)

// --- Server management API methods ---

// ServerStatus returns current server resource usage (CPU, RAM, disk, etc).
func (c *Client) ServerStatus(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/status")
}

// RestartXray restarts the Xray proxy service.
func (c *Client) RestartXray(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/server/restartXrayService")
}

// StopXray stops the Xray proxy service.
func (c *Client) StopXray(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/server/stopXrayService")
}

// GetXrayConfig returns the current Xray JSON configuration.
func (c *Client) GetXrayConfig(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getConfigJson")
}

// GetXrayVersions returns available Xray versions.
func (c *Client) GetXrayVersions(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getXrayVersion")
}

// InstallXray installs a specific Xray version.
func (c *Client) InstallXray(ctx context.Context, version string) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/server/installXray/%s", version))
}

// GetLogs returns application logs.
func (c *Client) GetLogs(ctx context.Context, count int, level, syslog string) (*Response, error) {
	data := url.Values{}
	if level != "" {
		data.Set("level", level)
	}
	if syslog != "" {
		data.Set("syslog", syslog)
	}
	return c.PostForm(ctx, fmt.Sprintf("panel/api/server/logs/%d", count), data)
}

// GetXrayLogs returns Xray proxy logs with optional filtering.
func (c *Client) GetXrayLogs(ctx context.Context, count int, filter string) (*Response, error) {
	data := url.Values{}
	if filter != "" {
		data.Set("filter", filter)
	}
	return c.PostForm(ctx, fmt.Sprintf("panel/api/server/xraylogs/%d", count), data)
}

// --- Settings API methods ---

// GetSettings returns all panel settings.
func (c *Client) GetSettings(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/setting/all")
}

// UpdateSettings updates panel settings.
func (c *Client) UpdateSettings(ctx context.Context, data map[string]any) (*Response, error) {
	return c.PostJSON(ctx, "panel/setting/update", data)
}

// GetDefaultXrayConfig returns the default Xray configuration template.
func (c *Client) GetDefaultXrayConfig(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/setting/getDefaultJsonConfig")
}

// RestartPanel restarts the 3x-ui panel itself.
func (c *Client) RestartPanel(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/setting/restartPanel")
}
