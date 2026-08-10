# Implementation Plan: Wails v2 Runtime Migration

**Branch**: `001-wails-v2-migration` | **Date**: 2026-08-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-wails-v2-migration/spec.md`

**Note**: This plan prepares the migration but does not authorize deleting the working Electron runtime before parity gates pass.

**Bugfix**: 2026-08-09 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-08-09 — BUG-002 Updated from bugfix patch

**Bugfix**: 2026-08-09 — BUG-003 Updated from bugfix patch

**Bugfix**: 2026-08-09 — BUG-004 Updated from bugfix patch

**Scope decision**: 2026-08-09 — The active macOS acceptance profile is personal use. Local/ad-hoc `.app` validation gates cutover; Developer ID signing, notarization, and DMG checks are conditional future public-release gates.

## Summary

Replace the Electron/Node runtime with a Go 1.26 modular desktop monolith hosted by Wails v2.13.0 while preserving the existing master and player browser experiences, version-1 session files, WebSocket messages, server-authoritative live state, optional authenticated ngrok access, and a personal-use macOS Apple Silicon application. The master UI moves into a vanilla-JavaScript Wails frontend behind a compatibility adapter; the player UI remains an independent browser application served by a dedicated embedded Go HTTP/WebSocket server because the Wails v2 asset server does not support WebSockets. Existing Developer ID/notarization automation remains available as an optional public-distribution profile but is not a cutover dependency.

The complete development system starts with one repository-root command, `wails dev`: Wails runs the configured frontend install/build/watcher steps and launches the Go composition root, which acquires the player listener before reporting the desktop ready. The packaged `.app` uses the same composition root so one application launch starts every required runtime component without Node, npm, a frontend command, or a separate server process.

The implementation follows a strangler sequence: add tested Go services beside the working JavaScript implementation, move one behavior boundary at a time, verify contract parity using existing fixtures and new Go tests, package the Wails candidate, and remove Electron only after all acceptance gates pass.

## Technical Context

**Language/Version**: Go 1.26 for desktop, domain, persistence, HTTP/WebSocket, and process code; existing browser JavaScript/HTML/CSS retained

**Primary Dependencies**: Wails v2.13.0; `github.com/coder/websocket` v1.8.15; Go standard library; Vite with vanilla JavaScript for the master frontend build; external ngrok executable remains optional

**Storage**: Backward-compatible version-1 JSON session files at user-selected paths; new/save dialogs default to `~/Documents/Fallout Terminal/Sessions/`; bundled demo remains read-only until explicitly copied; app metadata uses `~/Library/Application Support/com.vaulttec.fallout-terminal/`; ephemeral live/navigation/hacking/client state remains in Go memory

**Testing**: Go `testing`, `httptest`, and race detector with colocated `*_test.go` files; deterministic fakes for dialogs/filesystem/process seams; manual Wails/WebKit, real-browser, macOS storage, bundle-integrity, ad-hoc-signature, and single-launch scenarios from `quickstart.md`; conditional Gatekeeper/notarization scenarios only for the public profile

**Target Platform**: macOS 13+ on Apple Silicon, with current browser clients on local HTTP/WebSocket or authenticated ngrok HTTPS/WSS; Intel/universal macOS and Windows are deferred

**Project Type**: Single-process modular desktop monolith containing one Wails master window and one separately listening embedded player server

**Performance Goals**: Usable workspace and player address within 5 seconds on the reference host; 25 sequential mixed player actions converge across 4–7 clients; 20 rapid accepted edits persist the newest revision; graceful shutdown completes within 10 seconds before forced child termination

**Constraints**: Preserve session version and public protocol; keep privileged APIs narrow; keep live state server-authoritative; do not serve player WebSockets through Wails AssetServer; use `wails dev` as the sole repository-root development startup command after prerequisites; make packaged launch own all runtime services; one desktop window only; keep Electron releasable until parity; package all browser/audio/font/sample assets

**Scale/Scope**: One game-master process, one desktop window, one player listener, one active terminal, at most one ngrok child, and a supported and tested operating envelope of 4–7 concurrent player connections

## Constitution Check

*GATE: Evaluated before research and re-evaluated after design.*

### Pre-design assessment

| Constitution principle | Assessment | Plan evidence |
|---|---|---|
| I. Preserve Runtime Boundaries | PASS | `main.go`/`app.go` compose lifecycle; `frontend/`, `client/`, and `internal/` retain separate ownership. `wails dev` orchestrates them without merging their boundaries. |
| II. Keep Shared State Server-Authoritative | PASS | `internal/live` owns canonical navigation and hacking state and creates sanitized snapshots before player broadcasts. |
| III. Protect Desktop and Public-Access Boundaries | PASS | The registered bridge stays narrow, external URLs are allowlisted, and public tunnel startup remains credential-gated and cleanup-owned. |
| IV. Preserve Session Data Compatibility | PASS | Version-1 JSON remains user-owned, runtime state stays process-local, and macOS storage paths follow the governed policy. |
| V. Match Established Code Conventions | PASS | Go code uses standard package/test conventions; retained browser files keep existing JavaScript, CSS, JSON, and protocol naming. |
| Dependency Rules | PASS | Domain packages stay transport-independent and the player server remains independent of the master frontend. |
| Testing and Quality Gates | PASS | The plan includes formatting, vet, unit/integration/race checks, `wails dev`, personal-use packaged-app and rollback gates, plus conditional signing/notarization checks for future public distribution. |
| Development Workflow | PASS | Electron remains the behavioral oracle until the Wails package and one-command startup gates pass. |

### Governance amendment

Constitution v2.0.0 records the user-approved transition rules: Electron remains the behavioral oracle during migration, Go/Wails boundaries and quality gates govern the candidate, macOS Apple Silicon is the first package target, and legacy deletion remains gated on acceptance.

### Post-design assessment

The final design remains PASS for every row above. The startup contract changes orchestration only: it keeps frontend, desktop, player-server, domain, persistence, and tunnel ownership separate, adds no privileged frontend surface, and preserves the Electron release path until the final cutover task.

## Project Structure

### Documentation (this feature)

```text
specs/001-wails-v2-migration/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── desktop-bridge.md
│   ├── http-player-server.md
│   ├── session-v1.md
│   ├── startup.md
│   └── websocket-protocol.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Target source structure

