package handler

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// serverHandler holds the XUI client for server management tool handlers.
type serverHandler struct {
	client *xui.Client
}

func registerServerTools(s *server.MCPServer, client *xui.Client) {
	h := &serverHandler{client: client}

	s.AddTool(mcp.NewTool("server_status",
		readsPanel,
		mcp.WithDescription("Get current server resource usage: CPU, memory, disk, swap, network throughput, uptime, Xray version and state."),
	), h.status)

	s.AddTool(mcp.NewTool("restart_xray",
		interruptsService,
		mcp.WithDescription("Restart the Xray proxy service. Applies any pending configuration changes. All active connections will be briefly interrupted."),
	), h.restartXray)

	s.AddTool(mcp.NewTool("stop_xray",
		interruptsService,
		mcp.WithDescription("Stop the Xray proxy service. All proxy connections will be terminated until the service is restarted."),
	), h.stopXray)

	s.AddTool(mcp.NewTool("get_xray_config",
		readsPanel,
		mcp.WithDescription("Summarize the Xray configuration the core is running: how many inbounds, outbounds, routing rules and DNS servers it has, plus a link to the full JSON at xui://xray/config. Pass full=true to get the whole document inline instead."),
		mcp.WithBoolean("full",
			mcp.Description("Return the entire configuration inline instead of a summary and a link"),
			mcp.DefaultBool(false),
		),
	), h.getXrayConfig)

	s.AddTool(mcp.NewTool("get_xray_versions",
		probesRemote,
		mcp.WithDescription("Get a list of available Xray versions that can be installed."),
	), h.getXrayVersions)

	s.AddTool(mcp.NewTool("install_xray",
		installsRemote,
		mcp.WithDescription("Install or switch to a specific Xray version."),
		mcp.WithString("version",
			mcp.Required(),
			mcp.Description("Xray version to install (e.g. 'v1.8.24')"),
		),
	), h.installXray)

	s.AddTool(mcp.NewTool("get_logs",
		readsPanel,
		mcp.WithDescription("Get application logs from the 3x-ui panel."),
		mcp.WithNumber("count",
			mcp.Description("Number of log lines to retrieve"),
			mcp.DefaultNumber(50),
		),
		mcp.WithString("level",
			mcp.Description("Log level filter: debug, info, warning, error"),
		),
	), h.getLogs)

	s.AddTool(mcp.NewTool("get_xray_logs",
		readsPanel,
		mcp.WithDescription("Get Xray proxy access/error logs with optional filtering."),
		mcp.WithNumber("count",
			mcp.Description("Number of log lines to retrieve"),
			mcp.DefaultNumber(50),
		),
		mcp.WithString("filter",
			mcp.Description("Text filter to search within logs"),
		),
	), h.getXrayLogs)

	s.AddTool(mcp.NewTool("get_settings",
		readsPanel,
		mcp.WithDescription("Get all 3x-ui panel settings: web server config, Telegram bot, subscription, security, and more."),
	), h.getSettings)

	s.AddTool(mcp.NewTool("get_default_xray_config",
		readsPanel,
		mcp.WithDescription("Get the default Xray configuration template used by the panel."),
	), h.getDefaultXrayConfig)

	s.AddTool(mcp.NewTool("restart_panel",
		interruptsService,
		mcp.WithDescription("Restart the 3x-ui panel itself. The panel will be unavailable for a few seconds during restart."),
	), h.restartPanel)

	// --- Key material & Reality targets ---

	s.AddTool(mcp.NewTool("generate_key",
		readsPanel,
		mcp.WithDescription("Generate key material with the panel's own generators — use this instead of inventing values when building an inbound. Types:\n"+
			"- 'uuid': a UUID v4 for a client id\n"+
			"- 'x25519': X25519 keypair for Reality (privateKey goes into the inbound, publicKey into client links)\n"+
			"- 'vless_enc': VLESS encryption auth options; each entry pairs a decryption string for the inbound with the encryption string for clients\n"+
			"- 'mlkem768': ML-KEM-768 keypair (post-quantum KEM)\n"+
			"- 'mldsa65': ML-DSA-65 keypair (post-quantum signature)"),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Enum("uuid", "x25519", "vless_enc", "mlkem768", "mldsa65"),
			mcp.Description("What to generate"),
		),
	), h.generateKey)

	s.AddTool(mcp.NewTool("scan_reality_target",
		probesRemote,
		mcp.WithDescription("Probe one candidate REALITY target over live TLS and report whether it is usable: TLS 1.3, HTTP/2, X25519 and a trusted certificate, plus the certificate's SAN DNS names (candidates for serverNames)."),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Domain or host:port, e.g. 'www.cloudflare.com' or 'www.cloudflare.com:443'"),
		),
		mcp.WithNumber("xver",
			mcp.Description("PROXY protocol version to probe with (0 = none)"),
		),
	), h.scanRealityTarget)

	s.AddTool(mcp.NewTool("scan_reality_targets",
		probesRemote,
		mcp.WithDescription("Probe several REALITY candidates at once and get them ranked by feasibility, then latency. Each comma-separated token may be a domain (checked with SNI), a bare IP, or a CIDR range to discover by reading the certificates it serves."),
		mcp.WithString("targets",
			mcp.Required(),
			mcp.Description("Comma-separated domains, IPs or CIDR ranges, e.g. 'www.cloudflare.com,dl.google.com,104.16.0.0/24'"),
		),
	), h.scanRealityTargets)
}

