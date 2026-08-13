# Tasks: Wails v2 Runtime Migration

> **SUPERSEDED LEGACY PLAYER TRANSPORT — HISTORICAL, NON-AUTHORITATIVE.**
> Any WebSocket or handwritten JSON player-transport description in this retained
> completed feature document has been replaced by the generated ConnectRPC contract in
> [`specs/005-connectrpc-protobuf-migration/contracts/public-player.md`](../005-connectrpc-protobuf-migration/contracts/public-player.md).

**Input**: Design documents from `/specs/001-wails-v2-migration/`

**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, and `quickstart.md`

**Testing**: Automated tests are required by FR-022. Use Go `testing`, `httptest`, table-driven fixtures, deterministic boundary fakes, and `go test -race ./...`. Keep manual Wails/WebKit, real-browser, audio, ngrok, macOS storage, personal-use `.app`, and conditional public signing/notarization/DMG checks in `quickstart.md`.

**Organization**: Tasks are grouped by prioritized user story and explicit dependency waves. The Electron path remains runnable through US4 and is removed only after all parity gates pass.

**Bugfix**: 2026-08-09 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-08-09 — BUG-002 Updated from bugfix patch

**Bugfix**: 2026-08-09 — BUG-003 Updated from bugfix patch

**Bugfix**: 2026-08-09 — BUG-004 Updated from bugfix patch

**Scope decision**: 2026-08-09 — Personal-use `.app` acceptance is the active cutover gate; public Developer ID/notarization work is conditional and does not block migration completion.

## Phase 1: Setup and governed migration baseline

**Purpose**: Establish the approved runtime structure and reproducible tooling without disturbing the working Electron release.

**Wave 1 — completed governance prerequisite:**

- [x] **T001** Apply the user-approved Electron-to-Wails transition, macOS-first packaging, and Go quality-gate amendment · `.specify/memory/constitution.md`

**⟶ Wait for Wave 1 to finish, then Wave 2 — independent (different files):**

