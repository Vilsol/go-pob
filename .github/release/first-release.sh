#!/usr/bin/env bash
set -euo pipefail
if [[ -z $(git tag --list 'v[0-9]*' --merged HEAD) ]]; then
	echo "::error::no v* tag in this branch's history: push the one-time v0.0.0 tag first (CONTRIBUTING.md, Releasing)" >&2
	exit 1
fi