func (h *serverHandler) status(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ServerStatus(ctx))
}

func (h *serverHandler) restartXray(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.RestartXray(ctx))
}

func (h *serverHandler) stopXray(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.StopXray(ctx))
}

func (h *serverHandler) getXrayConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.GetXrayConfig(ctx)
	if req.GetBool("full", false) {
		return toResult(resp, err)
	}
	return linkedResult(resp, err, mcp.NewResourceLink(
		xrayConfigURI,
		"Running Xray configuration",
		"The full config the core is running. Read it to inspect inbounds, outbounds, routing or DNS in detail.",
		"application/json",
	), summarizeXrayConfig)
}

func (h *serverHandler) getXrayVersions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GetXrayVersions(ctx))
}

func (h *serverHandler) installXray(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return mcp.NewToolResultError("version is required"), nil
	}
	return toResult(h.client.InstallXray(ctx, version))
}

func (h *serverHandler) getLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := int(req.GetFloat("count", 50))
	level := req.GetString("level", "")
	return toResult(h.client.GetLogs(ctx, count, level, ""))
}

func (h *serverHandler) getXrayLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := int(req.GetFloat("count", 50))
	filter := req.GetString("filter", "")
	return toResult(h.client.GetXrayLogs(ctx, count, filter))
}

func (h *serverHandler) getSettings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GetSettings(ctx))
}

func (h *serverHandler) getDefaultXrayConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GetDefaultXrayConfig(ctx))
}

func (h *serverHandler) restartPanel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.RestartPanel(ctx))
}

func (h *serverHandler) generateKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	kind, err := req.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError("type is required"), nil
	}

	generate, ok := keyGenerators[kind]
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("unknown type %q — expected one of: %s", kind, strings.Join(keyGeneratorNames(), ", "))), nil
	}
	return toResult(generate(h.client, ctx))
}

// keyGenerators maps the generate_key type parameter to the panel endpoint
// that produces it.
var keyGenerators = map[string]func(*xui.Client, context.Context) (*xui.Response, error){
	"uuid":      (*xui.Client).GetNewUUID,
	"x25519":    (*xui.Client).GetNewX25519Cert,
	"vless_enc": (*xui.Client).GetNewVlessEnc,
	"mlkem768":  (*xui.Client).GetNewMLKEM768,
	"mldsa65":   (*xui.Client).GetNewMLDSA65,
}

// keyGeneratorNames lists the supported types in a stable order, for errors.
func keyGeneratorNames() []string {
	names := make([]string, 0, len(keyGenerators))
	for name := range keyGenerators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (h *serverHandler) scanRealityTarget(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := req.RequireString("target")
	if err != nil {
		return mcp.NewToolResultError("target is required"), nil
	}
	return toResult(h.client.ScanRealityTarget(ctx, target, int(req.GetFloat("xver", 0))))
}

func (h *serverHandler) scanRealityTargets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	targets, err := req.RequireString("targets")
	if err != nil {
		return mcp.NewToolResultError("targets is required"), nil
	}
	return toResult(h.client.ScanRealityTargets(ctx, targets))
}
