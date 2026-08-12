# Desktop Coordination Contract: Roster, Sessions, Broadcast, and Terminal Switching

**Bugfix**: 2026-08-12 — BUG-001 adds the durable player-config selection, association, and roster-save boundary.

## Boundary

The Wails `App` remains the only privileged browser-to-Go boundary. Every command validates untrusted IDs, display names, authored terminal payloads, and switch decisions before entering `internal/control`. Generated Go method names are PascalCase; `frontend/src/desktop-api.js` maps them to camelCase facade methods and returns structured results. No coordinator, filesystem, process, environment, or player-server object is exposed directly.

The trusted hacking operation name remains exactly `ForceHackSuccess`. This contract does not rename, duplicate, generalize, or publish it.

## Runtime status and event

`GetRuntimeStatus()` adds `coordinationState` to the existing status object. The game-master frontend subscribes to exactly one new Wails event, `coordination-state`, through `desktopAPI.onCoordinationState(callback)`. Event payload and status fallback have the same detached shape:

```json
{
  "revision": 31,
  "playerConfig": {
    "status": "loaded",
    "name": "Commonwealth Party",
    "filePath": "/campaign/commonwealth.players.json",
    "version": 1
  },
  "roster": [
    {"id": "character-1", "name": "Mara", "claimedBySessionId": "session-1"},
    {"id": "character-2", "name": "Boone", "claimedBySessionId": null}
  ],
  "sessions": [
    {
      "id": "session-1",
      "fallbackName": "PLAYER 3",
      "connected": true,
      "character": {"id": "character-1", "name": "Mara"},
      "role": "active"
    }
  ],
  "broadcast": {
    "id": "opaque-current-broadcast-id",
    "controllerSessionId": "session-1",
    "activeTerminalId": "terminal-1"
  },
  "pendingSwitch": null
}
```

`playerConfig` is null until a referenced, selected, or newly created config and its session association succeed. Its `status` is `loaded`; recoverable load errors are returned by the player-config operation and rendered separately rather than installing a partial handle. `broadcast` is null when ended. `character`, `controllerSessionId`, and `activeTerminalId` are nullable. Session role is `unassigned`, `active`, or `observer`. The projection includes every recognized process-local logical session so disconnected claims remain manageable; disconnected active sessions remain visibly `active` until reassigned or released. It excludes browser tokens, raw connection IDs, private hacking state, and mutable canonical references.

## Common result shapes

Ordinary coordination commands return:

```json
{
  "ok": true,
  "error": "",
  "state": {"revision": 32}
}
```

On failure, `ok` is false, `error` is a stable user-displayable explanation, and `state` is the unchanged current detached coordination snapshot. Conflict and eligibility failures do not partially mutate state.

Terminal-switch requests additionally return:

```json
{
  "ok": true,
  "error": "",
  "status": "decision-required",
  "switchId": "opaque-switch-id",
  "state": {"revision": 32}
}
```

`status` is `activated`, `cleared`, `decision-required`, or `cancelled`. `switchId` is present only for `decision-required`.

## Player-config operations — BUG-001

Player-config operations return `{ok:false,canceled:true}` on native-dialog cancellation, `{ok:false,error,state}` on failure, or `{ok:true,playerConfig,session,state}` on success. The successful `session` is the detached active document containing the saved relative association so later terminal autosaves cannot erase it. Operations require an active session file and no active broadcast.

### `LoadReferencedPlayerConfig()` / `desktopAPI.loadReferencedPlayerConfig()`

Resolves the active session's optional relative `playerConfig` reference. An absent reference returns `{ok:false}` without an error so the master can offer select/create. A missing, unreadable, unsupported, or invalid referenced file returns a visible error and leaves the previous active player config and roster unchanged. A valid file installs its roster as one coordinator transition.

### `NewPlayerConfig()` / `desktopAPI.newPlayerConfig()`

Uses a native JSON save dialog and creates a version-1 player config whose name is derived from the filename and whose roster is empty. It then saves the normalized relative reference into the active session. If association saving fails, the new standalone player-config file remains reusable, but the prior active association, roster, and coordination snapshot remain unchanged.

### `OpenPlayerConfig()` / `desktopAPI.openPlayerConfig()`

Uses a native single-file JSON open dialog, validates the complete player config, saves its normalized relative reference into the active session, and replaces the authored roster as one operation. Cancellation is not an error. Validation or association failure installs no partial roster.

## Roster commands

### `AddCharacter(name string)` / `desktopAPI.addCharacter(name)`

