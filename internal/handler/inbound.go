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
		readsPanel,
		mcp.WithDescription("List all inbound connections configured in the 3x-ui panel. Returns array of inbounds with their ports, protocols, remarks, traffic stats, and client statistics."),
	), h.list)

	s.AddTool(mcp.NewTool("get_inbound",
		readsPanel,
		mcp.WithDescription("Get detailed information about a specific inbound by its ID. Includes protocol settings, stream settings, sniffing config, and per-client stats."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
	), h.get)

	s.AddTool(mcp.NewTool("create_inbound",
		writesPanel,
		mcp.WithDescription("Create a new inbound connection. The 'settings', 'stream_settings' and 'sniffing' parameters are JSON strings in 3x-ui format; read the xui://docs/inbound-settings resource for a worked example per protocol and transport."),
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
		updatesPanel,
		mcp.WithDescription("Update an existing inbound. Pass only the fields you want to change — unspecified fields are preserved (read-modify-write against the current inbound). "+
			"Accepted fields: remark, port, protocol, listen, enable, settings, stream_settings (aka streamSettings), sniffing, expiry_time (aka expiryTime), total, tag. "+
			"snake_case and camelCase names are both accepted. "+
			"Refuses to write back if the merged result would have port=0 or empty protocol (which would silently disable the inbound)."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID to update"),
		),
		mcp.WithString("data",
			mcp.Required(),
			mcp.Description("JSON object with fields to change. Example: {\"remark\":\"new name\",\"enable\":false}"),
		),
	), h.update)

	s.AddTool(mcp.NewTool("delete_inbound",
		destroysPanel,
		mcp.WithDescription("Permanently delete an inbound and all its associated client data. This action cannot be undone."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID to delete"),
		),
	), h.delete)

	s.AddTool(mcp.NewTool("set_inbound_enable",
		updatesPanel,
		mcp.WithDescription("Enable or disable an inbound. Touches only the enable flag — preferred over update_inbound for a simple on/off, since it never rewrites the settings JSON."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
		mcp.WithBoolean("enable",
			mcp.Required(),
			mcp.Description("true to enable, false to disable"),
		),
	), h.setEnable)

	s.AddTool(mcp.NewTool("reset_inbound_traffic",
		destroysPanel,
		mcp.WithDescription("Zero the upload/download counters of a single inbound. Per-client counters are left untouched — use reset_client_traffic or bulk_reset_traffic for those."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
	), h.resetTraffic)

	s.AddTool(mcp.NewTool("delete_all_inbound_clients",
		destroysPanel,
		mcp.WithDescription("Remove every client attached to one inbound, keeping the inbound itself. Clients attached to other inbounds as well are deleted panel-wide, since clients are email-keyed entities. Cannot be undone."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
	), h.deleteAllClients)

	s.AddTool(mcp.NewTool("bulk_delete_inbounds",
		destroysPanel,
		mcp.WithDescription("Delete several inbounds in one call. The panel processes them sequentially, reports failures per id while the rest still proceed, and restarts Xray at most once. Cannot be undone."),
		mcp.WithString("ids",
			mcp.Required(),
			mcp.Description("Inbound IDs as a JSON array, e.g. [3,7,12]"),
		),
	), h.bulkDelete)

	s.AddTool(mcp.NewTool("get_all_inbound_links",
		readsPanel,
		mcp.WithDescription("Count the connection URLs (vless://, vmess://, trojan://, ss://, ...) for every client across every inbound and link to the full list at xui://inbounds/links. The URLs carry client credentials, so they stay behind the link unless full=true is passed."),
		mcp.WithBoolean("full",
			mcp.Description("Return every connection URL inline instead of a count and a link"),
			mcp.DefaultBool(false),
		),
	), h.allLinks)

	s.AddTool(mcp.NewTool("list_inbounds_slim",
		readsPanel,
		mcp.WithDescription("List inbounds with each client array stripped to email, enable and comment, and no UUID or subId enrichment. The cheap call for a panel with many clients — use list_inbounds when the credentials themselves are needed."),
	), h.listSlim)

	s.AddTool(mcp.NewTool("get_inbound_options",
		readsPanel,
		mcp.WithDescription("Picker projection of the inbounds: id, remark, tag, protocol, port and the server-computed capability flags (such as whether an inbound can carry XTLS flow). Cheaper than list_inbounds when only choosing one."),
	), h.options)

	s.AddTool(mcp.NewTool("get_inbound_fallbacks",
		readsPanel,
		mcp.WithDescription("List the fallback rules on a master VLESS/Trojan TCP-TLS inbound. Each rule routes traffic that matched no client to a child inbound, keyed by SNI, ALPN, path or dest."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Master inbound ID"),
		),
	), h.getFallbacks)

	s.AddTool(mcp.NewTool("set_inbound_fallbacks",
		destroysPanel,
		mcp.WithDescription("Replace the whole fallback list of a master inbound and restart Xray. This is not read-modify-write: whatever is passed becomes the complete set, so read get_inbound_fallbacks first and send the full list back with your change."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Master inbound ID"),
		),
		mcp.WithString("fallbacks",
			mcp.Required(),
			mcp.Description(`JSON array of fallback rules, e.g. [{"childId":11,"path":"/vlws","xver":2},{"childId":12,"alpn":"h2"}]. Pass [] to remove every fallback.`),
		),
	), h.setFallbacks)

	s.AddTool(mcp.NewTool("set_inbound_sub_sort_index",
		updatesPanel,
		mcp.WithDescription("Set an inbound's position in subscription output without touching anything else. Reads the stored inbound first, so reordering cannot revive a stale client list."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
		mcp.WithNumber("index",
			mcp.Required(),
			mcp.Description("Sort position; lower sorts first"),
		),
	), h.setSubSortIndex)

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

	var patch map[string]any
	if err := json.Unmarshal([]byte(dataStr), &patch); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid JSON in data: %s", err)), nil
	}

	// The 3x-ui update endpoint is a full replace, not a patch — any field
	// omitted from the body is zeroed out in the database. We fetch the
	// current inbound and merge the caller's patch on top so the tool's
	// "pass only the fields you want to change" contract holds.
	current, err := h.client.GetInbound(ctx, int(id))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("fetching current inbound %d: %s", int(id), err)), nil
	}
	if !current.Success {
		return mcp.NewToolResultError(fmt.Sprintf("API error fetching inbound %d: %s", int(id), current.Msg)), nil
	}

	merged, err := mergeInboundPatch(current.Obj, patch)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toResult(h.client.UpdateInbound(ctx, int(id), merged))
}

// inboundKeyAliases maps snake_case input keys (used by create_inbound and
// the tool schema) to the camelCase keys expected by the 3x-ui panel API.
// The update endpoint silently drops unknown keys, so "stream_settings" would
// blank streamSettings in the database unless we normalise it here.
var inboundKeyAliases = map[string]string{
	"stream_settings": "streamSettings",
	"expiry_time":     "expiryTime",
}

// normalizeInboundPatchKeys rewrites known snake_case keys to camelCase.
// Keys already in camelCase, or with no mapping, are passed through unchanged.
func normalizeInboundPatchKeys(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if alias, ok := inboundKeyAliases[k]; ok {
			out[alias] = v
			continue
		}
		out[k] = v
	}
	return out
}

