# Tasks: Player Sessions, Character Assignment, and Shared Terminal Control

**Input**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, and `contracts/` in `specs/004-player-sessions-control/`

**Tests**: Required by the specification and constitution. Each user-story phase writes its focused tests before implementation; the final phase owns the complete Success Criteria suite run.

**Task format**: `T### [P?] [US#] Description · exact/file/path`

## Phase 1: Setup

Configure the existing browser-test project to discover the new player-session specification without adding a dependency.

**Wave 1 — setup:**

- [x] **T001** Broaden Playwright discovery from the single hacking file to all `*.spec.mjs` browser specifications while retaining one worker and the existing static player server · `tests/browser/playwright.config.mjs`

---

## Phase 2: Foundational — Shared Runtime and Protocol Infrastructure

This phase establishes the transport-independent types, strict wire shapes, sender-aware connection callback, coordinator transaction boundary, deterministic seams, and serialization guardrails that block every user story.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [x] **T002** [P] Prove runtime coordination projections detach deeply and that browser tokens, logical sessions, roster, claims, controller, revisions, switches, and puzzles never enter version-1 session JSON · `internal/domain/model_test.go`
- [x] **T003** [P] Specify deterministic opaque IDs, monotonic accepted-transition revisions, unchanged revisions on rejection, detached effects, and enqueue-before-unlock ordering for the new coordinator · `internal/control/service_test.go`
- [x] **T004** [P] Specify strict exact-field decoding and JSON envelopes for handshake, selection, request preconditions, personalized player state, action results, and revisioned terminal messages · `internal/player/protocol_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — independent (different files):**

- [ ] **T005** [P] Add transport-independent runtime IDs, roles, phases, roster/session/broadcast models, terminal slots, action results, master snapshots, and personalized player projections beside the unchanged durable models · `internal/domain/model.go`
- [ ] **T006** [P] Extend constants, typed messages, strict decoders, and secret-free revisioned envelope constructors to the frozen player WebSocket contract · `internal/player/protocol.go`
- [ ] **T007** [P] Carry the originating `PlayerConnection` into decoded-message callbacks while retaining one reader, one bounded writer queue, and slow-client isolation · `internal/player/client.go`
- [x] **T008** [P] Add deterministic opaque-ID, clock-free revision, and ordered-effect fakes used by coordinator and server concurrency tests · `internal/testutil/fakes.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — coordinator foundation:**

- [ ] **T009** Create `internal/control.Service` with one mutex, process runtime root, ID sources, monotonic revision, detached snapshot builders, bounded request-result records, and non-reentrant effect enqueueing · `internal/control/service.go`

**Checkpoint**: Foundational types, strict contracts, deterministic seams, and the one-owner transaction skeleton compile; no story behavior has been enabled yet.

---

## Phase 3: User Story 1 — Join as a Character (Priority: P1) 🎯 MVP

**Goal**: A new browser profile establishes one logical session, sees the game-master-defined roster, claims one available character, and becomes the first controller only when no controller exists.

