package xui

import (
	"context"
	"fmt"
	"net/url"
)

// --- Multi-node API methods (panel v3.5.0+) ---
//
// A node is another 3x-ui panel this one drives over its API: inbounds and
// clients are pushed to it and its traffic is pulled back. The panel talks to a
// node with an API token or an mTLS client certificate, so most of these calls
// reach a remote host rather than staying local.
//
// The node API token is write-only from v3.6.0 on: responses report only
// whether a token is set, so a caller that omits it on update keeps the stored
// one and can never read it back.

const nodesBase = "panel/api/nodes/"

// ListNodes returns every configured node with health and last heartbeat.
func (c *Client) ListNodes(ctx context.Context) (*Response, error) {
	return c.Get(ctx, nodesBase+"list")
}

// GetNode returns one node by ID.
func (c *Client) GetNode(ctx context.Context, id int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf(nodesBase+"get/%d", id))
}

// NodeHistory returns a node's cpu or mem history, same {t, v} shape and same
// bucket list as the panel's own history routes.
func (c *Client) NodeHistory(ctx context.Context, id int, metric string, bucketSeconds int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf(nodesBase+"history/%d/%s/%d", id, url.PathEscape(metric), bucketSeconds))
}

// NodeWebCert returns the certificate and key paths that exist on the node, so
// an inbound assigned to it can be pointed at files the node actually has.
func (c *Client) NodeWebCert(ctx context.Context, id int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf(nodesBase+"webCert/%d", id))
}

// AddNode registers a node. The body is a NodeMutationRequest: name, address
// and port are required, and apiToken is required unless mTLS is configured.
func (c *Client) AddNode(ctx context.Context, node map[string]any) (*Response, error) {
	return c.PostJSON(ctx, nodesBase+"add", node)
}

// UpdateNode replaces a node's connection details. Omitting apiToken keeps the
// stored one; clearApiToken drops it, and the two are mutually exclusive.
func (c *Client) UpdateNode(ctx context.Context, id int, node map[string]any) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf(nodesBase+"update/%d", id), node)
}

// DeleteNode removes a node. Inbounds bound to it are not migrated anywhere.
func (c *Client) DeleteNode(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf(nodesBase+"del/%d", id))
}

// SetNodeEnable pauses or resumes traffic sync with a node.
func (c *Client) SetNodeEnable(ctx context.Context, id int, enable bool) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf(nodesBase+"setEnable/%d", id), map[string]any{
		"enable": enable,
	})
}

// ProbeNode probes a registered node and refreshes its cached health.
func (c *Client) ProbeNode(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf(nodesBase+"probe/%d", id))
}

// TestNode probes connection details without saving them, returning the same
// heartbeat snapshot a registered node would report.
func (c *Client) TestNode(ctx context.Context, node map[string]any) (*Response, error) {
	return c.PostJSON(ctx, nodesBase+"test", node)
}

// NodeInbounds lists the inbounds available on a node, for selective import.
func (c *Client) NodeInbounds(ctx context.Context, node map[string]any) (*Response, error) {
	return c.PostJSON(ctx, nodesBase+"inbounds", node)
}

// NodeCertFingerprint connects without verifying the certificate and returns
// the leaf's SHA-256, which is what pinning a self-signed node needs.
func (c *Client) NodeCertFingerprint(ctx context.Context, node map[string]any) (*Response, error) {
	return c.PostJSON(ctx, nodesBase+"certFingerprint", node)
}

// NodeMtlsCA returns this panel's node-auth CA in PEM. The CA and the master
// client certificate are minted lazily on the first call.
func (c *Client) NodeMtlsCA(ctx context.Context) (*Response, error) {
	return c.Post(ctx, nodesBase+"mtls/ca")
}

// SetNodeMtlsTrustCA sets the CA this panel trusts for incoming node-API client
// certificates, i.e. when this panel is itself a node. An empty PEM disables it.
func (c *Client) SetNodeMtlsTrustCA(ctx context.Context, caCert string) (*Response, error) {
	return c.PostJSON(ctx, nodesBase+"mtls/trustCA", map[string]any{"caCert": caCert})
}

// ReloadNodeMtlsClient revalidates the master mTLS credential and rebuilds the
// cached transports, which is how a rotated certificate takes effect without a
// panel restart.
func (c *Client) ReloadNodeMtlsClient(ctx context.Context) (*Response, error) {
	return c.Post(ctx, nodesBase+"mtls/reloadClient")
}

// UpdateNodePanels runs the panel self-updater on the given nodes. Offline or
// disabled nodes come back as skipped rather than failing the call.
func (c *Client) UpdateNodePanels(ctx context.Context, ids []int, dev bool) (*Response, error) {
	return c.PostJSON(ctx, nodesBase+"updatePanel", map[string]any{
		"ids": ids,
		"dev": dev,
	})
}
