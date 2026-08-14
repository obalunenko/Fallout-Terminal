#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repository_root}"
export BUF_CACHE_DIR="${BUF_CACHE_DIR:-${TMPDIR:-/tmp}/fallout-terminal-buf-cache}"

baseline="${repository_root}/proto/compatibility-baseline.binpb"
fixtures="${repository_root}/internal/testutil/testdata/protobuf/breaking-fixtures.json"

check_baseline() {
  test -s "${baseline}"
  go tool -modfile=tools/buf/go.mod buf breaking proto --against "${baseline}"
}

check_fixture() {
  local fixture_id="$1"
  local fixture_file before after matches temporary_root candidate
  matches="$(jq --arg id "${fixture_id}" '[.[] | select(.id == $id)] | length' "${fixtures}")"
  if [[ "${matches}" -ne 1 ]]; then
    printf 'unknown or duplicate breaking fixture: %s\n' "${fixture_id}" >&2
    exit 2
  fi
  fixture_file="$(jq -r --arg id "${fixture_id}" '.[] | select(.id == $id) | .file' "${fixtures}")"
  before="$(jq -r --arg id "${fixture_id}" '.[] | select(.id == $id) | .before' "${fixtures}")"
  after="$(jq -r --arg id "${fixture_id}" '.[] | select(.id == $id) | .after' "${fixtures}")"

  temporary_root="$(mktemp -d "${TMPDIR:-/tmp}/fallout-proto-breaking.XXXXXX")"
  trap 'rm -rf "${temporary_root}"' RETURN
  cp -R proto "${temporary_root}/schema"
  candidate="${temporary_root}/schema/${fixture_file}"
  if ! grep -Fq "${before}" "${candidate}"; then
    printf 'fixture %s no longer matches %s\n' "${fixture_id}" "${fixture_file}" >&2
    exit 1
  fi
  BEFORE="${before}" AFTER="${after}" perl -0pi -e 's/\Q$ENV{BEFORE}\E/$ENV{AFTER}/' "${candidate}"

  if go tool -modfile=tools/buf/go.mod buf breaking "${temporary_root}/schema" --against "${baseline}" >/dev/null 2>&1; then
    printf 'breaking fixture %s was not rejected\n' "${fixture_id}" >&2
    exit 1
  fi
  printf 'Breaking fixture rejected: %s\n' "${fixture_id}"
  rm -rf "${temporary_root}"
  trap - RETURN
}

case "${1:-}" in
  "")
    check_baseline
    ;;
  --fixture)
    test "$#" -eq 2
    check_fixture "$2"
    ;;
  --all-fixtures)
    check_baseline
    while IFS= read -r fixture_id; do
      check_fixture "${fixture_id}"
    done < <(jq -r '.[].id' "${fixtures}")
    ;;
  *)
    printf 'usage: %s [--fixture ID|--all-fixtures]\n' "$0" >&2
    exit 2
    ;;
esac