- [x] **T002** [P] Initialize the Go 1.26 module and pin Wails v2.13.0 plus coder/websocket v1.8.15 · `go.mod`, `go.sum`
- [x] **T003** [P] Create the vanilla Vite frontend package and lock-file definition · `frontend/package.json`, `frontend/package-lock.json`, `frontend/vite.config.js`
- [x] **T004** [P] Add the Wails app icon, macOS application metadata, and hardened-runtime entitlements · `build/appicon.png`, `build/darwin/Info.plist`, `build/darwin/entitlements.plist`
- [x] **T005** [P] Extend ignores for Go/Wails binaries, frontend build output, generated bindings, and dependency directories while retaining source assets · `.gitignore`
- [x] **T006** [P] Add version-1 session and protocol golden fixtures derived from the Electron oracle · `internal/testutil/testdata/session-v1.json`, `internal/testutil/testdata/protocol/*.json`
- [x] **T007** [P] Add reusable fake clock, filesystem, dialog, browser, process, player-server, tunnel, and event sinks · `internal/testutil/fakes.go`
- [x] **T008** [P] Run the untouched Electron baseline journey and record the behavioral oracle · `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T009** Create the Wails composition root and configure `wails.json` so root `wails dev` owns frontend install/build/watch and Go launch without deleting Electron · `main.go`, `app.go`, `wails.json`

**Checkpoint**: Pinned tooling and fixtures are ready, root `wails dev` has a single configured orchestration boundary, and Electron remains runnable.

---

## Phase 2: Foundational Go domain and lifecycle work

**Purpose**: Build tested models, authoritative state, and lifecycle infrastructure that block every user story.

**Wave 1:**

- [x] **T010** Add failing version-1 model, unknown-field round-trip, size/depth, duplicate-ID, and invalid-root tests · `internal/domain/model_test.go`, `internal/domain/validate_test.go`

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T011** Implement Session, Terminal, recursive ContentNode, NavState, private HackState, and public projection types · `internal/domain/model.go`, `internal/domain/json.go`

**⟶ Wait for T011 to finish, then:**

- [x] **T012** Implement complete version-1 validation and configured size/depth bounds · `internal/domain/validate.go`

**⟶ Wait for T012 to finish, then Wave 4 — independent (different files):**

- [x] **T013** [P] Add failing table-driven shared-navigation and revalidation tests · `internal/nav/nav_test.go`
- [x] **T014** [P] Add failing hacking generation, guess, administrator, force-success, and secret-filter tests · `internal/hack/hack_test.go`

**⟶ Wait for Wave 4 to finish, then Wave 5 — independent (different files):**

- [x] **T015** [P] Port shared-navigation behavior and direct-child revalidation · `internal/nav/nav.go`
- [x] **T016** [P] Port word-bank data and hacking behavior with private/public state separation · `internal/hack/wordbank.go`, `internal/hack/hack.go`

**⟶ Wait for Wave 5 to finish, then:**

- [x] **T017** Add failing concurrent live-lifecycle, immutable snapshot, and secret-redaction tests · `internal/live/service_test.go`

**⟶ Wait for T017 to finish, then:**

- [x] **T018** Implement mutex-protected set/update/clear/action transitions and immutable public snapshots · `internal/live/service.go`

**⟶ Wait for T018 to finish, then:**

- [x] **T019** Add failing application tests for player-listener-before-ready ordering, partial-start unwinding, reverse shutdown, and idempotency · `app_test.go`

**⟶ Wait for T019 to finish, then:**

- [x] **T020** Define injectable SessionService, PlayerServer, TunnelService, DesktopRuntime, and EventSink interfaces and implement the shared development/packaged lifecycle · `app.go`

**Checkpoint**: `go test -race ./internal/domain ./internal/nav ./internal/hack ./internal/live` passes, private puzzle state cannot cross a boundary, and the composition root has testable startup/shutdown ownership.

---

## Phase 3: User Story 1 — Run the game-master workspace without behavior loss (Priority: P1) 🎯 MVP

**Goal**: Launch the Wails master workspace with one root command and preserve session authoring, save ordering, native dialogs, live controls, status events, and privileged-boundary validation.

**Independent Test**: From a prepared checkout run only `wails dev`; within five seconds obtain the workspace and player address, then create/open/edit/save/reopen a version-1 session and exercise live controls without Electron or separate frontend/server commands.

### Tests

**Wave 1 — independent (different files):**

- [x] **T021** [P] [US1] Add failing macOS storage, demo-copy, cancellation, invalid-file, active-path, atomic-write, and 20-revision ordered-save tests · `internal/session/service_test.go`, `internal/session/storage_test.go`
- [x] **T022** [P] [US1] Add failing bridge validation, RuntimeStatus snapshot, event, URL allowlist, and lifecycle-cleanup tests with fakes · `app_test.go`
- [x] **T023** [P] [US1] Add a failing startup-contract test that validates the required `wails.json` frontend install/build/watcher keys and single root entry · `internal/platform/startup_test.go`

### Implementation

**⟶ Wait for Test Wave 1 to finish, then Wave 2 — independent (different files):**

- [x] **T024** [P] [US1] Implement Documents defaults, read-only bundled demo, explicit copy, Application Support isolation, and atomic storage · `internal/platform/paths.go`, `internal/session/storage.go`
- [x] **T025** [P] [US1] Implement Wails native dialog and HTTP(S)-only external-browser wrappers · `internal/platform/desktop.go`
- [x] **T026** [P] [US1] Migrate the master HTML/CSS/JavaScript without redesign and retain restrictive CSP · `frontend/src/index.html`, `frontend/src/master.css`, `frontend/src/master.js`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T027** [US1] Implement validated create/open and revision-ordered same-directory atomic saves · `internal/session/service.go`

**⟶ Wait for T027 to finish, then:**

- [x] **T028** [US1] Implement `GetRuntimeStatus`, session commands, live-control commands, structured results, and privileged input validation · `app.go`

**⟶ Wait for T028 to finish, then Wave 5 — independent (different files):**

- [x] **T029** [P] [US1] Implement the `window.electronAPI` compatibility facade with generated bindings and unsubscribe-safe Wails events · `frontend/src/desktop-api.js`
- [x] **T030** [P] [US1] Ignore stale save completions and show only the newest durable revision · `frontend/src/master.js`

**⟶ Wait for Wave 5 to finish, then:**

- [x] **T031** [US1] Wire Wails startup, DOM-ready status, window constraints/theme, embedded master assets, and ordered shutdown through the shared lifecycle · `main.go`, `app.go`

**⟶ Wait for T031 to finish, then:**

- [x] **T032** [US1] Run the one-command startup and game-master/session journeys and record command count, readiness time, player URL, and observed results · `specs/001-wails-v2-migration/quickstart.md`

**Checkpoint**: US1 is independently functional; one root `wails dev` invocation reaches a usable workspace/player address, Electron remains rollback, and the master has no general filesystem/process access.

---

## Phase 4: User Story 2 — Preserve synchronized player gameplay (Priority: P1)

**Goal**: Serve the retained player application from the in-process Go server and preserve authoritative navigation, hacking, reconnect, optional audio, and client-count behavior.

**Independent Test**: Connect four, five, six, and seven clients, alternate at least 25 navigation/hacking actions, reconnect one client, and verify identical sanitized state after every accepted action.

### Tests

**Wave 1 — independent (different files):**

- [x] **T033** [P] [US2] Add failing static-route, traversal, security-header, sound allowlist, and missing-asset tests · `internal/player/http_test.go`
- [x] **T034** [P] [US2] Add failing malformed-message, size-limit, unknown-type, target-validation, and secret-filter contract tests · `internal/player/protocol_test.go`
- [x] **T035** [P] [US2] Add failing 4–7-client convergence, slow-client isolation, reconnect, clear-state, and graceful-shutdown tests · `internal/player/server_test.go`

### Implementation

**⟶ Wait for Test Wave 1 to finish, then Wave 2 — independent (different files):**

- [x] **T036** [P] [US2] Implement strict JSON decoding and typed envelopes with the exact protocol identifiers · `internal/player/protocol.go`
- [x] **T037** [P] [US2] Implement embedded static assets, security headers, index fallback, and allowlisted sound discovery · `internal/player/http.go`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T038** [US2] Implement PlayerConnection reader loop, bounded queue, single writer, origin policy, and idempotent close · `internal/player/client.go`

**⟶ Wait for T038 to finish, then:**

- [x] **T039** [US2] Implement listener acquisition, connection registry, live dispatch, snapshots, client-count callbacks, and graceful shutdown · `internal/player/server.go`

**⟶ Wait for T039 to finish, then Wave 5 — independent (different files):**

- [x] **T040** [P] [US2] ⚠️ Reopened — Preserve player CSP, same-origin reconnect, presentation, and optional-audio degradation; correct state visibility so `hidden` containers with class-level `display` rules occupy no layout space (reopened — BUG-001) · `client/index.html`, `client/client.css`, `client/client.js`, `client/sound.js`
- [x] **T041** [P] [US2] Embed and pass the retained player filesystem separately from Wails master assets · `main.go`
- [x] **T042** [P] [US2] Wire player-server status, client-count events, public hack events, and reverse-order shutdown · `app.go`

**⟶ Wait for Wave 5 to finish, then:**

- [x] **T043** [US2] Run race-enabled live/player tests and fix every reported race, leak, or unsafe snapshot · `internal/live/`, `internal/player/`

**⟶ Wait for T043 to finish, then:**

- [x] **T044** [US2] ⚠️ Reopened — Complete the four-through-seven-browser, reconnect, presentation, and audio journeys; verify idle, normal, hacking, and blocked exclusivity at both ends of an overflowing menu and record parity evidence (reopened — BUG-001) · `specs/001-wails-v2-migration/quickstart.md`

**BUG-001 regression wave — run before the reopened implementation task:**

- [x] **T072** [US2] Add a failing player-state visibility contract test that requires hidden idle, hacking, and blocked containers to be layout-excluded despite their class-level `display` rules and covers overflowing menu markup · `internal/platform/assets_test.go`, `client/index.html`, `client/client.css`

**⟶ Wait for T072 to finish, then complete reopened T040; wait for T040 to finish, then complete reopened T044.**

**Checkpoint**: Both P1 stories work on Wails; all clients converge, reconnect does not reset puzzles, and private hacking fields remain server-side.

---

## Phase 5: User Story 3 — Operate protected local and public access (Priority: P2)

**Goal**: Preserve opt-in authenticated ngrok access, non-secret status, private temporary policy handling, and deterministic child cleanup without adding another startup command.

**Independent Test**: Exercise disabled, invalid-credential, missing-binary, early-exit, timeout, success, and shutdown paths with fakes, then verify authenticated HTTP/WSS through the same single application invocation.

### Tests

**Wave 1 — independent (different files):**

- [x] **T045** [P] [US3] Add failing credential precedence, policy escaping/permissions, HTTPS URL parsing, timeout, diagnostic, and cleanup tests · `internal/tunnel/service_test.go`
- [x] **T046** [P] [US3] Add failing process start, graceful stop, forced stop, early-exit, and idempotent-shutdown tests · `internal/tunnel/process_test.go`
- [x] **T047** [P] [US3] Add failing application tests proving invalid public configuration starts zero processes and preserves local readiness · `app_test.go`

### Implementation

**⟶ Wait for Test Wave 1 to finish, then Wave 2 — independent (different files):**

- [x] **T048** [P] [US3] Implement environment/argument configuration, credential validation, private policy files, and cleanup registration · `internal/tunnel/config.go`, `internal/tunnel/policy.go`
- [x] **T049** [P] [US3] Implement Darwin process-group ownership/termination and portable process defaults · `internal/tunnel/process_darwin.go`, `internal/tunnel/process_other.go`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T050** [US3] Implement bounded output parsing, HTTPS discovery, timeout, process ownership, graceful termination, and escalation · `internal/tunnel/service.go`, `internal/tunnel/process.go`

**⟶ Wait for T050 to finish, then Wave 4 — independent (different files):**

- [x] **T051** [P] [US3] Start the tunnel only after local readiness, emit public/local/error ServerInfo, and stop it before the listener · `app.go`
- [x] **T052** [P] [US3] Preserve local/public/error presentation and safe link opening through the compatibility facade · `frontend/src/master.js`, `frontend/src/desktop-api.js`

**⟶ Wait for Wave 4 to finish, then:**

- [x] **T053** [US3] Run local, invalid, and authenticated public-access journeys through one application invocation and record redacted evidence · `specs/001-wails-v2-migration/quickstart.md`

**Checkpoint**: Local mode starts no tunnel, invalid configuration fails closed, valid public mode protects HTTP/WSS, and shutdown leaves no owned process or policy directory.

---

## Phase 6: User Story 4 — Install, validate, and roll back the personal-use application (Priority: P2)

**Goal**: Produce a repeatable personal-use macOS Apple Silicon `.app` whose single launch owns the complete system, with clean-checkout gates and intact Electron rollback until personal-use acceptance. Preserve signed/notarized DMG automation as a conditional future public profile.

**Independent Test**: Build on clean macOS Apple Silicon, launch the local/ad-hoc `.app` once without developer tooling, repeat the P1 journeys, validate integrity/storage/cleanup, and demonstrate rollback remains usable until cutover. Run the DMG/Gatekeeper branch only when the public profile and Developer ID credentials are available.

### Tests and packaging

**Wave 1 — independent (different files):**

- [x] **T054** [P] [US4] Add an asset-manifest test covering master/player files, fonts, every sound category, and demo session · `internal/platform/assets_test.go`
- [x] **T055** [P] [US4] Add clean-checkout Go test, vet, frontend build, startup-config test, and unsigned Apple Silicon app jobs · `.github/workflows/wails-macos.yml`
- [x] **T056** [P] [US4] Document prerequisites, sole development command, local/public operation, storage, builds, signing, recovery, and deferred platforms · `README.md`
- [x] **T057** [P] [US4] Document acceptance ownership and Electron rollback procedure · `docs/wails-migration-rollback.md`

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T058** [US4] Configure product identity, bundle metadata, icons, entitlements, embedded assets, and production startup entry · `wails.json`, `main.go`, `build/darwin/`

**⟶ Wait for T058 to finish, then:**

- [x] **T059** [US4] Implement reproducible arm64 personal-use app build plus optional fail-closed Developer ID signing, hardened-runtime verification, DMG, notarization, stapling, and Gatekeeper commands without secret output · `scripts/build-macos.sh`, `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T059 to finish, then:**

