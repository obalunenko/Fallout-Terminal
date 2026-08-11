# Tasks: Phase 1 Generation-Bound Hacking Patterns

**Input**: `spec.md`, `planning-handoff.md`, `plan.md`, `research.md`, `data-model.md`, and `contracts/hacking-interface.md`

Task IDs continue at `T027` so the prior completed task journal cannot mark this corrective task set complete accidentally.

## Phase 1: Setup

The corrective implementation uses the existing Go module, WebSocket server, browser assets, deterministic randomness seam, and test infrastructure. No package, dependency, generated binding, directory, or tooling setup is required.

## Phase 2: Foundational Identity and Projection Model

This phase establishes the generation-bound identity and minimal public shape required by every story.

**Wave 1:**

- [x] **T027** Replace coordinate-only pattern models with `GenerationID`, comparable generation/row/start/end identity, private discovery metadata, and minimal detached public `id`/`row`/`start`/`end`/`used` fields · `internal/domain/model.go`

**Checkpoint**: Canonical and public types compile with no public `column` or `pair`, and all later story work can target the complete generation-bound identity.

## Phase 3: User Story 1 — Solve Without Player Cheats (P1)

**Goal**: Preserve the removal of every player-accessible cheat and retain ordinary password/filler behavior while the model changes.

**Independent Test**: Exercise normal candidate and filler guesses, the removed administrator command, and bundled player surfaces; verify unchanged attempts and no force-success or bulk-dud path.

### Tests

**Wave 1 — independent (different files):**

- [x] **T028** [P] [US1] Update and strengthen ordinary candidate, filler-click, removed `SUCCESS`, and terminal-state regression cases against the generation-bound model · `internal/hack/hack_test.go`
- [x] **T029** [P] [US1] Strengthen bundled-player checks for removed administrator inputs, query/keyboard/global cheat paths, and unchanged ordinary cell dispatch · `internal/platform/assets_test.go`

**Checkpoint**: User Story 1 is independently verified; the corrective work has not restored any player cheat or changed ordinary attempt rules.

## Phase 4: User Story 2 — Discover and Use Special Patterns (P1)

**Goal**: Publish only final boards with `3–6` discovered patterns and apply the exact weighted outcome mapping with one required outcome draw per accepted activation.

**Independent Test**: Generate 1,000 final boards, cover all four pairs, drive all 100 equiprobable outcome values, exercise no-dud fallback, and interact through row-based inclusive browser coordinates.

### Tests

**Wave 1 — independent (different files):**

- [x] **T030** [P] [US2] Add failing final-board regeneration, `3–6` discovery, 100-value `80/20` mapping, accepted no-dud RNG consumption, rejected zero-RNG, secret-preservation, restoration, and minimal-projection tests · `internal/hack/hack_test.go`
- [x] **T031** [P] [US2] Add failing bundled-player assertions for row-local inclusive hover/click lookup, opaque `patternId` echo, used-state handling, and absence of public `column`/`pair` dependencies · `internal/platform/assets_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2:**

- [x] **T032** [US2] Implement generation-aware row discovery, final-board regeneration through the production scanner, mandatory weighted outcome draw before fallback, dud selection, and minimal detached current-pattern projection · `internal/hack/hack.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3:**

- [x] **T033** [US2] Replace column-based pattern interaction with rendered-row and row-local inclusive offsets while continuing to send the opaque server-issued `patternId` without optimistic mutation · `client/client.js`

**Checkpoint**: User Story 2 is independently functional and testable across generation, weighted effects, projection, and browser interaction.

## Phase 5: User Story 3 — Discover Stacked and Dynamic Patterns (P1)

**Goal**: Keep shared-closer openings independent and preserve permanent coordinate-pair used history across dynamic discovery changes.

**Independent Test**: Mutate controlled boards so one opening changes its first compatible closer and a previously used pair disappears and reappears; verify new identity availability and old identity unavailability.

### Tests

**Wave 1:**

