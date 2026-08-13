# Tasks: Protobuf-First ConnectRPC Migration

**Input**: Design documents in `specs/005-connectrpc-protobuf-migration/`
**Required**: `spec.md`, `plan.md`
**Supporting**: `research.md`, `data-model.md`, `contracts/`

**Bugfix**: 2026-08-13 — BUG-001 Updated from bugfix patch
**Bugfix**: 2026-08-13 — BUG-002 Reopened authenticated-ngrok streaming, soak, and validation work because local protected-forwarding plus unary-RPC evidence did not prove that a real public generated `Subscribe` delivered the first snapshot and terminal.

Tests are required by the specification and constitution. In every user-story phase, write the listed tests first and confirm they fail for the intended missing behavior before implementation. Go tests must use Testify, table-driven cases where flows repeat, `t.Context()` for test-scoped contexts, and `protocmp`/`prototest` for protobuf values and descriptors.

## Phase 1: Setup

**Purpose**: Freeze the migration ledger and install the pinned schema, generator, and browser-build prerequisites.

**Wave 1 — independent (different files):**

- [x] **T001** [P] Complete the application-owned public/private/persistence/configuration inventory with owner, producer, consumers, classification, schema/exclusion, and exposure, leaving zero unclassified rows · `specs/005-connectrpc-protobuf-migration/contracts/inventory.md`
- [x] **T002** [P] Add the Buf v2 module, lint policy, and public/private package boundary configuration · `proto/buf.yaml`
- [x] **T003** [P] Pin deterministic Go protobuf and Connect generators with isolated `internal/gen` output · `proto/buf.gen.go.yaml`
- [x] **T004** [P] Pin deterministic Protobuf-ES and Connect-ES generation for the public graph only · `proto/buf.gen.es.yaml`
- [x] **T005** [P] Add pinned protobuf, Connect-Go, generator/tool directives, Testify, and protobuf-aware test dependencies without removing Wails · `go.mod`
- [x] **T006** [P] Define the pinned Connect-ES, Protobuf-ES, and Vite scripts for offline player generation and bundling · `client/package.json`
- [x] **T007** [P] Extend one-command Wails development/build orchestration to build the generated player bundle before embedding · `wails.json`

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T008** Resolve and commit the exact player dependency graph without unpinned transitive updates · `client/package-lock.json`

---

## Phase 2: Foundational Contract and Boundary Infrastructure

**Purpose**: Define the versioned graphs and detached shared boundary types that block every user story.

**⚠️ CRITICAL**: No user-story implementation begins until this phase is complete.

**Wave 1 — independent schema files:**

- [x] **T009** [P] Define `PlayerService`, Subscribe, SelectCharacter, ActionResult, recognition, player-state, snapshot, compound-update, role, phase, roster, and revision contracts with required presence/`oneof` rules · `proto/fallout/terminal/player/v1/player.proto`
- [x] **T010** [P] Define recursive public content and exclusive live/no-live terminal presentation contracts without secret or private values · `proto/fallout/terminal/player/v1/terminal.proto`
- [x] **T011** [P] Define complete navigation projections and typed back/enter/command/entry request variants · `proto/fallout/terminal/player/v1/navigation.proto`
- [x] **T012** [P] Define bounded public hacking projections, word/filler guess variants, and opaque generation-bound pattern activation without `HACK_ADMIN` · `proto/fallout/terminal/player/v1/hacking.proto`
- [x] **T013** [P] Define the allowlisted SoundManifest unary category/result contract with `UNSPECIFIED` zero value · `proto/fallout/terminal/player/v1/sound.proto`
- [x] **T014** [P] Define every inventoried Wails request/result/session-operation/player-config-operation/command/terminal-switch semantic value · `proto/fallout/terminal/private/v1/desktop.proto`
- [x] **T015** [P] Define detached roster, logical-session coordination, broadcast, terminal-switch, and game-master state semantics · `proto/fallout/terminal/private/v1/coordination.proto`
- [x] **T016** [P] Define runtime-status, server-information, client-count, hack-state, coordination-state, startup, and save event/status semantics · `proto/fallout/terminal/private/v1/runtime.proto`
- [x] **T017** [P] Define all known session-v1, terminal, recursive content-node, and association semantics without replacing the JSON representation · `proto/fallout/terminal/persistence/v1/session.proto`
- [x] **T018** [P] Define all known strict player-config-v1 and roster semantics without permitting unknown JSON fields · `proto/fallout/terminal/persistence/v1/player_config.proto`
- [x] **T019** [P] Define serializable application/player-server/coordination/queue/timeout/path/tunnel/startup/shutdown values while excluding credentials and injected dependencies · `proto/fallout/terminal/config/v1/config.proto`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent generated outputs and canonical detached types:**

