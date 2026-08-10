# Fallout Terminal Constitution

## Project Identity

Fallout Terminal is a desktop application for tabletop RPG game masters. The
desktop master interface edits and publishes Fallout-style terminal content,
while an embedded HTTP and WebSocket server synchronizes that content with
browser-based player clients. Saved campaign state is stored as versioned JSON
session files; live terminal, navigation, and hacking state is process-local.

The repository is a modular monolith. During the approved Wails v2 migration,
the existing Electron/Node application remains the behavioral oracle and
rollback release while equivalent Go/Wails components are added beside it. The
legacy runtime MUST be removed only after the migration acceptance gates pass.

## Core Principles

### I. Preserve Runtime Boundaries

- The desktop composition root owns application lifecycle, native dialogs,
  filesystem-backed persistence, player-server startup, and optional tunnel
  startup. It is `main.js` during the Electron baseline and `main.go`/`app.go`
  in the Wails candidate.
- The master browser frontend MUST access privileged operations only through a
  narrow registered bridge. It MUST NOT gain direct Node, Go filesystem,
  process, or environment access.
- `master/` owns the Electron game-master frontend during transition;
  `frontend/` owns the Wails game-master frontend.
- `server/` owns the Electron HTTP/WebSocket implementation during transition;
  Go packages under `internal/` own the Wails player server and domain services.
- Navigation and hacking logic MUST remain transport-independent and
  server-authoritative.
- `client/` owns the browser player experience and MUST operate without desktop
  runtime APIs.
- `sessions/` contains versioned JSON examples/data, not executable logic.

Changes crossing these boundaries MUST identify every affected surface in the
feature specification and implementation plan.

### II. Keep Shared State Server-Authoritative

Player navigation and hacking actions are requests, not local state changes.
The server MUST validate actions, mutate canonical live state, and broadcast the
resulting state to all connected clients. Clients MUST converge on server
messages such as `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `NAV_STATE`, and
`HACK_STATE`; divergent optimistic client-only transitions are prohibited.

Protocol changes MUST document direction, payload shape, validation,
reconnection behavior, and compatibility impact. Secret hacking data MUST stay
server-side and MUST NOT be included in public state payloads.

### III. Protect Desktop and Public-Access Boundaries

The Electron baseline MUST retain `nodeIntegration: false`,
`contextIsolation: true`, and `sandbox: true`. The Wails candidate MUST register
only the Go methods needed by the master frontend. Both bridges MUST validate
untrusted inputs again at the privileged boundary. Content Security Policy MUST
remain restrictive, and external URL handling MUST allowlist HTTP and HTTPS.

Public ngrok mode MUST fail closed without valid credentials. Credentials and
generated traffic-policy files MUST NOT be committed, stored in session data,
or disclosed through UI/log output. Temporary policy material MUST be private
and cleaned up on success, failure, and shutdown paths.

### IV. Preserve Session Data Compatibility

Session files are user-owned persistent data. Specifications that change the
session shape MUST define versioning, validation, defaults, and migration or
backward-compatibility behavior before implementation. Saving MUST remain
explicit about the target file and MUST NOT silently overwrite, relocate, or
transform unrelated user data.

On macOS, user-created sessions SHOULD default to
`~/Documents/Fallout Terminal/Sessions/` after explicit confirmation. Bundled
samples inside the read-only application bundle MUST be copied only after an
explicit user action. App-managed metadata belongs in
`~/Library/Application Support/com.vaulttec.fallout-terminal/`. Autosave MUST
continue targeting the explicitly selected session path.

Runtime-only live, navigation, connection, hacking, and application metadata
MUST NOT be added to session JSON unless a feature explicitly changes the
persistence contract.

### V. Match Established Code Conventions

Browser JavaScript uses lowercase filenames, camelCase variables/functions,
`UPPER_SNAKE_CASE` constants, two-space indentation, semicolons, and primarily
single-quoted strings. CSS classes use kebab-case. JSON properties use camelCase.
WebSocket message types use uppercase snake case.

Go code follows standard package naming, `gofmt`, exported-name documentation,
and small interfaces at integration boundaries. Renderer files remain browser
code. The migration MAY introduce a root Go module and a frontend-scoped npm
package; it MUST NOT retain Node as an application runtime after cutover.

There is no repository-wide Conventional Commits requirement. Migration work
uses a dedicated feature branch based on `develop` and targets `develop` for
integration.

## Dependency Rules

- Root `main.go`/`app.go` MAY depend on Wails and internal application services.
- `internal/domain`, `internal/nav`, and `internal/hack` MUST remain independent
  of Wails, HTTP, WebSocket, and frontend code.
- `internal/player` MAY depend on HTTP/WebSocket libraries and domain services,
  but MUST NOT depend on the master frontend.
- `frontend/` MAY call only generated Wails bindings and runtime events exposed
  by the registered bridge.
- `client/` MAY use browser APIs, player HTTP endpoints, static assets, and the
  WebSocket protocol; it MUST NOT depend on Electron, Wails, or filesystem APIs.
- During transition, legacy Electron dependency rules remain applicable to
  `main.js`, `preload.js`, `master/`, and `server/` until those paths are removed.
- Every new runtime dependency MUST have a concrete need recorded in the plan
  and be pinned reproducibly.

## Testing and Quality Gates

The Go/Wails candidate MUST use colocated Go tests and deterministic fakes at
filesystem, dialog, process, clock, network, and event boundaries. Concurrency
code MUST pass the race detector. Browser, native-dialog, audio, public-tunnel,
and packaged-application checks MAY remain documented manual gates where robust
automation is not yet present; unavailable checks MUST be reported, not claimed.

Applicable migration commands MUST succeed before cutover:

- `gofmt -l .` produces no Go source paths.
- `go vet ./...` succeeds.
- `go test ./...` and `go test -race ./...` succeed.
- `wails dev` passes the affected interactive journeys.
- A clean macOS Apple Silicon `wails build` produces a self-contained `.app`.
- Release candidates pass signing, hardened-runtime, notarization, stapling, and
  DMG smoke checks when release credentials are available.

Until cutover, `npm start` remains the Electron behavioral smoke check and
rollback proof. Reviews MUST additionally verify module boundaries, session
compatibility, server/client synchronization, privileged-interface exposure,
public-access security, macOS storage behavior, and owned-resource shutdown.

## Development Workflow

1. Branch from `develop` into a dedicated feature branch.
2. Specify user-visible behavior and independently testable scenarios.
3. Identify affected runtime surfaces and freeze compatibility contracts.
4. Plan persistence, synchronization, security, packaging, and rollback impacts.
5. Implement the smallest coherent vertical slice while retaining the Electron
   behavioral oracle.
6. Run the verification defined in the plan and record unavailable checks.
7. Remove the old runtime only after all migration and package gates pass.
8. Update README and compatibility specifications when setup, operation, or
   governed paths change.

## Governance

This constitution governs Spec Kit artifacts and feature work in this repository.
Amendments MUST reflect an intentional project decision, document their rationale,
and update the version below. Existing behavior is evidence, but an accidental
inconsistency does not automatically become a new standard.

Constitution checks are required during planning and after design. Any violation
MUST be listed in the plan's Complexity Tracking table with a concrete rationale
and a rejected simpler alternative.

**Version**: 2.0.0 | **Ratified**: 2026-08-09 | **Last Amended**: 2026-08-09
