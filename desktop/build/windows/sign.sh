#!/usr/bin/env bash
set -euo pipefail
osslsigncode sign -pkcs12 "$WINDOWS_CERT_PFX" -pass "$WINDOWS_CERT_PASSWORD" -h sha256 -n go-pob \
	-t "${TIMESTAMP_URL:-http://timestamp.digicert.com}" -in "$1" -out "$1.signed"
mv "$1.signed" "$1"