- [x] **T034** [US3] Add failing stacked shared-closer, first-compatible-close, same-row, alphabetic-interior, changed-closer, disappeared/reappeared used-pair, and post-dud dynamic projection cases · `internal/hack/hack_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2:**

- [x] **T035** [US3] Complete row-local scanning and private complete-identity history so changed coordinate pairs are new and rediscovered used pairs remain unavailable · `internal/hack/hack.go`

**Checkpoint**: User Story 3 is independently functional and testable for stacked, changed, newly created, and rediscovered patterns.

## Phase 6: User Story 4 — Share One Atomic Puzzle State (P1)

**Goal**: Reject stale and duplicate generation-bound requests without RNG or mutation, publish accepted transitions in the mandated mutex order, and converge connected/reconnecting clients on detached state.

**Independent Test**: Race duplicate requests, delay an ID across two puzzle generations with coincident coordinates, mutate returned snapshots, and reconnect a client; verify one acceptance, one outcome draw, one ordered publication, and current state convergence.

### Tests

**Wave 1 — independent (different files):**

- [x] **T036** [P] [US4] Add failing deterministic generation-ID, stale-generation, duplicate zero-RNG, exact one-publication, callback-order, terminal-state, fresh-set, and detached-projection live-service tests · `internal/live/service_test.go`
- [x] **T037** [P] [US4] Update strict `HACK_PATTERN` decoder and envelope tests for opaque generation-bearing IDs, missing/unknown/invalid fields, and minimal public pattern JSON · `internal/player/protocol_test.go`
- [x] **T038** [P] [US4] Add failing multi-client duplicate, stale-generation, no-broadcast rejection, accepted ordered fanout, reconnect-current-state, and process-local reset cases · `internal/player/server_test.go`
- [x] **T039** [P] [US4] Update public-pattern fixtures and deepen returned-projection mutation tests across runtime status and game-master events · `app_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2:**