- [x] **T020** [P] Generate and commit deterministic Go values and Connect handlers for all four schema graphs; never hand-edit generated output · `internal/gen/fallout/terminal/player/v1/player.pb.go`
- [x] **T021** [P] Generate and commit deterministic ECMAScript values and the PlayerService client from only the public graph · `client/gen/fallout/terminal/player/v1/player_pb.js`
- [x] **T022** [P] Add transport-independent recognition, physical-stream, snapshot, compound-update, replay, action-result, pending, presentation, and manifest value types without making protobuf canonical · `internal/domain/model.go`
- [x] **T023** [P] Add finite validation rules for public identifiers and action targets while retaining existing persistence validation · `internal/domain/validate.go`
- [x] **T024** [P] Add deterministic protobuf-aware fakes/spies for handles, requests, streams, revisions, clocks, randomness, and canonical-service invocation · `internal/testutil/fakes.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent boundary foundations:**

- [x] **T025** [P] Establish coordinator-owned recognition, logical-session, physical-stream, revision, replay, and personalized-publication state without transport coupling · `internal/control/service.go`
- [x] **T026** [P] Add exhaustive generated-public ↔ transport-independent request/projection adapters with no duplicated public DTOs or generic maps · `internal/player/adapter.go`
- [x] **T027** [P] Centralize the 4 KiB message, 8 KiB encoded body, decompression, identifier, target, and sound-category bounds · `internal/player/limits.go`

---

## Phase 3: User Story 1 — First-Time Player Joins and Selects (P1) 🎯 MVP

**Goal**: A clean browser receives one complete generated snapshot, persists only its opaque handle, and selects an available character through a typed unary call.

**Independent Test**: Start a fresh process with an active broadcast and roster, subscribe from a clean profile, assert one complete first snapshot, select a character, and observe the correlated result plus authoritative assignment update.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T028** [P] [US1] Add failing Connect tests for absent/invalid/unknown Subscribe handles, exactly-one first snapshot, typed SelectCharacter, error codes, same-origin routing, and zero side effects on boundary rejection · `internal/player/handler_test.go`
- [x] **T029** [P] [US1] Add failing coordinator tests for exactly-one new logical session, complete personalized snapshot, unassigned selection eligibility, one mutation/revision/update, and no unauthorized pre-assignment shared action · `internal/control/service_test.go`
- [x] **T030** [P] [US1] Add a failing clean-profile browser journey for generated Subscribe, handle-only storage, roster render, selection result, and authoritative assignment · `tests/browser/connectrpc-player.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent core pieces:**

- [x] **T031** [P] [US1] Implement absent/known/unknown recognition resolution, atomic first attachment, complete snapshot capture, and selection transaction semantics · `internal/control/service.go`
- [x] **T032** [P] [US1] Implement one bounded physical-stream queue whose first application value is reserved for the complete snapshot · `internal/player/stream.go`
- [x] **T033** [P] [US1] Replace first-time browser transport with generated binary Connect Subscribe/SelectCharacter calls and handle-only storage · `client/client.js`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T034** [US1] Implement generated PlayerService Subscribe and SelectCharacter handlers with structural/semantic error mapping and typed authoritative results · `internal/player/handler.go`

**⟶ Wait for T034 to finish, then:**

- [x] **T035** [US1] Mount generated Connect paths beside ordinary same-origin player/static assets while temporarily retaining branch-only parity coexistence · `internal/player/http.go`

**⟶ Wait for T035 to finish, then:**

- [x] **T036** [US1] Compose the generated handler, coordinator, static assets, and effect routing into application startup · `main.go`

**Checkpoint**: User Story 1 is independently functional and testable as the thin local vertical slice.

---

## Phase 4: User Story 2 — Reconnect Without Resetting Gameplay (P1)

**Goal**: A recognized player reconnects to current canonical assignment, terminal, navigation, and hacking state without puzzle generation or replayed cues.

**Independent Test**: Disconnect and reconnect during an active changed puzzle, compare the new snapshot with canonical state, and assert zero puzzle-generation and transition-cue increments.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T037** [P] [US2] Add failing snapshot/reconnect tests proving current complete projections, explicit no-live-terminal, and zero puzzle generation · `internal/live/service_test.go`
- [x] **T038** [P] [US2] Extend the browser journey with known-handle reattach, unknown-handle replacement, three-second reconnect, latest-state recovery, and no replayed cues/actions · `tests/browser/connectrpc-player.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent projection/client behavior:**

- [x] **T039** [P] [US2] Expose detached complete current terminal/navigation/hacking projections without invoking puzzle generation · `internal/live/service.go`
- [x] **T040** [P] [US2] Implement generated snapshot replacement/reattach application and exact three-second reconnect behavior without optimistic canonical state · `client/client.js`
- [x] **T041** [P] [US2] Suppress ambient/outcome cues for reconnect snapshots, stale revisions, rejected actions, exact replays, and rerenders · `client/sound.js`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T042** [US2] Build recognized/replacement reconnect snapshots from current coordinator and live projections without restoring expired authority · `internal/control/service.go`

**Checkpoint**: User Story 2 is independently functional and testable through disconnect/reconnect and refresh.

---

## Phase 5: User Story 3 — Multiple Tabs Share One Logical Session (P1)

**Goal**: Concurrent tabs converge on one handle/session, receive equivalent revisions, and maintain aggregate presence until the final stream closes.

**Independent Test**: Open clean tabs concurrently, verify handle/session convergence and raw stream counts, then close them one at a time and observe final-detach presence semantics.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T043** [P] [US3] Add failing concurrent attach/detach tests for one logical session, equivalent per-stream updates, raw stream count, aggregate presence, retained claim, and retained controller identity · `internal/control/service_test.go`
- [x] **T044** [P] [US3] Add failing Web Locks/storage-lease multi-tab races and sibling-stream disconnect journeys · `tests/browser/connectrpc-player.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent server/browser convergence:**

