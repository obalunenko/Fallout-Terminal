# Tasks: Player Intelligence and Hacker Perk Management

**Input**: Design artifacts from `specs/011-player-intelligence-hacker/`

**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/player-config-v1.md`, `contracts/private-desktop.md`, `contracts/master-ui.md`

**Testing**: This change affects versioned protobuf, strict portable player-config JSON, atomic coordinator mutations, the private Wails bridge, and the master browser UI. Each user story begins with failing focused tests; the final phase is the single owner of full Success Criteria validation because no post-implement hook declares `owns: validation`.

**Organization**: Tasks are grouped by prioritized user story. `[P]` means independent work in different files within the same wave; `[US#]` maps work to the corresponding story in `spec.md`.

## Phase 1: Setup — additive contracts and generated baseline

**Purpose**: Establish compatible schema sources before changing any producer or consumer.

**Wave 1 — independent (different files):**

- [ ] **T001** [P] Add optional roster Intelligence field 3 and Hacker-perk-availability field 4 without changing player-config version or existing field numbers · `proto/fallout/terminal/persistence/v1/player_config.proto`
- [ ] **T002** [P] Add master-only roster projection fields plus complete add/update/delete inputs with Hacker presence and expected revision · `proto/fallout/terminal/private/v1/coordination.proto`, `proto/fallout/terminal/private/v1/desktop.proto`

**⟶ Wait for Wave 1 to finish, then:**

- [ ] **T003** Regenerate pinned Go protobuf output and the reviewed schema revision without editing generated files or the compatibility baseline manually · `internal/gen/fallout/terminal/persistence/v1/player_config.pb.go`, `internal/gen/fallout/terminal/private/v1/coordination.pb.go`, `internal/gen/fallout/terminal/private/v1/desktop.pb.go`, `proto/schema-revision.txt`

---

## Phase 2: Foundational — canonical profiles and conditional persistence

**Purpose**: Build the compatible canonical model, strict representation adapters, stale-file protection, and detached private projection that block every user story.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [ ] **T004** [P] Cover Intelligence 1/10 acceptance, explicit 0/11/fraction/string/null rejection, legacy missing-field defaults, canonical JSON emission, unknown/trailing rejection, stable IDs/order, and clone behavior · `internal/domain/model_test.go`, `internal/domain/validate_test.go`
- [ ] **T005** [P] Cover persistence-protobuf presence/default mapping, create/open/save/reopen, content-digest advance, missing/replaced/unreadable conflicts, and unchanged content after atomic failure · `internal/playerconfig/contract_test.go`, `internal/playerconfig/service_test.go`
- [ ] **T006** [P] Cover private CharacterState field numbers/values, detached mapping, and absence of Intelligence/Hacker data from public player descriptors and projections · `app_contract_test.go`, `internal/player/adapter_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [ ] **T007** Add canonical profile/digest fields, deep-copy propagation, strict presence-aware player-config decoding, canonical encoding, and Intelligence validation · `internal/domain/model.go`, `internal/domain/json.go`, `internal/domain/validate.go`

**⟶ Wait for T007 to finish, then Wave 3 — independent (different files):**

- [ ] **T008** [P] Map optional persistence fields and implement SHA-256 conditional complete-file saves while reusing the existing atomic storage replacement · `internal/playerconfig/contract.go`, `internal/playerconfig/service.go`
- [ ] **T009** [P] Propagate profile values and the refreshed content digest through active-config installation, clone-safe snapshots, roster candidates, and the persistence-before-publication seam · `internal/control/service.go`
- [ ] **T010** [P] Map Intelligence and Hacker availability through the private coordination bootstrap/result/event projection without changing public player state · `app_contract.go`

**Checkpoint**: Legacy and canonical player configs, strict validation, conditional atomic storage, and detached private profile projections are independently testable foundations.

---

## Phase 3: User Story 1 — add players with gameplay attributes (Priority: P1) 🎯 MVP

**Goal**: The game master adds one complete player profile from the dedicated dialog while no broadcast is active and sees the durably stored authoritative result.

**Independent Test**: Open an inactive session with a selected player config, add players using Intelligence boundaries and both Hacker choices, reopen the config, and verify exact values and validation failures.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [ ] **T011** [P] [US1] Add control/App tests for complete add payload validation, explicit Hacker false/presence, inactive/config/revision guards, duplicate retry, persistence failure, one revision/effect, and canonical reopen · `internal/control/service_test.go`, `app_test.go`, `app_contract_test.go`
- [ ] **T012** [P] [US1] Add failing browser/API tests for opening the dialog, required fields, Intelligence 1/10 and invalid inputs, both Hacker choices, exact expected-revision payload, authoritative result rendering, and reopen persistence · `tests/browser/player-management.spec.mjs`, `tests/browser/desktop-api.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then Wave 2 — independent (different files):**

- [ ] **T013** [P] [US1] Implement revision-conditional inactive-only AddCharacter candidate persistence, delayed ID allocation, digest refresh, and one authoritative commit/effect · `internal/control/service.go`
- [ ] **T014** [P] [US1] Add the player-management trigger, base modal/detail/add markup, required controls, accessible status/error regions, and responsive dialog styling · `frontend/src/index.html`, `frontend/src/master.css`
- [ ] **T015** [P] [US1] Add deterministic dialog roster state, add mutation, reload, invalid-input, and failure support for browser journeys · `tests/browser/fixtures/desktop-bindings.js`, `tests/browser/fixture-server/main.go`

**⟶ Wait for Wave 2 to finish, then Wave 3 — independent (different files):**

- [ ] **T016** [P] [US1] Route the structured create payload through the private protobuf adapter, validate it at App, and expose the typed AddCharacter Wails method · `app.go`, `app_contract.go`, `desktop_service.go`
- [ ] **T017** [P] [US1] Implement the typed desktop facade and authoritative dialog add/render flow with no optimistic roster mutation · `frontend/src/desktop-api.js`, `frontend/src/master.js`

**⟶ Wait for Wave 3 to finish, then:**

- [ ] **T018** [US1] Regenerate the Wails bindings, update the exact binding inventory, and run the focused Go/frontend/browser tests until the complete add journey passes without generated drift · `frontend/bindings/github.com/obalunenko/Fallout-Terminal/desktopservice.js`, `frontend/bindings/github.com/obalunenko/Fallout-Terminal/models.js`, `frontend/bindings/github.com/obalunenko/Fallout-Terminal/internal/domain/models.js`, `scripts/wails-bindings-check.sh`

**Checkpoint**: User Story 1 independently adds and reloads complete valid profiles from the popup, while every invalid or failed add preserves the stored roster.

---

## Phase 4: User Story 2 — edit the inactive player roster (Priority: P1)

**Goal**: The game master atomically updates or deletes stored players while inactive; active, stale, duplicate, and storage-conflict requests are rejected and the dialog becomes read-only during broadcasts.

**Independent Test**: Update all editable attributes, delete another player, reload, then start a broadcast while the dialog is open and prove UI and crafted mutations cannot change the roster.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [ ] **T019** [P] [US2] Add control/App tests for full-profile update, stable identity/order, delete, no-op behavior, expected-revision replay, active-broadcast rejection, stale file digest, atomic-save failure, and authoritative error state · `internal/control/service_test.go`, `app_test.go`, `app_contract_test.go`
- [ ] **T020** [P] [US2] Extend browser/API tests for update/delete confirmation and cancellation, explicit Hacker false, storage/stale errors, live event to read-only, crafted active mutation refusal, and unchanged detailed values · `tests/browser/player-management.spec.mjs`, `tests/browser/desktop-api.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then Wave 2 — independent (different files):**

- [ ] **T021** [P] [US2] Replace name-only correction with full-profile UpdateCharacter and enforce revision/config/broadcast guards plus conditional persistence for update and delete · `internal/control/service.go`
- [ ] **T022** [P] [US2] Add row edit/delete-confirmation/read-only markup and styles, typed update/delete facade calls, authoritative failure rendering, and immediate event-driven mutation disabling · `frontend/src/index.html`, `frontend/src/master.css`, `frontend/src/desktop-api.js`, `frontend/src/master.js`
- [ ] **T023** [P] [US2] Extend deterministic fixture mutations for update/delete, fail-next save, stale revision, external-file conflict, and broadcast-start-while-open scenarios · `tests/browser/fixtures/desktop-bindings.js`, `tests/browser/fixture-server/main.go`

**⟶ Wait for Wave 2 to finish, then:**

- [ ] **T024** [US2] Route complete update/delete payloads, replace the exposed RenameCharacter Wails path with UpdateCharacter, preserve safe authoritative failure results, and remove the live name-only path · `app.go`, `app_contract.go`, `desktop_service.go`

**⟶ Wait for T024 to finish, then:**

- [ ] **T025** [US2] Regenerate Wails bindings, update the exact allowlist, and run focused race/frontend/browser tests until update, delete, storage conflict, stale retry, and active-broadcast refusal pass without drift · `frontend/bindings/github.com/obalunenko/Fallout-Terminal/desktopservice.js`, `frontend/bindings/github.com/obalunenko/Fallout-Terminal/models.js`, `scripts/wails-bindings-check.sh`

**Checkpoint**: User Story 2 independently edits and removes inactive profiles, preserves identity/order and atomic storage, and renders the same roster read-only throughout an active broadcast.

---

## Phase 5: User Story 3 — review details in a dedicated window (Priority: P2)

**Goal**: The dedicated popup fully replaces crowded inline durable editing, presents populated and empty rosters accessibly, closes without mutation, and preserves existing live assignment correction elsewhere.

**Independent Test**: Open and close populated and empty dialogs by mouse and keyboard, verify focus restoration and zero mutation calls, and exercise existing assignment/transfer controls after inline roster removal.

### Tests

**Wave 1 — write these tests to fail first:**

- [ ] **T026** [US3] Complete Playwright coverage for detailed populated/empty presentation, accessible labels, Escape/close-without-call, focus restoration, responsive scrolling, read-only detail visibility, and retained logical-session assignment/transfer controls · `tests/browser/player-management.spec.mjs`

### Implementation

**⟶ Wait for T026 to finish, then:**

- [ ] **T027** [US3] Remove the inline durable-roster editor, finalize modal/empty-state/delete dialog semantics and responsive layout, and keep configuration status plus the popup trigger on the master screen · `frontend/src/index.html`, `frontend/src/master.css`

**⟶ Wait for T027 to finish, then:**

- [ ] **T028** [US3] Finalize open/close/Escape/focus lifecycle, no-submit behavior, empty/detail rendering, and relocation of claimed-character transfer controls into logical-session management · `frontend/src/master.js`

**Checkpoint**: User Story 3 independently provides a focused, accessible detailed window without changing state on close or regressing live assignment correction.

---

## Final Phase: Polish and Success-Criteria validation

**Purpose**: Complete examples, prove contract separation and generated integrity, then validate every measurable outcome once.

**Wave 1:**

- [ ] **T029** Update the bundled player-config example with explicit attributes and cover its packaged-asset decode while retaining public-player non-disclosure assertions · `sessions/demo-players.json`, `internal/platform/assets_test.go`, `internal/player/adapter_test.go`

**⟶ Wait for T029 to finish, then:**

- [ ] **T030** Run the single complete automated Success Criteria gate—protobuf format/lint/generation/breaking, Wails binding drift, formatting, vet, Go and race tests, frontend build, focused/full Playwright, owned build—and record exact PASS/FAIL/NOT RUN evidence for SC-001–SC-006 · `specs/011-player-intelligence-hacker/validation.md`

**⟶ Wait for T030 to finish, then:**

- [ ] **T031** Run `go run ./cmd/build dev` for the native master add/update/delete/reopen/read-only journey, verify the under-60-second add outcome and rollback caveat, and append honest interactive evidence or NOT RUN reasons · `specs/011-player-intelligence-hacker/validation.md`

---

## Dependencies & Execution Order

- Phase order: Setup → Foundational → US1 (MVP) → US2 → US3 → Polish. Each later story builds on the previous complete vertical slice; Polish begins only after every story checkpoint passes.
- Phase 1: schema Wave 1 (`T001–T002`) → generated join `T003`.
- Phase 2: failing test Wave 1 (`T004–T006`) → canonical model `T007` → implementation Wave 3 (`T008–T010`).
- Phase 3: failing test Wave 1 (`T011–T012`) → backend/UI/fixture Wave 2 (`T013–T015`) → bridge/integration Wave 3 (`T016–T017`) → generated/focused join `T018`.
- Phase 4: failing test Wave 1 (`T019–T020`) → coordinator/UI/fixture Wave 2 (`T021–T023`) → private bridge `T024` → generated/focused join `T025`.
- Phase 5: failing browser task `T026` → structural cutover `T027` → dialog and assignment integration `T028`.
- Polish: examples/separation `T029` → single automated validation owner `T030` → interactive acceptance `T031`.
- Parallel opportunities are limited to tasks explicitly marked `[P]`; work revisiting `internal/control/service.go`, `app.go`, `app_contract.go`, `frontend/src/master.js`, generated Wails bindings, or `tests/browser/player-management.spec.mjs` remains ordered by phase and wave.
