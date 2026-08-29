package xui

import (
	"context"
	"fmt"
	"net/url"
)

// --- Observability API methods (panel v3.6.0+) ---
//
// The history endpoints all answer with the same {t, v} sample shape and take
// the same bucket list, so they share one caller-facing contract: pick a
// metric, pick seconds per sample, get 60 samples back. Because the count is
// fixed, the bucket picks the window too — 360s of bucket is six hours of
// history. Allowed buckets live in the handler layer, which rejects the rest
// before the panel does.

// MetricsHistory returns an aggregated time-series for one host metric.
// metric is one of cpu, mem, netUp, netDown, online, load1, load5, load15.
func (c *Client) MetricsHistory(ctx context.Context, metric string, bucketSeconds int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf("panel/api/server/history/%s/%d", url.PathEscape(metric), bucketSeconds))
}

// XrayMetricsState reports whether the Xray config exposes a metrics block,
// which expvar keys are flowing, and their current values.
func (c *Client) XrayMetricsState(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/xrayMetricsState")
}

// XrayMetricsHistory returns the time-series for one Xray runtime metric.
// metric is one of xrAlloc, xrSys, xrHeapObjects, xrNumGC, xrPauseNs.
func (c *Client) XrayMetricsHistory(ctx context.Context, metric string, bucketSeconds int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf("panel/api/server/xrayMetricsHistory/%s/%d", url.PathEscape(metric), bucketSeconds))
}

// XrayObservatory returns the latest observatory snapshot: per-outbound
// latency, health and last probe time. Empty unless Xray has an observatory.
func (c *Client) XrayObservatory(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/xrayObservatory")
}

// XrayObservatoryHistory returns probe results over time for one outbound tag.
func (c *Client) XrayObservatoryHistory(ctx context.Context, tag string, bucketSeconds int) (*Response, error) {
	return c.Get(ctx, fmt.Sprintf("panel/api/server/xrayObservatoryHistory/%s/%d", url.PathEscape(tag), bucketSeconds))
}

// PanelUpdateInfo checks GitHub for a newer 3x-ui release.
func (c *Client) PanelUpdateInfo(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "panel/api/server/getPanelUpdateInfo")
}
