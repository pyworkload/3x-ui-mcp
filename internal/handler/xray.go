package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// xrayHandler holds the XUI client for Xray config & routing tools.
type xrayHandler struct {
	client *xui.Client
}

func registerXrayTools(s *server.MCPServer, client *xui.Client) {
	h := &xrayHandler{client: client}

	// --- Full template tools ---

	s.AddTool(mcp.NewTool("get_xray_template",
		mcp.WithDescription("Get the full Xray template configuration. This includes routing rules, outbounds, DNS settings, and all other Xray config. The template is merged with auto-generated inbound configs by the panel."),
	), h.getTemplate)

	s.AddTool(mcp.NewTool("update_xray_template",
		mcp.WithDescription("Replace the entire Xray template configuration. The template is a JSON string containing routing, outbounds, DNS, policy, etc. WARNING: this replaces the whole config — make sure to include all existing sections you want to keep."),
		mcp.WithString("xray_setting",
			mcp.Required(),
			mcp.Description("Full Xray template config as JSON string"),
		),
		mcp.WithString("test_url",
			mcp.Description("Outbound test URL (default: https://www.google.com/generate_204)"),
		),
	), h.updateTemplate)

	// --- Routing rules convenience tools ---

	s.AddTool(mcp.NewTool("get_routing_rules",
		mcp.WithDescription("Get only the routing section from the Xray template config. Returns the routing object with domainStrategy and rules array."),
	), h.getRoutingRules)

	s.AddTool(mcp.NewTool("add_routing_rule",
		mcp.WithDescription(`Add a new routing rule to the Xray config. The rule is a JSON object. Common fields:
- "type": "field" (always)
- "outboundTag": target outbound name (e.g. "direct", "blocked", "proxy")
- "domain": array of domain patterns (e.g. ["geosite:google", "domain:example.com"])
- "ip": array of IP patterns (e.g. ["geoip:private", "geoip:ru"])
- "port": port or range (e.g. "80,443", "1000-2000")
- "protocol": array of protocols (e.g. ["bittorrent"])
- "inboundTag": array of inbound tags to match
- "network": "tcp", "udp", or "tcp,udp"

Example: {"type":"field","outboundTag":"direct","domain":["geosite:category-ru"]}`),
		mcp.WithString("rule",
			mcp.Required(),
			mcp.Description("Routing rule as JSON object"),
		),
		mcp.WithNumber("index",
			mcp.Description("Position to insert at (0-based). If omitted, appends to end. Rules are evaluated in order — earlier rules take priority."),
		),
	), h.addRule)

	s.AddTool(mcp.NewTool("remove_routing_rule",
		mcp.WithDescription("Remove a routing rule by its index (0-based). Use get_routing_rules first to see current rules and their indices."),
		mcp.WithNumber("index",
			mcp.Required(),
			mcp.Description("Rule index to remove (0-based)"),
		),
	), h.removeRule)

	s.AddTool(mcp.NewTool("update_routing_rule",
		mcp.WithDescription("Replace a routing rule at a specific index with a new rule. Use get_routing_rules first to see current rules."),
		mcp.WithNumber("index",
			mcp.Required(),
			mcp.Description("Rule index to replace (0-based)"),
		),
		mcp.WithString("rule",
			mcp.Required(),
			mcp.Description("New routing rule as JSON object"),
		),
	), h.updateRule)

	// --- Outbound tools ---

	s.AddTool(mcp.NewTool("get_outbounds",
		mcp.WithDescription("Get the outbound configurations from the Xray template. Shows all defined outbounds (proxy, direct, blocked, etc.) with their protocols and settings."),
	), h.getOutbounds)

	s.AddTool(mcp.NewTool("get_outbounds_traffic",
		mcp.WithDescription("Get traffic statistics for all outbound connections."),
	), h.getOutboundsTraffic)

	s.AddTool(mcp.NewTool("reset_outbound_traffic",
		mcp.WithDescription("Reset traffic counters for a specific outbound by its tag name."),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("Outbound tag name"),
		),
	), h.resetOutboundTraffic)

	s.AddTool(mcp.NewTool("test_outbound",
		mcp.WithDescription("Test an outbound configuration for connectivity and measure latency."),
		mcp.WithString("outbound",
			mcp.Required(),
			mcp.Description("Outbound config as JSON string"),
		),
	), h.testOutbound)
}

