#!/usr/bin/env bash
set -euo pipefail
dmg=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
build=$(cd "$(dirname "$2")" && pwd)/$(basename "$2")
here=$(cd "$(dirname "$0")" && pwd)
link=${LINK:-pob://pobbin/h-M3H3It2fDw}
secs=${SECS:-300}
work=$(mktemp -d)
mnt=$work/mnt
cleanup() {
	pkill -x go-pob 2>/dev/null || true
	hdiutil detach -quiet "$mnt" 2>/dev/null || true
	rm -rf "$work"
}
trap cleanup EXIT

hdiutil attach -quiet -nobrowse -readonly -mountpoint "$mnt" "$dmg"
cp -R "$mnt/go-pob.app" "$work/"
hdiutil detach -quiet "$mnt"
app=$work/go-pob.app
codesign --verify --strict --verbose=2 "$app"
codesign -dv "$app" 2>&1 | grep -E '^(Identifier|Format|Signature|TeamIdentifier|Runtime)' || true
lipo -archs "$app/Contents/MacOS/go-pob"

probe() {
	printf 'window.__SMOKE = { phase: "%s", link: "%s" };\n' "$1" "$link" >"$work/probe-$1.js"
	cat "$here/../../e2e/webview.js" >>"$work/probe-$1.js"
	echo "$work/probe-$1.js"
}

rss() { ps -axo rss=,comm= | awk '/go-pob|com\.apple\.WebKit/ { s += $1 } END { print s + 0 }'; }

wait_app() {
	local end=$((SECONDS + secs)) now
	peak=0
	while pgrep -x go-pob >/dev/null; do
		if ((SECONDS > end)); then
			pkill -x go-pob || true
			echo "go-pob still running after $secs s" >&2
			return 1
		fi
		now=$(rss)
		((now > peak)) && peak=$now
		sleep 0.5
	done
}

read_log() {
	local log="$2/Library/Application Support/go-pob/settings/probe-log.json"
	[[ -f $log ]] || { echo "phase $1: the probe wrote no log (it never ran)" >&2; return 1; }
	python3 -I -c 'import json, sys; [print(json.dumps(e)) for e in json.load(open(sys.argv[1]))]' "$log"
	grep -q "\"done\": *\"$1\"" "$log" || { echo "phase $1 failed" >&2; return 1; }
}

echo "--- the app, launched directly"
base=$(rss)
home=$work/home-main
mkdir -p "$home"
set +e
HOME=$home GOPOB_DESKTOP_PROBE=$(probe main) GOPOB_DESKTOP_OPEN=$build "$app/Contents/MacOS/go-pob" 2>"$work/main.err" &
pid=$!
wait_app
wait "$pid"
code=$?
set -e
grep -v 'probe: ' "$work/main.err" | tail -n 20 || true
read_log main "$home"
echo "exit $code; peak RSS, go-pob + WebKit: $(((peak - base) / 1024)) MB above a $((base / 1024)) MB baseline"
((code == 0))

echo "--- $link through Launch Services (Apple Event)"
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$app"
home=$work/home-link
mkdir -p "$home"
open --env "HOME=$home" --env "GOPOB_DESKTOP_PROBE=$(probe link)" "$link"
end=$((SECONDS + 30))
until pgrep -x go-pob >/dev/null; do
	((SECONDS < end)) || { echo "$link launched no go-pob" >&2; exit 1; }
	sleep 0.2
done
wait_app
read_log link "$home"
