#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
port=${PORT:-5180}
bin=desktop/bin/go-pob-dev
desktop/scripts/build.sh "$bin" "$@"
(cd frontend && VITE_DESKTOP=1 exec npx vite dev --port "$port" --strictPort) &
vite=$!
trap 'kill $vite 2>/dev/null || true' EXIT
until curl -fs -o /dev/null "http://localhost:$port/"; do
	kill -0 $vite 2>/dev/null || { echo "dev-desktop: vite dev exited" >&2; exit 1; }
	sleep 0.5
done
GOPOB_DESKTOP_DEVSERVER="http://localhost:$port" "$bin"
