package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel(),
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n\nRequired environment variables:\n  XUI_HOST      - Panel URL (e.g. http://localhost:2053)\n  XUI_USERNAME   - Admin username\n  XUI_PASSWORD   - Admin password\n\nOptional:\n  XUI_BASE_PATH  - Panel base path (default: /)\n  XUI_LOG_LEVEL  - Log level: debug, info, warn, error (default: info)\n", err)
		os.Exit(1)
	}

	client := xui.NewClient(cfg, logger)

	s := server.NewMCPServer(
		"3x-ui",
		version,
		server.WithToolCapabilities(true),
	)

	handler.RegisterAll(s, client)

	logger.Info("starting 3x-ui MCP server",
		"version", version,
		"commit", commit,
		"date", date,
		"host", cfg.Host,
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
