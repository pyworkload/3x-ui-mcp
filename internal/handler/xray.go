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
		readsPanel,
		mcp.WithDescription("Summarize the saved Xray template — routing rules, balancers, outbounds and DNS — and link to the full JSON at xui://xray/template. The panel merges this template with the auto-generated inbound configs. Pass full=true to get the whole document inline, which is what update_xray_template expects back."),
		mcp.WithBoolean("full",
			mcp.Description("Return the entire template inline instead of a summary and a link"),
			mcp.DefaultBool(false),
		),
	), h.getTemplate)

	s.AddTool(mcp.NewTool("update_xray_template",
		destroysPanel,
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
		readsPanel,
		mcp.WithDescription("Get only the routing section from the Xray template config. Returns the routing object with domainStrategy and rules array."),
	), h.getRoutingRules)

	s.AddTool(mcp.NewTool("add_routing_rule",
		writesPanel,
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
		destroysPanel,
		mcp.WithDescription("Remove a routing rule by its index (0-based). Use get_routing_rules first to see current rules and their indices."),
		mcp.WithNumber("index",
			mcp.Required(),
			mcp.Description("Rule index to remove (0-based)"),
		),
	), h.removeRule)

	s.AddTool(mcp.NewTool("update_routing_rule",
		updatesPanel,
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
		readsPanel,
		mcp.WithDescription("Get the outbound configurations from the Xray template. Shows all defined outbounds (proxy, direct, blocked, etc.) with their protocols and settings."),
	), h.getOutbounds)

	s.AddTool(mcp.NewTool("get_outbounds_traffic",
		readsPanel,
		mcp.WithDescription("Get traffic statistics for all outbound connections."),
	), h.getOutboundsTraffic)

	s.AddTool(mcp.NewTool("reset_outbound_traffic",
		destroysPanel,
		mcp.WithDescription("Reset traffic counters for a specific outbound by its tag name."),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("Outbound tag name"),
		),
	), h.resetOutboundTraffic)

	s.AddTool(mcp.NewTool("test_outbound",
		probesRemote,
		mcp.WithDescription("Test an outbound configuration for connectivity and measure latency."),
		mcp.WithString("outbound",
			mcp.Required(),
			mcp.Description("Outbound config as JSON string"),
		),
	), h.testOutbound)

	// --- Balancer & routing diagnostics ---

	s.AddTool(mcp.NewTool("get_balancers",
		readsPanel,
		mcp.WithDescription("List the load balancers defined in routing.balancers together with their live state in the running Xray core. "+
			"For each balancer: the saved definition (selector, fallbackTag, strategy) plus 'live' with running, override (the outbound it is pinned to, if any) and selected (what the strategy picks right now). "+
			"Routing rules reference a balancer through 'balancerTag' instead of 'outboundTag'."),
	), h.getBalancers)

	s.AddTool(mcp.NewTool("set_balancer_override",
		updatesPanel,
		mcp.WithDescription("Pin a balancer to one outbound in the running core, bypassing its strategy. Applies immediately without a restart, and is cleared when Xray restarts. "+
			"Leave 'target' empty to release the override and hand control back to the strategy. Use get_balancers for valid tags and get_outbounds for target names."),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("Balancer tag to override"),
		),
		mcp.WithString("target",
			mcp.Description("Outbound tag to pin to. Empty clears the override."),
		),
	), h.setBalancerOverride)

	s.AddTool(mcp.NewTool("test_route",
		readsPanel,
		mcp.WithDescription("Ask the running Xray core which outbound its router would pick for a synthetic connection. No traffic is sent. "+
			"Returns matched (false means no rule matched and the default outbound would be used), outboundTag, and groupTags for the balancer chain the decision went through. "+
			"Either domain or ip is required."),
		mcp.WithString("domain",
			mcp.Description("Target domain, e.g. 'www.youtube.com'"),
		),
		mcp.WithString("ip",
			mcp.Description("Target IP, used when domain is empty or alongside it"),
		),
		mcp.WithNumber("port",
			mcp.Description("Destination port, e.g. 443"),
		),
		mcp.WithString("network",
			mcp.Description("'tcp' (default) or 'udp'"),
		),
		mcp.WithString("protocol",
			mcp.Description("Sniffed protocol to simulate: http, tls, bittorrent, ..."),
		),
		mcp.WithString("inbound_tag",
			mcp.Description("Simulate the connection arriving on this inbound tag"),
		),
		mcp.WithString("email",
			mcp.Description("Client email, for rules that match on user"),
		),
	), h.testRoute)

	// --- Outbound subscriptions ---

	s.AddTool(mcp.NewTool("list_outbound_subs",
		readsPanel,
		mcp.WithDescription("List the outbound subscriptions — remote URLs that supply extra outbounds, fetched on a timer and merged into the running Xray config. "+
			"Shows each subscription's id, remark, url, tagPrefix, enabled flag, updateInterval, priority, prepend, outboundCount, lastUpdated and lastError."),
	), h.listOutboundSubs)

	s.AddTool(mcp.NewTool("preview_outbound_sub",
		probesRemote,
		mcp.WithDescription("Fetch and parse a subscription URL without saving it — use this to see what outbounds it would contribute before creating it."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Subscription URL"),
		),
		mcp.WithBoolean("allow_private",
			mcp.Description("Accept outbounds pointing at private/LAN addresses"),
			mcp.DefaultBool(false),
		),
		mcp.WithBoolean("allow_insecure",
			mcp.Description("Accept TLS outbounds with certificate verification disabled"),
			mcp.DefaultBool(false),
		),
	), h.previewOutboundSub)

	s.AddTool(mcp.NewTool("create_outbound_sub",
		fetchesRemote,
		mcp.WithDescription("Add an outbound subscription. The panel fetches the URL, parses it into outbounds with stable tags derived from tag_prefix, and merges them into the config additively — the template's own outbounds stay."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Subscription URL"),
		),
		mcp.WithString("remark",
			mcp.Description("Display name for the subscription"),
		),
		mcp.WithString("tag_prefix",
			mcp.Description("Prefix for the generated outbound tags, e.g. 'sub1-'"),
		),
		mcp.WithBoolean("enabled",
			mcp.Description("Fetch and use this subscription"),
			mcp.DefaultBool(true),
		),
		mcp.WithNumber("update_interval",
			mcp.Description("Seconds between refreshes (default 600)"),
			mcp.DefaultNumber(600),
		),
		mcp.WithBoolean("allow_private",
			mcp.Description("Accept outbounds pointing at private/LAN addresses"),
			mcp.DefaultBool(false),
		),
		mcp.WithBoolean("allow_insecure",
			mcp.Description("Accept TLS outbounds with certificate verification disabled"),
			mcp.DefaultBool(false),
		),
		mcp.WithBoolean("prepend",
			mcp.Description("Place these outbounds before the template's own — the first outbound is Xray's default route"),
			mcp.DefaultBool(false),
		),
	), h.createOutboundSub)

	s.AddTool(mcp.NewTool("update_outbound_sub",
		updatesPanel,
		mcp.WithDescription("Update an outbound subscription. Pass only the fields you want to change — the rest are read from the current subscription first, since the panel endpoint overwrites every field."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Subscription ID (from list_outbound_subs)"),
		),
		mcp.WithString("url",
			mcp.Description("Subscription URL"),
		),
		mcp.WithString("remark",
			mcp.Description("Display name"),
		),
		mcp.WithString("tag_prefix",
			mcp.Description("Prefix for generated outbound tags"),
		),
		mcp.WithBoolean("enabled",
			mcp.Description("Fetch and use this subscription"),
		),
		mcp.WithNumber("update_interval",
			mcp.Description("Seconds between refreshes"),
		),
		mcp.WithBoolean("allow_private",
			mcp.Description("Accept outbounds pointing at private/LAN addresses"),
		),
		mcp.WithBoolean("allow_insecure",
			mcp.Description("Accept TLS outbounds with certificate verification disabled"),
		),
		mcp.WithBoolean("prepend",
			mcp.Description("Place these outbounds before the template's own"),
		),
	), h.updateOutboundSub)

	s.AddTool(mcp.NewTool("refresh_outbound_sub",
		fetchesRemote,
		mcp.WithDescription("Re-fetch a subscription right now instead of waiting for its interval, and return the outbounds it parsed."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Subscription ID"),
		),
	), h.refreshOutboundSub)

	s.AddTool(mcp.NewTool("move_outbound_sub",
		writesPanel,
		mcp.WithDescription("Move a subscription one step up or down in priority. Order decides where its outbounds sit in the merged config."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Subscription ID"),
		),
		mcp.WithString("direction",
			mcp.Required(),
			mcp.Enum("up", "down"),
			mcp.Description("Direction to move"),
		),
	), h.moveOutboundSub)

	s.AddTool(mcp.NewTool("delete_outbound_sub",
		destroysPanel,
		mcp.WithDescription("Delete an outbound subscription. Its outbounds disappear from the merged config on the next Xray reload; routing rules that referenced their tags will stop matching."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Subscription ID"),
		),
	), h.deleteOutboundSub)
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

