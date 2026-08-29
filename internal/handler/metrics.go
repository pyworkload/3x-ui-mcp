package handler

import (
	"context"
	"fmt"
	"slices"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// metricsHandler holds the XUI client for observability tool handlers.
type metricsHandler struct {
	client *xui.Client
}

// historyBuckets are the sample sizes the panel accepts, in seconds. Every
// history route answers with 60 samples, so the bucket also fixes the window it
// covers: 2s→2m, 30s→30m, 60s→1h, 180s→3h, 360s→6h, 720s→12h, 1440s→24h,
// 2880s→2d, 10080s→7d. Anything else comes back as "invalid bucket", so the
// tools check it first and name the list. (The published openapi.json documents
// a different set; these are the values panel/api actually enforces.)
var historyBuckets = []int{2, 30, 60, 180, 360, 720, 1440, 2880, 10080}

// hostMetrics and xrayMetrics are the series each history route serves; they
// are advertised as enums so a wrong name never reaches the panel.
var (
	hostMetrics = []string{"cpu", "mem", "netUp", "netDown", "online", "load1", "load5", "load15"}
	xrayMetrics = []string{"xrAlloc", "xrSys", "xrHeapObjects", "xrNumGC", "xrPauseNs"}
)

func checkBucket(bucket int) error {
	if slices.Contains(historyBuckets, bucket) {
		return nil
	}
	return fmt.Errorf("bucket must be one of %v seconds, got %d", historyBuckets, bucket)
}

func registerMetricsTools(s *server.MCPServer, client *xui.Client) {
	h := &metricsHandler{client: client}

	s.AddTool(mcp.NewTool("get_metrics_history",
		readsPanel,
		mcp.WithDescription("Time-series for one host metric as {t, v} samples over a window from 2 minutes to 7 days. Use it to see load over time instead of the single instant server_status reports."),
		mcp.WithString("metric",
			mcp.Required(),
			mcp.Description("Metric to chart"),
			mcp.Enum(hostMetrics...),
		),
		mcp.WithNumber("bucket",
			mcp.Description("Seconds per sample. The panel always returns 60 samples, so this also picks the window: 2 (2m), 30 (30m), 60 (1h), 180 (3h), 360 (6h), 720 (12h), 1440 (24h), 2880 (2d), 10080 (7d)."),
			mcp.DefaultNumber(360),
		),
	), h.metricsHistory)

	s.AddTool(mcp.NewTool("get_xray_metrics",
		readsPanel,
		mcp.WithDescription("Xray runtime metrics state: whether the Xray config has a metrics block, which expvar keys are flowing, and their current values. Returns an empty state when metrics are not configured."),
	), h.xrayMetrics)

	s.AddTool(mcp.NewTool("get_xray_metrics_history",
		readsPanel,
		mcp.WithDescription("Time-series for one Xray runtime metric (memory, GC and heap counters) over a window from 2 minutes to 7 days."),
		mcp.WithString("metric",
			mcp.Required(),
			mcp.Description("Xray runtime metric to chart"),
			mcp.Enum(xrayMetrics...),
		),
		mcp.WithNumber("bucket",
			mcp.Description("Seconds per sample across 60 samples: 2 (2m), 30 (30m), 60 (1h), 180 (3h), 360 (6h), 720 (12h), 1440 (24h), 2880 (2d), 10080 (7d)"),
			mcp.DefaultNumber(360),
		),
	), h.xrayMetricsHistory)

	s.AddTool(mcp.NewTool("get_xray_observatory",
		readsPanel,
		mcp.WithDescription("Latest Xray observatory snapshot: per-outbound latency, health status and last probe time. Populated only when the Xray config has an observatory configured."),
	), h.observatory)

	s.AddTool(mcp.NewTool("get_xray_observatory_history",
		readsPanel,
		mcp.WithDescription("Observatory probe results over time for one outbound tag — the history behind a balancer's current pick."),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("Outbound tag from the observatory config"),
		),
		mcp.WithNumber("bucket",
			mcp.Description("Seconds per sample across 60 samples: 2 (2m), 30 (30m), 60 (1h), 180 (3h), 360 (6h), 720 (12h), 1440 (24h), 2880 (2d), 10080 (7d)"),
			mcp.DefaultNumber(360),
		),
	), h.observatoryHistory)

	s.AddTool(mcp.NewTool("get_panel_update_info",
		probesRemote,
		mcp.WithDescription("Check GitHub for a newer 3x-ui release than the one running."),
	), h.updateInfo)
}

func (h *metricsHandler) metricsHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	metric, err := req.RequireString("metric")
	if err != nil {
		return mcp.NewToolResultError("metric is required"), nil
	}
	bucket := int(req.GetFloat("bucket", 360))
	if err := checkBucket(bucket); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.MetricsHistory(ctx, metric, bucket))
}

func (h *metricsHandler) xrayMetrics(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.XrayMetricsState(ctx))
}

func (h *metricsHandler) xrayMetricsHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	metric, err := req.RequireString("metric")
	if err != nil {
		return mcp.NewToolResultError("metric is required"), nil
	}
	bucket := int(req.GetFloat("bucket", 360))
	if err := checkBucket(bucket); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.XrayMetricsHistory(ctx, metric, bucket))
}

func (h *metricsHandler) observatory(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.XrayObservatory(ctx))
}

func (h *metricsHandler) observatoryHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := req.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError("tag is required"), nil
	}
	bucket := int(req.GetFloat("bucket", 360))
	if err := checkBucket(bucket); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toResult(h.client.XrayObservatoryHistory(ctx, tag, bucket))
}

func (h *metricsHandler) updateInfo(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return toResult(h.client.PanelUpdateInfo(ctx))
}
