# Tasks: CRT Rendering and Motion Effects

**Input**: Design documents from `/specs/008-crt-rendering/`

**Tests**: Required by the feature specification and constitution. Story tests are added before their implementation and must fail for the intended missing behavior before the corresponding implementation wave begins.

**Organization**: Tasks are grouped by user story and ordered into explicit dependency waves. `[P]` means tasks in that wave touch different files and may be completed independently.

**Bugfix**: 2026-08-16 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-08-17 — BUG-002 Updated from bugfix patch

## Phase 1: Setup

Create the shared browser-test harness used by every story.

**Wave 1:**

- [x] **T001** Create the CRT Playwright spec harness with helpers for player assignment, deterministic fixture activation, action observation, animation freezing, viewport checks, and audio-failure injection · `tests/browser/crt-rendering.spec.mjs`

---

## Phase 2: Foundational

Provide deterministic content and state transitions that block the story-level browser journeys.

**⟶ Wait for Phase 1 to finish, then Wave 1:**

- [x] **T002** Add CRT fixture data and local endpoints for a 25-row folder, empty folder, multiline record and command, markup-like authored text, unchanged/replacement updates, hacking, and blocked states without changing production contracts · `tests/browser/fixture-server/main.go`

**Checkpoint**: The existing player can be driven through every deterministic CRT acceptance state.

---

## Phase 3: User Story 1 — Experience a Cohesive CRT Display (Priority: P1) 🎯 MVP

**Goal**: Every supported player state uses the historical CRT shell, exact state colors, aligned decorative layers, and non-blocking interaction.

**Independent Test**: Exercise connection, idle, selection, waiting, list, record, command, hacking, and blocked states at all three required viewports; verify shell consistency, overlay hit testing, and historical-color snapshots.

### Tests

**Wave 1 — independent (different files):**

- [x] **T003** [P] [US1] Add failing `TestPlayerCRTVisualShellAssetContract` assertions for shell/layer identifiers, explicit decorative semantics, pointer transparency, historical state colors, responsive bounds, CSP, and safe player capability boundaries using Testify · `internal/platform/assets_test.go`
- [x] **T004** [P] [US1] Add failing Playwright journeys for all nine presentation states, screen/layer geometry, overlay hit testing, connection precedence, historical selection/focus/active/hacking-hover states, and 360×640, 768×720, and 1440×900 containment · `tests/browser/crt-rendering.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to fail for the intended missing behavior, then Wave 2 — independent (different files):**

- [x] **T005** [P] [US1] Mark scanline and vignette nodes explicitly decorative while preserving the existing CRT DOM order, connection overlay, and player control identifiers · `client/index.html`
- [x] **T006** [P] [US1] Preserve the exact historical shell and state palette, align scanline/vignette clipping with the responsive screen radius, and retain pointer-transparent overlays without adding a motion preference branch · `client/client.css`

**⟶ Wait for Wave 2 to finish, then Wave 3:**

- [x] **T007** [US1] Generate, review, and commit approved stable snapshots for selection, focus, active, and hacking-hover states at every viewport where each state is exercised · `tests/browser/crt-rendering.spec.mjs-snapshots/`

**Checkpoint**: User Story 1 is independently functional and testable as the CRT visual MVP.

---

## Phase 4: User Story 2 — Receive Purposeful Terminal Motion (Priority: P2)

**Goal**: Persistent CRT motion uses the exact historical timing while new content reveals once in order and stale or layout-only renders never replay.

**Independent Test**: Inspect browser animation data, reveal a 25-row folder, record, and command, then exercise unchanged updates, replacement, pagination, resize, font readiness, and hacking fitting.

### Tests

**Wave 1 — independent (different files):**

- [x] **T008** [P] [US2] Add failing `TestPlayerCRTMotionAndRevealAssetContract` assertions for the exact six-second flicker checkpoints, one-second hard-step blink/pulse rules, `REVEAL_DELAY_MS = 40`, generation-safe reveal completion/cancellation structure, identity suppression, and absence of reduced-motion overrides · `internal/platform/assets_test.go`
- [x] **T009** [P] [US2] Add failing Playwright checks for exact flicker keyframes/timing, hard-step indicators, ordered 25-row completion within 1.2 seconds, unchanged-identity suppression, replacement cancellation, and no replay from pagination, resize, font readiness, or hacking fitting · `tests/browser/crt-rendering.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to fail for the intended missing behavior, then Wave 2:**