- [x] **T060** [US4] Build and accept the personal-use `.app`, inventory assets, and record architecture, hashes, ad-hoc signature, and bundle integrity; record optional public signing/notarization/DMG evidence as not applicable when credentials are unavailable · `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T060 to finish, then:**

- [x] **T061** [US4] Launch the personal-use `.app` once without Go, Node, npm, Vite, or Wails and verify workspace/player readiness with no separately invoked frontend or server; repeat for a public candidate only when produced · `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T061 to finish, then:**

- [x] **T062** [US4] Validate personal-use session locations, demo copy, autosave path, port-conflict errors, local launch/manual-approval expectations, and shutdown cleanup; repeat Gatekeeper checks for a public candidate only when produced · `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T062 to finish, then:**

- [x] **T063** [US4] Compare version-1 round trips and protocol goldens between Electron and Wails and record zero intentional differences · `internal/testutil/testdata/`, `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T063 to finish, then Wave 2 — independent (different files):**

- [x] **T073** [P] [US4] Update the user guide so the personal-use `.app` is the active acceptance profile and public Developer ID signing is conditional · `README.md`
- [x] **T074** [P] [US4] ⚠️ Reopened — Synchronize acceptance ownership and rollback guidance with the canonical final post-Electron candidate commit and executable SHA-256 while public publication remains separately gated (reopened — BUG-002) · `docs/wails-migration-rollback.md`, `specs/001-wails-v2-migration/quickstart.md`

