package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// inboundHandler holds the XUI client for inbound tool handlers.
type inboundHandler struct {
	client *xui.Client
}

func registerInboundTools(s *server.MCPServer, client *xui.Client) {
	h := &inboundHandler{client: client}

	s.AddTool(mcp.NewTool("list_inbounds",
		mcp.WithDescription("List all inbound connections configured in the 3x-ui panel. Returns array of inbounds with their ports, protocols, remarks, traffic stats, and client statistics."),
	), h.list)

	s.AddTool(mcp.NewTool("get_inbound",
		mcp.WithDescription("Get detailed information about a specific inbound by its ID. Includes protocol settings, stream settings, sniffing config, and per-client stats."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
	), h.get)

	s.AddTool(mcp.NewTool("create_inbound",
		mcp.WithDescription("Create a new inbound connection. The 'settings', 'stream_settings', and 'sniffing' parameters are JSON strings matching 3x-ui format. Example settings for VLESS: {\"clients\":[{\"id\":\"uuid\",\"flow\":\"xtls-rprx-vision\",\"email\":\"user1\",\"limitIp\":0,\"totalGB\":0,\"expiryTime\":0,\"enable\":true,\"tgId\":\"\",\"subId\":\"\"}],\"decryption\":\"none\",\"fallbacks\":[]}"),
		mcp.WithString("remark",
			mcp.Required(),
			mcp.Description("Display name for the inbound"),
		),
		mcp.WithNumber("port",
			mcp.Required(),
			mcp.Description("Listen port"),
		),
		mcp.WithString("protocol",
			mcp.Required(),
			mcp.Description("Protocol: vmess, vless, trojan, shadowsocks, dokodemo-door, socks, http"),
		),
		mcp.WithString("settings",
			mcp.Required(),
			mcp.Description("Protocol-specific settings as JSON string (includes clients array)"),
		),
		mcp.WithString("stream_settings",
			mcp.Description("Transport/stream settings as JSON string. Default: TCP with no TLS"),
			mcp.DefaultString(`{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`),
		),
		mcp.WithString("sniffing",
			mcp.Description("Sniffing configuration as JSON string"),
			mcp.DefaultString(`{"enabled":true,"destOverride":["http","tls","quic","fakedns"],"metadataOnly":false,"routeOnly":false}`),
		),
		mcp.WithString("listen",
			mcp.Description("Listen address (empty = all interfaces)"),
			mcp.DefaultString(""),
		),
		mcp.WithBoolean("enable",
			mcp.Description("Enable the inbound immediately"),
			mcp.DefaultBool(true),
		),
		mcp.WithNumber("expiry_time",
			mcp.Description("Expiry time as Unix timestamp in milliseconds (0 = never)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("total",
			mcp.Description("Total traffic limit in bytes (0 = unlimited)"),
			mcp.DefaultNumber(0),
		),
	), h.create)

	s.AddTool(mcp.NewTool("update_inbound",
		mcp.WithDescription("Update an existing inbound. Pass only the fields you want to change. Fields: remark, port, protocol, listen, enable, settings, stream_settings, sniffing, expiry_time, total."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID to update"),
		),
		mcp.WithString("data",
			mcp.Required(),
			mcp.Description("JSON object with fields to update. Example: {\"remark\":\"new name\",\"enable\":false}"),
		),
	), h.update)

	s.AddTool(mcp.NewTool("delete_inbound",
		mcp.WithDescription("Permanently delete an inbound and all its associated client data. This action cannot be undone."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID to delete"),
		),
	), h.delete)
}

func (h *inboundHandler) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListInbounds(ctx))
}

func (h *inboundHandler) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.GetInbound(ctx, int(id)))
}

func (h *inboundHandler) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	remark, err := req.RequireString("remark")
	if err != nil {
		return mcp.NewToolResultError("remark is required"), nil
	}
	port, err := req.RequireFloat("port")
	if err != nil {
		return mcp.NewToolResultError("port is required"), nil
	}
	protocol, err := req.RequireString("protocol")
	if err != nil {
		return mcp.NewToolResultError("protocol is required"), nil
	}
	settings, err := req.RequireString("settings")
	if err != nil {
		return mcp.NewToolResultError("settings is required"), nil
	}

	streamSettings := req.GetString("stream_settings", `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`)
	sniffing := req.GetString("sniffing", `{"enabled":true,"destOverride":["http","tls","quic","fakedns"],"metadataOnly":false,"routeOnly":false}`)
	listen := req.GetString("listen", "")
	enable := req.GetBool("enable", true)
	expiryTime := req.GetFloat("expiry_time", 0)
	total := req.GetFloat("total", 0)

	data := map[string]any{
		"remark":         remark,
		"port":           int(port),
		"protocol":       protocol,
		"settings":       settings,
		"streamSettings": streamSettings,
		"sniffing":       sniffing,
		"listen":         listen,
		"enable":         enable,
		"expiryTime":     int64(expiryTime),
		"total":          int64(total),
	}

	return toResult(h.client.CreateInbound(ctx, data))
}

func (h *inboundHandler) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	dataStr, err := req.RequireString("data")
	if err != nil {
		return mcp.NewToolResultError("data JSON is required"), nil
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid JSON in data: %s", err)), nil
	}

	return toResult(h.client.UpdateInbound(ctx, int(id), data))
}

func (h *inboundHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.DeleteInbound(ctx, int(id)))
}
