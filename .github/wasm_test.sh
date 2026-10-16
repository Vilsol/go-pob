#!/usr/bin/env bash

set -ex

export GOOS=js
export GOARCH=wasm

test_files=$(find . -name "*_test.go" -not -path './scripts/*' -not -path './desktop/*' -not -path '*/testdata/*' -not -path '*/node_modules/*' | grep -v 'go-pob-data')

dirs=()
for f in $test_files; do
  dirs+=("$(dirname $f)")
done

unique_dirs=$(echo "${dirs[@]}" | tr ' ' '\n' | sort -u | tr '\n' ' ')

failed=()
for d in $unique_dirs; do
  echo $d
  pushd $d
  go test -c -o test.bin ./
  NODE_BIN=$(which node)
  env --ignore-environment $NODE_BIN "$(go env GOROOT)/lib/wasm/wasm_exec_node.js" test.bin -test.v -test.timeout=60m || failed+=("$d")
  rm -rf test.bin
  popd
done

if [ ${#failed[@]} -gt 0 ]; then
  echo "FAILED: ${failed[*]}"
  exit 1
fi
