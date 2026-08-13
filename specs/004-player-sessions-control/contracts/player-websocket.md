# Player WebSocket Contract: Sessions, Selection, Roles, and Ordered Actions

> **SUPERSEDED LEGACY PLAYER TRANSPORT — HISTORICAL, NON-AUTHORITATIVE.**
> Any WebSocket or handwritten JSON player-transport description in this retained
> completed feature document has been replaced by the generated ConnectRPC contract in
> [`specs/005-connectrpc-protobuf-migration/contracts/public-player.md`](../../005-connectrpc-protobuf-migration/contracts/public-player.md).

## Scope and compatibility

This contract extends the existing same-origin player WebSocket. The embedded server and bundled player are released together; there is no negotiation with older browser assets. JSON objects remain bounded to 4 KiB for client messages, strictly decoded, and rejected on duplicate, unknown, missing, blank, or wrong-typed fields. JSON field names are camelCase and message types are uppercase snake case.

The only trusted forced-success operation remains exactly `ForceHackSuccess`. It has no player message, endpoint, query parameter, browser global, DOM control, or keyboard path. `HACK_ADMIN` and every other unsupported type remain rejected.

## Connection handshake

No roster, assignment, terminal snapshot, or action is accepted before the first valid handshake frame.

### `SESSION_HELLO`

First connection for an origin/profile:

```json
{
  "type": "SESSION_HELLO"
}
```

Reconnect or another tab:

```json
{
  "type": "SESSION_HELLO",
  "browserToken": "opaque-server-issued-token"
}
```

`browserToken` is optional, but when present it must be a nonblank string. Any other field is prohibited. An absent or server-unknown token creates a fresh logical session and replacement token. The server never treats token possession as controller authorization.

### `SESSION_WELCOME`

```json
{
  "type": "SESSION_WELCOME",
  "browserToken": "opaque-server-issued-token",
  "state": {
    "revision": 17,
    "sessionId": "opaque-process-session-id",
    "fallbackName": "PLAYER 3",
    "character": null,
    "role": "unassigned",
    "phase": "selecting",
    "broadcastId": "opaque-current-broadcast-id",
    "activeTerminalId": "terminal-1",
    "roster": [
      {"id": "character-1", "name": "Mara", "status": "available"},
      {"id": "character-2", "name": "Boone", "status": "claimed"}
    ]
  }
}
```

The browser stores only `browserToken` in same-origin `localStorage`. It does not persist fallback name, `sessionId`, character, role, roster, broadcast, terminal, navigation, or hacking state. The connection overlay remains active until `SESSION_WELCOME` is applied.

## Player-to-server requests

### `CHARACTER_SELECT`

```json
{
  "type": "CHARACTER_SELECT",
  "requestId": "client-opaque-request-id",
  "broadcastId": "opaque-current-broadcast-id",
  "characterId": "character-1"
}
```

All four fields are required and must be nonblank strings. Acceptance requires the sending connection's logical session to be unassigned in that broadcast and the stable character ID to be available when processed. A player already assigned cannot replace its assignment. Conflict leaves the claimant unassigned and publishes the current `PLAYER_STATE` plus a rejected `ACTION_RESULT`.

### `NAV_ACTION`

```json
{
  "type": "NAV_ACTION",
  "requestId": "client-opaque-request-id",
  "broadcastId": "opaque-current-broadcast-id",
  "terminalId": "terminal-1",
  "action": "enter",
  "nodeId": "folder-2"
}
```

`requestId`, `broadcastId`, `terminalId`, and `action` are required. Existing action values remain exactly `enter`, `command`, `entry`, and `back`. `nodeId` is required and nonblank for `enter`, `command`, and `entry`; it is optional for `back` and, if present, must be nonblank. Any other field or action is rejected before dispatch.

### `HACK_GUESS`

```json
{
  "type": "HACK_GUESS",
  "requestId": "client-opaque-request-id",
  "broadcastId": "opaque-current-broadcast-id",
  "terminalId": "terminal-1",
  "targetId": "A1"
}
```

All fields are required and nonblank. Existing candidate/filler validation, likeness, attempts, special-pattern-adjacent behavior, dud removal, restoration, logs, success, and failure rules do not change.

### `HACK_PATTERN`

```json
{
  "type": "HACK_PATTERN",
  "requestId": "client-opaque-request-id",
  "broadcastId": "opaque-current-broadcast-id",
  "terminalId": "terminal-1",
  "patternId": "opaque-generation-bound-pattern-id"
}
```

All fields are required and nonblank. `patternId` remains opaque and is validated by the existing generation-bound pattern rules. A concurrent or rapid duplicate cannot accept more than once.

### Common authorization and ordering

For every selection or terminal action, the server:

1. resolves the sending connection to one current logical session;
2. deduplicates `requestId` within that logical session and broadcast;
3. validates `broadcastId` against the current broadcast;
4. for terminal actions, requires a current assignment, connected presence, controller identity, and matching active `terminalId`;
5. applies the unchanged type-specific rule;
6. commits one coordinator revision for an accepted mutation;
7. enqueues detached state effects and then the correlated result before another transition can publish over it.

Unknown, unassigned, observer, disconnected, former-controller, stale-broadcast, stale-terminal, invalid, and expired requests leave all canonical terminal, navigation, hacking, random, attempt, pattern, log, and outcome state unchanged.

## Server-to-player state

### `PLAYER_STATE`