- [x] **T045** [P] [US3] Implement first-attach/final-detach aggregate presence and equivalent fan-out to all responsive physical streams for one logical session · `internal/control/service.go`
- [x] **T046** [P] [US3] Hold first-tab election until the first snapshot handle is stored so concurrent clean tabs converge on one accepted handle · `client/client.js`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T047** [US3] Report client count from active public streams rather than logical sessions and keep event delivery race-free · `internal/player/server.go`

**Checkpoint**: User Story 3 is independently functional and testable with concurrent tabs and staged disconnects.

---

## Phase 6: User Story 4 — Players Converge Through Typed Actions (P1)

**Goal**: Selection, navigation, word/filler guesses, and pattern activation commit once and converge through one personalized compound update per affected logical session.

**Independent Test**: Drive four to seven streams through at least 25 mixed actions, replays, rejections, and reconnects; compare every final personalized projection with canonical state.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T048** [P] [US4] Add failing coordinator tests for one canonical mutation/revision/logical update, complete changed components, personalized projections, strictly increasing relevant revisions, and irrelevant revision skips · `internal/control/service_test.go`
- [x] **T049** [P] [US4] Add failing generated-handler tests for separate Navigate, Guess, ActivatePattern, and typed result procedures with no generic command/dispatcher · `internal/player/handler_test.go`
- [x] **T050** [P] [US4] Add failing mixed multi-player convergence and result-first/update-first pending-action journeys · `tests/browser/connectrpc-player.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent transaction, adapter, and browser pieces:**

- [x] **T051** [P] [US4] Commit each accepted action once and build one complete personalized compound update per affected session before returning · `internal/control/service.go`
- [x] **T052** [P] [US4] Map every typed navigation/guess/pattern request and complete public projection exhaustively between generated and domain values · `internal/player/adapter.go`
- [x] **T053** [P] [US4] Apply snapshots/compound updates authoritatively and reconcile accepted pending actions only after result plus applicable stream revision in either order · `client/client.js`
- [x] **T054** [P] [US4] Trigger ambient and hacking cues once from newly applied authoritative transitions only · `client/sound.js`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T055** [US4] Implement separate generated Navigate, Guess, and ActivatePattern unary handlers and return only stable ActionResult reasons · `internal/player/handler.go`

**⟶ Wait for T055 to finish, then:**

- [x] **T112** [US4] Replace the legacy fixture server protocol with production-shaped generated Connect handlers and static assets · `tests/browser/fixture-server/main.go`

**⟶ Wait for T112 to finish, then:**

- [x] **T056** [US4] Migrate the existing feature-003 and feature-004 parity journeys to generated typed actions and compound updates · `tests/browser/player-sessions-control.spec.mjs`

**Checkpoint**: User Story 4 is independently functional and testable across mixed multiplayer actions.

---

## Phase 7: User Story 5 — Exact Selection and Controller Authority (P1)

**Goal**: Selection remains available to a connected unassigned session, while all shared terminal mutations enforce active-controller and current-identity authority with effect-free rejection.

**Independent Test**: Exercise selection and shared actions across active, observer, unassigned, disconnected, stale, unknown, and malformed contexts while comparing every effect counter.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T057** [P] [US5] Add table-driven failing authority tests for selection eligibility, controller establishment, observer/unassigned/non-controller/disconnected/stale rejection, and zero state/revision/replay/random effects · `internal/control/service_test.go`
- [x] **T058** [P] [US5] Add failing transport tests for malformed/blank/oversized/unknown handles, illegal variants, required `UNSPECIFIED`, cancellation, unavailable, internal, and safe redacted errors · `internal/player/handler_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent validation and authority:**

- [x] **T059** [P] [US5] Implement bounded handle/request/broadcast/terminal/generation/target validation that distinguishes structural Connect errors from typed domain rejection · `internal/domain/validate.go`
- [x] **T060** [P] [US5] Order selection and shared-action authorization so only accepted actions mutate, advance revision, publish, write replay state, consume attempts, or consume randomness · `internal/control/service.go`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T061** [US5] Enforce structural decoding before adapters, map domain reasons exactly, and redact public errors without exposing private values · `internal/player/handler.go`

**⟶ Wait for T061 to finish, then:**

- [x] **T062** [US5] Add browser assertions that rejected actions clear pending state immediately and never optimistically alter canonical views · `tests/browser/connectrpc-player.spec.mjs`

**Checkpoint**: User Story 5 is independently functional and testable with the full authority/error matrix.

---

## Phase 8: User Story 6 — Safe Bounded Request Replay (P1)

**Goal**: Exact retained requests replay their original result without a second effect; changed fingerprints return duplicate; evicted identities are evaluated anew.

