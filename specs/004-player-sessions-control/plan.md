# Implementation Plan: Player Sessions, Character Assignment, and Shared Terminal Control

**Branch**: `004-player-sessions-control` | **Date**: 2026-08-12 | **Spec**: `specs/004-player-sessions-control/spec.md`

## Summary

Add process-local logical browser sessions, a game-master-managed character roster, broadcast-scoped exclusive claims, and one explicitly assigned controller while preserving the existing server-authoritative navigation and hacking rules. A single ordered coordinator will serialize browser presence, roster and assignment changes, controller changes, player actions, broadcast lifecycle, active-terminal switches, and puzzle-preservation decisions; it will emit detached revisioned projections to the Wails master and browser clients before another transition can overtake them. The browser will use a profile-scoped opaque recognition token, personalized player context, and correlated authoritative action results so refreshes and multiple tabs converge without optimistic shared mutation. Durable version-1 session JSON remains unchanged, and `ForceHackSuccess` remains an exact private Wails-only operation.

## Project Structure

```text
specs/004-player-sessions-control/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
└── contracts/
    ├── desktop-coordination.md
    └── player-websocket.md

app.go                                      # Validated Wails coordination commands and runtime status
app_test.go                                 # Bridge validation, event replay, and private-operation coverage
main.go                                     # Coordinator, player server, and master-event composition

internal/
├── domain/
│   ├── model.go                            # Runtime broadcast/session/roster projections beside durable models
│   └── model_test.go                       # Detached projections and unchanged version-1 JSON boundary
├── live/
│   ├── service.go                          # Terminal/nav/hack mechanics and exact private puzzle checkpoints
│   └── service_test.go                     # Preserve/discard/restore and unchanged gameplay rules
├── control/
│   ├── service.go                          # Single ordered process-local coordination aggregate
│   └── service_test.go                     # Claims, roles, lifetimes, authorization, ordering, concurrency
├── player/
│   ├── client.go                           # Connection-to-logical-session dispatch context
│   ├── client_test.go                      # Queue and sender-context behavior
│   ├── protocol.go                         # Strict handshake, selection, action, context, and result envelopes
│   ├── protocol_test.go                    # Exact-field decoding and secret-free projection contracts
│   ├── server.go                           # Presence aggregation and selective/session-wide fanout
│   └── server_test.go                      # Multi-tab, reconnect, convergence, and selective delivery
├── platform/
│   └── assets_test.go                      # Player/master authority and UI-source contracts
└── testutil/testdata/protocol/             # Golden JSON for new player messages and projections

frontend/src/
├── desktop-api.js                          # Narrow coordination command/event facade
├── index.html                              # Roster/session panel and switch-decision dialog
├── master.css                              # Existing-aesthetic coordination and dialog styling
└── master.js                               # Authoritative master snapshot rendering and awaited commands

client/
├── index.html                              # Selection, waiting, role, and pending surfaces
├── client.css                              # Terminal-styled observer/read-only and selection states
└── client.js                               # Browser identity, personalized context, gating, and outcomes

tests/browser/
├── playwright.config.mjs                   # Run all player browser specifications
├── hacking-camouflage.spec.mjs             # Existing hacking interaction regression
└── player-sessions-control.spec.mjs         # Identity, selection, observer, pending, and switching journeys
```

**Structure Decision**: Add `internal/control` as the one outer transaction owner and keep `internal/live` focused on transport-independent terminal, navigation, and hacking mechanics. This avoids overloading the existing durable `internal/session` vocabulary, gives Wails and WebSocket callers the same ordering boundary, and retains the established `internal/player` and browser-only `client/` boundaries without a new runtime dependency.

## Constitution Check

The pre-research gate passes. The design stays within the modular-monolith boundaries, strengthens server authority, and deliberately keeps every new identity, claim, controller, connection, broadcast, and suspended-puzzle value outside version-1 session JSON.

| Principle | Assessment |
|---|---|
| I. Preserve Runtime Boundaries | PASS: `app.go` remains the validated private desktop facade; `internal/control` and `internal/live` own transport-independent coordination and gameplay; `internal/player` owns HTTP/WebSocket concerns; `client/` remains browser-only; and all affected surfaces are listed above. |
| II. Keep Shared State Server-Authoritative | PASS: every claim, control change, terminal action, and switch is validated and committed by one coordinator. Clients wait for revisioned server projections and correlated `ACTION_RESULT` messages and never apply canonical optimistic state. Secret hacking data stays private. |
| III. Protect Desktop and Public-Access Boundaries | PASS: the Wails facade exposes only the coordination methods in the desktop contract, validates identifiers and display names, and retains `ForceHackSuccess` only on the trusted master surface. No credential, filesystem, process, or desktop capability enters the player protocol. |
| IV. Preserve Session Data Compatibility | PASS: roster entries, logical sessions, browser recognition, presence, assignments, controller state, broadcast epochs, revisions, pending switches, and puzzle checkpoints are process-local. Durable `domain.Session` and version-1 JSON gain no field and require no migration. |
| V. Match Established Code Conventions | PASS: Go types and small integration interfaces follow existing packages; browser JavaScript, CSS, JSON, and uppercase message identifiers follow repository conventions; existing strict decoding and deterministic fakes are extended. No dependency is added. |

The post-design re-check uses the same assessments. `data-model.md` keeps runtime and durable entities separate, and both contracts preserve transport, privilege, and persistence boundaries; no Complexity Tracking table is required.

## Implementation Strategy

