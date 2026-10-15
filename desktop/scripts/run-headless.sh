#!/usr/bin/env bash
set -euo pipefail
bin=$(realpath "$1")
probe=$(realpath "$2")
secs=${3:-120}
shift $(($# < 3 ? $# : 3))
[[ ${1:-} == -- ]] && shift

home=${DESK_HOME:-}
fresh=0
if [[ -z $home ]]; then
	home=$(mktemp -d "${TMPDIR:-/tmp}/go-pob-desktop-home.XXXXXX")
	fresh=1
fi
mkdir -p "$home"

display=
for n in $(seq 90 120); do
	if [[ ! -e /tmp/.X11-unix/X$n && ! -e /tmp/.X$n-lock ]]; then
		display=:$n
		break
	fi
done
[[ -n $display ]] || { echo "run-headless: no free X display" >&2; exit 2; }

Xvfb "$display" -screen 0 1440x900x24 -nolisten tcp >/dev/null 2>&1 &
xvfb=$!
cleanup() {
	kill "$xvfb" 2>/dev/null || true
	wait "$xvfb" 2>/dev/null || true
	if [[ $fresh == 1 ]]; then rm -rf "$home"; fi
}
trap cleanup EXIT
for _ in $(seq 50); do [[ -e /tmp/.X11-unix/X${display#:} ]] && break; sleep 0.1; done

gio=${GIO_EXTRA_MODULES:-}
schemas=${GSETTINGS_SCHEMA_DIR:-}
if command -v nix-build >/dev/null; then
	[[ -n $gio ]] || gio=$(nix-build --no-out-link '<nixpkgs>' -A glib-networking 2>/dev/null)/lib/gio/modules
	[[ -n $schemas ]] || schemas=$(echo "$(nix-build --no-out-link '<nixpkgs>' -A gtk3 2>/dev/null)"/share/gsettings-schemas/*/glib-2.0/schemas)
fi

set +e
env -u WAYLAND_DISPLAY -u XDG_SESSION_TYPE \
	DISPLAY="$display" GDK_BACKEND=x11 \
	HOME="$home" XDG_CONFIG_HOME="$home/.config" XDG_CACHE_HOME="$home/.cache" XDG_DATA_HOME="$home/.local/share" \
	GIO_EXTRA_MODULES="$gio" ${schemas:+GSETTINGS_SCHEMA_DIR="$schemas"} GOPOB_DESKTOP_PROBE="$probe" TMPDIR="$home" \
	DESK_BIN="$bin" DESK_SIDE="${DESK_SIDE:-}" \
	timeout "$secs" dbus-run-session -- bash -c '
		"$DESK_BIN" "$@" &
		app=$!
		if [[ -n $DESK_SIDE ]]; then
			bash -c "$DESK_SIDE"
			echo "run-headless: side command exited $?" >&2
		fi
		wait $app' run-headless "$@"
code=$?
set -e
echo "run-headless: exit $code (124 = timeout)" >&2
exit 0
