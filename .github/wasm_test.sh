#!/usr/bin/env bash

set -euo pipefail

export GOOS=js
export GOARCH=wasm

selection=(
  "./wasm/exposition *"
  "./calculator TestWhatIfsLetTheGCMarkWorkerRun"
  "./calculator TestTornadoShotSecondaryProjectilesReachTheOracle CORPUS=true"
  "./calculator TestElusiveRampMatchesOracle CORPUS=true"
  "./calculator TestLifeFlaskChargesReachesTheOracle CORPUS=true"
  "./calculator TestEnergyShieldFromHelmetReachesTheOracle CORPUS=true"
  "./calculator TestPartyBuffsReachTheOracle CORPUS=true"
)

node_bin=$(which node)
wasm_exec="$(go env GOROOT)/lib/wasm/wasm_exec_node.js"
bins=$(mktemp -d)
trap 'rm -rf "$bins"' EXIT

failed=()
for line in "${selection[@]}"; do
  read -r dir test envs <<<"$line"
  bin="$bins/$(echo "$dir" | tr '/.' '__').wasm"
  [ -f "$bin" ] || go test -c -o "$bin" "$dir"
  if [ ! -f "$bin" ]; then
    echo "REFUSING: $dir built no test binary"
    failed+=("$line")
    continue
  fi

  args=(-test.v -test.timeout=30m)
  if [ "$test" != "*" ]; then
    args+=(-test.run "^${test}\$")
  fi
  log="$bins/run.log"
  echo "::group::$dir $test"
  status=0
  # shellcheck disable=SC2086 # envs is a list of VAR=value words
  (cd "$dir" && env --ignore-environment $envs "$node_bin" "$wasm_exec" "$bin" "${args[@]}") >"$log" 2>&1 || status=$?
  cat "$log"
  echo "::endgroup::"

  if [ "$status" -ne 0 ]; then
    failed+=("$line")
  elif [ "$test" = "*" ] && ! grep -q '^--- PASS: ' "$log"; then
    echo "REFUSING: no test in $dir passed"
    failed+=("$line")
  elif [ "$test" != "*" ] && ! grep -q "^--- PASS: ${test} " "$log"; then
    echo "REFUSING: $test did not run and pass in $dir (renamed, removed or skipped?)"
    failed+=("$line")
  fi
done

if [ ${#failed[@]} -gt 0 ]; then
  printf 'FAILED: %s\n' "${failed[@]}"
  exit 1
fi