// inboundBaseFromResponse turns a GetInbound response obj into a map of just
// the writable fields. Round-tripping through xui.Inbound strips any runtime
// fields (e.g. clientStats) that must not be echoed back into the update body.
func inboundBaseFromResponse(obj json.RawMessage) (map[string]any, error) {
	if len(obj) == 0 || string(obj) == "null" {
		return nil, fmt.Errorf("empty inbound response")
	}
	var in xui.Inbound
	if err := json.Unmarshal(obj, &in); err != nil {
		return nil, fmt.Errorf("parsing current inbound: %w", err)
	}
	b, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("remarshal inbound: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("inbound to map: %w", err)
	}
	return m, nil
}

// mergeInboundPatch normalises the patch keys, overlays them on the current
// inbound state, and validates the result. Returns the body to POST.
func mergeInboundPatch(currentObj json.RawMessage, patch map[string]any) (map[string]any, error) {
	base, err := inboundBaseFromResponse(currentObj)
	if err != nil {
		return nil, err
	}
	for k, v := range normalizeInboundPatchKeys(patch) {
		base[k] = v
	}
	if err := validateMergedInbound(base); err != nil {
		return nil, fmt.Errorf("refusing to update — merge would leave inbound unusable: %w", err)
	}
	return base, nil
}