```text
main.go                         # Wails configuration and embedded assets
app.go                          # Lifecycle and bound desktop facade
go.mod
go.sum
wails.json                      # Wails build plus frontend install/build/dev orchestration
build/                          # Wails icons and macOS application metadata
frontend/
├── package.json                # Vite build dependencies only
├── package-lock.json
├── vite.config.js
├── src/
│   ├── index.html              # migrated master/index.html
│   ├── master.css
│   ├── master.js
│   └── desktop-api.js          # electronAPI-compatible Wails adapter
└── wailsjs/                    # generated Wails bindings/runtime declarations
internal/
├── domain/
│   ├── model.go                # Session, Terminal, Node, NavState, live projections
│   └── validate.go
├── hack/
│   ├── hack.go
│   └── wordbank.go
├── nav/
│   └── nav.go
├── session/
│   ├── service.go              # dialogs, active path, ordered atomic writes
│   └── storage.go
├── live/
│   └── service.go              # canonical live state and public projections
├── player/
│   ├── server.go               # net/http routes and lifecycle
│   ├── client.go               # per-connection outbound queue
│   └── protocol.go
├── tunnel/
│   ├── service.go
│   ├── process.go
│   ├── process_darwin.go
│   └── process_other.go
└── platform/
    └── paths.go
client/                         # retained player HTML/CSS/JS/font/sound assets
sessions/
└── demo.json                   # unchanged version-1 fixture/sample
```

The existing `main.js`, `preload.js`, `server/`, root npm files, and Electron packaging configuration remain during parallel migration and are deleted only in the final cutover phase. Generated build output and dependencies remain ignored.

**Structure Decision**: Go packages follow the current behavioral boundaries rather than mechanically mirroring JavaScript files. `frontend/` contains only the game-master webview; `client/` remains separate because it is served to remote browsers. Root `main.go` owns both embedded filesystems so Go embed patterns never traverse parent directories. `wails.json` makes `wails dev` the sole development orchestrator, while the Wails facade composes runtime services without absorbing domain logic.

## Contract and State Design

### Session JSON

The persisted schema remains version 1 and is documented in [contracts/session-v1.md](contracts/session-v1.md). Typed validation covers terminal and recursive node variants, bounds, and required root semantics. Custom decoding/marshalling preserves compatible unknown fields rather than silently deleting them. Saves are serialized by monotonically increasing revision, written to a same-directory temporary file, flushed, and atomically replaced where the platform permits. Failed opens never replace the active path. On macOS, the app bundle is never treated as writable storage: user sessions default to Documents after confirmation, the bundled demo is copied only on explicit action, and internal metadata is isolated in Application Support.

### Desktop bridge

