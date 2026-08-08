---
status: migrated
feature: Live Broadcast and Shared Navigation
source: existing implementation
---

# Implementation Plan: Live Broadcast and Shared Navigation

**Migration status**: Reconstructed from the existing implementation on 2026-08-09  
**Specification**: `specs/live-broadcast-shared-navigation/spec.md`

## Summary

The existing application starts an Express and WebSocket server inside the Electron main process, exposes broadcast controls through a narrow preload bridge, and stores a single in-memory live-terminal object in `server/server.js`. Player browsers request navigation changes over WebSocket. Transport-independent functions in `server/nav.js` validate and mutate the canonical navigation object, after which the server broadcasts the resulting state to every connected player. Full broadcasts reset navigation; content updates revalidate it against the edited tree; reconnections receive the current snapshot.

This is a documentation-only migration. It does not change source, runtime behavior, dependencies, session data, or protocol compatibility.

## Technical Context

| Area | Existing choice |
|---|---|
| Language | JavaScript with CommonJS in main/server modules and browser scripts in renderers |
| Desktop runtime | Electron 28 with a sandboxed, context-isolated master renderer |
| HTTP transport | Express 4 static application |
| Realtime transport | `ws` 8 WebSocket server and native browser `WebSocket` client |
| Server bind | `0.0.0.0:3690` by default |
| Canonical live state | One process-global in-memory `live` object in `server/server.js` |
| Canonical navigation | Mutable `{ path, mode, viewEntryId, commandNodeId }` object |
| Master integration | Fire-and-forget Electron IPC exposed through `preload.js` |
| Player convergence | `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `TERMINAL_CLEAR`, and `NAV_STATE` |
| Persistence | None; live, navigation, socket, and connection-count state are runtime-only |
| Automated tests | No test framework, test directory, coverage threshold, lint command, or CI workflow is configured |

## Detected Scope

### Navigation domain

- `server/nav.js` — creates default navigation, resolves the current folder, validates direct-child actions, applies folder/entry/command/back transitions, and repairs navigation after live content changes.

### Server and live-state transport

- `server/server.js` — starts Express and WebSocket services, discovers the local address, serves the player application, owns connected sockets and canonical live state, handles player messages, broadcasts public state, and reports connection counts.

### Electron main process and preload boundary

- `main.js` — starts the server before creating the master window, forwards server information and connection counts, routes broadcast IPC, and allowlists external HTTP(S) URL opening.
- `preload.js` — exposes explicit server-information listeners and make-live, update-live, and clear-live methods to the sandboxed master renderer.

### Game-master controls

- `master/index.html` — provides the player address, connected count, live indicator, make-live/restart, publish-update, and stop-broadcast controls.
- `master/master.js` — tracks the locally selected live terminal, maps controls to preload calls, displays server information and client counts, and prevents publish actions for a different edited terminal.

### Player protocol integration

- `client/client.js` — connects and reconnects WebSocket transport, dispatches live/navigation messages, sends navigation requests, mirrors canonical navigation, and keeps only row highlighting local.

### Related but out of scope

- `server/hack.js`, `server/wordbank.js`, and hacking portions of `server/server.js` attach a public hacking projection to live state but are specified in `specs/hacking-game/`.
- Player markup, rendering effects, input styling, and sound behavior are specified in `specs/player-terminal-presentation/`.
- Session loading and autosave are specified in `specs/session-persistence/`.
- `server/ngrok.js` and public-access credential handling are not part of local live broadcast and should receive their own migration.

## Existing Architecture and Data Flow

```text
master/master.js
  → preload broadcast method
  → main.js IPC route
  → server/server.js canonical live state
  → TERMINAL_LIVE / TERMINAL_UPDATE / TERMINAL_CLEAR
  → all player browsers

player browser NAV_ACTION
  → server/server.js message parser
  → server/nav.js validation and mutation
  → NAV_STATE broadcast
  → all player browsers converge
