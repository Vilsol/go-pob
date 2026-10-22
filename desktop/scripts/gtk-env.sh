#!/usr/bin/env bash
set -euo pipefail
if pkg-config --exists webkit2gtk-4.1 gtk+-3.0 2>/dev/null; then
	exec "$@"
elif command -v nix-shell >/dev/null; then
	inc=()
	nix-instantiate --find-file nixpkgs >/dev/null 2>&1 || inc=(-I "nixpkgs=$(nix eval --raw nixpkgs#path)")
	exec nix-shell "${inc[@]}" -p pkg-config gcc gtk3 webkitgtk_4_1 libsoup_3 --run "$(printf '%q ' "$@")"
fi
echo "desktop: needs the webkit2gtk-4.1 and gtk+-3.0 development packages (or nix-shell)" >&2
exit 1
