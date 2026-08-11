# Tasks: Hacking Game Evolution

**Input**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, and `contracts/hacking-interface.md`

## Phase 1: Setup

The feature uses the existing Go module, browser assets, deterministic random seam, test packages, and WebSocket fixtures. No new directory, dependency, generated binding, or tooling setup is required.

## Phase 2: Foundational Models

This phase establishes the shared types needed by every user story.

**Wave 1:**

- [x] **T001** Add private used-pattern state and detached public pattern projection types, and remove administrator-only fields · `internal/domain/model.go`

**Checkpoint**: Domain types compile and define the canonical/public boundary used by later story slices.

## Phase 3: User Story 1 — Solve Without Player Cheats (P1)

**Goal**: Remove every player-accessible administrator shortcut while preserving ordinary guesses.

**Independent Test**: Generate a board, send the former board/keyboard/protocol shortcuts, and verify no administrator entry exists, `HACK_ADMIN` is rejected, state is unchanged, and normal candidate guesses still work.

### Tests

**Wave 1 — independent (different files):**

- [x] **T002** [P] [US1] Add failing domain coverage for boards without `SUCCESS` or administrator metadata while retaining normal guess transitions · `internal/hack/hack_test.go`
- [x] **T003** [P] [US1] Add failing strict-decoder coverage proving `HACK_ADMIN` is unsupported and malformed replacements do not dispatch · `internal/player/protocol_test.go`
- [x] **T004** [P] [US1] Add failing bundled-player assertions that the keyboard shortcut and player cheat identifiers are absent · `internal/platform/assets_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — independent (different files):**

- [x] **T005** [P] [US1] Remove administrator generation, lookup, activation, logging, and public metadata without changing word/filler guess rules · `internal/hack/hack.go`
- [x] **T006** [P] [US1] Remove `HACK_ADMIN` from the accepted player protocol and client-message model · `internal/player/protocol.go`
- [x] **T007** [P] [US1] Remove the typed `1` administrator dispatch while retaining ordinary hacking keyboard and cell interactions · `client/client.js`

**Checkpoint**: User Story 1 is independently functional and testable; players have no force-success or bulk-dud-removal path.

## Phase 4: User Story 2 — Use Special Patterns (P1)

**Goal**: Generate, highlight, and atomically activate one-use patterns with exact dud-removal and attempt-restoration outcomes.

**Independent Test**: Exercise all four bracket pairs with deterministic random values, verify `3–6` initial patterns at every difficulty, activate each outcome, and observe one shared public update.

### Tests

**Wave 1 — independent (different files):**

- [x] **T008** [P] [US2] Add failing discovery, 1,000-board count, all-pairs, exact 80/20, one-use, dud-removal, restore, and no-dud fallback tests · `internal/hack/hack_test.go`
- [x] **T009** [P] [US2] Add failing `HACK_PATTERN` decoding/public-envelope tests and update the golden public pattern payload · `internal/player/protocol_test.go`, `internal/testutil/testdata/protocol/hack-state.json`
- [x] **T010** [P] [US2] Add failing live-service coverage for an accepted pattern effect and detached public pattern snapshots · `internal/live/service_test.go`
- [x] **T011** [P] [US2] Add failing player-server coverage for pattern dispatch, one `HACK_STATE` broadcast, and game-master callback publication · `internal/player/server_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2:**

- [x] **T012** [US2] Implement exact-count pattern construction, current-board discovery, coordinate identities, first-use effects, logs, dud replacement, attempt restoration, and public projection · `internal/hack/hack.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent (different files):**

- [x] **T013** [P] [US2] Add strict `HACK_PATTERN` decoding with `patternId` and retain the extended `HACK_STATE`/`TERMINAL_LIVE` envelope shapes · `internal/player/protocol.go`
- [x] **T014** [P] [US2] Add the mutex-protected live-service pattern transition returning a detached accepted state · `internal/live/service.go`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4:**

- [x] **T015** [US2] Dispatch accepted `HACK_PATTERN` actions through the live service and publish the resulting player/master state · `internal/player/server.go`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5:**

- [x] **T016** [US2] Render coordinate-aware pattern openings/ranges, inclusive hover preview, used-state behavior, and `HACK_PATTERN` requests without optimistic mutation · `client/client.js`

**Checkpoint**: User Story 2 is independently functional and testable across domain, protocol, live service, server, and browser surfaces.

## Phase 5: User Story 3 — Discover Stacked and Dynamic Patterns (P1)

**Goal**: Keep distinct stacked openings usable and reveal newly valid spans immediately after a dud becomes periods.

**Independent Test**: Use controlled boards with two openings sharing one close and with an alphabetic dud between matching brackets, then verify separate one-use identities and immediate rediscovery after removal.

### Tests

**Wave 1:**

- [x] **T017** [US3] Add failing stacked-opening, first-compatible-close, row-boundary, alphabetic-interior, and post-dud dynamic-discovery cases · `internal/hack/hack_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2:**