- [x] **T010** [US2] Refactor `revealInto` into an idempotent container-scoped controller with ordered nodes, next index, generation validation, one pending timer, synchronous completion, safe cancellation before replacement, and existing folder/record/command identity semantics · `client/client.js`

**Checkpoint**: User Story 2 is independently functional and testable on top of the CRT shell.

---

## Phase 5: User Story 3 — Skip Progressive Reveal and Retain Readability (Priority: P3)

**Goal**: Any physical key completes the active visible-page reveal within 100ms, that press and its repeats perform no normal action, and later input and later content work normally.

**Independent Test**: Skip folder, record, and command reveals with navigation, activation, page, back, and hacking keys; hold the consumed key, release it, use a later key, open new content, simulate audio failure, and verify literal authored content and all viewports.

### Tests

**Wave 1 — independent (different files):**

- [x] **T011** [P] [US3] Add failing `TestPlayerCRTRevealSkipAssetContract` assertions for the capture-phase key guard, visible-page controller registry, `preventDefault`, dispatch consumption, same-press repeat suppression through keyup, teardown cleanup, text-only node creation, and no persisted/shared skip state · `internal/platform/assets_test.go`
- [x] **T012** [P] [US3] Add failing Playwright checks for sub-100ms completion by any key, zero action from the consumed press and repeats, normal next-press handling, independent later reveals, always-on remaining CRT effects, literal markup-like content, audio discovery/playback failure isolation, and supported viewport usability · `tests/browser/crt-rendering.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to fail for the intended missing behavior, then Wave 2:**

- [x] **T013** [US3] Add the capture-phase reveal-skip guard and current-page controller registry, consume the triggering event, suppress repeats for the same physical key until keyup, clear guard/registry state on completion or teardown, and leave later keyboard handling and persistent CRT effects unchanged · `client/client.js`

**Checkpoint**: User Story 3 is independently functional and testable; reveal skipping is one-shot per active page and is not a motion mode.

---

## Phase 6: Polish and Cross-Cutting Validation

Finish documentation alignment and run the single task-owned Success Criteria validation sequence; no post-implement hook claims validation ownership.

**Wave 1:**

- [x] **T014** Reconcile the verification guide with the implemented selectors, fixture routes, snapshot names, focused commands, and honestly observable manual steps; remove any stale reduced-motion or audio call-count instructions · `specs/008-crt-rendering/quickstart.md`

**⟶ Wait for Wave 1 to finish, then Wave 2:**

- [x] **T015** Validate SC-001 through SC-008 by running `npm ci --prefix client`, `npm run build --prefix client`, `gofmt -l .`, `go vet ./...`, `go test ./...`, clean Wails binding generation with a drift check, `npm ci --prefix tests/browser`, the focused CRT Playwright spec, and the full `npm test --prefix tests/browser` regression suite · `specs/008-crt-rendering/quickstart.md`

**⟶ Wait for Wave 2 to pass, then Wave 3:**

- [x] **T016** Run `go run ./cmd/build dev` for the complete interactive reveal/skip/state/viewport journey, then run `go run ./cmd/build build` and verify the unsigned application embeds the tested player assets; report any unavailable manual check rather than inferring it · `specs/008-crt-rendering/quickstart.md`

---

## Dependencies & Execution Order

### Phase dependencies

```text
Phase 1 Setup
  → Phase 2 Foundational
      → Phase 3 US1 (P1 visual MVP)
          → Phase 4 US2 (motion and reveal lifecycle)
              → Phase 5 US3 (consumed-key reveal skip)
                  → Phase 6 Polish and validation
                      → Phase 7 Convergence
                          → Phase 8 BUG-001 hacking-code reveal
                              → Phase 9 BUG-002 dud-removal reconciliation
