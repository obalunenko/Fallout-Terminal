#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
default_app_path="${repository_root}/build/dev/Fallout Terminal.app"
default_fixture="${repository_root}/internal/testutil/testdata/session-v1-state-changing.json"
app_path="${1:-${default_app_path}}"
fixture_path="${2:-${default_fixture}}"
local_url='http://127.0.0.1:3690/'
smoke_deadline_seconds=10
smoke_root=''
app_pid=''

fail() {
  printf 'state-changing-reset-native-smoke: FAIL: %s\n' "$1" >&2
  exit 1
}

pass() {
  printf 'state-changing-reset-native-smoke: PASS: %s\n' "$1"
}

not_run() {
  printf 'state-changing-reset-native-smoke: NOT RUN: %s\n' "$1"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

process_ids() {
  pgrep -f -- "${app_path}/Contents/MacOS/Fallout Terminal" 2>/dev/null || true
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

wait_for_file_reset() {
  local session_path="$1"
  local terminal_id="$2"
  local deadline=$((SECONDS + smoke_deadline_seconds))
  while (( SECONDS <= deadline )); do
    if python3 -c 'import json,sys; d=json.load(open(sys.argv[1], encoding="utf-8")); t=next(x for x in d["terminals"] if x["id"]==sys.argv[2]); raise SystemExit(0 if not t.get("commandStates") else 1)' "${session_path}" "${terminal_id}"; then
      return 0
    fi
  done
  return 1
}

cleanup() {
  if [[ "${app_pid}" =~ ^[0-9]+$ ]]; then
    kill -TERM "${app_pid}" 2>/dev/null || true
  fi
  if [[ -n "${smoke_root}" ]]; then
    case "${smoke_root}" in
      /private/tmp/fallout-native-reset.*|/tmp/fallout-native-reset.*) rm -rf -- "${smoke_root}" ;;
      *) printf 'state-changing-reset-native-smoke: refusing unexpected cleanup path: %s\n' "${smoke_root}" >&2 ;;
    esac
  fi
}

validate_command_only_fixture() {
  python3 -c '
import json, sys
document = json.load(open(sys.argv[1], encoding="utf-8"))
for terminal in document.get("terminals", []):
    nodes = {}
    stack = [terminal.get("root", {})]
    while stack:
        node = stack.pop()
        nodes[node.get("id")] = node.get("type")
        stack.extend(node.get("children", []))
    for command_id in terminal.get("commandStates", {}):
        if nodes.get(command_id) != "command":
            raise SystemExit(f"commandStates[{command_id!r}] does not reference a command node")
' "$1"
}

run_self_test() {
  validate_command_only_fixture "${default_fixture}"
  grep -Fq "resetConfirmationDialog.id = 'resetConfirmationDialog';" "${repository_root}/frontend/src/master.js" ||
    fail 'master reset confirmation dialog is missing'
  grep -Fq 'desktopAPI.resetTerminalCommandStates({ terminalId: term.id })' "${repository_root}/frontend/src/master.js" ||
    fail 'master reset-all control is not wired to the generated desktop facade'
  grep -Fq 'resetTerminalCommandStates: desktopService.ResetTerminalCommandStates' "${repository_root}/frontend/src/desktop-api.js" ||
    fail 'desktop facade is not wired to the generated Wails binding'
  if grep -F 'window.confirm(`Сбросить' "${repository_root}/frontend/src/master.js" >/dev/null; then
    fail 'command-state reset still depends on unsupported native window.confirm'
  fi
  pass 'command-only fixture, in-app confirmation, and generated Wails reset chain are present'
}

if [[ "${1:-}" == '--self-test' ]]; then
  run_self_test
  exit 0
fi

[[ "$#" -le 2 ]] || fail 'usage: scripts/state-changing-reset-native-smoke.sh [APP_PATH] [SESSION_FIXTURE]'
[[ "$(uname -s)" == Darwin ]] || fail 'native master-click smoke requires macOS'
for command in curl grep mktemp open osascript pgrep python3; do
  require_command "${command}"
done
[[ -d "${app_path}" ]] || fail 'application bundle is missing'
[[ -x "${app_path}/Contents/MacOS/Fallout Terminal" ]] || fail 'application executable is missing'
[[ -f "${fixture_path}" ]] || fail 'session fixture is missing'
app_path="$(cd "$(dirname "${app_path}")" && pwd)/$(basename "${app_path}")"
fixture_path="$(cd "$(dirname "${fixture_path}")" && pwd)/$(basename "${fixture_path}")"
validate_command_only_fixture "${fixture_path}"
[[ -z "$(process_ids)" ]] || fail 'a matching application process is already running'