**Independent Test**: Start a broadcast with two roster entries, connect one new browser profile, select one character, and verify the claim, personalized identity, active role, and current-terminal-or-waiting result.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T010** [P] [US1] Cover roster creation, fresh broadcast selection, 100 concurrent same-character claims, 100 concurrent different first assignments, one claim per session, one session per character, and exactly one initial controller · `internal/control/service_test.go`
- [ ] **T011** [P] [US1] Cover exact `SESSION_HELLO`, `SESSION_WELCOME`, `CHARACTER_SELECT`, `PLAYER_STATE`, and `ACTION_RESULT` shapes, roster privacy, stale broadcast rejection, and unknown-field rejection · `internal/player/protocol_test.go`
- [ ] **T012** [P] [US1] Cover first handshake, token issuance, fallback-name uniqueness, selection success/conflict, personalized roster refresh, and assigned terminal/waiting delivery over real sockets · `internal/player/server_test.go`
- [ ] **T013** [P] [US1] Cover validated `AddCharacter`, `StartBroadcast`, coordination-status replay, detached master events, and failure without partial state · `app_test.go`
- [ ] **T014** [P] [US1] Add browser tests for handshake gating, terminal-styled available/claimed selection, one pending selection, conflict recovery, escaped names, and progression after authoritative acceptance · `tests/browser/player-sessions-control.spec.mjs`
- [ ] **T015** [P] [US1] Assert selection markup/style contracts, hidden-state layout, text-safe rendering, no browser token in URLs/markup, and no player `ForceHackSuccess` path · `internal/platform/assets_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — independent (different files):**

- [ ] **T016** [P] [US1] Implement session creation/fallback names, process-local roster add, broadcast start, exclusive player selection, conflict effects, and atomic first-controller establishment · `internal/control/service.go`
- [ ] **T017** [P] [US1] Add semantic character-selection, player-identity, role, and assigned-waiting regions without exposing a privileged operation · `client/index.html`
- [ ] **T018** [P] [US1] Add responsive terminal-styled selection, claimed/available, identity, and waiting presentation within the existing bounded desktop/tablet layout · `client/client.css`
- [ ] **T019** [P] [US1] Add the minimal game-master roster and broadcast controls needed to define characters and start the MVP broadcast · `frontend/src/index.html`
- [ ] **T020** [P] [US1] Style the roster/broadcast controls within the existing desktop master layout and terminal aesthetic · `frontend/src/master.css`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent integrations (different files):**

- [ ] **T021** [P] [US1] Add coordinator interfaces, validated roster/start commands, `coordinationState` runtime status, and `coordination-state` event emission to the Wails facade · `app.go`
- [ ] **T022** [P] [US1] Require handshake before state/actions, bind connections to sessions, dispatch selection, send personalized state, and fan out the accepted claim to affected tabs · `internal/player/server.go`
- [ ] **T023** [P] [US1] Implement welcome-gated connection state, token storage, selection rendering/submission, authoritative player-state application, and character-primary identity · `client/client.js`
- [ ] **T024** [P] [US1] Add the exact roster/start methods plus `onCoordinationState` status-replay subscription to the narrow desktop facade · `frontend/src/desktop-api.js`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — independent composition (different files):**

- [ ] **T025** [P] [US1] Render the authoritative roster/broadcast snapshot, await add/start results, surface conflicts, and avoid optimistic master runtime state · `frontend/src/master.js`
- [ ] **T026** [P] [US1] Construct the coordinator with the existing live service, connect player/master effect sinks, and preserve lifecycle shutdown ownership · `main.go`

**Checkpoint**: User Story 1 is independently functional and testable: a game master can define a roster/start a broadcast, and concurrent players can establish exclusive identities and claims with one initial controller.

---

## Phase 4: User Story 2 — Control One Shared Terminal (Priority: P1)

**Goal**: Only the connected assigned controller can mutate the canonical terminal; observers mirror state with local-only feedback, and every request ends via a correlated authoritative result.

**Independent Test**: Connect one controller and two observers, exercise every navigation/hacking action from each, and verify only controller requests mutate state while all clients converge and every pending request resolves.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T027** [P] [US2] Cover controller/unassigned/observer/unknown/stale-terminal authorization, exact no-mutation and zero-RNG rejection, duplicate request fingerprints, request/reassignment ordering, and unchanged gameplay rules · `internal/control/service_test.go`
- [ ] **T028** [P] [US2] Cover 4–7 client convergence, initiating-socket action results, shared revision order, crafted observer rejection, duplicate one-use requests, and slow-client isolation · `internal/player/server_test.go`
- [ ] **T029** [P] [US2] Cover visibly read-only observers, local hover/focus/paging/preview, zero outbound observer actions, controller pending state, and accepted/rejected result completion without optimistic mutation · `tests/browser/player-sessions-control.spec.mjs`
- [ ] **T030** [P] [US2] Assert every pointer/keyboard/back/guess/pattern send path is role/pending gated while `ForceHackSuccess` and `HACK_ADMIN` remain absent from player assets/protocol · `internal/platform/assets_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — independent prerequisites (different files):**