// validateMergedInbound guards against writing a body that would disable the
// inbound (the failure mode reported in BUG-update_inbound.md).
func validateMergedInbound(m map[string]any) error {
	if p, _ := m["protocol"].(string); p == "" {
		return fmt.Errorf("protocol is empty")
	}
	port, ok := numberAsInt(m["port"])
	if !ok {
		return fmt.Errorf("port is missing or not a number")
	}
	if port == 0 {
		return fmt.Errorf("port is 0")
	}
	return nil
}

// numberAsInt accepts the several numeric types that may appear in a
// map[string]any after JSON round-tripping.
func numberAsInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func (h *inboundHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.DeleteInbound(ctx, int(id)))
}

func (h *inboundHandler) setEnable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	enable, err := req.RequireBool("enable")
	if err != nil {
		return mcp.NewToolResultError("enable is required"), nil
	}
	return toResult(h.client.SetInboundEnable(ctx, int(id), enable))
}

func (h *inboundHandler) resetTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.ResetInboundTraffic(ctx, int(id)))
}

func (h *inboundHandler) deleteAllClients(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.DeleteAllInboundClients(ctx, int(id)))
}

func (h *inboundHandler) bulkDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idsStr, err := req.RequireString("ids")
	if err != nil {
		return mcp.NewToolResultError("ids is required"), nil
	}
	ids, err := parseInboundIDs(idsStr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.BulkDeleteInbounds(ctx, ids))
}

// parseInboundIDs reads the JSON array of inbound IDs accepted by
// bulk_delete_inbounds. An empty list is rejected rather than sent, since the
// panel would happily answer "0 deleted" and hide the mistake.
func parseInboundIDs(raw string) ([]int, error) {
	var ids []int
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("ids must be a JSON array of numbers, e.g. [3,7,12]: %w", err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("ids is empty — nothing to delete")
	}
	return ids, nil
}

func (h *inboundHandler) allLinks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.GetAllInboundLinks(ctx)
	if req.GetBool("full", false) {
		return toResult(resp, err)
	}
	return linkedResult(resp, err, mcp.NewResourceLink(
		inboundLinksURI,
		"Connection links",
		"Every client's connection URL. Read it when the links themselves are needed, e.g. to hand one to a user.",
		"application/json",
	), summarizeLinks)
}

func (h *inboundHandler) listSlim(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListInboundsSlim(ctx))
}

func (h *inboundHandler) options(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.InboundOptions(ctx))
}

func (h *inboundHandler) getFallbacks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.GetInboundFallbacks(ctx, int(id)))
}

func (h *inboundHandler) setFallbacks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	raw, err := req.RequireString("fallbacks")
	if err != nil {
		return mcp.NewToolResultError("fallbacks is required"), nil
	}
	var rules []map[string]any
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("fallbacks must be a JSON array: %v", err)), nil
	}
	return toResult(h.client.SetInboundFallbacks(ctx, int(id), rules))
}

func (h *inboundHandler) setSubSortIndex(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	index, err := req.RequireFloat("index")
	if err != nil {
		return mcp.NewToolResultError("index is required"), nil
	}
	return toResult(h.client.SetInboundSubSortIndex(ctx, int(id), int(index)))
}
