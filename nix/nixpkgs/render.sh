#!/usr/bin/env bash
set -euo pipefail
version=$1
hash=${2:-}
pkg=$(dirname "$0")/../package.nix
if [[ -z $hash ]]; then
	hash=$(nix flake prefetch --json "github:Vilsol/go-pob/v$version" | sed -n 's/.*"hash":"\([^"]*\)".*/\1/p')
	[[ -n $hash ]] || { echo "render: no hash for v$version" >&2; exit 1; }
fi
if [[ $hash == fake ]]; then hash_expr=lib.fakeHash; else hash_expr="\"$hash\""; fi

awk -v version="$version" -v hash="$hash_expr" '
/^  src,$/ || /^  version,$/ { next }
/^  buildGoModule,$/ { print; print "  fetchFromGitHub,"; next }
/^  inherit src version;$/ {
	print "  version = \"" version "\";"
	print ""
	print "  src = fetchFromGitHub {"
	print "    owner = \"Vilsol\";"
	print "    repo = \"go-pob\";"
	print "    tag = \"v${finalAttrs.version}\";"
	print "    hash = " hash ";"
	print "  };"
	next
}
{ print }
' "$pkg"