- [ ] **T031** [P] [US2] Refactor terminal/navigation/hacking operations behind a coordinator-callable runtime boundary while retaining all current password, likeness, attempt, pattern, log, and forced-success semantics · `internal/live/service.go`
- [ ] **T032** [P] [US2] Add observer/read-only and shared-input-pending classes without suppressing harmless local feedback · `client/client.css`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent canonical/client behavior (different files):**

- [ ] **T033** [P] [US2] Implement request fingerprinting, assignment/controller/connected/terminal authorization, ordered live mutation, revisioned effects, and accepted/rejected cached action results · `internal/control/service.go`
- [ ] **T034** [P] [US2] Add request IDs and broadcast/terminal preconditions, gate all shared send paths by role/pending, apply revisioned snapshots, and clear pending only from authoritative results · `client/client.js`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — independent boundary integrations (different files):**

- [ ] **T035** [P] [US2] Dispatch sender-aware navigation/hacking commands through the coordinator, enqueue canonical fanout before per-request results, and reject all pre-handshake or unauthorized actions · `internal/player/server.go`
- [ ] **T036** [P] [US2] Route exact private `ForceHackSuccess` through the same ordered coordinator revision without granting any player capability · `app.go`
- [ ] **T037** [P] [US2] Update the existing hacking browser fixture for hello/welcome, request preconditions, revisions, and action results while preserving camouflage and no-optimistic-mutation assertions · `tests/browser/hacking-camouflage.spec.mjs`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5 — composition join:**

- [ ] **T038** [US2] Wire ordered terminal effects and the unchanged private hack-state event through the composed coordinator/player/master sinks · `main.go`

**Checkpoint**: User Story 2 is independently functional and testable: one controller drives unchanged canonical gameplay, observers remain local-only, and every shared request has an authoritative completion.

---

## Phase 5: User Story 3 — Reuse One Device Session (Priority: P1)

**Goal**: Refreshes, reopens, reconnects, and multiple tabs from one recognized browser profile reuse one process-local logical session and aggregate presence correctly.

**Independent Test**: Assign one browser profile, open at least three tabs, close/reopen them in varied order, and verify one logical identity/claim remains connected until the final tab closes.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T039** [P] [US3] Cover known/absent/unknown token attachment, unique replacement tokens, first/last connection presence transitions, stable fallback/claim/role, and no release on a new unrecognized session · `internal/control/service_test.go`
- [ ] **T040** [P] [US3] Cover three-tab membership, one-close continuity, final-close presence, refresh/reconnect snapshots, different tokens/profiles, and stale token replacement over real sockets · `internal/player/server_test.go`
- [ ] **T041** [P] [US3] Cover reload, reopen, three pages in one BrowserContext, another context, cleared storage, private-context-equivalent isolation, and first-use cross-tab handshake serialization · `tests/browser/player-sessions-control.spec.mjs`
- [ ] **T042** [P] [US3] Prove recognition tokens are absent from HTTP URLs/query endpoints/loggable asset paths and existing same-host Origin/CSP/nosniff behavior is unchanged · `internal/player/http_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — canonical membership:**

- [ ] **T043** [US3] Implement token lookup/replacement, connection-set attach/detach idempotency, aggregate presence effects, and stable logical-session reuse without claim release · `internal/control/service.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent transport/browser behavior (different files):**

- [ ] **T044** [P] [US3] Track connection-to-session association through close, emit only first/last presence changes, and resend current personalized/canonical snapshots on reconnect · `internal/player/server.go`
- [ ] **T045** [P] [US3] Serialize initial token issuance with Web Locks plus a storage fallback, reuse only the opaque token from `localStorage`, overwrite stale tokens, and keep the connection overlay until welcome · `client/client.js`

**Checkpoint**: User Story 3 is independently functional and testable: one recognized profile maps to one logical session across ordinary browser lifecycles and multiple tabs.

---

## Phase 6: User Story 4 — Manage Roster and Assignments (Priority: P1)

**Goal**: The game master can rename sessions, add/rename/delete roster entries, and assign/release/move claims without mutating terminal or puzzle state.

