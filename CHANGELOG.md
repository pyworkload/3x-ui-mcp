# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is 0, a new tool or a changed tool signature goes in a
minor release and fixes go in a patch release.

Two things a reader of this file usually wants:

- **Tool counts** are given per release, because a tool that exists in one
  version and not another is the most common source of confusion.
- **Panel compatibility** is called out whenever it moves. A 404 from a panel
  older than the endpoint is the most likely surprise this project produces; the
  table in the [README](README.md#panel-versions) is the per-tool reference.

## [Unreleased]

## [0.5.0] - 2026-09-21

Panel v3.8.5 support — **167 tools**, and the panel's API is now fully wrapped.

### Added

- `get_client_links` — the connection URLs for one client, keyed by email, one
  per inbound it is attached to. `get_subscription_links` cannot answer this: it
  is keyed by subId and returns the links of every client sharing it, so a
  client with no subId had no tool at all. This was the last panel route without
  one, which puts the tool count at **167** and the API fully wrapped.
- Fourteen parameters on `add_client` and `update_client` for client columns the
  panel already stored but no tool could set: `limit_hwid`, `reset_day`,
  `reset_max`, `traffic_reset`, `traffic_reset_day`, `reverse_tag`, `secret`,
  `ad_tag`, `private_key`, `public_key`, `pre_shared_key`, `allowed_ips`,
  `keep_alive` and `forwarded_ports`. The `xui://docs/client-fields` resource
  describes each one, keeping the detail out of both tool schemas.
- `make test-panel` — integration tests that run against a throwaway panel
  rather than a mock, behind the `panel` build tag so `make test` never reaches
  for one. They pin the read-modify-write contract by diffing a whole inbound
  and a whole client across an update, and pin the panel's own no-merge
  behaviour so the others cannot pass vacuously.
- Community health files: `CONTRIBUTING.md`, `SECURITY.md`,
  `CODE_OF_CONDUCT.md`, two issue forms and a pull request template.
- This changelog.

### Changed

- Inbound and client update bodies are built as maps with a blacklist instead of
  structs. The panel's update endpoints are full replaces and every release adds
  columns, so a struct silently dropped whatever it did not model. The current
  row is now read into a map and only the fields the panel recomputes per
  request are removed, which lets a newer panel's columns ride along untouched.

### Fixed

- `update_inbound` failed outright against 3x-ui v3.8.5. Since panel v3.3.1,
  `settings`, `streamSettings` and `sniffing` arrive as nested objects rather
  than the JSON-encoded strings older releases sent, and the old string fields
  could not parse them. Only reads ever broke, which is why this surfaced when a
  read had to feed a write.
- An `XUI_API_TOKEN` the panel refuses no longer fails the call when credentials
  are also configured. Panel v3.8.0 answers 401 where older releases masked the
  same case as a 404; the client now logs a warning naming the token and falls
  back to a session login, and reports a terminal error naming it when no
  credentials can back the retry.

### Security

- `.codex/`, which can hold panel credentials, is gitignored.

## [0.4.3] - 2026-09-02

### Fixed

- The CLI is published as `@pyworkload/3x-ui-mcp`. npm compares names with
  separators stripped and refused `3x-ui-mcp` as too similar to an unrelated
  `3xui-mcp`. The installed command is unchanged — `3x-ui-mcp` or `xui-mcp` —
  only the install line grows a scope.

## [0.4.2] - 2026-09-02

### Fixed

- The six platform packages are published under the `@pyworkload` scope. Six
  similar unscoped names arriving in a row read as name-squatting to npm's spam
  heuristics, which is why every project shipping per-platform binaries this way
  puts them under a scope.

## [0.4.1] - 2026-09-02

### Added

- An npm package, so `npx @pyworkload/3x-ui-mcp` runs the server without a Go
  toolchain. Platform binaries are published per OS and architecture and
  resolved as optional dependencies.
- Seven agent skills for panel operations, behind a dispatcher `SKILL.md`.

### Fixed

- The release pipeline now actually runs on a tag.

### Changed

- The README was restructured, its stale tool counts corrected, and the retired
  Go Report Card badge dropped.

## [0.4.0] - 2026-08-29

Panel v3.7.0 coverage — **166 tools**, up from 65.

### Added

- 31 tools for metrics, client groups, geodata, HWID devices and the slim
  inbound projections.
- 28 tools for host groups, subscription balancers, scoped API tokens and the
  Warp, NordVPN and PIA outbound providers.
- 16 tools for the multi-node API: CRUD, probing, mTLS and node panel updates.
- 26 tools finishing the panel API surface, including the maintenance calls.
- MCP resources and resource links, plus `XUI_TOOLSETS`, which registers only
  the named tool groups. All 166 schemas cost roughly 28k tokens of context in
  every session whether or not they are used, so narrowing them matters.
- An annotation preset on every tool — read-only, destructive, idempotent,
  open-world — so a client can tell what is safe to run unattended, and the
  panel conventions a client cannot read off the schemas are shipped as server
  instructions.

### Fixed

- The server starts without a panel configuration, so its tools stay listable
  when credentials are absent.

### Changed

- Linting moved to golangci-lint, and the CI actions moved off Node 20.

## [0.3.0] - 2026-08-04

**65 tools**, up from 46.

### Added

- 19 tools for balancers, subscriptions, key generation and inbounds.

### Changed

- **Requires 3x-ui v3.3.0 or newer.** That release moved the settings and Xray
  endpoints under `/panel/api/` and removed the old paths. Panels up to v3.2.8
  are served by [v0.2.0](https://github.com/pyworkload/3x-ui-mcp/releases/tag/v0.2.0).

### Fixed

- The `/panel/api/` prefix is used for the Xray and settings endpoints. Thanks
  to [@hermanmak](https://github.com/hermanmak) for the report and the fix.
- A response from the panel's SPA shell is reported as an error. On a path the
  panel does not route, a GET can answer **200 with `index.html`** rather than
  404, which used to surface as a successful API response.

## [0.2.0] - 2026-06-06

**46 tools**, up from 40.

### Added

- Support for 3x-ui v3.2.8: the CSRF middleware it added on `/panel`,
  `/panel/api` and `/login`, an optional `XUI_API_TOKEN` sent as a Bearer token,
  and the email-keyed client API that release introduced.

## [0.1.1] - 2026-04-19

### Fixed

- `update_inbound` silently replaced fields with zero values when they were
  omitted.

## [0.1.0] - 2026-04-11

### Added

- Initial release: **40 MCP tools** over the 3x-ui panel API, served over stdio.

[Unreleased]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.4.3...v0.5.0
[0.4.3]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/pyworkload/3x-ui-mcp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/pyworkload/3x-ui-mcp/releases/tag/v0.1.0
