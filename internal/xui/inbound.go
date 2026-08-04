package xui

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// --- Inbound API methods ---

// ListInbounds returns all inbounds configured on the panel.
func (c *Client) ListInbounds(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/inbounds/list")
}

// GetInbound returns a single inbound by ID.
func (c *Client) GetInbound(ctx context.Context, id int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf("panel/api/inbounds/get/%d", id))
}

// CreateInbound creates a new inbound. data should contain all inbound fields.
func (c *Client) CreateInbound(ctx context.Context, data map[string]any) (*Response, error) {
	return c.PostJSON(ctx, "panel/api/inbounds/add", data)
}

// UpdateInbound updates an existing inbound by ID.
func (c *Client) UpdateInbound(ctx context.Context, id int, data map[string]any) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf("panel/api/inbounds/update/%d", id), data)
}

// DeleteInbound removes an inbound by ID.
func (c *Client) DeleteInbound(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/del/%d", id))
}

// ImportInbound imports an inbound from JSON data (form field "data").
func (c *Client) ImportInbound(ctx context.Context, jsonData string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/inbounds/import", map[string][]string{
		"data": {jsonData},
	})
}

// ResetAllTraffics resets traffic counters for all inbounds.
func (c *Client) ResetAllTraffics(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/inbounds/resetAllTraffics")
}

// SetInboundEnable toggles just the enable flag of an inbound, without
// rewriting its settings JSON.
func (c *Client) SetInboundEnable(ctx context.Context, id int, enable bool) (*Response, error) {
	return c.PostForm(ctx, fmt.Sprintf("panel/api/inbounds/setEnable/%d", id), url.Values{
		"enable": {strconv.FormatBool(enable)},
	})
}

// ResetInboundTraffic zeroes the upload/download counters of one inbound.
// Per-client counters are left alone.
func (c *Client) ResetInboundTraffic(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/%d/resetTraffic", id))
}

// DeleteAllInboundClients removes every client attached to an inbound while
// keeping the inbound itself.
func (c *Client) DeleteAllInboundClients(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/inbounds/%d/delAllClients", id))
}

// BulkDeleteInbounds deletes several inbounds in one call. The panel processes
// the list sequentially, reports failures per id, and restarts Xray at most once.
func (c *Client) BulkDeleteInbounds(ctx context.Context, ids []int) (*Response, error) {
	return c.PostJSON(ctx, "panel/api/inbounds/bulkDel", map[string]any{"ids": ids})
}

// GetAllInboundLinks returns every protocol URL across all inbounds and their
// clients, rendered through the panel's subscription engine.
func (c *Client) GetAllInboundLinks(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/inbounds/allLinks")
}