**Independent Test**: Exercise every roster and assignment correction, including claimed-delete refusal, and deep-compare canonical terminal/puzzle state before and after each operation.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T046** [P] [US4] Cover name validation/uniqueness, duplicate character names, rename stability, claimed-delete refusal, GM assign, release, move, inverse-index invariants, controller clearing, and byte-equivalent terminal/puzzle state · `internal/control/service_test.go`
- [ ] **T047** [P] [US4] Cover validated roster/session/assignment Wails payloads, conflict errors with unchanged snapshots, detached events, and no calls into durable session saving · `app_test.go`
- [ ] **T048** [P] [US4] Cover all-tab roster/assignment fanout, release-to-selection, rename propagation, transfer, claimed-delete refusal, and absence of claimant/connection data in player JSON · `internal/player/server_test.go`
- [ ] **T049** [P] [US4] Cover live roster rename, claimed/available updates, release back to selection, character-primary/fallback-secondary identity, and no canonical terminal change · `tests/browser/player-sessions-control.spec.mjs`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — independent canonical/markup/style work (different files):**

- [ ] **T050** [P] [US4] Implement character rename/delete, logical-session rename, GM assign/release/move, inverse claim indexes, controller-clear rules, and detached roster/player/master effects · `internal/control/service.go`
- [ ] **T051** [P] [US4] Add complete roster CRUD, logical-session labels, assignment/release/move controls, and accessible error/status regions to the coordination panel · `frontend/src/index.html`
- [ ] **T052** [P] [US4] Style roster/session rows, claim states, correction controls, and responsive overflow within the existing master layout · `frontend/src/master.css`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent boundary/client integrations (different files):**

- [ ] **T053** [P] [US4] Add validated `RenameCharacter`, `DeleteCharacter`, `RenameLogicalSession`, `AssignCharacter`, `ReleaseCharacter`, and `MoveCharacter` Wails methods · `app.go`
- [ ] **T054** [P] [US4] Add exact lower-camel roster/session/assignment commands to the frozen desktop facade · `frontend/src/desktop-api.js`
- [ ] **T055** [P] [US4] Route coordinator roster/assignment effects into complete per-session state for every affected tab without exposing private session data · `internal/player/server.go`
- [ ] **T056** [P] [US4] Apply roster/identity/assignment changes by stable IDs, return released sessions to selection, and retain all canonical terminal mirrors · `client/client.js`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — game-master integration:**

- [ ] **T057** [US4] Render and await all roster/session/claim corrections, show claimed-delete and conflict refusals, and retain authoritative terminal/puzzle UI state · `frontend/src/master.js`

**Checkpoint**: User Story 4 is independently functional and testable: the game master can repair names and claims without affecting gameplay state.

---

## Phase 7: User Story 5 — Reassign Terminal Control (Priority: P1)

**Goal**: The game master can atomically designate a connected assigned observer as controller, including during action races, without changing claims or gameplay state.

**Independent Test**: Reassign between two connected assigned sessions during navigation and hacking and verify ordered authority, all-tab role updates, and exact puzzle continuity.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T058** [P] [US5] Cover eligible/ineligible reassignment, 100 interleaved action-versus-reassignment trials, former-controller rejection, one controller invariant, and exact assignment/terminal/puzzle preservation · `internal/control/service_test.go`
- [ ] **T059** [P] [US5] Cover validated `SetActiveController`, connected-assigned eligibility, refusal snapshots, and ordered master event emission · `app_test.go`
- [ ] **T060** [P] [US5] Cover all-tab active/observer fanout and before/after reassignment action ordering over concurrent real sockets · `internal/player/server_test.go`
- [ ] **T061** [P] [US5] Cover live role swap, former-controller read-only transition, new-controller send eligibility, and unchanged displayed character/terminal/puzzle state · `tests/browser/player-sessions-control.spec.mjs`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — canonical reassignment:**

