#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
bin=${BIN:-desktop/bin/go-pob}
corpus=${CORPUS_DIR:-testdata/corpus-3.29}
mapfile -t picks < <(ls "$corpus"/*.xml | head -n "${N:-3}")
home=$(mktemp -d "${TMPDIR:-/tmp}/go-pob-desktop-e2e.XXXXXX")
trap 'rm -rf "$home"' EXIT
pob="$home/.wine/drive_c/users/me/Documents/Path of Building"
mkdir -p "$pob/Builds"
names=()
for i in "${!picks[@]}"; do names+=("e2e-$((i + 1))"); done
probe() {
	printf 'window.__E2E = { phase: "%s", builds: [%s], link: "%s" };\n' "$1" "$(printf '"%s",' "${names[@]}")" "${LINK-pob://pobbin/h-M3H3It2fDw}"
	cat desktop/e2e/probe.js
}

run() {
	probe "$1" >"$home/probe-$1.js"
	local t0=$SECONDS
	DESK_HOME=$home GOPOB_DESKTOP_OPEN=$(realpath "${picks[@]}" | paste -sd:) GOPOB_DESKTOP_SAVE_TO=$home/exports \
		desktop/scripts/run-headless.sh "$bin" "$home/probe-$1.js" "${SECS:-240}" 2>&1 | grep -E 'probe: |run-headless' | sed 's/^[0-9/]* [0-9:]* probe: //'
	echo "launch $1: $((SECONDS - t0)) s wall"
}

run first | tee "$home/first.log"
echo "--- PoB's Builds folder after launch 1:"
ls -la "$pob/Builds"
echo "--- disk cache: $(find "$home/.cache/go-pob/webview" -type f | wc -l) files, $(du -sh "$home/.cache/go-pob/webview" | cut -f1)"
if [[ -n ${KEEP:-} ]]; then
	mkdir -p "$KEEP" && cp "$pob/Builds"/*.xml "$KEEP"/ && printf '%s\n' "${picks[@]}" >"$KEEP/sources.txt"
fi
mkdir -p "$home/exports"
run second | tee "$home/second.log"
echo "--- the export's zip:"
python3 -I -c 'import sys, zipfile; [print(f"{i.filename} {i.file_size}") for p in sys.argv[1:] for i in zipfile.ZipFile(p).infolist()]' "$home"/exports/*.zip
! grep -q '"fail"' "$home/first.log" "$home/second.log"
