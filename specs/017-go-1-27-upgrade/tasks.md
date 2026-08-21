# Tasks: Go 1.27 Upgrade

**Input**: Design documents from `specs/017-go-1-27-upgrade/`

**Bugfix**: 2026-08-21 — BUG-001 updated from bugfix patch.

## Phase 1: Setup

No new dependencies or project structure are required.

## Phase 2: Foundational - Declare the Baseline

**Wave 1 — independent (different files):**

- [x] **T001** [P] Set the root application module minimum to Go 1.27.0 without changing dependency pins · go.mod
- [x] **T002** [P] Set every isolated Go development-tool module minimum to Go 1.27.0 without changing tool pins · tools/buf/go.mod, tools/protoc-gen-connect-go/go.mod, tools/protoc-gen-go/go.mod, tools/wails/go.mod

## Phase 3: User Story 1 - Build with the Current Go Release (P1)

**Goal**: Make active module validation create and enforce the Go 1.27.0 baseline.

**Independent Test**: Inspect all five module declarations, run module maintenance and isolation checks with Go 1.27, and verify no dependency or ownership drift.

### Tests and implementation

**⟶ Wait for Foundational Wave 1 to finish, then:**

**Wave 1 — independent (different files):**

- [x] **T003** [P] [US1] Update the tool-module acceptance assertion to require Go 1.27.0 · internal/platform/startup_test.go
- [x] **T004** [P] [US1] Update active module fixtures to emit Go 1.27.0 while preserving each fixture's negative-test purpose · scripts/tool-modules-check.sh, scripts/wails-v3-contract-check.sh, scripts/wails-v3-cutover-check.sh

**⟶ Wait for User Story 1 Wave 1 to finish, then:**

**Wave 2:**

- [x] **T005** [US1] Run Go 1.27 module maintenance and targeted module/isolation checks; accept no dependency-pin, checksum-ownership, or generated-output drift · go.mod, go.sum, tools/*/go.mod, tools/*/go.sum

**Checkpoint**: User Story 1 is independently functional when every module and fixture uses Go 1.27.0 and the isolated module checks pass without graph drift.

## Phase 4: User Story 2 - Keep Automation and Guidance Consistent (P2)

**Goal**: Align active CI, release validation, contributor guidance, planning defaults, and governance with Go 1.27.

**Independent Test**: Scan active non-historical files for stale Go 1.26 requirements, exercise release/version fixtures, and confirm CI and guidance identify the Go 1.27 release line.

### Implementation

**⟶ Wait for User Story 1 to finish, then:**

**Wave 1 — independent (different files):**

- [x] **T006** [P] [US2] Select Go 1.27.x in CI and require stable Go 1.27 patch output in macOS release preflight · .github/workflows/wails-macos.yml, scripts/build-macos.sh
- [x] **T007** [P] [US2] Update active contributor guidance, the default planning runtime, and constitution governance to Go 1.27 with a patch amendment record · README.md, .specify/templates/plan-template.md, .specify/memory/constitution.md

**Checkpoint**: User Story 2 is independently functional when every active operational and guidance surface agrees on Go 1.27 and historical records remain untouched.

## Final Phase: Polish and Validation

**⟶ Wait for User Story 2 Wave 1 to finish, then:**

**Wave 1:**

- [x] **T008** Repair stale quality-gate prerequisites/baselines, then run the active-version scan, formatting, vet, unit, race, module-isolation, Wails contract/cutover, protobuf generation, binding, and reproducibility checks with Go 1.27 · Makefile, scripts/wails-v3-contract-check.sh, repository validation surfaces

**⟶ Wait for Validation Wave 1 to finish, then:**

**Wave 2:**

- [x] **T009** Build the macOS application package and verify the final diff contains no unrelated dependency, generated-source, or historical-evidence changes · build/bin/Fallout Terminal.app, repository diff

**⟶ Wait for Validation Wave 2 to finish, then:**

**Wave 3:**

- [x] **T010** [US1] Run the canonical Go test target with a fresh writable build cache and verify the captured output contains zero linker target-version warnings · Makefile, specs/017-go-1-27-upgrade/bugs/BUG-001.md

## Dependencies & Execution Order

- Foundational Wave 1 establishes the shared module baseline; User Story 1 updates and validates module enforcement; User Story 2 aligns automation and guidance; Polish runs the full verification graph and package build.
- T001 and T002 are independent. T003 and T004 wait for both module declarations. T005 waits for their fixture/assertion updates. T006 and T007 are independent after User Story 1. T008 blocks T009, and T009 blocks the BUG-001 regression task T010.