[contracts/desktop-bridge.md](contracts/desktop-bridge.md) replaces Electron IPC with bound Go methods and Wails events. During staged migration, `frontend/src/desktop-api.js` exposes the current `window.electronAPI` method names so `master.js` needs minimal behavioral change. BUG-004 makes that name explicitly transitional: final active source publishes and consumes the same narrow contract as `window.desktopAPI`, with no Electron-specific global remaining. Commands include session new/open/save, live set/update/clear, force-success, validated URL open, and initial runtime status. Events remain `server-info`, `client-count`, and `hack-state`. User cancellations and expected failures use structured results, not unhandled promise rejection.

### HTTP and WebSocket

[contracts/http-player-server.md](contracts/http-player-server.md) and [contracts/websocket-protocol.md](contracts/websocket-protocol.md) preserve current routes and messages. A dedicated `net/http.Server` listens on `0.0.0.0:3690`, serves the embedded `client/` filesystem and sound-list endpoint, and upgrades the root path to WebSocket when requested. Each player connection has one writer goroutine and a bounded outbound queue; canonical state mutations are protected by the live-service lock, and immutable snapshots are broadcast outside the lock.

### Live-state lifecycle

`LiveService` owns either no live state or one `LiveState`. Set-live creates a fresh navigation state and optional puzzle; update-live replaces content and revalidates navigation without resetting puzzle state; clear-live removes all canonical state. Player actions validate against the locked current state and produce sanitized immutable projections. New connections request the current projection. Runtime state never enters session persistence.

### Startup orchestration

[contracts/startup.md](contracts/startup.md) defines the single entry boundary. From a prepared checkout, `wails dev` runs at the repository root and delegates frontend install/build/watch behavior through `wails.json`; users never start Vite or the player server separately. Both development and packaged launches call the same idempotent application lifecycle: acquire the player listener, publish ready status only after the listener and desktop bridge are usable, optionally start the tunnel, and unwind owned resources in reverse order on failure or shutdown. BUG-003 extends this boundary to handled supervisor interruption: tunnel ownership cannot depend solely on the native window shutdown callback, and the regression harness must prove that interrupting the sole development command removes the tunnel, listener, and policy material within the shutdown timeout.

## Implementation Phases

### Phase 0: Research and decisions

- Pin Wails v2.13.0, Go 1.26, and coder/websocket v1.8.15.
- Confirm one Wails window and a separate player HTTP/WebSocket server.
- Select vanilla JavaScript + Vite with an `electronAPI` compatibility adapter.
- Select Go standard testing/race tooling and deterministic boundary fakes.
- Define macOS application-bundle, session-location, child-process, signing, notarization, and DMG behavior.
- Record all decisions and rejected alternatives in [research.md](research.md).

### Phase 1: Contracts and data design

- Freeze version-1 session JSON and current player protocol before porting.
- Define typed durable, private runtime, public projection, connection, and tunnel models in [data-model.md](data-model.md).
- Define desktop, HTTP, WebSocket, and session contracts under `contracts/`.
- Define the repository command, packaged-launch, readiness, failure, and shutdown contract in [contracts/startup.md](contracts/startup.md).
- Define runnable parity and packaging validation in [quickstart.md](quickstart.md).
- Apply the user-approved constitution amendment before implementation alters the governed runtime paths.

### Phase 2: Go foundations

- Scaffold the Wails project without removing Electron.
- Configure `wails.json` so `wails dev` owns frontend dependency installation, asset building, and the Vite watcher from the repository root.
- Implement models/validation and ordered atomic session persistence.
- Port navigation, hacking, word bank, and sanitized projections with table-driven tests.
- Implement live-state synchronization and race-tested concurrent access.
- Implement the player server, connection queues, embedded assets, sound endpoint, and graceful shutdown.

### Phase 3: Game-master and player integration

- Migrate the master assets into the Wails frontend.
- Add generated bindings and the compatibility adapter; retain the transitional Electron-specific global only until final cutover, then expose the same facade under a runtime-neutral name.
- Implement native dialogs, initial status retrieval, Wails events, external URL validation, and lifecycle wiring.
- Exercise unchanged player rendering, reconnect, audio degradation, and multi-client convergence against the Go server.
- Preserve one authoritative player-state visibility mechanism so inactive idle, normal, hacking, and blocked containers are excluded from layout even when the terminal body overflows and scrolls.

### Phase 4: Public access and profile-aware packaging

