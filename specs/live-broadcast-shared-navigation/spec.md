---
status: migrated
feature: Live Broadcast and Shared Navigation
source: existing implementation
---

# Feature Specification: Live Broadcast and Shared Navigation

**Migration status**: Reverse-engineered from the existing implementation on 2026-08-09  
**Scope**: Local player-server startup, live-terminal lifecycle, connection reporting, server-authoritative shared navigation, live content refresh, and reconnect synchronization

## Purpose

Live broadcast turns one game-master-selected terminal into a shared browser experience. The Wails/Go application starts an embedded HTTP and WebSocket server, the game master controls which terminal is live, and every connected player observes one canonical navigation position. Player input is sent as a request; only state returned by the server changes the shared folder, entry, or command view.

The hacking rules and player-facing visual treatment are documented by their own migrated features. This specification describes only how those systems attach to the live snapshot where necessary for compatibility.

## User Scenarios and Acceptance

### User Story 1 — Publish a terminal to players (Priority: P1)

As a game master, I can make the terminal I am editing live so that connected players receive its current content and begin at a predictable navigation position.

**Independent verification**: Open a session, select a terminal, make it active, and inspect two connected player browsers.

**Acceptance scenarios**:

1. **Given** an editable terminal is selected, **when** the game master makes it active, **then** its ID, name, content tree, hacking level, introduction text, and a root navigation state become the canonical live state.
2. **Given** one or more players are connected, **when** a terminal becomes live, **then** every connected player receives a `TERMINAL_LIVE` snapshot of the same public state.
3. **Given** a terminal is already live and players have navigated away from the root, **when** the game master restarts that broadcast or makes another terminal active, **then** shared navigation resets to the root list.
4. **Given** no terminal is selected in the editor, **when** broadcast controls render, **then** the make-live action is unavailable.
5. **Given** a terminal is live, **when** the game master stops the broadcast, **then** canonical live state is cleared and all connected players receive `TERMINAL_CLEAR`.

---

### User Story 2 — Join or rejoin the current broadcast (Priority: P1)

As a player, I can connect after a broadcast has begun or reconnect after losing the connection and receive the current shared state rather than an obsolete starting state.

**Independent verification**: Start a broadcast, navigate into nested content in one browser, then connect or reconnect a second browser.

**Acceptance scenarios**:

1. **Given** a terminal is live, **when** a player WebSocket connection opens, **then** that connection receives a `TERMINAL_LIVE` snapshot containing the current live content and current navigation state.
2. **Given** no terminal is live, **when** a player connects, **then** no live snapshot is sent and the player remains in the waiting state.
3. **Given** a connected browser loses its WebSocket, **when** the close is observed, **then** it displays a connection-loss state and schedules another connection attempt after approximately three seconds.
4. **Given** shared navigation changes while a player is disconnected, **when** that player reconnects, **then** the initial live snapshot reflects the new canonical position.
5. **Given** the page is served through HTTPS, **when** its WebSocket is created, **then** it uses `wss`; otherwise it uses `ws`.

---

### User Story 3 — Navigate one shared terminal (Priority: P1)

As a group of players, we share one folder, entry, or command position so that an action from any browser updates the terminal for everyone.

**Independent verification**: Connect two browsers and alternate folder, entry, command, and back actions between them while comparing both views.

**Acceptance scenarios**:

1. **Given** players are viewing a folder, **when** one player requests entry into a direct child folder, **then** the server appends that folder to the canonical path and broadcasts the resulting `NAV_STATE`.
2. **Given** players are viewing a folder, **when** one player requests a direct child entry, **then** the server switches canonical navigation to entry mode, records that entry, clears active command output, and broadcasts the result.
3. **Given** players are viewing a folder, **when** one player requests a direct child command, **then** the server records that command and all players derive its output from the shared content tree.
4. **Given** players are viewing an entry, **when** any player requests back, **then** canonical navigation returns to the containing folder list.
5. **Given** players are inside a nested folder list, **when** any player requests back, **then** the final folder is removed from the canonical path.
6. **Given** a navigation target is not a direct child of the current folder with the requested type, **when** the request is applied, **then** canonical navigation does not move to that target.
7. **Given** multiple player browsers receive a navigation result, **when** they apply it, **then** folder path, entry mode, viewed entry, and active command converge while pointer or keyboard highlight remains local to each browser.

---

### User Story 4 — Refresh edited live content safely (Priority: P1)

