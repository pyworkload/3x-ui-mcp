# 3x-ui MCP Server

[![CI](https://github.com/pyworkload/3x-ui-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/pyworkload/3x-ui-mcp/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/pyworkload/3x-ui-mcp)](https://github.com/pyworkload/3x-ui-mcp/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

MCP (Model Context Protocol) server for [3x-ui](https://github.com/MHSanaei/3x-ui) — an Xray/V2Ray proxy management panel. Exposes the 3x-ui HTTP API as MCP tools so LLMs can manage inbounds, clients, routing rules, Xray service, and server settings.

## Features

- 65 MCP tools covering the full 3x-ui API (3x-ui v3.3.0+ — see [Panel versions](#panel-versions))
- Two auth modes: session login with CSRF, or a Bearer API token (`XUI_API_TOKEN`)
- Automatic session management with transparent re-authentication and CSRF refresh
- Email-keyed client model: attach/detach across inbounds, bulk operations, paged listing
- Stdio transport for seamless LLM integration
- Zero external dependencies beyond the MCP SDK

## Usage

Add to your MCP config (`claude_desktop_config.json` or `.mcp.json`):

```json
{
  "mcpServers": {
    "3x-ui": {
      "command": "go",
      "args": ["run", "github.com/pyworkload/3x-ui-mcp/cmd/xui-mcp@latest"],
      "env": {
        "XUI_HOST": "http://localhost:2053",
        "XUI_USERNAME": "admin",
        "XUI_PASSWORD": "your-password"
      }
    }
  }
}
```

Requires [Go 1.23+](https://go.dev/dl/). First run downloads and compiles automatically, subsequent runs use cache.

### With pre-built binary

Download from [Releases](https://github.com/pyworkload/3x-ui-mcp/releases), then:

```json
{
  "mcpServers": {
    "3x-ui": {
      "command": "/path/to/xui-mcp",
      "env": {
        "XUI_HOST": "http://localhost:2053",
        "XUI_USERNAME": "admin",
        "XUI_PASSWORD": "your-password"
      }
    }
  }
}
```

## Configuration

| Variable | Required | Description | Example |
|---|---|---|---|
| `XUI_HOST` | Yes | Panel URL | `http://localhost:2053` |
| `XUI_USERNAME` | Yes¹ | Admin username | `admin` |
| `XUI_PASSWORD` | Yes¹ | Admin password | `admin` |
| `XUI_API_TOKEN` | No | Bearer API token. Bypasses CSRF for `/panel/api/*` | `eyJ…` |
| `XUI_BASE_PATH` | No | Panel base path (default: `/`) | `/xui/` |
| `XUI_LOG_LEVEL` | No | Log level (default: `info`) | `debug`, `info`, `warn`, `error` |

¹ Provide **either** `XUI_USERNAME`+`XUI_PASSWORD` **or** `XUI_API_TOKEN`.

**Auth modes (3x-ui protects the panel with CSRF and supports API tokens):**

- **Session + CSRF** (username/password): works for every tool. The client logs in,
  tracks the session CSRF token, and replays it on write requests, refreshing
  automatically when it goes stale.
- **Bearer API token** (`XUI_API_TOKEN`): the token is sent on every request and
  the panel accepts it for `/panel/api/*` routes without CSRF — on v3.3.0+ that
  covers every tool here, settings and Xray templates included.
  Create a token in the panel under **Settings → API Tokens**.

## Panel versions

Requires **3x-ui v3.3.0 or newer**. That release moved the settings and Xray
endpoints from `/panel/setting/*` and `/panel/xray/*` under `/panel/api/`
(upstream [`c6f15cd5`](https://github.com/MHSanaei/3x-ui/commit/c6f15cd5), a
documented breaking change), and this server targets the current layout.

On an older panel the inbound, client and server tools still work, but anything
touching settings or the Xray template answers HTTP 404. For panels up to
v3.2.8, use release [v0.2.0](https://github.com/pyworkload/3x-ui-mcp/releases/tag/v0.2.0)
instead.

A few tools wrap endpoints that landed after v3.3.0 and answer 404 below the
version listed:

| Tool | Needs |
|---|---|
| `get_balancers`, `set_balancer_override`, `test_route` | v3.3.1 |
| `scan_reality_target`, `scan_reality_targets`, `get_all_inbound_links` | v3.4.2 |

## MCP Tools

### Inbound Management (10 tools)

| Tool | Description |
|---|---|
| `list_inbounds` | List all inbound connections |
| `get_inbound` | Get inbound by ID |
| `create_inbound` | Create a new inbound |
| `update_inbound` | Update an existing inbound |
| `delete_inbound` | Delete an inbound |
| `set_inbound_enable` | Enable/disable an inbound without rewriting its settings |
| `reset_inbound_traffic` | Zero one inbound's traffic counters |
| `delete_all_inbound_clients` | Remove every client from an inbound, keeping the inbound |
| `bulk_delete_inbounds` | Delete several inbounds in one call |
| `get_all_inbound_links` | Connection URLs for every client across every inbound |

### Client Management (21 tools)

Clients are email-keyed entities that can be attached to several inbounds at once.

| Tool | Description |
|---|---|
| `add_client` | Create a client and attach it to one or more inbounds (`inbound_ids`) |
| `update_client` | Update a client by email (only supplied fields change; UUID preserved) |
| `delete_client` | Delete a client by email (optional `keep_traffic`) |
| `get_client` | Get a client's full config and its inbound attachments, by email |
| `list_clients` | Paged, searchable, filterable client list |
| `attach_client` | Attach an existing client to more inbounds |
| `detach_client` | Detach a client from given inbounds |
| `bulk_create_clients` | Create many clients across the same inbounds |
| `bulk_delete_clients` | Delete many clients by email |
| `get_client_traffic` | Get client traffic stats by email |
| `get_client_ips` | Get IPs used by a client |
| `clear_client_ips` | Clear recorded client IPs |
| `reset_client_traffic` | Reset traffic counters for a client by email |
| `reset_all_traffics` | Reset all inbound traffic counters |
| `reset_all_client_traffics` | Reset traffic for every client (panel-wide) |
| `bulk_reset_traffic` | Reset traffic for a specific set of clients |
| `delete_depleted_clients` | Delete clients with exhausted traffic/expired (panel-wide) |
| `get_online_clients` | List currently connected clients |
| `get_last_online` | Last-online timestamp for every client |
| `update_client_traffic` | Set specific upload/download byte counters for a client |
| `get_subscription_links` | Connection URLs served under a subscription ID, as JSON |

### Server Management (14 tools)

| Tool | Description |
|---|---|
| `server_status` | Get server system status (CPU, RAM, disk, uptime) |
| `restart_xray` | Restart Xray service |
| `stop_xray` | Stop Xray service |
| `get_xray_config` | Get current Xray runtime configuration |
| `get_xray_versions` | List available Xray versions |
| `install_xray` | Install a specific Xray version |
| `get_logs` | Get panel service logs |
| `get_xray_logs` | Get Xray core logs |
| `get_settings` | Get panel settings |
| `get_default_xray_config` | Get default Xray configuration |
| `restart_panel` | Restart the 3x-ui panel |
| `generate_key` | Generate key material: UUID, X25519 (Reality), VLESS encryption, ML-KEM-768, ML-DSA-65 |
| `scan_reality_target` | Probe one REALITY candidate and report whether it is usable |
| `scan_reality_targets` | Probe domains, IPs or CIDR ranges, ranked by feasibility and latency |

### Xray Configuration (20 tools)

| Tool | Description |
|---|---|
| `get_xray_template` | Get Xray JSON template |
| `update_xray_template` | Update Xray JSON template |
| `get_routing_rules` | List all routing rules (and the balancers they reference) |
| `add_routing_rule` | Add a routing rule |
| `remove_routing_rule` | Remove a routing rule by index |
| `update_routing_rule` | Update a routing rule by index |
| `get_outbounds` | List all outbounds |
| `get_outbounds_traffic` | Get outbound traffic statistics |
| `reset_outbound_traffic` | Reset traffic for an outbound tag |
| `test_outbound` | Test connectivity of an outbound |
| `get_balancers` | Balancer definitions plus their live state in the running core |
| `set_balancer_override` | Pin a balancer to one outbound, or release it back to its strategy |
| `test_route` | Ask the core which outbound it would pick for a synthetic connection |
| `list_outbound_subs` | List outbound subscriptions — remote URLs supplying extra outbounds |
| `preview_outbound_sub` | Parse a subscription URL without saving it |
| `create_outbound_sub` | Add an outbound subscription |
| `update_outbound_sub` | Update a subscription (only supplied fields change) |
| `refresh_outbound_sub` | Re-fetch a subscription now and return its outbounds |
| `move_outbound_sub` | Move a subscription up/down in priority |
| `delete_outbound_sub` | Delete an outbound subscription |

## Architecture

```
cmd/xui-mcp/main.go        Entry point, config loading, signal handling
internal/config/            Configuration from environment variables
internal/xui/              HTTP client with session management
internal/handler/          MCP tool definitions and request handlers
```

## Development

```bash
make test       # Run tests
make lint       # Run golangci-lint (config in .golangci.yml)
make fmt        # Format code
make build      # Build binary
```

## License

 MIT
