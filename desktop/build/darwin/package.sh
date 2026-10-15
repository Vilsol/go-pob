#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$1"
out=$(cd "$1" && pwd)
cd "$(dirname "$0")/../.."
[[ $(uname) == Darwin ]] || { echo "package: the macOS app builds on macOS only" >&2; exit 1; }
[[ -f dist/index.html ]] || { echo "package: desktop/dist has no index.html (run task build-desktop)" >&2; exit 1; }
version=${VERSION:-0.0.0}
numeric=${version%%[-+]*}
[[ $numeric =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "package: VERSION $version is not semver" >&2; exit 1; }
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

export CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=12.0 CGO_CFLAGS=-mmacosx-version-min=12.0 CGO_LDFLAGS=-mmacosx-version-min=12.0
for arch in ${ARCHS:-arm64 amd64}; do
	app=$stage/$arch/go-pob.app
	mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
	GOOS=darwin GOARCH=$arch go build -trimpath -buildvcs=false -tags production -o "$app/Contents/MacOS/go-pob" \
		-ldflags "-s -w -X main.version=$version -X main.updateURL=${UPDATE_URL:-} -X main.updatePublicKey=${UPDATE_PUBLIC_KEY:-}" .
	cp build/darwin/icons.icns "$app/Contents/Resources/"
	sed "s#<string>0\.0\.0</string>#<string>$numeric</string>#g" build/darwin/Info.plist >"$app/Contents/Info.plist"
	plutil -lint "$app/Contents/Info.plist"
	if [[ -n ${SIGN_IDENTITY:-} ]]; then
		codesign --force --options runtime --timestamp --sign "$SIGN_IDENTITY" "$app"
	else
		codesign --force --sign - "$app"
	fi
	codesign --verify --strict --verbose=2 "$app"

	mkdir "$stage/$arch/dmg"
	cp -R "$app" "$stage/$arch/dmg/"
	ln -s /Applications "$stage/$arch/dmg/Applications"
	dmg=$out/go-pob-macos-$arch.dmg
	rm -f "$dmg"
	hdiutil create -quiet -volname go-pob -srcfolder "$stage/$arch/dmg" -format UDZO "$dmg"
	if [[ -n ${SIGN_IDENTITY:-} ]]; then
		codesign --force --timestamp --sign "$SIGN_IDENTITY" "$dmg"
		if [[ -n ${NOTARY_PROFILE:-} ]]; then
			xcrun notarytool submit "$dmg" --keychain-profile "$NOTARY_PROFILE" --wait
			xcrun stapler staple "$dmg"
			xcrun stapler staple "$app"
		fi
	fi
	rm -f "$out/go-pob-macos-$arch.zip"
	ditto -c -k --keepParent "$app" "$out/go-pob-macos-$arch.zip"
done
ls -la "$out"
