#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
manifest=(desktop/build/flatpak/*.yml)
app=${APP_ID:-$(basename "${manifest[0]}" .yml)}
home=$(mktemp -d "${TMPDIR:-/tmp}/go-pob-flatpak-portal.XXXXXX")
trap 'rm -rf "$home"' EXIT
outside=$home/outside
mkdir -p "$outside/pob" "$home/io"
corpus=(testdata/corpus-3.29/*.xml)
cp "${corpus[0]}" "$outside/build.xml"

doc=
if [[ ${XDG_RUNTIME_DIR:-} != "/run/user/$(id -u)" ]]; then
	doc="--filesystem=$XDG_RUNTIME_DIR/doc"
fi
cat >"$home/io/app" <<EOF
exec env -u GIO_EXTRA_MODULES -u GSETTINGS_SCHEMA_DIR flatpak run --filesystem="\$GOPOB_DESKTOP_PROBE":ro $doc $app "\$@"
EOF
chmod +x "$home/io/app"

echo "--- control: the sandbox alone cannot read $outside"
if HOME=$home flatpak run --command=cat "$app" "$outside/build.xml" >/dev/null 2>&1; then
	echo "control FAILED: the sandbox reads the file without the portal"
	exit 1
fi
echo "control ok: not readable"

cat >"$home/io/side.sh" <<'EOF'
answer() {
	local id
	id=$(timeout 60 xdotool search --sync --onlyvisible --name "^$1\$" | head -n1) || { echo "side: no \"$1\" dialog" >&2; return 1; }
	sleep 1
	xdotool windowfocus --sync "$id"
	xdotool key ctrl+a
	xdotool type --delay 20 "$2"
	sleep 0.5
	xdotool key Return
	echo "side: answered \"$1\" with $2" >&2
	sleep 2
	echo "side: granted to $APP: $(cd "$XDG_RUNTIME_DIR/doc/by-app/$APP" && find . -mindepth 2 -maxdepth 2 | sort | paste -sd' ')" >&2
}
answer "Builds folder" "$OUTSIDE/pob/"
sleep 2
if xdotool search --onlyvisible --name "^Builds folder\$" >/dev/null; then xdotool key Return; echo "side: accepted the folder" >&2; fi
answer "Open build file" "$OUTSIDE/build.xml"
answer "Save" "$OUTSIDE/portal-save.txt"
EOF

run() {
	{
		echo "window.__PORTAL = { phase: \"$1\" };"
		cat desktop/flatpak-e2e/portal.js
	} >"$home/io/probe-$1.js"
	DESK_HOME=$home OUTSIDE=$outside DESK_SIDE=${2:-} desktop/scripts/run-headless.sh "$home/io/app" "$home/io/probe-$1.js" "${SECS:-180}" 2>&1 | tee -a "${KEEP_LOG:-/dev/null}" |
		grep -E 'probe: |side: |run-headless' | sed 's/^[0-9/]* [0-9:]* probe: //'
}

export OUTSIDE=$outside APP=$app
run first "bash $home/io/side.sh" | tee "$home/first.log"
run second | tee "$home/second.log"
echo "--- on the host:"
cat "$outside/portal-save.txt"
echo
ls "$outside/pob"
grep -q '"done":"first"' "$home/first.log" && grep -q '"done":"second"' "$home/second.log"
test -f "$outside/pob/portal-build.xml"
grep -q 'saved through the portal' "$outside/portal-save.txt"