**Historical cutover ordering (already satisfied by the original T073/T074 completions):** ~~Wait for T073 and T074 to finish, then~~ T064 ran only after those original documentation gates; reopened T074 is now a post-cutover BUG-002 refresh and does not gate or rerun T064.

- [x] **T064** [US4] Remove Electron orchestration, preload, server, root npm files, and build configuration only after T073/T074 and personal-use package acceptance; do not wait for optional public credentials · `main.js`, `preload.js`, `server/`, `package.json`, `package-lock.json`

**⟶ Wait for T064 to finish, then:**

- [x] **T065** [US4] Remove unreferenced master duplicates, tighten ignores, and verify the final source/asset manifest · `master/`, `.gitignore`, `main.go`, `frontend/`

**Checkpoint**: The Wails candidate is accepted for personal use, one app launch starts the complete system, session data needs no rollback transform, and legacy deletion occurs only after personal-use package acceptance. Public distribution remains prohibited until its conditional trust gates pass.

---

## Bugfix Remediation Phase: Final migration cleanup

**Wave 1 — independent failing checks and evidence selection:**

- [x] **T075** [P] [US4] Establish the canonical final post-Electron candidate from the current cutover source, record its commit and executable SHA-256 in one acceptance record, and add a consistency check that rejects conflicting accepted-artifact digests (BUG-002) · `specs/001-wails-v2-migration/quickstart.md`, `docs/wails-migration-rollback.md`
- [x] **T076** [P] [US3] Add a failing real-process lifecycle harness that starts public mode through the documented development supervisor, sends a handled interrupt, and asserts zero owned ngrok processes, port-3690 listeners, and credential-policy directories after the shutdown timeout (BUG-003) · `internal/tunnel/`, `internal/platform/`, `specs/001-wails-v2-migration/contracts/startup.md`, `specs/001-wails-v2-migration/quickstart.md`
- [x] **T079** [P] [US1] Add a failing frontend/static contract test that requires the active privileged facade to be `window.desktopAPI` and rejects `window.electronAPI` definitions or consumers outside historical migration and rollback documentation (BUG-004) · `internal/platform/assets_test.go`, `frontend/src/desktop-api.js`, `frontend/src/master.js`, `specs/001-wails-v2-migration/contracts/desktop-bridge.md`

