#!/usr/bin/env bash
set -euo pipefail

[ $# -ge 5 ] || { echo "usage: dump_each.sh <pob-dir> <repo-root> <builds-dir> <list-file> <out-dir> [jobs]" >&2; exit 2; }
pob=$(realpath "$1"); repo=$(realpath "$2"); builds=$(realpath "$3"); list=$4; out=$5; jobs=${6:-4}
luajit=${LUAJIT:-$repo/.luajit-det/bin/luajit}
[ -x "$luajit" ] || { echo "no deterministic luajit at $luajit (task luajit-det)" >&2; exit 2; }
[ -d "$pob/src" ] || { echo "no $pob/src" >&2; exit 2; }
[ -s "$list" ] || { echo "the build list $list is empty or missing" >&2; exit 2; }
if [ -d "$out" ] && [ -n "$(ls -A "$out")" ]; then
	echo "$out exists and is NOT EMPTY — refusing to overwrite another run's output" >&2
	exit 3
fi
mkdir -p "$out/.work"
out=$(realpath "$out")

LUA_PATH=$(cd "$pob/src" && mise x -- luajit -e 'io.write(package.path)')
LUA_CPATH=$(cd "$pob/src" && mise x -- luajit -e 'io.write(package.cpath)')
export LUA_PATH LUA_CPATH pob repo builds out luajit
machine=$(uname -m)
export machine

one() {
	n=${1%.xml}
	w="$out/.work/$n"
	mkdir -p "$w/in" "$w/dump"
	ln -s "$builds/$n.xml" "$w/in/$n.xml"
	setarch "$machine" -R unshare -rn -- sh -c \
		"ip link set lo up; cd '$pob/src' && exec '$luajit' '$repo/oracle/dump_build.lua' '$w/in' '$w/dump'" \
		>"$w/dump.log" 2>&1 || true
	if [ -s "$w/dump/$n.json" ]; then
		mv "$w/dump/$n.json" "$out/$n.json"
		rm -rf "$w"
	else
		echo "$n" >>"$out/FAILED"
	fi
}
export -f one

grep -v '^\s*\(#\|$\)' "$list" | xargs -P "$jobs" -I{} bash -c 'one "$(basename "{}")"'

done=$(find "$out" -maxdepth 1 -name '*.json' | wc -l)
failed=$( [ -f "$out/FAILED" ] && wc -l <"$out/FAILED" || echo 0)
echo "dump_each: $done dumped, $failed failed -> $out"
[ "$failed" -eq 0 ]