- [ ] **T062** [US5] Implement connected-assigned validation and atomic controller replacement in the same order as player actions while preserving claims and every terminal runtime field · `internal/control/service.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent boundary fanout (different files):**

- [ ] **T063** [P] [US5] Add the validated `SetActiveController` Wails method and updated detached coordination status · `app.go`
- [ ] **T064** [P] [US5] Add `setActiveController` to the narrow desktop facade · `frontend/src/desktop-api.js`
- [ ] **T065** [P] [US5] Fan one reassignment revision to every connection of the former and new controller sessions before later action results · `internal/player/server.go`
- [ ] **T066** [P] [US5] Apply authoritative active/observer changes across tabs and immediately gate shared sends without changing local feedback or character identity · `client/client.js`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — game-master integration:**

- [ ] **T067** [US5] Render eligible controller controls and active/observer status, await reassignment, and show refusal without optimistic role changes · `frontend/src/master.js`

**Checkpoint**: User Story 5 is independently functional and testable: controller changes are atomic, globally ordered, and gameplay-neutral.

---

## Phase 8: User Story 6 — Handle Controller Disconnects (Priority: P1)

**Goal**: Last-connection loss retains the controller and claim without promotion; reconnect restores control unless the game master reassigned it meanwhile.

**Independent Test**: Disconnect a multi-tab controller, reconnect before and after reassignment, and verify presence, role, claim, and puzzle continuity in player and master views.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T068** [P] [US6] Cover observer/controller last-close transitions, retained claim/controller, zero promotion, reconnect-before/after-reassignment roles, and byte-equivalent terminal/puzzle state · `internal/control/service_test.go`
- [ ] **T069** [P] [US6] Cover multi-tab controller close order, final disconnect, no observer promotion, reconnect snapshots, and reassigned former-controller observer status · `internal/player/server_test.go`
- [ ] **T070** [P] [US6] Cover master projection/event replay for a disconnected active session and unchanged claims/controller across transient connection changes · `app_test.go`
- [ ] **T071** [P] [US6] Cover connection overlay/reconnect for unchanged controller and reassigned former controller without selection or canonical-state reset · `tests/browser/player-sessions-control.spec.mjs`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — canonical disconnect semantics:**

- [ ] **T072** [US6] Preserve claim/controller on final detach, prohibit automatic promotion, and emit presence-only effects ordered with reassignments and actions · `internal/control/service.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent projections (different files):**

- [ ] **T073** [P] [US6] Publish last-close presence and reconnect role snapshots without clearing or regenerating terminal state · `internal/player/server.go`
- [ ] **T074** [P] [US6] Restore the welcomed authoritative role after reconnect and never infer promotion from connection loss · `client/client.js`
- [ ] **T075** [P] [US6] Keep disconnected sessions visible, mark a disconnected controller distinctly, and offer reassignment without changing its claim · `frontend/src/master.js`

**Checkpoint**: User Story 6 is independently functional and testable: network interruption never elects a controller or changes gameplay, and reconnect behavior follows explicit reassignment history.

---

## Phase 9: User Story 7 — Follow the Active Terminal (Priority: P1)

**Goal**: One broadcast-wide active terminal, or none, is authoritative; all assigned players follow switches while sessions, claims, and controller remain unchanged.

**Independent Test**: Switch among configured terminals at least ten times with active and observer sessions, include a no-terminal waiting interval and a late assignment, and verify convergence plus stale-terminal rejection.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T076** [P] [US7] Cover checkpointable terminal runtime creation/update, navigation revalidation, fresh puzzle generation, detached public state, and no reset on active content update · `internal/live/service_test.go`
- [ ] **T077** [P] [US7] Cover broadcast-wide active terminal/no-terminal state, direct completed-puzzle switches, session/claim/controller preservation, late-assignee current-terminal join, and stale/inactive terminal rejection · `internal/control/service_test.go`
- [ ] **T078** [P] [US7] Cover validated terminal activation/clear/update bridge results, authoritative status, and no optimistic live-terminal mutation · `app_test.go`
- [ ] **T079** [P] [US7] Cover ordered revisioned `TERMINAL_LIVE`/`TERMINAL_CLEAR` fanout, ten switches, late assignment, reconnect, and stale-terminal action results · `internal/player/server_test.go`
- [ ] **T080** [P] [US7] Cover terminal reveal transition, assigned waiting, ten automatic switches, late selection into the current terminal, and stable identity/character/role · `tests/browser/player-sessions-control.spec.mjs`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — live runtime boundary:**