- [x] **T018** [US3] Complete row-local scanning and post-mutation rediscovery so stacked identities remain independent and new spans enter the same public projection · `internal/hack/hack.go`

**Checkpoint**: User Story 3 is independently functional and testable with stacked and dynamically created patterns.

## Phase 6: User Story 4 — Let the Game Master Resolve the Puzzle (P1)

**Goal**: Preserve the trusted game-master solve control and normal shared success publication.

**Independent Test**: Force an active puzzle through the Wails application boundary, verify no attempt is consumed, verify player/master solved publication, and verify ineligible states are rejected/disabled.

### Tests

**Wave 1 — independent (different files):**

- [x] **T019** [P] [US4] Extend bridge tests for game-master solve eligibility, unchanged attempts, shared publication, and detached pattern metadata · `app_test.go`
- [x] **T020** [P] [US4] Assert the bundled master retains `#btnHackSuccess`, `forceHackSuccess`, and solved/failed disabling while the player bundle lacks equivalent authority · `internal/platform/assets_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2:**

- [x] **T021** [US4] Deep-clone public pattern metadata across runtime status/events while retaining the existing `ForceHackSuccess` command and publication path · `app.go`

**Checkpoint**: User Story 4 is independently functional and testable; only the game master retains forced success.

## Phase 7: User Story 5 — Preserve Shared Progress Across Connections (P2)

**Goal**: Guarantee single-effect concurrency, no-op rejection, reconnect convergence, and fresh-puzzle reset.

**Independent Test**: Race two clients on one pattern, reconnect another client after used/dynamic changes, send stale and terminal-state actions, then start a fresh puzzle and compare every public field.

### Tests

**Wave 1 — independent (different files):**

- [x] **T022** [P] [US5] Add concurrent same-pattern, rejected-action immutability, detached snapshot, and fresh-set reset coverage · `internal/live/service_test.go`
- [x] **T023** [P] [US5] Add multi-client convergence, no-broadcast rejection, terminal-state no-op, and reconnect-current-pattern coverage · `internal/player/server_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — independent (different files):**

- [x] **T024** [P] [US5] Distinguish accepted pattern mutations from absent/stale/repeated/terminal no-ops at the mutex boundary and reset used state on fresh `Set` · `internal/live/service.go`
- [x] **T025** [P] [US5] Suppress player broadcasts and game-master callbacks for rejected pattern actions while preserving accepted convergence · `internal/player/server.go`

**Checkpoint**: User Story 5 is independently functional and testable under concurrency and reconnection.

## Phase 8: Polish and Success-Criteria Validation

**Wave 1:**

- [x] **T026** Run `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`, and `npm run build` in `frontend/`; fix any feature-caused failures and confirm SC-001 through SC-009 · `.`

## Dependencies & Execution Order

- Phase 1 confirms no setup work; Phase 2 blocks every story by defining the shared model.
- User Story 1 removes the old authority before User Story 2 adds its replacement; User Story 3 then extends the same discovery engine; User Story 4 verifies the separate trusted override; User Story 5 hardens shared-state behavior; Polish validates the complete vertical slice.
- Phase 3 Wave 1 blocks Wave 2. Phase 4 Wave 1 blocks Wave 2, which blocks Wave 3, then Wave 4, then Wave 5. Phase 5 Wave 1 blocks Wave 2. Phase 6 Wave 1 blocks Wave 2. Phase 7 Wave 1 blocks Wave 2. Phase 8 runs only after every story checkpoint.
- Tasks tagged `[P]` touch different files within their wave and may be completed in any order; all joins must complete before the next wave begins.
