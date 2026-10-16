#!/usr/bin/env bash

set -euo pipefail

selection=(
  "./calculator TestConcurrentEvaluationMatchesSerial TestSharedCacheConcurrentCalculations"
  "./data/raw TestDeferredTablesLoadOnFirstUse"
)

failed=()
for line in "${selection[@]}"; do
  read -r pkg tests <<<"$line"
  log=$(mktemp)
  status=0
  go test -race -v -timeout 30m -run "^($(echo "$tests" | tr ' ' '|'))\$" "$pkg" >"$log" 2>&1 || status=$?
  grep -E '^(=== RUN|--- |ok|FAIL|PASS)|WARNING: DATA RACE|^panic|^fatal' "$log" | grep -v '^=== RUN.*/' || true
  if [ "$status" -ne 0 ]; then
    cat "$log"
    failed+=("$line")
  fi
  for test in $tests; do
    if ! grep -q "^--- PASS: ${test} " "$log"; then
      echo "REFUSING: $test did not run and pass in $pkg (renamed, removed or skipped?)"
      failed+=("$pkg $test")
    fi
  done
  rm -f "$log"
done

if [ ${#failed[@]} -gt 0 ]; then
  printf 'FAILED: %s\n' "${failed[@]}"
  exit 1
fi
