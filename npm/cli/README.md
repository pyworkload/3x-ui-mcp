# @pyworkload/3x-ui-mcp

MCP (Model Context Protocol) server for [3x-ui](https://github.com/MHSanaei/3x-ui) —
an Xray/V2Ray proxy management panel. It exposes the panel's HTTP API as 166 MCP
tools, so an LLM agent can manage inbounds, clients, routing rules, balancers,
nodes and the Xray service itself.

This package ships prebuilt binaries: nothing is compiled and nothing is
downloaded at install time. The Go toolchain is not required.

## Use it

```json
{
  "mcpServers": {
    "3x-ui": {
      "command": "npx",
      "args": ["-y", "@pyworkload/3x-ui-mcp"],
      "env": {
        "XUI_HOST": "http://localhost:2053",
        "XUI_USERNAME": "admin",
        "XUI_PASSWORD": "your-password"
      }
    }
  }
}
```

For Claude Code:

```bash
claude mcp add 3x-ui \
  --env XUI_HOST=http://localhost:2053 \
  --env XUI_USERNAME=admin \
  --env XUI_PASSWORD=your-password \
  -- npx -y @pyworkload/3x-ui-mcp
```

Provide either `XUI_USERNAME` + `XUI_PASSWORD` or `XUI_API_TOKEN`. Optional:
`XUI_BASE_PATH`, `XUI_LOG_LEVEL`, and `XUI_TOOLSETS` to load only the tool groups
you need instead of all 166.

Requires 3x-ui **v3.3.0+**. Some tools need newer panels; the
[main README](https://github.com/pyworkload/3x-ui-mcp#panel-versions) has the table.

## Platforms

Linux, macOS and Windows on x64 and arm64. The binary is statically linked
(`CGO_ENABLED=0`), so the Linux build runs on glibc and musl alike.

The matching binary arrives as an optional dependency, one package per platform,
so an install pulls exactly one of them. Installing with `--omit=optional` skips
it and the CLI will say so rather than failing obscurely.

## Documentation

Full tool reference, agent skills and panel-version notes:
<https://github.com/pyworkload/3x-ui-mcp>

MIT licensed.
