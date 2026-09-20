package handler

import (
	"context"
	"encoding/json"
	"fmt"
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
		append([]mcp.ToolOption{
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
			mcp.WithNumber("reset",
				mcp.Description("Traffic auto-reset period in days (0 = off)"),
			),
		}, clientFieldParams()...)...,
	), h.add)

	s.AddTool(mcp.NewTool("update_client",
		append([]mcp.ToolOption{
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
		}, clientFieldParams()...)...,
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

	s.AddTool(mcp.NewTool("get_client_links",
		readsPanel,
		mcp.WithDescription("Get the connection URLs for one client, one per inbound it is attached to. Keyed by email, so it answers for that client alone — unlike get_subscription_links, which is keyed by subId and returns the links of every client sharing it. Works for a client with no subId."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email (the client key)"),
		),
	), h.getClientLinks)

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

	s.AddTool(mcp.NewTool("export_clients",
		readsPanel,
		mcp.WithDescription("Count the clients in the panel and link to the full export at xui://clients/export. The export is the {client, inboundIds} array import_clients takes, so it round-trips — but it carries every UUID and password, so it stays behind the link unless full=true is passed."),
		mcp.WithBoolean("full",
			mcp.Description("Return the whole export inline instead of a count and a link"),
			mcp.DefaultBool(false),
		),
	), h.export)

	s.AddTool(mcp.NewTool("import_clients",
		writesPanel,
		mcp.WithDescription("Create clients from an exported array. Items already present are reported in a 'skipped' list rather than overwritten, so an import cannot clobber a live client."),
		mcp.WithString("data",
			mcp.Required(),
			mcp.Description(`The export as a JSON string, e.g. [{"client":{"email":"alice","enable":true},"inboundIds":[7]}]`),
		),
	), h.importClients)

	s.AddTool(mcp.NewTool("set_client_external_links",
		destroysPanel,
		mcp.WithDescription("Replace a client's external links and external subscriptions — servers from elsewhere that this panel serves alongside its own. The panel swaps the whole set, so send every row you want kept; an empty array clears them."),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Client email"),
		),
		mcp.WithString("links",
			mcp.Required(),
			mcp.Description(`JSON array of rows, e.g. [{"kind":"link","value":"vless://...","remark":"DE","enable":true,"expiryTime":0},{"kind":"subscription","value":"https://provider.example/sub/abc","remark":"Provider","enable":true}]. Pass [] to remove them all.`),
		),
	), h.setExternalLinks)

	s.AddTool(mcp.NewTool("list_clients_paged",
		readsPanel,
		mcp.WithDescription("Filter, sort and page through clients on the server instead of pulling the whole list. Rows are slim — no UUID, password or flow — so this is the tool for finding clients on a large panel; use get_client for the credentials of one."),
		mcp.WithNumber("page",
			mcp.Description("1-indexed page number"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Rows per page, capped at 200"),
			mcp.DefaultNumber(25),
		),
		mcp.WithString("search",
			mcp.Description("Case-insensitive match on email, subId or comment"),
		),
		mcp.WithString("filter",
			mcp.Description("Status bucket to restrict to"),
			mcp.Enum("online", "active", "deactive", "depleted", "expiring"),
		),
		mcp.WithString("protocol",
			mcp.Description("Only clients attached to an inbound of this protocol"),
		),
		mcp.WithString("sort",
			mcp.Description("Sort key"),
			mcp.Enum("enable", "email", "inboundIds", "traffic", "remaining", "expiryTime"),
		),
		mcp.WithString("order",
			mcp.Description("Sort direction"),
			mcp.Enum("ascend", "descend"),
		),
	), h.listPaged)

	s.AddTool(mcp.NewTool("bulk_attach_clients",
		updatesPanel,
		mcp.WithDescription("Attach many existing clients to many inbounds in one call. Each client keeps its email, UUID, subId and its shared traffic row, so the same credentials work on every inbound it is attached to."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to attach"),
			mcp.WithStringItems(),
		),
		mcp.WithArray("inbound_ids",
			mcp.Required(),
			mcp.Description("Inbounds to attach them to"),
			mcp.WithNumberItems(),
		),
	), h.bulkAttach)

	s.AddTool(mcp.NewTool("bulk_detach_clients",
		destroysPanel,
		mcp.WithDescription("Detach many clients from many inbounds without deleting them. A client detached from its last inbound keeps existing but is served nowhere — delete_orphan_clients is what removes those."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to detach"),
			mcp.WithStringItems(),
		),
		mcp.WithArray("inbound_ids",
			mcp.Required(),
			mcp.Description("Inbounds to detach them from"),
			mcp.WithNumberItems(),
		),
	), h.bulkDetach)

	s.AddTool(mcp.NewTool("get_active_inbounds",
		readsPanel,
		mcp.WithDescription("List the inbound tags that carried traffic in the last heartbeat window, grouped by the node hosting them. Pairs with get_online_clients_by_node to see which node is actually serving what."),
	), h.activeInbounds)

	s.AddTool(mcp.NewTool("get_online_clients_by_node",
		readsPanel,
		mcp.WithDescription("List online client emails grouped by the node each one is connected to. get_online_clients answers the same question for this panel alone; this one attributes them across the cluster."),
	), h.onlinesByNode)

	s.AddTool(mcp.NewTool("get_client_ips_by_node",
		readsPanel,
		mcp.WithDescription("List per-client source IPs grouped by the node that observed them — the cluster-wide view behind per-client IP limits."),
	), h.clientIPsByNode)

}

func (h *clientHandler) getSubscriptionLinks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subID, err := req.RequireString("sub_id")
	if err != nil {
		return mcp.NewToolResultError("sub_id is required"), nil
	}
	return toResult(h.client.GetSubscriptionLinks(ctx, subID))
}

