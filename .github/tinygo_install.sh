#!/usr/bin/env bash
# tinygo_install.sh: CI's TinyGo, at the version mise.toml pins and the checksum mise.lock records (linux-x64)
set -euo pipefail

cd "$(dirname "$0")/.."
v=$(sed -n 's/^tinygo = "\(.*\)"$/\1/p' mise.toml)
sum=$(awk '/^\[tools\.tinygo\."platforms\.linux-x64"\]/ { f = 1; next } f && /^checksum/ { gsub(/"|sha256:/, "", $3); print $3; exit }' mise.lock)
if [ -z "$v" ] || [ -z "$sum" ]; then
  echo "tinygo_install.sh: no tinygo pin in mise.toml or no linux-x64 checksum in mise.lock" >&2
  exit 2
fi
dir=${RUNNER_TEMP:-$(mktemp -d)}/tinygo-$v
curl -fsSL --retry 3 -o "$dir.tar.gz" "https://github.com/tinygo-org/tinygo/releases/download/v$v/tinygo$v.linux-amd64.tar.gz"
echo "$sum  $dir.tar.gz" | sha256sum -c -
mkdir -p "$dir"
tar -xzf "$dir.tar.gz" -C "$dir"
"$dir/tinygo/bin/tinygo" version
echo "$dir/tinygo/bin" >>"${GITHUB_PATH:-/dev/null}"
