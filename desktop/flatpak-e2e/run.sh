#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
manifest=(desktop/build/flatpak/*.yml)
app=${APP_ID:-$(basename "${manifest[0]}" .yml)}
corpus=${CORPUS_DIR:-testdata/corpus-3.29}
all=("$corpus"/*.xml)
picks=()
for f in "${all[@]:0:${N:-3}}"; do picks+=("$(realpath "$f")"); done
home=$(mktemp -d "${TMPDIR:-/tmp}/go-pob-flatpak-e2e.XXXXXX")
trap 'rm -rf "$home"' EXIT
pob="$home/.wine/drive_c/users/me/Documents/Path of Building"
mkdir -p "$pob/Builds" "$home/exports" "$home/io"
names=()
for i in "${!picks[@]}"; do names+=("e2e-$((i + 1))"); done

cat >"$home/io/app" <<EOF
args=(--filesystem="\$GOPOB_DESKTOP_PROBE":ro --filesystem="$home/exports")
IFS=: read -ra open <<<"\${GOPOB_DESKTOP_OPEN:-}"
for f in "\${open[@]}"; do args+=(--filesystem="\$f":ro); done
exec env -u GIO_EXTRA_MODULES -u GSETTINGS_SCHEMA_DIR flatpak run "\${args[@]}" $app "\$@"
EOF
chmod +x "$home/io/app"

probe() {
	local hint=false
	[[ $1 != first ]] || hint=true
	printf 'window.__E2E = { phase: "%s", builds: [%s], link: "%s", source: "app", hint: %s, folder: false };\n' "$1" "$(printf '"%s",' "${names[@]}")" "${LINK-pob://pobbin/h-M3H3It2fDw}" "$hint"
	cat desktop/e2e/probe.js
}

run() {
	probe "$1" >"$home/io/probe-$1.js"
	local t0=$SECONDS
	DESK_HOME=$home GOPOB_DESKTOP_OPEN=$(printf '%s\n' "${picks[@]}" | paste -sd:) GOPOB_DESKTOP_SAVE_TO=$home/exports \
		desktop/scripts/run-headless.sh "$home/io/app" "$home/io/probe-$1.js" "${SECS:-240}" 2>&1 | tee -a "$home/raw.log" | grep -E 'probe: |run-headless' | sed 's/^[0-9/]* [0-9:]* probe: //'
	echo "launch $1: $((SECONDS - t0)) s wall"
}

run first | tee "$home/first.log"
echo "--- the app's builds folder after launch 1 (PoB's stays empty):"
ls -la "$home/.var/app/$app/config/go-pob/builds"
[[ -z $(ls -A "$pob/Builds") ]]
cache=$home/.var/app/$app/cache/go-pob/webview
echo "--- disk cache in the app's folder: $(find "$cache" -type f | wc -l) files, $(du -sh "$cache" | cut -f1)"
run second | tee "$home/second.log"
echo "--- the export's zip:"
python3 -I -c 'import sys, zipfile; [print(f"{i.filename} {i.file_size}") for p in sys.argv[1:] for i in zipfile.ZipFile(p).infolist()]' "$home"/exports/*.zip

entry=${FLATPAK_USER_DIR:-$HOME/.local/share/flatpak}/exports/share/applications/$app.desktop
grep -q '^MimeType=.*x-scheme-handler/pob;' "$entry"
t0=$SECONDS
DESK_HOME=$home DESK_SIDE="timeout 60 xdotool search --sync --onlyvisible --name '^go-pob\$' >/dev/null; sleep 5; gio launch '$entry' '${LINK-pob://pobbin/h-M3H3It2fDw}'" \
	desktop/scripts/run-headless.sh "$home/io/app" desktop/flatpak-e2e/link.js "${SECS:-240}" 2>&1 | tee -a "$home/raw.log" |
	grep -E 'probe: |run-headless' | sed 's/^[0-9/]* [0-9:]* probe: //' | tee "$home/link.log"
echo "launch link: $((SECONDS - t0)) s wall"
[[ -z ${KEEP_LOG:-} ]] || cp "$home/raw.log" "$KEEP_LOG"
if grep -q '"fail"' "$home/first.log" "$home/second.log" "$home/link.log"; then exit 1; fi
grep -q '"done":"first"' "$home/first.log"
grep -q '"done":"second"' "$home/second.log"
grep -q '"done":"link"' "$home/link.log"
