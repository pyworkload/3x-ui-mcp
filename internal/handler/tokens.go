package handler

import (
	"context"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// tokenHandler holds the XUI client for API token tool handlers.
type tokenHandler struct {
	client *xui.Client
}

func registerAPITokenTools(s *server.MCPServer, client *xui.Client) {
	h := &tokenHandler{client: client}

	s.AddTool(mcp.NewTool("list_api_tokens",
		readsPanel,
		mcp.WithDescription("List the panel's API tokens with their scope, expiry and enabled state. The token values themselves are never returned — the panel stores only hashes."),
	), h.list)

	s.AddTool(mcp.NewTool("create_api_token",
		writesPanel,
		mcp.WithDescription("Mint a scoped API token. The plaintext is in the response and nowhere else — the panel keeps a SHA-256 hash — so it has to be captured now or the token is unusable. Scope 'admin' grants the whole API (what this MCP server needs), 'monitor' only the status and metrics routes, 'node-sync' only the panel-to-node allowlist."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Label for the token, e.g. 'mcp-server'"),
		),
		mcp.WithString("scope",
			mcp.Description("What the token may reach"),
			mcp.Enum("admin", "monitor", "node-sync"),
			mcp.DefaultString("admin"),
		),
		mcp.WithNumber("expires_at",
			mcp.Description("Expiry as Unix milliseconds; 0 never expires"),
			mcp.DefaultNumber(0),
		),
	), h.create)

	s.AddTool(mcp.NewTool("set_api_token_enabled",
		updatesPanel,
		mcp.WithDescription("Enable or disable a token without deleting it. A disabled token is rejected on its next request, and can be re-enabled later."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Token ID from list_api_tokens"),
		),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("true to enable, false to disable"),
		),
		mcp.WithString("scope",
			mcp.Required(),
			mcp.Description("The token's stored scope. The panel fails closed unless this matches, so a token cannot be flipped by guessing its ID — read it from list_api_tokens."),
		),
	), h.setEnabled)

	s.AddTool(mcp.NewTool("delete_api_token",
		destroysPanel,
		mcp.WithDescription("Permanently delete an API token. Anything authenticating with it stops working immediately and the token cannot be recovered — set_api_token_enabled is the reversible option."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Token ID from list_api_tokens"),
		),
		mcp.WithString("scope",
			mcp.Required(),
			mcp.Description("The token's stored scope; the panel refuses the delete unless it matches"),
		),
	), h.delete)
}

func (h *tokenHandler) list(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListAPITokens(ctx))
}

func (h *tokenHandler) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}
	return toResult(h.client.CreateAPIToken(ctx,
		name,
		req.GetString("scope", "admin"),
		int64(req.GetFloat("expires_at", 0)),
	))
}

func (h *tokenHandler) setEnabled(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	enabled, err := req.RequireBool("enabled")
	if err != nil {
		return mcp.NewToolResultError("enabled is required"), nil
	}
	scope, err := req.RequireString("scope")
	if err != nil {
		return mcp.NewToolResultError("scope is required: the panel checks it against the stored scope"), nil
	}
	return toResult(h.client.SetAPITokenEnabled(ctx, int(id), scope, enabled))
}

func (h *tokenHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	scope, err := req.RequireString("scope")
	if err != nil {
		return mcp.NewToolResultError("scope is required: the panel checks it against the stored scope"), nil
	}
	return toResult(h.client.DeleteAPIToken(ctx, int(id), scope))
}
