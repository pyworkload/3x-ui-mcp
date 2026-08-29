package xui

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// --- Subscription balancer API methods (panel v3.7.0+) ---
//
// A subscription balancer is a client-side construct: it appears in the JSON
// subscription of every client sitting on at least one of its inbounds, and the
// client app picks a server by the given strategy. This is unrelated to the
// Xray routing balancers behind get_balancers, which run inside the core.
//
// Unlike most /panel/api routes, these take form data with repeated inboundIds
// keys rather than JSON.

const subBalancersBase = "panel/api/sub-balancers"

// ListSubBalancers returns every subscription balancer in sort order.
func (c *Client) ListSubBalancers(ctx context.Context) (*Response, error) {
	return c.Get(ctx, subBalancersBase)
}

// CreateSubBalancer adds a subscription balancer over the given inbounds.
func (c *Client) CreateSubBalancer(ctx context.Context, remark, strategy string, inboundIDs []int, sortOrder int, enabled *bool) (*Response, error) {
	return c.PostForm(ctx, subBalancersBase, subBalancerForm(remark, strategy, inboundIDs, sortOrder, enabled))
}

// UpdateSubBalancer replaces a balancer's row. The panel keeps the stored
// enabled flag when the key is absent, so a nil enabled leaves it alone.
func (c *Client) UpdateSubBalancer(ctx context.Context, id int, remark, strategy string, inboundIDs []int, sortOrder int, enabled *bool) (*Response, error) {
	return c.PostForm(ctx, fmt.Sprintf("%s/%d", subBalancersBase, id), subBalancerForm(remark, strategy, inboundIDs, sortOrder, enabled))
}

// DeleteSubBalancer removes a balancer by id. The POST alias is used rather
// than DELETE, since both routes exist and POST needs no method negotiation.
func (c *Client) DeleteSubBalancer(ctx context.Context, id int) (*Response, error) {
	return c.Post(ctx, fmt.Sprintf("%s/%d/del", subBalancersBase, id))
}

func subBalancerForm(remark, strategy string, inboundIDs []int, sortOrder int, enabled *bool) url.Values {
	form := url.Values{
		"remark":    {remark},
		"strategy":  {strategy},
		"sortOrder": {strconv.Itoa(sortOrder)},
	}
	for _, id := range inboundIDs {
		form.Add("inboundIds", strconv.Itoa(id))
	}
	if enabled != nil {
		form.Set("enabled", strconv.FormatBool(*enabled))
	}
	return form
}
