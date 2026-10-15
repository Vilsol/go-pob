#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
old=$(realpath "$1")
new=$(realpath "$2")
key=$(realpath "$3")
port=${PORT:-18731}
read -ra wails3 <<<"${WAILS3:-env CGO_ENABLED=0 go run github.com/wailsapp/wails/v3/cmd/wails3@$(env -C desktop go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)}"
version=$(basename "$(ls "$new"/go-pob_*_amd64.deb)")
version=${version#go-pob_}
version=${version%_amd64.deb}
work=$(mktemp -d)
server=
cleanup() {
	[[ -z $server ]] || kill "$server" 2>/dev/null || true
	rm -rf "$work"
}
trap cleanup EXIT

manifest() { # manifest <dir> [--key k]
	(cd "$1" && "${wails3[@]}" updater manifest --version "$version" --output update.json "${@:2}" go-pob-linux-amd64 >/dev/null)
}
mkdir -p "$work"/{signed,tampered,unsigned,otherkey}
for s in signed tampered unsigned otherkey; do cp "$new/go-pob-linux-amd64" "$work/$s/"; done
manifest "$work/signed" --key "$key"
cp "$work/signed/update.json" "$work/tampered/"
printf 'X' | dd of="$work/tampered/go-pob-linux-amd64" bs=1 seek=4096 conv=notrunc status=none
manifest "$work/unsigned"
(cd "$work/otherkey" && "${wails3[@]}" updater genkey --out other.key >/dev/null)
manifest "$work/otherkey" --key "$work/otherkey/other.key"
rm "$work/otherkey"/other.key*

failed=0
scenario() { # scenario <release> <mode> <expect...> ("!x": x must not appear)
	local release=$1 mode=$2
	shift 2
	python3 -m http.server --bind 127.0.0.1 "$port" -d "$work/$release" >/dev/null 2>&1 &
	server=$!
	sleep 0.5
	local probe=$mode
	[[ $mode != appimage ]] || probe=check
	local script="printf 'window.__RELEASE = { update: \"$probe\" };\n' >/tmp/probe.js && cat /src/desktop/release-e2e/probe.js >>/tmp/probe.js"
	if [[ $mode == appimage ]]; then
		script="$script && /src/desktop/scripts/run-headless.sh /old/go-pob-x86_64.AppImage /tmp/probe.js 60 -- --appimage-extract-and-run"
	else
		script="$script && mkdir -p /tmp/app /tmp/h && cp /old/go-pob-linux-amd64 /tmp/app/go-pob &&
			DESK_HOME=/tmp/h DESK_SIDE='sleep 15' /src/desktop/scripts/run-headless.sh /tmp/app/go-pob /tmp/probe.js 60;
			echo \"on disk: \$(sha256sum </tmp/app/go-pob | cut -c1-16) (old $(sha256sum <"$old/go-pob-linux-amd64" | cut -c1-16), new $(sha256sum <"$new/go-pob-linux-amd64" | cut -c1-16))\";
			cat /tmp/h/wails-update-*.log 2>/dev/null | sed 's/^/helper: /'"
	fi
	echo "=== $release ($mode)"
	local log
	log=$(docker run --rm --network host -v "$old:/old:ro" -v "$PWD:/src:ro" go-pob-desktop-run:ubuntu-24.04 bash -c "$script" 2>&1 |
		grep -E 'probe: |on disk|helper:' | sed 's/^[0-9/]* [0-9:]* probe: //' | awk '!seen[$0]++' || true)
	kill "$server" && wait "$server" 2>/dev/null || true
	server=
	echo "$log"
	local ok=1
	for want in "$@"; do
		if [[ $want == !* ]]; then
			! grep -qF -- "${want#!}" <<<"$log" || { ok=0 && echo "present: ${want#!}"; }
		else
			grep -qF -- "$want" <<<"$log" || { ok=0 && echo "missing: $want"; }
		fi
	done
	echo "=== $release ($mode): $([[ $ok == 1 ]] && echo as expected || echo UNEXPECTED)"
	[[ $ok == 1 ]] || failed=1
}

new_sha=$(sha256sum <"$new/go-pob-linux-amd64" | cut -c1-16)
scenario signed appimage "\"version\":\"$version\"" '"install":false'
scenario signed restart "\"version\":\"$version\"" '"install":true' '"ready":true' 'helper: ' 'swap succeeded' "on disk: $new_sha"
scenario tampered install 'digest mismatch' '!"ready":true'
scenario unsigned install "release $version is not signed" '!"ready":true'
scenario otherkey install 'ed25519ph signature did not verify' '!"ready":true'
exit $failed
