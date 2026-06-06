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

// bytesPerGB converts the GB units used by MCP tool params to the bytes the panel stores.
const bytesPerGB = 1073741824

// clientRecordView parses the "client" object returned by GET clients/get/:email.
// The panel returns a ClientRecord there, whose UUID lives under "uuid" (not "id"),
// so we map it explicitly back onto a ClientConfig for round-tripping into updates.
type clientRecordView struct {
	UUID       string `json:"uuid"`
	Security   string `json:"security"`
	Password   string `json:"password"`
	Auth       string `json:"auth"`
	Flow       string `json:"flow"`
	Email      string `json:"email"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	TgID       int64  `json:"tgId"`
	SubID      string `json:"subId"`
	Group      string `json:"group"`
	Comment    string `json:"comment"`
	Reset      int    `json:"reset"`
}

func (v clientRecordView) toConfig() xui.ClientConfig {
	return xui.ClientConfig{
		ID:         v.UUID,
		Security:   v.Security,
		Password:   v.Password,
		Auth:       v.Auth,
		Flow:       v.Flow,
		Email:      v.Email,
		LimitIP:    v.LimitIP,
		TotalGB:    v.TotalGB,
		ExpiryTime: v.ExpiryTime,
		Enable:     v.Enable,
		TgID:       v.TgID,
		SubID:      v.SubID,
		Group:      v.Group,
		Comment:    v.Comment,
		Reset:      v.Reset,
	}
}

// parseClient extracts the current client config from a GET clients/get/:email response.
func parseClient(resp *xui.Response) (xui.ClientConfig, error) {
	var wrap struct {
		Client clientRecordView `json:"client"`
	}
	if err := json.Unmarshal(resp.Obj, &wrap); err != nil {
		return xui.ClientConfig{}, fmt.Errorf("parsing client record: %w", err)
	}
	return wrap.Client.toConfig(), nil
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
