#!/usr/bin/env bash
set -euo pipefail
out=$(realpath -m "$1")
shift
cd "$(dirname "$0")/.."
tags=gtk3
for t in "$@"; do tags+=",$t"; done
exec ./scripts/gtk-env.sh go build -tags "$tags" -o "$out" .
