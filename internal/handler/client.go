package handler

import (
	"context"
	"encoding/json"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// clientHandler holds the XUI client for client management tool handlers.
type clientHandler struct {
	client *xui.Client
}

func registerClientTools(s *server.MCPServer, client *xui.Client) {
	h := &clientHandler{client: client}

	s.AddTool(mcp.NewTool("add_client",
		mcp.WithDescription("Add a new client (user) to an existing inbound. For VMess/VLESS inbounds, a UUID is auto-generated if not provided. For Trojan/Shadowsocks, provide a password instead."),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Target inbound ID"),
		),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Unique client identifier/email"),
		),
		mcp.WithString("uuid",
			mcp.Description("Client UUID (for VMess/VLESS). Auto-generated if empty"),
		),
		mcp.WithString("password",
			mcp.Description("Client password (for Trojan/Shadowsocks)"),
		),
		mcp.WithString("flow",
			mcp.Description("XTLS flow setting (for VLESS, e.g. 'xtls-rprx-vision')"),
		),
		mcp.WithNumber("total_gb",
			mcp.Description("Traffic limit in GB (0 = unlimited)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("expiry_time",
			mcp.Description("Expiry as Unix timestamp in milliseconds (0 = never)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("limit_ip",
			mcp.Description("Max simultaneous IP connections (0 = unlimited)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithBoolean("enable",
			mcp.Description("Enable the client immediately"),
			mcp.DefaultBool(true),
		),
		mcp.WithNumber("tg_id",
			mcp.Description("Telegram user ID for notifications"),
			mcp.DefaultNumber(0),
		),
		mcp.WithString("sub_id",
			mcp.Description("Subscription ID for subscription links"),
		),
		mcp.WithString("comment",
			mcp.Description("Optional comment about the client"),
		),
	), h.add)

	s.AddTool(mcp.NewTool("update_client",
		mcp.WithDescription("Update an existing client's configuration. Provide the client_id (UUID for VMess/VLESS, or the original client ID) and the inbound_id it belongs to."),
		mcp.WithString("client_id",
			mcp.Required(),
			mcp.Description("Client UUID/ID to update"),
		),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Inbound ID the client belongs to"),
		),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email (must be provided even if unchanged)"),
		),
		mcp.WithString("uuid",
			mcp.Description("New UUID (for VMess/VLESS)"),
		),
		mcp.WithString("password",
			mcp.Description("New password (for Trojan/Shadowsocks)"),
		),
		mcp.WithString("flow",
			mcp.Description("XTLS flow setting"),
		),
		mcp.WithNumber("total_gb",
			mcp.Description("Traffic limit in GB (0 = unlimited)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("expiry_time",
			mcp.Description("Expiry as Unix timestamp in ms (0 = never)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("limit_ip",
			mcp.Description("Max simultaneous IPs (0 = unlimited)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithBoolean("enable",
			mcp.Description("Enable/disable the client"),
			mcp.DefaultBool(true),
		),
		mcp.WithNumber("tg_id",
			mcp.Description("Telegram user ID"),
			mcp.DefaultNumber(0),
		),
		mcp.WithString("sub_id",
			mcp.Description("Subscription ID"),
		),
		mcp.WithString("comment",
			mcp.Description("Comment"),
		),
	), h.update)

	s.AddTool(mcp.NewTool("delete_client",
		mcp.WithDescription("Remove a client from an inbound by client UUID/ID."),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
		mcp.WithString("client_id",
			mcp.Required(),
			mcp.Description("Client UUID/ID to delete"),
		),
	), h.delete)

	s.AddTool(mcp.NewTool("delete_client_by_email",
		mcp.WithDescription("Remove a client from an inbound by email address."),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email to delete"),
		),
	), h.deleteByEmail)

	s.AddTool(mcp.NewTool("get_client_traffic",
		mcp.WithDescription("Get upload/download traffic statistics for a client by email. Returns current usage, limits, and enable status."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.getTraffic)

	s.AddTool(mcp.NewTool("get_client_traffic_by_id",
		mcp.WithDescription("Get traffic statistics for a client by their UUID."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Client UUID"),
		),
	), h.getTrafficByID)

	s.AddTool(mcp.NewTool("get_client_ips",
		mcp.WithDescription("Get IP addresses recorded for a client, with timestamps."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.getIPs)

	s.AddTool(mcp.NewTool("clear_client_ips",
		mcp.WithDescription("Clear all recorded IP addresses for a client."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.clearIPs)

	s.AddTool(mcp.NewTool("reset_client_traffic",
		mcp.WithDescription("Reset traffic counters (upload/download) for a specific client to zero."),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.resetTraffic)

	s.AddTool(mcp.NewTool("reset_all_traffics",
		mcp.WithDescription("Reset all traffic counters across all inbounds. Use with caution."),
	), h.resetAllTraffics)

	s.AddTool(mcp.NewTool("reset_all_client_traffics",
		mcp.WithDescription("Reset traffic counters for all clients within a specific inbound."),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
	), h.resetAllClientTraffics)

	s.AddTool(mcp.NewTool("delete_depleted_clients",
		mcp.WithDescription("Remove all clients from an inbound that have exhausted their traffic or expired."),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
	), h.deleteDepleted)

	s.AddTool(mcp.NewTool("get_online_clients",
		mcp.WithDescription("Get a list of currently connected/active clients."),
	), h.getOnline)

	s.AddTool(mcp.NewTool("update_client_traffic",
		mcp.WithDescription("Set specific upload/download byte values for a client's traffic counter."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
		mcp.WithNumber("upload",
			mcp.Required(),
			mcp.Description("Upload bytes"),
		),
		mcp.WithNumber("download",
			mcp.Required(),
			mcp.Description("Download bytes"),
		),
	), h.updateTraffic)
}

func (h *clientHandler) buildClient(req mcp.CallToolRequest) xui.ClientConfig {
	uuid := req.GetString("uuid", "")
	password := req.GetString("password", "")

	// Auto-generate UUID for VMess/VLESS if neither uuid nor password provided
	if uuid == "" && password == "" {
		uuid = generateUUID()
	}

	return xui.ClientConfig{
		ID:         uuid,
		Password:   password,
		Flow:       req.GetString("flow", ""),
		Email:      req.GetString("email", ""),
		LimitIP:    int(req.GetFloat("limit_ip", 0)),
		TotalGB:    int64(req.GetFloat("total_gb", 0)) * 1073741824, // GB to bytes
		ExpiryTime: int64(req.GetFloat("expiry_time", 0)),
		Enable:     req.GetBool("enable", true),
		TgID:       int64(req.GetFloat("tg_id", 0)),
		SubID:      req.GetString("sub_id", ""),
		Comment:    req.GetString("comment", ""),
	}
}

func (h *clientHandler) add(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inboundID, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}

	client := h.buildClient(req)
	client.Email = email

	settings, err := buildClientSettings(client)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data := map[string]any{
		"id":       int(inboundID),
		"settings": settings,
	}

	resp, apiErr := h.client.AddClient(ctx, data)
	result, _ := toResult(resp, apiErr)

	// On success, include the generated UUID/password for reference
	if apiErr == nil && resp != nil && resp.Success {
		info := map[string]any{
			"message": resp.Msg,
			"email":   email,
		}
		if client.ID != "" {
			info["uuid"] = client.ID
		}
		if client.Password != "" {
			info["password"] = client.Password
		}
		out, _ := json.MarshalIndent(info, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	}

	return result, nil
}

func (h *clientHandler) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clientID, err := req.RequireString("client_id")
	if err != nil {
		return mcp.NewToolResultError("client_id is required"), nil
	}
	inboundID, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}

	client := h.buildClient(req)
	if client.ID == "" {
		client.ID = clientID
	}

	settings, err := buildClientSettings(client)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data := map[string]any{
		"id":       int(inboundID),
		"settings": settings,
	}

	return toResult(h.client.UpdateClient(ctx, clientID, data))
}

func (h *clientHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inboundID, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}
	clientID, err := req.RequireString("client_id")
	if err != nil {
		return mcp.NewToolResultError("client_id is required"), nil
	}
	return toResult(h.client.DeleteClient(ctx, int(inboundID), clientID))
}

func (h *clientHandler) deleteByEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inboundID, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.DeleteClientByEmail(ctx, int(inboundID), email))
}

func (h *clientHandler) getTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.GetClientTraffic(ctx, email))
}

func (h *clientHandler) getTrafficByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.GetClientTrafficByID(ctx, id))
}

func (h *clientHandler) getIPs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.GetClientIPs(ctx, email))
}

func (h *clientHandler) clearIPs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.ClearClientIPs(ctx, email))
}

func (h *clientHandler) resetTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inboundID, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.ResetClientTraffic(ctx, int(inboundID), email))
}

func (h *clientHandler) resetAllTraffics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ResetAllTraffics(ctx))
}

func (h *clientHandler) resetAllClientTraffics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inboundID, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}
	return toResult(h.client.ResetAllClientTraffics(ctx, int(inboundID)))
}

func (h *clientHandler) deleteDepleted(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inboundID, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}
	return toResult(h.client.DeleteDepletedClients(ctx, int(inboundID)))
}

func (h *clientHandler) getOnline(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GetOnlineClients(ctx))
}

func (h *clientHandler) updateTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	upload, err := req.RequireFloat("upload")
	if err != nil {
		return mcp.NewToolResultError("upload is required"), nil
	}
	download, err := req.RequireFloat("download")
	if err != nil {
		return mcp.NewToolResultError("download is required"), nil
	}
	return toResult(h.client.UpdateClientTraffic(ctx, email, int64(upload), int64(download)))
}

