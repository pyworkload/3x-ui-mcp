# Contributing

Thanks for taking the time. This server wraps a live proxy panel, so the bar for
a change is that it matches what the panel actually does — not what its
documentation says it does. Several traps below come from exactly that gap.

Participation is covered by the [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting set up

Requirements:

- **Go 1.23+** (the version CI builds with)
- **golangci-lint v2** — CI pins `v2.13.2`:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- **goreleaser** only if you touch the npm packaging (`make npm-pack`)

```bash
git clone https://github.com/pyworkload/3x-ui-mcp.git
cd 3x-ui-mcp
make build
make test
```

The test suite runs against `httptest` panels, so no 3x-ui instance is needed to
build or test. You do need one to verify behaviour against a real panel — see
[Verifying against a panel](#verifying-against-a-panel).

## Before you open a PR

```bash
make fmt     # gofmt -s -w .
make test    # go test -v -race ./...
make lint    # golangci-lint run ./...
```

CI runs the same lint and test jobs, then cross-builds for linux/darwin/windows
on amd64 and arm64. All three have to be green before a PR merges.

Branch off `main` (`feat/short-name`, `fix/short-name`, `docs/short-name`) and
open the PR against `main`. Commit subjects follow Conventional Commits —
`feat:`, `fix:`, `docs:`, `chore:` — and say what changed for a user of the
server, not which files moved.

## Adding a tool

The code is three layers, and a new tool touches each of them:

1. **API method** — `internal/xui/<area>.go`. Go through `client.Get`,
   `PostJSON`, `PostForm` or `Post`; they all funnel into `do()`, which is where
   login, CSRF refresh, the 403 retry and the 404/redirect re-login live. A
   hand-rolled `http.Do` skips all of it.
2. **Tool definition and handler** — `internal/handler/<area>.go`. The first
   option to `mcp.NewTool` is an annotation preset from `helpers.go`
   (`readsPanel`, `probesRemote`, `writesPanel`, `updatesPanel`,
   `destroysPanel`, `interruptsService`, `fetchesRemote`, `updatesRemote`,
   `managesRemote`, `installsRemote`). Pick the one that matches the effect
   rather than spelling out the hints — clients use them to decide what may run
   unattended. Return API-level failures with `mcp.NewToolResultError`, not a Go
   error, so the model sees the panel's message.
3. **Registration** — add it to its group's `register*Tools` function. The groups
   themselves are the `toolsets` map in `register.go`, which is what
   `XUI_TOOLSETS` selects from.
4. **Tests** — cover the handler logic that is not a straight pass-through:
   overlays, validation, parameter shaping.
5. **Docs** — the tool table for its section in `README.md`, the tool count in
   the Features list, and the file map in `CLAUDE.md` if you added a file.

**Context budget is a design constraint.** Every registered schema costs tokens
in every session — 166 tools are roughly 28k. Prefer one tool with an optional
parameter over two near-duplicates, keep parameter lists to what a caller
realistically sets, and for a tool that answers with a large document, return a
summary plus a `resource_link` with `full=true` as the opt-in (see
`get_xray_config`).

## Panel conventions that are easy to get wrong

- **Updates are read-modify-write.** `update_client`, `update_inbound`,
  `update_node`, `update_host_group` and `update_outbound_sub` all read the
  current record and overlay only the supplied fields. The panel replaces the
  row wholesale, so an omitted field that is not carried over is silently
  destroyed — a client's UUID, a group's host list. A new update tool follows
  the same contract, and gets a test that proves omitted fields survive.
- **Not every write takes JSON.** The subscription balancer routes and the
  Warp/NordVPN/PIA provider routes read `url.Values` form fields; sending JSON
  loses them silently with no error. The panel's `openapi.json` shows JSON
  examples for both and is wrong.
- **Double-encoded JSON.** `settings`, `streamSettings` and `sniffing` arrive as
  JSON strings inside JSON. Use `unmarshalFlexible()` / `extractXraySetting()`.
- **A 200 is not proof of a route.** On a path the panel does not route, a GET
  can hit the SPA shell and return 200 with `index.html`. `rawDo()` detects
  `text/html` and reports it as a missing route; don't work around it.
- **Endpoints drift by panel version.** The baseline is v3.3.0, but balancer
  status and `routeTest` need v3.3.1, Reality scanning and `inbounds/allLinks`
  need v3.4.2. If a tool needs something newer, add the row to the panel-version
  table in the README.

## Tests

Table-driven, named after the behaviour they pin down, with a comment saying why
that behaviour matters — `TestHostGroupBody_OverlaysOnlySuppliedFields` is the
shape to copy. Panels are stubbed with `httptest`; `errcheck` is relaxed inside
`_test.go` so a stub can encode a response inline.

## Verifying against a panel

Point the server at a panel with:

```
XUI_HOST=http://localhost:2053
XUI_USERNAME=admin
XUI_PASSWORD=admin
# or XUI_API_TOKEN=… with the admin scope
```

If a change was checked against a live panel, put the panel version in the PR
description. Where the panel and its `openapi.json` disagree, the panel wins —
and that is worth a comment in the code, because the next reader will check the
spec first.

## Reporting a bug

Open an issue with:

- the **3x-ui panel version** and the **server version** (logged at startup)
- the tool you called and the parameters
- what the panel returned — the raw JSON is the useful part
- `XUI_LOG_LEVEL=debug` output if the failure is in the HTTP layer

Please redact panel hostnames, tokens, client UUIDs and emails before pasting.

## Security

Do not open a public issue for a vulnerability in this server — see
[SECURITY.md](SECURITY.md) for what is in scope and how to report it privately.

## License

By contributing you agree that your contribution is licensed under the
[MIT License](LICENSE) that covers this project.
