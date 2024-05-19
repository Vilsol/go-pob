#!/usr/bin/env bash

set -ex

export GOOS=js
export GOARCH=wasm

test_files=$(find . -name "*_test.go")

dirs=()
for f in $test_files; do
  dirs+=("$(dirname $f)")
done

unique_dirs=$(echo "${dirs[@]}" | tr ' ' '\n' | sort -u | tr '\n' ' ')

WASM_DIR=$(mktemp -d)
curl -o $WASM_DIR/wasm_exec.js https://raw.githubusercontent.com/golang/go/$(go env GOVERSION)/misc/wasm/wasm_exec.js
curl -o $WASM_DIR/wasm_exec_node.js https://raw.githubusercontent.com/golang/go/$(go env GOVERSION)/misc/wasm/wasm_exec_node.js

for d in $unique_dirs; do
  echo $d
  pushd $d
  go test -c -o test.bin ./
  NODE_BIN=$(which node)
  env --ignore-environment $NODE_BIN "$WASM_DIR/wasm_exec_node.js" test.bin -test.v || (rm -rf test.bin && exit 1)
  rm -rf test.bin
  popd
done

rm -rf $WASM_DIR
