#!/usr/bin/env bash
set -euo pipefail
next=$1
if [[ -n ${NEXT_VERSION_FILE:-} ]]; then
	printf '%s\n' "$next" >"$NEXT_VERSION_FILE"
fi
if [[ -n ${EXPECTED_VERSION:-} && $next != "$EXPECTED_VERSION" ]]; then
	echo "the release would be $next, but the artifacts were built as $EXPECTED_VERSION" >&2
	exit 1
fi
