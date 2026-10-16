#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
old=$(basename "$(echo ./*.yml)" .yml)
new=$1
for f in "$old".*; do git mv "$f" "$new${f#"$old"}"; done
sed -i "s/$old/$new/g" "$new".*