func (h *clientHandler) getClientLinks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	return toResult(h.client.GetClientLinks(ctx, email))
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

		ResetDay:        int(req.GetFloat("reset_day", 0)),
		ResetMax:        int(req.GetFloat("reset_max", 0)),
		TrafficReset:    req.GetString("traffic_reset", ""),
		TrafficResetDay: int(req.GetFloat("traffic_reset_day", 0)),
		LimitHwid:       int(req.GetFloat("limit_hwid", 0)),
		Secret:          req.GetString("secret", ""),
		AdTag:           req.GetString("ad_tag", ""),
		PrivateKey:      req.GetString("private_key", ""),
		PublicKey:       req.GetString("public_key", ""),
		PreSharedKey:    req.GetString("pre_shared_key", ""),
		AllowedIPs:      req.GetStringSlice("allowed_ips", nil),
		ForwardedPorts:  req.GetString("forwarded_ports", ""),
		Reverse:         reverseFromParam(req),
		KeepAlive:       keepAliveFromParam(req),
	}
}

// reverseFromParam builds the VLESS reverse object, or nil when the caller said
// nothing — the panel reads a missing key as "no reverse proxy".
func reverseFromParam(req mcp.CallToolRequest) *xui.ClientReverse {
	if tag := req.GetString("reverse_tag", ""); tag != "" {
		return &xui.ClientReverse{Tag: tag}
	}
	return nil
}

