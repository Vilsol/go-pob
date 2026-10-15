#!/usr/bin/env bash
set -euo pipefail
if [[ -z ${WINE:-} ]] && ! command -v wine >/dev/null; then
	exec nix-shell -p wineWow64Packages.stable zip --run "$(printf '%q ' "$(realpath "$0")" "$@")"
fi
cd "$(dirname "$0")/.."
port=${PORT:-18732}
work=$(mktemp -d)
server=
cleanup() {
	[[ -z $server ]] || kill "$server" 2>/dev/null || true
	WINEPREFIX=$work/pfx wineserver -k 2>/dev/null || true
	rm -rf "$work"
}
trap cleanup EXIT
wine=${WINE:-wine}
read -ra wails3 <<<"${WAILS3:-env CGO_ENABLED=0 go run github.com/wailsapp/wails/v3/cmd/wails3@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)}"

for v in 1.0.0 1.1.0; do
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=$v" -o "$work/winupdate-$v.exe" ./release-e2e/winupdate
done
(cd "$work" && "${wails3[@]}" updater genkey --out key >/dev/null && "${wails3[@]}" updater genkey --out other.key >/dev/null)
pub=$(sed '1d;$d' "$work/key.pub" | base64 -d | tail -c 32 | base64)

manifest() { # manifest <dir> [--key k]
	(cd "$1" && "${wails3[@]}" updater manifest --version 1.1.0 --output update.json "${@:2}" go-pob-windows-amd64.zip >/dev/null)
}
mkdir -p "$work"/{signed,tampered,unsigned,otherkey}
cp "$work/winupdate-1.1.0.exe" "$work/go-pob.exe"
(cd "$work" && zip -q go-pob-windows-amd64.zip go-pob.exe)
for s in signed tampered unsigned otherkey; do cp "$work/go-pob-windows-amd64.zip" "$work/$s/"; done
manifest "$work/signed" --key "$work/key"
cp "$work/signed/update.json" "$work/tampered/"
printf 'X' | dd of="$work/tampered/go-pob-windows-amd64.zip" bs=1 seek=4096 conv=notrunc status=none
manifest "$work/unsigned"
manifest "$work/otherkey" --key "$work/other.key"

export WINEPREFIX=$work/pfx WINEDEBUG=-all WINEDLLOVERRIDES='mscoree,mshtml=' HOME=$work/home
unset DISPLAY WAYLAND_DISPLAY
mkdir -p "$HOME"
old_sha=$(sha256sum <"$work/winupdate-1.0.0.exe" | cut -c1-16)
new_sha=$(sha256sum <"$work/winupdate-1.1.0.exe" | cut -c1-16)
failed=0
scenario() { # scenario <release> <mode> <expect...> ("!x": x must not appear)
	local release=$1 mode=$2 app=$work/app-$1
	shift 2
	python3 -m http.server --bind 127.0.0.1 "$port" -d "$work/$release" >/dev/null 2>&1 &
	server=$!
	sleep 0.5
	mkdir -p "$app"
	cp "$work/winupdate-1.0.0.exe" "$app/go-pob.exe"
	WINUPDATE_LOG=$app/log.txt timeout 120 "$wine" "$app/go-pob.exe" "$mode" "http://127.0.0.1:$port/update.json" "$pub" >/dev/null 2>&1 || true
	timeout 60 wineserver -w || true
	kill "$server" && wait "$server" 2>/dev/null || true
	server=
	local log
	log="$(cat "$app/log.txt" 2>/dev/null)
on disk: $(sha256sum <"$app/go-pob.exe" | cut -c1-16) (old $old_sha, new $new_sha)
$(cat "$work"/pfx/drive_c/users/*/AppData/Local/Temp/wails-update-*.log 2>/dev/null | sed 's/^/helper: /' || true)"
	rm -f "$work"/pfx/drive_c/users/*/AppData/Local/Temp/wails-update-*.log
	echo "=== $release ($mode)"
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

scenario signed restart 'check: version 1.1.0, go-pob-windows-amd64.zip' 'install: ready' 'quit' 'running 1.1.0' "on disk: $new_sha"
scenario tampered install 'digest mismatch' '!install: ready' "on disk: $old_sha"
scenario unsigned install 'release 1.1.0 is not signed' '!install: ready' "on disk: $old_sha"
scenario otherkey install 'ed25519ph signature did not verify' '!install: ready' "on disk: $old_sha"
exit $failed
