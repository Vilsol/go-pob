#!/usr/bin/env bash
set -euo pipefail
out=$(realpath -m "$1")
cd "$(dirname "$0")/../.."
[[ -f dist/index.html ]] || { echo "package: desktop/dist has no index.html (run task build-desktop)" >&2; exit 1; }
command -v makensis >/dev/null || { echo "package: needs makensis (apt install nsis / nix-shell -p nsis)" >&2; exit 1; }
version=${VERSION:-0.0.0}
numeric=${version%%[-+]*}
[[ $numeric =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "package: VERSION $version is not semver" >&2; exit 1; }
stage=$(mktemp -d)
syso=go-pob_windows_amd64.syso
trap 'rm -rf "$stage" "$syso"' EXIT
mkdir -p "$out"

read -ra wails3 <<<"${WAILS3:-env CGO_ENABLED=0 go run github.com/wailsapp/wails/v3/cmd/wails3@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)}"
cp -r build/windows "$stage/windows"
sed -i "s/\"0\.0\.0\"/\"$numeric\"/g" "$stage/windows/info.json"
sed -i "s/version=\"0\.0\.0\.0\"/version=\"$numeric.0\"/" "$stage/windows/go-pob.exe.manifest"
"${wails3[@]}" generate syso -arch amd64 -icon "$stage/windows/icon.ico" -manifest "$stage/windows/go-pob.exe.manifest" \
	-info "$stage/windows/info.json" -out "$syso" >/dev/null

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -tags production -o "$stage/go-pob.exe" \
	-ldflags "-s -w -H windowsgui -X main.version=$version -X main.updateURL=${UPDATE_URL:-} -X main.updatePublicKey=${UPDATE_PUBLIC_KEY:-}" .
[[ -z ${SIGN_CMD:-} ]] || $SIGN_CMD "$stage/go-pob.exe"
rm -f "$out/go-pob-windows-amd64.zip"
(cd "$stage" && zip -q -9 "$out/go-pob-windows-amd64.zip" go-pob.exe)

"${wails3[@]}" generate webview2bootstrapper -dir "$stage/windows/nsis" >/dev/null 2>&1
sign=()
[[ -z ${SIGN_CMD:-} ]] || sign=(-DSIGN_CMD="cd $(printf %q "$PWD") && $SIGN_CMD")
(cd "$stage/windows/nsis" && makensis -V2 -DARG_WAILS_AMD64_BINARY="$stage/go-pob.exe" -DINFO_PRODUCTVERSION="$numeric" \
	-DOUTFILE="$out/go-pob-amd64-installer.exe" "${sign[@]}" project.nsi)
[[ -z ${SIGN_CMD:-} ]] || $SIGN_CMD "$out/go-pob-amd64-installer.exe"
ls -la "$out"
