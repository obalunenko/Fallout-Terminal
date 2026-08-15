#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app_path="${1:-${repository_root}/build/bin/Fallout Terminal.app}"
executable_path="${app_path}/Contents/MacOS/Fallout Terminal"
plist_path="${app_path}/Contents/Info.plist"
resources_path="${app_path}/Contents/Resources"
notices_path="${resources_path}/THIRD_PARTY_NOTICES.md"
expected_entitlements="${repository_root}/build/darwin/entitlements.plist"

fail() {
  printf 'verify-macos-app: %s\n' "$1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

[[ "$#" -le 1 ]] || fail 'usage: scripts/verify-macos-app.sh [APP_PATH]'
[[ "$(uname -s)" == Darwin ]] || fail 'verification requires macOS'
for command in codesign lipo otool plutil rg shasum strings; do
  require_command "${command}"
done

[[ -d "${app_path}" ]] || fail "application bundle is missing: ${app_path}"
[[ -x "${executable_path}" ]] || fail "application executable is missing: ${executable_path}"
[[ -f "${plist_path}" ]] || fail "Info.plist is missing: ${plist_path}"
[[ -d "${resources_path}" ]] || fail "Resources directory is missing: ${resources_path}"

[[ "$(lipo -archs "${executable_path}")" == arm64 ]] || fail 'application executable must contain only arm64'
plutil -lint "${plist_path}" >/dev/null || fail 'Info.plist is invalid'
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "${plist_path}")" == com.vaulttec.fallout-terminal ]] || fail 'bundle identifier is incorrect'
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "${plist_path}")" == 'Fallout Terminal' ]] || fail 'bundle executable metadata is incorrect'
[[ "$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "${plist_path}")" == 13.0 ]] || fail 'Info.plist minimum macOS is not 13.0'

binary_minimum="$(otool -l "${executable_path}" | awk '$1 == "cmd" && $2 == "LC_BUILD_VERSION" { in_build=1; next } in_build && $1 == "minos" { print $2; exit }')"
[[ "${binary_minimum}" == 13.0 ]] || fail "binary minimum macOS is ${binary_minimum:-missing}, want 13.0"

[[ -s "${resources_path}/icon.icns" ]] || fail 'application icon is missing or empty'
[[ -s "${resources_path}/sessions/demo.json" ]] || fail 'bundled demo is missing or empty'
cmp -s "${repository_root}/sessions/demo.json" "${resources_path}/sessions/demo.json" || fail 'bundled demo differs from the reviewed source resource'
[[ "$(stat -f '%Lp' "${resources_path}/sessions/demo.json")" == 444 ]] || fail 'bundled demo must be read-only mode 0444'
[[ -s "${notices_path}" ]] || fail 'third-party notices are missing or empty'
cmp -s "${repository_root}/THIRD_PARTY_NOTICES.md" "${notices_path}" || fail 'packaged third-party notices differ from the reviewed source inventory'
[[ "$(stat -f '%Lp' "${notices_path}")" == 444 ]] || fail 'third-party notices must be read-only mode 0444'

linked_frameworks="$(otool -L "${executable_path}")"
grep -Fq '/System/Library/Frameworks/Security.framework/' <<<"${linked_frameworks}" || fail 'Security.framework linkage is missing'
grep -Fq '/System/Library/Frameworks/CoreFoundation.framework/' <<<"${linked_frameworks}" || fail 'CoreFoundation.framework linkage is missing'

bundled_provider="$(find "${app_path}/Contents" -type f \( -iname 'ngrok' -o -iname 'ngrok.exe' -o -iname 'ngrok-*' \) -print -quit 2>/dev/null || true)"
[[ -z "${bundled_provider}" ]] || fail 'application bundle contains a provider executable'
if strings "${executable_path}" | rg -q 'NGROK_BIN|fallout-terminal\.ngrok\.app|tunnel-guardian'; then
  fail 'application executable contains a legacy CLI, PATH, guardian, or shared-domain runtime marker'
fi

entitlements_dump="$(mktemp "${TMPDIR:-/tmp}/fallout-entitlements.XXXXXX")"
trap 'rm -f "${entitlements_dump}"' EXIT HUP INT TERM
codesign -d --entitlements :- "${app_path}" >"${entitlements_dump}" 2>/dev/null || fail 'cannot read application entitlements'
plutil -lint "${entitlements_dump}" >/dev/null || fail 'signed entitlements are invalid'
for entitlement in com.apple.security.cs.allow-jit com.apple.security.network.client com.apple.security.network.server; do
  [[ "$(/usr/libexec/PlistBuddy -c "Print :${entitlement}" "${entitlements_dump}")" == true ]] || fail "signed entitlement is missing: ${entitlement}"
  [[ "$(/usr/libexec/PlistBuddy -c "Print :${entitlement}" "${expected_entitlements}")" == true ]] || fail "source entitlement is missing: ${entitlement}"
done

for distribution in "${repository_root}/frontend/dist" "${repository_root}/client/dist"; do
  [[ -s "${distribution}/index.html" ]] || fail "offline distribution is incomplete: ${distribution}"
  if rg -n 'https://cdn\.|http://localhost:|http://127\.0\.0\.1:5173|@vite/client|frontend/wailsjs|window\.(go|runtime)' "${distribution}"; then
    fail "offline distribution contains a development, CDN, or legacy runtime dependency: ${distribution}"
  fi
done

GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/fallout-terminal-go-cache}" \
  go test "${repository_root}/internal/buildtool" -run '^TestPackagePlanCompletesResourcesBeforeFinalSignature$' -count=1 >/dev/null
GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/fallout-terminal-go-cache}" \
  go test "${repository_root}/internal/buildtool" -run '^TestPackagePlan' -count=1 >/dev/null
"${repository_root}/scripts/dependency-license-check.sh" >/dev/null
codesign --verify --deep --strict --verbose=2 "${app_path}"

bundle_digest="$("${repository_root}/scripts/hash-macos-app.sh" "${app_path}")"
printf 'Verified personal-use macOS app: arm64, macOS 13.0, native frameworks, reviewed notices, no provider executable/PATH runtime, complete resources/entitlements, offline assets, final valid signature.\n'
printf 'Canonical bundle-manifest SHA-256: %s\n' "${bundle_digest}"
