package handler

import (
	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/server"
)

// RegisterAll registers all MCP tools on the server.
func RegisterAll(s *server.MCPServer, client *xui.Client) {
	registerInboundTools(s, client)
	registerClientTools(s, client)
	registerServerTools(s, client)
	registerXrayTools(s, client)
}
