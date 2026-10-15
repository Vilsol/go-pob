#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
out=$(realpath -m "${1:-desktop/bin/release-windows}")
if command -v makensis >/dev/null && command -v zip >/dev/null; then
	exec desktop/build/windows/package.sh "$out"
elif command -v nix-shell >/dev/null; then
	exec nix-shell -p nsis zip --run "$(printf '%q ' desktop/build/windows/package.sh "$out")"
fi
echo "package-windows: needs makensis (nsis) and zip, or nix-shell" >&2
exit 1
