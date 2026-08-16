#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
module_file="$repository_root/go.mod"
notice_file="$repository_root/THIRD_PARTY_NOTICES.md"

readonly ngrok_module='golang.ngrok.com/ngrok/v2'
readonly ngrok_version='v2.1.4'
readonly keychain_module='github.com/keybase/go-keychain'
readonly keychain_version='v0.0.1'

fail() {
  printf 'dependency/license check: %s\n' "$1" >&2
  return 1
}

require_exact_pin() {
  local module="$1"
  local expected="$2"
  local actual

  actual="$(awk -v wanted="$module" '$1 == wanted { print $2 }' "$module_file")"
  [[ "$actual" == "$expected" ]] || {
    fail "$module must be pinned exactly at $expected (found ${actual:-missing})"
    return 1
  }
}

resolved_runtime_modules() {
  go list -deps \
    -f '{{with .Module}}{{if .Version}}{{.Path}}@{{.Version}}{{end}}{{end}}' \
    "$ngrok_module" "$keychain_module" | sed '/^$/d' | sort -u
}

check_notices() {
  local module_version
  local missing=0
  local runtime_list="$1"

  [[ -f "$notice_file" ]] || {
    fail 'THIRD_PARTY_NOTICES.md is missing'
    return 1
  }

  while IFS= read -r module_version; do
    [[ -n "$module_version" ]] || continue
    if ! grep -Fq -- "- $module_version —" "$notice_file"; then
      printf 'dependency/license check: missing reviewed notice inventory entry for %s\n' \
        "$module_version" >&2
      missing=1
    fi
  done <"$runtime_list"

  [[ "$missing" == 0 ]] || return 1
  grep -Fq '## ngrok-go' "$notice_file" || fail 'missing ngrok-go notice text'
  grep -Fq '## go-keychain' "$notice_file" || fail 'missing go-keychain notice text'
}

check_tree() {
  local scratch
  scratch="$(mktemp -d)"
  trap 'rm -rf "$scratch"' RETURN

  [[ -f "$module_file" ]] || { fail 'go.mod is missing'; return 1; }
  if grep -En '(@latest|[[:space:]]latest([[:space:]]|$))' "$module_file"; then
    fail 'go.mod contains a floating dependency version'
    return 1
  fi

  require_exact_pin "$ngrok_module" "$ngrok_version"
  require_exact_pin "$keychain_module" "$keychain_version"

  resolved_runtime_modules >"$scratch/runtime-modules"
  check_notices "$scratch/runtime-modules"

  printf 'Embedded public-access runtime dependency inventory:\n'
  sed 's/^/  /' "$scratch/runtime-modules"
  printf 'Dependency pins and reviewed notices are complete.\n'
}

case "${1:-}" in
  '')
    check_tree
    ;;
  --list)
    require_exact_pin "$ngrok_module" "$ngrok_version"
    require_exact_pin "$keychain_module" "$keychain_version"
    scratch="$(mktemp -d)"
    trap 'rm -rf "$scratch"' EXIT
    resolved_runtime_modules
    ;;
  *)
    fail "unknown argument: $1"
    ;;
esac
