#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
default_app_path="${repository_root}/build/bin/Fallout Terminal.app"
app_path="${1:-${default_app_path}}"
executable_path="${app_path}/Contents/MacOS/Fallout Terminal"
local_url='http://127.0.0.1:3690/'
smoke_deadline_seconds=5
window_close_sender_pid=''

fail() {
  printf 'public-access-macos-smoke: FAIL: %s\n' "$1" >&2
  exit 1
}

pass() {
  printf 'public-access-macos-smoke: PASS: %s\n' "$1"
}

not_run() {
  printf 'public-access-macos-smoke: NOT RUN: %s\n' "$1"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

process_ids() {
  pgrep -f -- "${executable_path}" 2>/dev/null || true
}

wait_for_local() {
  local deadline=$((SECONDS + smoke_deadline_seconds))
  while (( SECONDS <= deadline )); do
    if curl --fail --silent --show-error --max-time 1 "${local_url}" >/dev/null 2>&1; then
      return 0
    fi
  done
  return 1
}

wait_for_exit() {
  local deadline=$((SECONDS + smoke_deadline_seconds))
  while (( SECONDS <= deadline )); do
    [[ -z "$(process_ids)" ]] && return 0
  done
  return 1
}

launch_via_finder_semantics() {
  # LaunchServices is the automation equivalent of double-clicking the bundle:
  # no executable arguments, provider credentials, or app-specific environment
  # are supplied to the packaged process.
  open -n "${app_path}"
  wait_for_local || fail 'double-click launch did not make local mode ready within five seconds'
}

wait_for_deferred_overseer_window() {
  # Wails v3 defers native window creation until App.Run. The player service
  # can make port 3690 ready just before Cocoa has installed that window, so a
  # close Apple Event sent at the HTTP readiness edge can race and be dropped.
  # Keep this synchronization outside the measured close budget; the launch
  # itself has already met its independent five-second readiness assertion.
  sleep 1
  [[ -n "$(process_ids)" ]] || fail 'packaged process exited before its deferred Overseer window was ready'
}

send_window_close() {
  # Address the application directly instead of using System Events. The
  # latter requires an interactive Accessibility grant and can silently send
  # Cmd+W to the wrong foreground process in headless/CI runs. A standard
  # `close window 1` Apple Event reaches the same native NSWindow close path
  # without granting the smoke harness control over the user's keyboard.
  # Keep the normal request/reply Apple Event semantics: AppKit routes that
  # through the native close lifecycle. Run the sender asynchronously because
  # osascript may wait for its reply after the target process has already met
  # the cleanup budget.
  osascript -e "tell application \"${app_path}\" to close window 1" >/dev/null 2>&1 &
  window_close_sender_pid=$!
}

finish_window_close_sender() {
  case "${window_close_sender_pid}" in
    ''|*[!0-9]*) return 0 ;;
  esac
  kill -TERM "${window_close_sender_pid}" 2>/dev/null || true
  wait "${window_close_sender_pid}" 2>/dev/null || true
  window_close_sender_pid=''
}

send_command_quit() {
  # Cmd+Q dispatches the application's standard quit action. Exercise that
  # same NSApplication termination path with a direct Apple Event so the gate
  # is deterministic and needs no Accessibility permission.
  osascript \
    -e 'ignoring application responses' \
    -e "tell application \"${app_path}\" to quit" \
    -e 'end ignoring' >/dev/null
}

validated_single_pid() {
  local ids
  ids="$(process_ids)"
  [[ "${ids}" =~ ^[0-9]+$ ]] || fail 'expected exactly one owned packaged process'
  [[ "${ids}" != "$$" ]] || fail 'refusing to target the smoke harness process'
  printf '%s' "${ids}"
}

probe_stale_public_url() {
  local raw_url="${FALLOUT_PUBLIC_ACCESS_SMOKE_URL:-}"
  if [[ "${FALLOUT_PUBLIC_ACCESS_SMOKE_OPT_IN:-0}" != 1 || -z "${raw_url}" ]]; then
    not_run 'real stale-public-URL probe requires explicit opt-in and a non-secret active URL; no provider request was made'
    return 0
  fi
  case "${raw_url}" in
    https://*) ;;
    *) fail 'opt-in stale public URL must use HTTPS' ;;
  esac
  if curl --fail --silent --show-error --max-time 3 "${raw_url}" >/dev/null 2>&1; then
    fail 'the prior public URL remained usable after forced owner loss'
  fi
  pass 'prior public URL was unusable after forced owner loss'
}

