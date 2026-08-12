# Fallout Terminal Constitution

## Project Identity

Fallout Terminal is a desktop application for tabletop RPG game masters. The
native master interface edits and publishes Fallout-style terminal content,
while an embedded HTTP and WebSocket server synchronizes that content with
browser-based player clients. Saved campaign state is stored as versioned JSON
session files; live terminal, navigation, hacking, and connection state is
process-local.

The repository is a Go 1.26 modular monolith built with Wails v2. The root Go
module owns the desktop runtime and embedded player server. `frontend/` is a
Vite-built, browser-JavaScript master interface; `client/` is a separate
browser-JavaScript player interface embedded into the same application binary.
Node.js is build and browser-test tooling, not an application runtime. The
supported deployment profile is macOS 13+ on Apple Silicon (`arm64`).

## Core Principles

### I. Preserve Runtime Boundaries

- Root `main.go` and `app.go` own application composition, lifecycle, native
  dialogs, filesystem-backed persistence, player-server startup, Wails events,
  and optional tunnel startup.
- The master frontend in `frontend/src/` MUST access privileged operations only
  through the narrow generated Wails bindings and runtime events exposed by the
  registered Go bridge. It MUST NOT gain direct filesystem, process, or
  environment access.
- `internal/domain/` owns persistent and public models, JSON codecs, cloning,
  and validation. `internal/nav/` and `internal/hack/` own transport-independent
  navigation and hacking rules.
- `internal/live/` owns canonical live terminal state. `internal/control/` owns
  logical player sessions, roster assignments, controller authority, broadcast
  lifetime, and ordered coordination effects.
- `internal/session/` and `internal/playerconfig/` own durable JSON storage and
  native selection workflows. `internal/player/` owns the player HTTP,
  WebSocket, and protocol boundary.
- `internal/platform/` owns desktop and platform paths. `internal/tunnel/` owns
  optional public-access process and credential-material lifecycles.
- `client/` owns the browser player experience and MUST operate without Wails,
  native desktop, or filesystem APIs.
- `sessions/` contains versioned JSON examples/data, not executable logic.

Changes crossing these boundaries MUST identify every affected producer,
consumer, state owner, and verification surface in the feature specification
and implementation plan.

### II. Keep Shared State Server-Authoritative

Player navigation, character selection, controller actions, and hacking actions
are requests, not local state changes. The Go services MUST validate actions,
mutate canonical process state, and publish detached public projections. Player
clients MUST converge on server messages such as `TERMINAL_LIVE`,
`TERMINAL_UPDATE`, `NAV_STATE`, `HACK_STATE`, and `PLAYER_STATE`; divergent
optimistic client-only transitions are prohibited.

Protocol changes MUST document direction, payload shape, validation, rejection
behavior, ordering or revision semantics, reconnection behavior, and
compatibility impact. Secret hacking data and trusted coordination state MUST
stay server-side and MUST NOT be included in public payloads.

### III. Protect Desktop and Public-Access Boundaries

The Wails application MUST register only the Go methods needed by the master
frontend. Browser-controlled payloads, file references, runtime commands, and
external URLs MUST be validated again at the privileged Go boundary. Content
Security Policy MUST remain restrictive, and external URL handling MUST
allowlist HTTP and HTTPS.

The player WebSocket endpoint MUST enforce its same-host origin policy and
bounded input handling. Public ngrok mode MUST fail closed without valid
credentials. Credentials and generated traffic-policy files MUST NOT be
committed, stored in session data, or disclosed through UI or log output.
Temporary policy material and owned processes MUST be private and cleaned up on
success, failure, and shutdown paths.

### IV. Preserve Session Data Compatibility

Session and player-configuration files are user-owned persistent data.
Specifications that change either JSON shape MUST define versioning,
validation, defaults, references, and migration or backward-compatibility
behavior before implementation. Saving MUST remain explicit about the target
file and MUST NOT silently overwrite, relocate, or transform unrelated user
data.

On macOS, user-created sessions SHOULD default to
`~/Documents/Fallout Terminal/Sessions/` after explicit confirmation. Bundled
samples inside the read-only application bundle MUST be copied only after an
explicit user action. App-managed metadata belongs in
`~/Library/Application Support/com.vaulttec.fallout-terminal/`. Autosave MUST
continue targeting the explicitly selected session path.

Runtime-only live, navigation, connection, hacking, coordination, and
application metadata MUST NOT be added to persistent JSON unless a feature
explicitly changes the persistence contract.

### V. Match Established Code Conventions

Browser JavaScript uses lowercase filenames, camelCase variables/functions,
`UPPER_SNAKE_CASE` constants, two-space indentation, semicolons, and primarily
single-quoted strings. CSS classes use kebab-case. JSON properties use camelCase.
WebSocket message types use uppercase snake case.