**Independent Test**: Replay accepted requests, change procedure/payload under one ID, exceed the 256-record bound, clear/restart, and inspect results and canonical effects.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T063** [P] [US6] Add failing table-driven and stress tests for exact replay, changed procedure/payload duplicate, per-session/per-broadcast bound 256, deterministic eviction, clear/removal/restart, and zero second effects · `internal/control/service_test.go`
- [x] **T064** [P] [US6] Add failing browser retries for retained results, lost unary responses, duplicate identities, and post-eviction no-guarantee behavior · `tests/browser/connectrpc-player.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — deterministic fingerprint input:**

- [x] **T065** [US6] Produce procedure-qualified deterministic protobuf request fingerprints after structural validation and before canonical mutation · `internal/player/adapter.go`

**⟶ Wait for T065 to finish, then:**

- [x] **T066** [US6] Implement bounded replay lookup/store/duplicate/eviction semantics scoped by logical session and broadcast · `internal/control/service.go`

**Checkpoint**: User Story 6 is independently functional and testable across retained and evicted request identities.

---

## Phase 9: User Story 7 — Concurrent Pattern Activation Uses Randomness Once (P1)

**Goal**: Exactly one contender activates a current unused pattern and only that winner consumes the unchanged feature-003 random sequence.

**Independent Test**: Run at least 100 deterministic races and assert one acceptance/revision/board effect, one outcome draw, at most one dud-selection draw, and zero losing draws.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T067** [P] [US7] Add 100 failing coordinator races covering duplicate/stale/used/losing pattern contenders and exact mutation/revision/random call counts · `internal/control/service_test.go`
- [x] **T068** [P] [US7] Add deterministic regression tests for unchanged pattern probabilities, dud removal, attempt reset, board generation, and one-plus-optional-one winning draws · `internal/live/service_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T069** [US7] Expose a single accepted pattern transition that consumes its outcome and optional dud-selection draws only after current-generation eligibility · `internal/live/service.go`

**⟶ Wait for T069 to finish, then:**

- [x] **T070** [US7] Serialize pattern validation, replay, canonical commit, revision, publication, and randomness ownership so exactly one contender wins · `internal/control/service.go`

**Checkpoint**: User Story 7 is independently functional and testable under the required race count.

---

## Phase 10: User Story 8 — Slow Subscribers Are Isolated (P1)

**Goal**: Blocked, overflowing, canceled, or disconnected streams cannot block mutations, responsive streams, detachment, or idempotent shutdown.

**Independent Test**: Block one subscriber, overflow its queue, mutate with a healthy subscriber, and shut down under five seconds.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T071** [P] [US8] Add deterministic failing tests for queue size 32, nonblocking offers, isolated overflow/cancellation, no retry of old increments, sibling continuity, and snapshot-only recovery · `internal/player/stream_test.go`
- [x] **T072** [P] [US8] Add failing active/blocked/canceled stream shutdown tests with resource accounting and a five-second deadline · `internal/player/server_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent publication and physical-stream lifecycle:**

- [x] **T073** [P] [US8] Make per-session compound-update offers nonblocking and detach only overflowing physical sinks without retrying old increments · `internal/control/service.go`
- [x] **T074** [P] [US8] Implement bounded queue delivery, send cancellation, overflow termination, and resource release for one physical subscription · `internal/player/stream.go`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T075** [US8] Make listener/stream/tunnel worker shutdown ordered, bounded, idempotent, and independent of blocked sends · `internal/player/server.go`

**Checkpoint**: User Story 8 is independently functional and testable with a deterministic unhealthy subscriber.

---

## Phase 11: User Story 9 — Preserve the Private Desktop Experience (P1)

**Goal**: Every named Wails method/event retains exact native compatibility while private protobuf contracts exhaustively govern semantics and remain unreachable publicly.

**Independent Test**: Exercise the complete method/event inventory against baseline shapes, fail on unmapped private fields, and enumerate public capabilities for private leaks.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T076** [P] [US9] Add descriptor-driven failing tests that enumerate every private message field/enum/`oneof` and require an explicit Wails/domain adapter mapping using protobuf-aware comparison · `app_contract_test.go`
- [x] **T077** [P] [US9] Add failing compatibility tests for every bound App method, result shape, lifecycle boundary, and four exact native event payloads · `app_test.go`
- [x] **T078** [P] [US9] Add failing public descriptor/route tests proving no desktop, native file/dialog/URL, ForceHackSuccess/reset, runtime status, server information, credential, private hacking, or other-session capability · `internal/player/handler_test.go`
- [x] **T079** [P] [US9] Add player-bundle scans for private imports and Wails protobuf-binary/Base64/ProtoJSON/generic-dispatch carriers · `internal/platform/assets_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T080** [US9] Implement exhaustive compatibility DTO ↔ private protobuf ↔ transport-independent adapters for all inventoried requests, results, statuses, and events · `app_contract.go`

**⟶ Wait for T080 to finish, then:**

- [x] **T081** [US9] Route unchanged named Wails methods and events through explicit private adapters while retaining native structured-object transport and eligibility · `app.go`

**Checkpoint**: User Story 9 is independently functional and testable through the complete trusted desktop matrix.

---

## Phase 12: User Story 10 — Preserve Existing Session and Player-Config Files (P1)

**Goal**: Existing version-1 JSON files retain exact names, validation, extras behavior, reference normalization, ordered saves, and atomic publication ordering.