```

- **Phase 1**: T001 establishes the shared browser harness.
- **Phase 2**: T002 adds the deterministic fixture after the harness shape is known.
- **Phase 3**: T003 and T004 form the independent failing-test wave; T005 and T006 form the independent implementation wave; T007 waits for both.
- **Phase 4**: T008 and T009 form the independent failing-test wave; T010 waits for both.
- **Phase 5**: T011 and T012 form the independent failing-test wave; T013 waits for both and for the reveal controller from T010.
- **Phase 6**: T014 documentation alignment blocks T015 automated validation; T016 interactive/build verification runs only after all automated gates pass.
- **Phase 7**: T017 and T018 establish the completed convergence baseline.
- **Phase 8**: T019 establishes the fixture; T020 and T021 are the failing-test wave; T022 implements the fix; T023 updates the guide; T024 owns final validation.
- **Phase 9**: T025 establishes deterministic dud removal; T026 and T027 are the failing-test wave; T028 implements reconciliation; T029 updates the guide; T030 owns final validation.

### Story delivery order

- **US1** is the MVP and proves the shared CRT shell independently.
- **US2** builds on the shell and proves exact persistent motion plus safe reveal lifecycle.
- **US3** builds on the reveal controller and proves consumed-key completion, safety, and continued usability.
- **BUG-001 / US2** extends the completed reveal controller to a new hacking-generation identity after the convergence baseline.
- **BUG-002 / US2** preserves the completed BUG-001 reveal across same-generation dud-removal mutations.
- Within `[P]` waves, tasks touch different files and may be executed in any order; every join must complete before the next wave begins.

## Phase 7: Convergence

- [x] T017 Preserve completed or active reveal identity across authoritative terminal snapshots and reconnects when kind, key, and text are unchanged, while continuing to cancel and restart genuinely replaced content; add mutation-based Playwright coverage that detects remove/reappend replay per FR-010 (partial)
- [x] T018 Expand the CRT browser journey to exercise connection, idle, selection, waiting, list, record, command, active hacking, and blocked hacking states at 360×640, 768×720, and 1440×900, verifying containment and essential-control operability per SC-006 (partial)

## Phase 8: BUG-001 — Progressive Hacking-Code Reveal

**Goal**: A newly generated hacking board reveals complete code rows at the 40ms cadence, preserves progress across same-identity updates and fit work, cancels stale generations, and consumes a skip key before hacking input.

**Independent Test**: Enter a deterministic new hacking generation, observe fewer than all rows on the first frame and ordered row growth, verify unrevealed targets are unreachable, exercise unchanged updates/reconnect/fit, replace the generation mid-reveal, and skip with a hacking key before using a later key normally.

### Fixture

**Wave 1:**

- [x] **T019** [US2] Extend the deterministic CRT hacking fixture with stable generation and board identities plus same-identity update, reconnect, and replacement transitions for BUG-001 reveal observation · `tests/browser/fixture-server/main.go`

### Tests

**⟶ Wait for T019, then Wave 2 — independent (different files):**

- [x] **T020** [P] [US2] Add failing player-asset contract assertions for safe complete-row hacking DOM construction, generation-plus-board identity, controller registration/cancellation, and absence of atomic full-grid insertion · `internal/platform/assets_test.go`
- [x] **T021** [P] [US2] Add failing Playwright checks for first-frame partial hacking content, 40ms deterministic row order, unrevealed-target non-interactivity, same-identity progress preservation, replacement cancellation, sub-100ms key completion, zero action from the consumed press/repeats, and normal later hacking input · `tests/browser/crt-rendering.spec.mjs`

### Implementation

**⟶ Wait for T020 and T021 to fail for the intended BUG-001 behavior, then Wave 3:**

- [x] **T022** [US2] Implement generation-safe hacking code-row construction and reveal through the active-page controller, preserve progress across attempts/log/hover/reconnect/fit updates, cancel replacement generations, and gate unrevealed targets while retaining existing hacking rules · `client/client.js`

### Validation

**⟶ Wait for T022, then Wave 4:**

- [x] **T023** [US2] Update the CRT verification guide with the deterministic hacking reveal, skip, same-identity, replacement, and interactive runtime journey · `specs/008-crt-rendering/quickstart.md`

**⟶ Wait for T023, then Wave 5:**

- [x] **T024** [US2] Validate FR-017 through FR-021 and SC-009 through SC-010 with the focused asset/browser checks, full Go and browser regressions, player build, interactive first-entry/replacement/skip journey, and unsigned application build · `specs/008-crt-rendering/quickstart.md`

### BUG-001 dependency DAG

```text
T019 fixture
  ├─ T020 asset contract ─┐
  └─ T021 browser tests ──┴─→ T022 implementation
                                  → T023 guide
                                      → T024 validation
