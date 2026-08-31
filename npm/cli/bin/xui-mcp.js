#!/usr/bin/env node
// Locates the prebuilt xui-mcp binary for this platform and hands the process
// over to it. The binary ships in a per-platform package pulled in through
// optionalDependencies, so nothing is downloaded or compiled at install time.
'use strict';

const { spawnSync } = require('node:child_process');
const path = require('node:path');

// process.platform + process.arch -> the package carrying that build.
const PACKAGES = {
  'darwin arm64': '3x-ui-mcp-darwin-arm64',
  'darwin x64': '3x-ui-mcp-darwin-x64',
  'linux arm64': '3x-ui-mcp-linux-arm64',
  'linux x64': '3x-ui-mcp-linux-x64',
  'win32 arm64': '3x-ui-mcp-win32-arm64',
  'win32 x64': '3x-ui-mcp-win32-x64',
};

function fail(message) {
  // stdout carries the MCP protocol, so diagnostics go to stderr only.
  process.stderr.write(`3x-ui-mcp: ${message}\n`);
  process.exit(1);
}

function resolveBinary() {
  const target = `${process.platform} ${process.arch}`;
  const pkg = PACKAGES[target];
  if (!pkg) {
    fail(
      `no prebuilt binary for ${target}. Supported: ${Object.keys(PACKAGES).join(', ')}. ` +
        'Build from source instead: go install github.com/pyworkload/3x-ui-mcp/cmd/xui-mcp@latest',
    );
  }

  const exe = process.platform === 'win32' ? 'xui-mcp.exe' : 'xui-mcp';
  try {
    // Resolve through package.json: the binary itself is not a module, and this
    // works no matter how the installer laid the tree out (nested or hoisted).
    return path.join(path.dirname(require.resolve(`${pkg}/package.json`)), 'bin', exe);
  } catch {
    fail(
      `the platform package ${pkg} is missing. It installs automatically as an optional ` +
        'dependency, so this usually means installation ran with --no-optional or ' +
        '--omit=optional. Reinstall with optional dependencies enabled.',
    );
  }
}

const binary = resolveBinary();

// stdio: 'inherit' gives the Go process the real file descriptors, which the MCP
// stdio transport needs — anything piped through Node here would add buffering
// between the client and the server.
const result = spawnSync(binary, process.argv.slice(2), {
  stdio: 'inherit',
  windowsHide: true,
});

if (result.error) {
  if (result.error.code === 'ENOENT') {
    fail(`binary not found at ${binary}. Reinstall the package.`);
  }
  if (result.error.code === 'EACCES') {
    fail(`binary at ${binary} is not executable. Reinstall the package.`);
  }
  fail(`failed to start ${binary}: ${result.error.message}`);
}

// Report a signal death the way a shell would, so a supervisor sees the real cause.
if (result.signal) {
  process.kill(process.pid, result.signal);
}

process.exit(result.status === null ? 1 : result.status);
