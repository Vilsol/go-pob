#!/usr/bin/env bash
set -euo pipefail
out=$(realpath -m "$1")
cd "$(dirname "$0")/../.."
[[ -f dist/index.html ]] || { echo "package: desktop/dist has no index.html (run task build-desktop)" >&2; exit 1; }
version=${VERSION:-0.0.0}
tools=${TOOLS:-$out/.tools}
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$out" "$tools"

go build -trimpath -buildvcs=false -tags gtk3,production -o "$stage/go-pob" \
	-ldflags "-s -w -X main.version=$version -X main.updateURL=${UPDATE_URL:-} -X main.updatePublicKey=${UPDATE_PUBLIC_KEY:-}" .
cp "$stage/go-pob" "$out/go-pob-linux-amd64"

fetch() {
	if [[ ! -f $3 ]] || ! echo "$2  $3" | sha256sum -c --quiet 2>/dev/null; then
		curl -fsSLo "$3" "$1"
		echo "$2  $3" | sha256sum -c --quiet
	fi
}
fetch https://github.com/AppImage/appimagetool/releases/download/1.9.1/appimagetool-x86_64.AppImage \
	ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0 "$tools/appimagetool-x86_64.AppImage"
fetch https://github.com/AppImage/type2-runtime/releases/download/20251108/runtime-x86_64 \
	2fca8b443c92510f1483a883f60061ad09b46b978b2631c807cd873a47ec260d "$tools/runtime-x86_64"
chmod +x "$tools/appimagetool-x86_64.AppImage"

app=$stage/go-pob.AppDir
mkdir -p "$app/usr/bin" "$app/usr/lib/gio/modules" "$app/usr/share/glib-2.0/schemas"
cp "$stage/go-pob" "$app/usr/bin/"
cp build/linux/AppRun build/linux/go-pob.desktop "$app/"
cp icon.png "$app/go-pob.png"
ln -s go-pob.png "$app/.DirIcon"
cp "$(pkg-config --variable=giomoduledir gio-2.0)/libgiognutls.so" "$app/usr/lib/gio/modules/"
cp "$(pkg-config --variable=schemasdir gio-2.0)"/org.gtk.Settings.*.gschema.xml "$app/usr/share/glib-2.0/schemas/"
glib-compile-schemas "$app/usr/share/glib-2.0/schemas"
ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 "$tools/appimagetool-x86_64.AppImage" --no-appstream \
	--runtime-file "$tools/runtime-x86_64" "$app" "$out/go-pob-x86_64.AppImage"

read -ra wails3 <<<"${WAILS3:-env CGO_ENABLED=0 go run github.com/wailsapp/wails/v3/cmd/wails3@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)}"
for format in deb rpm; do
	VERSION=$version STAGE=$stage MAINTAINER=${MAINTAINER:-Vilsol} \
		"${wails3[@]}" tool package -name go-pob -format "$format" -config build/linux/nfpm.yaml -out "$stage"
done
mv "$stage/go-pob.deb" "$out/go-pob_${version}_amd64.deb"
mv "$stage/go-pob.rpm" "$out/go-pob-${version}-1.x86_64.rpm"
ls -la "$out"
