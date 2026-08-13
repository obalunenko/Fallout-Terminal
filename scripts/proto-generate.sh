#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repository_root}"
export BUF_CACHE_DIR="${BUF_CACHE_DIR:-${TMPDIR:-/tmp}/fallout-terminal-buf-cache}"

sync_revision=false
case "${1-}" in
  '') ;;
  --sync-revision) sync_revision=true ;;
  *)
    printf 'usage: scripts/proto-generate.sh [--sync-revision]\n' >&2
    exit 1
    ;;
esac
if [[ "$#" -gt 1 ]]; then
  printf 'usage: scripts/proto-generate.sh [--sync-revision]\n' >&2
  exit 1
fi

schema_revision() {
  find proto/fallout/terminal -type f -name '*.proto' -print \
    | LC_ALL=C sort \
    | while IFS= read -r source; do shasum -a 256 "${source}"; done \
    | shasum -a 256 \
    | awk '{print $1}'
}

require_version() {
  local description="$1"
  local actual="$2"
  local expected="$3"
  if [[ "${actual}" != "${expected}" ]]; then
    printf '%s version is %s; expected %s\n' "${description}" "${actual}" "${expected}" >&2
    exit 1
  fi
}

require_version "Buf" "$(go tool buf --version)" "1.72.0"
require_version "protoc-gen-go" "$(go tool protoc-gen-go --version)" "protoc-gen-go v1.36.11"
require_version "protoc-gen-connect-go" "$(go tool protoc-gen-connect-go --version)" "1.20.0"
require_version "protoc-gen-es" "$(node -p "require('./client/node_modules/@bufbuild/protoc-gen-es/package.json').version")" "2.13.0"

actual_revision="$(schema_revision)"
if [[ "${sync_revision}" == true ]]; then
  printf '%s\n' "${actual_revision}" > proto/schema-revision.txt
fi
expected_revision="$(tr -d '[:space:]' < proto/schema-revision.txt)"
if [[ "${actual_revision}" != "${expected_revision}" ]]; then
  printf 'schema revision is %s; expected %s\nUpdate proto/schema-revision.txt with reviewed schema changes.\n' "${actual_revision}" "${expected_revision}" >&2
  exit 1
fi

if [[ "$(grep -Ec '^    out: \.\./internal/gen$' proto/buf.gen.go.yaml)" -ne 2 ]]; then
  printf 'Go generation outputs must remain isolated under internal/gen\n' >&2
  exit 1
fi
if [[ "$(grep -Ec '^    out: \.\./client/gen$' proto/buf.gen.es.yaml)" -ne 1 ]]; then
  printf 'ECMAScript generation output must remain isolated under client/gen\n' >&2
  exit 1
fi

(
  cd proto
  go tool buf generate --template buf.gen.go.yaml
  go tool buf generate --template buf.gen.es.yaml
)

if [[ "$(schema_revision)" != "${expected_revision}" ]]; then
  printf 'generation modified schema inputs\n' >&2
  exit 1
fi

printf 'Generated Go and ECMAScript contracts from schema revision %s\n' "${expected_revision}"
