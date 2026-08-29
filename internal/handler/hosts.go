package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// hostHandler holds the XUI client for host group tool handlers.
type hostHandler struct {
	client *xui.Client
}

// Shared parameter set for add_host_group and update_host_group. The panel's
// HostGroup entity carries ~30 fields; these are the ones a group is normally
// built from, and raw_json is the escape hatch for the rest (mux, sockopt, ECH,
// final mask, vless route).
func hostGroupParams() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithArray("inbound_ids",
			mcp.Description("Inbound IDs this group publishes"),
			mcp.WithNumberItems(),
		),
		mcp.WithString("remark",
			mcp.Description("Display name for the group"),
		),
		mcp.WithArray("hosts",
			mcp.Description(`External addresses, e.g. ["cdn.example.com","cdn2.example.com:443"]`),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("port",
			mcp.Description("Port to advertise; 0 keeps the inbound's own port"),
		),
		mcp.WithString("security",
			mcp.Description("TLS mode to advertise; 'same' inherits the inbound's"),
			mcp.Enum("same", "tls", "none", "reality"),
		),
		mcp.WithString("sni",
			mcp.Description("SNI to advertise"),
		),
		mcp.WithString("host_header",
			mcp.Description("HTTP Host header"),
		),
		mcp.WithString("path",
			mcp.Description("Transport path, e.g. /ws"),
		),
		mcp.WithArray("alpn",
			mcp.Description("ALPN values to advertise"),
			mcp.WithStringItems(),
		),
		mcp.WithString("fingerprint",
			mcp.Description("uTLS fingerprint, e.g. chrome"),
		),
		mcp.WithArray("tags",
			mcp.Description("Free-form tags used to filter host groups"),
			mcp.WithStringItems(),
		),
		mcp.WithString("server_description",
			mcp.Description("Note shown next to the group (max 64 chars)"),
		),
		mcp.WithBoolean("is_disabled",
			mcp.Description("Disable the group without deleting it"),
		),
		mcp.WithBoolean("is_hidden",
			mcp.Description("Keep the group out of subscription output"),
		),
		mcp.WithBoolean("allow_insecure",
			mcp.Description("Let clients skip certificate verification"),
		),
		mcp.WithString("raw_json",
			mcp.Description("Complete HostGroup JSON. When given it is sent as-is and every other parameter is ignored — use it for fields this tool does not expose."),
		),
	}
}

func registerHostTools(s *server.MCPServer, client *xui.Client) {
	h := &hostHandler{client: client}

	s.AddTool(mcp.NewTool("list_hosts",
		readsPanel,
		mcp.WithDescription("List every host group across all inbounds. A host group publishes one or more inbounds under external addresses — a CDN hostname, a second port, a different SNI — and the panel renders subscription links against them."),
	), h.list)

	s.AddTool(mcp.NewTool("get_host_group",
		readsPanel,
		mcp.WithDescription("Get one host group by its group ID (a string, not a number)."),
		mcp.WithString("group_id",
			mcp.Required(),
			mcp.Description("Host group ID from list_hosts"),
		),
	), h.get)

	s.AddTool(mcp.NewTool("get_inbound_hosts",
		readsPanel,
		mcp.WithDescription("List the host groups attached to one inbound — what addresses that inbound is currently published under."),
		mcp.WithNumber("inbound_id",
			mcp.Required(),
			mcp.Description("Inbound ID"),
		),
	), h.byInbound)

	s.AddTool(mcp.NewTool("list_host_tags",
		readsPanel,
		mcp.WithDescription("List the distinct tags used across all host groups."),
	), h.tags)

	s.AddTool(mcp.NewTool("add_host_group",
		append([]mcp.ToolOption{
			writesPanel,
			mcp.WithDescription("Create a host group. inbound_ids and remark are required unless raw_json carries them."),
		}, hostGroupParams()...)...,
	), h.add)

	s.AddTool(mcp.NewTool("update_host_group",
		append([]mcp.ToolOption{
			updatesPanel,
			mcp.WithDescription("Update a host group. The panel replaces the whole group, so this reads the stored one first and overlays only the fields you pass — same contract as update_inbound and update_client."),
			mcp.WithString("group_id",
				mcp.Required(),
				mcp.Description("Host group ID to update"),
			),
		}, hostGroupParams()...)...,
	), h.update)

	s.AddTool(mcp.NewTool("delete_host_group",
		destroysPanel,
		mcp.WithDescription("Delete a host group and every host inside it. The inbounds it published are untouched."),
		mcp.WithString("group_id",
			mcp.Required(),
			mcp.Description("Host group ID to delete"),
		),
	), h.delete)

	s.AddTool(mcp.NewTool("set_host_group_enable",
		updatesPanel,
		mcp.WithDescription("Enable or disable one host group. A disabled group stays saved but drops out of subscription output."),
		mcp.WithString("group_id",
			mcp.Required(),
			mcp.Description("Host group ID"),
		),
		mcp.WithBoolean("enable",
			mcp.Required(),
			mcp.Description("true to enable, false to disable"),
		),
	), h.setEnable)

	s.AddTool(mcp.NewTool("reorder_host_groups",
		updatesPanel,
		mcp.WithDescription("Set the sort order of host groups by listing their IDs in the order wanted. Order decides which address a subscription lists first."),
		mcp.WithArray("group_ids",
			mcp.Required(),
			mcp.Description("Host group IDs in the desired order"),
			mcp.WithStringItems(),
		),
	), h.reorder)

	s.AddTool(mcp.NewTool("bulk_delete_host_groups",
		destroysPanel,
		mcp.WithDescription("Delete several host groups in one call."),
		mcp.WithArray("group_ids",
			mcp.Required(),
			mcp.Description("Host group IDs to delete"),
			mcp.WithStringItems(),
		),
	), h.bulkDelete)

	s.AddTool(mcp.NewTool("bulk_set_host_groups_enable",
		updatesPanel,
		mcp.WithDescription("Enable or disable several host groups at once."),
		mcp.WithArray("group_ids",
			mcp.Required(),
			mcp.Description("Host group IDs to flip"),
			mcp.WithStringItems(),
		),
		mcp.WithBoolean("enable",
			mcp.Required(),
			mcp.Description("true to enable, false to disable"),
		),
	), h.bulkSetEnable)
}