// --- Template helpers ---

// unmarshalFlexible handles both JSON objects and double-encoded JSON strings.
// 3x-ui may return obj as either a raw JSON object or a string-encoded JSON.
func unmarshalFlexible(raw json.RawMessage) (map[string]any, error) {
	// Try direct unmarshal to map first
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err == nil {
		return result, nil
	}

	// obj is probably a JSON string — unwrap one layer
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return nil, fmt.Errorf("cannot parse as object or string")
	}

	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return nil, fmt.Errorf("unwrapped string is not valid JSON: %w", err)
	}

	return result, nil
}

// extractXraySetting extracts the Xray template from a parsed response object.
// Handles: xraySetting as string (JSON-in-JSON) or as an already-parsed object.
func extractXraySetting(outer map[string]any) (map[string]any, error) {
	raw, ok := outer["xraySetting"]
	if !ok {
		// Maybe the response IS the template directly (has "routing", "outbounds", etc.)
		if _, hasRouting := outer["routing"]; hasRouting {
			return outer, nil
		}
		if _, hasOutbounds := outer["outbounds"]; hasOutbounds {
			return outer, nil
		}
		return nil, fmt.Errorf("xraySetting not found in response")
	}

	switch v := raw.(type) {
	case string:
		var template map[string]any
		if err := json.Unmarshal([]byte(v), &template); err != nil {
			return nil, fmt.Errorf("parsing xraySetting string: %w", err)
		}
		return template, nil
	case map[string]any:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected xraySetting type: %T", v)
	}
}

// fetchTemplate fetches the current Xray template config and parses it into a Go map.
func (h *xrayHandler) fetchTemplate(ctx context.Context) (map[string]any, error) {
	resp, err := h.client.GetXrayTemplate(ctx)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	outer, err := unmarshalFlexible(resp.Obj)
	if err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return extractXraySetting(outer)
}

// saveTemplate marshals the template map back and saves it via the API.
func (h *xrayHandler) saveTemplate(ctx context.Context, template map[string]any) (*xui.Response, error) {
	data, err := json.Marshal(template)
	if err != nil {
		return nil, fmt.Errorf("marshaling template: %w", err)
	}
	return h.client.UpdateXrayTemplate(ctx, string(data), "")
}

// getRouting extracts the routing.rules array from a template.
func getRouting(template map[string]any) (map[string]any, []any, error) {
	routingRaw, ok := template["routing"]
	if !ok {
		// No routing section — create one
		routing := map[string]any{
			"domainStrategy": "AsIs",
			"rules":          []any{},
		}
		template["routing"] = routing
		return routing, []any{}, nil
	}

	routing, ok := routingRaw.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("routing section is not an object")
	}

	rulesRaw, ok := routing["rules"]
	if !ok {
		routing["rules"] = []any{}
		return routing, []any{}, nil
	}

	rules, ok := rulesRaw.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("routing.rules is not an array")
	}

	return routing, rules, nil
}

// --- Tool handlers ---

func (h *xrayHandler) getTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	template, err := h.fetchTemplate(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pretty, _ := json.MarshalIndent(template, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

func (h *xrayHandler) updateTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xraySetting, err := req.RequireString("xray_setting")
	if err != nil {
		return mcp.NewToolResultError("xray_setting is required"), nil
	}
	// Validate it's valid JSON
	var check map[string]any
	if err := json.Unmarshal([]byte(xraySetting), &check); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid JSON: %s", err)), nil
	}
	testUrl := req.GetString("test_url", "")
	return toResult(h.client.UpdateXrayTemplate(ctx, xraySetting, testUrl))
}

