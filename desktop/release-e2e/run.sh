#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
pkg=$(realpath "$1")
shift
cases=("$@")
[[ ${#cases[@]} -gt 0 ]] || cases=(appimage appimage-bare binary-no-tls binary-no-schemas deb rpm)
image=go-pob-desktop-run:ubuntu-24.04
docker build -q -t "$image" desktop/release-e2e >/dev/null

no_tls='rm /usr/lib/x86_64-linux-gnu/gio/modules/libgiognutls.so'
no_schemas='rm /usr/share/glib-2.0/schemas/org.gtk.Settings.*.xml && glib-compile-schemas /usr/share/glib-2.0/schemas'
headless='printf "window.__RELEASE = { chooser: true };\n" >/tmp/probe.js && cat /src/desktop/release-e2e/probe.js >>/tmp/probe.js && /src/desktop/scripts/run-headless.sh'
failed=0
for c in "${cases[@]}"; do
	run_image=$image
	case $c in
	appimage) script="$headless /pkg/go-pob-x86_64.AppImage /tmp/probe.js 60 -- --appimage-extract-and-run" ;;
	appimage-bare) script="$no_tls && $no_schemas && $headless /pkg/go-pob-x86_64.AppImage /tmp/probe.js 60 -- --appimage-extract-and-run" ;;
	binary-no-tls) script="$no_tls && $headless /pkg/go-pob-linux-amd64 /tmp/probe.js 60" ;;
	binary-no-schemas) script="$no_schemas && $headless /pkg/go-pob-linux-amd64 /tmp/probe.js 60" ;;
	deb)
		script='apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq /pkg/go-pob_*_amd64.deb >/dev/null &&
			echo "xdg-mime x-scheme-handler/pob: $(xdg-mime query default x-scheme-handler/pob)" &&
			echo "xdg-mime application/xml: $(xdg-mime query default application/xml)" && '"$headless /usr/bin/go-pob /tmp/probe.js 60"
		;;
	rpm)
		run_image=fedora:42
		script='dnf install -y -q --setopt=install_weak_deps=False /pkg/go-pob-*.x86_64.rpm xorg-x11-server-Xvfb dbus-daemon xdg-utils >/dev/null &&
			echo "xdg-mime x-scheme-handler/pob: $(xdg-mime query default x-scheme-handler/pob)" && '"$headless /usr/bin/go-pob /tmp/probe.js 60"
		;;
	*)
		echo "run.sh: unknown case $c" >&2
		exit 2
		;;
	esac
	echo "=== $c"
	log=$(docker run --rm --network host -v "$pkg:/pkg:ro" -v "$PWD:/src:ro" "$run_image" bash -c "$script" 2>&1 |
		grep -E 'probe: |run-headless|xdg-mime|GLib|Gtk|TLS|abort|rror' | sed 's/^[0-9/]* [0-9:]* probe: //' | awk '!seen[$0]++' || true)
	echo "$log"
	ok=0
	grep -q 'file chooser open, process alive' <<<"$log" && ! grep -q '"fail"' <<<"$log" && ok=1
	case $c in
	binary-no-tls) ok=0 && grep -q '"step":"boot"' <<<"$log" && ! grep -q '"step":"new build"' <<<"$log" && ok=1 ;;
	binary-no-schemas) ok=0 && grep -q '"step":"new build"' <<<"$log" && ! grep -q 'process alive' <<<"$log" && ok=1 ;;
	esac
	echo "=== $c: $([[ $ok == 1 ]] && echo as expected || echo UNEXPECTED)"
	[[ $ok == 1 ]] || failed=1
done
exit $failed