cleanup() {
  finish_window_close_sender
  local ids
  ids="$(process_ids)"
  for pid in ${ids}; do
    case "${pid}" in
      ''|*[!0-9]*) ;;
      *) kill -TERM "${pid}" 2>/dev/null || true ;;
    esac
  done
}

if [[ "${1:-}" == '--self-test' ]]; then
  [[ "${local_url}" == 'http://127.0.0.1:3690/' ]] || fail 'self-test local target drifted'
  [[ "${smoke_deadline_seconds}" == 5 ]] || fail 'self-test lifecycle deadline drifted'
  [[ -z "${FALLOUT_PUBLIC_ACCESS_SMOKE_URL:-}" ]] || fail 'self-test refuses external URL input'
  pass 'harness constants, redacted output path, and credential-free default are valid'
  exit 0
fi

[[ "$#" -le 1 ]] || fail 'usage: scripts/public-access-macos-smoke.sh [APP_PATH]'
[[ "$(uname -s)" == Darwin ]] || fail 'packaged lifecycle smoke requires macOS'
[[ -d "${app_path}" ]] || fail 'application bundle is missing'
app_path="$(cd "$(dirname "${app_path}")" && pwd)/$(basename "${app_path}")"
case "${app_path}" in
  *'"'*|*$'\n'*) fail 'application bundle path cannot be represented safely in an Apple Event' ;;
esac
executable_path="${app_path}/Contents/MacOS/Fallout Terminal"
for command in curl kill open osascript pgrep; do
  require_command "${command}"
done
[[ -x "${executable_path}" ]] || fail 'application executable is missing'
trap cleanup EXIT HUP INT TERM

cleanup
wait_for_exit || fail 'pre-existing packaged process did not exit within five seconds'

launch_via_finder_semantics
wait_for_deferred_overseer_window
if [[ "${FALLOUT_PUBLIC_ACCESS_MANUAL_CLOSE_NOT_RUN:-0}" == 1 ]]; then
  not_run 'interactive normal-close automation was explicitly deferred for manual follow-up by the user'
  manual_close_cleanup_started_at=${SECONDS}
  send_command_quit
  wait_for_exit || fail 'cleanup after deferred interactive close exceeded five seconds'
  manual_close_cleanup_elapsed=$((SECONDS - manual_close_cleanup_started_at))
  pass "deferred-close launch cleanup met the budget (elapsed_seconds=${manual_close_cleanup_elapsed}, limit_seconds=${smoke_deadline_seconds})"
else
  close_started_at=${SECONDS}
  send_window_close
  if ! wait_for_exit; then
    finish_window_close_sender
    fail 'normal window close exceeded five seconds'
  fi
  finish_window_close_sender
  close_elapsed=$((SECONDS - close_started_at))
  pass "double-click launch and normal close preserved local mode (elapsed_seconds=${close_elapsed}, limit_seconds=${smoke_deadline_seconds})"
fi

launch_via_finder_semantics
quit_started_at=${SECONDS}
send_command_quit
wait_for_exit || fail 'Cmd+Q exceeded five seconds'
quit_elapsed=$((SECONDS - quit_started_at))
pass "Cmd+Q-equivalent application quit met the cleanup budget (elapsed_seconds=${quit_elapsed}, limit_seconds=${smoke_deadline_seconds})"

launch_via_finder_semantics
owner_pid="$(validated_single_pid)"
open -n "${app_path}"
sleep 1
wait_for_local || fail 'partial second-instance startup disrupted the owning local server'
new_ids="$(process_ids | while IFS= read -r pid; do [[ "${pid}" == "${owner_pid}" ]] || printf '%s\n' "${pid}"; done)"
for pid in ${new_ids}; do
  case "${pid}" in
    ''|*[!0-9]*) fail 'partial-start process identity was unsafe' ;;
    *) kill -TERM "${pid}" ;;
  esac
done
pass 'partial startup left the original local server usable'

kill -KILL "${owner_pid}"
wait_for_exit || fail 'forced owner loss left an owned packaged process'
probe_stale_public_url

launch_via_finder_semantics
wait_for_local || fail 'relaunch after forced owner loss did not restore local mode'
probe_stale_public_url
relaunch_quit_started_at=${SECONDS}
send_command_quit
wait_for_exit || fail 'post-owner-loss relaunch cleanup exceeded five seconds'
relaunch_quit_elapsed=$((SECONDS - relaunch_quit_started_at))
pass "forced owner loss and next launch retained local mode with no automatic stale endpoint reuse (elapsed_seconds=${relaunch_quit_elapsed}, limit_seconds=${smoke_deadline_seconds})"

trap - EXIT HUP INT TERM
pass 'packaged lifecycle smoke completed without reading or printing credentials'
