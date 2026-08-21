# Tasks: Context Propagation and Causal Cancellation

**Bugfix**: 2026-08-21 — [BUG-001] Updated from bugfix patch.

**Bugfix**: 2026-08-21 — [BUG-002] Updated from bugfix patch.

## Phase 1: Setup

**Wave 1:**

- [x] **T001** [US3] Extend the context-convention AST gate to cover all affected test files and reject lower-layer production background/placeholder fallback creation while allowing executable roots · `internal/platform/test_conventions_test.go`

**Checkpoint**: The repository has an executable guard that turns the known context-ownership violations red before production edits.

## Phase 2: Foundational

**Wave 1 — independent (different files):**

- [x] **T002** [P] [US1] Add root/application lifecycle propagation tests, required-constructor-context tests, and shutdown-cause expectations rooted in `t.Context()` · `app_test.go`, `wails_host_test.go`
- [x] **T003** [P] [US1] Add coordinator-to-session context marker coverage and update the command-state fake contract to accept context · `internal/control/service_test.go`
- [x] **T004** [P] [US1] Add player server/subscription propagation and cancellation-cause coverage rooted in `t.Context()` · `internal/player/stream_test.go`, `internal/player/handler_test.go`
- [x] **T005** [P] [US1] Add public-access manager, provider endpoint, and ingress context/cause coverage rooted in `t.Context()` · `internal/tunnel/manager_test.go`, `internal/tunnel/ngrok_test.go`, `internal/tunnel/public_ingress_test.go`, `internal/tunnel/service_test.go`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2:**

- [x] **T006** [US1] Reconcile the focused tests so they fail only for missing production propagation/cause behavior, not stale test APIs · `app_test.go`, `wails_host_test.go`, `internal/control/service_test.go`, `internal/player/*_test.go`, `internal/tunnel/*_test.go`

**Checkpoint**: Focused tests describe the required context lineage and cancellation causes before implementation.

## Phase 3: User Story 1 — Preserve operation lifetime end to end (P1)

**Goal**: Context-aware production work is rooted in application composition or the initiating operation and never silently receives an absent or replacement root.

**Independent Test**: Marker values from the process/application/test context reach lifecycle, persistence, player, platform, and tunnel fakes; absent contexts fail before work begins.

### Implementation

**Wave 1 — independent (different files):**

- [x] **T007** [P] [US1] Require and propagate the process context through composition, application construction, logger lookup, retained application context, and Wails lifecycle cleanup · `main.go`, `app.go`, `wails_host.go`
- [x] **T008** [P] [US1] Thread the application operation context through command-execution coordination and the command-state persistence adapter; reject absent session/player-config operation contexts · `internal/control/service.go`, `internal/session/service.go`, `internal/playerconfig/service.go`
- [x] **T009** [P] [US1] ⚠️ Reopened: Require the startup/request parent in the player server and subscription lifetimes, but limit the bounded startup context to pre-commit acquisition and preserve the application/player-server owned context as the committed HTTP/stream parent (reopened — BUG-001) · `internal/player/server.go`, `internal/player/stream.go`, `internal/player/handler.go`
- [x] **T010** [P] [US1] Remove lower-layer replacement roots from public-access manager, ingress, and embedded-provider operations; carry initiating values into bounded cleanup · `internal/tunnel/manager.go`, `internal/tunnel/public_ingress.go`, `internal/tunnel/ngrok.go`
- [x] **T011** [P] [US1] Require the lifecycle context in the retained desktop adapter and operation contexts at the Darwin credential boundary · `internal/platform/desktop.go`, `internal/platform/keychain_darwin.go`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2:**

- [x] **T012** [US1] ⚠️ Reopened: Update root/application, coordinator, player, platform, and tunnel call sites so production supplies the correct acquisition versus owned-lifetime context and tests supply equivalent `t.Context()` derivatives (reopened — BUG-001) · `main.go`, `app.go`, `app_test.go`, `wails_host_test.go`, `internal/**/*_test.go`

**Checkpoint**: User Story 1 is independently functional; no context-aware path passes an absent context or creates a lower-layer replacement root.

## Phase 4: User Story 2 — Explain why owned work stopped (P1)

**Goal**: Every manual application-owned cancellation has a stable semantic cause, and bounded cleanup preserves values and timeout reasons.

**Independent Test**: Closing, replacing, overflowing, timing out, or aborting representative owned lifetimes yields the expected `context.Cause`, and repeated closes retain the first cause.

### Implementation

**Wave 1 — independent (different files):**

- [x] **T013** [P] [US2] ⚠️ Reopened: Convert application, Wails cleanup, player server, and player subscription ownership to causal cancellation with stable shutdown/failure/overflow/replacement reasons, excluding successful startup completion from committed player-server causes (reopened — BUG-001) · `app.go`, `wails_host.go`, `internal/player/server.go`, `internal/player/stream.go`
- [x] **T014** [P] [US2] Convert public-access start and embedded endpoint ownership to causal cancellation; apply explicit causes to completion, timeout, request cancellation, abort, and cleanup · `internal/tunnel/manager.go`, `internal/tunnel/ngrok.go`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2:**

- [x] **T015** [US2] ⚠️ Reopened: Reconcile successful player-listener handoff, shutdown, and race seams so cleanup remains bounded, first-cause wins, and partial-start/repeated-close behavior stays idempotent (reopened — BUG-001) · `app.go`, `internal/player/server.go`, `internal/tunnel/manager.go`, `internal/tunnel/ngrok.go`

