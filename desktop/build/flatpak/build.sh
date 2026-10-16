#!/usr/bin/env bash
set -euo pipefail
id=$1
src=$PWD
version=$(sed -n 's/.*<release version="\([^"]*\)".*/\1/p' "desktop/build/flatpak/$id.metainfo.xml" | head -n1)
export GOPROXY=file://$src/goproxy GOFLAGS="-mod=readonly -modcacherw" GOSUMDB=off GOTOOLCHAIN=local
export GOPATH=$src/.gopath GOCACHE=$src/.gocache

GOFLAGS="$GOFLAGS -buildvcs=false" desktop/scripts/wasm.sh

python3 -I -m http.server 48173 --bind 127.0.0.1 --directory npm-registry >/dev/null 2>&1 &
registry=$!
trap 'kill $registry 2>/dev/null || true' EXIT
for _ in $(seq 50); do python3 -I -c 'import socket; socket.create_connection(("127.0.0.1", 48173))' 2>/dev/null && break; sleep 0.1; done
pnpm=(node "$src/pnpm-dist/package/bin/pnpm.cjs")
tar -xzf pnpm-dist/pnpm-*.tgz -C pnpm-dist
(
	cd frontend
	CI=1 "${pnpm[@]}" install --frozen-lockfile --registry=http://127.0.0.1:48173/ --store-dir="$src/.pnpm-store" \
		--config.manage-package-manager-versions=false --reporter=append-only
	VITE_DESKTOP=1 node node_modules/vite/bin/vite.js build --logLevel warn
)
kill $registry

mkdir -p desktop/dist
(cd frontend/build && find . -type f ! -name go-pob.wasm ! -name go-pob.wasm.br ! -name service-worker.js ! -name _headers \
	-exec install -Dm644 {} "$src/desktop/dist/{}" \;)

cd desktop
go build -trimpath -buildvcs=false -tags gtk3,production -ldflags "-s -w -X main.version=$version -X main.updateManager=Flathub" -o /app/bin/go-pob .
install -Dm644 icon.png "/app/share/icons/hicolor/512x512/apps/$id.png"
install -Dm644 "build/flatpak/$id.desktop" "/app/share/applications/$id.desktop"
install -Dm644 "build/flatpak/$id.metainfo.xml" "/app/share/metainfo/$id.metainfo.xml"
