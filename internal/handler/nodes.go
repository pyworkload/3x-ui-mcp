package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// nodeHandler holds the XUI client for multi-node tool handlers.
type nodeHandler struct {
	client *xui.Client
}

// Connection details shared by add_node, update_node, test_node,
// list_node_inbounds and get_node_cert_fingerprint. The panel validates name,
// address and port on every one of them, including the probe-only routes.
func nodeParams(required bool) []mcp.ToolOption {
	name := []mcp.PropertyOption{mcp.Description("Node name, unique in this panel")}
	address := []mcp.PropertyOption{mcp.Description("Node hostname or IP")}
	if required {
		name = append(name, mcp.Required())
		address = append(address, mcp.Required())
	}
	return []mcp.ToolOption{
		mcp.WithString("name", name...),
		mcp.WithString("address", address...),
		mcp.WithNumber("port",
			mcp.Description("Node panel port"),
		),
		mcp.WithString("scheme",
			mcp.Description("How to reach the node panel"),
			mcp.Enum("http", "https"),
		),
		mcp.WithString("base_path",
			mcp.Description("Node panel base path, e.g. / or /xui/"),
		),
		mcp.WithString("api_token",
			mcp.Description("The node's API token. Write-only: the panel never returns it, and omitting it on an update keeps the stored one."),
		),
		mcp.WithBoolean("clear_api_token",
			mcp.Description("Drop the stored API token — for a node switched to mTLS. Cannot be combined with api_token."),
		),
		mcp.WithString("remark",
			mcp.Description("Free-text note"),
		),
		mcp.WithBoolean("enable",
			mcp.Description("Whether traffic sync with this node runs"),
		),
		mcp.WithBoolean("allow_private_address",
			mcp.Description("Permit a private or loopback address, which the panel otherwise rejects"),
		),
		mcp.WithString("tls_verify_mode",
			mcp.Description("How the node's certificate is checked: verify (normal), skip, pin (see get_node_cert_fingerprint), or mtls"),
			mcp.Enum("verify", "skip", "pin", "mtls"),
		),
		mcp.WithString("pinned_cert_sha256",
			mcp.Description("Base64 SHA-256 of the node's leaf certificate, for tls_verify_mode=pin"),
		),
		mcp.WithString("inbound_sync_mode",
			mcp.Description("Push every inbound to the node, or only the tagged ones"),
			mcp.Enum("all", "selected"),
		),
		mcp.WithArray("inbound_tags",
			mcp.Description("Inbound tags to sync when inbound_sync_mode=selected"),
			mcp.WithStringItems(),
		),
		mcp.WithString("outbound_tag",
			mcp.Description("Reach the node through this Xray outbound instead of directly"),
		),
	}
}

