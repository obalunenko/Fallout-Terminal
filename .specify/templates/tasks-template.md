---
description: "Task list template for Fallout Terminal feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`

**Prerequisites**: `plan.md` and `spec.md`; include `research.md`, `data-model.md`, `contracts/`, and `quickstart.md` when the plan requires them

**Testing**: No automated runner is currently configured. Include automated test tasks when the specification or plan requires them; if introducing a runner, first add the chosen dependency, test location, and npm script. Always include concrete manual verification tasks for affected Electron and browser journeys.

**Organization**: Group tasks by prioritized user story so each story remains independently implementable and verifiable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Safe to execute in parallel because files do not overlap and prerequisites are complete
- **[Story]**: User-story traceability label such as `[US1]`
- Every task MUST name exact repository paths
- Contract changes MUST identify both producer and consumer tasks

## Repository Paths

- Electron orchestration and persistence: `main.js`
- Privileged renderer bridge: `preload.js`
- Game-master interface: `master/index.html`, `master/master.js`, `master/master.css`
- Server and domain logic: `server/server.js`, `server/hack.js`, `server/nav.js`, `server/wordbank.js`
- Optional public tunnel: `server/ngrok.js`
- Player interface: `client/index.html`, `client/client.js`, `client/sound.js`, `client/client.css`
- Persistent examples: `sessions/*.json`
- Package/build configuration: `package.json`, `package-lock.json`

<!--
The task generator MUST replace all examples below with feature-specific tasks.
Do not add database, authentication, API-framework, or generic src/ setup tasks
unless the feature actually introduces those changes.
-->

## Phase 1: Setup and Contract Baseline

**Purpose**: Confirm affected boundaries and establish contracts before implementation.

- [ ] T001 Review the feature's affected paths and current behavior in [exact paths]
- [ ] T002 Record changed session, IPC, HTTP, or WebSocket contracts in `specs/[###-feature]/contracts/[contract].md`
- [ ] T003 Define verification steps in `specs/[###-feature]/quickstart.md`
- [ ] T004 [P] Update dependency/build configuration in `package.json` and `package-lock.json` only if required by the approved plan

**Checkpoint**: Contracts, compatibility behavior, and verification approach are explicit.

---

## Phase 2: Foundational Runtime Work

**Purpose**: Implement shared behavior that blocks all user stories.

- [ ] T005 Implement pure domain/state behavior in `server/[module].js`
- [ ] T006 Implement transport validation and canonical-state changes in `server/server.js`
- [ ] T007 Implement native/session orchestration in `main.js` if required
- [ ] T008 Expose the smallest required privileged API in `preload.js` if required
- [ ] T009 Add or update focused automated tests in [planned test path] if the plan introduces or uses test tooling

**Checkpoint**: Shared runtime behavior and cross-boundary contracts are ready for user-story integration.

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Observable value delivered]

**Independent Test**: [Concrete standalone verification journey]

### Verification for User Story 1

- [ ] T010 [P] [US1] Add focused domain/contract tests in [planned test path], if configured
- [ ] T011 [US1] Document the manual Electron/browser scenario in `specs/[###-feature]/quickstart.md`

### Implementation for User Story 1

- [ ] T012 [P] [US1] Implement GM-facing changes in `master/[exact file]`
- [ ] T013 [P] [US1] Implement player-facing changes in `client/[exact file]`
- [ ] T014 [US1] Integrate IPC/WebSocket producers and consumers in [exact paths]
- [ ] T015 [US1] Update session defaults, validation, versioning, or `sessions/demo.json` if the persistent contract changes
- [ ] T016 [US1] Verify the independent journey with one master and [client count] player browsers

**Checkpoint**: User Story 1 works independently and all connected clients converge on expected state.

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Observable value delivered]

**Independent Test**: [Concrete standalone verification journey]

### Verification for User Story 2

- [ ] T017 [P] [US2] Add focused automated verification in [planned test path], if configured
- [ ] T018 [US2] Add the manual journey to `specs/[###-feature]/quickstart.md`

### Implementation for User Story 2

- [ ] T019 [P] [US2] Implement runtime/domain changes in `server/[exact file]`
- [ ] T020 [P] [US2] Implement presentation changes in `master/[exact file]` or `client/[exact file]`
- [ ] T021 [US2] Integrate and validate changed contracts in [producer path] and [consumer path]
- [ ] T022 [US2] Verify initial connection, multi-client behavior, and reconnection as applicable

**Checkpoint**: User Stories 1 and 2 remain independently functional.

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Observable value delivered]

**Independent Test**: [Concrete standalone verification journey]

### Implementation and Verification for User Story 3

- [ ] T023 [P] [US3] Implement isolated changes in [exact path]
- [ ] T024 [US3] Integrate cross-boundary behavior in [exact paths]
- [ ] T025 [US3] Add automated verification in [planned test path], if configured
- [ ] T026 [US3] Verify the documented independent journey

**Checkpoint**: All selected user stories are independently functional.

---

## Final Phase: Cross-Cutting Verification and Polish

- [ ] T027 [P] Review Electron sandbox, context isolation, CSP, and external URL handling in `main.js`, `preload.js`, and `master/index.html`
- [ ] T028 [P] Review WebSocket input validation, secret-state filtering, and reconnect synchronization in `server/` and `client/`
- [ ] T029 [P] Open and save existing compatible files from `sessions/` without data loss
- [ ] T030 Run all automated commands defined in `specs/[###-feature]/quickstart.md`
- [ ] T031 Run `npm start` and complete the documented master/player smoke journeys
- [ ] T032 Run `npm run build:dir` for packaging-sensitive changes when the required environment is available
- [ ] T033 Update `README.md` when setup, operation, environment variables, or user-visible workflows changed

---

## Dependencies and Execution Order

- Contract/setup tasks precede changes to contract producers and consumers.
- Foundational server, `main.js`, or `preload.js` work blocks dependent UI stories.
- Within a story, pure domain behavior precedes transport integration; producer and consumer changes precede end-to-end verification.
- Session migration/default logic precedes validation with older session files.
- User stories may proceed in parallel only after shared foundations are stable and their exact files do not overlap.
- Cross-cutting verification follows all selected user stories.

## Parallel Opportunities

- Independent `master/` and `client/` presentation work may run in parallel after their shared contract is fixed.
- Pure domain tests may run in parallel with isolated CSS/HTML work.
- Security, documentation, and session-fixture review may run in parallel when they touch different files.
- Tasks changing `server/server.js`, `main.js`, shared renderer state, or the same contract are not parallel merely because they have different story labels.

## Implementation Strategy

1. Deliver the smallest P1 vertical slice across every required boundary.
2. Verify it with the documented master/player journey.
3. Add P2 and P3 stories incrementally without breaking earlier journeys.
4. Finish with multi-client, reconnection, session compatibility, security, and packaging checks proportional to the change.

## Notes

- Do not claim test, lint, coverage, or CI success unless such tooling exists and ran.
- Record unavailable build environments or manual checks explicitly.
- Keep runtime-only live state out of session JSON unless persistence is an approved requirement.
- Avoid vague tasks, imaginary `src/` paths, generic database/authentication work, and producer-only contract changes.
