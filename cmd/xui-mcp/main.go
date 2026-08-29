package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
	"github.com/pyworkload/3x-ui-mcp/internal/handler"
	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/server"
)

// Set via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// instructions ship with the server's capabilities: they carry the panel
// conventions a client cannot infer from the tool schemas alone.
const instructions = `Control a 3x-ui panel (Xray/V2Ray) over its HTTP API. Tool results are the panel's own JSON.

Conventions that are easy to get wrong:
- Clients are keyed by email, not UUID. add_client attaches a new client to inbound IDs; update_client reads the current record first and overlays only the fields you pass, so anything you omit — the UUID above all — survives.
- update_inbound and update_outbound_sub follow the same read-modify-write contract: pass only what changes.
- Routing rules are edited through the whole Xray template, so add/update/remove_routing_rule rewrite it; update_xray_template replaces it outright.
- A balancer override from set_balancer_override lives in the running core only — Xray forgets it on restart.
- "Subscription" means two different things: create/refresh/list_outbound_subs pull remote outbound lists into this panel, while get_subscription_links returns the links this panel serves to its own users.
- Client traffic limits are given in GB in tool parameters; the panel stores bytes.

Every tool is annotated with its effect. Read-only tools are safe to call while exploring; the ones marked destructive delete data, zero counters, or drop live connections.`

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel(),
	}))

	// An incomplete configuration is not fatal: the server still starts and
	// answers tools/list, so a client can inspect it — and a registry crawler
	// can index it — without a reachable panel. Every panel call then fails
	// with this same error.
	cfg, err := config.Load()
	if err != nil {
		logger.Warn("configuration incomplete; tools are listed but every panel call will fail", "error", err)
		fmt.Fprintf(os.Stderr, "Warning: %v\n\nRequired environment variables:\n  XUI_HOST      - Panel URL (e.g. http://localhost:2053)\n  XUI_USERNAME   - Admin username\n  XUI_PASSWORD   - Admin password\n\nOptional:\n  XUI_API_TOKEN  - Bearer API token, replaces username/password (3x-ui v3.2.8+)\n  XUI_BASE_PATH  - Panel base path (default: /)\n  XUI_TOOLSETS   - Comma-separated tool groups to expose (default: all)\n  XUI_LOG_LEVEL  - Log level: debug, info, warn, error (default: info)\n", err)
	}

	client := xui.NewClient(cfg, logger)

	s := server.NewMCPServer(
		"3x-ui",
		version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, false),
		server.WithInstructions(instructions),
	)

	if err := handler.RegisterAll(s, client, cfg.Toolsets); err != nil {
		logger.Error("toolset selection", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	enabled := cfg.Toolsets
	if len(enabled) == 0 {
		enabled = handler.ToolsetNames()
	}
	logger.Info("starting 3x-ui MCP server",
		"version", version,
		"commit", commit,
		"date", date,
		"host", cfg.Host,
		"toolsets", strings.Join(enabled, ","),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ServeStdio(s)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down gracefully")
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}
}

func logLevel() slog.Level {
	switch os.Getenv("XUI_LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
