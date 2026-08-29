package xui

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
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

// --- Maintenance and diagnostics (panel v3.5.0+) ---

// Fail2banStatus reports whether per-client IP limits can be enforced here,
// which depends on fail2ban being installed and usable.
func (c *Client) Fail2banStatus(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/fail2banStatus")
}

// WebCertFiles returns this panel's own web TLS certificate and key paths.
func (c *Client) WebCertFiles(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getWebCertFiles")
}

// CertHash computes the SHA-256 of a certificate for pinning. Pass either a
// path on the panel host or the certificate content inline.
func (c *Client) CertHash(ctx context.Context, certFile, certContent string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/server/getCertHash", url.Values{
		"certFile":    {certFile},
		"certContent": {certContent},
	})
}

// RemoteCertHash runs `xray tls ping` against a remote server and returns its
// live leaf certificate hashes.
func (c *Client) RemoteCertHash(ctx context.Context, server string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/server/getRemoteCertHash", url.Values{"server": {server}})
}

// NewEchCert generates an ECH keypair and config list for one SNI.
func (c *Client) NewEchCert(ctx context.Context, sni string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/server/getNewEchCert", url.Values{"sni": {sni}})
}

// UpdateGeofile refreshes the GeoIP/GeoSite data files. An empty fileName
// refreshes the default set; otherwise the name goes in the path.
func (c *Client) UpdateGeofile(ctx context.Context, fileName string) (*Response, error) {
	if fileName == "" {
		return c.Post(ctx, "panel/api/server/updateGeofile")
	}
	return c.Post(ctx, "panel/api/server/updateGeofile/"+url.PathEscape(fileName))
}

// PanelUpdateStatus reports how the last self-update run ended.
func (c *Client) PanelUpdateStatus(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getUpdateStatus")
}

// SetUpdateChannel switches the panel between the stable and dev channels.
func (c *Client) SetUpdateChannel(ctx context.Context, dev bool) (*Response, error) {
	return c.PostForm(ctx, "panel/api/server/setUpdateChannel", url.Values{
		"dev": {strconv.FormatBool(dev)},
	})
}

// UpdatePanel self-updates the panel; the server restarts on success.
func (c *Client) UpdatePanel(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/server/updatePanel")
}

// AmneziaWGLogs returns live AmneziaWG peer activity plus the panel's own
// AmneziaWG event lines.
func (c *Client) AmneziaWGLogs(ctx context.Context, count int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/server/amneziawglogs/%d", count))
}

// Descendants summarizes the nodes this panel manages, as a node reports them
// to its parent.
func (c *Client) Descendants(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/descendants")
}

// ClientIPsTable returns the aggregated inbound_client_ips table, the
// cluster-wide view behind per-client IP limits.
func (c *Client) ClientIPsTable(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/clientIps")
}

// BackupToTelegram sends a fresh database backup to every configured admin chat.
func (c *Client) BackupToTelegram(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/backuptotgbot")
}

// TestSMTP runs the SMTP connection test, reporting the stage it reached.
func (c *Client) TestSMTP(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/setting/testSmtp")
}

// TestTelegramBot sends a test message to the configured Telegram chat.
func (c *Client) TestTelegramBot(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/setting/testTgBot")
}

// ValidateRegex compiles a regular expression with the panel's Go RE2 engine
// without saving it anywhere.
func (c *Client) ValidateRegex(ctx context.Context, regex string) (*Response, error) {
	return c.PostJSON(ctx, "panel/api/setting/validateRegex", map[string]any{"regex": regex})
}

// DefaultSettings returns the settings a fresh install would compute for this
// host — a preview, not a write.
func (c *Client) DefaultSettings(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/setting/defaultSettings")
}

// FactorySettings returns the shipped default per setting key, so a stored
// value can be told apart from the default it would fall back to.
func (c *Client) FactorySettings(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/setting/factoryDefaults")
}

// UpdateAdminUser changes the panel admin credentials, verifying the current
// ones first. Anything authenticating with the old pair stops working.
func (c *Client) UpdateAdminUser(ctx context.Context, oldUsername, oldPassword, newUsername, newPassword string) (*Response, error) {
	return c.PostJSON(ctx, "panel/api/setting/updateUser", map[string]any{
		"oldUsername": oldUsername,
		"oldPassword": oldPassword,
		"newUsername": newUsername,
		"newPassword": newPassword,
	})
}
