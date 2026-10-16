#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
GOOS=js GOARCH=wasm go build -trimpath -tags=timetzdata -o frontend/static/go-pob.wasm ./wasm
gzip -nkf9 frontend/static/go-pob.wasm