// keepAliveFromParam distinguishes an unset keep_alive from an explicit 0,
// which the panel reads as "send no keepalive packets".
func keepAliveFromParam(req mcp.CallToolRequest) *int {
	if _, ok := req.GetArguments()["keep_alive"]; !ok {
		return nil
	}
	seconds := int(req.GetFloat("keep_alive", 0))
	return &seconds
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

// clientFieldParams are the per-client panel fields both add_client and
// update_client accept. They were carried through update_client's
// read-modify-write long before they could be set, so this only opens the
// fields the panel already stored — no new shape reaches it.
func clientFieldParams() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithNumber("limit_hwid",
			mcp.Description("Max devices (HWIDs) that may register for this client, 0 = unlimited. See list_client_devices"),
		),
		mcp.WithNumber("reset_day",
			mcp.Description("Calendar day of the month the traffic quota renews, 1-31. 0 keeps the interval mode set by 'reset'"),
		),
		mcp.WithNumber("reset_max",
			mcp.Description("How many times the quota may auto-renew, 0 = unlimited"),
		),
		mcp.WithString("traffic_reset",
			mcp.Description("Per-client traffic reset cycle, independent of the inbound's own"),
			mcp.Enum("never", "hourly", "daily", "weekly", "monthly"),
		),
		mcp.WithNumber("traffic_reset_day",
			mcp.Description("Day of the cycle the per-client reset fires, 1-31"),
		),
		mcp.WithString("reverse_tag",
			mcp.Description("VLESS simple reverse proxy tag. Pass an empty string to clear it"),
		),
		mcp.WithString("secret",
			mcp.Description("MTProto per-client secret, used to build the tg://proxy link"),
		),
		mcp.WithString("ad_tag",
			mcp.Description("MTProto advertising tag from @MTProxybot: exactly 32 hex characters, or the panel rejects it"),
		),
		mcp.WithString("private_key",
			mcp.Description("WireGuard/AmneziaWG peer private key"),
		),
		mcp.WithString("public_key",
			mcp.Description("WireGuard/AmneziaWG peer public key"),
		),
		mcp.WithString("pre_shared_key",
			mcp.Description("WireGuard/AmneziaWG pre-shared key"),
		),
		mcp.WithArray("allowed_ips",
			mcp.Description("WireGuard/AmneziaWG peer addresses, e.g. [\"10.0.0.2/32\", \"fd00::2/128\"]"),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("keep_alive",
			mcp.Description("WireGuard PersistentKeepalive in seconds; 0 sends none. Omit to keep the stored value"),
		),
		mcp.WithString("forwarded_ports",
			mcp.Description("AmneziaWG per-client port forwarding, e.g. \"80,443,8000-8100\""),
		),
	}
}

// clientStringParams maps an update_client string parameter to the body key
// the panel reads it from.
var clientStringParams = map[string]string{
	"uuid":      "id",
	"password":  "password",
	"security":  "security",
	"flow":      "flow",
	"sub_id":    "subId",
	"group":     "group",
	"comment":   "comment",
	"new_email": "email",

	"traffic_reset":   "trafficReset",
	"secret":          "secret",
	"ad_tag":          "adTag",
	"private_key":     "privateKey",
	"public_key":      "publicKey",
	"pre_shared_key":  "preSharedKey",
	"forwarded_ports": "forwardedPorts",
}

// clientIntParams maps an update_client numeric parameter to its body key, for
// the ones that are a plain integer with no unit conversion.
var clientIntParams = map[string]string{
	"limit_hwid":        "limitHwid",
	"reset_day":         "resetDay",
	"reset_max":         "resetMax",
	"traffic_reset_day": "trafficResetDay",
	"keep_alive":        "keepAlive",
}