As a game master, I can publish edits to the live terminal without unnecessarily restarting the broadcast, while stale shared navigation is repaired against the new tree.

**Independent verification**: Navigate into nested content, edit and publish unrelated content, then remove the active folder, entry, or command and publish again.

**Acceptance scenarios**:

1. **Given** the edited terminal is the live terminal, **when** the game master publishes its content, **then** the server replaces the live tree, updates the introduction text, revalidates navigation, and broadcasts `TERMINAL_UPDATE`.
2. **Given** the current folder path still exists after an update, **when** navigation is revalidated, **then** that path remains active.
3. **Given** one or more folders in the current path no longer exist or are no longer folders, **when** navigation is revalidated, **then** the path is truncated at the last valid folder.
4. **Given** the active entry no longer exists or is no longer an entry, **when** navigation is revalidated, **then** the shared view returns to list mode and clears the entry reference.
5. **Given** the active command is no longer a direct child of the revalidated current folder, **when** navigation is revalidated, **then** the command reference is cleared.
6. **Given** no terminal is live, **when** a live-update request reaches the server, **then** it has no effect.
7. **Given** a live update is published, **when** players apply it, **then** the active hacking puzzle is not regenerated and the live terminal identity is not replaced.

---

### User Story 5 — Monitor player access (Priority: P2)

As a game master, I can see where players should connect and how many player WebSockets are currently registered so that I can monitor the table setup.

**Independent verification**: Start the application, open and close multiple player browsers, and observe the address and connected count in the master window.

**Acceptance scenarios**:

1. **Given** the embedded server starts successfully, **when** the master window loads, **then** it receives the selected non-internal IPv4 address and port as the player URL.
2. **Given** no non-internal IPv4 interface is found, **when** the URL is constructed, **then** `localhost` is used as the host fallback.
3. **Given** a player WebSocket connects, **when** it is registered, **then** the connected-client count increases and is reported to the master frontend.
4. **Given** a registered WebSocket closes or errors, **when** it is removed, **then** the current connected-client count is reported to the master frontend.
5. **Given** the game master selects the displayed HTTP or HTTPS address, **when** the bound desktop method accepts it, **then** the address opens externally; malformed and unsupported protocols are ignored.

## Functional Requirements

### Server startup and connection lifecycle

- **FR-001**: The Wails/Go application MUST start the embedded server before presenting the master interface.
- **FR-002**: The server MUST listen on port `3690` by default and bind to `0.0.0.0`.
- **FR-003**: The HTTP application MUST serve the browser player application from `client/`.
- **FR-004**: The server MUST report a player URL using the first detected non-internal IPv4 address, falling back to `localhost`.
- **FR-005**: The server MUST maintain the set of registered player WebSockets and notify the desktop event layer of count changes on connection, close, and error.
- **FR-006**: A newly connected player MUST receive the current public live snapshot when one exists.
- **FR-007**: The player client MUST reconnect approximately three seconds after a WebSocket close and MUST select `ws` or `wss` from the page protocol.

### Live-terminal lifecycle

- **FR-008**: The server MUST own either one canonical live-terminal object or no live terminal.
- **FR-009**: Starting or restarting a broadcast MUST install the supplied terminal identity, tree, hacking configuration, and introduction text and MUST reset navigation to its default state.
- **FR-010**: A public `TERMINAL_LIVE` payload MUST include `terminalId`, `terminalName`, `tree`, `hackLevel`, `introText`, the public hacking projection, and `nav`.
- **FR-011**: Stopping a broadcast MUST clear the canonical live object and broadcast `TERMINAL_CLEAR`.
- **FR-012**: Updating a live terminal MUST replace its tree, optionally replace its introduction text, revalidate navigation, and broadcast `TERMINAL_UPDATE` without replacing its identity or hacking state.
- **FR-013**: Live updates received when no terminal is live MUST be ignored.

### Server-authoritative navigation

