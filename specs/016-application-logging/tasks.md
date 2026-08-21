# Tasks: Application Logging

**Input**: Design documents from `specs/016-application-logging/`

## Phase 1: Setup

**Wave 1:**

- [x] **T001** Pin `github.com/obalunenko/logger` v1.2.0 as a direct production dependency and record reproducible checksums · go.mod, go.sum

## Phase 2: Foundational

**⟶ Wait for Phase 1 to finish, then:**

**Wave 1 — independent (different files):**

- [x] **T002** [P] Add a concurrency-safe recording implementation of `logger.Logger` with structured field and level capture for deterministic tests · internal/testutil/logger.go
- [x] **T003** [P] Add injected logger ownership and safe record/event-delivery helpers to the root application without changing command results · app.go
- [x] **T004** [P] Add injected logger ownership to player server configuration with a package-default fallback · internal/player/server.go

## Phase 3: User Story 1 - Diagnose Application Lifecycle (P1)

**Goal**: Record startup, readiness, absorbed startup failure, shutdown, and unexpected serving events while preserving lifecycle behavior.

**Independent Test**: Run the focused application, Wails lifecycle, and player-server tests with recording loggers and verify the required milestones and normal-shutdown silence.

### Tests

**⟶ Wait for Foundational Wave 1 to finish, then:**

**Wave 1 — independent (different files):**

- [x] **T005** [P] [US1] Add failing tests for application startup/readiness/shutdown milestone records and unchanged lifecycle outcomes · app_test.go
- [x] **T006** [P] [US1] Add a failing test that an application startup error absorbed by the Wails lifecycle adapter is recorded exactly once · wails_host_test.go
- [x] **T007** [P] [US1] Add failing tests that unexpected HTTP serving failure is recorded once and expected server closure is not · internal/player/server_test.go

### Implementation

**⟶ Wait for Lifecycle Test Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T008** [P] [US1] Initialize the production logger exactly once and replace standard-library fatal logging with contextual structured fatal records · main.go
- [x] **T009** [P] [US1] Emit root application startup, player-ready, desktop-ready, shutdown-started, and shutdown-completed records with safe fields · app.go
- [x] **T010** [P] [US1] Record the startup error intentionally absorbed by the Wails lifecycle adapter while continuing to expose runtime status · wails_host.go
- [x] **T011** [P] [US1] Record non-shutdown HTTP serving failures from the player server background worker · internal/player/server.go

**Checkpoint**: User Story 1 is independently functional when lifecycle tests observe the required records and existing startup/shutdown assertions remain unchanged.

## Phase 4: User Story 2 - Trace Important Operator Actions (P2)

**Goal**: Record safe outcomes for important trusted commands and event-delivery failures.

**Independent Test**: Exercise session, player-configuration, broadcast, public-access, and swallowed event-delivery outcomes and inspect structured operation/outcome fields.

### Tests

**⟶ Wait for User Story 1 to finish, then:**

**Wave 1:**

- [x] **T012** [US2] Add failing command-outcome and event-delivery tests covering success, cancellation, expected failure, redacted public-access failure, and forbidden-value absence · app_test.go

### Implementation

**⟶ Wait for Command Test Wave 1 to finish, then:**

**Wave 2:**

- [x] **T013** [US2] Emit safe structured outcomes for required session, player-configuration, broadcast, and public-access operations and route swallowed event errors through the logging helper · app.go

**Checkpoint**: User Story 2 is independently functional when important command outcomes and swallowed event errors are observable without logging payloads, paths, names, or credentials.

## Phase 5: User Story 3 - Keep Diagnostics Safe and Stable (P3)

**Goal**: Prove logging does not disclose protected values or change application behavior.

**Independent Test**: Run the focused logging tests with distinctive forbidden markers and the complete repository checks.

### Tests and validation

**⟶ Wait for User Story 2 to finish, then:**

**Wave 1:**

- [x] **T014** [US3] Format the change and run Go vet, full unit tests, race tests, and the repository secret-leak check against the feature's Success Criteria · affected Go files, scripts/secret-leak-check.sh

**Checkpoint**: User Story 3 is independently functional when all applicable checks pass and captured records contain zero forbidden raw values.

## Dependencies & Execution Order

- Phase 1 blocks logger imports; Phase 2 blocks all behavior tests; Phase 3 lifecycle tests block lifecycle implementation; Phase 4 command tests block command implementation; Phase 5 validates the complete change.
- Within each marked independent wave, tasks touch different files and may run in any order; every `⟶ Wait` line is a required join before the next wave.
