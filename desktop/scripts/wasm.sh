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
