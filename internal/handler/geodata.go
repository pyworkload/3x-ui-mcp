package handler

import (
	"context"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// geodataHandler holds the XUI client for geodata tool handlers.
type geodataHandler struct {
	client *xui.Client
}

func registerGeodataTools(s *server.MCPServer, client *xui.Client) {
	h := &geodataHandler{client: client}

	s.AddTool(mcp.NewTool("list_geodata_files",
		readsPanel,
		mcp.WithDescription("List the geo databases (.dat files) in Xray's asset folder, with their detected layout, size, modification time and category count. Start here before writing a routing rule against geosite/geoip categories."),
	), h.files)

	s.AddTool(mcp.NewTool("list_geodata_categories",
		readsPanel,
		mcp.WithDescription("List the categories inside one geo database, each with its entry count and the attributes its domains carry (e.g. 'ads', 'cn'). These are the codes a routing rule references as geosite:<code> or geoip:<code>."),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Database file name from list_geodata_files, e.g. 'geosite.dat'"),
		),
		mcp.WithString("query",
			mcp.Description("Case-insensitive substring filter on the category code"),
		),
		mcp.WithNumber("offset",
			mcp.Description("Rows to skip"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("limit",
			mcp.Description("Rows to return, capped at 500. Omit to return every category — the index is small."),
			mcp.DefaultNumber(0),
		),
	), h.categories)

	s.AddTool(mcp.NewTool("list_geodata_entries",
		readsPanel,
		mcp.WithDescription("List the rules inside one category: domain rules typed as domain/full/keyword/regexp for geosite databases, CIDRs for geoip ones. Use it to check what a category actually matches before routing traffic by it."),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Database file name, e.g. 'geosite.dat'"),
		),
		mcp.WithString("code",
			mcp.Required(),
			mcp.Description("Category code, case-insensitive, e.g. 'google'"),
		),
		mcp.WithString("query",
			mcp.Description("Case-insensitive substring filter on the rule value"),
		),
		mcp.WithNumber("offset",
			mcp.Description("Rows to skip"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("limit",
			mcp.Description("Rows to return, capped at 500 (the default)"),
			mcp.DefaultNumber(0),
		),
	), h.entries)

	s.AddTool(mcp.NewTool("validate_geodata_tokens",
		readsPanel,
		mcp.WithDescription("Check routing tokens against the databases on disk and report only the ones that do not resolve, each with a reason. Plain domains and CIDRs are ignored. Worth running before saving a routing rule, since Xray fails at load time on an unknown category."),
		mcp.WithString("tokens",
			mcp.Required(),
			mcp.Description("Comma-separated tokens, e.g. 'geosite:google,geosite:cn@ads' or 'geoip:cn,10.0.0.0/8'"),
		),
		mcp.WithString("kind",
			mcp.Description("Token grammar: 'domain' for geosite/ext tokens, 'ip' for geoip ones"),
			mcp.Enum("domain", "ip"),
			mcp.DefaultString("domain"),
		),
	), h.validate)
}

func (h *geodataHandler) files(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GeodataFiles(ctx))
}

func (h *geodataHandler) categories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file, err := req.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError("file is required"), nil
	}
	return toResult(h.client.GeodataCategories(ctx,
		file,
		req.GetString("query", ""),
		int(req.GetFloat("offset", 0)),
		int(req.GetFloat("limit", 0)),
	))
}

func (h *geodataHandler) entries(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file, err := req.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError("file is required"), nil
	}
	code, err := req.RequireString("code")
	if err != nil {
		return mcp.NewToolResultError("code is required"), nil
	}
	return toResult(h.client.GeodataEntries(ctx,
		file,
		code,
		req.GetString("query", ""),
		int(req.GetFloat("offset", 0)),
		int(req.GetFloat("limit", 0)),
	))
}

func (h *geodataHandler) validate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tokens, err := req.RequireString("tokens")
	if err != nil {
		return mcp.NewToolResultError("tokens is required"), nil
	}
	return toResult(h.client.ValidateGeodataTokens(ctx, req.GetString("kind", "domain"), tokens))
}
