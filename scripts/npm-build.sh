#!/usr/bin/env bash
# Assembles the npm packages from GoReleaser's build output.
#
#   scripts/npm-build.sh <version> [dist-dir] [out-dir]
#
# Produces out-dir/cli (the "3x-ui-mcp" package users install) and one package
# per platform holding that platform's binary. The platform packages are what
# the CLI pulls in through optionalDependencies, so npm downloads exactly one
# of them and never compiles anything.
set -euo pipefail

VERSION="${1:?usage: npm-build.sh <version> [dist-dir] [out-dir]}"
VERSION="${VERSION#v}"
DIST="${2:-dist}"
OUT="${3:-dist/npm}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# npm platform+arch  ->  GoReleaser's build directory suffix
TARGETS=(
  "linux-x64:linux_amd64_v1"
  "linux-arm64:linux_arm64_v8.0"
  "darwin-x64:darwin_amd64_v1"
  "darwin-arm64:darwin_arm64_v8.0"
  "win32-x64:windows_amd64_v1"
  "win32-arm64:windows_arm64_v8.0"
)

rm -rf "$OUT"
mkdir -p "$OUT"

# --- the platform packages -------------------------------------------------

for target in "${TARGETS[@]}"; do
  npm_target="${target%%:*}"
  go_target="${target#*:}"
  os="${npm_target%-*}"
  arch="${npm_target#*-}"

  exe="xui-mcp"
  [ "$os" = "win32" ] && exe="xui-mcp.exe"

  # GoReleaser v2 names build dirs "<id>_<goos>_<goarch>_<variant>"; the id is
  # unset in .goreleaser.yml so it defaults to the project name.
  src="$(find "$DIST" -type f -path "*${go_target}*" -name "$exe" -print -quit)"
  if [ -z "$src" ]; then
    echo "npm-build: no $exe built for $go_target under $DIST" >&2
    exit 1
  fi

  pkg_dir="$OUT/3x-ui-mcp-$npm_target"
  mkdir -p "$pkg_dir/bin"
  cp "$src" "$pkg_dir/bin/$exe"
  chmod +x "$pkg_dir/bin/$exe"

  # "os" and "cpu" make npm skip this package on every other platform, which is
  # what keeps an install to one binary instead of six.
  cat > "$pkg_dir/package.json" <<EOF
{
  "name": "3x-ui-mcp-$npm_target",
  "version": "$VERSION",
  "description": "Prebuilt xui-mcp binary for $os $arch. Installed automatically by the 3x-ui-mcp package.",
  "homepage": "https://github.com/pyworkload/3x-ui-mcp#readme",
  "repository": {
    "type": "git",
    "url": "git+https://github.com/pyworkload/3x-ui-mcp.git"
  },
  "license": "MIT",
  "author": "pyworkload",
  "os": ["$os"],
  "cpu": ["$arch"],
  "files": ["bin/$exe"],
  "preferUnplugged": true
}
EOF
  echo "npm-build: packaged 3x-ui-mcp-$npm_target ($(du -h "$pkg_dir/bin/$exe" | cut -f1))"
done

# --- the CLI package -------------------------------------------------------

cp -r "$REPO_ROOT/npm/cli" "$OUT/cli"
cp "$REPO_ROOT/LICENSE" "$OUT/cli/LICENSE"

# Stamp the real version into the CLI and pin every optional dependency to it,
# so a published CLI can only ever pull binaries from its own release.
node - "$OUT/cli/package.json" "$VERSION" <<'NODE'
const fs = require('node:fs');
const [file, version] = process.argv.slice(2);
const pkg = JSON.parse(fs.readFileSync(file, 'utf8'));
pkg.version = version;
for (const dep of Object.keys(pkg.optionalDependencies ?? {})) {
  pkg.optionalDependencies[dep] = version;
}
fs.writeFileSync(file, JSON.stringify(pkg, null, 2) + '\n');
NODE

echo "npm-build: staged $OUT/cli and 6 platform packages at version $VERSION"
