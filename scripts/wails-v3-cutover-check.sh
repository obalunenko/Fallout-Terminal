#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  printf 'wails-v3 cutover check: %s\n' "$1" >&2
  return 1
}

scan_tree() {
  local root="$1"
  local matches

  for obsolete in wails.json build/darwin/postbuild.sh production_resources_bindings.go; do
    [[ ! -e "${root}/${obsolete}" ]] || { fail "obsolete active artifact remains: ${obsolete}"; return 1; }
  done
  if find "${root}" -type d \( -name .git -o -name node_modules \) -prune -o -type f \( -name Taskfile.yml -o -name Taskfile.yaml \) -print | grep -q .; then
    fail 'Taskfile orchestration is present'
    return 1
  fi

  matches="$(find "${root}" \( -path '*/.git' -o -path '*/node_modules' -o -path '*/specs' \) -prune -o -type f -name '*.go' ! -name '*_test.go' \
    -exec grep -EnH 'github\.com/wailsapp/wails/v2|WAILS_V2|USE_WAILS_V2|legacyWails|dual.?runtime' {} + || true)"
  [[ -z "${matches}" ]] || { printf '%s\n' "${matches}" >&2; fail 'active Go source contains v2 or dual-runtime code'; return 1; }

  if grep -En 'github\.com/wailsapp/wails/v2' "${root}/go.mod" "${root}/go.sum"; then
    fail 'application module still resolves Wails v2'
    return 1
  fi

  matches="$(grep -ERIn 'frontend/(overseer/)?wailsjs|window\.(go|runtime)|electronAPI|WAILS_V2|USE_WAILS_V2|legacyWails|dual.?runtime' \
    "${root}/frontend/overseer/src" "${root}/frontend/overseer/bindings" "${root}/frontend/overseer/dist" 2>/dev/null || true)"
  [[ -z "${matches}" ]] || { printf '%s\n' "${matches}" >&2; fail 'frontend source/generated/bundle contains a v2 global or dual-runtime fallback'; return 1; }

  matches="$(find "${root}/README.md" "${root}/scripts" "${root}/.github/workflows" -type f \
    ! -name 'wails-v3-cutover-check.sh' ! -name 'wails-bindings-check.sh' ! -name 'verify-macos-app.sh' \
    ! -name 'wails-v3-contract-check.sh' ! -name 'tool-modules-check.sh' \
    -exec grep -EnH \
      'go[[:space:]]+install[[:space:]]+github\.com/wailsapp/wails|(^|[[:space:]`;&|])wails[[:space:]]+(dev|build|generate)([[:space:]]|$)|@wailsio/runtime.*(latest|\^|~|\*)|github\.com/wailsapp/wails/v3@latest' \
      {} + 2>/dev/null || true)"
  [[ -z "${matches}" ]] || { printf '%s\n' "${matches}" >&2; fail 'active command/documentation uses v2, global, or floating Wails resolution'; return 1; }

  [[ -f "${root}/specs/001-wails-v2-migration/spec.md" ]] || { fail 'historical Wails v2 spec is missing'; return 1; }
  [[ -f "${root}/docs/wails-migration-rollback.md" ]] || { fail 'historical Electron-to-Wails rollback record is missing'; return 1; }
  grep -Eq 'specs/006-wails-v3-migration/quickstart\.md' "${root}/README.md" || { fail 'README does not link the active Wails v3 quickstart'; return 1; }
  grep -Eq 'docs/wails-v3-migration-rollback\.md' "${root}/README.md" || { fail 'README does not link the active Wails v3 rollback record'; return 1; }
  grep -Eqi 'histor' "${root}/README.md" || { fail 'README does not identify earlier migration records as history'; return 1; }
}

self_test() {
  local fixture
  fixture="$(mktemp -d "${TMPDIR:-/tmp}/fallout-cutover-check.XXXXXX")"
  trap 'rm -rf "${fixture}"' RETURN
  mkdir -p "${fixture}/build/darwin" "${fixture}/frontend/overseer/src" "${fixture}/frontend/overseer/bindings" "${fixture}/frontend/overseer/dist" \
    "${fixture}/internal/app" "${fixture}/scripts" "${fixture}/.github/workflows" \
    "${fixture}/specs/001-wails-v2-migration" "${fixture}/docs"
  printf 'module example.test/app\n\ngo 1.26\n\nrequire github.com/wailsapp/wails/v3 v3.0.0-beta.10\n' >"${fixture}/go.mod"
  : >"${fixture}/go.sum"
  printf 'package main\nimport _ "github.com/wailsapp/wails/v3/pkg/application"\n' >"${fixture}/main.go"
  printf 'export const ready = true;\n' >"${fixture}/frontend/overseer/src/app.js"
  printf 'export const generated = true;\n' >"${fixture}/frontend/overseer/bindings/service.js"
  printf '<!doctype html>\n' >"${fixture}/frontend/overseer/dist/index.html"
  printf 'Active: specs/006-wails-v3-migration/quickstart.md and docs/wails-v3-migration-rollback.md. Earlier records are historical evidence.\n' >"${fixture}/README.md"
  printf '# Historical v2 spec\n' >"${fixture}/specs/001-wails-v2-migration/spec.md"
  printf '# Historical rollback\n' >"${fixture}/docs/wails-migration-rollback.md"
  scan_tree "${fixture}"

  printf 'package app\nimport _ "github.com/wailsapp/wails/v2"\n' >"${fixture}/internal/app/app.go"
  if scan_tree "${fixture}" >/dev/null 2>&1; then fail 'self-test accepted an active v2 Go import'; return 1; fi
  printf 'package app\n' >"${fixture}/internal/app/app.go"

  printf '{}\n' >"${fixture}/wails.json"
  if scan_tree "${fixture}" >/dev/null 2>&1; then fail 'self-test accepted wails.json'; return 1; fi
  rm "${fixture}/wails.json"

  printf 'window.go.main.App();\n' >"${fixture}/frontend/overseer/src/app.js"
  if scan_tree "${fixture}" >/dev/null 2>&1; then fail 'self-test accepted a generated v2 global'; return 1; fi
  printf 'export const ready = true;\n' >"${fixture}/frontend/overseer/src/app.js"

  printf 'wails build\n' >>"${fixture}/README.md"
  if scan_tree "${fixture}" >/dev/null 2>&1; then fail 'self-test accepted a bare v2 Wails command'; return 1; fi

  printf 'Wails v3 cutover scan self-test passed.\n'
}

case "${1:-}" in
  --self-test)
    [[ "$#" -eq 1 ]] || { fail 'usage: scripts/wails-v3-cutover-check.sh [--self-test]'; exit 1; }
    self_test
    ;;
  '')
    scan_tree "${repository_root}"
    "${repository_root}/scripts/tool-modules-check.sh"
    "${repository_root}/scripts/wails-bindings-check.sh"
    git -C "${repository_root}" diff --exit-code -- specs/001-wails-v2-migration docs/wails-migration-rollback.md
    if go -C "${repository_root}" list -m all | grep -En '^github\.com/wailsapp/wails/v2([[:space:]]|$)'; then
      fail 'resolved module graph still contains Wails v2'
      exit 1
    fi
    printf 'Wails v3 cutover scan passed: no active v2, dual-runtime, floating-tool, generated-global, dependency, bundle, script, CI, or operating-document surface remains.\n'
    ;;
  *)
    fail 'usage: scripts/wails-v3-cutover-check.sh [--self-test]'
    exit 1
    ;;
esac