**Independent Test**: Round-trip representative/adversarial fixtures and inject atomic-write failures while checking durable/runtime separation and publication order.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T082** [P] [US10] Add protobuf-aware adapter and fixture tests for every known session-v1 field plus recursive compatible unknown-field preservation and normalized references · `internal/session/contract_test.go`
- [x] **T083** [P] [US10] Add adapter/fixture tests for every known player-config-v1 field and all strict unknown/trailing/version/identity/duplicate failures · `internal/playerconfig/contract_test.go`
- [x] **T084** [P] [US10] Add failure-injection tests proving complete-file atomic save precedes roster/association/claim/controller/terminal/puzzle publication · `internal/playerconfig/service_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent persistence adapters:**

- [x] **T085** [P] [US10] Map all known session semantics through generated persistence values while leaving JSON-v1 codecs, field names, defaults, selected path, extras, and atomic replace authoritative · `internal/session/contract.go`
- [x] **T086** [P] [US10] Map all known player-config semantics through generated persistence values while retaining strict JSON-v1 decoding and normalized relative references · `internal/playerconfig/contract.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent service integration:**

- [x] **T087** [P] [US10] Integrate session contract verification without persisting recognition, live, assignment, controller, revision, pending, or replay runtime state · `internal/session/service.go`
- [x] **T088** [P] [US10] Gate player-config association publication on successful complete-file atomic save through the explicit contract adapter · `internal/playerconfig/service.go`

**Checkpoint**: User Story 10 is independently functional and testable with existing user-owned files.

---

## Phase 13: User Story 11 — Reproducible Contract Evolution (P2)

**Goal**: Developers generate both languages from one revision and receive deterministic format/lint/drift/breaking/dependency/adapter feedback.

**Independent Test**: Generate twice, compare a clean tree, inject representative breaking edits/private imports/unmapped fields, and confirm each verifier fails.

### Tests

**Wave 1 — independent failing verification surfaces:**

- [x] **T089** [P] [US11] Add descriptor/import/revision tests for exact service shape, public/private separation, reserved identifiers, `UNSPECIFIED`, semantic `optional`, and required `oneof` use · `internal/platform/assets_test.go`
- [x] **T090** [P] [US11] Add negative compatibility fixtures for field-number/type, enum, package, and service breaks against the established baseline · `internal/testutil/testdata/protobuf/breaking-fixtures.json`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent tooling commands:**

- [x] **T091** [P] [US11] ⚠️ Reopened (reopened — BUG-001): ~~Add a pinned clean generation command that writes only isolated checked-in Go/ECMAScript trees and verifies a shared schema revision.~~ Restore both language generations to the pinned `buf generate` entry point using `proto/buf.gen.go.yaml` and `proto/buf.gen.es.yaml`, with no direct `protoc` invocation · `scripts/proto-generate.sh`
- [x] **T092** [P] [US11] ⚠️ Reopened (reopened — BUG-001): ~~Add Buf format/lint, two-pass zero-diff generation, generated-header, descriptor import, bundle graph, and adapter exhaustiveness checks.~~ Retain those checks while accepting Buf-generated `protoc unknown` headers and validating generated markers, plugin pins, schema revision, deterministic hashes, output isolation, and absence of a standalone compiler path · `scripts/proto-check.sh`
- [x] **T093** [P] [US11] Add breaking-change checks against the committed compatibility baseline with representative negative-edit support · `scripts/proto-breaking.sh`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T094** [US11] ⚠️ Reopened (reopened — BUG-001): ~~Enforce schema checks, deterministic generation, compilation, tests, race checks, browser builds, and package gates in CI.~~ Preserve those gates while removing standalone `protoc` installation/version checks and enforcing generation exclusively through the pinned Buf CLI and checked-in Buf v2 templates · `.github/workflows/wails-macos.yml`

**Checkpoint**: User Story 11 is independently functional and testable from a clean checkout.

---

## Phase 14: User Story 12 — Preserve Local, Protected Public, and Packaged Use (P2)

**Goal**: Generated same-origin RPC/static/sound traffic works locally, through fail-closed authenticated ngrok, and from a fully offline packaged app.

**Independent Test**: Run the vertical slice and full journeys locally, through valid/invalid ngrok auth, across size/category attacks, and in a clean offline package.

### Tests

**Wave 1 — independent failing tests:**