1. Introduce transport-independent runtime types for logical sessions, roster entries, broadcast epochs, exclusive assignments, controller identity, terminal runtime slots, ordered revisions, master snapshots, personalized player context, and action results. Keep the existing durable `Session` and `Terminal` JSON models unchanged and add explicit serialization regressions.
2. Build `internal/control.Service` around one mutex and one monotonic revision. Route connection attach/detach, roster changes, assignment changes, control reassignment, broadcast lifecycle, terminal switching, and all player actions through this service. During each accepted or rejected command, construct detached effects and enqueue a non-reentrant publication callback before releasing the transaction so mutation order and publication order cannot diverge.
3. Retain `internal/live` as the terminal mechanics boundary, but allow the coordinator to own multiple private terminal runtime slots. Preserve moves the exact active private state into a suspended slot; discard removes it; cancel changes nothing. Reactivation restores the private puzzle exactly while applying the latest authored terminal content through the existing navigation revalidation path. Deleting an active or suspended terminal must use the same explicit discard decision rather than bypassing the switch guard.
4. Separate process lifetime, roster lifetime, broadcast lifetime, and active-terminal lifetime. Process restart clears everything runtime-only; broadcast end clears claims, controller, active terminal, pending switch, and suspended puzzles but retains logical sessions, fallback names, and roster; a new broadcast issues a new opaque `broadcastId`; and a running broadcast may intentionally have no active terminal.
5. Create logical sessions from a server-issued browser-profile recognition token rather than from raw socket order. Under a cross-tab initialization lock, the browser sends the stored token when one exists; otherwise it completes one token-issuing handshake and saves the returned opaque value in `localStorage` before another tab connects. The returned process-local logical `sessionId` is read-only. The server maps one valid token to one logical session for its process lifetime, counts connection membership, changes presence only on first attach/last detach, and replaces an unknown token after restart while creating a fresh session and fallback name.
6. Extend strict WebSocket decoding with `SESSION_HELLO` and `CHARACTER_SELECT`. Add `requestId`, `broadcastId`, and `terminalId` to every shared player action so stale or duplicate requests can be rejected before gameplay mutation. Keep the existing `NAV_ACTION`, `HACK_GUESS`, and `HACK_PATTERN` identifiers and their gameplay-specific fields.
7. Add a complete personalized `PLAYER_STATE` projection for identity, current assignment/role, broadcast, roster availability, and active-terminal identity. Continue using `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `TERMINAL_CLEAR`, `NAV_STATE`, and `HACK_STATE` for canonical terminal content, but add the committed revision to their envelopes. Send role and assignment changes to every tab of the affected logical session without exposing connection details to players.
8. Add `ACTION_RESULT` for every selection or shared terminal request. The result correlates by `requestId`, identifies acceptance or a stable rejection reason, and carries the committed revision. The initiating tab remains pending until it has both the result and any required authoritative projection at that revision; a rejection with no canonical mutation releases pending immediately. Duplicate request IDs for one logical session return the original result and never repeat a mutation.
9. Authorize every navigation and hacking request inside the coordinator transaction: the connection must resolve to the current logical session, that session must hold a current-broadcast character assignment, it must be the controller, it must be connected, and `terminalId` must still be active. Only then invoke the unchanged `internal/live` navigation or hacking operation. Observer, unassigned, disconnected, stale, invalid, and inactive-terminal requests return rejection effects with no canonical gameplay, log, attempt, random, or outcome mutation.
10. Add validated Wails methods and one detached `coordination-state` event/status projection for roster CRUD, fallback-name changes, assignment/release/transfer, controller reassignment, broadcast start/end, terminal activation, and switch resolution. Update the master frontend to render that authoritative snapshot and await command results before changing visible runtime state. Preserve the existing `ForceHackSuccess` name, eligibility, and publication path.
11. Add an immersive character-selection view, assigned waiting view, active/observer badge, visibly read-only observer state, and pending state to the player frontend. Keep hover, focus, paging, sound, and other local feedback available where harmless, but gate every shared send path in the UI and rely on the same server authorization for crafted requests. Active-terminal changes reuse the existing loading presentation and never require identity or character selection again.
12. Extend deterministic Go, asset-contract, and Playwright coverage. Include the specification's 100-trial claim/controller races, multi-tab last-close presence, reconnect/restart lifetimes, stale broadcast and terminal actions, action-versus-reassignment ordering, exact puzzle preservation, broadcast resets, all-tab convergence, authoritative pending completion, and proof that player assets and protocol expose no `ForceHackSuccess` path.

## Verification Gates

- `gofmt -l .` reports no Go source paths.
- `go vet ./...` succeeds.
- `go test ./...` succeeds, including 100 concurrent same-character trials, 100 concurrent first-controller trials, detached projections, exact no-mutation rejection comparisons, process/broadcast/terminal lifetime coverage, and unchanged version-1 JSON.
- `go test -race ./...` succeeds for concurrent claims, first-controller assignment, connection membership, action/reassignment ordering, duplicate requests, and ordered publication.
- `npm --prefix frontend run build` succeeds with the generated-binding fallback intact.
- `npm --prefix tests/browser test` succeeds for selection, multi-tab identity, observer read-only behavior, accepted/rejected pending outcomes, loading transitions, terminal switches, and existing hacking camouflage interactions.
- `wails dev` passes the private game-master journeys for roster/session management, claim corrections, controller reassignment, disconnected-controller status, preserve/discard/cancel, broadcast end/restart, and the unchanged `ForceHackSuccess` control. If unavailable, report the reason rather than claiming success.
- A clean `wails build` remains the packaging gate for a self-contained macOS application; signing/notarization checks remain release-only when credentials are available.
- Review confirms there are no new version-1 session fields, no player path to `ForceHackSuccess`, no private connection data in player projections, and no optimistic canonical browser transition.