- [ ] **T081** [US7] Add private terminal runtime create/update/project helpers that preserve existing navigation/hacking state and support coordinator-owned active/suspended slots · `internal/live/service.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — canonical terminal selection:**

- [ ] **T082** [US7] Implement direct active-terminal activation/clear, runtime-slot ownership, late-assignee snapshots, stale-terminal guards, and identity/claim/controller preservation · `internal/control/service.go`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — independent boundary/client integrations (different files):**

- [ ] **T083** [P] [US7] Replace set/clear conflation with validated `RequestTerminalActivation`, `RequestTerminalClear`, and ordered `UpdateLiveTerminal` Wails methods · `app.go`
- [ ] **T084** [P] [US7] Add request-terminal-activation/clear methods while retaining update-live and exact private forced-success methods · `frontend/src/desktop-api.js`
- [ ] **T085** [P] [US7] Emit revisioned terminal live/update/clear effects selectively to assigned sessions and reject actions for inactive terminals · `internal/player/server.go`
- [ ] **T086** [P] [US7] Apply active-terminal revisions through the existing reveal presentation, show assigned waiting when null, and retain identity/assignment/role through switches · `client/client.js`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5 — game-master integration:**

- [ ] **T087** [US7] Drive make-active and clear-active controls from coordinator status, await results, and distinguish no active terminal from broadcast end · `frontend/src/master.js`

**Checkpoint**: User Story 7 is independently functional and testable: every assigned player follows the single selected terminal or waiting state without losing broadcast identity or authority.

---

## Phase 10: User Story 8 — Decide an Unfinished Puzzle's Fate (Priority: P1)

**Goal**: Switching away from an unfinished puzzle requires preserve, discard, or cancel; preservation restores the exact private puzzle and inactive terminals reject actions.

**Independent Test**: Exercise all three decisions, switch back after preserve/discard, compare every puzzle field, and verify the source remains active while the decision is pending or cancelled.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T088** [P] [US8] Deep-compare secret word, generation, board, attempts, candidates, removed duds, patterns, log, outcome, and navigation across private suspend/reactivate; verify discard regenerates and content refresh revalidates navigation · `internal/live/service_test.go`
- [ ] **T089** [P] [US8] Cover decision-required detection, opaque switch IDs, preserve/discard/cancel, stale decision refusal, continued source actions while pending, active/preserved deletion guard, and inactive action rejection · `internal/control/service_test.go`
- [ ] **T090** [P] [US8] Cover switch result shapes, decision validation, stale refusal, detached status, and exact `ForceHackSuccess` eligibility while a decision is pending · `app_test.go`
- [ ] **T091** [P] [US8] Cover preserve/discard/cancel player fanout, no premature terminal clear, exact restored public puzzle, and inactive-terminal crafted-request rejection · `internal/player/server_test.go`
- [ ] **T092** [P] [US8] Cover unchanged source display while pending/cancelled, preserve restore, discard fresh puzzle, and no identity/assignment/role changes across the decision · `tests/browser/player-sessions-control.spec.mjs`
- [ ] **T093** [P] [US8] Assert accessible preserve/discard/cancel dialog markup, destructive-action emphasis, hidden-state behavior, and no player exposure of switch or private puzzle controls · `internal/platform/assets_test.go`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — exact live checkpoints:**

- [ ] **T094** [US8] Implement exact private suspend/reactivate/discard helpers, retain checkpoint generation level, apply latest name/tree/intro, and revalidate navigation without exposing private state · `internal/live/service.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — canonical switch transaction:**

- [ ] **T095** [US8] Implement pending switch tokens, preserve/discard/cancel resolution, stale/source/broadcast checks, deletion guard, and ordered source/target effects · `internal/control/service.go`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — independent game-master surfaces (different files):**