func registerNodeTools(s *server.MCPServer, client *xui.Client) {
	h := &nodeHandler{client: client}

	s.AddTool(mcp.NewTool("list_nodes",
		readsPanel,
		mcp.WithDescription("List the remote nodes this panel drives, with their health, client counts and last heartbeat. A node is another 3x-ui panel that this one pushes inbounds and clients to and pulls traffic back from."),
	), h.list)

	s.AddTool(mcp.NewTool("get_node",
		readsPanel,
		mcp.WithDescription("Get one node by ID, including its connection details and sync state. The API token is never returned — only whether one is set."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Node ID"),
		),
	), h.get)

	s.AddTool(mcp.NewTool("get_node_history",
		readsPanel,
		mcp.WithDescription("CPU or memory history for one node, in the same {t, v} shape as get_metrics_history."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Node ID"),
		),
		mcp.WithString("metric",
			mcp.Required(),
			mcp.Description("Metric to chart"),
			mcp.Enum("cpu", "mem"),
		),
		mcp.WithNumber("bucket",
			mcp.Description("Seconds per sample across 60 samples: 2 (2m), 30 (30m), 60 (1h), 180 (3h), 360 (6h), 720 (12h), 1440 (24h), 2880 (2d), 10080 (7d)"),
			mcp.DefaultNumber(360),
		),
	), h.history)

	s.AddTool(mcp.NewTool("get_node_web_cert",
		probesRemote,
		mcp.WithDescription("Ask a node for its own web TLS certificate and key paths, so an inbound assigned to that node points at files that exist there rather than on this panel."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Node ID"),
		),
	), h.webCert)

	s.AddTool(mcp.NewTool("test_node",
		append([]mcp.ToolOption{
			probesRemote,
			mcp.WithDescription("Probe connection details without saving them and return the heartbeat a registered node would report: status, latency, CPU, memory, panel and Xray versions. Run this before add_node."),
		}, nodeParams(true)...)...,
	), h.test)

	s.AddTool(mcp.NewTool("list_node_inbounds",
		append([]mcp.ToolOption{
			probesRemote,
			mcp.WithDescription("List the inbounds that exist on a node, using connection details that need not be saved yet. This is what inbound_sync_mode=selected picks from."),
		}, nodeParams(true)...)...,
	), h.inbounds)

	s.AddTool(mcp.NewTool("get_node_cert_fingerprint",
		append([]mcp.ToolOption{
			probesRemote,
			mcp.WithDescription("Connect to a node without verifying its certificate and return the leaf's SHA-256 in base64 — the value to pass as pinned_cert_sha256 for a self-signed node."),
		}, nodeParams(true)...)...,
	), h.certFingerprint)

	s.AddTool(mcp.NewTool("add_node",
		append([]mcp.ToolOption{
			fetchesRemote,
			mcp.WithDescription("Register a node. The panel probes it before saving, so unreachable details are rejected rather than stored. api_token is required unless the node is set up for mTLS."),
		}, nodeParams(true)...)...,
	), h.add)

	s.AddTool(mcp.NewTool("update_node",
		append([]mcp.ToolOption{
			updatesRemote,
			mcp.WithDescription("Update a node's connection details. Reads the stored node first and overlays only what you pass, so omitted fields survive — including the API token, which the panel never hands back."),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("Node ID to update"),
			),
		}, nodeParams(false)...)...,
	), h.update)

	s.AddTool(mcp.NewTool("set_node_enable",
		updatesPanel,
		mcp.WithDescription("Pause or resume traffic sync with a node. The node keeps running; this panel just stops pushing to it and pulling from it."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Node ID"),
		),
		mcp.WithBoolean("enable",
			mcp.Required(),
			mcp.Description("true to resume sync, false to pause it"),
		),
	), h.setEnable)

	s.AddTool(mcp.NewTool("probe_node",
		fetchesRemote,
		mcp.WithDescription("Probe a registered node now and refresh its cached health state, instead of waiting for the next heartbeat."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Node ID"),
		),
	), h.probe)

	s.AddTool(mcp.NewTool("delete_node",
		destroysPanel,
		mcp.WithDescription("Delete a node. Inbounds bound to it are not migrated anywhere, so they stop being served until they are reassigned."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Node ID to delete"),
		),
	), h.delete)

	s.AddTool(mcp.NewTool("get_node_mtls_ca",
		writesPanel,
		mcp.WithDescription("Return this panel's node-auth CA certificate in PEM, to paste into a node's trust setting. The CA and the master client certificate are minted on the first call, which is why this is not a pure read."),
	), h.mtlsCA)

	s.AddTool(mcp.NewTool("set_node_mtls_trust_ca",
		updatesPanel,
		mcp.WithDescription("Set the CA this panel trusts for incoming node-API client certificates — the setting used when this panel is itself a node. Pass the managing panel's CA from its get_node_mtls_ca; an empty string disables mTLS."),
		mcp.WithString("ca_cert",
			mcp.Required(),
			mcp.Description("CA certificate in PEM, or an empty string to disable"),
		),
	), h.trustCA)

	s.AddTool(mcp.NewTool("reload_node_mtls_client",
		updatesPanel,
		mcp.WithDescription("Revalidate the master mTLS credential and rebuild the cached transports, so a rotated certificate takes effect without restarting the panel."),
	), h.reloadMtls)

	s.AddTool(mcp.NewTool("update_node_panels",
		installsRemote,
		mcp.WithDescription("Run the 3x-ui self-updater on the given nodes: each downloads the latest release and restarts. Only enabled, online nodes are updated; the rest come back as skipped."),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Description("Node IDs to update"),
			mcp.WithNumberItems(),
		),
		mcp.WithBoolean("dev",
			mcp.Description("Install the dev build instead of the latest stable release"),
			mcp.DefaultBool(false),
		),
	), h.updatePanels)
}

func (h *nodeHandler) list(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ListNodes(ctx))
}

func (h *nodeHandler) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.GetNode(ctx, int(id)))
}

func (h *nodeHandler) history(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	metric, err := req.RequireString("metric")
	if err != nil {
		return mcp.NewToolResultError("metric is required"), nil
	}
	bucket := int(req.GetFloat("bucket", 360))
	if err := checkBucket(bucket); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.NodeHistory(ctx, int(id), metric, bucket))
}

