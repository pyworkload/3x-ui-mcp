package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
)

// toResult converts an XUI API response into an MCP tool result.
// API errors (success=false) are returned as tool errors, not Go errors,
// so the LLM sees the message instead of a transport failure.
func toResult(resp *xui.Response, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !resp.Success {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %s", resp.Msg)), nil
	}
	return mcp.NewToolResultText(formatResponse(resp)), nil
}

// formatResponse pretty-prints an API response for the LLM.
func formatResponse(resp *xui.Response) string {
	if resp.Obj == nil || string(resp.Obj) == "null" {
		if resp.Msg != "" {
			return resp.Msg
		}
		return `{"success": true}`
	}

	// Try to pretty-print the obj field
	var obj any
	if err := json.Unmarshal(resp.Obj, &obj); err == nil {
		pretty, err := json.MarshalIndent(obj, "", "  ")
		if err == nil {
			return string(pretty)
		}
	}

	return string(resp.Obj)
}

// generateUUID generates a random UUID v4.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// buildClientSettings constructs the "settings" JSON string for adding/updating a client.
func buildClientSettings(client xui.ClientConfig) (string, error) {
	settings := xui.InboundSettings{
		Clients: []xui.ClientConfig{client},
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return "", fmt.Errorf("marshaling client settings: %w", err)
	}
	return string(data), nil
}