~~Creates a process-local roster entry with a new opaque stable ID.~~ Under BUG-001, creates a roster entry with a new opaque ID stable in the active player config. The trimmed name must be nonblank and at most 80 Unicode code points. Duplicate names are allowed. An active player config is required. The complete candidate config is atomically saved before the entry and revision are published; failure leaves the prior config and coordination state unchanged. The entry is immediately available in the current broadcast, if any.

### `RenameCharacter(payload CharacterRenamePayload)` / `desktopAPI.renameCharacter(payload)`

```json
{"characterId": "character-1", "name": "Mara Voss"}
```

Retains durable character identity and any runtime claim, atomically saves the complete active player config, and then updates every master/player projection. A save failure publishes nothing. It does not change session identity, assignment, role, terminal, navigation, puzzle, attempts, randomness, log, or outcome.

### `DeleteCharacter(characterID string)` / `desktopAPI.deleteCharacter(characterId)`

Deletes only an existing unclaimed roster entry from the active player config. A claimed entry is refused until release or transfer. The complete candidate config is atomically saved before publication; a failure changes nothing. No terminal or puzzle state changes.

## Logical-session and assignment commands

### `RenameLogicalSession(payload LogicalSessionRenamePayload)` / `desktopAPI.renameLogicalSession(payload)`

```json
{"sessionId": "session-1", "fallbackName": "TABLET LEFT"}
```

The trimmed fallback name must be nonblank, unique among recognized sessions, and at most 80 Unicode code points. Identity, presence, claim, controller, terminal, and puzzle remain unchanged.

### `AssignCharacter(payload AssignmentPayload)` / `desktopAPI.assignCharacter(payload)`

```json
{"sessionId": "session-2", "characterId": "character-2"}
```

Requires a current broadcast, an existing unassigned session, and an available roster entry. It uses the same exclusive claim transaction as player selection. If no controller exists, this new eligible assignment may establish initial control; otherwise the session becomes an observer.

### `ReleaseCharacter(sessionID string)` / `desktopAPI.releaseCharacter(sessionId)`

Removes the session's current claim and makes the roster entry available. Every connected tab of the former assignee returns to selection. If this was the controller, control becomes empty and no observer is promoted. Terminal and puzzle state remain byte-for-byte equivalent.

### `MoveCharacter(payload MoveCharacterPayload)` / `desktopAPI.moveCharacter(payload)`

```json
{"characterId": "character-1", "toSessionId": "session-3"}
```

Requires the destination session to be unassigned. Removes the existing owner when any and assigns the same stable character to the destination as one transaction. Moving the active controller's character clears control and does not transfer it implicitly. Terminal and puzzle state remain unchanged.

### `SetActiveController(sessionID string)` / `desktopAPI.setActiveController(sessionId)`

Requires a current broadcast and a connected, character-assigned destination. Sets that session active and every other assigned session observer as one transition. Reassignment changes no claim, terminal, navigation, puzzle, attempt, random, log, or outcome state.

## Broadcast lifecycle commands

### `StartBroadcast()` / `desktopAPI.startBroadcast()`

Requires no current broadcast. Creates a fresh opaque broadcast ID with no claims, no controller, no active terminal, and no suspended puzzle. Recognized sessions and roster entries remain. Connected players enter selection.

### `EndBroadcast()` / `desktopAPI.endBroadcast()`

Requires a current broadcast and clears its claims, controller, active terminal, suspended runtimes, pending switch, and per-broadcast request cache in one transition. It emits player clear/context updates but does not delete recognized sessions, fallback names, roster entries, configured durable terminals, or durable unlocked-terminal data.

### `RequestTerminalActivation(payload LiveTerminalPayload)` / `desktopAPI.requestTerminalActivation(payload)`

Uses the existing validated payload fields exactly: `terminalId`, `terminalName`, `tree`, `hackLevel`, and `introText`. A current broadcast is required.

- If the requested terminal is already active, the command acts as a validated content update and does not regenerate its puzzle.
- If the current terminal has no unfinished puzzle, the switch commits immediately.
- If it has an unfinished puzzle, the command returns `decision-required` and leaves the source terminal active and actionable.
- Activating a preserved target restores its private puzzle and generation level exactly while applying the latest authored name, tree, and intro text and revalidating navigation; a changed authored hack level applies after that runtime is discarded and regenerated.
- Activating a target with no preserved runtime creates fresh runtime state under existing navigation/hacking generation rules.

### `RequestTerminalClear()` / `desktopAPI.requestTerminalClear()`

Keeps the broadcast active but requests no active terminal. It follows the same unfinished-puzzle decision behavior as activation. Assigned sessions retain character and role and enter the waiting phase.

### `ResolveTerminalSwitch(payload TerminalSwitchDecisionPayload)` / `desktopAPI.resolveTerminalSwitch(payload)`

```json
{"switchId": "opaque-switch-id", "decision": "preserve"}
```

