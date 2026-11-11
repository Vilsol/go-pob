#!/usr/bin/env bash
# tinygo_build.sh <wasm|wasm2> <out.wasm> [tinygo build flags...]: the TinyGo build of a wasm main, as every build of
# it is made. -stack-size: PoE1's calculation overflows the default goroutine stack, and PoE2's pages (Skills, Party)
# overflow it on larger builds; every awaited promise costs two stacks of heap, so PoE2 takes a smaller one
set -euo pipefail

main=$1 out=$2
shift 2
case $main in
  wasm) stack=(-stack-size=4MB) ;;
  wasm2) stack=(-stack-size=1MB) ;;
  *) echo "tinygo_build.sh: unknown main $main (wasm or wasm2)" >&2; exit 2 ;;
esac
cd "$(dirname "$0")/.."
tinygo build -target wasm -no-debug -opt=z "${stack[@]}" "$@" -o "$out" "./$main"