- **FR-014**: Default navigation MUST be `{ path: ['root'], mode: 'list', viewEntryId: null, commandNodeId: null }`.
- **FR-015**: Player navigation MUST be represented as a `NAV_ACTION` request and MUST NOT be applied canonically by the browser before the server responds.
- **FR-016**: The server MUST ignore malformed JSON and all player messages while no terminal is live.
- **FR-017**: The server MUST process navigation messages only when `type` is `NAV_ACTION` and `action` is a string.
- **FR-018**: Folder entry, entry opening, and command activation MUST resolve only direct children of the current folder and MUST require the matching node type.
- **FR-019**: Back navigation MUST close an entry before moving up a folder path and MUST never move above the root.
- **FR-020**: After handling a syntactically eligible navigation request, the server MUST broadcast the resulting canonical `NAV_STATE`, including when validation leaves it unchanged.
- **FR-021**: Player clients MUST replace their mirrored path, viewed entry, and active command from server state and MUST reset their local selection index.
- **FR-022**: A hacking gate MAY prevent the displayed mode from changing immediately, but incoming navigation fields MUST still mirror the server state for use after the gate resolves.

### Update revalidation and desktop boundary

- **FR-023**: Revalidation MUST preserve only the contiguous valid folder prefix beginning at `root`.
- **FR-024**: Revalidation MUST clear an invalid entry reference and MUST clear a command that is not a direct child of the revalidated current folder.
- **FR-025**: The sandboxed master frontend MUST control broadcast operations only through explicit bound methods exposed by the compatibility facade.
- **FR-026**: The Go composition root MUST route make-live, update-live, and clear-live calls to the authoritative live/player services.
- **FR-027**: Runtime live, navigation, and connection state MUST remain in memory and MUST NOT be added to saved session JSON.

## Protocol Contract

| Message | Direction | Existing payload | Existing validation and effect |
|---|---|---|---|
| `TERMINAL_LIVE` | Server → player | Public live state | Sent to all players on start/restart and directly to a newly connected player when live state exists |
| `TERMINAL_UPDATE` | Server → player | `tree`, `introText`, `nav` | Sent after replacing live content and revalidating navigation |
| `TERMINAL_CLEAR` | Server → player | Type only | Sent after canonical live state is cleared |
| `NAV_ACTION` | Player → server | `action`, optional `nodeId` | Requires live state and string `action`; target membership and type are checked by the navigation domain |
| `NAV_STATE` | Server → player | `nav` | Broadcast after every syntactically eligible navigation request, whether or not navigation changed |

The existing protocol has no version field. This migration documents current behavior and introduces no compatibility change.

## Success Criteria

- **SC-001**: In a two-browser manual test, every accepted folder, entry, command, and back action results in matching server-owned navigation fields in both browsers without reloading either page.
- **SC-002**: A browser connected after navigation has changed renders the same current folder or entry as already connected browsers from its first live snapshot.
- **SC-003**: Restarting or switching a broadcast returns all connected browsers to root navigation, while publishing an ordinary live edit preserves every still-valid navigation component.
- **SC-004**: Removing the active folder, entry, or command and publishing leaves no player browser referencing deleted content.
- **SC-005**: Stopping a broadcast returns every connected player to the waiting state and a later connection receives no stale live snapshot.
- **SC-006**: Opening and closing two player WebSockets causes the master display to report the corresponding registered-client counts after each state change.
- **SC-007**: Malformed JSON and navigation targets outside the current folder do not move canonical navigation or crash the server.

## Assumptions and Existing Constraints

- One application process owns one embedded server and at most one live terminal.
- All player browsers intentionally share navigation; only the highlighted row is local per browser.
- The content tree is expected to have a root node whose ID is `root` and folder nodes with `children` arrays.
- Editing session data does not automatically publish all changes; the game master explicitly updates the live terminal, except that applying live introduction settings also sends a live update.
- Renaming a live terminal is intentionally deferred until the next full broadcast because a full broadcast also resets runtime state.
- Authentication and public tunneling are outside this feature and are documented separately when migrated.

## Migration Status of Previously Identified Gaps

1. Automated tests now cover the pure navigation domain, WebSocket protocol, desktop bridge contract, reconnect snapshot, and multi-client convergence.
2. Bound broadcast payloads are structurally validated at the privileged Go boundary.
3. `setLiveTerminal` assumes a valid payload and content tree.
4. `TERMINAL_UPDATE` intentionally carries no terminal ID; the master frontend's local state and the live service's active-state check guard the update boundary.
5. Unknown string navigation actions produce a redundant unchanged `NAV_STATE` instead of an explicit rejection.
6. WebSocket `error` and `close` can both remove the same socket and emit redundant count notifications.
7. The connection-count callback is registered after server startup, so a connection in that narrow interval is not reported until another count change.
8. The HTTP/WebSocket server exposes no explicit graceful-shutdown operation.