```

- Phase 8 starts after the completed convergence baseline in Phase 7.
- No completed task is a false completion under the pre-BUG-001 specification, so T001–T018 remain closed.
- T020 and T021 may run in parallel after T019 because they touch different verification files; both must fail for the intended missing behavior before T022 begins.
- T024 is the sole validation owner for the added BUG-001 requirements and success criteria.

## Phase 9: BUG-002 — Same-Generation Dud-Removal Reconciliation

**Goal**: Removing a dud within the active hacking generation updates only affected board content and related pattern/log state while preserving unaffected row nodes, active reveal progress, interaction gating, and normal later input.

**Independent Test**: Activate a deterministic delimiter pattern that removes a dud in an already revealed row and in a not-yet-revealed row; observe zero unrelated row removals, stable unaffected node identity, no second reveal, an authoritative affected candidate update, no early pending-row target, and normal later hacking behavior.

### Fixture

**Wave 1:**

- [x] **T025** [US2] Extend the deterministic CRT hacking fixture with a pattern activation that guarantees dud removal while retaining the same authoritative generation, with selectable revealed-row and pending-row outcomes · `tests/browser/fixture-server/main.go`

### Tests

**⟶ Wait for T025, then Wave 2 — independent (different files):**

- [x] **T026** [P] [US2] Add failing player-asset contract assertions for generation-only hacking reveal identity, generation-local board snapshots, coordinate-keyed revealed/queued row reconciliation, and absence of full-container replacement for same-generation dud removal · `internal/platform/assets_test.go`
- [x] **T027** [P] [US2] Add failing Playwright checks that deterministic dud removal preserves unaffected revealed-row node identity and active cadence, removes zero unrelated rows, updates the affected candidate and pattern/log state, leaves a pending row unavailable until its original turn, starts zero additional reveals, and permits normal later hacking input · `tests/browser/crt-rendering.spec.mjs`

### Implementation

**⟶ Wait for T026 and T027 to fail for the intended BUG-002 behavior, then Wave 3:**

- [x] **T028** [US2] Separate hacking generation identity from mutable board snapshots and reconcile dud-removal deltas into existing revealed or queued row descriptors without clearing the board, replacing unaffected nodes, restarting timers, exposing pending targets, or changing authoritative hacking rules · `client/client.js`

### Validation

**⟶ Wait for T028, then Wave 4:**

- [x] **T029** [US2] Update the CRT verification guide with deterministic revealed-row and pending-row dud-removal journeys, DOM-identity observation, interaction gating, and replacement-generation regression steps · `specs/008-crt-rendering/quickstart.md`

**⟶ Wait for T029, then Wave 5:**

- [x] **T030** [US2] Validate FR-018, FR-019, FR-022, SC-010, and SC-011 with focused asset/browser checks, full Go and browser regressions, player build, interactive dud-removal/replacement journey, generated-binding drift check, and unsigned application build · `specs/008-crt-rendering/quickstart.md`

### BUG-002 dependency DAG

```text
T025 fixture
  ├─ T026 asset contract ─┐
  └─ T027 browser tests ──┴─→ T028 implementation
                                  → T029 guide
                                      → T030 validation
```

- Phase 9 starts after the completed BUG-001 baseline in Phase 8.
- No completed task is a false completion under the pre-BUG-002 specification, so T001–T024 remain closed.
- T026 and T027 may run in parallel after T025 because they touch different verification files; both must fail for the intended missing behavior before T028 begins.
- T030 is the sole validation owner for the added BUG-002 requirement and success criterion.