`decision` must be exactly `preserve`, `discard`, or `cancel`.

- `preserve` suspends the exact current private terminal runtime, then activates or clears the pending target.
- `discard` removes the source runtime, then activates or clears the pending target; a later activation creates a fresh puzzle.
- `cancel` removes the pending decision and leaves the source terminal and puzzle unchanged.
- A missing, replaced, wrong-broadcast, or wrong-source switch ID is refused without mutation.

While a decision is pending, normal player actions and game-master `ForceHackSuccess` remain ordered against the still-active source. Resolving re-evaluates the latest source state; it never applies a stale public puzzle copy.

### `UpdateLiveTerminal(payload LiveUpdatePayload)` / `desktopAPI.updateLiveTerminal(payload)`

Retains its existing tree/intro update purpose for the active terminal. It validates authored content, revalidates navigation, preserves the current puzzle, and emits a revisioned update through the coordinator order. Edits to an inactive durable terminal remain ordinary local session edits until its next activation payload.

## Trusted hacking command

### `ResetFailedHack(payload LiveTerminalPayload)` / `desktopAPI.resetFailedHack(payload)`

```json
{"terminalId":"terminal-1","terminalName":"Overseer","tree":{"id":"root","type":"folder","name":"ROOT","children":[]},"hackLevel":2,"introText":"LATEST"}
```

This exact private game-master command is eligible only while the named terminal is still the active terminal and its current puzzle is failed, unsolved, and active. The trusted App boundary validates and normalizes the latest authored terminal payload before the coordinator atomically replaces that terminal's private runtime slot with one fresh puzzle. The replacement uses a new generation, full attempts, an empty log, default navigation, and the supplied latest authored name, tree, hacking level, and intro text.

The accepted transition keeps the broadcast ID, active terminal ID, logical sessions, connections, fallback names, roster, assignments, controller, other terminal runtime slots, pending durable terminal/session data, and unlock state unchanged. It publishes one revision containing the complete fresh terminal projection to every assigned active and observer session. A duplicate call, a non-failed/solved/absent puzzle, a stale terminal identity, or an unavailable lifecycle is rejected without mutation or revision advance. Old-generation word and pattern identities cannot act on the replacement.

`ResetFailedHack` is absent from player WebSocket messages, player DOM and JavaScript globals, keyboard shortcuts, query parameters, and public HTTP endpoints. It does not broaden `ForceHackSuccess`.

### `ForceHackSuccess()` / `desktopAPI.forceHackSuccess()`

The exact existing private operation remains eligible only for an active unsolved and unfailed puzzle. It spends no attempt, uses no player identity or controller privilege, and publishes the resulting canonical success through the same ordered revision stream. It is unavailable from all player assets and protocol messages.

## Master UI behavior

- Render roster, logical sessions, presence, assignments, and controller from `coordination-state`; never infer them from raw socket count.
- Keep the existing raw `client-count` status labeled as browser connections, while coordination presence is logical-session status.
- Show character name as primary and fallback name as secondary when assigned.
- Keep a disconnected controller visibly active until reassignment or claim release.
- After session create/open, automatically load its valid referenced player config; otherwise present explicit select-existing and create-new actions before enabling roster or broadcast controls.
- Keep terminal authoring available when player-config selection is cancelled or its reference is missing/invalid, show recoverable errors, and never render a partial roster.
- Display the active player-config name/path separately from transient roster availability and session assignments.
- Await every command result before changing visible broadcast, terminal, claim, role, or pending-switch state.
- Present preserve, discard, and cancel in a blocking game-master dialog whenever `decision-required` is returned.
- While the active puzzle is failed, replace the obsolete broadcast-restart instruction with a visible enabled `ПОВТОРИТЬ ВЗЛОМ` control; await `ResetFailedHack`, disable it while pending, and surface any authoritative refusal without changing local puzzle state optimistically.
- Refusal to delete a claimed character or resolve a stale switch must be visible and must not close over stale local state.
- Roster/session controls and switch dialogs follow the existing plain-DOM render functions, restrictive CSP, terminal aesthetic, and escaped-text rules.

## Persistence and compatibility

~~No method in this contract changes the version-1 session schema.~~ BUG-001 adds only the optional normalized relative `playerConfig` reference and saves it through the existing privileged session boundary. Authored terminal edits continue through existing explicit save behavior. Reusable roster IDs/names are encoded only in the separate player-config JSON. Logical sessions, fallback labels, recognition, connections, presence, claims, assignments, controller, broadcast, revisions, switches, terminal runtime, and puzzles are never encoded into either durable file. The same player config may be referenced by multiple sessions and is re-read when each session is opened; background watching and concurrent multi-process merge are outside this contract.