```

The main process owns application startup but does not own navigation. The master renderer owns editing state but does not communicate directly with server modules. The player browser never treats folder, entry, or command activation as canonical until the server echoes state.

## Runtime State Model

The canonical live object is held only in memory and has the effective shape:

```js
{
  terminalId,
  terminalName,
  tree,
  hackLevel,
  introText,
  hack,
  nav: {
    path: ['root'],
    mode: 'list',
    viewEntryId: null,
    commandNodeId: null,
  },
}
```

The player receives this object only through `publicLiveState()`, which substitutes the public hacking projection. For this feature, `tree` and `nav` are authoritative. Player variables such as selection index, reveal history, and generated server display number remain browser-local presentation state.

## Existing Protocol

| Message | Direction | Payload | Validation | Reconnection behavior |
|---|---|---|---|---|
| `TERMINAL_LIVE` | Server → player | Public live object spread beside `type` | Constructed from canonical state | Sent directly on connection when live state exists |
| `TERMINAL_UPDATE` | Server → player | `tree`, `introText`, `nav` | Server ignores update when no live state exists; navigation is revalidated | Incorporated into later full snapshots |
| `TERMINAL_CLEAR` | Server → player | `{ type }` | Emitted after setting `live` to `null` | Later connections receive no live snapshot |
| `NAV_ACTION` | Player → server | `action`, optional `nodeId` | JSON must parse, live state must exist, action must be a string, and node target/type is checked by `server/nav.js` | Not replayed; current result is included in the reconnect snapshot |
| `NAV_STATE` | Server → player | Canonical `nav` | Produced after an eligible navigation request | Current navigation is also carried by `TERMINAL_LIVE` |

There is no protocol version or acknowledgement/error message. Existing consumers must continue to tolerate unknown server message types by doing nothing. This migration adds no message and changes no payload.

## Reconstructed Implementation Phases

### Phase 1 — Transport-independent shared navigation

- Established root-list defaults for canonical navigation.
- Added tree traversal and current-folder resolution.
- Implemented direct-child validation for folder, entry, and command actions.
- Implemented entry-close and folder-pop back behavior.
- Added navigation revalidation for trees changed during a live broadcast.

### Phase 2 — Embedded HTTP/WebSocket server

- Created an Express application and HTTP server bound to the local network.
- Served the player client as static content.
- Registered player WebSockets and reported changes in the connection set.
- Added a broadcast helper that sends only to open sockets.
- Added one canonical live object and public snapshot projection.
- Parsed player requests, delegated navigation to the domain module, and broadcast the canonical result.

### Phase 3 — Broadcast lifecycle

- Added full broadcast start/restart with fresh default navigation.
- Added non-destructive tree/introduction updates that retain other runtime state.
- Revalidated navigation before publishing updated content.
- Added broadcast clearing and waiting-state notification.
- Sent the current live snapshot to late or reconnecting players.

### Phase 4 — Electron and game-master integration

- Started the local server as part of Electron window creation.
- Forwarded player address and connection counts into the master window.
- Exposed narrow preload methods for broadcast operations.
- Added make-live/restart, explicit update, and stop controls.
- Kept the game master's editor selection distinct from the selected live terminal.

### Phase 5 — Player convergence

- Added protocol-aware `ws`/`wss` connection setup and delayed reconnect.
- Applied full snapshots, incremental content updates, navigation updates, and clear events.
- Sent typed navigation requests rather than mutating shared navigation locally.
- Retained browser-local selection highlighting while replacing canonical navigation fields from the server.

## Key Technical Decisions

1. **One canonical live terminal** keeps the tabletop experience intentionally shared and avoids per-client session state.
2. **Navigation domain logic is transport-independent** so tree validation does not depend on Express, WebSocket, Electron, or DOM APIs.
3. **Player actions are requests** and are reflected only after a server message, preventing ordinary client divergence.
4. **Full broadcast and live update have different semantics**: full broadcast resets runtime navigation, while update preserves and repairs it.
5. **Reconnections use a complete snapshot** instead of replaying missed actions.
6. **Command output is derived from the shared tree** using the canonical command node ID rather than duplicated in navigation state.
7. **Runtime state is not persisted** because broadcast position and connected sockets are ephemeral table state.
8. **The master renderer uses fire-and-forget IPC** for broadcast controls; the existing implementation has no acknowledgement or structured failure path.

## Constitution Check

| Principle | Assessment |
|---|---|
| Preserve runtime boundaries | Pass: Electron startup/IPC stays in `main.js`, bridge methods stay in `preload.js`, transport/live state stays in `server/server.js`, navigation logic stays in `server/nav.js`, and the player remains browser-only. |
| Keep shared state server-authoritative | Pass: clients send `NAV_ACTION` requests and converge on `NAV_STATE` or live snapshots. |
| Protect desktop and public-access boundaries | Pass for existing scope: the renderer remains sandboxed and context-isolated, preload methods are explicit, and external URL opening accepts only HTTP(S). Payload validation gaps are recorded below. |
| Preserve session data compatibility | Pass: no runtime broadcast, navigation, socket, or connection state is written to session JSON. |
| Match established code conventions | Pass: CommonJS/browser boundaries, naming, indentation, and uppercase WebSocket message types match the repository. |
| Testing and quality gates | Gap recorded: acceptance checks can be run manually, but there is no configured automated test framework and no automated checks can be claimed. |

No constitutional violation requiring a complexity exception was found. The documented validation and test gaps are deficiencies in the current implementation, not approved exceptions for future work.

## Complexity Assessment

| Dimension | Existing scope |
|---|---|
| Files containing the feature | 7 implementation/markup files |
| Total size of those files | Approximately 1,803 lines, including adjacent persistence, hacking, editor, and presentation behavior |
| Dedicated domain module | 1 file, 102 lines |
| Live transport module | 1 file, 186 lines, including adjacent hacking and sound integration |
| Runtime boundaries crossed | Player browser → WebSocket server → navigation domain; master renderer → preload → Electron main → server |
| External runtime dependencies | Existing `express` and `ws`; no migration change |
| Persistent schema impact | None |
| Existing automated coverage | None detected |

## Verification Strategy for the Existing Feature

Because the repository has no configured test framework, linter, formatter, CI workflow, or canonical test command, this migration records verification that should be performed rather than claiming it has passed.

### Focused automated tests recommended for follow-up

Before adding tests, adopt Node's built-in `node:test` runner to avoid a new runtime dependency:

- Location: `test/server/nav.test.js` and `test/server/server.test.js`.
- Command: add `"test": "node --test"` to `package.json`, then run `npm test`.
- Navigation cases: root defaults, valid direct-child actions, invalid IDs/types, back transitions, removed folders, removed entries, and removed commands.
- Protocol cases: malformed JSON, no-live requests, multi-client convergence, reconnect snapshots, live clear, and updates with repaired navigation.

### Manual Electron/browser checks

1. Run `npm start` and confirm the master displays a local player URL.
2. Open two player browsers and verify connection count changes.
3. Make a terminal live and confirm both browsers receive the same root state.
4. Alternate folder, entry, command, and back actions between browsers and verify convergence.
5. Connect a third browser after navigation and verify its initial snapshot matches.
6. Edit unrelated content and publish; verify valid navigation remains.
7. Delete the active folder, entry, and command in separate checks and verify update revalidation.
8. Stop the broadcast and verify every player returns to waiting state.
9. Disconnect and reconnect a browser and verify the connection overlay and current snapshot.

`npm run build:dir` is not required for this documentation-only migration. It should be run for future packaging-sensitive changes to server startup, preload wiring, or bundled client files when the Windows build environment is available.

## Identified Follow-up Gaps

1. **No automated verification** — pure navigation and protocol behavior have no regression suite.
2. **Unvalidated Electron payloads** — `main.js` forwards renderer-provided broadcast objects without validating their shape or terminal identity.
3. **Assumed tree shape** — full broadcast and navigation helpers assume a usable root and child collections.
4. **Update identity ambiguity** — update messages do not identify the intended live terminal and depend on the master renderer's local guard.
5. **Redundant unchanged broadcasts** — any string action, including unknown values, results in `NAV_STATE` after the domain leaves state unchanged.
6. **Duplicate count notifications** — the same socket can be processed by both error and close handlers.
7. **Initial count registration window** — the server's count callback is attached after startup and does not immediately replay the current set size.
8. **No explicit server teardown** — process exit is relied upon to close HTTP and WebSocket resources.
9. **No IPC acknowledgement** — the master UI cannot distinguish accepted broadcast operations from main/server failures.

