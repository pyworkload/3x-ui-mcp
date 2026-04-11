package xui

import (
	"context"
	"fmt"
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