- [x] **T095** [P] [US12] Add HTTP tests for same-origin static/RPC/sound resources, no wildcard CORS, 4 KiB uncompressed, 8 KiB encoded/decompression limits, compressed/unknown-field growth, SoundManifest categories, and zero canonical calls on rejection · `internal/player/http_test.go`
- [x] **T096** [P] [US12] ⚠️ Reopened (reopened — BUG-002): ~~Add tunnel tests for fixed public domain, credential-pair validation, fail-closed HTTP 401 before capabilities, Connect/static forwarding, precedence, and credential redaction~~ Extend protected-forwarding coverage with a browser-equivalent generated `Subscribe` stream and assert forwarded Host/Origin, Basic Auth handling, Connect content type/status, incremental response flushing, cancellation/reconnect, and credential redaction; the unary SoundManifest check alone is insufficient · `internal/tunnel/service_test.go`, `tests/browser/fixture-server/main.go`
- [x] **T097** [P] [US12] ⚠️ Reopened (reopened — BUG-002): ~~Add local/authenticated-ngrok/invalid-auth journeys covering page, snapshot, all five unary responsibilities, compound updates, multi-tab, reconnect, sounds, and playback~~ Add a credential-gated clean-browser journey against the actual configured ngrok public URL that proves generated `Subscribe` first-snapshot delivery, connection-overlay dismissal, current-terminal rendering, all five unary responsibilities, and reconnect; the local protected fixture cannot satisfy this task · `tests/browser/connectrpc-player.spec.mjs`
- [x] **T098** [P] [US12] Add clean packaged-offline embed tests for generated client, fonts, images, scripts, sounds, and absence of CDN/dev-server dependencies · `internal/platform/assets_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent transport/security/sound pieces:**

- [x] **T099** [P] [US12] Enforce encoded-body, decompression/message, Origin/Host, unsupported-path, cancellation, and temporary-unavailability boundaries before application invocation · `internal/player/http.go`
- [x] **T100** [P] [US12] Preserve fixed-domain protected forwarding, exact Basic Auth failure, configuration precedence, process-local credentials, and redaction · `internal/tunnel/policy.go`
- [x] **T101** [P] [US12] Implement typed SoundManifest discovery with eight allowlisted categories, five extensions, sorted safe relative assets, and empty-success behavior · `internal/player/handler.go`
- [x] **T102** [P] [US12] Consume typed same-origin manifests asynchronously and non-blockingly without accepting caller paths · `client/sound.js`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T103** [US12] Wire serializable config defaults (port 3690, queue 32, startup/shutdown bounds) while keeping credentials and injected dependencies native/private · `main.go`

**⟶ Wait for T103 to finish, then:**

- [x] **T104** [US12] Build both frontends and generated player assets into the clean packaged application without external runtime dependencies · `scripts/build-macos.sh`

**Checkpoint**: User Story 12 is independently functional and testable in local, protected-public, and packaged-offline modes.

---

## Phase 15: User Story 13 — Complete the One-Protocol Cutover (P2)

**Goal**: After parity is proven, remove all active WebSocket/handwritten JSON player implementation, dependencies, fixtures, policy allowances, and authoritative documentation.

**Independent Test**: Scan source, dependencies, routes, built assets, fixtures, CSP, and documentation and find exactly one application-owned public player protocol.

### Tests

**Wave 1 — independent failing cutover assertions:**

- [x] **T105** [P] [US13] Extend source/dependency/route/CSP/fixture/bundle/doc scans to fail on active WebSocket, handwritten player envelopes, generic dispatch, legacy identifiers, or permanent dual-stack artifacts · `internal/platform/assets_test.go`
- [x] **T106** [P] [US13] Make browser parity suites require generated Connect exclusively and fail if any WebSocket constructor or legacy JSON message path is exercised · `tests/browser/connectrpc-player.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent legacy removals from different files:**

- [x] **T107** [P] [US13] Delete the handwritten legacy player envelope types and decoder implementation after all typed parity gates pass · `internal/player/protocol.go`
- [x] **T108** [P] [US13] Remove WebSocket upgrade/route/CSP handling and the structured `/api/sounds/{folder}` endpoint, leaving generated RPC and ordinary static resources · `internal/player/http.go`
- [x] **T109** [P] [US13] Remove browser WebSocket construction, legacy JSON dispatch, and superseded message identifiers from the player · `client/client.js`
- [x] **T110** [P] [US13] Remove the direct `github.com/coder/websocket` requirement after no production import remains · `go.mod`
- [x] **T111** [P] [US13] Remove WebSocket `ws:`/`wss:` policy allowances from the player document while retaining required same-origin Connect/static policy · `client/index.html`
- [x] **T113** [P] [US13] Replace legacy JSON protocol fixtures with bounded protobuf, malformed, compressed, and unknown-field test fixtures · `internal/testutil/testdata/protobuf/README.md`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — dependency/test cleanup:**

- [x] **T114** [P] [US13] Remove superseded protocol unit tests and retain equivalent generated contract/handler coverage · `internal/player/protocol_test.go`
- [x] **T115** [P] [US13] Regenerate Go dependency checksums after the direct legacy transport removal · `go.sum`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — independent documentation cutover:**

- [x] **T116** [P] [US13] Document ConnectRPC operation, same-origin/browser generation, troubleshooting, and the absence of a supported legacy player protocol · `README.md`
- [x] **T117** [P] [US13] Mark the retained completed WebSocket contract as superseded and non-authoritative historical context · `specs/001-wails-v2-migration/contracts/websocket-protocol.md`

**Checkpoint**: User Story 13 is independently testable with one active public player protocol and zero permanent dual stack.

---

## Phase 16: Polish and Cross-Cutting Validation

**Purpose**: Close cross-cutting documentation, soak, inventory, security, and Success Criteria evidence after all story checkpoints pass.

**Wave 1 — independent final evidence preparation:**

