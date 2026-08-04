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

// --- Key generation & Reality probing ---

// GetNewUUID returns a fresh UUID v4 from the panel, for use as a client ID.
func (c *Client) GetNewUUID(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getNewUUID")
}

// GetNewX25519Cert returns a new X25519 keypair for a Reality inbound.
func (c *Client) GetNewX25519Cert(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getNewX25519Cert")
}

// GetNewVlessEnc returns VLESS encryption auth options — each entry pairs the
// decryption string for the inbound with the encryption string for clients.
func (c *Client) GetNewVlessEnc(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getNewVlessEnc")
}

// GetNewMLKEM768 returns a new ML-KEM-768 keypair (post-quantum KEM).
func (c *Client) GetNewMLKEM768(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getNewmlkem768")
}

// GetNewMLDSA65 returns a new ML-DSA-65 keypair (post-quantum signature).
func (c *Client) GetNewMLDSA65(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getNewmldsa65")
}

// ScanRealityTarget probes one candidate Reality target over TLS and reports
// whether it is usable (TLS 1.3 + h2 + X25519 + trusted cert), along with the
// certificate's SAN DNS names. xver 0 leaves the PROXY-protocol version unset.
func (c *Client) ScanRealityTarget(ctx context.Context, target string, xver int) (*Response, error) {
	data := url.Values{"target": {target}}
	if xver > 0 {
		data.Set("xver", fmt.Sprintf("%d", xver))
	}
	return c.PostForm(ctx, "panel/api/server/scanRealityTarget", data)
}

// ScanRealityTargets probes several candidates at once and returns them ranked
// by feasibility then latency. Each comma-separated token may be a domain, a
// bare IP, or a CIDR range to discover.
func (c *Client) ScanRealityTargets(ctx context.Context, targets string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/server/scanRealityTargets", url.Values{
		"targets": {targets},
	})
}

// --- Settings API methods ---

// GetSettings returns all panel settings.
func (c *Client) GetSettings(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/setting/all")
}

// UpdateSettings updates panel settings.
func (c *Client) UpdateSettings(ctx context.Context, data map[string]any) (*Response, error) {
	return c.PostJSON(ctx, "panel/api/setting/update", data)
}

// GetDefaultXrayConfig returns the default Xray configuration template.
func (c *Client) GetDefaultXrayConfig(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/setting/getDefaultJsonConfig")
}

// RestartPanel restarts the 3x-ui panel itself.
func (c *Client) RestartPanel(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/setting/restartPanel")
}
