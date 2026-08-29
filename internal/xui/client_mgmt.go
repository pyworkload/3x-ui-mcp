package xui

import (
	"context"
	"fmt"
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

// GetSubscriptionLinks returns the connection URLs for every enabled client
// carrying the given subscription ID — the same set the public /sub/<subId>
// endpoint serves, as a plain JSON array instead of base64.
func (c *Client) GetSubscriptionLinks(ctx context.Context, subID string) (*Response, error) {
	return c.Get(ctx, clientsBase+"subLinks/"+url.PathEscape(subID))
}

// UpdateClientTraffic sets specific upload/download byte values for a client.
func (c *Client) UpdateClientTraffic(ctx context.Context, email string, upload, download int64) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"updateTraffic/"+url.PathEscape(email), map[string]int64{
		"upload":   upload,
		"download": download,
	})
}

// --- Client device and bulk-state methods (panel v3.6.0+/v3.7.0+) ---

// GetClientsByTelegramID looks up clients by Telegram user ID. Several clients
// can share one ID, so the panel answers with an array.
func (c *Client) GetClientsByTelegramID(ctx context.Context, tgID int64) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf(clientsBase+"get/tgId/%d", tgID))
}

// ListClientDevices returns the HWID devices registered for a client. The
// hashes themselves are never exposed, only the metadata around them.
func (c *Client) ListClientDevices(ctx context.Context, email string) (*Response, error) {
	return c.Post(ctx, clientsBase+"hwids/"+url.PathEscape(email))
}

// ClearClientDevices drops every registered device for a client, freeing all
// slots under its HWID limit.
func (c *Client) ClearClientDevices(ctx context.Context, email string) (*Response, error) {
	return c.Delete(ctx, clientsBase+"hwids/"+url.PathEscape(email))
}

// DeleteClientDevice removes one registered device by the id from the list.
func (c *Client) DeleteClientDevice(ctx context.Context, email string, deviceID int) (*Response, error) {
	return c.Delete(ctx, fmt.Sprintf(clientsBase+"hwids/%s/%d", url.PathEscape(email), deviceID))
}

// BulkEnableClients enables many clients, one read-modify-write per inbound.
func (c *Client) BulkEnableClients(ctx context.Context, emails []string) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"bulkEnable", map[string]any{"emails": emails})
}

// BulkDisableClients disables many clients, one read-modify-write per inbound.
func (c *Client) BulkDisableClients(ctx context.Context, emails []string) (*Response, error) {
	return c.PostJSON(ctx, clientsBase+"bulkDisable", map[string]any{"emails": emails})
}

// DeleteOrphanClients deletes every client attached to no inbound, along with
// its traffic record, IP log, devices and external links.
func (c *Client) DeleteOrphanClients(ctx context.Context) (*Response, error) {
	return c.Post(ctx, clientsBase+"delOrphans")
}

// BulkAdjustClients shifts expiry and quota for many clients at once; both
// deltas may be negative. Clients on unlimited expiry or unlimited traffic are
// reported back as skipped rather than converted to a limit. An empty flow
// leaves each client's flow alone.
func (c *Client) BulkAdjustClients(ctx context.Context, emails []string, addDays int, addBytes int64, flow string) (*Response, error) {
	payload := map[string]any{
		"emails":   emails,
		"addDays":  addDays,
		"addBytes": addBytes,
	}
	if flow != "" {
		payload["flow"] = flow
	}
	return c.PostJSON(ctx, clientsBase+"bulkAdjust", payload)
}