- [x] **T118** [P] Reconcile the final generated descriptors and code against the inventory and record zero unclassified DTO/config fields plus explicit third-party/native exclusions · `specs/005-connectrpc-protobuf-migration/contracts/inventory.md`
- [x] **T119** [P] Add clean-checkout generation, local/ngrok operation, offline packaging, and rollback-free cutover instructions · `specs/005-connectrpc-protobuf-migration/quickstart.md`
- [x] **T120** [P] ⚠️ Reopened (reopened — BUG-002): ~~Run and record a representative three-to-four-hour local and authenticated-ngrok idle-stream/reconnect soak~~ Run and document the authenticated streaming/reconnect portion through the actual configured ngrok public endpoint; retain synthetic component evidence separately, and record unavailable external preconditions as `NOT RUN` rather than `PASS` · `specs/005-connectrpc-protobuf-migration/soak.md`

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T121** ⚠️ Reopened (reopened — BUG-002): ~~Validate every Success Criterion by running Buf format/lint/breaking/two-pass generation, `gofmt`, `go vet`, `go test`, `go test -race`, frontend/player builds, Playwright journeys, source/bundle/security scans, and the affected macOS packaged smoke; record commands and evidence~~ Validate every Success Criterion with those gates, including SC-035; SC-030 and SC-031 require actual configured ngrok public-endpoint evidence and cannot pass from local protected-forwarding simulation · `specs/005-connectrpc-protobuf-migration/validation.md`

**⟶ Wait for T121 to finish, then:**

- [x] **T122** ⚠️ Reopened (reopened — BUG-002): ~~Resolve every failed criterion or explicitly document an external manual-only result without weakening the specification~~ Resolve BUG-002 and every failed criterion, or explicitly document an owner, rationale, and follow-up; unavailable real-ngrok evidence remains `NOT RUN` and cannot be promoted to `PASS` · `specs/005-connectrpc-protobuf-migration/validation.md`

---

## Dependencies & Execution Order

### Phase Dependencies

1. **Setup (Phase 1)** blocks Foundational because the pinned generator/toolchain and inventory ledger must exist first.
2. **Foundational (Phase 2)** blocks every user story because all generated values, detached models, adapters, limits, and coordinator ownership depend on the completed schema graphs.
3. **P1 stories (Phases 3–12)** execute in listed order: the MVP stream/selection slice enables reconnect and multi-tab; those enable complete typed actions, authority, replay, randomness, stream isolation, desktop compatibility, and persistence compatibility.
4. **P2 stories (Phases 13–15)** execute after P1 parity: reproducible governance and production modes must pass before the legacy protocol can be removed.
5. **Polish (Phase 16)** begins only after the one-protocol cutover and owns the single full Success Criteria suite run.
6. **BUG-002 recovery (Phase 19)** overrides the original document placement of its reopened tasks: reproduce first, encode failing protected-stream and real-public browser regressions, apply the confirmed boundary fix, then replace soak and Success Criteria evidence.

### Wave Restatement

- **Phase 1**: independent ledger/config manifests → dependency lock.
- **Phase 2**: independent schema files → generated outputs/models/fakes → coordinator, adapter, and limits foundations.
- **Phase 3 / US1**: failing server/coordinator/browser tests → coordinator/stream/browser pieces → handler → HTTP mount → composition.
- **Phase 4 / US2**: failing live/browser tests → live/client/sound behavior → coordinator snapshot integration.
- **Phase 5 / US3**: failing coordinator/browser tests → server/browser convergence → client-count server integration.
- **Phase 6 / US4**: failing coordinator/handler/browser tests → transaction/adapter/client/sound work → handlers → production-shaped browser fixture → parity journey migration.
- **Phase 7 / US5**: failing authority/transport tests → validation and coordinator authority → handler mapping → browser rejection proof.
- **Phase 8 / US6**: failing coordinator/browser replay tests → deterministic fingerprint → replay store.
- **Phase 9 / US7**: failing coordinator/live randomness tests → accepted live transition → serialized coordinator commit.
- **Phase 10 / US8**: failing stream/server tests → nonblocking publication and stream lifecycle → bounded server shutdown.
- **Phase 11 / US9**: failing adapter/App/public-surface scans → private adapters → Wails composition.
- **Phase 12 / US10**: failing session/player-config/save tests → independent persistence adapters → independent service integration.
- **Phase 13 / US11**: failing descriptor/breaking fixtures → BUG-001 restoration of pinned Buf-only generation and compatible provenance checks → CI removal of standalone `protoc` setup → Buf regeneration and full gate rerun.
- **Phase 14 / US12**: failing HTTP/tunnel/browser/package tests → independent boundary/security/sound work → application config → package build.
- **Phase 15 / US13**: failing cutover scans → independent production removals (fixture migration completed as the Phase 6 browser prerequisite) → dependency/test cleanup → documentation cutover.
- **Phase 16**: independent inventory/quickstart/soak evidence → single full Success Criteria validation → remediation closeout.
- **Phase 19 / BUG-002**: T133 real authenticated-ngrok reproduction → reopened T096 and T097 regression coverage → T134 confirmed minimal fix → reopened T120 public soak → T121 criterion validation → T122 remediation closeout.

### Parallel Opportunities

Only tasks tagged `[P]` within the same wave are independent. They touch different files and have no incomplete dependency in that phase. Do not parallelize tasks across a join, tasks that edit the same file, generated-output writers, coordinator changes that depend on a live-service contract, or the final validation/remediation pair.

## Phase 17: Convergence

