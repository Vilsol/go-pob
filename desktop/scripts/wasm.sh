#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
GOOS=js GOARCH=wasm go build -trimpath -tags=timetzdata -o frontend/static/go-pob.wasm ./wasm
gzip -nkf9 frontend/static/go-pob.wasm
# only the .gz reaches static/, so no copy of the build embeds the raw PoE2 module
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
GOOS=js GOARCH=wasm go build -trimpath -tags=timetzdata -o "$tmp/go-pob2.wasm" ./wasm2
gzip -nc9 "$tmp/go-pob2.wasm" >frontend/static/go-pob2.wasm.gz
# the TinyGo builds Settings can switch to, compressed the same way; a host without TinyGo (the Nix and Flatpak
# sandboxes) ships without them, and choosing TinyGo there falls back to Go with a notice
rm -f frontend/static/go-pob-tiny.wasm.gz frontend/static/go-pob2-tiny.wasm.gz
if ! command -v tinygo >/dev/null; then
  echo "wasm.sh: no tinygo on PATH, so no TinyGo engine in this build" >&2
  exit 0
fi
./.github/tinygo_build.sh wasm "$tmp/go-pob-tiny.wasm" -tags=timetzdata
gzip -nc9 "$tmp/go-pob-tiny.wasm" >frontend/static/go-pob-tiny.wasm.gz
./.github/tinygo_build.sh wasm2 "$tmp/go-pob2-tiny.wasm" -tags=timetzdata
gzip -nc9 "$tmp/go-pob2-tiny.wasm" >frontend/static/go-pob2-tiny.wasm.gz
