package xui

import (
	"context"
	"net/url"
)

// --- Client group API methods (panel v3.6.0+) ---
//
// A group is a label on a client, stored both in the clients table and inside
// every owning inbound's settings JSON; the panel keeps the two in sync in one
// transaction. Empty groups exist as placeholder rows, so listing groups is not
// the same as listing the labels currently in use.

const groupsBase = clientsBase + "groups"

// ListClientGroups returns every group with its member count.
func (c *Client) ListClientGroups(ctx context.Context) (*Response, error) {
	return c.Get(ctx, groupsBase)
}

// ClientGroupEmails returns the emails of the clients in one group.
func (c *Client) ClientGroupEmails(ctx context.Context, name string) (*Response, error) {
	return c.Get(ctx, groupsBase+"/"+url.PathEscape(name)+"/emails")
}

// CreateClientGroup creates an empty group, selectable before it has members.
func (c *Client) CreateClientGroup(ctx context.Context, name string) (*Response, error) {
	return c.PostJSON(ctx, groupsBase+"/create", map[string]any{"name": name})
}

// RenameClientGroup renames a group and repoints every member at the new name.
func (c *Client) RenameClientGroup(ctx context.Context, oldName, newName string) (*Response, error) {
	return c.PostJSON(ctx, groupsBase+"/rename", map[string]any{
		"oldName": oldName,
		"newName": newName,
	})
}

// DeleteClientGroup drops the group and clears the label from its members.
// The clients themselves are kept.
func (c *Client) DeleteClientGroup(ctx context.Context, name string) (*Response, error) {
	return c.PostJSON(ctx, groupsBase+"/delete", map[string]any{"name": name})
}

// AddClientsToGroup labels many clients with one group in a single call.
func (c *Client) AddClientsToGroup(ctx context.Context, emails []string, group string) (*Response, error) {
	return c.PostJSON(ctx, groupsBase+"/bulkAdd", map[string]any{
		"emails": emails,
		"group":  group,
	})
}

// RemoveClientsFromGroup clears the group label on many clients.
func (c *Client) RemoveClientsFromGroup(ctx context.Context, emails []string) (*Response, error) {
	return c.PostJSON(ctx, groupsBase+"/bulkRemove", map[string]any{"emails": emails})
}

// ResetClientGroupTraffic zeroes the group-level counter by snapshotting the
// members' current totals as a baseline; per-client counters are untouched.
func (c *Client) ResetClientGroupTraffic(ctx context.Context, name string) (*Response, error) {
	return c.PostJSON(ctx, groupsBase+"/resetTraffic", map[string]any{"name": name})
}