Go code uses lowercase package names, snake_case multiword filenames, `gofmt`,
exported-name documentation, and small interfaces at integration boundaries.
Go tests remain colocated as `*_test.go`; browser journeys remain in
`tests/browser/*.spec.mjs`. Generated Wails bindings MUST NOT become the source
of domain behavior.

There is no repository-wide Conventional Commits requirement. Feature work uses
a dedicated branch based on `develop` and targets `develop` for integration.

## Dependency Rules

- Root `main.go` and `app.go` MAY depend on Wails and all internal application
  services because they are the composition and privileged bridge boundary.
- `internal/domain/` MUST remain independent of Wails, HTTP, WebSocket, and
  frontend code.
- `internal/nav/` and `internal/hack/` MAY depend on `internal/domain/` but MUST
  remain transport-independent.
- `internal/live/` MAY depend on `internal/domain/`, `internal/nav/`, and
  `internal/hack/` but MUST NOT depend on Wails or frontend code.
- `internal/session/` MAY depend on `internal/domain/`.
  `internal/playerconfig/` MAY depend on `internal/domain/` and reuse the
  session filesystem boundary.
- `internal/control/` MAY depend on `internal/domain/` and narrow service
  interfaces but MUST NOT depend on HTTP, WebSocket, Wails, or either frontend.
- `internal/player/` MAY depend on HTTP/WebSocket libraries and the control,
  domain, and live services, but MUST NOT depend on the master frontend.
- `internal/platform/` contains Wails/platform adapters. `internal/tunnel/`
  contains optional public-process integration; neither package owns domain
  rules.
- `frontend/` MAY call only generated Wails bindings and runtime events exposed
  by the registered bridge.
- `client/` MAY use browser APIs, player HTTP endpoints, static assets, and the
  WebSocket protocol; it MUST NOT depend on Wails or filesystem APIs.
- Every new runtime or build dependency MUST have a concrete need recorded in
  the plan and be pinned reproducibly in `go.mod` or the appropriate npm lockfile.

## Testing and Quality Gates

Go code MUST use colocated tests and deterministic fakes at filesystem, dialog,
process, random, network, and event boundaries. Concurrency-sensitive code MUST
pass the race detector. Browser journeys use Playwright under `tests/browser/`.
No numeric coverage threshold or repository-wide linter is currently defined;
plans MUST choose verification proportionate to the affected behavior instead
of inventing or claiming either gate.

Applicable commands MUST succeed before a change is considered complete:

- `gofmt -l .` produces no Go source paths.
- `go vet ./...` succeeds.
- `go test ./...` succeeds.
- `go test -race ./...` succeeds for changes affecting concurrent runtime,
  player, live, control, session, or tunnel behavior.
- `npm ci --prefix frontend` and `npm run build --prefix frontend` succeed for
  frontend, bridge, embedding, or packaging changes.
- `npm ci --prefix tests/browser` and `npm test --prefix tests/browser` succeed
  for affected player-browser journeys when the required local environment is
  available.
- `wails dev` passes affected interactive master/player journeys.
- A clean `wails build -clean -platform darwin/arm64` produces a self-contained
  application for packaging-sensitive changes.
- Release candidates pass signing, hardened-runtime, notarization, stapling,
  DMG, and Gatekeeper checks when release credentials are available.

The GitHub Actions workflow MUST continue to enforce its configured Go test,
Go vet, frontend clean-build, startup-contract, and unsigned arm64 packaging
gates. Native-dialog, audio, public-tunnel, multi-browser, and signed-release
checks MAY remain documented manual gates where reliable automation or required
credentials are unavailable; unavailable checks MUST be reported, not claimed.

Reviews MUST additionally verify module boundaries, persistent-data
compatibility, server/client synchronization, privileged-interface exposure,
public-access security, macOS storage behavior, and owned-resource shutdown.

## Development Workflow

1. Branch from `develop` into a dedicated feature branch.
2. Specify user-visible behavior and independently testable scenarios.
3. Identify every affected runtime surface and freeze compatibility contracts.
4. Plan persistence, synchronization, security, packaging, and rollback impacts
   that actually apply to the feature.
5. Implement the smallest coherent vertical slice through the necessary Go and
   browser boundaries.
6. Run the automated and interactive verification defined in the plan and
   record unavailable checks.
7. Update README, protocol contracts, fixtures, and compatibility specifications
   when setup, operation, or governed behavior changes.

## Governance

This constitution governs Spec Kit artifacts and feature work in this repository.
Amendments MUST reflect an intentional project decision, document their
rationale, and update the version below. Existing behavior is evidence, but an
accidental inconsistency does not automatically become a new standard.

Constitution checks are required during planning and after design. Any violation
MUST be listed in the plan's Complexity Tracking table with a concrete rationale
and a rejected simpler alternative.

**Version**: 2.1.0 | **Ratified**: 2026-08-09 | **Last Amended**: 2026-08-13
