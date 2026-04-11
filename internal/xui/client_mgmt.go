package xui

import (
	"context"
	"fmt"
	"net/url"
)

// --- Client management API methods ---

// AddClient adds a client to an inbound. data must contain "id" (inbound ID)
// and "settings" (JSON string with clients array).
func (c *Client) AddClient(ctx context.Context, data map[string]any) (*Response, error) {
	return c.PostJSON(ctx, "panel/api/inbounds/addClient", data)
}

// UpdateClient updates a client within an inbound.
// clientID is the UUID/ID of the client to update.
func (c *Client) UpdateClient(ctx context.Context, clientID string, data map[string]any) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf("panel/api/inbounds/updateClient/%s", clientID), data)
}

// DeleteClient removes a client from an inbound by inbound ID and client UUID.
func (c *Client) DeleteClient(ctx context.Context, inboundID int, clientID string) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/%d/delClient/%s", inboundID, clientID))
}

// DeleteClientByEmail removes a client from an inbound by email.
func (c *Client) DeleteClientByEmail(ctx context.Context, inboundID int, email string) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/%d/delClientByEmail/%s", inboundID, email))
}

// GetClientTraffic returns traffic stats for a client by email.
func (c *Client) GetClientTraffic(ctx context.Context, email string) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf("panel/api/inbounds/getClientTraffics/%s", email))
}

// GetClientTrafficByID returns traffic stats for a client by UUID.
func (c *Client) GetClientTrafficByID(ctx context.Context, id string) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf("panel/api/inbounds/getClientTrafficsById/%s", id))
}

// GetClientIPs returns IP addresses recorded for a client.
func (c *Client) GetClientIPs(ctx context.Context, email string) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/clientIps/%s", email))
}

// ClearClientIPs clears recorded IP addresses for a client.
func (c *Client) ClearClientIPs(ctx context.Context, email string) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/clearClientIps/%s", email))
}

// ResetClientTraffic resets traffic counters for a client within an inbound.
func (c *Client) ResetClientTraffic(ctx context.Context, inboundID int, email string) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/%d/resetClientTraffic/%s", inboundID, email))
}

// ResetAllTraffics resets all inbound traffic counters.
func (c *Client) ResetAllTraffics(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/inbounds/resetAllTraffics")
}

// ResetAllClientTraffics resets traffic for all clients in an inbound.
func (c *Client) ResetAllClientTraffics(ctx context.Context, inboundID int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/resetAllClientTraffics/%d", inboundID))
}

// DeleteDepletedClients removes clients that exhausted their traffic/time.
func (c *Client) DeleteDepletedClients(ctx context.Context, inboundID int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/delDepletedClients/%d", inboundID))
}

// GetOnlineClients returns currently connected clients.
func (c *Client) GetOnlineClients(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/inbounds/onlines")
}

// UpdateClientTraffic sets specific traffic values for a client.
func (c *Client) UpdateClientTraffic(ctx context.Context, email string, upload, download int64) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf("panel/api/inbounds/updateClientTraffic/%s", email), map[string]int64{
		"upload":   upload,
		"download": download,
	})
}

// GetLastOnline returns last online timestamps for all clients.
func (c *Client) GetLastOnline(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/inbounds/lastOnline")
}

// SearchInbounds searches inbounds (via query, not a standard API — reserved for extension).
func (c *Client) SearchInbounds(ctx context.Context, query string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/inbounds/search", url.Values{"query": {query}})
}
