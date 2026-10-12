# shellcheck shell=bash

_gv_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
_gv_sh="$(cd "$_gv_root" && go run ./scripts/gopob gate-versions --sh)" || {
  echo "REFUSING: \`gopob gate-versions --sh\` failed; the version table is unreadable." >&2
  return 1 2>/dev/null || exit 1
}
if [ -z "$_gv_sh" ]; then
  echo "REFUSING: \`gopob gate-versions --sh\` printed nothing; sourcing that would leave every gate-version function undefined." >&2
  return 1 2>/dev/null || exit 1
fi
eval "$_gv_sh"
unset _gv_sh _gv_root