func (h *hostHandler) list(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListHosts(ctx))
}

func (h *hostHandler) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupID, err := req.RequireString("group_id")
	if err != nil {
		return mcp.NewToolResultError("group_id is required"), nil
	}
	return toResult(h.client.GetHostGroup(ctx, groupID))
}

func (h *hostHandler) byInbound(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("inbound_id")
	if err != nil {
		return mcp.NewToolResultError("inbound_id is required"), nil
	}
	return toResult(h.client.InboundHosts(ctx, int(id)))
}

func (h *hostHandler) tags(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.HostTags(ctx))
}

func (h *hostHandler) add(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	group, err := hostGroupBody(req, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if _, ok := group["inboundIds"]; !ok {
		return mcp.NewToolResultError("inbound_ids is required: a host group must publish at least one inbound"), nil
	}
	if remark, _ := group["remark"].(string); remark == "" {
		return mcp.NewToolResultError("remark is required"), nil
	}
	return toResult(h.client.AddHostGroup(ctx, group))
}

func (h *hostHandler) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupID, err := req.RequireString("group_id")
	if err != nil {
		return mcp.NewToolResultError("group_id is required"), nil
	}

	cur, err := h.client.GetHostGroup(ctx, groupID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !cur.Success {
		return mcp.NewToolResultError("API error: " + cur.Msg), nil
	}
	stored := map[string]any{}
	if err := json.Unmarshal(cur.Obj, &stored); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parsing the stored host group: %v", err)), nil
	}

	group, err := hostGroupBody(req, stored)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.UpdateHostGroup(ctx, groupID, group))
}

func (h *hostHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupID, err := req.RequireString("group_id")
	if err != nil {
		return mcp.NewToolResultError("group_id is required"), nil
	}
	return toResult(h.client.DeleteHostGroup(ctx, groupID))
}

func (h *hostHandler) setEnable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupID, err := req.RequireString("group_id")
	if err != nil {
		return mcp.NewToolResultError("group_id is required"), nil
	}
	enable, err := req.RequireBool("enable")
	if err != nil {
		return mcp.NewToolResultError("enable is required"), nil
	}
	return toResult(h.client.SetHostGroupEnable(ctx, groupID, enable))
}

func (h *hostHandler) reorder(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ids := req.GetStringSlice("group_ids", nil)
	if len(ids) == 0 {
		return mcp.NewToolResultError("group_ids is required (at least one group ID)"), nil
	}
	return toResult(h.client.ReorderHostGroups(ctx, ids))
}

func (h *hostHandler) bulkDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ids := req.GetStringSlice("group_ids", nil)
	if len(ids) == 0 {
		return mcp.NewToolResultError("group_ids is required (at least one group ID)"), nil
	}
	return toResult(h.client.BulkDeleteHostGroups(ctx, ids))
}

func (h *hostHandler) bulkSetEnable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ids := req.GetStringSlice("group_ids", nil)
	if len(ids) == 0 {
		return mcp.NewToolResultError("group_ids is required (at least one group ID)"), nil
	}
	enable, err := req.RequireBool("enable")
	if err != nil {
		return mcp.NewToolResultError("enable is required"), nil
	}
	return toResult(h.client.BulkSetHostGroupsEnable(ctx, ids, enable))
}

// hostGroupBody overlays the supplied parameters onto base, which is empty for
// a create and the stored group for an update. raw_json replaces the lot.
func hostGroupBody(req mcp.CallToolRequest, base map[string]any) (map[string]any, error) {
	if raw := req.GetString("raw_json", ""); raw != "" {
		group := map[string]any{}
		if err := json.Unmarshal([]byte(raw), &group); err != nil {
			return nil, fmt.Errorf("raw_json must be a JSON object: %w", err)
		}
		return group, nil
	}

	args := req.GetArguments()
	supplied := func(param string) bool {
		_, ok := args[param]
		return ok
	}

	for param, key := range map[string]string{
		"remark":             "remark",
		"security":           "security",
		"sni":                "sni",
		"host_header":        "hostHeader",
		"path":               "path",
		"fingerprint":        "fingerprint",
		"server_description": "serverDescription",
	} {
		if supplied(param) {
			base[key] = req.GetString(param, "")
		}
	}
	for param, key := range map[string]string{
		"is_disabled":    "isDisabled",
		"is_hidden":      "isHidden",
		"allow_insecure": "allowInsecure",
	} {
		if supplied(param) {
			base[key] = req.GetBool(param, false)
		}
	}
	for param, key := range map[string]string{
		"hosts": "hosts",
		"alpn":  "alpn",
		"tags":  "tags",
	} {
		if supplied(param) {
			base[key] = req.GetStringSlice(param, nil)
		}
	}
	if supplied("port") {
		base["port"] = int(req.GetFloat("port", 0))
	}
	if supplied("inbound_ids") {
		base["inboundIds"] = req.GetIntSlice("inbound_ids", nil)
	}
	return base, nil
}
