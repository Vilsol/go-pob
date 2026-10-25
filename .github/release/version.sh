#!/usr/bin/env bash
set -euo pipefail
next=$1
if ! awk -v head="VERSION[$next][" 'index($0, head) == 1 { found = 1 } END { exit !found }' frontend/static/changelog.txt; then
	echo "::error::frontend/static/changelog.txt has no VERSION[$next] entry: run task changelog and commit it before releasing" >&2
	exit 1
fi
if [[ -n ${NEXT_VERSION_FILE:-} ]]; then
	printf '%s\n' "$next" >"$NEXT_VERSION_FILE"
fi
if [[ -n ${EXPECTED_VERSION:-} && $next != "$EXPECTED_VERSION" ]]; then
	echo "the release would be $next, but the artifacts were built as $EXPECTED_VERSION" >&2
	exit 1
fi
