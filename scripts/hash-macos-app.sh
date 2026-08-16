#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
default_app="${repository_root}/build/bin/Fallout Terminal.app"

fail() {
  printf 'hash-macos-app: %s\n' "$1" >&2
  exit 1
}

entry_state() {
  stat -f '%d:%i:%p:%z:%m:%c' "$1"
}

canonical_digest() {
  local app="$1"
  local workspace inventory_before inventory_after manifest path relative type mode before after first_digest second_digest target first_target

  [[ -d "${app}" ]] || fail "application bundle is missing: ${app}"
  [[ -r "${app}" && -x "${app}" ]] || fail "application bundle is unreadable: ${app}"

  workspace="$(mktemp -d "${TMPDIR:-/tmp}/fallout-app-hash.XXXXXX")"
  inventory_before="${workspace}/inventory-before"
  inventory_after="${workspace}/inventory-after"
  manifest="${workspace}/manifest"
  trap 'rm -rf "${workspace}"' RETURN

  find -P "${app}" -mindepth 1 -print0 | LC_ALL=C sort -z >"${inventory_before}"
  : >"${manifest}"

  while IFS= read -r -d '' path; do
    [[ -e "${path}" || -L "${path}" ]] || fail "bundle entry disappeared: ${path}"
    relative="${path#${app}/}"
    before="$(entry_state "${path}")" || fail "cannot stat bundle entry: ${relative}"
    mode="$(stat -f '%Lp' "${path}")" || fail "cannot read mode: ${relative}"

    if [[ -L "${path}" ]]; then
      type=symlink
      first_target="$(readlink "${path}")" || fail "cannot read symlink: ${relative}"
      target="$(readlink "${path}")" || fail "cannot reread symlink: ${relative}"
      [[ "${first_target}" == "${target}" ]] || fail "symlink changed while hashing: ${relative}"
      printf '%s\0%s\0%s\0%s\0' "${relative}" "${type}" "${mode}" "${target}" >>"${manifest}"
    elif [[ -f "${path}" ]]; then
      type=file
      [[ -r "${path}" ]] || fail "regular file is unreadable: ${relative}"
      first_digest="$(shasum -a 256 "${path}" | awk '{print $1}')" || fail "cannot hash file: ${relative}"
      second_digest="$(shasum -a 256 "${path}" | awk '{print $1}')" || fail "cannot rehash file: ${relative}"
      [[ "${first_digest}" == "${second_digest}" ]] || fail "file changed while hashing: ${relative}"
      printf '%s\0%s\0%s\0%s\0' "${relative}" "${type}" "${mode}" "${first_digest}" >>"${manifest}"
    elif [[ -d "${path}" ]]; then
      type=directory
      [[ -r "${path}" && -x "${path}" ]] || fail "directory is unreadable: ${relative}"
      printf '%s\0%s\0%s\0\0' "${relative}" "${type}" "${mode}" >>"${manifest}"
    else
      fail "unsupported bundle entry type: ${relative}"
    fi

    after="$(entry_state "${path}")" || fail "bundle entry disappeared: ${relative}"
    [[ "${before}" == "${after}" ]] || fail "bundle entry changed while hashing: ${relative}"
  done <"${inventory_before}"

  find -P "${app}" -mindepth 1 -print0 | LC_ALL=C sort -z >"${inventory_after}"
  cmp -s "${inventory_before}" "${inventory_after}" || fail 'bundle inventory changed while hashing'
  shasum -a 256 "${manifest}" | awk '{print $1}'

  rm -rf "${workspace}"
  trap - RETURN
}

self_test() {
  local fixture baseline repeated changed
  fixture="$(mktemp -d "${TMPDIR:-/tmp}/fallout-app-hash-self-test.XXXXXX")"
  trap 'rm -rf "${fixture}"' RETURN
  mkdir -p "${fixture}/Contents/MacOS" "${fixture}/Contents/Resources"
  printf 'binary-v1\n' >"${fixture}/Contents/MacOS/Fallout Terminal"
  printf 'resource-v1\n' >"${fixture}/Contents/Resources/resource.txt"
  printf 'target-two\n' >"${fixture}/Contents/Resources/second.txt"
  chmod 0755 "${fixture}/Contents/MacOS/Fallout Terminal"
  chmod 0644 "${fixture}/Contents/Resources/resource.txt" "${fixture}/Contents/Resources/second.txt"
  ln -s resource.txt "${fixture}/Contents/Resources/current"

  baseline="$(canonical_digest "${fixture}")"
  repeated="$(canonical_digest "${fixture}")"
  [[ "${baseline}" == "${repeated}" ]] || fail 'self-test unchanged repeat changed digest'

  printf 'resource-v2\n' >"${fixture}/Contents/Resources/resource.txt"
  changed="$(canonical_digest "${fixture}")"
  [[ "${baseline}" != "${changed}" ]] || fail 'self-test did not detect changed file content'
  printf 'resource-v1\n' >"${fixture}/Contents/Resources/resource.txt"

  chmod 0600 "${fixture}/Contents/Resources/resource.txt"
  changed="$(canonical_digest "${fixture}")"
  [[ "${baseline}" != "${changed}" ]] || fail 'self-test did not detect changed mode'
  chmod 0644 "${fixture}/Contents/Resources/resource.txt"

  rm "${fixture}/Contents/Resources/current"
  ln -s second.txt "${fixture}/Contents/Resources/current"
  changed="$(canonical_digest "${fixture}")"
  [[ "${baseline}" != "${changed}" ]] || fail 'self-test did not detect changed symlink target'
  rm "${fixture}/Contents/Resources/current"
  ln -s resource.txt "${fixture}/Contents/Resources/current"

  printf 'added\n' >"${fixture}/Contents/Resources/added.txt"
  changed="$(canonical_digest "${fixture}")"
  [[ "${baseline}" != "${changed}" ]] || fail 'self-test did not detect added entry'
  rm "${fixture}/Contents/Resources/added.txt"

  rm "${fixture}/Contents/Resources/second.txt"
  changed="$(canonical_digest "${fixture}")"
  [[ "${baseline}" != "${changed}" ]] || fail 'self-test did not detect removed entry'

  printf 'Canonical macOS app hash self-test passed.\n'
  rm -rf "${fixture}"
  trap - RETURN
}

case "${1:-}" in
  --self-test)
    [[ "$#" -eq 1 ]] || fail 'usage: scripts/hash-macos-app.sh [APP_PATH|--self-test]'
    self_test
    ;;
  '')
    canonical_digest "${default_app}"
    ;;
  *)
    [[ "$#" -eq 1 ]] || fail 'usage: scripts/hash-macos-app.sh [APP_PATH|--self-test]'
    canonical_digest "$1"
    ;;
esac
