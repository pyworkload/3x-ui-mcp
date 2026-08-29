package xui

import (
	"context"
	"net/url"
	"strconv"
)

// --- Geodata API methods (panel v3.7.0+) ---
//
// These read the geosite/geoip .dat files in Xray's asset folder, so a routing
// rule can be written against categories that actually exist on this server
// rather than against a remembered list.

const geodataBase = "panel/api/xray/geodata"

// GeodataFiles lists the geo databases on disk with their detected layout,
// size, modification time and category count.
func (c *Client) GeodataFiles(ctx context.Context) (*Response, error) {
	return c.Get(ctx, geodataBase+"/files")
}

// GeodataCategories returns one page of a database's categories. An empty
// limit returns every category, which the panel allows because the index is
// small; entries are the large side.
func (c *Client) GeodataCategories(ctx context.Context, file, query string, offset, limit int) (*Response, error) {
	params := url.Values{"file": {file}}
	if query != "" {
		params.Set("q", query)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.Get(ctx, geodataBase+"/categories?"+params.Encode())
}

// GeodataEntries returns one page of the rules inside a category: typed domain
// rules for geosite databases, CIDRs for geoip ones.
func (c *Client) GeodataEntries(ctx context.Context, file, code, query string, offset, limit int) (*Response, error) {
	params := url.Values{"file": {file}, "code": {code}}
	if query != "" {
		params.Set("q", query)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.Get(ctx, geodataBase+"/entries?"+params.Encode())
}

// ValidateGeodataTokens checks routing tokens against the databases on disk and
// returns only the ones that do not resolve. Unlike the other geodata routes
// this one takes form data, and kind selects the token grammar: "domain" for
// geosite/ext tokens, "ip" for geoip ones.
func (c *Client) ValidateGeodataTokens(ctx context.Context, kind, tokens string) (*Response, error) {
	return c.PostForm(ctx, geodataBase+"/validate", url.Values{
		"kind":   {kind},
		"tokens": {tokens},
	})
}
