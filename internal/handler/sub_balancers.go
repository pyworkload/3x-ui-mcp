package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// subBalancerHandler holds the XUI client for subscription balancer handlers.
type subBalancerHandler struct {
	client *xui.Client
}

// subBalancerRow is the panel's stored balancer, read back for updates.
type subBalancerRow struct {
	ID         int    `json:"id"`
	Remark     string `json:"remark"`
	Strategy   string `json:"strategy"`
	InboundIds []int  `json:"inboundIds"`
	SortOrder  int    `json:"sortOrder"`
	Enabled    bool   `json:"enabled"`
}

func registerSubBalancerTools(s *server.MCPServer, client *xui.Client) {
	h := &subBalancerHandler{client: client}

	s.AddTool(mcp.NewTool("list_sub_balancers",
		readsPanel,
		mcp.WithDescription("List the subscription balancers. These are client-side: each one appears in the JSON subscription of every client sitting on at least one of its inbounds, and the client app picks a server by the strategy. Unrelated to get_balancers, which reports the routing balancers running inside Xray."),
	), h.list)

	s.AddTool(mcp.NewTool("create_sub_balancer",
		writesPanel,
		mcp.WithDescription("Create a subscription balancer over a set of inbounds."),
		mcp.WithString("remark",
			mcp.Required(),
			mcp.Description("Display name, shown to clients"),
		),
		mcp.WithArray("inbound_ids",
			mcp.Required(),
			mcp.Description("Inbounds this balancer spreads across"),
			mcp.WithNumberItems(),
		),
		mcp.WithString("strategy",
			mcp.Description("How the client app picks a server"),
			mcp.Enum("random", "roundRobin", "leastPing", "leastLoad"),
			mcp.DefaultString("random"),
		),
		mcp.WithNumber("sort_order",
			mcp.Description("Position in the balancer list; 1 sorts first"),
			mcp.DefaultNumber(1),
		),
		mcp.WithBoolean("enabled",
			mcp.Description("Whether the balancer is served to clients"),
			mcp.DefaultBool(true),
		),
	), h.create)

	s.AddTool(mcp.NewTool("update_sub_balancer",
		updatesPanel,
		mcp.WithDescription("Update a subscription balancer. The panel replaces the whole row, so this reads the stored balancer first and overlays only the fields you pass."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Balancer ID from list_sub_balancers"),
		),
		mcp.WithString("remark",
			mcp.Description("Display name"),
		),
		mcp.WithArray("inbound_ids",
			mcp.Description("Inbounds this balancer spreads across"),
			mcp.WithNumberItems(),
		),
		mcp.WithString("strategy",
			mcp.Description("How the client app picks a server"),
			mcp.Enum("random", "roundRobin", "leastPing", "leastLoad"),
		),
		mcp.WithNumber("sort_order",
			mcp.Description("Position in the balancer list"),
		),
		mcp.WithBoolean("enabled",
			mcp.Description("Whether the balancer is served to clients"),
		),
	), h.update)

	s.AddTool(mcp.NewTool("delete_sub_balancer",
		destroysPanel,
		mcp.WithDescription("Delete a subscription balancer. Clients stop seeing it on their next subscription fetch; the inbounds behind it are untouched."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Balancer ID to delete"),
		),
	), h.delete)
}

func (h *subBalancerHandler) list(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListSubBalancers(ctx))
}

func (h *subBalancerHandler) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	remark, err := req.RequireString("remark")
	if err != nil {
		return mcp.NewToolResultError("remark is required"), nil
	}
	inboundIDs := req.GetIntSlice("inbound_ids", nil)
	if len(inboundIDs) == 0 {
		return mcp.NewToolResultError("inbound_ids is required (at least one inbound)"), nil
	}
	enabled := req.GetBool("enabled", true)
	return toResult(h.client.CreateSubBalancer(ctx,
		remark,
		req.GetString("strategy", "random"),
		inboundIDs,
		int(req.GetFloat("sort_order", 1)),
		&enabled,
	))
}

func (h *subBalancerHandler) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	cur, ferr := h.findSubBalancer(ctx, int(id))
	if ferr != nil {
		return mcp.NewToolResultError(ferr.Error()), nil
	}

	args := req.GetArguments()
	inboundIDs := cur.InboundIds
	if _, ok := args["inbound_ids"]; ok {
		inboundIDs = req.GetIntSlice("inbound_ids", nil)
	}
	enabled := cur.Enabled
	if _, ok := args["enabled"]; ok {
		enabled = req.GetBool("enabled", true)
	}

	return toResult(h.client.UpdateSubBalancer(ctx,
		cur.ID,
		req.GetString("remark", cur.Remark),
		req.GetString("strategy", cur.Strategy),
		inboundIDs,
		int(req.GetFloat("sort_order", float64(cur.SortOrder))),
		&enabled,
	))
}

func (h *subBalancerHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.DeleteSubBalancer(ctx, int(id)))
}

// findSubBalancer reads the stored row an update is based on. The panel has no
// single-balancer route, so the list is filtered here.
func (h *subBalancerHandler) findSubBalancer(ctx context.Context, id int) (*subBalancerRow, error) {
	resp, err := h.client.ListSubBalancers(ctx)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}
	var rows []subBalancerRow
	if err := json.Unmarshal(resp.Obj, &rows); err != nil {
		return nil, fmt.Errorf("parsing the balancer list: %w", err)
	}
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i], nil
		}
	}
	return nil, fmt.Errorf("no subscription balancer with id %d", id)
}