- [ ] **T096** [P] [US8] Add validated `ResolveTerminalSwitch` and switch-result payloads while keeping `ForceHackSuccess` exact and private · `app.go`
- [ ] **T097** [P] [US8] Add `resolveTerminalSwitch` and normalized switch-command results to the desktop facade · `frontend/src/desktop-api.js`
- [ ] **T098** [P] [US8] Add the blocking preserve/discard/cancel dialog and stale/error status region to the master document · `frontend/src/index.html`
- [ ] **T099** [P] [US8] Style the switch dialog, destructive discard option, focus states, and responsive bounds in the master aesthetic · `frontend/src/master.css`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5 — game-master integration:**

- [ ] **T100** [US8] Await decision-required activation/clear results, drive preserve/discard/cancel resolution, keep the source authoritative while pending, and guard active/preserved terminal deletion · `frontend/src/master.js`

**Checkpoint**: User Story 8 is independently functional and testable: no unfinished puzzle can be silently lost, altered, or acted on while inactive.

---

## Phase 11: User Story 9 — End and Restart Broadcast Lifetimes (Priority: P2)

**Goal**: Broadcast end clears claims/control/runtime terminals while retaining process sessions/roster; the next broadcast requires selection, and process restart restores no runtime identity or ownership.

**Independent Test**: End a populated broadcast, start another in the same process, then restart the application and verify each required lifetime boundary plus unchanged durable terminal data.

### Tests

**Wave 1 — independent (different files; write failing tests first):**

- [ ] **T101** [P] [US9] Cover end/start lifetime cleanup, retained sessions/fallback names/roster, cleared claims/controller/active/suspended/request caches, fresh broadcast ID, and fresh initial controller · `internal/control/service_test.go`
- [ ] **T102** [P] [US9] Compare version-1 encoding before/after complete runtime activity and prove no runtime coordination or puzzle field is persisted and durable unlocked-terminal behavior is unchanged · `internal/domain/model_test.go`
- [ ] **T103** [P] [US9] Cover end-broadcast player clear/context, same-process reselection, stale old-broadcast requests, and unknown prior-process token replacement with no restored state · `internal/player/server_test.go`
- [ ] **T104** [P] [US9] Cover validated `EndBroadcast`, second `StartBroadcast`, shutdown cleanup, status replay, and no durable terminal deletion/mutation · `app_test.go`
- [ ] **T105** [P] [US9] Cover end-to-selection/waiting, second-broadcast reselection, stale pending completion, and fresh welcome after simulated process restart · `tests/browser/player-sessions-control.spec.mjs`

**⟶ Wait for Wave 1 to finish, then:**

### Implementation

**Wave 2 — canonical lifetime cleanup:**

