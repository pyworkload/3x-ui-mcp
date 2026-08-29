package xui

import (
	"context"
	"net/url"
)

// --- Outbound provider API methods (Warp v3.x, Nord v3.x, PIA v3.7.0+) ---
//
// Each provider hangs off a single route with the operation in the path and its
// arguments in the form body. The panel keeps the resulting credentials itself,
// so these tools register an account and hand the outbound to Xray rather than
// returning anything the caller has to store.

// WarpAction runs one Cloudflare Warp operation: data, config, reg, changeIp,
// license, interval or del.
func (c *Client) WarpAction(ctx context.Context, action string, params url.Values) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/warp/"+url.PathEscape(action), params)
}

// NordAction runs one NordVPN operation: countries, servers, reg, setKey, data
// or del.
func (c *Client) NordAction(ctx context.Context, action string, params url.Values) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/nord/"+url.PathEscape(action), params)
}

// PiaAction runs one PIA WireGuard operation: countries, servers, reg, addKey,
// data or del.
func (c *Client) PiaAction(ctx context.Context, action string, params url.Values) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/pia/"+url.PathEscape(action), params)
}