**⟶ Wait for T075 to finish, then complete reopened documentation:**

- Reopened **T074** synchronizes the canonical T075 identity and digest across rollback guidance and acceptance evidence.

**⟶ Wait for T076 to finish, then:**

- [x] **T077** [US3] Implement handled development-supervisor interruption ownership so an active tunnel, player listener, and temporary policy material are removed without relying solely on the native window shutdown callback; update the startup contract with the bounded handled-interrupt guarantee (BUG-003) · `main.go`, `app.go`, `internal/tunnel/`, `internal/platform/`, `specs/001-wails-v2-migration/contracts/startup.md`

**⟶ Wait for T077 to finish, then:**

- [x] **T078** [US3] Run the authenticated public-mode development journey, interrupt the sole supervisor command through the handled path, and record redacted zero-resource cleanup evidence (BUG-003) · `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T079 to finish, then:**

- [x] **T080** [US1] Rename the active compatibility facade and every consumer from `window.electronAPI` to `window.desktopAPI` without changing command, event, validation, or unsubscribe behavior; update the desktop-bridge contract and verify Electron bridge references remain only in historical documentation (BUG-004) · `frontend/src/desktop-api.js`, `frontend/src/master.js`, `internal/platform/assets_test.go`, `specs/001-wails-v2-migration/contracts/desktop-bridge.md`

**Checkpoint**: T074, T075, T078, and T080 are complete; acceptance evidence names one final artifact, handled public-mode development interruption leaves zero owned resources, and active frontend source uses only the runtime-neutral bridge name.

---

## Final Phase: Polish, validation, and handoff

**Wave 1:**

- [x] **T066** Review privileged exposure, CSP, URL parsing, WebSocket origin/input checks, and secret filtering · `app.go`, `frontend/src/index.html`, `internal/player/`, `internal/live/`

**⟶ Wait for T066 to finish, then:**

- [x] **T067** Review temporary-file permissions, atomic replacement, credential cleanup, process escalation, listener shutdown, and goroutine ownership · `internal/session/`, `internal/tunnel/`, `internal/player/`

**⟶ Wait for T067 to finish, then:**

- [x] **T068** Run `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...` from a clean checkout and record exact results · `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T068 to finish, then:**

