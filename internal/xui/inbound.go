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

// --- Inbound projections and fallbacks (panel v3.6.0+/v3.7.0+) ---

// ListInboundsSlim is List with the client arrays stripped to {email, enable,
// comment} and no UUID/subId enrichment — the cheap call for a wide panel.
func (c *Client) ListInboundsSlim(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/inbounds/list/slim")
}

// InboundOptions returns a picker projection of the inbounds: id, remark, tag,
// protocol, port and the server-computed capability flags.
func (c *Client) InboundOptions(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/inbounds/options")
}

// GetInboundFallbacks lists the fallback rules on a master VLESS/Trojan
// TCP-TLS inbound, each linking a child inbound to SNI/ALPN/path conditions.
func (c *Client) GetInboundFallbacks(ctx context.Context, id int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf("panel/api/inbounds/%d/fallbacks", id))
}

// SetInboundFallbacks replaces the whole fallback list for a master inbound
// and restarts Xray. Unlike the update endpoints this is not read-modify-write:
// whatever is passed becomes the complete set.
func (c *Client) SetInboundFallbacks(ctx context.Context, id int, fallbacks any) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf("panel/api/inbounds/%d/fallbacks", id), map[string]any{
		"fallbacks": fallbacks,
	})
}

// SetInboundSubSortIndex sets only the subscription sort order, reading the
// stored inbound first so a reorder cannot revive a stale client list.
func (c *Client) SetInboundSubSortIndex(ctx context.Context, id, index int) (*Response, error) {
	return c.PostJSON(ctx, fmt.Sprintf("panel/api/inbounds/%d/subSortIndex", id), map[string]any{
		"subSortIndex": index,
	})
}