- [x] **T040** [US4] Issue collision-resistant runtime generation IDs independently of gameplay RNG and execute validation, used marking, outcome, mutation, rediscovery, detached projection, and one publication callback under the live mutex · `internal/live/service.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent (different files):**

- [x] **T041** [P] [US4] Retain strict `HACK_PATTERN` decoding while documenting and carrying the opaque generation-bearing `patternId` with no coordinate-only assumptions · `internal/player/protocol.go`
- [x] **T042** [P] [US4] Replace the golden public pattern object with `id`, `row`, inclusive `start`/`end`, and `used` only · `internal/testutil/testdata/protocol/hack-state.json`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4:**

- [x] **T043** [US4] Supply the non-reentrant publication callback, enqueue exactly one accepted `HACK_STATE` plus detached game-master notification under the live transition, and suppress every rejected publication · `internal/player/server.go`

**Checkpoint**: User Story 4 is independently functional and testable under concurrency, stale delivery, projection mutation, and reconnect.

## Phase 7: User Story 5 — Let the Game Master Resolve the Puzzle Privately (P1)

**Goal**: Preserve `ForceHackSuccess` only through the private desktop/Wails boundary after the public model and publication changes.

**Independent Test**: Invoke the trusted control for an eligible puzzle, verify unchanged attempts and normal shared success, reject ineligible states, and inspect every player surface for equivalent authority.

### Tests

**Wave 1 — independent (different files):**

- [x] **T044** [P] [US5] Update generation-bound public fixtures and verify eligible/ineligible `ForceHackSuccess`, unchanged attempts, detached events, and existing shared success publication · `app_test.go`
- [x] **T045** [P] [US5] Verify the private master control and Wails call remain intact while WebSocket messages, browser globals, DOM controls, keyboard shortcuts, query parameters, and player assets expose no equivalent · `internal/platform/assets_test.go`

**Checkpoint**: User Story 5 is independently verified; only the trusted game-master boundary can force success.

## Phase 8: Polish and Success-Criteria Validation

**Wave 1:**

- [x] **T046** Add a version-1 encode/decode regression proving generation IDs, patterns, used history, removed duds, attempts, outcomes, unlocked state, and puzzle seeds never enter persisted session JSON · `internal/domain/model_test.go`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2:**

- [x] **T047** Run `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`, and `npm --prefix frontend run build`; fix only feature-caused failures and verify the pre-camouflage definitions of SC-001 through SC-013. The amended SC-003 and SC-004 remain unverified until Phase 10 · `.`

## Dependencies & Execution Order

- Setup adds no work. Foundational T027 blocks every user-story phase.
- User Story 1 protects existing behavior before User Story 2 changes discovery/outcomes; User Story 3 then extends the same hacking engine; User Story 4 integrates generation, concurrency, projection, protocol, and publication; User Story 5 verifies the separate trusted boundary; Polish closes persistence and full-suite gates.
- Phase 3 Wave 1 is independent after T027. Phase 4 Wave 1 blocks T032, which blocks T033. Phase 5 T034 blocks T035. Phase 6 Wave 1 blocks T040, which blocks Wave 3, which blocks T043. Phase 7 Wave 1 follows the public integration. Phase 8 T046 blocks the final T047 validation.
- Tasks tagged `[P]` touch different files within their wave and may be completed in any order; every explicit join must complete before the next wave begins.

## Phase 9: Convergence

- [x] **T048** Make browser special-pattern hover and click resolution row-local and inclusive across every offset from `start` through `end`, and add an asset-contract regression covering full-span activation · `client/client.js`, `internal/platform/assets_test.go` per plan: client hover/click mapping and T031/T033 (partial)

## Phase 10: Board Camouflage and Delimiter Decoys

**Goal**: Camouflage the unchanged special-pattern discovery rules among words, ordinary filler, non-empty pattern interiors, word-interrupted spans, and inert delimiter decoys, then publish only complete boards that pass every final-board gate.

**Independent Test**: Generate 1,000 publishable boards and prove each has `3–6` production-discovered patterns, decoy parity, a non-empty valid interior, an alphabetic-interrupted span, at least two occupied rows per candidate-word, valid-endpoint, and standalone-decoy category, pairwise-overlapping occupied-row intervals, ordinary filler in at least two rows, accidental-pattern accounting, and a valid-only public projection; then execute valid-pattern, candidate-word, delimiter-decoy, computed-style, and dud-created rediscovery interactions in a real browser context.

**Verification status**: The current SC-003, SC-004, and SC-014–SC-017 are pending T054; completed T047 does not satisfy their amended definitions.

### Tests

**Wave 1 — independent (different files):**

- [x] **T049** [P] [US6] Extend generator and domain regressions with the 1,000-board camouflage gate; exact occupied-row counts and pairwise interval overlap; adjacent-empty and non-empty patterns; unmatched, mismatched, and first-closer decoys; accidental-pattern accounting; valid-only projection; inert direct delimiter targets; ordinary word selection inside invalid spans; and post-dud rediscovery while preserving the existing scanner fixtures unchanged · `internal/hack/hack_test.go`
- [x] **T050** [P] [US6] Extend bundled-player asset and style contracts for valid-only pattern lookup, candidate-word dispatch, inert delimiter dispatch, unchanged non-delimiter filler behavior, no persistent validity-dependent class, and static styling parity across the governed player stylesheet · `internal/platform/assets_test.go`, `client/client.css`
- [x] **T051** [P] [US6] Add an isolated, exactly pinned and locked executable browser-test harness plus controlled board fixtures covering valid-pattern hover/focus/click, ordinary candidate selection inside an alphabetic-interrupted span, standalone delimiter no-op behavior, outbound message capture, equal pre-interaction computed styles, no optimistic mutation, and activation only after a server snapshot publishes a dud-created pattern · `tests/browser/package.json`, `tests/browser/package-lock.json`, `tests/browser/playwright.config.mjs`, `tests/browser/hacking-camouflage.spec.mjs`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2:**

- [x] **T052** [US6] Replace bracket-free isolated-pair construction with normal-row camouflage placement, including a non-empty valid interior, alphabetic-interrupted candidate span, and standalone delimiter-decoy candidates; run unchanged production discovery on the complete board; compute occupied-row counts and pairwise interval overlap; and regenerate unless every pattern-count, decoy, interior, interruption, distribution, and projection gate passes · `internal/hack/hack.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3:**

- [x] **T053** [US6] Make rendered delimiter cells outside all current valid pattern ranges inert in canonical filler-target handling and browser dispatch while retaining ordinary candidate, non-delimiter filler, and valid-pattern behavior; change the governed stylesheet only if static-parity tests expose a difference · `internal/hack/hack.go`, `client/client.js`, `client/client.css`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4:**

- [x] **T054** Run formatting, static analysis, full tests, race tests, browser syntax checks, `npm --prefix tests/browser test`, the frontend build, and explicit SC-003/SC-004/SC-014–SC-017 verification; confirm the 1,000-board gate, executable browser interactions, and unchanged discovery fixtures pass · `.`

**Checkpoint**: User Story 6 is independently functional and the board reveals pattern validity only through normal valid-pattern interaction, never through construction grouping, public decoy metadata, static styling, or decoy side effects.

## Updated Dependencies & Execution Order

- Phase 10 starts from the completed T027–T048 baseline. T049, T050, and T051 may run in parallel; all three block T052. T052 blocks T053, and T053 blocks the final T054 verification gate.
- The existing discovery, identity, activation, probability, concurrency, projection, reconnect, private-control, and persistence rules remain prerequisites and are not reopened except where a new regression explicitly proves they remain unchanged.
