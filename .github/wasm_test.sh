#!/usr/bin/env bash

set -ex

export GOOS=js
export GOARCH=wasm

test_files=$(find -name "*_test.go")

dirs=()
for f in $test_files; do
  dirs+=("$(dirname $f)")
done

unique_dirs=$(echo "${dirs[@]}" | tr ' ' '\n' | sort -u | tr '\n' ' ')

for d in $unique_dirs; do
  echo $d
  pushd $d
  go test -c -o test.bin ./
  node "$(go env GOROOT)/misc/wasm/wasm_exec_node.js" test.bin -test.v || (rm -rf test.bin && exit 1)
  rm -rf test.bin
  popd
done