```json
{
  "type": "PLAYER_STATE",
  "state": {
    "revision": 23,
    "sessionId": "opaque-process-session-id",
    "fallbackName": "PLAYER 3",
    "character": {"id": "character-1", "name": "Mara"},
    "role": "observer",
    "phase": "observing",
    "broadcastId": "opaque-current-broadcast-id",
    "activeTerminalId": "terminal-1",
    "roster": [
      {"id": "character-1", "name": "Mara", "status": "claimed"},
      {"id": "character-2", "name": "Boone", "status": "available"}
    ]
  }
}
```

`character`, `broadcastId`, and `activeTerminalId` are nullable. `role` is `unassigned`, `active`, or `observer`. `phase` is one of `no-broadcast`, `selecting`, `waiting`, `controlling`, or `observing`. Roster `status` is only `available` or `claimed`; claimant, fallback names, presence, raw connection count, and other session data are prohibited.

The complete personalized state is sent after handshake, roster/assignment/role changes, broadcast lifecycle changes, active-terminal changes, and relevant reconnects. Every open tab mapped to the same logical session receives the same state revision.

### `ACTION_RESULT`

Accepted example:

```json
{
  "type": "ACTION_RESULT",
  "requestId": "client-opaque-request-id",
  "accepted": true,
  "reason": "accepted",
  "revision": 24
}
```

Rejected example:

```json
{
  "type": "ACTION_RESULT",
  "requestId": "client-opaque-request-id",
  "accepted": false,
  "reason": "not-controller",
  "revision": 24
}
```

The result goes to the initiating connection. Stable public reasons are `accepted`, `invalid-session`, `stale-broadcast`, `unassigned`, `not-controller`, `controller-disconnected`, `stale-terminal`, `invalid-action`, `conflict`, and `duplicate`. An exact replay of a prior `requestId` and payload returns the cached original result without another mutation. Reusing that ID with different type-specific or precondition fields is rejected as `duplicate`.

For an accepted request, the initiating browser keeps shared input pending until it has applied the relevant state envelope at `revision` or later and has received the matching result. For a rejection, the matching result ends pending immediately because canonical state did not change. Pending is never cleared solely by an animation delay.

## Revisioned terminal envelopes

The existing families remain, with `revision` added to every envelope:

- `TERMINAL_LIVE`: `type`, `revision`, `terminalId`, `terminalName`, `tree`, `hackLevel`, `introText`, `hack`, `nav`;
- `TERMINAL_UPDATE`: `type`, `revision`, `terminalId`, `tree`, `introText`, `nav`;
- `TERMINAL_CLEAR`: `type`, `revision`;
- `NAV_STATE`: `type`, `revision`, `terminalId`, `nav`;
- `HACK_STATE`: `type`, `revision`, `terminalId`, `hack`.

Existing nested navigation and hacking shapes remain unchanged. Public hacking values remain detached and secret-free. `TERMINAL_LIVE` on first assignment, reconnect, or active-terminal change carries the current exact canonical terminal state. `TERMINAL_CLEAR` means the current personalized state has no active player terminal; `PLAYER_STATE` distinguishes no broadcast, selection, and assigned waiting.

When `activeTerminalId` changes, the bundled player applies the existing typewriter/reveal presentation before exposing the new `TERMINAL_LIVE` content. It does not alter identity, assignment, or controller role during the transition. An inactive terminal ID in any player request is rejected.

## Player presentation contract

- An unassigned session sees the current roster in an immersive terminal-styled selection state. Available and claimed entries are visually distinguishable; only available IDs can be submitted.
- An assigned session with no active terminal sees an immersive waiting state while retaining character and role.
- Character name is primary after assignment; fallback name remains a secondary technical label.
- An active controller can invoke existing shared navigation and hacking sends when no request is pending.
- An observer sees the same canonical terminal and may use hover, focus, paging, preview, typed text, sound, and other local feedback, but shared controls are visibly read-only and invoke no send path.
- UI gating covers pointer, keyboard, back, row activation, candidate, filler, and special-pattern paths. The server repeats all authorization for crafted messages.
- No client mutates tree position, puzzle, attempts, board, log, pattern state, outcome, assignment, role, roster, or active-terminal identity optimistically.
- Authored and game-master-supplied names are rendered as text, not executable markup. Browser tokens never appear in markup, query strings, console output, or URLs.

## Reconnection and multi-tab behavior

- Multiple tabs using one valid token attach to one logical session and receive identical `PLAYER_STATE` changes.
- Closing one tab does not change logical presence while another tab remains connected.
- Closing the final tab marks the logical session disconnected but retains its claim and controller identity.
- Reconnecting the unchanged controller restores its ability to send after welcome; reconnecting after game-master reassignment returns as observer.
- A token unknown after process restart receives a replacement and a fresh session with no restored fallback rename, claim, presence, control, terminal, or puzzle state.
- The recognition scope is the exact browser origin/profile. A different origin, browser profile, private context, or cleared storage establishes a different logical session.

## Security and privacy

- The existing same-host Origin check, CSP, bounded reads, strict decoder, and slow-client queue isolation remain.
- Browser recognition is not authentication and never bypasses per-action assignment/controller checks.
- Player projections expose no raw connection identifiers/counts, other-session fallback names, claimant identity, private candidate truth, secret word, random source, or trusted desktop capability.
- No player-accessible identifier or operation invokes `ForceHackSuccess`.