- [x] **T069** ⚠️ Reopened — Run the complete one-command development, browser, public-access, storage, personal-use packaging, and single-launch acceptance guide, including BUG-001 player-state exclusivity, BUG-002 canonical artifact consistency, BUG-003 handled active-tunnel supervisor cleanup, and BUG-004 runtime-neutral bridge verification; record conditional public-distribution checks as not applicable unless that profile is selected (reopened — BUG-003; coverage extended — BUG-002, BUG-004) · `specs/001-wails-v2-migration/quickstart.md`

**⟶ Wait for T069 to finish, then:**

- [x] **T070** ⚠️ Reopened — Verify every functional requirement and success criterion, including FR-027–FR-028 and SC-010–SC-012, against completed tests/evidence and update the quality checklist (reopened — BUG-003; coverage extended — BUG-002, BUG-004) · `specs/001-wails-v2-migration/spec.md`, `specs/001-wails-v2-migration/checklists/requirements.md`

**Historical governed-spec ordering (already satisfied before the bugfix patch):** ~~Wait for T070 to finish, then~~ T071 remains complete because BUG-002–BUG-004 add migration-handoff requirements without changing the six inherited gameplay behavior contracts; reopened T070 verifies the new migration requirements separately.

- [x] **T071** Update the six migrated behavioral specs only where runtime-specific wording changed, without altering behavior contracts · `specs/hacking-game/`, `specs/live-broadcast-shared-navigation/`, `specs/player-terminal-presentation/`, `specs/public-ngrok-access/`, `specs/session-persistence/`, `specs/terminal-content-authoring/`