// getBalancersFromTemplate returns the routing.balancers array, or nil when the
// template defines none. A malformed section is treated as absent — balancers
// are optional, and no caller should fail because of a broken one.
func getBalancersFromTemplate(template map[string]any) []any {
	routing, ok := template["routing"].(map[string]any)
	if !ok {
		return nil
	}
	balancers, ok := routing["balancers"].([]any)
	if !ok {
		return nil
	}
	return balancers
}

// balancerTags collects the tag of every balancer definition, skipping entries
// without one (the panel ignores those too).
func balancerTags(balancers []any) []string {
	tags := make([]string, 0, len(balancers))
	for _, raw := range balancers {
		b, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if tag, ok := b["tag"].(string); ok && tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// mergeBalancerStatus overlays the live per-tag state onto the saved balancer
// definitions. Definitions with no matching status keep their config only, so
// a balancer that exists in the template but not in the running core still shows up.
func mergeBalancerStatus(balancers []any, status map[string]any) []any {
	merged := make([]any, 0, len(balancers))
	for _, raw := range balancers {
		b, ok := raw.(map[string]any)
		if !ok {
			merged = append(merged, raw)
			continue
		}
		entry := make(map[string]any, len(b)+1)
		for k, v := range b {
			entry[k] = v
		}
		if tag, ok := b["tag"].(string); ok {
			if live, found := status[tag]; found {
				entry["live"] = live
			}
		}
		merged = append(merged, entry)
	}
	return merged
}

// --- Tool handlers ---

func (h *xrayHandler) getTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.GetXrayTemplate(ctx)
	if req.GetBool("full", false) {
		return toResult(resp, err)
	}
	return linkedResult(resp, err, mcp.NewResourceLink(
		xrayTemplateURI,
		"Xray template",
		"The saved template in full. Read it before editing, since update_xray_template replaces the whole document.",
		"application/json",
	), summarizeXrayConfig)
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

	// Return routing section with numbered rules for convenience. Balancers
	// come along because a rule's balancerTag is unreadable without them.
	result := map[string]any{
		"domainStrategy": routing["domainStrategy"],
		"rules_count":    len(rules),
		"rules":          rules,
	}
	if balancers := getBalancersFromTemplate(template); len(balancers) > 0 {
		result["balancers"] = balancers
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

func (h *xrayHandler) getBalancers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	template, err := h.fetchTemplate(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	balancers := getBalancersFromTemplate(template)
	if len(balancers) == 0 {
		return mcp.NewToolResultText("No balancers defined in routing.balancers."), nil
	}

	result := map[string]any{"balancers_count": len(balancers)}

	// The live state is a bonus on top of the saved config: if the core can't
	// be reached the definitions are still worth returning, with the reason.
	status, err := h.balancerStatus(ctx, balancerTags(balancers))
	if err != nil {
		result["live_status_error"] = err.Error()
		result["balancers"] = balancers
	} else {
		result["balancers"] = mergeBalancerStatus(balancers, status)
	}

	pretty, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

// balancerStatus fetches the live state of the given tags, keyed by tag.
func (h *xrayHandler) balancerStatus(ctx context.Context, tags []string) (map[string]any, error) {
	if len(tags) == 0 {
		return map[string]any{}, nil
	}
	resp, err := h.client.GetBalancersStatus(ctx, tags)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}
	var status map[string]any
	if err := json.Unmarshal(resp.Obj, &status); err != nil {
		return nil, fmt.Errorf("parsing balancer status: %w", err)
	}
	return status, nil
}

func (h *xrayHandler) setBalancerOverride(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError("tag is required"), nil
	}
	target := req.GetString("target", "")

	resp, err := h.client.OverrideBalancer(ctx, tag, target)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !resp.Success {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %s", resp.Msg)), nil
	}
	if target == "" {
		return mcp.NewToolResultText(fmt.Sprintf("Override cleared for balancer %q — its strategy is in control again.", tag)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Balancer %q is now pinned to outbound %q. The override is lost on the next Xray restart.", tag, target)), nil
}

func (h *xrayHandler) testRoute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := xui.RouteTestRequest{
		InboundTag: req.GetString("inbound_tag", ""),
		Domain:     req.GetString("domain", ""),
		IP:         req.GetString("ip", ""),
		Port:       int(req.GetFloat("port", 0)),
		Network:    req.GetString("network", ""),
		Protocol:   req.GetString("protocol", ""),
		Email:      req.GetString("email", ""),
	}
	if params.Domain == "" && params.IP == "" {
		return mcp.NewToolResultError("either domain or ip is required"), nil
	}
	return toResult(h.client.TestRoute(ctx, params))
}

// --- Outbound subscription handlers ---

// outboundSub is the subset of a stored subscription needed to re-send it on
// update. The panel's update endpoint replaces every field, so a partial edit
// has to start from the current row.
type outboundSub struct {
	ID             int    `json:"id"`
	Remark         string `json:"remark"`
	URL            string `json:"url"`
	TagPrefix      string `json:"tagPrefix"`
	Enabled        bool   `json:"enabled"`
	UpdateInterval int    `json:"updateInterval"`
	AllowPrivate   bool   `json:"allowPrivate"`
	AllowInsecure  bool   `json:"allowInsecure"`
	Prepend        bool   `json:"prepend"`
}

func (s outboundSub) params() xui.OutboundSubParams {
	return xui.OutboundSubParams{
		Remark:         s.Remark,
		URL:            s.URL,
		TagPrefix:      s.TagPrefix,
		Enabled:        s.Enabled,
		UpdateInterval: s.UpdateInterval,
		AllowPrivate:   s.AllowPrivate,
		AllowInsecure:  s.AllowInsecure,
		Prepend:        s.Prepend,
	}
}

// findOutboundSub fetches the subscription list and returns the entry with the
// given id.
func (h *xrayHandler) findOutboundSub(ctx context.Context, id int) (outboundSub, error) {
	resp, err := h.client.ListOutboundSubs(ctx)
	if err != nil {
		return outboundSub{}, err
	}
	if !resp.Success {
		return outboundSub{}, fmt.Errorf("API error: %s", resp.Msg)
	}
	var subs []outboundSub
	if err := json.Unmarshal(resp.Obj, &subs); err != nil {
		return outboundSub{}, fmt.Errorf("parsing subscription list: %w", err)
	}
	for _, sub := range subs {
		if sub.ID == id {
			return sub, nil
		}
	}
	return outboundSub{}, fmt.Errorf("no outbound subscription with id %d", id)
}

func (h *xrayHandler) listOutboundSubs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListOutboundSubs(ctx))
}

func (h *xrayHandler) previewOutboundSub(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawURL, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError("url is required"), nil
	}
	return toResult(h.client.ParseOutboundSubURL(ctx, rawURL,
		req.GetBool("allow_private", false),
		req.GetBool("allow_insecure", false),
	))
}

