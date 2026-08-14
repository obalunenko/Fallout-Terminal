#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repository_root}"

temporary="$(mktemp -d "${TMPDIR:-/tmp}/fallout-reproducible-build.XXXXXX")"
trap 'rm -rf "${temporary}"' EXIT HUP INT TERM

fail() {
  printf 'reproducible build check: %s\n' "$1" >&2
  exit 1
}

tree_digest() {
  local path="$1"
  [[ -e "${path}" ]] || fail "missing build output: ${path}"
  find "${path}" -type f -print0 \
    | LC_ALL=C sort -z \
    | while IFS= read -r -d '' file; do
        printf '%s\0' "${file#${path}/}"
        stat -f '%Lp' "${file}"
        shasum -a 256 "${file}"
      done \
    | shasum -a 256 \
    | awk '{print $1}'
}

tracked_state() {
  local destination="$1"
  {
    git status --porcelain=v1 --untracked-files=all
    git diff --no-ext-diff --binary -- .
  } >"${destination}"
}

run_once() {
  local run="$1"

  go run ./cmd/build build
  tree_digest internal/gen >"${temporary}/${run}.internal-gen"
  tree_digest client/gen >"${temporary}/${run}.client-gen"
  tree_digest frontend/bindings >"${temporary}/${run}.bindings"
  tree_digest client/dist >"${temporary}/${run}.client-dist"
  tree_digest frontend/dist >"${temporary}/${run}.frontend-dist"
  shasum -a 256 "build/bin/Fallout Terminal" | awk '{print $1}' >"${temporary}/${run}.native"
}

scripts/tool-modules-check.sh
scripts/wails-v3-contract-check.sh
tracked_state "${temporary}/before"

run_once first
run_once second

for output in internal-gen client-gen bindings client-dist frontend-dist native; do
  cmp "${temporary}/first.${output}" "${temporary}/second.${output}" \
    || fail "two clean runs produced different ${output} output"
done

tracked_state "${temporary}/after"
cmp "${temporary}/before" "${temporary}/after" \
  || fail 'the repeated build changed tracked or untracked repository state'

printf 'Two complete protobuf, player, binding, master, and native builds were byte-reproducible with zero repository drift.\n'
