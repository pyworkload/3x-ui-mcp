# 3x-ui MCP Server

[![CI](https://github.com/pyworkload/3x-ui-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/pyworkload/3x-ui-mcp/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/pyworkload/3x-ui-mcp)](https://github.com/pyworkload/3x-ui-mcp/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/pyworkload/3x-ui-mcp/total.svg)](https://github.com/pyworkload/3x-ui-mcp/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pyworkload/3x-ui-mcp.svg)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/pyworkload/3x-ui-mcp.svg)](https://pkg.go.dev/github.com/pyworkload/3x-ui-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

MCP (Model Context Protocol) server for [3x-ui](https://github.com/MHSanaei/3x-ui) — an Xray/V2Ray proxy management panel. Exposes the 3x-ui HTTP API as MCP tools so LLMs can manage inbounds, clients, routing rules, Xray service, and server settings.

## Contents

- [Features](#features)
- [Quick start](#quick-start)
- [Example prompts](#example-prompts)
- [Agent skills](#agent-skills)
- [Configuration](#configuration)
- [Panel versions](#panel-versions)
- [MCP Tools](#mcp-tools)
- [Resources](#resources)
- [Architecture](#architecture)
- [Development](#development)
- [License](#license)

## Features

- **The whole panel API** — 166 tools across inbounds, clients, groups, hosts, nodes, routing, balancers, geodata, metrics, tokens and maintenance. Everything 3x-ui exposes except four file-transfer and node-sync routes (3x-ui v3.3.0+ — see [Panel versions](#panel-versions)).
- **Two auth modes** — session login with CSRF, or a Bearer API token (`XUI_API_TOKEN`), with transparent re-authentication and CSRF refresh when either goes stale.
- **Email-keyed clients** — one client attaches to several inbounds; bulk create, adjust, enable and delete, plus server-side paging over large panels.
- **Annotated tools** — every tool declares its effect, so an MCP client can tell a read-only call from a destructive one before running it.
- **Lazy by default** — bulky payloads answer with a summary plus a `resource_link`, fetched only when something actually needs the document.
- **Context-budgeted** — `XUI_TOOLSETS` loads only the tool groups you name, cutting what the schemas occupy in every session.
- **Agent skills included** — seven procedures for the real jobs, verified against the Xray-core and 3x-ui sources (see [Agent skills](#agent-skills)).
- **Stdio transport**, and zero external dependencies beyond the MCP SDK.

> [!IMPORTANT]
> These tools drive a live proxy panel. The ones annotated destructive delete
> clients, zero traffic counters, or restart Xray and drop every connection —
> several act panel-wide with no undo. Give an agent an API token scoped to what
> it should reach, and take a backup before bulk operations.

## Quick start

`npx` needs only Node 16+ — the package ships prebuilt binaries for Linux, macOS
and Windows on x64 and arm64, so nothing is compiled and no Go toolchain is
needed. With Go already installed, `go run …@latest` works just as well; see
[other ways to run it](#other-ways-to-run-it).

### Claude Code

```bash
claude mcp add 3x-ui \
  --env XUI_HOST=http://localhost:2053 \
  --env XUI_USERNAME=admin \
  --env XUI_PASSWORD=your-password \
  -- npx -y 3x-ui-mcp
```

### Claude Desktop / Cursor

Add to `claude_desktop_config.json` (Claude Desktop) or `.cursor/mcp.json` (Cursor):

```json
{
  "mcpServers": {
    "3x-ui": {
      "command": "npx",
      "args": ["-y", "3x-ui-mcp"],
      "env": {
        "XUI_HOST": "http://localhost:2053",
        "XUI_USERNAME": "admin",
        "XUI_PASSWORD": "your-password"
      }
    }
  }
}
```

### VS Code

Add to `.vscode/mcp.json`:

```json
{
  "servers": {
    "3x-ui": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "3x-ui-mcp"],
      "env": {
        "XUI_HOST": "http://localhost:2053",
        "XUI_USERNAME": "admin",
        "XUI_PASSWORD": "your-password"
      }
    }
  }
}
```

### Other ways to run it

Swap `command` and `args` in any of the configs above:

| How | `command` | `args` |
|---|---|---|
| Go toolchain | `go` | `["run", "github.com/pyworkload/3x-ui-mcp/cmd/xui-mcp@latest"]` |
| [Pre-built binary](https://github.com/pyworkload/3x-ui-mcp/releases) | `/path/to/xui-mcp` | — |
| Installed globally | `xui-mcp` | — |

`go run …@latest` compiles on first use and runs from cache afterwards.
`npm install -g 3x-ui-mcp` puts `xui-mcp` on the PATH if you would rather not go
through `npx` each time.

## Example prompts

Once connected, ask your agent things like:

- *"Create 20 trial clients on inbound 3 with 10 GB and a 7-day expiry, and give me their subscription links."*
- *"Which clients are past 90% of their quota? Extend them all by 30 days and 50 GB."*
- *"Scan 104.16.0.0/24 for usable REALITY targets and put the fastest one on my VLESS inbound."*
- *"Register a Warp account and route `*.netflix.com` through it."*
- *"Nobody can connect — check the Xray logs, the panel logs, and when each client was last online."*
- *"Which node is serving the most online clients right now, and how loaded is it?"*
- *"Back up the database to Telegram, then run the panel self-update on every node."*

## Agent skills

[`skills/`](skills/) holds seven procedures for the jobs people actually bring to a
panel — they encode operating knowledge the tool schemas cannot carry: what order
to do things in, and what the panel will not warn you about.

| Skill | Covers |
|---|---|
| [`reality-inbound`](skills/reality-inbound/SKILL.md) | Choosing and verifying a REALITY camouflage target, keys, shortIds, repairing a burnt inbound |
| [`diagnose-connectivity`](skills/diagnose-connectivity/SKILL.md) | Why a client cannot connect, cheapest cause first, with core error strings mapped to causes |
| [`panel-hardening`](skills/panel-hardening/SKILL.md) | Scoped tokens instead of admin passwords, rotation, fail2ban, cert pinning, backups |
| [`cdn-fallback`](skills/cdn-fallback/SKILL.md) | WebSocket/gRPC/xhttp behind a CDN, and sharing port 443 through fallbacks |
| [`client-lifecycle`](skills/client-lifecycle/SKILL.md) | Onboarding, subscription links, bulk renewals, reports, cleanup |
| [`smart-routing`](skills/smart-routing/SKILL.md) | Split routing by domain and geo token, provider outbounds, verifying rules with `test_route` |
| [`multi-node`](skills/multi-node/SKILL.md) | Adding nodes, TLS trust modes and mTLS, selective sync, cluster monitoring |

```bash
cp -r skills/* ~/.claude/skills/     # Claude Code, everywhere
cp -r skills/* .claude/skills/       # this project only
```

## Configuration

| Variable | Required | Description | Example |
|---|---|---|---|
| `XUI_HOST` | Yes | Panel URL | `http://localhost:2053` |
| `XUI_USERNAME` | Yes¹ | Admin username | `admin` |
| `XUI_PASSWORD` | Yes¹ | Admin password | `admin` |
| `XUI_API_TOKEN` | No | Bearer API token. Bypasses CSRF for `/panel/api/*` | `eyJ…` |
| `XUI_TOOLSETS` | No | Tool groups to expose (default: all) | `clients,metrics` |
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
  Create a token in the panel under **Settings → Security → API Token**.

  Since **v3.7.0** tokens are scoped and may carry an expiry, so give this server the
  **`admin`** scope. The other two do not fit it: `monitor` is a short allowlist of
  status and metrics routes (of the tools here only `server_status` and
  `get_xray_versions` fall inside it), and `node-sync` is a fixed allowlist for
  panel-to-node traffic. The plaintext token is shown once at creation — the panel
  keeps a SHA-256 hash. Also note that `x-ui setting -getApiToken` on the CLI now
  *rotates* a single shared `cli-fallback` token instead of minting a new one, so
  running it again invalidates a token you handed out earlier.

**Narrowing the toolset:** all 166 tool schemas together occupy roughly 28k tokens
of model context in every session — worth narrowing. `XUI_TOOLSETS` loads only
the groups you need: `inbounds`, `clients`, `server`, `xray`, `metrics`,
`groups`, `geodata`, `hosts`, `nodes`, `tokens`, `providers`, `maintenance`, or
`all`. `XUI_TOOLSETS=clients,metrics` leaves 44 tools at about 7k tokens. An
unknown name fails at startup rather than silently loading nothing.

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
| Observability, client groups, bulk client state, inbound projections and fallbacks | v3.5.0 |
| Host groups, API tokens, client export/import, external links, Warp and NordVPN | v3.5.0 |
| Multi-node tools | v3.5.0 |
| Maintenance, certificate helpers, cluster client views, `test_outbounds` | v3.5.0 |
| Geodata, HWID devices, `get_clients_by_telegram_id`, `set_inbound_sub_sort_index` | v3.7.0 |
| Subscription balancers, PIA, `reload_node_mtls_client`, and token `scope`/`expires_at` | v3.7.0 |
| `validate_regex`, `get_factory_defaults`, `get_panel_update_status`, `get_amneziawg_logs` | v3.7.0 |

The last two rows are bounded by the panel's own generated `openapi.json`, which
only starts at v3.5.0 — some of those routes may predate it.

Tested against **v3.7.0**, the current panel release: every tool here was run
against a live v3.7.0 panel.

## MCP Tools

Every tool is annotated with its effect on the panel — read-only, additive,
destructive, or service-interrupting — and with whether it reaches a host outside
the panel. MCP clients use those hints to auto-approve safe calls and to ask
before the rest.

### Inbound Management

<details>
<summary><b>15 tools</b> — CRUD, enable toggle, traffic reset, fallbacks, bulk delete, links</summary>

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
| `get_all_inbound_links` | Counts the connection URLs and links to the full list (`full=true` inlines them) |
| `list_inbounds_slim` | Inbounds with client arrays trimmed — the cheap listing for large panels |
| `get_inbound_options` | Picker projection: id, remark, tag, protocol, port, capability flags |
| `get_inbound_fallbacks` | Fallback rules on a master VLESS/Trojan TCP-TLS inbound |
| `set_inbound_fallbacks` | Replace the whole fallback list (restarts Xray) |
| `set_inbound_sub_sort_index` | Set an inbound's position in subscription output |

</details>

### Client Management

Clients are email-keyed entities that can be attached to several inbounds at once.

<details>
<summary><b>38 tools</b> — CRUD by email, attach/detach, bulk operations, traffic, devices, online state</summary>

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
| `get_clients_by_telegram_id` | Find clients by Telegram user ID |
| `list_client_devices` | HWID devices registered for a client |
| `delete_client_device` | Remove one registered device, freeing an HWID slot |
| `clear_client_devices` | Drop every registered device for a client |
| `bulk_enable_clients` | Enable many clients, one rewrite per owning inbound |
| `bulk_disable_clients` | Disable many clients, one rewrite per owning inbound |
| `bulk_adjust_clients` | Shift expiry and quota for many clients — the bulk renewal |
| `delete_orphan_clients` | Delete clients attached to no inbound, with their records |
| `export_clients` | Counts the clients and links to the full export (`full=true` inlines it) |
| `import_clients` | Create clients from an exported array; existing emails are skipped |
| `set_client_external_links` | Replace a client's external links and subscriptions |
| `list_clients_paged` | Server-side filtering, sorting and paging over clients |
| `bulk_attach_clients` | Attach many clients to many inbounds at once |
| `bulk_detach_clients` | Detach many clients from many inbounds at once |
| `get_active_inbounds` | Inbound tags that carried traffic, grouped by node |
| `get_online_clients_by_node` | Online clients grouped by the node serving them |
| `get_client_ips_by_node` | Per-client source IPs grouped by the node that saw them |

</details>

### Server Management

<details>
<summary><b>14 tools</b> — status, logs, Xray service control, key generation, REALITY scans</summary>

| Tool | Description |
|---|---|
| `server_status` | Get server system status (CPU, RAM, disk, uptime) |
| `restart_xray` | Restart Xray service |
| `stop_xray` | Stop Xray service |
| `get_xray_config` | Summarizes the running Xray config and links to the full JSON (`full=true` inlines it) |
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

</details>

### Xray Configuration

<details>
<summary><b>21 tools</b> — template, routing rules, outbounds, balancers, outbound subscriptions</summary>

| Tool | Description |
|---|---|
| `get_xray_template` | Summarizes the template and links to the full JSON (`full=true` inlines it) |
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

</details>

### Host Groups

A host group publishes one or more inbounds under external addresses — a CDN
hostname, a second port, a different SNI — and the panel renders subscription
links against them. Groups are addressed by a string `group_id`.

<details>
<summary><b>11 tools</b> — CRUD, ordering, bulk state</summary>

| Tool | Description |
|---|---|
| `list_hosts` | Every host group across all inbounds |
| `get_host_group` | One group by its ID |
| `get_inbound_hosts` | The groups attached to one inbound |
| `list_host_tags` | Distinct tags used across the groups |
| `add_host_group` | Create a group over a set of inbounds |
| `update_host_group` | Update a group, keeping the fields you omit |
| `delete_host_group` | Delete a group and its hosts |
| `set_host_group_enable` | Enable or disable one group |
| `reorder_host_groups` | Set the order groups appear in |
| `bulk_delete_host_groups` | Delete several groups at once |
| `bulk_set_host_groups_enable` | Flip several groups at once |

</details>

`add_host_group` and `update_host_group` expose the fields a group is normally
built from; `raw_json` sends a complete HostGroup for the rest (mux, sockopt,
ECH, final mask).

### Subscription Balancers

Client-side balancers: each appears in the JSON subscription of every client on
one of its inbounds, and the client app picks a server by the strategy. Not to
be confused with `get_balancers`, which reports the routing balancers running
inside Xray.

<details>
<summary><b>4 tools</b> — CRUD in sort order</summary>

| Tool | Description |
|---|---|
| `list_sub_balancers` | Every subscription balancer in sort order |
| `create_sub_balancer` | Add one over a set of inbounds |
| `update_sub_balancer` | Update one, keeping the fields you omit |
| `delete_sub_balancer` | Remove one |

</details>

### API Tokens

<details>
<summary><b>4 tools</b> — mint, disable, delete</summary>

| Tool | Description |
|---|---|
| `list_api_tokens` | Token metadata; the values themselves are never returned |
| `create_api_token` | Mint a scoped token — the plaintext is shown only once |
| `set_api_token_enabled` | Disable or re-enable a token without deleting it |
| `delete_api_token` | Delete a token permanently |

</details>

`delete_api_token` and `set_api_token_enabled` take the token's stored `scope`;
the panel fails closed unless it matches, so a token cannot be flipped by
guessing its ID.

### Outbound Providers

Cloudflare Warp, NordVPN and PIA each get a read tool and a write tool, so
querying a country list is annotated differently from erasing credentials.

<details>
<summary><b>6 tools</b> — a read and a write tool per provider</summary>

| Tool | Description |
|---|---|
| `get_warp_data` | Warp account state or the stored WireGuard config |
| `manage_warp` | Register, rotate the IP, apply a license, set rotation, or erase |
| `get_nordvpn_data` | Countries, servers in a country, or the stored account |
| `manage_nordvpn` | Register with a token, set a key, or erase |
| `get_pia_data` | Regions, servers in a region, or the stored account |
| `manage_pia` | Log in, register a key against a server, or erase |

</details>

### Panel Maintenance

<details>
<summary><b>19 tools</b> — health, certificates, geo files, self-update, backups, integration tests</summary>

| Tool | Description |
|---|---|
| `get_fail2ban_status` | Whether per-client IP limits can be enforced on this host |
| `get_web_cert_files` | This panel's own TLS certificate and key paths |
| `get_cert_hash` | SHA-256 of a certificate, for pinning |
| `get_remote_cert_hash` | SHA-256 of a remote server's live certificate |
| `generate_ech_cert` | An ECH keypair and config list for one SNI |
| `update_geofile` | Download fresh GeoIP/GeoSite data files |
| `get_panel_update_status` | How the last panel self-update ended |
| `set_update_channel` | Switch between the stable and dev channels |
| `update_panel` | Self-update the panel (it restarts) |
| `get_amneziawg_logs` | Live AmneziaWG peer activity and events |
| `get_node_descendants` | The nodes below this panel in the cluster tree |
| `get_client_ips_table` | The aggregated client-IP table behind IP limits |
| `backup_to_telegram` | Send a database backup to the admin Telegram chats |
| `test_smtp` | Test SMTP with stage-by-stage reporting |
| `test_telegram_bot` | Send a test message through the configured bot |
| `validate_regex` | Compile a pattern with the panel's Go RE2 engine |
| `get_default_settings` | Preview what a fresh install would compute |
| `get_factory_defaults` | The shipped default per setting key |
| `update_admin_credentials` | Change the panel admin username and password |

</details>

`update_admin_credentials` invalidates the credentials this server itself uses:
update `XUI_USERNAME` / `XUI_PASSWORD` and restart it afterwards, or switch to
an API token first.

### Multi-Node

A node is another 3x-ui panel this one drives: inbounds and clients are pushed
to it and its traffic is pulled back. Most of these calls reach the node itself,
not just this panel.

<details>
<summary><b>16 tools</b> — CRUD, probing, mTLS, node panel updates</summary>

| Tool | Description |
|---|---|
| `list_nodes` | Configured nodes with health, counts and last heartbeat |
| `get_node` | One node's connection details and sync state |
| `get_node_history` | CPU or memory history for one node |
| `get_node_web_cert` | The certificate and key paths that exist on the node |
| `test_node` | Probe connection details without saving them |
| `list_node_inbounds` | The inbounds available on a node, for selective sync |
| `get_node_cert_fingerprint` | SHA-256 of the node's leaf certificate, for pinning |
| `add_node` | Register a node (probed before it is saved) |
| `update_node` | Update a node, keeping the fields you omit |
| `set_node_enable` | Pause or resume traffic sync with a node |
| `probe_node` | Probe a registered node and refresh its cached health |
| `delete_node` | Delete a node; its inbounds are not migrated |
| `get_node_mtls_ca` | This panel's node-auth CA, minted on first call |
| `set_node_mtls_trust_ca` | The CA this panel trusts when acting as a node |
| `reload_node_mtls_client` | Apply a rotated client certificate without a restart |
| `update_node_panels` | Run the 3x-ui self-updater on the given nodes |

</details>

The node API token is write-only from v3.6.0: the panel reports only whether one
is set, so `update_node` omits it unless you pass a new one, and `clear_api_token`
is how a node moved to mTLS drops the old token.

### Client Groups

A group is a label carried both by the client record and by every owning
inbound's settings; the panel keeps the two in step in one transaction.

<details>
<summary><b>8 tools</b> — CRUD and bulk membership</summary>

| Tool | Description |
|---|---|
| `list_client_groups` | Every group with its member count, including empty ones |
| `get_client_group_emails` | Emails in one group — the input for a bulk action |
| `create_client_group` | Create an empty, selectable group |
| `rename_client_group` | Rename a group and repoint every member |
| `delete_client_group` | Drop a group, keeping its clients |
| `add_clients_to_group` | Label many clients with one group |
| `remove_clients_from_group` | Clear the group label on many clients |
| `reset_client_group_traffic` | Zero the group counter, leaving per-client counters running |

</details>

### Observability

<details>
<summary><b>6 tools</b> — metric history, observatory, update check</summary>

| Tool | Description |
|---|---|
| `get_metrics_history` | Time-series for one host metric (cpu, mem, netUp/Down, online, load) |
| `get_xray_metrics` | Xray runtime metrics state and current expvar values |
| `get_xray_metrics_history` | Time-series for one Xray runtime metric |
| `get_xray_observatory` | Latest per-outbound latency and health snapshot |
| `get_xray_observatory_history` | Probe results over time for one outbound tag |
| `get_panel_update_info` | Whether a newer 3x-ui release is out |

</details>

The history tools take a `bucket` in seconds and always return 60 samples, so
the bucket picks the window too: 2 (2m), 30 (30m), 60 (1h), 180 (3h), 360 (6h),
720 (12h), 1440 (24h), 2880 (2d), 10080 (7d).

### Geodata

<details>
<summary><b>4 tools</b> — browse geo databases, validate routing tokens</summary>

| Tool | Description |
|---|---|
| `list_geodata_files` | Geo databases in Xray's asset folder, with layout and category count |
| `list_geodata_categories` | Categories in one database, with entry counts and attributes |
| `list_geodata_entries` | The rules inside a category — typed domains or CIDRs |
| `validate_geodata_tokens` | Report routing tokens that do not resolve, with a reason |

</details>

**Deliberately not wrapped:** `server/getDb`, `server/getMigration` and
`server/importDB` move database files as attachments and multipart uploads
rather than JSON, so they belong in a browser or a backup script, not here.
`inbounds/pushClientTraffics` and `POST server/clientIps` are the inbound half
of node sync — endpoints a node receives on, not ones a client calls.

## Resources

Tools spend context on their results whether or not the caller needed all of it.
Resources are the lazy half: each costs one line in `resources/list` and is read
only when something asks for the URI. The three bulkiest tools answer with a
summary plus a `resource_link` to the matching resource — a running Xray config
is ~9k characters, and its summary is under 500.

| URI | Contents |
|---|---|
| `xui://docs/inbound-settings` | The settings, streamSettings and sniffing JSON, with an example per protocol |
| `xui://docs/client-fields` | What each client field means and the units it uses |
| `xui://xray/config` | The config the core is running right now |
| `xui://xray/template` | The saved template: routing, balancers, outbounds, DNS |
| `xui://inbounds` | Every inbound with settings and per-client stats |
| `xui://inbounds/links` | Connection URLs for every client |
| `xui://clients/export` | Every client as `{client, inboundIds}`, credentials included |
| `xui://inbound/{id}` | One inbound by numeric ID |
| `xui://client/{email}` | One client record by email |

The two `xui://docs/…` resources are static reference text, so reading them
costs no panel call.

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