func (h *clientHandler) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}

	// Read the current client so omitted params are preserved. The panel's
	// update is a full replace, so without this an empty UUID/password field
	// would make it regenerate the client key, and every field the tool has no
	// parameter for — the WireGuard peer keys, the VLESS reverse tag, the HWID
	// limit — would be cleared.
	cur, apiErr := h.client.GetClient(ctx, email)
	if apiErr != nil {
		return mcp.NewToolResultError(apiErr.Error()), nil
	}
	if !cur.Success {
		return mcp.NewToolResultError("API error: " + cur.Msg), nil
	}
	body, perr := clientBaseFromRecord(cur.Obj)
	if perr != nil {
		return mcp.NewToolResultError(perr.Error()), nil
	}

	args := req.GetArguments()
	supplied := func(param string) bool {
		_, ok := args[param]
		return ok
	}

	for param, key := range clientStringParams {
		if supplied(param) {
			body[key] = req.GetString(param, "")
		}
	}
	if supplied("limit_ip") {
		body["limitIp"] = int(req.GetFloat("limit_ip", 0))
	}
	if supplied("total_gb") {
		body["totalGB"] = int64(req.GetFloat("total_gb", 0)) * bytesPerGB
	}
	if supplied("expiry_time") {
		body["expiryTime"] = int64(req.GetFloat("expiry_time", 0))
	}
	if supplied("enable") {
		body["enable"] = req.GetBool("enable", true)
	}
	if supplied("tg_id") {
		body["tgId"] = int64(req.GetFloat("tg_id", 0))
	}
	if supplied("reset") {
		body["reset"] = int(req.GetFloat("reset", 0))
	}
	for param, key := range clientIntParams {
		if supplied(param) {
			body[key] = int(req.GetFloat(param, 0))
		}
	}
	// An empty list is dropped rather than sent as [], the same way
	// clientBaseFromRecord treats one: the panel reads an absent allowedIPs as
	// "keep the address this inbound already has".
	if supplied("allowed_ips") {
		if ips := req.GetStringSlice("allowed_ips", nil); len(ips) > 0 {
			body["allowedIPs"] = ips
		} else {
			delete(body, "allowedIPs")
		}
	}
	// reverse is an object on the panel, and an empty tag is how a caller drops
	// it — sending {"tag": ""} would leave a reverse entry pointing nowhere.
	if supplied("reverse_tag") {
		if tag := req.GetString("reverse_tag", ""); tag != "" {
			body["reverse"] = map[string]any{"tag": tag}
		} else {
			delete(body, "reverse")
		}
	}

	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	return toResult(h.client.UpdateClient(ctx, email, body, inboundIDs))
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

func (h *clientHandler) export(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.ExportClients(ctx)
	if req.GetBool("full", false) {
		return toResult(resp, err)
	}
	return linkedResult(resp, err, mcp.NewResourceLink(
		clientsExportURI,
		"Client export",
		"Every client as {client, inboundIds}, credentials included. Read it to back up or to feed import_clients.",
		"application/json",
	), summarizeClientExport)
}

func (h *clientHandler) importClients(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	data, err := req.RequireString("data")
	if err != nil {
		return mcp.NewToolResultError("data is required"), nil
	}
	if !json.Valid([]byte(data)) {
		return mcp.NewToolResultError("data must be a JSON array of {client, inboundIds} objects"), nil
	}
	return toResult(h.client.ImportClients(ctx, data))
}

func (h *clientHandler) setExternalLinks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	raw, err := req.RequireString("links")
	if err != nil {
		return mcp.NewToolResultError("links is required"), nil
	}
	var links []map[string]any
	if err := json.Unmarshal([]byte(raw), &links); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("links must be a JSON array: %v", err)), nil
	}
	return toResult(h.client.SetClientExternalLinks(ctx, email, links))
}

func (h *clientHandler) listPaged(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := url.Values{
		"page":     {strconv.Itoa(int(req.GetFloat("page", 1)))},
		"pageSize": {strconv.Itoa(int(req.GetFloat("page_size", 25)))},
	}
	for param, key := range map[string]string{
		"search":   "search",
		"filter":   "filter",
		"protocol": "protocol",
		"sort":     "sort",
		"order":    "order",
	} {
		if v := req.GetString(param, ""); v != "" {
			query.Set(key, v)
		}
	}
	return toResult(h.client.ListClientsPaged(ctx, query))
}

func (h *clientHandler) bulkAttach(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	if len(inboundIDs) == 0 {
		return mcp.NewToolResultError("inbound_ids is required (at least one inbound)"), nil
	}
	return toResult(h.client.BulkAttachClients(ctx, emails, inboundIDs))
}

func (h *clientHandler) bulkDetach(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	if len(inboundIDs) == 0 {
		return mcp.NewToolResultError("inbound_ids is required (at least one inbound)"), nil
	}
	return toResult(h.client.BulkDetachClients(ctx, emails, inboundIDs))
}

func (h *clientHandler) activeInbounds(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ActiveInbounds(ctx))
}

func (h *clientHandler) onlinesByNode(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.OnlinesByNode(ctx))
}

func (h *clientHandler) clientIPsByNode(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ClientIPsByNode(ctx))
}