smoke_root="$(mktemp -d /private/tmp/fallout-native-reset.XXXXXX)"
session_path="${smoke_root}/session.json"
cp "${fixture_path}" "${session_path}"
chmod u+w "${session_path}"
terminal_id="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1], encoding="utf-8")); print(next(t["id"] for t in d["terminals"] if t.get("commandStates")))' "${session_path}")"
before_selected_states="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1], encoding="utf-8")); t=next(x for x in d["terminals"] if x["id"]==sys.argv[2]); print(len(t.get("commandStates", {})))' "${session_path}" "${terminal_id}")"
before_other_states="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1], encoding="utf-8")); print(sum(len(t.get("commandStates", {})) for t in d["terminals"] if t["id"] != sys.argv[2]))' "${session_path}" "${terminal_id}")"
[[ "${before_selected_states}" =~ ^[1-9][0-9]*$ ]] || fail 'selected fixture terminal has no completed command snapshots'

trap cleanup EXIT HUP INT TERM
open -n "${app_path}"
wait_for_local || fail 'application did not make local mode ready'
app_pid="$(process_ids)"
[[ "${app_pid}" =~ ^[0-9]+$ ]] || fail 'expected exactly one native application process'

# This is intentionally UI automation, not a direct App invocation. The first
# click opens the native file picker; the second clicks the real master reset
# control; the final key sequence activates the in-app confirmation button.
if ! osascript - "${app_pid}" "${session_path}" >/dev/null <<'APPLESCRIPT'
on findNamedButton(containerElement, buttonName)
  tell application "System Events"
    try
      if role of containerElement is "AXButton" and name of containerElement is buttonName then
        return containerElement
      end if
      set childElements to UI elements of containerElement
    on error
      set childElements to {}
    end try
  end tell
  repeat with childElement in childElements
    set candidate to my findNamedButton(childElement, buttonName)
    if candidate is not missing value then return candidate
  end repeat
  return missing value
end findNamedButton

on clickButton(processID, buttonName, attempts)
  repeat attempts times
    tell application "System Events"
      tell first process whose unix id is processID
        set frontmost to true
        try
          set candidate to my findNamedButton(window 1, buttonName)
          if candidate is not missing value then
            click candidate
            return true
          end if
        end try
      end tell
    end tell
    delay 0.1
  end repeat
  return false
end clickButton

on clickResetConfirmation(processID)
  tell application "System Events"
    tell first process whose unix id is processID
      set frontmost to true
      set windowPosition to position of window 1
      set windowSize to size of window 1
      -- WebKit exposes the modal itself but hides its HTML button descendants
      -- from the macOS Accessibility tree. The dialog is centered and the
      -- confirm action is the left half, so derive the real click from the
      -- current native window geometry rather than fixed screen coordinates.
      set confirmX to (item 1 of windowPosition) + ((item 1 of windowSize) div 2) - 130
      set confirmY to (item 2 of windowPosition) + ((item 2 of windowSize) div 2) + 52
      click at {confirmX, confirmY}
    end tell
  end tell
end clickResetConfirmation

on run argv
  set processID to (item 1 of argv) as integer
  set sessionPath to item 2 of argv
  if not clickButton(processID, "ОТКРЫТЬ СЕССИЮ", 100) then error "open-session button not found"
  delay 0.5
  tell application "System Events" to tell first process whose unix id is processID
    set frontmost to true
    keystroke "g" using {command down, shift down}
    delay 0.2
    keystroke sessionPath
    key code 36
    delay 0.5
    key code 36
  end tell
  if not clickButton(processID, "СБРОСИТЬ ВСЕ СОСТОЯНИЯ", 150) then error "reset-all button not found"
  delay 0.5
  clickResetConfirmation(processID)
end run
APPLESCRIPT
then
  not_run 'Accessibility automation could not drive the native window; grant Accessibility access to the invoking terminal and rerun'
  exit 2
fi

wait_for_file_reset "${session_path}" "${terminal_id}" ||
  fail 'native master click did not remove canonical commandStates within the deadline'
after_selected_states="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1], encoding="utf-8")); t=next(x for x in d["terminals"] if x["id"]==sys.argv[2]); print(len(t.get("commandStates", {})))' "${session_path}" "${terminal_id}")"
after_other_states="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1], encoding="utf-8")); print(sum(len(t.get("commandStates", {})) for t in d["terminals"] if t["id"] != sys.argv[2]))' "${session_path}" "${terminal_id}")"
[[ "${after_selected_states}" == 0 ]] || fail 'native reset left a completed command snapshot in the selected terminal'
[[ "${after_other_states}" == "${before_other_states}" ]] || fail 'native reset changed command snapshots in another terminal'
validate_command_only_fixture "${session_path}"
pass "native click traversed in-app confirmation and generated Wails reset for terminal ${terminal_id}; selected commandStates changed ${before_selected_states}->${after_selected_states}, other terminals preserved ${before_other_states}"