func (h *nodeHandler) webCert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.NodeWebCert(ctx, int(id)))
}

func (h *nodeHandler) test(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	node, err := nodeBody(req, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.TestNode(ctx, node))
}

func (h *nodeHandler) inbounds(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	node, err := nodeBody(req, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.NodeInbounds(ctx, node))
}

func (h *nodeHandler) certFingerprint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	node, err := nodeBody(req, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.NodeCertFingerprint(ctx, node))
}

func (h *nodeHandler) add(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	node, err := nodeBody(req, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.AddNode(ctx, node))
}

func (h *nodeHandler) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	cur, err := h.client.GetNode(ctx, int(id))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !cur.Success {
		return mcp.NewToolResultError("API error: " + cur.Msg), nil
	}
	stored := map[string]any{}
	if err := json.Unmarshal(cur.Obj, &stored); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parsing the stored node: %v", err)), nil
	}

	node, err := nodeBody(req, storedNodeFields(stored))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.UpdateNode(ctx, int(id), node))
}

func (h *nodeHandler) setEnable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	enable, err := req.RequireBool("enable")
	if err != nil {
		return mcp.NewToolResultError("enable is required"), nil
	}
	return toResult(h.client.SetNodeEnable(ctx, int(id), enable))
}

func (h *nodeHandler) probe(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.ProbeNode(ctx, int(id)))
}

func (h *nodeHandler) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}
	return toResult(h.client.DeleteNode(ctx, int(id)))
}

func (h *nodeHandler) mtlsCA(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.NodeMtlsCA(ctx))
}

func (h *nodeHandler) trustCA(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	caCert, err := req.RequireString("ca_cert")
	if err != nil {
		return mcp.NewToolResultError("ca_cert is required (pass an empty string to disable mTLS)"), nil
	}
	return toResult(h.client.SetNodeMtlsTrustCA(ctx, caCert))
}

func (h *nodeHandler) reloadMtls(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.ReloadNodeMtlsClient(ctx))
}

func (h *nodeHandler) updatePanels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ids := req.GetIntSlice("ids", nil)
	if len(ids) == 0 {
		return mcp.NewToolResultError("ids is required (at least one node ID)"), nil
	}
	return toResult(h.client.UpdateNodePanels(ctx, ids, req.GetBool("dev", false)))
}

// storedNodeFields keeps the parts of a node view that belong in a mutation
// request. The view also carries heartbeat and counter fields, which the
// panel's validator would reject.
func storedNodeFields(view map[string]any) map[string]any {
	body := map[string]any{}
	for _, key := range []string{
		"name", "remark", "scheme", "address", "port", "basePath", "enable",
		"allowPrivateAddress", "tlsVerifyMode", "pinnedCertSha256",
		"inboundSyncMode", "inboundTags", "outboundTag",
	} {
		if v, ok := view[key]; ok {
			body[key] = v
		}
	}
	return body
}

// nodeBody overlays the supplied parameters onto base — empty for a create or
// probe, the stored node for an update. apiToken is only ever sent when the
// caller passed one, since the panel treats its absence as "keep the stored
// token" and never returns it for us to echo back.
func nodeBody(req mcp.CallToolRequest, base map[string]any) (map[string]any, error) {
	args := req.GetArguments()
	supplied := func(param string) bool {
		_, ok := args[param]
		return ok
	}

	for param, key := range map[string]string{
		"name":               "name",
		"remark":             "remark",
		"scheme":             "scheme",
		"address":            "address",
		"base_path":          "basePath",
		"api_token":          "apiToken",
		"tls_verify_mode":    "tlsVerifyMode",
		"pinned_cert_sha256": "pinnedCertSha256",
		"inbound_sync_mode":  "inboundSyncMode",
		"outbound_tag":       "outboundTag",
	} {
		if supplied(param) {
			base[key] = req.GetString(param, "")
		}
	}
	for param, key := range map[string]string{
		"clear_api_token":       "clearApiToken",
		"enable":                "enable",
		"allow_private_address": "allowPrivateAddress",
	} {
		if supplied(param) {
			base[key] = req.GetBool(param, false)
		}
	}
	if supplied("port") {
		base["port"] = int(req.GetFloat("port", 0))
	}
	if supplied("inbound_tags") {
		base["inboundTags"] = req.GetStringSlice("inbound_tags", nil)
	}

	if supplied("api_token") && req.GetBool("clear_api_token", false) {
		return nil, fmt.Errorf("api_token and clear_api_token are mutually exclusive: pass one or the other")
	}
	return base, nil
}