- Port credential validation, temporary policy creation, URL discovery, timeout, diagnostics, and cleanup.
- Add Darwin-specific process-group/termination behavior and bounded graceful/forced shutdown.
- Add a handled development-supervisor interruption harness and an ownership strategy that removes the active tunnel, player listener, and temporary policy material even when the native window shutdown callback is bypassed.
- Configure macOS Apple Silicon assets and application metadata; produce and validate the local/ad-hoc personal-use `.app` as the current required candidate.
- Preserve hardened runtime, Developer ID signing, `notarytool` submission, stapling, and DMG creation as an optional public-release profile that fails closed when selected without credentials.
- Run all automated and manual parity gates required by the selected distribution profile.

### Phase 5: Cutover

- Update README and rollback guidance to identify personal use as the current accepted profile and public distribution as conditional.
- Remove Electron orchestration, preload, Express/`ws`, electron-builder dependencies, and obsolete root npm files.
- Replace the transitional `window.electronAPI` facade name with `window.desktopAPI` in active source while preserving the narrow command/event contract.
- Re-run clean-checkout tests, production builds, session round trips, four-to-seven-browser tests, public-access checks, and packaged shutdown checks.
- After the last post-Electron build, designate one canonical candidate commit and executable SHA-256 and synchronize that identity across quickstart evidence, rollback guidance, and release handoff.
- Retain the last Electron release artifact as rollback until the personal-use Wails candidate is accepted; public-profile acceptance is required only before later public publication.

## Verification Plan

| Surface | Automated check | Manual check | Expected result |
|---|---|---|---|
| Models, navigation, hacking | `go test ./internal/domain ./internal/nav ./internal/hack` | Compare representative scenarios with existing specs | Exact transitions and sanitized projections |
| Session persistence | `go test ./internal/session` | Create/open/edit/reopen version-1 fixtures | Latest revision round-trips without semantic loss |
| Live and protocol | `go test -race ./internal/live ./internal/player` | 4, 5, 6, and 7 browsers; 25 actions; reconnect | Race-free identical state after every action |
| Desktop facade | `go test ./...` with fake dialog/storage/server/process seams plus an active-source bridge-name contract check | `wails dev` master authoring journey | Narrow methods/events and actionable failures through `window.desktopAPI`; no active `window.electronAPI` remains |
| Startup orchestration | Application lifecycle tests in `app_test.go` with fake player/tunnel/event seams plus the handled-supervisor-interrupt harness | From a prepared clean checkout run only `wails dev`; interrupt a public-mode development run; separately launch the packaged `.app` once | Workspace and player address are ready within 5 seconds; no separate frontend or player-server command is required; handled interrupt leaves zero owned resources |
| Player presentation | Server integration tests, HTTP asset checks, and the player-state visibility contract test | Pointer, keyboard, reveal, missing-audio, and full overflow-scroll journey in every presentation state | Existing player behavior remains usable and inactive state messages never enter the scrollable layout |
| Public tunnel | `go test ./internal/tunnel` with fake process and handled development-interrupt coverage | Authenticated HTTP/WSS ngrok smoke followed by native Quit and supervisor interrupt when credentials are supplied | Fail-closed startup and deterministic cleanup without manual process termination |
| Static analysis | `go vet ./...` and `gofmt -l .` | Inspect CSP and generated package contents | No vet issues or unformatted Go files |
| Packaging | Required: `wails build -clean -platform darwin/arm64`, bundle/architecture/ad-hoc-signature checks; conditional public profile: Developer ID, `xcrun notarytool`, stapling | Required: personal-use `.app` single-launch and P1 smoke; conditional: DMG and Gatekeeper smoke | Personal-use app contains all assets and launches on the owner's Mac; a public candidate, when selected, launches without bypass |
| Acceptance evidence | Compare the canonical candidate commit and SHA-256 across quickstart, rollback guidance, and release handoff | Verify the named artifact is the final post-Electron candidate | Exactly one accepted artifact identity and digest are recorded |
| Legacy regression | Existing `npm start` before cutover | Baseline session and player journey | Rollback path stays usable until final deletion |

## Complexity Tracking

No unresolved constitution deviations remain. Constitution v2.0.0 explicitly governs the parallel migration, Go/Wails boundaries, macOS-first packaging, and final Electron cutover.

BUG-002 adds an ordering constraint between the final post-Electron build and
rollback-document synchronization. BUG-003 adds a development-supervisor
process-ownership edge case that requires real-process verification in addition
to service-level fakes. BUG-004 adds a low-risk frontend compatibility rename;
the method/event contract remains unchanged while the transitional global name
is removed.