func (h *xrayHandler) createOutboundSub(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawURL, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError("url is required"), nil
	}
	return toResult(h.client.CreateOutboundSub(ctx, xui.OutboundSubParams{
		Remark:         req.GetString("remark", ""),
		URL:            rawURL,
		TagPrefix:      req.GetString("tag_prefix", ""),
		Enabled:        req.GetBool("enabled", true),
		UpdateInterval: int(req.GetFloat("update_interval", 600)),
		AllowPrivate:   req.GetBool("allow_private", false),
		AllowInsecure:  req.GetBool("allow_insecure", false),
		Prepend:        req.GetBool("prepend", false),
	}))
}

func (h *xrayHandler) updateOutboundSub(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	current, err := h.findOutboundSub(ctx, int(id))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Every omitted parameter falls back to what the subscription already has.
	params := current.params()
	params.Remark = req.GetString("remark", params.Remark)
	params.URL = req.GetString("url", params.URL)
	params.TagPrefix = req.GetString("tag_prefix", params.TagPrefix)
	params.Enabled = req.GetBool("enabled", params.Enabled)
	params.UpdateInterval = int(req.GetFloat("update_interval", float64(params.UpdateInterval)))
	params.AllowPrivate = req.GetBool("allow_private", params.AllowPrivate)
	params.AllowInsecure = req.GetBool("allow_insecure", params.AllowInsecure)
	params.Prepend = req.GetBool("prepend", params.Prepend)

	return toResult(h.client.UpdateOutboundSub(ctx, int(id), params))
}

func (h *xrayHandler) refreshOutboundSub(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.RefreshOutboundSub(ctx, int(id)))
}

func (h *xrayHandler) moveOutboundSub(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	direction, err := req.RequireString("direction")
	if err != nil {
		return mcp.NewToolResultError("direction is required"), nil
	}
	switch direction {
	case "up", "down":
	default:
		return mcp.NewToolResultError(fmt.Sprintf("direction must be 'up' or 'down', got %q", direction)), nil
	}
	return toResult(h.client.MoveOutboundSub(ctx, int(id), direction == "up"))
}

func (h *xrayHandler) deleteOutboundSub(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.DeleteOutboundSub(ctx, int(id)))
}