**Checkpoint**: User Story 2 is independently functional and causal cancellation is observable across all owned lifecycle classes.

## Phase 5: User Story 3 — Keep test lifetimes isolated (P2)

**Goal**: Every affected context-sensitive test is owned by its active `testing.T` and verifies the new lifecycle contract.

**Independent Test**: The AST convention gate and focused tests detect any background/placeholder test root or absent context passed to an affected constructor/operation.

### Implementation

**Wave 1:**

- [x] **T016** [US3] Replace remaining affected test roots/manual cancels with `t.Context()`-derived contexts and explicit test cancellation causes; update source-wiring assertions for context-aware signatures · `app_test.go`, `wails_host_test.go`, `production_resources_test.go`, `internal/platform/assets_test.go`, `internal/platform/desktop_test.go`, `internal/player/*_test.go`, `internal/control/service_test.go`, `internal/session/service_test.go`, `internal/tunnel/*_test.go`

**Checkpoint**: User Story 3 is independently functional; context-sensitive tests are test-owned and convention-enforced.

## Phase 6: Polish

**Wave 1:**

- [x] **T017** Validate formatting, static analysis, the complete Go suite, and context scans against all Functional Requirements and Success Criteria · `gofmt -l .`, `go vet ./...`, `go test ./...`, `rg`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2:**

- [x] **T018** Run concurrency and public-access safety validation, including `go test -race ./...` and the secret-leak check, then record any unavailable check honestly · `go test -race ./...`, `scripts/secret-leak-check.sh`

## Phase 7: BUG-001 Player-listener Handoff Correction

**Wave 1:**

- [x] **T019** [US1] Add a RED real-server regression using the production non-zero application startup timeout; after `App.Start` completes and its acquisition context is canceled, prove a later `Subscribe` receives a complete snapshot and a subsequent authoritative update while the committed server context remains active · `app_test.go`, `internal/player/stream_test.go`

**⟶ T019 must fail for the reported lifecycle bug; then complete reopened T009, T012, T013, and T015 before Wave 2.**

**Wave 2:**

- [x] **T020** [US1] Add public-ingress/browser regression coverage proving an authenticated player can remain connected after character selection and receive a later on-air update without the lost-connection overlay · `internal/player/public_stream_test.go`, `tests/browser/player-sessions-control.spec.mjs`

**⟶ Wait for T009, T012, T013, T015, and T020 to finish, then:**

**Wave 3:**

- [x] **T021** [US1] Validate BUG-001 with focused player/application/public-stream tests, formatting, vet, the complete Go suite, the race-enabled Go suite, and the secret-leak check · `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`, `scripts/secret-leak-check.sh`

## Phase 8: BUG-002 Player-page Console Hygiene

**Wave 1:**

- [x] **T022** [US1] Add RED source/HTTP and clean Playwright regressions that capture page diagnostics before navigation, require zero first-party console warnings/errors and failed same-origin static-resource responses, prove `/favicon.ico` does not return 404, require `frame-ancestors 'none'` only in the HTTP CSP header, and keep extension-injected script diagnostics outside first-party attribution · `internal/player/http_test.go`, `internal/platform/assets_test.go`, `tests/browser/player-sessions-control.spec.mjs`

**⟶ T022 must fail for the reported CSP-meta warning and favicon request before implementation.**

**Wave 2:**

- [x] **T023** [US1] Remove the unsupported `frame-ancestors 'none'` directive from the player HTML meta CSP while retaining it in the server-supplied `Content-Security-Policy` header and updating the static source contract accordingly · `frontend/client/index.html`, `internal/player/http_test.go`, `internal/platform/assets_test.go`

**⟶ T023 must pass the CSP portion of T022 before Wave 3.**

**Wave 3:**

- [x] **T024** [US1] Declare an intentional empty data favicon in the player document so clean Chromium navigation makes no failing `/favicon.ico` request, then satisfy the source and browser request assertions without adding a dependency or binary asset · `frontend/client/index.html`, `internal/platform/assets_test.go`, `tests/browser/player-sessions-control.spec.mjs`

**⟶ T024 must pass all focused T022 regressions before Wave 4.**

**Wave 4:**

- [x] **T025** [US1] Validate BUG-002 with the client production build, focused player HTTP/platform tests, the focused Playwright player journey, formatting, vet, the complete Go suite, and the race-enabled Go suite · `npm run build:client --prefix frontend`, `go test ./internal/player ./internal/platform`, `npm test --prefix tests/browser -- player-sessions-control.spec.mjs`, `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`

## Dependencies & Execution Order

- Setup T001 establishes the red convention gate before foundational tests.
- Foundational Wave 1 runs T002–T005 independently; T006 joins their API expectations.
- User Story 1 Wave 1 runs T007–T011 independently by subsystem; T012 updates and reconciles all call sites after those APIs settle.
- User Story 2 Wave 1 runs T013–T014 independently; T015 joins their lifecycle and race behavior.
- User Story 3 T016 follows the production API/cause changes so every test uses the final signatures.
- Polish T017 runs functional verification before the heavier race and secret-safety validation in T018.
- BUG-001 correction begins with RED regression T019; reopened T009 and T013 correct the player-server ownership/cause implementation, reopened T012 reconciles production call sites, and reopened T015 joins the handoff and race seams.
- T020 follows the reopened implementation work and proves public-player behavior; T021 runs only after T009, T012, T013, T015, and T020 complete.
- BUG-002 begins with RED console/request coverage in T022; T023 corrects CSP delivery before T024 resolves the overlapping player-document favicon behavior, and T025 runs only after all focused T022 assertions pass.
