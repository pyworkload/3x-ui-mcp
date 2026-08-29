package handler

import (
	"context"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// groupHandler holds the XUI client for client-group tool handlers.
type groupHandler struct {
	client *xui.Client
}

func registerGroupTools(s *server.MCPServer, client *xui.Client) {
	h := &groupHandler{client: client}

	s.AddTool(mcp.NewTool("list_client_groups",
		readsPanel,
		mcp.WithDescription("List every client group with its member count. Includes empty groups, which exist as placeholders before anyone is added to them."),
	), h.list)

	s.AddTool(mcp.NewTool("get_client_group_emails",
		readsPanel,
		mcp.WithDescription("List the emails of the clients in one group — the input for fanning a bulk operation across the whole group."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Group name"),
		),
	), h.emails)

	s.AddTool(mcp.NewTool("create_client_group",
		writesPanel,
		mcp.WithDescription("Create an empty client group. It becomes selectable in client forms before it has any members. Fails if the name is taken."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Group name to create"),
		),
	), h.create)

	s.AddTool(mcp.NewTool("rename_client_group",
		updatesPanel,
		mcp.WithDescription("Rename a client group. The new name is propagated to every member, in both the client records and the owning inbounds' settings."),
		mcp.WithString("old_name",
			mcp.Required(),
			mcp.Description("Current group name"),
		),
		mcp.WithString("new_name",
			mcp.Required(),
			mcp.Description("New group name"),
		),
	), h.rename)

	s.AddTool(mcp.NewTool("delete_client_group",
		destroysPanel,
		mcp.WithDescription("Delete a client group and clear the label from its members. The clients themselves are kept — use bulk_delete_clients to remove them."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Group name to delete"),
		),
	), h.delete)

	s.AddTool(mcp.NewTool("add_clients_to_group",
		updatesPanel,
		mcp.WithDescription("Label many clients with one group in a single call. A client belongs to at most one group, so this replaces any group it was in."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to add"),
			mcp.WithStringItems(),
		),
		mcp.WithString("group",
			mcp.Required(),
			mcp.Description("Group to put them in"),
		),
	), h.add)

	s.AddTool(mcp.NewTool("remove_clients_from_group",
		updatesPanel,
		mcp.WithDescription("Clear the group label on many clients. The clients and their group's other members are untouched."),
		mcp.WithArray("emails",
			mcp.Required(),
			mcp.Description("Emails of the clients to unlabel"),
			mcp.WithStringItems(),
		),
	), h.remove)

	s.AddTool(mcp.NewTool("reset_client_group_traffic",
		destroysPanel,
		mcp.WithDescription("Zero the group-level traffic counter by snapshotting its members' current totals as a baseline. Per-client counters keep running — use reset_client_traffic to zero those."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Group name"),
		),
	), h.resetTraffic)
}

func (h *groupHandler) list(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListClientGroups(ctx))
}

func (h *groupHandler) emails(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}
	return toResult(h.client.ClientGroupEmails(ctx, name))
}

func (h *groupHandler) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}
	return toResult(h.client.CreateClientGroup(ctx, name))
}

func (h *groupHandler) rename(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	oldName, err := req.RequireString("old_name")
	if err != nil {
		return mcp.NewToolResultError("old_name is required"), nil
	}
	newName, err := req.RequireString("new_name")
	if err != nil {
		return mcp.NewToolResultError("new_name is required"), nil
	}
	return toResult(h.client.RenameClientGroup(ctx, oldName, newName))
}

func (h *groupHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}
	return toResult(h.client.DeleteClientGroup(ctx, name))
}

func (h *groupHandler) add(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	group, err := req.RequireString("group")
	if err != nil {
		return mcp.NewToolResultError("group is required"), nil
	}
	return toResult(h.client.AddClientsToGroup(ctx, emails, group))
}

func (h *groupHandler) remove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	emails := req.GetStringSlice("emails", nil)
	if len(emails) == 0 {
		return mcp.NewToolResultError("emails is required (at least one email)"), nil
	}
	return toResult(h.client.RemoveClientsFromGroup(ctx, emails))
}

func (h *groupHandler) resetTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}
	return toResult(h.client.ResetClientGroupTraffic(ctx, name))
}