func (h *xrayHandler) getRoutingRules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	template, err := h.fetchTemplate(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	routing, rules, err := getRouting(template)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Return routing section with numbered rules for convenience
	result := map[string]any{
		"domainStrategy": routing["domainStrategy"],
		"rules_count":    len(rules),
		"rules":          rules,
	}

	pretty, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

func (h *xrayHandler) addRule(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ruleStr, err := req.RequireString("rule")
	if err != nil {
		return mcp.NewToolResultError("rule is required"), nil
	}

	var rule map[string]any
	if err := json.Unmarshal([]byte(ruleStr), &rule); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid rule JSON: %s", err)), nil
	}

	template, err := h.fetchTemplate(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	routing, rules, err := getRouting(template)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	index := int(req.GetFloat("index", -1))
	if index >= 0 && index < len(rules) {
		// Insert at position
		rules = append(rules[:index], append([]any{rule}, rules[index:]...)...)
	} else {
		// Append to end
		rules = append(rules, rule)
		index = len(rules) - 1
	}

	routing["rules"] = rules

	resp, saveErr := h.saveTemplate(ctx, template)
	if saveErr != nil {
		return mcp.NewToolResultError(saveErr.Error()), nil
	}
	if !resp.Success {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %s", resp.Msg)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Rule added at index %d. Total rules: %d", index, len(rules))), nil
}

func (h *xrayHandler) removeRule(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	indexF, err := req.RequireFloat("index")
	if err != nil {
		return mcp.NewToolResultError("index is required"), nil
	}
	index := int(indexF)

	template, err := h.fetchTemplate(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	routing, rules, err := getRouting(template)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if index < 0 || index >= len(rules) {
		return mcp.NewToolResultError(fmt.Sprintf("index %d out of range (0-%d)", index, len(rules)-1)), nil
	}

	// Show what's being removed
	removed, _ := json.Marshal(rules[index])

	rules = append(rules[:index], rules[index+1:]...)
	routing["rules"] = rules

	resp, saveErr := h.saveTemplate(ctx, template)
	if saveErr != nil {
		return mcp.NewToolResultError(saveErr.Error()), nil
	}
	if !resp.Success {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %s", resp.Msg)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Removed rule at index %d: %s\nRemaining rules: %d", index, string(removed), len(rules))), nil
}

func (h *xrayHandler) updateRule(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	indexF, err := req.RequireFloat("index")
	if err != nil {
		return mcp.NewToolResultError("index is required"), nil
	}
	index := int(indexF)

	ruleStr, err := req.RequireString("rule")
	if err != nil {
		return mcp.NewToolResultError("rule is required"), nil
	}

	var rule map[string]any
	if err := json.Unmarshal([]byte(ruleStr), &rule); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid rule JSON: %s", err)), nil
	}

	template, err := h.fetchTemplate(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	routing, rules, err := getRouting(template)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if index < 0 || index >= len(rules) {
		return mcp.NewToolResultError(fmt.Sprintf("index %d out of range (0-%d)", index, len(rules)-1)), nil
	}

	rules[index] = rule
	routing["rules"] = rules

	resp, saveErr := h.saveTemplate(ctx, template)
	if saveErr != nil {
		return mcp.NewToolResultError(saveErr.Error()), nil
	}
	if !resp.Success {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %s", resp.Msg)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated rule at index %d. Total rules: %d", index, len(rules))), nil
}

func (h *xrayHandler) getOutbounds(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	template, err := h.fetchTemplate(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	outbounds, ok := template["outbounds"]
	if !ok {
		return mcp.NewToolResultText("[]"), nil
	}

	pretty, _ := json.MarshalIndent(outbounds, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

func (h *xrayHandler) getOutboundsTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.GetOutboundsTraffic(ctx))
}

func (h *xrayHandler) resetOutboundTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError("tag is required"), nil
	}
	return toResult(h.client.ResetOutboundTraffic(ctx, tag))
}

func (h *xrayHandler) testOutbound(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	outbound, err := req.RequireString("outbound")
	if err != nil {
		return mcp.NewToolResultError("outbound is required"), nil
	}
	return toResult(h.client.TestOutbound(ctx, outbound))
}
