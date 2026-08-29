package xui

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// --- Xray settings & routing API methods ---

// GetXrayTemplate returns the current Xray template config, inbound tags, and test URL.
// Response obj contains: xraySetting (JSON string), inboundTags ([]string), outboundTestUrl (string).
func (c *Client) GetXrayTemplate(ctx context.Context) (*Response, error) {
	return c.Post(ctx, "panel/api/xray/")
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
	return c.PostForm(ctx, "panel/api/xray/update", data)
}

// GetOutboundsTraffic returns traffic statistics for all outbounds.
func (c *Client) GetOutboundsTraffic(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/xray/getOutboundsTraffic")
}

// ResetOutboundTraffic resets traffic counters for a specific outbound tag.
func (c *Client) ResetOutboundTraffic(ctx context.Context, tag string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/resetOutboundsTraffic", url.Values{
		"tag": {tag},
	})
}

// TestOutbound tests an outbound configuration for connectivity.
func (c *Client) TestOutbound(ctx context.Context, outbound string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/testOutbound", url.Values{
		"outbound": {outbound},
	})
}

// GetXrayResult returns the current Xray service operational status.
func (c *Client) GetXrayResult(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/xray/getXrayResult")
}

// --- Balancer & routing diagnostics (3x-ui v3.3.1+) ---

// GetBalancersStatus returns the live state of the given balancer tags in the
// running core: the current override and the outbounds the strategy prefers.
// Tags the core doesn't know (xray stopped, balancer saved but not applied)
// come back with running=false rather than failing the call.
func (c *Client) GetBalancersStatus(ctx context.Context, tags []string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/balancerStatus", url.Values{
		"tags": {strings.Join(tags, ",")},
	})
}

// OverrideBalancer pins a balancer to a single outbound in the running core.
// An empty target clears the override. The change applies live without a
// restart — and is lost when Xray restarts.
func (c *Client) OverrideBalancer(ctx context.Context, tag, target string) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/balancerOverride", url.Values{
		"tag":    {tag},
		"target": {target},
	})
}

// --- Outbound subscriptions (3x-ui v3.4.0+) ---

// OutboundSubParams carries the form fields an outbound subscription is created
// and updated with. The panel rewrites every field on update — an omitted
// remark or url is stored as empty — so callers must send a complete set.
type OutboundSubParams struct {
	Remark         string
	URL            string
	TagPrefix      string
	Enabled        bool
	UpdateInterval int
	AllowPrivate   bool
	AllowInsecure  bool
	Prepend        bool
}

func (p OutboundSubParams) values() url.Values {
	data := url.Values{
		"remark":        {p.Remark},
		"url":           {p.URL},
		"tagPrefix":     {p.TagPrefix},
		"enabled":       {strconv.FormatBool(p.Enabled)},
		"allowPrivate":  {strconv.FormatBool(p.AllowPrivate)},
		"allowInsecure": {strconv.FormatBool(p.AllowInsecure)},
		"prepend":       {strconv.FormatBool(p.Prepend)},
	}
	if p.UpdateInterval > 0 {
		data.Set("updateInterval", strconv.Itoa(p.UpdateInterval))
	}
	return data
}

// ListOutboundSubs returns all outbound subscriptions, newest first.
func (c *Client) ListOutboundSubs(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/xray/outbound-subs")
}

// CreateOutboundSub registers a remote outbound list. The panel fetches the
// URL, parses it into outbounds with stable tags, and merges them additively
// into the running config.
func (c *Client) CreateOutboundSub(ctx context.Context, params OutboundSubParams) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/outbound-subs", params.values())
}

// UpdateOutboundSub overwrites an existing subscription with the given fields.
func (c *Client) UpdateOutboundSub(ctx context.Context, id int, params OutboundSubParams) (*Response, error) {
	return c.PostForm(ctx, fmt.Sprintf("panel/api/xray/outbound-subs/%d", id), params.values())
}

// DeleteOutboundSub removes a subscription and its outbounds. Uses the POST
// alias so the request goes through the same path as every other write here.
func (c *Client) DeleteOutboundSub(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/xray/outbound-subs/%d/del", id))
}

// RefreshOutboundSub re-fetches a subscription now and returns the parsed
// outbounds, instead of waiting for its update interval.
func (c *Client) RefreshOutboundSub(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("panel/api/xray/outbound-subs/%d/refresh", id))
}

// MoveOutboundSub reorders a subscription one step, which changes where its
// outbounds land in the merged config.
func (c *Client) MoveOutboundSub(ctx context.Context, id int, up bool) (*Response, error) {
	dir := "down"
	if up {
		dir = "up"
	}
	return c.PostForm(ctx, fmt.Sprintf("panel/api/xray/outbound-subs/%d/move", id), url.Values{
		"dir": {dir},
	})
}

// ParseOutboundSubURL fetches and parses a subscription URL without keeping it.
func (c *Client) ParseOutboundSubURL(ctx context.Context, rawURL string, allowPrivate, allowInsecure bool) (*Response, error) {
	return c.PostForm(ctx, "panel/api/xray/outbound-subs/parse", url.Values{
		"url":           {rawURL},
		"allowPrivate":  {strconv.FormatBool(allowPrivate)},
		"allowInsecure": {strconv.FormatBool(allowInsecure)},
	})
}

// RouteTestRequest describes the synthetic connection to route. Domain or IP
// is required; the rest narrow the match. No traffic is sent.
type RouteTestRequest struct {
	InboundTag string
	Domain     string
	IP         string
	Port       int
	Network    string
	Protocol   string
	Email      string
}

// TestRoute asks the running core which outbound its router would pick for the
// described connection.
func (c *Client) TestRoute(ctx context.Context, req RouteTestRequest) (*Response, error) {
	data := url.Values{}
	for key, value := range map[string]string{
		"inboundTag": req.InboundTag,
		"domain":     req.Domain,
		"ip":         req.IP,
		"network":    req.Network,
		"protocol":   req.Protocol,
		"email":      req.Email,
	} {
		if value != "" {
			data.Set(key, value)
		}
	}
	if req.Port > 0 {
		data.Set("port", strconv.Itoa(req.Port))
	}
	return c.PostForm(ctx, "panel/api/xray/routeTest", data)
}

// TestOutbounds probes a batch of outbounds (max 50) through one shared temp
// Xray instance, returning results in input order. mode picks the probe depth:
// "tcp" dials only, "real" measures a cold full request, anything else routes a
// real HTTP request through each outbound.
func (c *Client) TestOutbounds(ctx context.Context, outboundsJSON, allOutboundsJSON, mode string) (*Response, error) {
	form := url.Values{"outbounds": {outboundsJSON}}
	if allOutboundsJSON != "" {
		form.Set("allOutbounds", allOutboundsJSON)
	}
	if mode != "" {
		form.Set("mode", mode)
	}
	return c.PostForm(ctx, "panel/api/xray/testOutbounds", form)
}