---

## Dependencies & Execution Order

### Phase dependencies

- Setup blocks Foundational work because it creates the module, fixtures, frontend package, and Wails entry configuration.
- Foundational blocks every user story because it defines domain models, authoritative state, and shared lifecycle ownership.
- US1 and US2 both depend on Foundational; their isolated session/frontend and player-server surfaces may be delivered independently after that gate.
- US3 depends on US1 RuntimeStatus/events and the US2 player listener.
- US4 depends on US1–US3 because personal-use packaging must exercise the complete runtime; public-signing checks are a conditional branch, not a dependency of cutover.
- BUG-001 remediation requires T072 before reopened T040, reopened T040 before reopened T044, and reopened T044 before T064, T069, and T070 can complete.
- T064 depended on the original pre-cutover completion of T073/T074 plus T060–T063 and remains historically satisfied; reopened T074 is a post-cutover BUG-002 refresh that depends on T075 and does not require rerunning legacy deletion.
- BUG-002 remediation requires T075 before reopened T074, and both before reopened T069 and T070.
- BUG-003 remediation requires T076 before T077, T077 before T078, and T078 before reopened T069 and T070.
- BUG-004 remediation requires T079 before T080, and T080 before reopened T069 and T070.
- Reopened T069 depends on reopened T074, T075, T078, and T080; reopened T070 depends on reopened T069.
- Polish depends on US4 cutover and revalidates the clean post-Electron tree.

### Wave restatement

- Setup: governance → independent tooling/fixtures/baseline → Wails single-command composition.
- Foundational: domain tests → models → validation → nav/hack tests → nav/hack implementations → live tests → live implementation → lifecycle tests → lifecycle implementation.
- US1: independent session/bridge/startup tests → independent storage/platform/frontend work → session service → app commands → adapter/status UI → lifecycle wiring → manual evidence.
- US2: independent HTTP/protocol/server tests → independent HTTP/protocol code → connection code → server code → independent client/embed/app wiring → race gate → BUG-001 failing visibility contract test → corrected player-state visibility → repeated browser evidence including full overflow scrolling.
- US3: independent tunnel/application tests → independent configuration/platform process code → tunnel service → independent app/frontend integration → public-access evidence.
- US4: independent manifest/CI/docs work → package configuration → personal-use build and optional public-release script → personal-use build evidence → single-launch evidence → storage/shutdown evidence → parity comparison → parallel profile/rollback doc updates (T073/T074) → personal-use-gated Electron deletion → final manifest cleanup.
- Bugfix remediation: parallel canonical-artifact selection (T075), supervisor-interrupt regression (T076), and neutral-facade regression (T079) → reopened rollback sync (T074), supervisor cleanup implementation/evidence (T077/T078), and facade rename (T080) → reopened full acceptance (T069) → reopened traceability (T070).
- Polish: retained security/resource/automated gates → bugfix remediation → reopened full acceptance → reopened traceability → governed spec updates.

## Implementation Strategy

1. Complete Setup and Foundational work while retaining Electron as the oracle.
2. Deliver US1 as the first independently testable Wails slice and prove root `wails dev` startup.
3. Deliver US2 local multi-browser gameplay, then US3 protected public access.
4. Package and validate the personal-use US4 profile, including one installed-app launch without developer tooling; run public-release checks only if that profile is selected later.
5. Delete Electron only after personal-use acceptance evidence and updated profile/rollback documentation exist, then run Polish from a clean checkout.

## Notes

- Do not mark implementation tasks complete from artifact review alone.
- Do not claim package, signing, notarization, Gatekeeper, browser, audio, or live-ngrok checks passed unless they actually ran; mark public-only checks `N/A (personal profile)` rather than failed or blocked.
- Keep credentials and generated traffic-policy files out of repository content and logs.
- Preserve version-1 JSON and exact WebSocket identifiers throughout the migration.
- Treat `go test -race ./...` as mandatory for live/player code.
- Use `write-context.py` for Companion task completion and final workflow status; never hand-edit `.spec-context.json`.
