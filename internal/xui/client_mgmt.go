package xui

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// --- Client management API methods (3x-ui v3.2.8 ClientController, /panel/api/clients/*) ---
//
// Clients are first-class, email-keyed entities that can be attached to several
// inbounds at once. All addressing is by email.

const clientsBase = "panel/api/clients/"

// joinInts renders []int as a comma-separated string for query params.
func joinInts(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

// AddClient creates a client and attaches it to the inbounds in the payload.
func (c *Client) AddClient(ctx context.Context, payload ClientCreatePayload) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"add", payload)
}

// BulkCreateClients creates multiple clients in one call.
func (c *Client) BulkCreateClients(ctx context.Context, payloads []ClientCreatePayload) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"bulkCreate", payloads)
}

// UpdateClient updates a client identified by email. inboundIds, when non-empty,
// restricts the update to those attachments (the panel's ?inboundIds filter).
func (c *Client) UpdateClient(ctx context.Context, email string, client ClientConfig, inboundIds []int) (*Response, error) {
	path := clientsBase + "update/" + url.PathEscape(email)
	if len(inboundIds) > 0 {
		path += "?inboundIds=" + url.QueryEscape(joinInts(inboundIds))
	}
	return c.PostJSON(ctx, path, client)
}

// DeleteClient removes a client by email. keepTraffic preserves its traffic stats.
func (c *Client) DeleteClient(ctx context.Context, email string, keepTraffic bool) (*Response, error) {
	path := clientsBase + "del/" + url.PathEscape(email)
	if keepTraffic {
		path += "?keepTraffic=1"
	}
	return c.Post(ctx, path)
}

// BulkDeleteClients removes multiple clients by email.
func (c *Client) BulkDeleteClients(ctx context.Context, emails []string, keepTraffic bool) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"bulkDel", map[string]any{
		"emails":      emails,
		"keepTraffic": keepTraffic,
	})
}

// GetClient returns a single client (with its inbound attachments) by email.
func (c *Client) GetClient(ctx context.Context, email string) (*Response, error) {
	return c.Get(ctx, clientsBase+"get/"+url.PathEscape(email))
}

// ListClientsPaged returns a filtered, paginated client list.
// query may carry page, pageSize, search, filter, protocol, inbound, group, sort, order.
func (c *Client) ListClientsPaged(ctx context.Context, query url.Values) (*Response, error) {
	path := clientsBase + "list/paged"
	if enc := query.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.Get(ctx, path)
}

// AttachClient attaches an existing client to additional inbounds.
func (c *Client) AttachClient(ctx context.Context, email string, inboundIds []int) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+url.PathEscape(email)+"/attach", map[string]any{
		"inboundIds": inboundIds,
	})
}

// DetachClient detaches a client from the given inbounds.
func (c *Client) DetachClient(ctx context.Context, email string, inboundIds []int) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+url.PathEscape(email)+"/detach", map[string]any{
		"inboundIds": inboundIds,
	})
}

// GetClientTraffic returns traffic stats for a client by email.
func (c *Client) GetClientTraffic(ctx context.Context, email string) (*Response, error) {
	return c.Get(ctx, clientsBase+"traffic/"+url.PathEscape(email))
}

// GetClientIPs returns IP addresses recorded for a client.
func (c *Client) GetClientIPs(ctx context.Context, email string) (*Response, error) {
	return c.Post(ctx, clientsBase+"ips/"+url.PathEscape(email))
}

// ClearClientIPs clears recorded IP addresses for a client.
func (c *Client) ClearClientIPs(ctx context.Context, email string) (*Response, error) {
	return c.Post(ctx, clientsBase+"clearIps/"+url.PathEscape(email))
}

// ResetClientTraffic resets traffic counters for a single client by email.
func (c *Client) ResetClientTraffic(ctx context.Context, email string) (*Response, error) {
	return c.Post(ctx, clientsBase+"resetTraffic/"+url.PathEscape(email))
}

// ResetAllClientTraffics resets traffic counters for every client (panel-wide).
func (c *Client) ResetAllClientTraffics(ctx context.Context) (*Response, error) {
	return c.Post(ctx, clientsBase+"resetAllTraffics")
}

// BulkResetTraffic resets traffic counters for the given client emails.
func (c *Client) BulkResetTraffic(ctx context.Context, emails []string) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"bulkResetTraffic", map[string]any{
		"emails": emails,
	})
}

// DeleteDepletedClients removes clients that exhausted their traffic/time (panel-wide).
func (c *Client) DeleteDepletedClients(ctx context.Context) (*Response, error) {
	return c.Post(ctx, clientsBase+"delDepleted")
}

// GetOnlineClients returns currently connected clients.
func (c *Client) GetOnlineClients(ctx context.Context) (*Response, error) {
	return c.Post(ctx, clientsBase+"onlines")
}

// GetLastOnline returns last-online timestamps for all clients.
func (c *Client) GetLastOnline(ctx context.Context) (*Response, error) {
	return c.Post(ctx, clientsBase+"lastOnline")
}

// UpdateClientTraffic sets specific upload/download byte values for a client.
func (c *Client) UpdateClientTraffic(ctx context.Context, email string, upload, download int64) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"updateTraffic/"+url.PathEscape(email), map[string]int64{
		"upload":   upload,
		"download": download,
	})
}
