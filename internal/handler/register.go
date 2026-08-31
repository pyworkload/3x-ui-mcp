package handler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/server"
)

// toolsets maps a group name to the tools it registers.
//
// All 166 tool schemas together cost roughly 28k tokens of context in every
// session, whether or not the caller ever touches them. XUI_TOOLSETS narrows
// that: a deployment that only hands out keys can load "clients" and skip the
// Xray template, geodata and metrics tools entirely.
var toolsets = map[string]func(*server.MCPServer, *xui.Client){
	"inbounds": registerInboundTools,
	"clients":  registerClientTools,
	"server":   registerServerTools,
	"metrics":  registerMetricsTools,
	"groups":   registerGroupTools,
	"geodata":  registerGeodataTools,
	"hosts":    registerHostTools,
	"nodes":    registerNodeTools,
	"tokens":   registerAPITokenTools,
	// Panel upkeep: health checks, certificate helpers, geo files, panel
	// updates and the settings probes.
	"maintenance": registerMaintenanceTools,
	// The outbound providers reach Warp, NordVPN and PIA rather than the panel,
	// so they are opt-out separately from the rest of the Xray tooling.
	"providers": registerProviderTools,
	// Subscription balancers ride with the Xray group: they are the client-side
	// counterpart to the routing balancers already in there.
	"xray": func(s *server.MCPServer, client *xui.Client) {
		registerXrayTools(s, client)
		registerSubBalancerTools(s, client)
	},
}

// ToolsetNames lists the available groups in a stable order.
func ToolsetNames() []string {
	names := make([]string, 0, len(toolsets))
	for name := range toolsets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisterAll registers the requested tool groups plus the resources, which are
// always available: they cost one line each in resources/list and are only read
// when something asks for them. An empty or "all" selection enables every group.
func RegisterAll(s *server.MCPServer, client *xui.Client, enabled []string) error {
	registerResources(s, client)

	if len(enabled) == 0 {
		enabled = ToolsetNames()
	}
	for _, name := range enabled {
		if name == "all" {
			enabled = ToolsetNames()
			break
		}
	}

	var unknown []string
	for _, name := range enabled {
		register, ok := toolsets[name]
		if !ok {
			unknown = append(unknown, name)
			continue
		}
		register(s, client)
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown toolset(s) %s; available: %s",
			strings.Join(unknown, ", "), strings.Join(ToolsetNames(), ", "))
	}
	return nil
}
