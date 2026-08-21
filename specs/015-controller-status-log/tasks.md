# Tasks: Immersive Controller Status Log

**Input**: Design documents from `specs/015-controller-status-log/`
**Required**: [spec.md](./spec.md), [plan.md](./plan.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/player-status-line.md](./contracts/player-status-line.md)

## Phase 1: Setup

No new project structure, dependency, generated code, or tooling setup is required. The feature uses the existing player-client and Playwright workspaces.

## Phase 2: Foundational

**Goal**: Establish the single lower-chrome semantic surface that all role presentations use.

**Wave 1 — blocking markup cutover:**

- [x] **T001** Reshape and relocate the existing `playerIdentity` markup after terminal content/navigation and immediately before `termPrompt`, retaining stable identity/role identifiers and explicit separator elements · frontend/client/index.html

**Checkpoint**: One semantic identity surface exists in lower terminal chrome; the framed in-body surface is removed.

---

## Phase 3: User Story 1 — See controller ownership without breaking immersion (Priority: P1) 🎯 MVP

**Goal**: Render the approved active-controller system line above the prompt with no framed or coloured badge presentation.

**Independent Test**: Assign the default first player to Nick Valentine and verify the approved line, absence of the old presentation, and placement above the prompt.

### Tests

**Wave 1 — write the failing acceptance coverage:**

- [x] **T002** [US1] Add focused browser assertions for the exact approved active system line, compact `P1` label, stable semantics, and vertical order above `termPrompt` · tests/browser/terminal-navigation.spec.mjs

**⟶ Wait for Wave 1 to fail against the legacy UI, then:**

### Implementation

**Wave 2 — independent after T001 (different files):**

- [x] **T003** [P] [US1] Derive compact default input labels, optional character segments, and terminal-native active/observer/unassigned role text from authoritative `playerState`, updating the existing surface in place · frontend/client/client.js
- [x] **T004** [P] [US1] Replace framed/badge and role-colour styling with a dim wrapping lower-chrome log line that remains subordinate to terminal content · frontend/client/client.css

**⟶ Wait for Wave 2 to finish, then re-run T002 acceptance coverage.**

**Checkpoint**: User Story 1 is independently functional and the selected variant is visible for the active first-player scenario.

---

## Phase 4: User Story 2 — Retain clear status for every player role (Priority: P2)

**Goal**: Preserve role clarity through assignment, authority transfer, reconnect, and native smoke journeys.

**Independent Test**: Exercise active, observer, and unassigned projections and confirm one current terminal-native status message with no stale or duplicate segment.

### Tests

**Wave 1 — independent wording consumers (different files):**

- [x] **T005** [P] [US2] Update intentional active-role assertions to `АКТИВЕН` while retaining observer checks across approval, synchronization, session-control, and navigation journeys · tests/browser/state-changing-command-approval.spec.mjs, tests/browser/state-changing-command-sync.spec.mjs, tests/browser/player-sessions-control.spec.mjs, tests/browser/terminal-navigation.spec.mjs
- [x] **T006** [P] [US2] Update the native player reset smoke role assertion to the new canonical active wording · scripts/state-changing-reset-native-player-smoke.mjs

**Checkpoint**: User Story 2 is independently verified across browser and native smoke consumers of the role text.

---

## Phase 5: User Story 3 — Preserve the terminal across viewport sizes (Priority: P3)

**Goal**: Confirm the lower line does not overlap content, navigation, prompt, or specialised screens at supported viewport shapes.

**Independent Test**: Render wide, narrow, and short player viewports plus the hacking surface and compare visible element bounding boxes.

### Tests

**Wave 1 — geometry coverage after the lower-chrome implementation:**

- [x] **T007** [US3] Add representative narrow, short, and specialised-surface containment assertions for the status line, prompt, and terminal content · tests/browser/terminal-navigation.spec.mjs

**Checkpoint**: User Story 3 is independently verified with zero lower-status overlaps in representative layouts.

---

## Final Phase: Polish & Cross-Cutting Validation

**Wave 1 — validation after all stories join:**

- [x] **T008** Validate Success Criteria with the player-client production build and affected Playwright journeys; report unavailable environment-dependent checks without claiming them · frontend/package.json, tests/browser/package.json

## Dependencies & Execution Order

- Setup has no work; Foundational T001 blocks the rendering tasks.
- User Story 1 runs T002 first, then joins independent T003 and T004 before its acceptance re-run.
- User Story 2 depends on the canonical role wording from T003; T005 and T006 are independent consumers.
- User Story 3 depends on the final markup and styling from T001–T004, then runs T007.
- Polish T008 runs only after all story checkpoints.

Wave restatement: `T001 → T002 → (T003 ∥ T004) → (T005 ∥ T006) → T007 → T008`.
