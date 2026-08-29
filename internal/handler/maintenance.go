package handler

import (
	"context"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// maintenanceHandler holds the XUI client for the panel upkeep tools: health
// checks, certificate helpers, geo data, panel updates and the settings probes.
type maintenanceHandler struct {
	client *xui.Client
}

func registerMaintenanceTools(s *server.MCPServer, client *xui.Client) {
	h := &maintenanceHandler{client: client}

	s.AddTool(mcp.NewTool("get_fail2ban_status",
		readsPanel,
		mcp.WithDescription("Report whether per-client IP limits can actually be enforced on this host. Enforcement needs fail2ban installed and usable, so a client's limitIp is inert when this says otherwise."),
	), h.fail2ban)

	s.AddTool(mcp.NewTool("get_web_cert_files",
		readsPanel,
		mcp.WithDescription("Return this panel's own web TLS certificate and key file paths."),
	), h.webCertFiles)

	s.AddTool(mcp.NewTool("get_cert_hash",
		readsPanel,
		mcp.WithDescription("Compute the SHA-256 of a certificate for pinning (the pinned_peer_cert_sha256 an inbound or host group takes). Pass a path on the panel host, or the certificate inline."),
		mcp.WithString("cert_file",
			mcp.Description("Path to a certificate file on the panel host"),
		),
		mcp.WithString("cert_content",
			mcp.Description("Certificate content in PEM or DER, as an alternative to cert_file"),
		),
	), h.certHash)

	s.AddTool(mcp.NewTool("get_remote_cert_hash",
		probesRemote,
		mcp.WithDescription("Run `xray tls ping` against a remote server and return its live leaf-certificate SHA-256 — the value to pin when you cannot copy the certificate itself."),
		mcp.WithString("server",
			mcp.Required(),
			mcp.Description("Target as host:port, e.g. example.com:443"),
		),
	), h.remoteCertHash)

	s.AddTool(mcp.NewTool("generate_ech_cert",
		readsPanel,
		mcp.WithDescription("Generate an ECH (Encrypted Client Hello) keypair and config list for one SNI. Nothing is stored — the result goes into an inbound's stream settings."),
		mcp.WithString("sni",
			mcp.Required(),
			mcp.Description("Server name the ECH config is issued for"),
		),
	), h.echCert)

	s.AddTool(mcp.NewTool("update_geofile",
		fetchesRemote,
		mcp.WithDescription("Download fresh GeoIP/GeoSite data files. Without a file name the default set is refreshed; the geodata tools read what lands on disk."),
		mcp.WithString("file_name",
			mcp.Description("Single file to refresh, e.g. geoip.dat or geosite.dat. Omit to refresh the default set."),
		),
	), h.updateGeofile)

	s.AddTool(mcp.NewTool("get_panel_update_status",
		readsPanel,
		mcp.WithDescription("Report how the last panel self-update ended: run ID, state and exit code. Compare the run ID against the one update_panel returned to tell a finished run from a stale result."),
	), h.updateStatus)

	s.AddTool(mcp.NewTool("set_update_channel",
		updatesPanel,
		mcp.WithDescription("Switch the panel between the stable channel and the rolling per-commit dev release. Only effective on a panel already running a dev build."),
		mcp.WithBoolean("dev",
			mcp.Required(),
			mcp.Description("true for the dev channel, false for stable"),
		),
	), h.setUpdateChannel)

	s.AddTool(mcp.NewTool("update_panel",
		installsRemote,
		mcp.WithDescription("Self-update the 3x-ui panel to the latest release. The panel restarts on success, so this connection drops and every tool here fails until it is back. Poll get_panel_update_status afterwards with the returned run ID."),
	), h.updatePanel)

	s.AddTool(mcp.NewTool("get_amneziawg_logs",
		readsPanel,
		mcp.WithDescription("Return live AmneziaWG peer activity — last handshake, endpoint, transfer per peer — plus the panel's own AmneziaWG event lines."),
		mcp.WithNumber("count",
			mcp.Description("Maximum peer rows and event lines to return"),
			mcp.DefaultNumber(50),
		),
	), h.amneziawgLogs)

	s.AddTool(mcp.NewTool("get_node_descendants",
		readsPanel,
		mcp.WithDescription("Summarize the nodes below this panel in the cluster tree: guid, parent, name, address, status and versions. This is the view a node reports upward to its parent panel."),
	), h.descendants)

	s.AddTool(mcp.NewTool("get_client_ips_table",
		readsPanel,
		mcp.WithDescription("Return the aggregated client-IP table — the cluster-wide record of which addresses used which client, and the data per-client IP limits are enforced against."),
	), h.clientIPsTable)

	s.AddTool(mcp.NewTool("backup_to_telegram",
		probesRemote,
		mcp.WithDescription("Send a fresh database backup to every Telegram chat configured as an admin recipient. The backup contains every client credential, so it goes wherever those chats are."),
	), h.backupToTelegram)

	s.AddTool(mcp.NewTool("test_smtp",
		probesRemote,
		mcp.WithDescription("Test the configured SMTP settings by connecting, authenticating and sending, and report which stage was reached."),
	), h.testSMTP)

	s.AddTool(mcp.NewTool("test_telegram_bot",
		probesRemote,
		mcp.WithDescription("Send a test message through the configured Telegram bot to check the token and chat ID."),
	), h.testTelegramBot)

	s.AddTool(mcp.NewTool("validate_regex",
		readsPanel,
		mcp.WithDescription("Compile a regular expression with the panel's Go RE2 engine without saving it. Worth running before putting a pattern into a routing rule, since RE2 rejects backreferences and lookarounds that other flavours allow."),
		mcp.WithString("regex",
			mcp.Required(),
			mcp.Description("The pattern to compile"),
		),
	), h.validateRegex)

	s.AddTool(mcp.NewTool("get_default_settings",
		readsPanel,
		mcp.WithDescription("Preview the settings a fresh install would compute for this host. Reads only — nothing is applied."),
	), h.defaultSettings)

	s.AddTool(mcp.NewTool("get_factory_defaults",
		readsPanel,
		mcp.WithDescription("Return the shipped default for each setting key, so a stored value can be told apart from the default it would fall back to. Reads only."),
	), h.factoryDefaults)

	s.AddTool(mcp.NewTool("update_admin_credentials",
		interruptsService,
		mcp.WithDescription("Change the panel admin username and password, verifying the current pair first. Everything authenticating with the old credentials stops working immediately — including this MCP server, unless XUI_USERNAME and XUI_PASSWORD are updated and it is restarted."),
		mcp.WithString("old_username",
			mcp.Required(),
			mcp.Description("Current admin username"),
		),
		mcp.WithString("old_password",
			mcp.Required(),
			mcp.Description("Current admin password"),
		),
		mcp.WithString("new_username",
			mcp.Required(),
			mcp.Description("New admin username"),
		),
		mcp.WithString("new_password",
			mcp.Required(),
			mcp.Description("New admin password"),
		),
	), h.updateAdminUser)
}

func (h *maintenanceHandler) fail2ban(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.Fail2banStatus(ctx))
}

