package xui

import (
	"context"
	"fmt"
	"net/url"
)

// --- Host group API methods (panel v3.7.0+) ---
//
// A host group is a set of external addresses attached to one or more inbounds.
// The panel renders subscription links against them, so a group is how one
// inbound is published under a CDN hostname, a second port or a different SNI
// without touching the inbound itself. Groups are addressed by a string groupId,
// not a numeric one.

const hostsBase = "panel/api/hosts/"

// ListHosts returns every host group across all inbounds, in sort order.
func (c *Client) ListHosts(ctx context.Context) (*Response, error) {
	return c.Get(ctx, hostsBase+"list")
}

// GetHostGroup returns a single host group by its group ID.
func (c *Client) GetHostGroup(ctx context.Context, groupID string) (*Response, error) {
	return c.Get(ctx, hostsBase+"get/"+url.PathEscape(groupID))
}

// InboundHosts returns the host groups attached to one inbound.
func (c *Client) InboundHosts(ctx context.Context, inboundID int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf(hostsBase+"byInbound/%d", inboundID))
}

// HostTags returns the distinct tags used across all host groups.
func (c *Client) HostTags(ctx context.Context) (*Response, error) {
	return c.Get(ctx, hostsBase+"tags")
}

// AddHostGroup creates a host group. The body is the panel's HostGroup entity;
// inboundIds and remark are required by its validator.
func (c *Client) AddHostGroup(ctx context.Context, group map[string]any) (*Response, error) {
	return c.PostJSON(ctx, hostsBase+"add", group)
}

// UpdateHostGroup replaces a host group's contents. The panel takes the whole
// entity, so callers overlay their changes onto the stored group first.
func (c *Client) UpdateHostGroup(ctx context.Context, groupID string, group map[string]any) (*Response, error) {
	return c.PostJSON(ctx, hostsBase+"update/"+url.PathEscape(groupID), group)
}

// DeleteHostGroup removes a host group and the hosts inside it.
func (c *Client) DeleteHostGroup(ctx context.Context, groupID string) (*Response, error) {
	return c.Post(ctx, hostsBase+"del/"+url.PathEscape(groupID))
}

// SetHostGroupEnable enables or disables one host group.
func (c *Client) SetHostGroupEnable(ctx context.Context, groupID string, enable bool) (*Response, error) {
	return c.PostJSON(ctx, hostsBase+"setEnable/"+url.PathEscape(groupID), map[string]any{
		"enable": enable,
	})
}

// ReorderHostGroups sets sort order by the position of each groupId in the list.
func (c *Client) ReorderHostGroups(ctx context.Context, groupIDs []string) (*Response, error) {
	return c.PostJSON(ctx, hostsBase+"reorder", map[string]any{"ids": groupIDs})
}

// BulkDeleteHostGroups removes several host groups in one call.
func (c *Client) BulkDeleteHostGroups(ctx context.Context, groupIDs []string) (*Response, error) {
	return c.PostJSON(ctx, hostsBase+"bulk/del", map[string]any{"ids": groupIDs})
}

// BulkSetHostGroupsEnable flips several host groups at once.
func (c *Client) BulkSetHostGroupsEnable(ctx context.Context, groupIDs []string, enable bool) (*Response, error) {
	return c.PostJSON(ctx, hostsBase+"bulk/setEnable", map[string]any{
		"ids":    groupIDs,
		"enable": enable,
	})
}