- [ ] **T106** [US9] Implement atomic broadcast end/start cleanup and process shutdown disposal while retaining only process-scoped sessions/fallback names/roster between broadcasts · `internal/control/service.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent boundary/client integrations (different files):**

- [ ] **T107** [P] [US9] Add validated `EndBroadcast`, second-start behavior, shutdown coordinator cleanup, and unchanged durable-session bridge separation · `app.go`
- [ ] **T108** [P] [US9] Add `endBroadcast` and retained `startBroadcast` commands to the desktop facade · `frontend/src/desktop-api.js`
- [ ] **T109** [P] [US9] Publish terminal clear and fresh personalized contexts on end/start, reject stale broadcast IDs, and replace tokens unknown to a fresh process · `internal/player/server.go`
- [ ] **T110** [P] [US9] Clear pending/canonical mirrors on broadcast end, retain only server-confirmed identity display, and require new selection for every new broadcast · `client/client.js`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — game-master integration:**

- [ ] **T111** [US9] Drive broadcast end/start from authoritative coordination status, retain roster/session labels, clear live controls, and never mutate configured durable terminals · `frontend/src/master.js`

**Checkpoint**: User Story 9 is independently functional and testable: broadcast and process boundaries clear exactly the intended runtime state and preserve exactly the intended authored/process state.

---

## Phase 12: Polish and Cross-Cutting Validation

Finalize operational documentation, enforce security/presentation contracts across the complete surface, and run the Success Criteria suite exactly once here because no post-implement hook owns validation.

**Wave 1 — independent (different files):**

- [ ] **T112** [P] Document the game-master roster/assignment/controller/switch workflow, browser-profile recognition scope, broadcast lifetimes, and unchanged private `ForceHackSuccess` boundary · `README.md`
- [ ] **T113** [P] Consolidate cross-cutting asset assertions for restrictive CSP, escaped names, token secrecy, observer local-only behavior, pending resolution, responsive layouts, and absence of every player forced-success path · `internal/platform/assets_test.go`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — automated Success Criteria validation:**

- [ ] **T114** Run `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`, `npm --prefix frontend run build`, and `npm --prefix tests/browser test`; record each result and map SC-001 through SC-022 to evidence · `specs/004-player-sessions-control/validation.md`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — interactive/package gates:**

- [ ] **T115** Run affected `wails dev` master/player journeys and a clean macOS `wails build`, recording unavailable signing/notarization credentials rather than claiming those release-only gates · `specs/004-player-sessions-control/validation.md`

**Checkpoint**: All nine stories, cross-cutting boundaries, SC-001 through SC-022, browser interactions, concurrency/race behavior, and package-level smoke gates are verified or explicitly reported unavailable.

---

## Dependencies & Execution Order

### Phase dependencies

1. **Phase 1: Setup** enables discovery of new browser tests.
2. **Phase 2: Foundational** depends on Setup and blocks every story.
3. **Phase 3: US1** depends on Foundational and is the MVP identity/selection slice.
4. **Phase 4: US2** depends on US1 identities/roles and establishes authoritative shared-action control.
5. **Phase 5: US3** depends on US1 handshake/session state and hardens recognition and multi-tab presence.
6. **Phase 6: US4** depends on US1 claims and US2 ordered effects to add corrections safely.
7. **Phase 7: US5** depends on US2 authorization and US4 assignment operations.
8. **Phase 8: US6** depends on US3 presence and US5 reassignment semantics.
9. **Phase 9: US7** depends on US2 action preconditions and introduces broadcast-wide active-terminal state.
10. **Phase 10: US8** depends on US7 runtime slots/switching and adds unfinished-puzzle decisions.
11. **Phase 11: US9** depends on every process/broadcast/runtime aggregate that it clears.
12. **Phase 12: Polish** depends on all story checkpoints and owns the single complete validation run.

### Wave restatement

- **Setup**: Playwright discovery completes before any new browser specification is relied on.
- **Foundational**: failing tests → independent domain/protocol/connection/fake work → coordinator skeleton.
- **US1**: failing claim/handshake/UI tests → independent canonical/markup/style work → boundary integrations → master/composition join.
- **US2**: failing authorization/convergence/UI tests → live/CSS prerequisites → coordinator/client behavior → server/App/browser-fixture integrations → composition join.
- **US3**: failing membership/storage/security tests → canonical membership → server/browser reconnect behavior.
- **US4**: failing CRUD/correction tests → canonical/markup/style work → bridge/server/client integrations → master join.
- **US5**: failing reassignment/race tests → canonical reassignment → boundary fanout → master join.
- **US6**: failing disconnect/reconnect tests → canonical detach semantics → server/client/master projections.
- **US7**: failing live/control/bridge/fanout/browser tests → live runtime helpers → canonical terminal selection → boundary integrations → master join.
- **US8**: failing preservation/decision/UI tests → exact private checkpoints → canonical switch transaction → master boundary/markup/style → dialog integration.
- **US9**: failing lifetime/persistence/restart tests → canonical cleanup → boundary/client integrations → master join.
- **Polish**: independent documentation/security contracts → automated suites and SC evidence → interactive/package gates.

Tasks tagged `[P]` are parallel only inside their declared wave. Same-file work in later phases remains sequential even when conceptually independent, preserving one writer and the story checkpoint order.