func (h *maintenanceHandler) webCertFiles(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.WebCertFiles(ctx))
}

func (h *maintenanceHandler) certHash(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	certFile := req.GetString("cert_file", "")
	certContent := req.GetString("cert_content", "")
	if certFile == "" && certContent == "" {
		return mcp.NewToolResultError("pass cert_file or cert_content"), nil
	}
	return toResult(h.client.CertHash(ctx, certFile, certContent))
}

func (h *maintenanceHandler) remoteCertHash(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := req.RequireString("server")
	if err != nil {
		return mcp.NewToolResultError("server is required, as host:port"), nil
	}
	return toResult(h.client.RemoteCertHash(ctx, target))
}

func (h *maintenanceHandler) echCert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sni, err := req.RequireString("sni")
	if err != nil {
		return mcp.NewToolResultError("sni is required"), nil
	}
	return toResult(h.client.NewEchCert(ctx, sni))
}

func (h *maintenanceHandler) updateGeofile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.UpdateGeofile(ctx, req.GetString("file_name", "")))
}

func (h *maintenanceHandler) updateStatus(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.PanelUpdateStatus(ctx))
}

func (h *maintenanceHandler) setUpdateChannel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dev, err := req.RequireBool("dev")
	if err != nil {
		return mcp.NewToolResultError("dev is required"), nil
	}
	return toResult(h.client.SetUpdateChannel(ctx, dev))
}

func (h *maintenanceHandler) updatePanel(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.UpdatePanel(ctx))
}

func (h *maintenanceHandler) amneziawgLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.AmneziaWGLogs(ctx, int(req.GetFloat("count", 50))))
}

func (h *maintenanceHandler) descendants(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.Descendants(ctx))
}

func (h *maintenanceHandler) clientIPsTable(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ClientIPsTable(ctx))
}

func (h *maintenanceHandler) backupToTelegram(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.BackupToTelegram(ctx))
}

func (h *maintenanceHandler) testSMTP(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.TestSMTP(ctx))
}

func (h *maintenanceHandler) testTelegramBot(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.TestTelegramBot(ctx))
}

func (h *maintenanceHandler) validateRegex(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	regex, err := req.RequireString("regex")
	if err != nil {
		return mcp.NewToolResultError("regex is required"), nil
	}
	return toResult(h.client.ValidateRegex(ctx, regex))
}

func (h *maintenanceHandler) defaultSettings(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.DefaultSettings(ctx))
}

func (h *maintenanceHandler) factoryDefaults(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.FactorySettings(ctx))
}

func (h *maintenanceHandler) updateAdminUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	oldUsername, err := req.RequireString("old_username")
	if err != nil {
		return mcp.NewToolResultError("old_username is required"), nil
	}
	oldPassword, err := req.RequireString("old_password")
	if err != nil {
		return mcp.NewToolResultError("old_password is required"), nil
	}
	newUsername, err := req.RequireString("new_username")
	if err != nil {
		return mcp.NewToolResultError("new_username is required"), nil
	}
	newPassword, err := req.RequireString("new_password")
	if err != nil {
		return mcp.NewToolResultError("new_password is required"), nil
	}
	return toResult(h.client.UpdateAdminUser(ctx, oldUsername, oldPassword, newUsername, newPassword))
}
