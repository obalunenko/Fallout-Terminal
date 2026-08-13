#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repository_root}"

fixture="internal/gen/fallout/terminal/player/v1/player.pb.go"
backup="$(mktemp "${TMPDIR:-/tmp}/fallout-proto-drift.XXXXXX")"
cp "${fixture}" "${backup}"
restore_fixture() {
  cp "${backup}" "${fixture}"
  rm -f "${backup}"
}
trap restore_fixture EXIT

printf '\n// deliberate generated drift fixture\n' >> "${fixture}"
if scripts/proto-check.sh; then
  printf 'proto-check accepted a deliberate edit to checked-in generated code\n' >&2
  exit 1
fi
if ! cmp -s "${fixture}" "${backup}"; then
  printf 'proto-check did not restore the generated fixture through regeneration\n' >&2
  exit 1
fi

printf 'Checked-in generated drift is rejected before deterministic verification.\n'
