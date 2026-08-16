#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  printf 'secret leak check: %s\n' "$1" >&2
  return 1
}

list_active_files() {
  if git -C "$repository_root" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git -C "$repository_root" ls-files -co --exclude-standard -z
  else
    find "$repository_root" -type f -print0
  fi
}

scan_canary_file() {
  local canary_file="$1"
  local scan_root="${2:-$repository_root}"
  local file relative
  local leaked=0

  while IFS= read -r -d '' file; do
    if [[ "$file" = /* ]]; then
      relative="${file#"$scan_root"/}"
    else
      relative="$file"
      file="$scan_root/$file"
    fi
    case "$relative" in
      .git/*|specs/*|node_modules/*|frontend/node_modules/*|client/node_modules/*|tests/browser/node_modules/*|scripts/secret-leak-check.sh)
        continue
        ;;
    esac
    if grep -IlF -f "$canary_file" "$file" >/dev/null 2>&1; then
      printf 'secret leak check: canary detected in %s (value redacted)\n' "$relative" >&2
      leaked=1
    fi
  done < <(
    if [[ "$scan_root" == "$repository_root" ]]; then
      list_active_files
    else
      find "$scan_root" -type f -print0
    fi
  )
  [[ "$leaked" == 0 ]]
}

check_public_contracts() {
  local forbidden
  forbidden="$(grep -ERIl '(^|[[:space:]_])(authtoken|account_token|player_password|provider_token|credential|secret)[[:space:]]*=' \
    "$repository_root/proto/fallout/terminal/player" \
    "$repository_root/proto/fallout/terminal/persistence" 2>/dev/null || true)"
  [[ -z "$forbidden" ]] || fail 'public or persistent protobuf contract contains a secret field'

  grep -Fq 'optional string generated_password = 3;' \
    "$repository_root/proto/fallout/terminal/private/v1/public_access.proto" ||
    fail 'one-time generated-password result contract is missing'
  grep -Fq 'string replacement_provider_token = 5;' \
    "$repository_root/proto/fallout/terminal/private/v1/public_access.proto" ||
    fail 'narrow provider-token mutation input is missing'
  grep -Fq 'string replacement_player_password = 7;' \
    "$repository_root/proto/fallout/terminal/private/v1/public_access.proto" ||
    fail 'narrow player-password mutation input is missing'
}

check_active_sources() {
  local suspicious
  suspicious="$(git -C "$repository_root" grep -IlE \
    '(ngrok[_-]?authtoken|provider[_-]?token|player[_-]?password)[[:space:]]*[:=][[:space:]]*["'"'][^"'"']{8,}["'"']' \
    -- ':!specs/**' ':!**/*_test.go' ':!tests/browser/**' ':!scripts/secret-leak-check.sh' 2>/dev/null || true)"
  [[ -z "$suspicious" ]] || {
    while IFS= read -r file; do
      [[ -n "$file" ]] && printf 'secret leak check: suspicious literal in %s (value redacted)\n' "$file" >&2
    done <<<"$suspicious"
    return 1
  }

  if grep -E '(localStorage|sessionStorage).*(providerToken|playerPassword|generatedPassword)|(providerToken|playerPassword|generatedPassword).*(localStorage|sessionStorage)' \
    "$repository_root/frontend/src/"*.js >/dev/null 2>&1; then
    fail 'master frontend persists a public-access secret in browser storage'
  fi

  grep -Fq 'replacementProviderToken' "$repository_root/frontend/src/master.js" ||
    fail 'master secret mutation flow is not implemented yet'
  grep -Fq 'generatedPassword' "$repository_root/frontend/src/master.js" ||
    fail 'one-time generated-password presentation is not implemented yet'
}

check_generated_password_scope() {
  local file unexpected=''

  while IFS= read -r file; do
    [[ -n "$file" ]] || continue
    case "$file" in
      app.go|app_contract.go|internal/tunnel/secret.go|proto/fallout/terminal/private/v1/public_access.proto|internal/gen/fallout/terminal/private/v1/public_access.pb.go|frontend/src/desktop-api.js|frontend/src/master.js|frontend/src/index.html|frontend/src/master.css|frontend/bindings/github.com/obalunenko/Fallout-Terminal/models.js)
        ;;
      *)
        unexpected="${unexpected}${file}"$'\n'
        ;;
    esac
  done < <(git -C "$repository_root" grep -IlE 'generated[_-]?password|generatedPassword|GeneratedPassword' \
    -- ':!specs/**' ':!*_test.go' ':!**/*_test.go' ':!tests/**' ':!scripts/secret-leak-check.sh' 2>/dev/null || true)

  if [[ -n "$unexpected" ]]; then
    while IFS= read -r file; do
      [[ -n "$file" ]] && printf 'secret leak check: generated-password surface outside one-time flow: %s\n' "$file" >&2
    done <<<"$unexpected"
    return 1
  fi

  if grep -ERIl 'generated[_-]?password|generatedPassword|GeneratedPassword' \
    "$repository_root/proto/fallout/terminal/player" \
    "$repository_root/proto/fallout/terminal/persistence" \
    "$repository_root/internal/session" \
    "$repository_root/internal/playerconfig" >/dev/null 2>&1; then
    fail 'generated password escaped into a public or persistent contract'
    return 1
  fi

  grep -Fq "generatedPasswordValue.textContent = '';" "$repository_root/frontend/src/master.js" ||
    fail 'one-time generated-password presentation is not cleared on completion'
  grep -Fq 'btnCopyGeneratedPassword.onclick = null;' "$repository_root/frontend/src/master.js" ||
    fail 'one-time generated-password callback retains its value after completion'
  grep -Fq 'afterGenerated.generatedPassword).toBeUndefined()' \
    "$repository_root/tests/browser/desktop-api.spec.mjs" ||
    fail 'browser contract does not prove generated-password non-readback'
  grep -Fq "locator('#generatedPasswordValue')).toHaveText('')" \
    "$repository_root/tests/browser/public-access-settings.spec.mjs" ||
    fail 'browser journey does not prove one-time presentation clearing'
}

check_development_override_scope() {
  local override="$repository_root/internal/tunnel/test_override.go"
  [[ -f "$override" ]] || { fail 'development/test public-access override is missing'; return 1; }

  local name
  for name in \
    FALLOUT_NGROK_AUTHTOKEN \
    FALLOUT_NGROK_RESERVED_DOMAIN \
    FALLOUT_PUBLIC_TEST_USERNAME \
    FALLOUT_PUBLIC_TEST_PASSWORD; do
    grep -Fq "$name" "$override" || { fail "approved development override name is missing: $name"; return 1; }
  done
  if grep -Fq 'os.Environ' "$override"; then
    fail 'development override enumerates the process environment'
    return 1
  fi
  grep -Fq 'publicAccessStoresForProfile(publicSettings, publicSecrets, packaged, os.LookupEnv)' \
    "$repository_root/main.go" || fail 'root composition does not explicitly gate the development override from packaged production'
}

check_tree() {
  local canary_file="${1:-}"
  check_public_contracts
  check_active_sources
  check_generated_password_scope
  check_development_override_scope
  if [[ -n "$canary_file" ]]; then
    [[ -s "$canary_file" ]] || { fail 'canary file is missing or empty'; return 1; }
    scan_canary_file "$canary_file"
  fi
  printf 'Secret-bearing fields remain confined to narrow private inputs/results; no forbidden leak was detected.\n'
}

self_test() {
  local fixture_root canary_file surface
  local -a surfaces=(
    'errors/start-error.txt'
    'logs/application.log'
    'events/public-access-status.json'
    'protobuf/private-result.bin'
    'config/public-access.json'
    'Application Support/Fallout Terminal/public-access.json'
    'sessions/session-v1.json'
    'player-config/player-config-v1.json'
    'args/process.args'
    'fixtures/public-access.json'
    'frontend/local-storage.json'
    'package/Fallout Terminal.app/Contents/Resources/diagnostic.json'
  )
  fixture_root="$(mktemp -d)"
  trap 'rm -rf "$fixture_root"' RETURN
  canary_file="$fixture_root/.canaries"
  mkdir -p "$fixture_root/surfaces"
  {
    LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 96 || true
    printf '\n'
    LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 112 || true
    printf '\n'
    LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 128 || true
    printf '\n'
  } >"$canary_file"

  for surface in "${surfaces[@]}"; do
    mkdir -p "$fixture_root/surfaces/$(dirname "$surface")"
    cp "$canary_file" "$fixture_root/surfaces/$surface"
    if scan_canary_file "$canary_file" "$fixture_root/surfaces" >/dev/null 2>&1; then
      fail "self-test accepted a canary on $surface"
      return 1
    fi
    : >"$fixture_root/surfaces/$surface"
  done

  scan_canary_file "$canary_file" "$fixture_root/surfaces"
  printf 'Secret leak detector self-test passed across errors, logs, events, protobuf, config, Application Support, session, player-config, args, fixtures, frontend, and packaged resources.\n'
}

case "${1:-}" in
  '')
    check_tree
    ;;
  --canary-file)
    [[ "$#" == 2 ]] || { fail 'usage: secret-leak-check.sh --canary-file PATH'; exit 2; }
    check_tree "$2"
    ;;
  --self-test)
    self_test
    ;;
  *)
    fail 'usage: secret-leak-check.sh [--canary-file PATH|--self-test]'
    ;;
esac
