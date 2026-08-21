# Tasks: Overseer Action Clarity

## Phase 1: Setup

**Wave 1:**

- [x] **T001** Add failing static resource assertions for the canonical labels, contextual ownership, modal accessibility hooks, and removal of superseded action copy · `internal/platform/assets_test.go`

## Phase 2: Foundational

**Wave 1 — independent (different files):**

- [x] **T002** [P] Add the contextual action containers, additional-settings disclosure, named-creation dialog, and take-off-air confirmation markup with the specified stable IDs · `frontend/overseer/src/index.html`
- [x] **T003** [P] Add responsive selected-terminal, editor, broadcast-action, disclosure, validation, pending, and focus-visible styles without altering the player UI · `frontend/overseer/src/overseer.css`

**Checkpoint**: The Overseer DOM and presentation primitives exist for every clarified action while backend and persistence contracts remain unchanged.

## Phase 3: User Story 1 — Understand terminal actions at a glance (P1)

**Goal**: Make each terminal action's scope clear from its exact label, owning context, availability, and active status.

**Independent Test**: Select inactive and active terminals with and without a broadcast and verify the action matrix, status, secondary settings explanation, and command boundaries.

### Tests

**Wave 1:**

- [x] **T004** [US1] Add failing Playwright coverage for inactive/active/no-broadcast visibility, exact labels, additional-settings keyboard behavior, reapplication call mapping, and duplicate-request prevention · `tests/browser/state-changing-command-authoring.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T005** [US1] Derive selected-terminal and broadcast action visibility from authoritative state, show `В ЭФИРЕ`, expose explained `ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ` only in the additional menu, and preserve explicit activation command handling · `frontend/overseer/src/overseer.js`

**Checkpoint**: User Story 1 is independently functional and testable; every visible action has one contextual meaning.

## Phase 4: User Story 2 — Publish edits without changing live-session identity (P1)

**Goal**: Present publication as the sole primary content-update action for the selected active terminal while preserving existing live-update semantics.

**Independent Test**: Edit active-terminal content, publish it, verify one `UpdateLiveTerminal` request and unchanged active identity; select an inactive terminal and verify publication disappears.

### Tests

**Wave 1:**

- [x] **T006** [US2] Extend the authoring browser journey with publication payload, active-identity, pending, acknowledgement, failure, and inactive-selection assertions · `tests/browser/state-changing-command-authoring.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T007** [US2] Rename and gate publication in the editor context, keep `UpdateLiveTerminal` as its only command, and retain accurate pending/success/failure feedback · `frontend/overseer/src/overseer.js`

**Checkpoint**: User Story 2 is independently functional and testable; publishing cannot be confused with activation or full settings reapplication.

## Phase 5: User Story 3 — Create and remove terminal states safely (P2)

**Goal**: Require a named confirmation before creating a saved draft and a semantic confirmation before removing the active terminal from player view.

**Independent Test**: Cancel, reject blank, and confirm terminal creation; then cancel, fail, decision-chain, and confirm take-off-air while checking calls, preserved state, Escape, duplicate prevention, and restored focus.

### Tests

**Wave 1 — independent (different files):**

- [x] **T008** [P] [US3] Add deterministic fixture controls for successful, failed, delayed, and decision-required clear responses while continuing to record the existing desktop methods and payloads · `tests/browser/fixtures/desktop-bindings.js`
- [x] **T009** [P] [US3] Add failing browser journeys for named creation validation/cancellation/autosave and mandatory take-off-air confirmation/error/decision/focus behavior · `tests/browser/state-changing-command-authoring.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T010** [US3] Implement the accessible create-terminal dialog with trimmed required name, exactly-once draft creation/autosave, Escape cancellation, and focus restoration · `frontend/overseer/src/overseer.js`

**⟶ Wait for T010 to finish, then:**

- [x] **T011** [US3] Implement the always-on take-off-air confirmation with preserved-state explanation, exactly-once clear request, pending/error handling, existing unfinished-progress decision handoff, and surviving-control focus · `frontend/overseer/src/overseer.js`

**Checkpoint**: User Story 3 is independently functional and testable; creation, live clearing, durable deletion, and broadcast ending remain distinct operations.

## Phase 6: Polish & Cross-Cutting Concerns

**Wave 1:**

- [x] **T012** Run JavaScript syntax checks, focused static and Playwright journeys, the full Go and browser suites, `gofmt -l .`, `go vet ./...`, Overseer production build, and native application build against SC-001–SC-006 · `specs/013-overseer-action-clarity/tasks.md`

## Dependencies & Execution Order

- Phase order: Setup → Foundational → User Story 1 → User Story 2 → User Story 3 → Polish.
- Setup Wave 1 establishes failing static contracts. Foundational Wave 1 can proceed independently across HTML and CSS, then joins before story JavaScript work.
- User Story 1: T004 blocks T005. User Story 2: T006 blocks T007 and begins after User Story 1 so same-file browser and JavaScript edits stay serialized.
- User Story 3: T008 and T009 are independent, then T010 blocks T011 because both edit `overseer.js`.
- Polish T012 validates the integrated result only after all story checkpoints pass.
