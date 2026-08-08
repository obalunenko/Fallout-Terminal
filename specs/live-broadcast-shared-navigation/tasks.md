---
status: migrated
feature: Live Broadcast and Shared Navigation
source: existing implementation
---

# Tasks: Live Broadcast and Shared Navigation

**Migration status**: Reconstructed from the existing implementation on 2026-08-09  
**Completion convention**: Every implementation task is marked complete because it describes behavior already present in the repository.

## Phase 1 — Shared Navigation Domain

- [x] T001 Define root-list navigation defaults with folder path, display mode, entry reference, and command reference in `server/nav.js`.
- [x] T002 Add recursive node lookup and current-folder traversal helpers in `server/nav.js`.
- [x] T003 Validate folder entry against direct child folders of the canonical current folder in `server/nav.js`.
- [x] T004 Validate entry and command activation against direct children with matching node types in `server/nav.js`.
- [x] T005 Implement entry-close and folder-pop back behavior without allowing navigation above root in `server/nav.js`.
- [x] T006 Revalidate a folder path to its longest contiguous valid prefix after content changes in `server/nav.js`.
- [x] T007 Clear removed or mistyped entry references and commands outside the revalidated current folder in `server/nav.js`.

## Phase 2 — Embedded Server and Connection Tracking

- [x] T008 Create the Express application, HTTP server, and WebSocket server in `server/server.js`.
- [x] T009 Serve browser player assets from `client/` and listen on `0.0.0.0:3690` by default in `server/server.js`.
- [x] T010 Discover the first non-internal IPv4 address with a `localhost` fallback and return the player URL from server startup in `server/server.js`.
- [x] T011 Register connected WebSockets in a shared set, send only to open sockets, and report count changes in `server/server.js`.
- [x] T012 Parse incoming JSON defensively and ignore player messages when no terminal is live in `server/server.js`.

## Phase 3 — Canonical Live-Terminal Lifecycle

- [x] T013 Store at most one canonical live-terminal object with identity, content, configuration, hacking integration, and navigation in `server/server.js`.
- [x] T014 Start or restart a broadcast with default navigation and send a public `TERMINAL_LIVE` snapshot to all connected players in `server/server.js`.
- [x] T015 Send the current public live snapshot directly to each newly connected or reconnecting player in `server/server.js`.
- [x] T016 Replace live tree and introduction content, revalidate navigation, and broadcast `TERMINAL_UPDATE` without resetting other runtime state in `server/server.js`.
- [x] T017 Clear canonical live state and broadcast `TERMINAL_CLEAR` in `server/server.js`.
- [x] T018 Delegate eligible `NAV_ACTION` messages to the navigation domain and broadcast the resulting canonical `NAV_STATE` in `server/server.js`.

## Phase 4 — Electron and Game-Master Controls

- [x] T019 Start the embedded server before creating the master window and forward its player address in `main.js`.
- [x] T020 Forward server connection-count callbacks to the master renderer in `main.js`.
- [x] T021 Route make-live, update-live, and clear-live IPC messages to the server module in `main.js`.
- [x] T022 Expose narrow server-information, connection-count, and broadcast-control methods through `preload.js` while preserving renderer sandboxing.
- [x] T023 Render the player address, connected count, live marker, start/restart, publish, and stop controls in `master/index.html` and `master/master.js`.
- [x] T024 Keep editor selection separate from live-terminal selection and allow updates only when the edited terminal is locally recognized as live in `master/master.js`.
- [x] T025 Send terminal identity, tree, hacking configuration, and introduction content on full broadcast while sending only tree and introduction content on live update in `master/master.js`.
- [x] T026 Clear a live broadcast when its terminal is deleted and reset local live indicators after an explicit stop in `master/master.js`.

## Phase 5 — Player Protocol and Convergence

- [x] T027 Connect the player browser with `ws` or `wss` according to the page protocol and schedule reconnect after a close in `client/client.js`.
- [x] T028 Dispatch `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `NAV_STATE`, and `TERMINAL_CLEAR` into mirrored browser state in `client/client.js`.
- [x] T029 Send folder, entry, command, and back operations as typed `NAV_ACTION` requests without optimistic canonical transitions in `client/client.js`.
- [x] T030 Replace mirrored path, entry, and command fields from server navigation and reset only the local row selection in `client/client.js`.
- [x] T031 Preserve incoming navigation beneath the hacking display so canonical state is ready when the gate resolves in `client/client.js`.
- [x] T032 Derive command output from the current shared tree and canonical command node reference in `client/client.js`.
- [x] T033 Return the browser to its waiting state and stop live presentation behavior when `TERMINAL_CLEAR` arrives in `client/client.js`.

## Phase 6 — Existing Security and Compatibility Evidence

- [x] T034 Keep player transport and navigation within browser APIs, server modules, and preload IPC without granting renderers Node.js access.
- [x] T035 Keep live terminal, navigation, socket, and client-count state in memory rather than adding it to versioned session JSON.
- [x] T036 Restrict external player-address opening to valid HTTP and HTTPS URLs in `main.js`.
- [x] T037 Preserve the existing unversioned WebSocket payloads and reconnect-by-snapshot behavior without a migration-time protocol change.

## Gaps Identified During Migration

- No automated tests exist for `server/nav.js`, live lifecycle, WebSocket messages, IPC routing, reconnect snapshots, or multi-client convergence.
- Broadcast IPC and `setLiveTerminal` do not structurally validate their payloads or content tree.
- Update requests contain no terminal ID and rely on the master renderer's local live-terminal guard.
- Unknown string navigation actions trigger an unchanged `NAV_STATE` broadcast rather than an explicit rejection.
- WebSocket error and close handlers can produce redundant client-count notifications for the same socket.
- Registering the connection-count callback does not immediately report sockets connected before registration.
- The embedded HTTP/WebSocket server has no explicit graceful-shutdown API.
- Broadcast IPC is fire-and-forget, so the master renderer receives no success or failure acknowledgement.

