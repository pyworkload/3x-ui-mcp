package handler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

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
		writesPanel,
		mcp.WithDescription("Add a new client (user) and attach it to one or more inbounds. Clients are first-class, email-keyed entities. A UUID is auto-generated for VMess/VLESS if not provided; for Trojan/Shadowsocks/Hysteria the panel generates the key server-side when omitted. Field meanings and units: read the xui://docs/client-fields resource."),
		mcp.WithArray("inbound_ids",
			mcp.Required(),
			mcp.Description("IDs of the inbounds to attach this client to (at least one)"),
			mcp.WithNumberItems(),
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
		mcp.WithString("security",
			mcp.Description("Security/cipher method (e.g. Shadowsocks encryption method)"),
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
			mcp.Description("Subscription ID for subscription links (auto-generated if empty)"),
		),
		mcp.WithString("group",
			mcp.Description("Logical grouping label"),
		),
		mcp.WithString("comment",
			mcp.Description("Optional comment about the client"),
		),
	), h.add)

	s.AddTool(mcp.NewTool("update_client",
		updatesPanel,
		mcp.WithDescription("Update an existing client by email. Only the fields you pass are changed; everything else (including the UUID/password) is preserved by reading the current client first. Optionally restrict the update to specific inbounds via inbound_ids."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Email of the client to update"),
		),
		mcp.WithString("new_email",
			mcp.Description("Rename the client to this email"),
		),
		mcp.WithArray("inbound_ids",
			mcp.Description("Restrict the update to these inbound attachments (default: all of the client's inbounds)"),
			mcp.WithNumberItems(),
		),
		mcp.WithString("uuid",
			mcp.Description("New UUID (for VMess/VLESS)"),
		),
		mcp.WithString("password",
			mcp.Description("New password (for Trojan/Shadowsocks)"),
		),
		mcp.WithString("security",
			mcp.Description("Security/cipher method"),
		),
		mcp.WithString("flow",
			mcp.Description("XTLS flow setting"),
		),
		mcp.WithNumber("total_gb",
			mcp.Description("Traffic limit in GB (0 = unlimited)"),
		),
		mcp.WithNumber("expiry_time",
			mcp.Description("Expiry as Unix timestamp in ms (0 = never)"),
		),
		mcp.WithNumber("limit_ip",
			mcp.Description("Max simultaneous IPs (0 = unlimited)"),
		),
		mcp.WithBoolean("enable",
			mcp.Description("Enable/disable the client"),
		),
		mcp.WithNumber("tg_id",
			mcp.Description("Telegram user ID"),
		),
		mcp.WithString("sub_id",
			mcp.Description("Subscription ID"),
		),
		mcp.WithString("group",
			mcp.Description("Logical grouping label"),
		),
		mcp.WithString("comment",
			mcp.Description("Comment"),
		),
		mcp.WithNumber("reset",
			mcp.Description("Traffic auto-reset period in days (0 = off)"),
		),
	), h.update)

	s.AddTool(mcp.NewTool("delete_client",
		destroysPanel,
		mcp.WithDescription("Remove a client (from all its inbounds) by email."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email to delete"),
		),
		mcp.WithBoolean("keep_traffic",
			mcp.Description("Preserve the client's traffic statistics after deletion"),
			mcp.DefaultBool(false),
		),
	), h.delete)

	s.AddTool(mcp.NewTool("get_client",
		readsPanel,
		mcp.WithDescription("Get a single client's full configuration (UUID, limits, settings) and the inbounds it is attached to, by email."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.get)

	s.AddTool(mcp.NewTool("list_clients",
		readsPanel,
		mcp.WithDescription("List clients with pagination, search and filtering. Returns a compact page plus totals and a summary."),
		mcp.WithString("search",
			mcp.Description("Substring match on email/subId/comment"),
		),
		mcp.WithNumber("page",
			mcp.Description("1-based page number"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Items per page (default 25, max 200)"),
		),
		mcp.WithString("protocol",
			mcp.Description("Filter by inbound protocol (e.g. vless, vmess, trojan). Comma-separated for multiple"),
		),
		mcp.WithString("inbound",
			mcp.Description("Filter by inbound id/remark. Comma-separated for multiple"),
		),
		mcp.WithString("group",
			mcp.Description("Filter by group label"),
		),
		mcp.WithString("filter",
			mcp.Description("Status filter (e.g. active, depleted, expiring, deactive). Comma-separated for multiple"),
		),
		mcp.WithString("sort",
			mcp.Description("Sort field (e.g. email, expiryTime, totalGB)"),
		),
		mcp.WithString("order",
			mcp.Description("Sort order: asc or desc"),
		),
	), h.list)

	s.AddTool(mcp.NewTool("attach_client",
		updatesPanel,
		mcp.WithDescription("Attach an existing client to additional inbounds, by email."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
		mcp.WithArray("inbound_ids",
			mcp.Required(),
			mcp.Description("Inbound IDs to attach the client to"),
			mcp.WithNumberItems(),
		),
	), h.attach)

	s.AddTool(mcp.NewTool("detach_client",
		destroysPanel,
		mcp.WithDescription("Detach a client from the given inbounds, by email. The client itself is kept (use delete_client to remove it entirely)."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
		mcp.WithArray("inbound_ids",
			mcp.Required(),
			mcp.Description("Inbound IDs to detach the client from"),
			mcp.WithNumberItems(),
		),
	), h.detach)

	s.AddTool(mcp.NewTool("bulk_create_clients",
		writesPanel,
		mcp.WithDescription("Create many clients at once across the same set of inbounds, sharing common limits. Each client gets an auto-generated UUID."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to create"),
			mcp.WithStringItems(),
		),
		mcp.WithArray("inbound_ids",
			mcp.Required(),
			mcp.Description("Inbound IDs to attach every created client to"),
			mcp.WithNumberItems(),
		),
		mcp.WithNumber("total_gb",
			mcp.Description("Shared traffic limit in GB (0 = unlimited)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("expiry_time",
			mcp.Description("Shared expiry as Unix ms (0 = never)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("limit_ip",
			mcp.Description("Shared max simultaneous IPs (0 = unlimited)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithBoolean("enable",
			mcp.Description("Enable the clients immediately"),
			mcp.DefaultBool(true),
		),
		mcp.WithString("flow",
			mcp.Description("Shared XTLS flow setting (VLESS)"),
		),
		mcp.WithString("group",
			mcp.Description("Shared group label"),
		),
	), h.bulkCreate)

	s.AddTool(mcp.NewTool("bulk_delete_clients",
		destroysPanel,
		mcp.WithDescription("Delete many clients at once by email."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to delete"),
			mcp.WithStringItems(),
		),
		mcp.WithBoolean("keep_traffic",
			mcp.Description("Preserve traffic statistics after deletion"),
			mcp.DefaultBool(false),
		),
	), h.bulkDelete)

	s.AddTool(mcp.NewTool("get_client_traffic",
		readsPanel,
		mcp.WithDescription("Get upload/download traffic statistics for a client by email. Returns current usage, limits, and enable status."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.getTraffic)

	s.AddTool(mcp.NewTool("get_client_ips",
		readsPanel,
		mcp.WithDescription("Get IP addresses recorded for a client, with timestamps."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.getIPs)

	s.AddTool(mcp.NewTool("clear_client_ips",
		destroysPanel,
		mcp.WithDescription("Clear all recorded IP addresses for a client."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.clearIPs)

	s.AddTool(mcp.NewTool("reset_client_traffic",
		destroysPanel,
		mcp.WithDescription("Reset traffic counters (upload/download) for a specific client to zero, by email."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.resetTraffic)

	s.AddTool(mcp.NewTool("reset_all_traffics",
		destroysPanel,
		mcp.WithDescription("Reset all traffic counters across all inbounds. Use with caution."),
	), h.resetAllTraffics)

	s.AddTool(mcp.NewTool("reset_all_client_traffics",
		destroysPanel,
		mcp.WithDescription("Reset traffic counters for every client panel-wide. Use with caution."),
	), h.resetAllClientTraffics)

	s.AddTool(mcp.NewTool("bulk_reset_traffic",
		destroysPanel,
		mcp.WithDescription("Reset traffic counters for a specific set of clients, by email."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients whose traffic to reset"),
			mcp.WithStringItems(),
		),
	), h.bulkResetTraffic)

	s.AddTool(mcp.NewTool("delete_depleted_clients",
		destroysPanel,
		mcp.WithDescription("Remove all clients panel-wide that have exhausted their traffic or expired."),
	), h.deleteDepleted)

	s.AddTool(mcp.NewTool("get_online_clients",
		readsPanel,
		mcp.WithDescription("Get a list of currently connected/active clients."),
	), h.getOnline)

	s.AddTool(mcp.NewTool("get_last_online",
		readsPanel,
		mcp.WithDescription("Get the last-online timestamp for every client."),
	), h.getLastOnline)

	s.AddTool(mcp.NewTool("update_client_traffic",
		destroysPanel,
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

	s.AddTool(mcp.NewTool("get_subscription_links",
		readsPanel,
		mcp.WithDescription("Get the connection URLs served under a subscription ID — every enabled client whose subId matches, one URL per inbound (and per external proxy where configured). "+
			"Same set as the public /sub/<subId> endpoint, but as a JSON array instead of base64. A client's subId comes from get_client or list_clients."),
		mcp.WithString("sub_id",
			mcp.Required(),
			mcp.Description("Subscription ID (the client's subId field)"),
		),
	), h.getSubscriptionLinks)

	s.AddTool(mcp.NewTool("get_clients_by_telegram_id",
		readsPanel,
		mcp.WithDescription("Look up clients by Telegram user ID. Answers with an array, since several clients can share one Telegram account."),
		mcp.WithNumber("tg_id",
			mcp.Required(),
			mcp.Description("Telegram user ID (numeric)"),
		),
	), h.getByTelegramID)

	s.AddTool(mcp.NewTool("list_client_devices",
		readsPanel,
		mcp.WithDescription("List the HWID devices registered for a client: first and last seen, user agent, OS and model. The hashes themselves are never exposed."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.listDevices)

	s.AddTool(mcp.NewTool("delete_client_device",
		destroysPanel,
		mcp.WithDescription("Remove one registered device from a client, freeing a single slot under its HWID limit."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
		mcp.WithNumber("device_id",
			mcp.Required(),
			mcp.Description("Device id from list_client_devices"),
		),
	), h.deleteDevice)

	s.AddTool(mcp.NewTool("clear_client_devices",
		destroysPanel,
		mcp.WithDescription("Drop every registered device for a client so new ones can register — the fix for a user who changed phones and hit the HWID limit."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
	), h.clearDevices)

	s.AddTool(mcp.NewTool("bulk_enable_clients",
		updatesPanel,
		mcp.WithDescription("Enable many clients at once. Emails are grouped by inbound, so each inbound is rewritten once. Clients that do not exist come back in a 'skipped' list rather than failing the call."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to enable"),
			mcp.WithStringItems(),
		),
	), h.bulkEnable)

	s.AddTool(mcp.NewTool("bulk_disable_clients",
		updatesPanel,
		mcp.WithDescription("Disable many clients at once, one rewrite per owning inbound. Their configuration is kept, so bulk_enable_clients reverses it."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to disable"),
			mcp.WithStringItems(),
		),
	), h.bulkDisable)

	s.AddTool(mcp.NewTool("bulk_adjust_clients",
		writesPanel,
		mcp.WithDescription("Shift expiry and traffic quota for many clients in one call — the bulk renewal. Both deltas may be negative. Clients on unlimited expiry or unlimited traffic are reported as skipped instead of being given a limit."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to adjust"),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("add_days",
			mcp.Description("Days to add to each expiry date; negative shortens it"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("add_gb",
			mcp.Description("GB to add to each traffic quota; negative reduces it"),
			mcp.DefaultNumber(0),
		),
		mcp.WithString("flow",
			mcp.Description("Set this flow on every listed client (e.g. 'xtls-rprx-vision'). Omit to leave each client's flow alone."),
		),
	), h.bulkAdjust)

	s.AddTool(mcp.NewTool("delete_orphan_clients",
		destroysPanel,
		mcp.WithDescription("Delete every client attached to no inbound, together with its traffic record, IP log, devices and external links. Cleans up after inbounds were removed without their clients."),
	), h.deleteOrphans)

}

func (h *clientHandler) getSubscriptionLinks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subID, err := req.RequireString("sub_id")
	if err != nil {
		return mcp.NewToolResultError("sub_id is required"), nil
	}
	return toResult(h.client.GetSubscriptionLinks(ctx, subID))
}

// buildClientConfig assembles a ClientConfig from the request params (no auto-generation).
func (h *clientHandler) buildClientConfig(req mcp.CallToolRequest) xui.ClientConfig {
	return xui.ClientConfig{
		ID:         req.GetString("uuid", ""),
		Security:   req.GetString("security", ""),
		Password:   req.GetString("password", ""),
		Flow:       req.GetString("flow", ""),
		Auth:       req.GetString("auth", ""),
		Email:      req.GetString("email", ""),
		LimitIP:    int(req.GetFloat("limit_ip", 0)),
		TotalGB:    int64(req.GetFloat("total_gb", 0)) * bytesPerGB,
		ExpiryTime: int64(req.GetFloat("expiry_time", 0)),
		Enable:     req.GetBool("enable", true),
		TgID:       int64(req.GetFloat("tg_id", 0)),
		SubID:      req.GetString("sub_id", ""),
		Group:      req.GetString("group", ""),
		Comment:    req.GetString("comment", ""),
		Reset:      int(req.GetFloat("reset", 0)),
	}
}

func (h *clientHandler) add(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	if len(inboundIDs) == 0 {
		return mcp.NewToolResultError("inbound_ids is required (at least one inbound)"), nil
	}
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}

	client := h.buildClientConfig(req)
	client.Email = email

	// Auto-generate a UUID when no key material is supplied so we can echo it
	// back. (The panel also fills protocol defaults server-side per inbound.)
	if client.ID == "" && client.Password == "" && client.Auth == "" {
		client.ID = generateUUID()
	}

	payload := xui.ClientCreatePayload{Client: client, InboundIds: inboundIDs}
	resp, apiErr := h.client.AddClient(ctx, payload)
	result, _ := toResult(resp, apiErr)

	if apiErr == nil && resp != nil && resp.Success {
		info := map[string]any{
			"message":     resp.Msg,
			"email":       email,
			"inbound_ids": inboundIDs,
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
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}

	// Read the current client so omitted params are preserved. Without this, an
	// empty UUID/password field would make the panel regenerate the client key.
	cur, apiErr := h.client.GetClient(ctx, email)
	if apiErr != nil {
		return mcp.NewToolResultError(apiErr.Error()), nil
	}
	if !cur.Success {
		return mcp.NewToolResultError("API error: " + cur.Msg), nil
	}
	client, perr := parseClient(cur)
	if perr != nil {
		return mcp.NewToolResultError(perr.Error()), nil
	}

	args := req.GetArguments()

	// String fields: GetString falls back to the existing value when absent.
	client.ID = req.GetString("uuid", client.ID)
	client.Password = req.GetString("password", client.Password)
	client.Security = req.GetString("security", client.Security)
	client.Flow = req.GetString("flow", client.Flow)
	client.SubID = req.GetString("sub_id", client.SubID)
	client.Group = req.GetString("group", client.Group)
	client.Comment = req.GetString("comment", client.Comment)
	client.Email = req.GetString("new_email", client.Email)

	// Numeric/bool fields: only override when explicitly provided.
	if _, ok := args["limit_ip"]; ok {
		client.LimitIP = int(req.GetFloat("limit_ip", 0))
	}
	if _, ok := args["total_gb"]; ok {
		client.TotalGB = int64(req.GetFloat("total_gb", 0)) * bytesPerGB
	}
	if _, ok := args["expiry_time"]; ok {
		client.ExpiryTime = int64(req.GetFloat("expiry_time", 0))
	}
	if _, ok := args["enable"]; ok {
		client.Enable = req.GetBool("enable", true)
	}
	if _, ok := args["tg_id"]; ok {
		client.TgID = int64(req.GetFloat("tg_id", 0))
	}
	if _, ok := args["reset"]; ok {
		client.Reset = int(req.GetFloat("reset", 0))
	}

	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	return toResult(h.client.UpdateClient(ctx, email, client, inboundIDs))
}

func (h *clientHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	keepTraffic := req.GetBool("keep_traffic", false)
	return toResult(h.client.DeleteClient(ctx, email, keepTraffic))
}

func (h *clientHandler) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.GetClient(ctx, email))
}

func (h *clientHandler) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := url.Values{}
	if v := req.GetString("search", ""); v != "" {
		query.Set("search", v)
	}
	if v := req.GetInt("page", 0); v > 0 {
		query.Set("page", strconv.Itoa(v))
	}
	if v := req.GetInt("page_size", 0); v > 0 {
		query.Set("pageSize", strconv.Itoa(v))
	}
	if v := req.GetString("protocol", ""); v != "" {
		query.Set("protocol", v)
	}
	if v := req.GetString("inbound", ""); v != "" {
		query.Set("inbound", v)
	}
	if v := req.GetString("group", ""); v != "" {
		query.Set("group", v)
	}
	if v := req.GetString("filter", ""); v != "" {
		query.Set("filter", v)
	}
	if v := req.GetString("sort", ""); v != "" {
		query.Set("sort", v)
	}
	if v := req.GetString("order", ""); v != "" {
		query.Set("order", v)
	}
	return toResult(h.client.ListClientsPaged(ctx, query))
}

func (h *clientHandler) attach(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	if len(inboundIDs) == 0 {
		return mcp.NewToolResultError("inbound_ids is required (at least one inbound)"), nil
	}
	return toResult(h.client.AttachClient(ctx, email, inboundIDs))
}

func (h *clientHandler) detach(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	if len(inboundIDs) == 0 {
		return mcp.NewToolResultError("inbound_ids is required (at least one inbound)"), nil
	}
	return toResult(h.client.DetachClient(ctx, email, inboundIDs))
}

func (h *clientHandler) bulkCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	if len(inboundIDs) == 0 {
		return mcp.NewToolResultError("inbound_ids is required (at least one inbound)"), nil
	}

	limitIP := int(req.GetFloat("limit_ip", 0))
	totalGB := int64(req.GetFloat("total_gb", 0)) * bytesPerGB
	expiry := int64(req.GetFloat("expiry_time", 0))
	enable := req.GetBool("enable", true)
	flow := req.GetString("flow", "")
	group := req.GetString("group", "")

	payloads := make([]xui.ClientCreatePayload, 0, len(emails))
	generated := make(map[string]string, len(emails))
	for _, email := range emails {
		uuid := generateUUID()
		generated[email] = uuid
		payloads = append(payloads, xui.ClientCreatePayload{
			Client: xui.ClientConfig{
				ID:         uuid,
				Email:      email,
				Flow:       flow,
				Group:      group,
				LimitIP:    limitIP,
				TotalGB:    totalGB,
				ExpiryTime: expiry,
				Enable:     enable,
			},
			InboundIds: inboundIDs,
		})
	}

	resp, apiErr := h.client.BulkCreateClients(ctx, payloads)
	result, _ := toResult(resp, apiErr)
	if apiErr == nil && resp != nil && resp.Success {
		info := map[string]any{
			"message":     resp.Msg,
			"created":     generated, // email -> generated UUID (applies to VMess/VLESS)
			"inbound_ids": inboundIDs,
		}
		out, _ := json.MarshalIndent(info, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	}
	return result, nil
}

func (h *clientHandler) bulkDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	keepTraffic := req.GetBool("keep_traffic", false)
	return toResult(h.client.BulkDeleteClients(ctx, emails, keepTraffic))
}

func (h *clientHandler) getTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.GetClientTraffic(ctx, email))
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
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.ResetClientTraffic(ctx, email))
}

func (h *clientHandler) resetAllTraffics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ResetAllTraffics(ctx))
}

func (h *clientHandler) resetAllClientTraffics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ResetAllClientTraffics(ctx))
}

func (h *clientHandler) bulkResetTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	return toResult(h.client.BulkResetTraffic(ctx, emails))
}

func (h *clientHandler) deleteDepleted(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.DeleteDepletedClients(ctx))
}

func (h *clientHandler) getOnline(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GetOnlineClients(ctx))
}

func (h *clientHandler) getLastOnline(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GetLastOnline(ctx))
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

func (h *clientHandler) getByTelegramID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tgID, err := req.RequireFloat("tg_id")
	if err != nil {
		return mcp.NewToolResultError("tg_id is required"), nil
	}
	return toResult(h.client.GetClientsByTelegramID(ctx, int64(tgID)))
}

func (h *clientHandler) listDevices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.ListClientDevices(ctx, email))
}

func (h *clientHandler) deleteDevice(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	deviceID, err := req.RequireFloat("device_id")
	if err != nil {
		return mcp.NewToolResultError("device_id is required"), nil
	}
	return toResult(h.client.DeleteClientDevice(ctx, email, int(deviceID)))
}

func (h *clientHandler) clearDevices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.ClearClientDevices(ctx, email))
}

func (h *clientHandler) bulkEnable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	return toResult(h.client.BulkEnableClients(ctx, emails))
}

func (h *clientHandler) bulkDisable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	return toResult(h.client.BulkDisableClients(ctx, emails))
}

func (h *clientHandler) bulkAdjust(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	addDays := int(req.GetFloat("add_days", 0))
	addBytes := int64(req.GetFloat("add_gb", 0) * bytesPerGB)
	if addDays == 0 && addBytes == 0 && req.GetString("flow", "") == "" {
		return mcp.NewToolResultError("nothing to adjust: pass add_days, add_gb or flow"), nil
	}
	return toResult(h.client.BulkAdjustClients(ctx, emails, addDays, addBytes, req.GetString("flow", "")))
}

func (h *clientHandler) deleteOrphans(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.DeleteOrphanClients(ctx))
}
