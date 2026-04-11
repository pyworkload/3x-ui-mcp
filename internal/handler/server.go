package handler

import (
	"context"

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
		mcp.WithDescription("Get current server resource usage: CPU, memory, disk, swap, network throughput, uptime, Xray version and state."),
	), h.status)

	s.AddTool(mcp.NewTool("restart_xray",
		mcp.WithDescription("Restart the Xray proxy service. Applies any pending configuration changes. All active connections will be briefly interrupted."),
	), h.restartXray)

	s.AddTool(mcp.NewTool("stop_xray",
		mcp.WithDescription("Stop the Xray proxy service. All proxy connections will be terminated until the service is restarted."),
	), h.stopXray)

	s.AddTool(mcp.NewTool("get_xray_config",
		mcp.WithDescription("Get the current active Xray JSON configuration. Shows inbounds, outbounds, routing rules, and other Xray settings."),
	), h.getXrayConfig)

	s.AddTool(mcp.NewTool("get_xray_versions",
		mcp.WithDescription("Get a list of available Xray versions that can be installed."),
	), h.getXrayVersions)

	s.AddTool(mcp.NewTool("install_xray",
		mcp.WithDescription("Install or switch to a specific Xray version."),
		mcp.WithString("version",
			mcp.Required(),
			mcp.Description("Xray version to install (e.g. 'v1.8.24')"),
		),
	), h.installXray)

	s.AddTool(mcp.NewTool("get_logs",
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
		mcp.WithDescription("Get all 3x-ui panel settings: web server config, Telegram bot, subscription, security, and more."),
	), h.getSettings)

	s.AddTool(mcp.NewTool("get_default_xray_config",
		mcp.WithDescription("Get the default Xray configuration template used by the panel."),
	), h.getDefaultXrayConfig)

	s.AddTool(mcp.NewTool("restart_panel",
		mcp.WithDescription("Restart the 3x-ui panel itself. The panel will be unavailable for a few seconds during restart."),
	), h.restartPanel)
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
	return toResult(h.client.GetXrayConfig(ctx))
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
