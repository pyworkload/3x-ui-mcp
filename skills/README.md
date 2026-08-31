# Agent skills

Seven procedures for operating a 3x-ui panel through this MCP server. They are not
a second copy of the tool documentation — the tool schemas, the server's
`instructions` string and the `xui://docs/*` resources already tell an agent what
each parameter means. These carry the part that is nowhere in a schema: which order
to do things in, what the panel will not warn you about, and which knobs are
load-bearing.

Where a rule comes from Xray-core or the panel's own behaviour, the skill says so
and quotes the error you will actually see in the logs.

| Skill | Covers |
|---|---|
| [`reality-inbound`](reality-inbound/SKILL.md) | Choosing and verifying a REALITY camouflage target, keys, shortIds, and repairing an inbound that stopped getting through |
| [`diagnose-connectivity`](diagnose-connectivity/SKILL.md) | Why a client cannot connect, cheapest cause first, with the core's error strings mapped to causes |
| [`panel-hardening`](panel-hardening/SKILL.md) | Scoped API tokens instead of admin passwords, credential rotation, fail2ban, certificate pinning, backups |
| [`cdn-fallback`](cdn-fallback/SKILL.md) | WebSocket/gRPC/xhttp behind a CDN, and sharing port 443 through Xray fallbacks |
| [`client-lifecycle`](client-lifecycle/SKILL.md) | Onboarding, subscription links, bulk renewals, usage reports, cleanup |
| [`smart-routing`](smart-routing/SKILL.md) | Split routing by domain and geo token, provider outbounds, verifying rules with the router before trusting them |
| [`multi-node`](multi-node/SKILL.md) | Adding nodes, TLS trust modes and mTLS, selective inbound sync, cluster monitoring |

## Using them

**Claude Code** — copy the ones you want into your skills directory:

```bash
cp -r skills/* ~/.claude/skills/          # available everywhere
cp -r skills/* .claude/skills/            # this project only
```

They are plain Markdown with YAML frontmatter, so any agent runtime that reads the
skill format can use them; anything else can read them as documentation.

Each skill's `description` says when it applies, which is what the agent matches
against — you invoke them by describing the task ("this client can't connect",
"set up a REALITY inbound"), not by naming the file.

## Panel versions

The skills note version requirements where they matter. The baseline is 3x-ui
**v3.3.0+**, the same as the server; REALITY scanning needs **v3.4.2**, node and
token tools **v3.5.0**, and a few maintenance calls **v3.7.0**. The table in the
main [README](../README.md#panel-versions) is the reference.
