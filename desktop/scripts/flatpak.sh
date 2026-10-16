#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
manifest=$(echo build/flatpak/*.yml)
id=$(basename "$manifest" .yml)
flatpak remote-add --user --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo
flatpak-builder --user --install-deps-from=flathub --disable-rofiles-fuse --state-dir=bin/flatpak-state \
	--repo=bin/flatpak-repo --install --force-clean bin/flatpak-build "$manifest"
flatpak build-bundle --runtime-repo=https://dl.flathub.org/repo/flathub.flatpakrepo bin/flatpak-repo "bin/$id.flatpak" "$id"
ls -l "bin/$id.flatpak"
