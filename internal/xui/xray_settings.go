package xui

import (
	"context"
	"net/url"
)

// --- Xray settings & routing API methods ---

// GetXrayTemplate returns the current Xray template config, inbound tags, and test URL.
// Response obj contains: xraySetting (JSON string), inboundTags ([]string), outboundTestUrl (string).
func (c *Client) GetXrayTemplate(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/xray/")
}

// UpdateXrayTemplate saves a new Xray template configuration.
// xraySetting is the full Xray config as a JSON string.
func (c *Client) UpdateXrayTemplate(ctx context.Context, xraySetting string, outboundTestUrl string) (*Response, error) {
	data := url.Values{
		"xraySetting": {xraySetting},
	}
	if outboundTestUrl != "" {
		data.Set("outboundTestUrl", outboundTestUrl)
	}
	return c.PostForm(ctx, "panel/xray/update", data)
}

// GetOutboundsTraffic returns traffic statistics for all outbounds.
func (c *Client) GetOutboundsTraffic(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/xray/getOutboundsTraffic")
}

// ResetOutboundTraffic resets traffic counters for a specific outbound tag.
func (c *Client) ResetOutboundTraffic(ctx context.Context, tag string) (*Response, error) {
	return c.PostForm(ctx, "panel/xray/resetOutboundsTraffic", url.Values{
		"tag": {tag},
	})
}

// TestOutbound tests an outbound configuration for connectivity.
func (c *Client) TestOutbound(ctx context.Context, outbound string) (*Response, error) {
	return c.PostForm(ctx, "panel/xray/testOutbound", url.Values{
		"outbound": {outbound},
	})
}

// GetXrayResult returns the current Xray service operational status.
func (c *Client) GetXrayResult(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/xray/getXrayResult")
}
