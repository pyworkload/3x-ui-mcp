# Security Policy

## Supported versions

Fixes go into the latest release. There are no maintained backport branches, so
please reproduce on the newest version before reporting.

| Version | Supported |
| ------- | --------- |
| latest release | yes |
| older releases | no — upgrade first |

Note that the panel side has its own baseline: this server targets 3x-ui
**v3.3.0+**, and release **v0.2.0** is the last one serving panels up to v3.2.8.
A problem that only reproduces on a panel below the supported baseline is a
compatibility issue, not a vulnerability.

## Reporting a vulnerability

**Do not open a public issue.** Report privately through GitHub's
[security advisories](https://github.com/pyworkload/3x-ui-mcp/security/advisories/new),
which keeps the report and the discussion between you and the maintainers until
a fix ships.

Useful in a report:

- the version of this server and of the 3x-ui panel
- whether the setup uses session login or `XUI_API_TOKEN`
- the steps or the tool call that triggers it, and what an attacker gains
- a proof of concept if you have one

Expect an acknowledgement within a few days. This is a volunteer project with no
bug bounty; credit in the advisory and the release notes is offered unless you
would rather stay anonymous.

## What is in scope

This server is a local stdio process that holds credentials to a live proxy
panel, so the interesting classes are:

- leaking `XUI_PASSWORD` / `XUI_API_TOKEN`, session cookies, or client UUIDs —
  into logs, tool results, or error messages
- sending them anywhere other than the configured `XUI_HOST`
- a tool that reaches a host or performs a write outside what its parameters and
  its annotation preset describe
- a response from a panel or a remote provider that can steer the calling model
  into an unintended action

## What is not

- **Vulnerabilities in 3x-ui itself.** Report those to
  [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui/security). This server only
  calls the panel's own HTTP API.
- **The destructive tools doing what they say.** `delete_*`, `reset_*`,
  `restart_xray` and the bulk operations delete data, zero counters and drop
  live connections by design; several act panel-wide with no undo. That is why
  they carry destructive annotations, and why an agent should be given an API
  token scoped to what it may reach.
- **A panel exposed without TLS.** Use `https://` in `XUI_HOST`; over plain HTTP
  the credentials are on the wire and no change here can fix that.
