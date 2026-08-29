package handler

import (
	"context"
	"net/url"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// providerHandler holds the XUI client for the outbound provider integrations.
//
// Warp, NordVPN and PIA each sit behind one panel route with the operation in
// the path. They are split here into a read tool and a write tool per provider
// so the annotations stay honest: querying a country list is not the same kind
// of call as deleting the stored credentials.
type providerHandler struct {
	client *xui.Client
}

func registerProviderTools(s *server.MCPServer, client *xui.Client) {
	h := &providerHandler{client: client}

	s.AddTool(mcp.NewTool("get_warp_data",
		probesRemote,
		mcp.WithDescription("Read the Cloudflare Warp integration: 'data' returns the account and its quota, 'config' returns the WireGuard configuration the panel holds."),
		mcp.WithString("action",
			mcp.Description("What to read"),
			mcp.Enum("data", "config"),
			mcp.DefaultString("data"),
		),
	), h.warpRead)

	s.AddTool(mcp.NewTool("manage_warp",
		managesRemote,
		mcp.WithDescription("Change the Cloudflare Warp integration: 'reg' registers an account, 'changeIp' rotates the endpoint IP and marks Xray for restart, 'license' applies a Warp+ license, 'interval' sets the auto-rotation period in days (0 disables), 'del' erases the stored Warp data."),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Operation to run"),
			mcp.Enum("reg", "changeIp", "license", "interval", "del"),
		),
		mcp.WithString("private_key",
			mcp.Description("WireGuard private key, for action=reg"),
		),
		mcp.WithString("public_key",
			mcp.Description("WireGuard public key, for action=reg"),
		),
		mcp.WithString("license",
			mcp.Description("Warp+ license key, for action=license"),
		),
		mcp.WithNumber("interval",
			mcp.Description("Auto-rotation interval in days, for action=interval; 0 disables it"),
		),
	), h.warpWrite)

	s.AddTool(mcp.NewTool("get_nordvpn_data",
		probesRemote,
		mcp.WithDescription("Read the NordVPN integration: 'countries' lists available countries, 'servers' lists the servers in one country (pass country_id), 'data' returns the stored account state."),
		mcp.WithString("action",
			mcp.Description("What to read"),
			mcp.Enum("countries", "servers", "data"),
			mcp.DefaultString("data"),
		),
		mcp.WithString("country_id",
			mcp.Description("Country ID from action=countries, required for action=servers"),
		),
	), h.nordRead)

	s.AddTool(mcp.NewTool("manage_nordvpn",
		managesRemote,
		mcp.WithDescription("Change the NordVPN integration: 'reg' exchanges an access token for credentials, 'setKey' stores a WireGuard private key directly, 'del' erases the stored data."),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Operation to run"),
			mcp.Enum("reg", "setKey", "del"),
		),
		mcp.WithString("token",
			mcp.Description("NordVPN access token, for action=reg"),
		),
		mcp.WithString("key",
			mcp.Description("WireGuard private key, for action=setKey"),
		),
	), h.nordWrite)

	s.AddTool(mcp.NewTool("get_pia_data",
		probesRemote,
		mcp.WithDescription("Read the PIA WireGuard integration: 'countries' lists the regions from PIA's signed server list, 'servers' lists the servers in one region (pass country_code), 'data' returns the stored account state."),
		mcp.WithString("action",
			mcp.Description("What to read"),
			mcp.Enum("countries", "servers", "data"),
			mcp.DefaultString("data"),
		),
		mcp.WithString("country_code",
			mcp.Description("Region code from action=countries, required for action=servers"),
		),
	), h.piaRead)

	s.AddTool(mcp.NewTool("manage_pia",
		managesRemote,
		mcp.WithDescription("Change the PIA WireGuard integration: 'reg' logs in with a PIA username and password, 'addKey' registers a WireGuard key against one server hostname, 'del' erases the stored data."),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Operation to run"),
			mcp.Enum("reg", "addKey", "del"),
		),
		mcp.WithString("username",
			mcp.Description("PIA username, for action=reg"),
		),
		mcp.WithString("password",
			mcp.Description("PIA password, for action=reg"),
		),
		mcp.WithString("hostname",
			mcp.Description("Server hostname from get_pia_data servers, for action=addKey"),
		),
	), h.piaWrite)
}

func (h *providerHandler) warpRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.WarpAction(ctx, req.GetString("action", "data"), nil))
}

func (h *providerHandler) warpWrite(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action, err := req.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError("action is required"), nil
	}
	params := url.Values{}
	switch action {
	case "reg":
		params.Set("privateKey", req.GetString("private_key", ""))
		params.Set("publicKey", req.GetString("public_key", ""))
	case "license":
		license := req.GetString("license", "")
		if license == "" {
			return mcp.NewToolResultError("license is required for action=license"), nil
		}
		params.Set("license", license)
	case "interval":
		if _, ok := req.GetArguments()["interval"]; !ok {
			return mcp.NewToolResultError("interval is required for action=interval"), nil
		}
		params.Set("interval", formatInt(int(req.GetFloat("interval", 0))))
	}
	return toResult(h.client.WarpAction(ctx, action, params))
}

func (h *providerHandler) nordRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := req.GetString("action", "data")
	params := url.Values{}
	if action == "servers" {
		countryID := req.GetString("country_id", "")
		if countryID == "" {
			return mcp.NewToolResultError("country_id is required for action=servers"), nil
		}
		params.Set("countryId", countryID)
	}
	return toResult(h.client.NordAction(ctx, action, params))
}

func (h *providerHandler) nordWrite(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action, err := req.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError("action is required"), nil
	}
	params := url.Values{}
	switch action {
	case "reg":
		token := req.GetString("token", "")
		if token == "" {
			return mcp.NewToolResultError("token is required for action=reg"), nil
		}
		params.Set("token", token)
	case "setKey":
		key := req.GetString("key", "")
		if key == "" {
			return mcp.NewToolResultError("key is required for action=setKey"), nil
		}
		params.Set("key", key)
	}
	return toResult(h.client.NordAction(ctx, action, params))
}

func (h *providerHandler) piaRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := req.GetString("action", "data")
	params := url.Values{}
	if action == "servers" {
		code := req.GetString("country_code", "")
		if code == "" {
			return mcp.NewToolResultError("country_code is required for action=servers"), nil
		}
		params.Set("countryCode", code)
	}
	return toResult(h.client.PiaAction(ctx, action, params))
}

func (h *providerHandler) piaWrite(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action, err := req.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError("action is required"), nil
	}
	params := url.Values{}
	switch action {
	case "reg":
		username := req.GetString("username", "")
		password := req.GetString("password", "")
		if username == "" || password == "" {
			return mcp.NewToolResultError("username and password are required for action=reg"), nil
		}
		params.Set("username", username)
		params.Set("password", password)
	case "addKey":
		hostname := req.GetString("hostname", "")
		if hostname == "" {
			return mcp.NewToolResultError("hostname is required for action=addKey"), nil
		}
		params.Set("hostname", hostname)
	}
	return toResult(h.client.PiaAction(ctx, action, params))
}
