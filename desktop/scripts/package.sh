#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
out=$(realpath -m "${1:-desktop/bin/release}")
mkdir -p "$out/.cache"
image=go-pob-desktop-build:ubuntu-24.04
docker build -q -t "$image" desktop/build/linux >/dev/null
exec docker run --rm --network host --user "$(id -u):$(id -g)" \
	-e HOME=/tmp -e GOCACHE=/out/.cache/go-build -e GOMODCACHE=/gomod -e GOTOOLCHAIN=local -e GOFLAGS=-mod=mod \
	-e TOOLS=/out/.cache/tools -e VERSION -e UPDATE_URL -e UPDATE_PUBLIC_KEY -e MAINTAINER -e WAILS3 \
	-v "$(go env GOROOT):/usr/local/go:ro" -v "$(go env GOMODCACHE):/gomod" -v "$PWD:/src:ro" -v "$out:/out" \
	-w /src/desktop "$image" bash -c 'PATH=/usr/local/go/bin:$PATH exec build/linux/package.sh /out'