- [x] T123 CRITICAL make subscription attachment, complete snapshot capture, physical-stream registration, and post-snapshot update delivery one gap-free ordered boundary, with a deterministic concurrent attach/mutation regression in `internal/control/service.go`, `internal/player/handler.go`, and `internal/player/handler_test.go` per FR-043 (contradicts)
- [x] T124 CRITICAL assemble every accepted mutation's complete personalized compound updates from the committed revision and offer each affected logical session exactly once inside the coordinator transaction before the unary response, removing post-commit recomputation and duplicate offers in `internal/control/service.go`, `internal/player/handler.go`, and their tests per FR-144 (contradicts)
- [x] T125 CRITICAL implement and exercise exhaustive structured private request, result, status, and event adapters across Wails compatibility DTOs, generated private protobuf values, and transport-independent values, replacing discarded semantic conversions and making descriptor field/enum/`oneof` additions fail verification in `app_contract.go`, `app.go`, and `app_contract_test.go` per Constitution IV (missing)
- [x] T126 CRITICAL update all feature-scoped Go tests to use Testify assertions, table-driven cases for repeated flows, `t.Context()` roots, and `protocmp`/`prototest` for protobuf values and descriptors, then add an enforceable convention scan per Constitution: Testing and Quality Gates (contradicts)
- [x] T127 CRITICAL label every retained completed WebSocket/JSON feature document as superseded historical non-authoritative context, link the current Connect contract where applicable, and extend the final documentation scan beyond the feature-001 contract per Constitution VII (contradicts)
- [x] T128 return Connect `resource_exhausted` for encoded-body/decompression/message limit breaches and Connect `unimplemented` for unsupported public services or procedures before any adapter or canonical-service invocation, with raw framing/code and zero-side-effect tests in `internal/player/http.go` and `internal/player/http_test.go` per FR-074–FR-075 (contradicts)
- [x] T129 [US11] ⚠️ Reopened (reopened — BUG-001): ~~pin Protocol Buffers compiler/toolchain v35.0 in the reproducible generation path, verify its version and generated provenance, and fail generation drift when the compiler pin is absent or mismatched in `scripts/proto-generate.sh`, `scripts/proto-check.sh`, and CI per FR-133 (partial)~~ Remove `proto/toolchain.env`, `scripts/ensure-protoc.sh`, direct compiler invocation, checksum/download handling, and standalone compiler CI setup; after T091, T092, and T094, regenerate all checked-in outputs through Buf and rerun deterministic generation, compatibility, Go, browser, and package gates per corrected FR-133/FR-137 (replaced — BUG-001)

## Phase 18: Convergence

- [x] T130 CRITICAL make `scripts/proto-check.sh` capture the checked-in `internal/gen` and `client/gen` state before regeneration and fail on first-pass generation drift or manual edits, preserve the two-pass determinism check, and enforce the same failure in `.github/workflows/wails-macos.yml` with regression coverage per Constitution V and FR-134/FR-137 (contradicts)
- [x] T131 CRITICAL replace handwritten conditional assertions using `require.Fail*`, `reflect.DeepEqual`, direct `testing.T` failures, and non-exempt `context.Background()` roots across every feature-scoped Go test named by the plan or tasks with Testify helpers, table-driven cases, `t.Context()`, and `cmp`/`protocmp`/`prototest` as applicable, then extend `internal/platform/test_conventions_test.go` to detect those evasions per Constitution: Testing and Quality Gates (contradicts)
- [x] T132 remove the unused out-of-transaction `offerCurrentPlayerState` compatibility helper from `internal/player/handler.go` and keep authoritative stream publication exclusively on coordinator-assembled complete compound updates per plan: atomic compound publication and FR-144 (unrequested)

## Phase 19: BUG-002 — Authenticated ngrok Subscribe Recovery

**Purpose**: Reproduce the real public streaming failure, convert it into regression coverage, apply only the confirmed fix, and replace invalid authenticated-ngrok acceptance evidence.

**BUG-002 override order**: T133 → T096 and T097 → T134 → T120 → T121 → T122. Reopened tasks keep their original IDs but follow this recovery order instead of their earlier document position.

### Wave 1 — Reproduce and isolate

- [x] **T133** [US12] Reproduce the stuck `УСТАНОВКА СВЯЗИ...` state through the actual configured authenticated ngrok URL and capture redacted browser console/network timing, request/response status and headers, application logs, and ngrok diagnostics sufficient to distinguish Host/Origin rejection, Basic Auth propagation, Connect content-type/status failure, proxy buffering/flushing, and cancellation/reconnect defects · `specs/005-connectrpc-protobuf-migration/bugs/BUG-002-evidence.md`

**⟶ Wait for T133, then complete reopened T096 and T097 as failing regressions before implementation.**

### Wave 2 — Correct the confirmed boundary

- [x] **T134** [US12] Implement the minimal fix proven by T133 and reopened T096/T097 so authenticated public generated `Subscribe` delivers the first snapshot, dismisses the connection overlay, renders the terminal, and reconnects without weakening same-origin validation, public-surface Basic Auth, or credential redaction · `internal/player/http.go`, `internal/tunnel/policy.go`, `client/client.js` (as applicable to the confirmed cause)

**⟶ Wait for T134, then complete reopened T120, T121, and T122 against the corrected real public endpoint and all local regression gates.**
