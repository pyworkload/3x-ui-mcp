package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
)

// Tool annotation presets.
//
// The hints tell an MCP client what a call does to the panel before it runs:
// clients auto-approve read-only tools and warn on destructive ones. Every
// preset also sets openWorldHint, which here separates tools that stay inside
// the panel from those that reach hosts on the internet.
var (
	// readsPanel returns panel state and changes nothing.
	readsPanel = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:  mcp.ToBoolPtr(true),
		OpenWorldHint: mcp.ToBoolPtr(false),
	})

	// probesRemote reads only, but reaches a host outside the panel.
	probesRemote = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:  mcp.ToBoolPtr(true),
		OpenWorldHint: mcp.ToBoolPtr(true),
	})

	// writesPanel adds to the panel; running it twice adds twice.
	writesPanel = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	})

	// updatesPanel overwrites fields in place, keeping the rest; repeating is a no-op.
	updatesPanel = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	})

	// destroysPanel deletes data or zeroes counters; repeating changes nothing further.
	destroysPanel = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	})

	// interruptsService stops or restarts a running service, dropping live connections.
	interruptsService = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	})

	// fetchesRemote pulls data from an external URL into the panel.
	fetchesRemote = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	})

	// installsRemote replaces a running binary with one downloaded from the internet.
	installsRemote = mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	})
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

// linkedResult answers with a short summary plus a resource_link instead of the
// whole document. A running Xray config runs to ~9k characters, which every
// caller pays for even when it only wanted to know how many outbounds exist;
// the link lets whoever needs the body fetch it by URI. Tools that take
// full=true bypass this and return everything.
func linkedResult(resp *xui.Response, err error, link mcp.ResourceLink, summarize func(json.RawMessage) string) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !resp.Success {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %s", resp.Msg)), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(summarize(resp.Obj)),
			link,
		},
	}, nil
}

// summarizeXrayConfig reports the shape of a config or template: the counts a
// caller usually wants before deciding whether to read the whole thing.
func summarizeXrayConfig(raw json.RawMessage) string {
	var cfg struct {
		Inbounds  []json.RawMessage `json:"inbounds"`
		Outbounds []json.RawMessage `json:"outbounds"`
		Routing   struct {
			Rules     []json.RawMessage `json:"rules"`
			Balancers []struct {
				Tag string `json:"tag"`
			} `json:"balancers"`
		} `json:"routing"`
		DNS struct {
			Servers []json.RawMessage `json:"servers"`
		} `json:"dns"`
		Observatory *json.RawMessage `json:"observatory"`
		Metrics     *json.RawMessage `json:"metrics"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "Could not summarize the config; read the linked resource for the full JSON."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d inbounds, %d outbounds, %d routing rules, %d dns servers",
		len(cfg.Inbounds), len(cfg.Outbounds), len(cfg.Routing.Rules), len(cfg.DNS.Servers))
	if n := len(cfg.Routing.Balancers); n > 0 {
		tags := make([]string, 0, n)
		for _, bal := range cfg.Routing.Balancers {
			tags = append(tags, bal.Tag)
		}
		fmt.Fprintf(&b, "\nBalancers: %s", strings.Join(tags, ", "))
	}
	if cfg.Observatory != nil {
		b.WriteString("\nObservatory: configured")
	}
	if cfg.Metrics != nil {
		b.WriteString("\nMetrics: configured")
	}
	b.WriteString("\nThe full JSON is linked below — read that resource to inspect or edit it.")
	return b.String()
}

// summarizeLinks counts connection URLs by scheme. The URLs themselves carry
// client credentials, so they belong behind the link rather than in every
// answer that merely asked how many there are.
func summarizeLinks(raw json.RawMessage) string {
	var links []string
	if err := json.Unmarshal(raw, &links); err != nil {
		return "Could not summarize the links; read the linked resource for the full list."
	}
	if len(links) == 0 {
		return "No connection links: the panel has no clients to render."
	}

	counts := map[string]int{}
	for _, link := range links {
		scheme, _, found := strings.Cut(link, "://")
		if !found {
			scheme = "other"
		}
		counts[scheme]++
	}
	schemes := make([]string, 0, len(counts))
	for scheme := range counts {
		schemes = append(schemes, scheme)
	}
	sort.Strings(schemes)

	parts := make([]string, 0, len(schemes))
	for _, scheme := range schemes {
		parts = append(parts, fmt.Sprintf("%s: %d", scheme, counts[scheme]))
	}
	return fmt.Sprintf("%d connection links (%s).\nThe URLs carry client credentials and are linked below.",
		len(links), strings.Join(parts, ", "))
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
