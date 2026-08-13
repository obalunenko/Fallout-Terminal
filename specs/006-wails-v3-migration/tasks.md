# Tasks: Wails v3 Runtime Migration

**Bugfix**: 2026-08-14 — [ANALYZE-2026-08-14] Replaced the conflicting lifecycle-phase schema task, completed isolated protobuf-tool coverage, and moved final P1 journeys behind bridge integration.
**Bugfix**: 2026-08-14 — [ANALYZE-CUTOVER-2026-08-14] Ordered rollback and soak before v2 removal, preserved historical artifacts unchanged, and made frontend/status and soak dependencies explicit.
**Bugfix**: 2026-08-14 — [ANALYZE-FINAL-MATRIX-2026-08-14] Established one sequential final-verification owner and restart-on-change evidence semantics.
**Bugfix**: 2026-08-14 — [ANALYZE-FINAL-CANDIDATE-2026-08-14] Added clean browser-test installation, final-candidate native/soak reruns, and separate build-candidate/evidence identities.
**Bugfix**: 2026-08-14 — [ANALYZE-ATTRIBUTABLE-CANDIDATE-2026-08-14] Made candidate capture clean and committed, then split final native, package, and soak verification.
**Bugfix**: 2026-08-14 — [ANALYZE-BUNDLE-IDENTITY-2026-08-14] Added canonical bundle hashing and froze active documentation before candidate capture.

**Input**: Design documents from `/specs/006-wails-v3-migration/`

**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`, and constitution 3.3.1

**Testing**: This migration explicitly requires focused automated tests, deterministic generation, race testing, Playwright journeys, macOS package inspection, manual native journeys, soak, and rollback evidence. Tests use Testify, table-driven cases, `t.Context()`, and protobuf-aware comparison conventions.

**Organization**: Tasks are grouped by prioritized user story after shared setup and foundations. Constitution 3.3.1 supersedes older direct-CLI examples: every Go development tool is isolated under `tools/<tool>/`, invoked from the repository root with `go tool -modfile=tools/<tool>/go.mod <command>`, and absent from the application `go.mod`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Safe to execute in parallel because files do not overlap and prerequisites are complete
- **[Story]**: User-story traceability label such as `[US1]`
- Every task names exact repository paths
- Verification tasks name the command or observable manual journey
- An unchecked task is not evidence that its command or journey passed
- Task IDs are stable traceability identifiers; file order and explicit dependency statements govern execution, not numeric order alone

## Phase 1: Setup and Contract Reconciliation

**Purpose**: Reconcile post-plan governance, capture rollback authority, and isolate every Go development tool before runtime migration.

- [ ] T001 Add a read-only migration-contract preflight that rejects unqualified Wails commands, floating versions, root application tool declarations/tool-only dependencies, and serialized lifecycle-phase additions in active feature artifacts in `scripts/wails-v3-contract-check.sh`
- [ ] T002 Create the feature-006 rollback evidence shell with canonical source SHA `f1084b3df8b5630862bdf7a0f347b599156653ef`, empty real-evidence fields, triggers, owner, and cutover expiry in `docs/wails-v3-migration-rollback.md`
- [ ] T003 [P] Declare only `tool github.com/wailsapp/wails/v3/cmd/wails3` with exact parent-module `require github.com/wailsapp/wails/v3 v3.0.0-beta.8` and an explicit Go version in `tools/wails/go.mod`, then commit its isolated tidy resolution in `tools/wails/go.sum`
- [ ] T004 [P] Declare only `tool github.com/bufbuild/buf/cmd/buf` with exact parent-module `require github.com/bufbuild/buf v1.72.0` and an explicit Go version in `tools/buf/go.mod`, then commit its isolated tidy resolution in `tools/buf/go.sum`
- [ ] T005 [P] Declare only `tool google.golang.org/protobuf/cmd/protoc-gen-go` with exact parent-module `require google.golang.org/protobuf v1.36.11` and an explicit Go version in `tools/protoc-gen-go/go.mod`, then commit its isolated tidy resolution in `tools/protoc-gen-go/go.sum`
- [ ] T006 [P] Declare only `tool connectrpc.com/connect/cmd/protoc-gen-connect-go` with exact parent-module `require connectrpc.com/connect v1.20.0` and an explicit Go version in `tools/protoc-gen-connect-go/go.mod`, then commit its isolated tidy resolution in `tools/protoc-gen-connect-go/go.sum`
- [ ] T007 Remove the root `tool` block and every tool-only dependency/checksum while retaining only application dependencies in `go.mod` and `go.sum`, then prove each `tools/*/go.mod` can resolve without root module drift
- [ ] T008 Add a deterministic tool-module and forbidden-global-install verifier for `tools/*/go.mod`, root `go.mod`, active scripts, workflows, and docs in `scripts/tool-modules-check.sh`

**Checkpoint**: Rollback source is recorded; design artifacts agree with constitution 3.3.1; every current Go tool has one isolated module; root application module has zero tool declarations or tool-only dependencies.

---

## Phase 2: Foundational Wails v3 Host and Contract Prerequisites

**Purpose**: Establish exact runtime pins, visible build assets, host/service boundaries, and the private runtime-status contract required by all stories.

**⚠️ CRITICAL**: Complete this phase before user-story implementation.

- [ ] T009 Pin `github.com/wailsapp/wails/v3 v3.0.0-beta.8` as an application dependency and exact `@wailsio/runtime` `3.0.0-beta.8` plus its Vite plugin subpath in `go.mod`, `go.sum`, `frontend/package.json`, and `frontend/package-lock.json`
- [ ] T010 [P] Add beta.8-derived visible build assets with preserved product metadata, macOS 13 arm64 profile, and `build/bin` output in `Taskfile.yml`, `build/config.yml`, `build/Taskfile.yml`, `build/common/Taskfile.yml`, and `build/darwin/Taskfile.yml`
- [ ] T011 Convert Buf and protobuf generator calls to repository-root isolated-module invocations without recursive generation, update root-relative inputs/plugins/outputs, and prove generation leaves root `go.mod`/`go.sum` unchanged in `scripts/proto-generate.sh`, `scripts/proto-check.sh`, `scripts/proto-breaking.sh`, `scripts/proto-drift-test.sh`, `proto/buf.gen.go.yaml`, and `proto/buf.gen.es.yaml`
- [ ] T012 [P] Add table-driven assertions for isolated tool modules, zero root tool directives, exact Wails pin consistency, Taskfile/config ownership, and one root command in `internal/platform/startup_test.go`
- [ ] T013 Introduce narrow Wails v3 host capabilities and a dedicated 25-operation forwarding service, keeping core lifecycle methods on an unbound type in `wails_host.go` and `desktop_service.go`
- [ ] T014 Add reflection/inventory tests proving exactly 25 desktop operations and zero lifecycle/generic/native/player capabilities in `app_contract_test.go`
- [ ] T015 Add descriptor and compatibility assertions proving the feature-005 `RuntimeStatus` schema remains unchanged and contains no lifecycle-phase enum or field in `app_contract_test.go`, `internal/platform/assets_test.go`, and `scripts/wails-v3-contract-check.sh`, leaving `proto/fallout/terminal/private/v1/runtime.proto`, `proto/schema-revision.txt`, and `proto/compatibility-baseline.binpb` unchanged
- [ ] T016 Preserve internal lifecycle phase transitions in Go-only host/core state and project actionable master presentation through the existing `startupError` and server-information fields in `app.go`, `app_contract.go`, `app_contract_test.go`, and `app_test.go` without changing detached native runtime-status shapes
- [ ] T017 Replace v2-specific source assertions with Wails v3 application/service/window/assets seams while retaining master/player separation and resource checks in `internal/platform/assets_test.go` and `production_resources_test.go`

**Checkpoint**: Exact pins resolve reproducibly; root module remains application-only; v3 build/host seams compile; internal lifecycle phase is testable without serialization; the unchanged runtime-status schema and allowlisted service inventory are testable before frontend integration.

---

## Phase 3: User Story 1 — Run the Same Native Master Application (Priority: P1) 🎯 MVP

**Goal**: Launch one accepted native master window and preserve session, player-configuration, dialog, URL, and visible master behavior.

**Independent Test**: On macOS 13+ arm64, launch the candidate, verify the exact one-window properties and dark controls, then create/open/edit/copy/save/reopen representative session-v1 and player-config-v1 files with baseline-compatible results.

### Verification for User Story 1

- [ ] T018 [P] [US1] Add Wails v3 window/options/assets composition tests for one title, size, minimum, background, master filesystem, and macOS last-window behavior in `wails_host_test.go` and `internal/platform/assets_test.go`
- [ ] T019 [P] [US1] Add injected dialog/browser adapter tests for titles, JSON filters, default paths/filenames, alias policy, directory creation, cancel-as-empty, errors, and HTTP(S)-only URLs in `internal/platform/desktop_test.go`

### Implementation for User Story 1

- [ ] T020 [US1] Replace `wails.Run(options.App)` with Wails v3 application creation, master assets, late service registration, and exactly one accepted window in `main.go` and `wails_host.go`
- [ ] T021 [US1] Replace Wails v2 global dialog/browser calls with injected Wails v3 managers while preserving defaults, cancellation, validation, and redaction in `internal/platform/desktop.go` and `internal/platform/paths.go`
- [ ] T022 [US1] Wire every accepted session, player-config, coordination, terminal, hack, demo-copy, save, and URL operation through `desktop_service.go`, `app.go`, and `app_contract.go` without adding authored UI capabilities
- [ ] T023 [US1] Preserve session-v1/player-config-v1 paths, strictness, unknown-field behavior, explicit demo copy, and newest-revision durability in `app_test.go`, `internal/session/`, `internal/playerconfig/`, and `sessions/demo.json`
**Checkpoint**: The one-window host and accepted persistence/native-operation implementation slice is complete; its final native journey waits for the generated v3 facade/events integration checkpoint.

---

## Phase 4: User Story 2 — Start and Stop Every Owned Runtime Resource Once (Priority: P1)

**Goal**: Preserve bounded, observable startup and reverse, idempotent cleanup across player, tunnel, session worker, desktop adapter, and window triggers.

**Independent Test**: Exercise normal startup, occupied listener, desktop/event failure, tunnel failure/unsafe URL, timeout, partial acquisition, repeated shutdown, window close, Cmd+Q, and development interrupt while verifying status, local fallback, order, deadlines, and zero leaks.

### Verification for User Story 2

- [ ] T025 [P] [US2] Add table-driven lifecycle-host tests for Wails startup abort classification, application-lifetime command context, fresh shutdown context, partial acquisition, and repeated calls in `wails_host_test.go`
- [ ] T026 [P] [US2] Extend core tests for phase/status publication, tunnel acquired-before-validation ownership, failed first stop retry, reverse cleanup, timeouts, and safe redaction in `app_test.go` and `internal/tunnel/service_test.go`

### Implementation for User Story 2

- [ ] T027 [US2] Implement the Wails v3 lifecycle service with bounded core startup, application-lifetime operation context, and non-aborting handling for status-visible application failures in `wails_host.go` and `app.go`
- [ ] T028 [US2] Mark tunnel ownership before public-URL validation and preserve local-ready fallback plus retryable cleanup after an invalid or failed tunnel in `app.go` and `internal/tunnel/service.go`
- [ ] T029 [US2] Implement one fresh five-second background shutdown context and preserve tunnel → player → session worker → desktop cleanup without skipping later resources after errors in `wails_host.go` and `app.go`
- [ ] T030 [US2] Replace retained v2 startup/DomReady contexts with an application-lifetime event/desktop capability and bounded acquisition children in `main.go`, `app.go`, and `internal/platform/desktop.go`
**Checkpoint**: Automated lifecycle ownership and cleanup behavior is complete; final native trigger evidence waits for the generated v3 facade/events integration checkpoint.

---

## Phase 5: User Story 3 — Keep Browser-Player Gameplay and Privacy Unchanged (Priority: P1)

**Goal**: Preserve the generated ConnectRPC player surface, 4–7-client convergence, reconnect/replay/concurrency behavior, sounds, and private-field isolation.

**Independent Test**: Run four-, five-, six-, and seven-browser generated-player journeys with at least 25 mixed valid/rejected actions, reconnect, replay, slow/overflow streams, and sounds; inspect the listener/bundle for zero Wails/private/legacy exposure.

- [ ] T033 [P] [US3] Strengthen public descriptor, generated-bundle, route, CSP, and forbidden Wails/private/WebSocket surface assertions in `internal/platform/assets_test.go` and `internal/player/server_test.go`
- [ ] T034 [P] [US3] Extend generated ConnectRPC journeys to explicit 4–7-client mixed actions, reconnect, replay, slow/overflow, sound, and private-field assertions in `tests/browser/connectrpc-player.spec.mjs` and `tests/browser/player-sessions-control.spec.mjs`
- [ ] T035 [US3] Preserve separate `client/dist` embedding and same-origin generated `PlayerService` serving while adapting only Wails host asset composition in `main.go`, `production_resources.go`, and `internal/player/server.go`
- [ ] T036 [US3] Run `go test ./internal/player ./internal/control ./internal/live ./internal/hack`, then `npm ci --prefix tests/browser` and `npm test --prefix tests/browser`, recording actual generated-player results in `specs/006-wails-v3-migration/quickstart.md`
**Checkpoint**: Automated browser-player parity and isolation are complete; the final native 4–7-player journey waits for the generated v3 facade/events integration checkpoint.

---

## Phase 6: User Story 4 — Preserve the Private Desktop Facade and Events (Priority: P1)

**Goal**: Preserve exact 25-operation semantics and four event payloads/readiness rules behind the runtime-neutral facade, with no lifecycle or privileged fallback exposure.

**Independent Test**: Generate bindings twice, compare inventory/content, exercise every method/result/error/cancel path and all four event races/releases, then scan source and production bundle for required and forbidden surfaces.

### Verification for User Story 4

- [ ] T038 [P] [US4] Add deterministic double-generation, exact method inventory, content comparison, and forbidden lifecycle/native surface checks in `scripts/wails-bindings-check.sh`
- [ ] T039 [P] [US4] Add browser-level facade tests with structurally test-only generated-service/event doubles for calls, normalization, four events, listener-first snapshot races, release, and hot disposal in `tests/browser/desktop-api.spec.mjs`, `tests/browser/fixtures/desktop-bindings.js`, and `tests/browser/playwright.config.mjs`
- [ ] T040 [P] [US4] Extend private adapter/event/status tests for unchanged runtime-status fields, Go-only lifecycle phase, exact native shapes, detachment, cancellation, errors, and all 25 operations in `app_contract_test.go` and `app_test.go`

### Implementation for User Story 4

- [ ] T041 [US4] Register the exact four typed Wails v3 events and replace retained-context `runtime.EventsEmit` with the injected application event manager in `wails_host.go`, `main.go`, and `app.go`
- [ ] T042 [US4] Generate the allowlisted service only into `frontend/bindings` and configure the exact official runtime Vite plugin in `build/config.yml` and `frontend/vite.config.js`
- [ ] T043 [US4] Replace `frontend/wailsjs`, `window.go`, and `window.runtime` discovery with explicit v3 service/runtime imports and production-failing missing-binding behavior in `frontend/src/desktop-api.js`
- [ ] T044 [US4] Implement all-four-listeners-before-snapshot readiness, per-field newer-event precedence, `.data` unwrapping, local-URL retention, exact-once release, and no DomReady dependency in `frontend/src/desktop-api.js`
- [ ] T031 [US2] After T043–T044, render starting, ready-local, ready-public, and failed presentation from existing status fields with actionable redacted startup/tunnel errors in `frontend/src/master.js` and `frontend/src/master.css`
- [ ] T045 [US4] Enforce that `master.js` consumes only `window.desktopAPI`, the T031 presentation uses existing status fields without a serialized phase, and no `CopyDemo` or generic/native UI capability is added in `frontend/src/master.js`, `frontend/src/index.html`, and `internal/platform/assets_test.go`
- [ ] T046 [US4] Run the binding/facade/event suites and source/bundle scans, recording exact inventory and readiness evidence in `specs/006-wails-v3-migration/quickstart.md`

**Checkpoint**: The generated service contains exactly 25 accepted operations, the facade preserves native semantics, all four event subscriptions are race-safe, and production contains no lifecycle/global/generic capability.

---

## Phase 7: Integrated P1 Native Acceptance

**Purpose**: Run the native acceptance journeys only after the one-window host, lifecycle ownership, public player service, generated desktop bindings, facade, events, and readiness behavior are integrated.

- [ ] T024 [US1] Execute and record the one-window and representative master persistence journey through the generated v3 facade without prechecking results in `specs/006-wails-v3-migration/quickstart.md`
- [ ] T032 [US2] Execute and record normal close, Cmd+Q, handled `go tool -modfile=tools/wails/go.mod wails3 dev` interrupt, occupied port, tunnel failure, partial startup, and repeated shutdown evidence in `specs/006-wails-v3-migration/quickstart.md`
- [ ] T037 [US3] Execute the local 4–7-player parity journey through the integrated native host and record credential-gated public behavior as real evidence or `NOT RUN` in `specs/006-wails-v3-migration/quickstart.md`

**Checkpoint**: US1–US4 pass their integrated P1 journeys without a v2/global bridge fallback; application failures remain actionable, player behavior remains feature-005-compatible, and cleanup completes within five seconds.

---

## Phase 8: User Story 5 — Develop and Build Reproducibly from the Repository Root (Priority: P2)

**Goal**: Provide one root development entry and a deterministic nonrecursive protobuf → player → bindings → master → native graph using isolated Go tool modules.

**Independent Test**: From a clean checkout, resolve locked dependencies, run the one root development entry, generate twice, build both frontends directly, build native twice, and observe zero floating resolution, root module drift, stale output, or unexplained tracked diff.

- [ ] T047 [P] [US5] Add Taskfile graph/source tests for isolated tools, protobuf-before-player, bindings-before-master, run-once generation, no recursion, build-asset ownership, and `build/bin` output in `internal/platform/startup_test.go`
- [ ] T048 [US5] Implement the root `Taskfile.yml` and common build graph so `go tool -modfile=tools/wails/go.mod wails3 dev` owns protobuf verification, `client/` build, binding generation, `frontend/` build, host start, and optional configured tunnel without duplicate generation
- [ ] T049 [US5] Make locked direct frontend builds compatible with the graph and keep the player independent of Wails in `frontend/package.json`, `frontend/package-lock.json`, `frontend/vite.config.js`, `client/package.json`, `client/package-lock.json`, and `client/vite.config.js`
- [ ] T050 [US5] Add clean repeated protobuf/binding/frontend/native-build orchestration and tracked-drift checks in `scripts/reproducible-build-check.sh`
- [ ] T051 [US5] Update `.github/workflows/wails-macos.yml` to resolve every Go tool through `tools/*/go.mod`, cache every module sum, generate bindings before master compilation, run `npm ci --prefix tests/browser` before the Playwright suite, and run pin/tool/root-module-drift checks
- [ ] T052 [US5] Remove direct/global Wails and root-module Go tool instructions from active setup/build documentation in `README.md` and `specs/006-wails-v3-migration/quickstart.md`
- [ ] T053 [US5] Run both direct locked frontend builds, two clean generations, two clean native builds, and `scripts/tool-modules-check.sh`, recording actual outcomes in `specs/006-wails-v3-migration/quickstart.md`
- [ ] T054 [US5] Launch and stop the complete system once with `go tool -modfile=tools/wails/go.mod wails3 dev`, recording one-command startup and zero post-stop listener/process drift in `specs/006-wails-v3-migration/quickstart.md`

**Checkpoint**: Clean local and CI builds use the same isolated pins and ordered graph; the root application module stays tool-free; one repository-root development entry starts and stops the complete system.

---

## Phase 9: User Story 6 — Package a Self-Contained Personal-Use macOS App (Priority: P2)

**Goal**: Produce a final ad-hoc-signed macOS 13+ arm64 app at `build/bin/Fallout Terminal.app` with all resources inserted before signing and no runtime developer/network dependency.

**Independent Test**: Package from clean source, inspect architecture/metadata/entitlements/icon/resources/signature, disconnect external networking, launch one app/listener, run master/local-player smoke, and quit with zero owned resources.

- [ ] T055 [P] [US6] Add package/resource/sign-order/output/deployment-target assertions in `production_resources_test.go`, `internal/platform/assets_test.go`, and `internal/platform/startup_test.go`
- [ ] T056 [US6] Customize Darwin assembly to copy `sessions/demo.json` and every non-embedded resource before final ad-hoc signing while preserving plist, entitlements, icon, macOS 13, arm64, and `build/bin` in `build/darwin/Taskfile.yml`, `build/darwin/Info.plist`, `build/darwin/Info.dev.plist`, and `build/darwin/entitlements.plist`
- [ ] T057 [US6] Preserve/test `Contents/Resources` resolution and remove `production_resources_bindings.go` only after clean v3 generation/package proof, otherwise document and narrow it in `internal/platform/paths.go`, `production_resources.go`, `production_resources_bindings.go`, and `production_resources_test.go`
- [ ] T058 [US6] Add `scripts/hash-macos-app.sh` to produce a canonical bundle-manifest SHA-256 from byte-sorted bundle-relative paths, entry type, POSIX mode, regular-file content digest, and symlink target while excluding timestamps and host-specific extended attributes and rejecting missing, unreadable, or changing entries; add a `scripts/hash-macos-app.sh --self-test` contract covering unchanged-repeat, changed-file, changed-mode, changed-symlink, added-entry, and removed-entry fixtures; and extend `scripts/verify-macos-app.sh` with architecture, plist, minimum OS, entitlements, icon, resource, signature, offline-assets, and mutation-order verification
- [ ] T059 [US6] Adapt the proven Developer ID/notary/staple/DMG/Gatekeeper pipeline to the isolated pinned Wails tool without weakening preflight, redaction, or SHA-256 behavior in `scripts/build-macos.sh`
- [ ] T060 [US6] Package and inspect the arm64 personal-use app through isolated Wails tooling and upload the established path in `.github/workflows/wails-macos.yml`
- [ ] T061 [US6] Execute the offline personal-use package launch, one-listener master/player smoke, and clean quit; record conditional public trust gates as real results or `NOT RUN` in `specs/006-wails-v3-migration/quickstart.md`

**Checkpoint**: The final app is complete before signing, valid for the personal-use profile, launches offline from the established path, and cleans up owned runtime resources.

---

## Phase 10: User Story 7 — Cut Over Safely with Immutable Wails v2 Rollback (Priority: P2)

**Goal**: Prove rollback without data conversion, complete pre-removal parity/qualification-soak evidence, and remove every active Wails v2 path only after required gates pass.

**Independent Test**: Verify the canonical v2 SHA, use safety copies for an actual rollback source/artifact journey, restore unchanged version-1 data, then scan final source/dependencies/bindings/bundle/docs for zero active v2 or dual-runtime path.

- [ ] T062 [P] [US7] Add source rollback identity, optional-artifact evidence discipline, data safety, triggers, drill, and historical-document separation assertions in `internal/platform/startup_test.go` and `internal/platform/assets_test.go`
- [ ] T063 [P] [US7] Add comprehensive active-v2, dual-runtime, floating-tool, generated-global, dependency, bundle, CI/script, and documentation scans in `scripts/wails-v3-cutover-check.sh`
- [ ] T064 [US7] Complete actual candidate identity, rollback procedure, safety-copy steps, owner/expiry, trigger table, and evidence fields without inventing a v2 artifact hash in `docs/wails-v3-migration-rollback.md`
- [ ] T065 [US7] Update active Wails v2 operating links in `README.md` to point to `specs/006-wails-v3-migration/quickstart.md` and `docs/wails-v3-migration-rollback.md`, treating `specs/001-wails-v2-migration/` and `docs/wails-migration-rollback.md` as immutable historical evidence rather than edit targets
- [ ] T066 [US7] Perform the source or genuinely accepted-artifact rollback drill with safety copies and record actual version-1 master/player results in `docs/wails-v3-migration-rollback.md` and `specs/006-wails-v3-migration/quickstart.md`
- [ ] T067 [US7] Run the pre-removal qualification soak using the complete required 60-minute local workload with 4–7 players, 25 mixed operations, three reconnects, two save/reopen cycles, navigation, hacking, coordination, sound, convergence/revision checks, one listener, five `ps -o rss= -p <APP_PID>` samples ten seconds apart at minutes 15/30/60, failure when both later medians exceed 125% of the 15-minute median, and five-second post-quit cleanup; with real credentials/connectivity run the pre-removal 30-minute authenticated-ngrok qualification workload with 4–7 players, 15 mixed public operations, two reconnects, one unauthorized rejection, controlled tunnel-loss/local-fallback proof, convergence, isolation, and five-second cleanup, recording `PASS`, `FAIL`, or `NOT RUN` in `specs/006-wails-v3-migration/quickstart.md`; these qualification results permit v2 removal but do not replace T080 local and T077 public final-candidate reruns
- [ ] T068 [US7] Only after T066–T067 and every other required parity/package gate passes, remove Wails v2 imports/dependency, `wails.json`, v2 post-build hook, v2 generated/global assumptions, obsolete workaround, and every temporary dual path from `main.go`, `internal/platform/desktop.go`, `go.mod`, `go.sum`, `wails.json`, `build/darwin/postbuild.sh`, and `frontend/src/desktop-api.js`
- [ ] T069 [US7] After T068, commit all prior evidence separately; complete and commit `README.md` and every active operating document with the v2-free application/configuration source; require empty `git status --porcelain`; capture `BUILD_CANDIDATE_SHA=$(git rev-parse HEAD)` and the exact pin set; and only then record that identity in `docs/wails-v3-migration-rollback.md` and `specs/006-wails-v3-migration/quickstart.md`; before T070 and every later final task verify every tracked path except result/status fields in those two evidence files remains identical to the build candidate, never redefine it with an evidence-only commit, and commit a new candidate plus restart the final sequence at T070 if any frozen tracked path needs correction

**Checkpoint**: Rollback is real and data-compatible; the provisional candidate has zero active Wails v2 path or permanent switch and is ready for the one authoritative final matrix; unavailable external checks are never represented as passes.

---

## Phase 11: Cross-Cutting Verification and Polish

**Purpose**: Run the one authoritative complete matrix sequentially against the provisional v2-free candidate and reconcile all evidence without weakening existing assertions.

- [ ] T070 Run `scripts/tool-modules-check.sh`, isolated Buf format/lint/build/generation, `scripts/proto-drift-test.sh`, `scripts/proto-breaking.sh --all-fixtures`, graph-isolation checks, and `git diff --exit-code -- internal/gen client/gen`, leaving zero root-module or generated drift
- [ ] T071 Run two clean Wails binding generations plus `scripts/wails-bindings-check.sh`, leaving identical content/inventory, the exact allowlisted surface, and zero unexplained tracked drift in `frontend/bindings`
- [ ] T072 Run locked clean installs and production builds for `frontend/` and `client/`, then `scripts/reproducible-build-check.sh` and bundle scans for zero CDN/runtime download, v2 global, private generated JavaScript, privileged fallback, stale binding, or root-module drift in `frontend/dist/` and `client/dist/`
- [ ] T073 Run `gofmt -l .`, `go vet ./...`, and `go test ./...`, fixing only migration-caused failures in `main.go`, `app.go`, `wails_host.go`, `desktop_service.go`, and affected packages under `internal/`
- [ ] T074 Run `go test -race ./...` and resolve migration-caused lifecycle, stream, session-worker, event, or process ownership races in `app.go`, `wails_host.go`, `internal/player/`, `internal/session/`, `internal/tunnel/`, and `internal/platform/`
- [ ] T075 Run `npm ci --prefix tests/browser` followed by the complete Playwright suite in `tests/browser/`, preserving all feature-005 multi-client, authority, replay, reconnect, overflow, sound, and privacy assertions
- [ ] T076 Against the unchanged v2-free build candidate, rerun `go tool -modfile=tools/wails/go.mod wails3 dev` launch/stop plus the T024 one-window/persistence, T032 lifecycle-trigger, and T037 local 4–7-player journeys, recording runtime/native results in `specs/006-wails-v3-migration/quickstart.md`
- [ ] T081 Run `scripts/hash-macos-app.sh --self-test`, verify repeated identical output and sensitivity to file, mode, symlink, addition, and removal changes, and leave zero tracked drift in `scripts/hash-macos-app.sh` and `scripts/verify-macos-app.sh`
- [ ] T079 Build the final personal-use macOS arm64 package from the unchanged build candidate, run `scripts/verify-macos-app.sh` and `scripts/wails-v3-cutover-check.sh`, calculate its canonical bundle-manifest SHA-256 with `scripts/hash-macos-app.sh`, repeat offline launch/one-listener/master-player-smoke/clean-quit against `build/bin/Fallout Terminal.app`, verify the same identity afterward, and record package results plus personal-use bundle identity in `specs/006-wails-v3-migration/quickstart.md`
- [ ] T080 Verify the unchanged T079 canonical bundle-manifest SHA-256 with `scripts/hash-macos-app.sh`, execute the complete T067 60-minute local workload and RSS thresholds against `build/bin/Fallout Terminal.app`, verify the same identity afterward, and record local-soak results against that exact personal-use bundle identity in `specs/006-wails-v3-migration/quickstart.md`
- [ ] T077 Against the same unchanged build candidate and with real prerequisites, verify the T079 canonical bundle-manifest SHA-256 with `scripts/hash-macos-app.sh` before and after the complete T067 30-minute authenticated-ngrok workload, then run `scripts/build-macos.sh` Developer ID/notary/staple/DMG/Gatekeeper gates and record a distinct canonical bundle-manifest digest for each Developer ID app plus a distinct file SHA-256 for each DMG/profile; otherwise record each unavailable conditional result as `NOT RUN` in `specs/006-wails-v3-migration/quickstart.md`
- [ ] T078 Verify `README.md` and every frozen tracked path remain identical to `BUILD_CANDIDATE_SHA`, reconcile the immutable build candidate SHA, exact pins, commands, automated/manual results, `NOT RUN` reasons, artifact identities, final-candidate soak, rollback, and cutover status without modifying those frozen paths, update only result/status fields in `specs/006-wails-v3-migration/quickstart.md` and `docs/wails-v3-migration-rollback.md`, then rerun `scripts/wails-v3-cutover-check.sh`; if any frozen path requires correction, commit a new candidate and restart at T070

**Checkpoint**: Every required gate, including clean browser tests, final native journeys, root development launch, package smoke, and local soak, is tied to one clean committed build candidate SHA/pin set and canonical profile-specific artifact identities; conditional evidence is honest; no test or contract assertion was weakened to obtain acceptance. Any frozen tracked-path change during the final sequence invalidates later evidence and restarts Phase 11 at T070 against a new build candidate. Evidence-only result/status changes require the final T078 documentation/cutover rescan but do not redefine or rebuild the unchanged candidate.

---

## Dependencies and Execution Order

### Phase dependencies

```text
Phase 1 Setup / artifact reconciliation
                │
                ▼
Phase 2 isolated pins + host/service/status foundation
                │
       ┌────────┼───────────┬───────────┐
       ▼        ▼           ▼           ▼
     US1      US2         US3         US4
       │        │           │           │
       └────────┴───────────┴─────┬─────┘
                                  ▼
                    Integrated P1 native acceptance
                                  │
                                  ▼
                           US5 build graph
                                  │
                                  ▼
                            US6 package
                                  │
                                  ▼
                      US7 rollback/cutover
                                  │
                                  ▼
                     Cross-cutting final matrix
```

- Phase 1 blocks all implementation because current root tooling predates constitution 3.3.1 and the migration-contract preflight must govern every later slice.
- Phase 2 blocks all stories because every v3 slice needs exact runtime/tool pins, host seams, the unchanged runtime-status contract, and the service inventory.
- US1–US4 implementation and automated-test slices may advance as coordinated P1 work after Phase 2, but tasks touching `main.go`, `app.go`, `wails_host.go`, `desktop_service.go`, or `desktop-api.js` remain sequential.
- T024, T032, and T037 run only after T041–T046 complete, so no final native journey can pass through a v2/global binding fallback or before readiness behavior exists.
- US5 requires the integrated P1 checkpoint; US6 requires the stable build graph; US7 requires all parity and personal-package gates.
- Within US7, T066 rollback and T067 soak must pass before T068 removes v2; T069 then records the provisional v2-free candidate and hands it to the authoritative Phase 11 matrix without accepting it.
- Final cross-cutting verification runs only after the cutover candidate contains no active v2 implementation.

### User-story completion order

| Story | Depends on | Independently complete when |
|---|---|---|
| US1 — native master | Phase 2 implementation; T024 after T041–T046 | one window and representative persistence/native workflows match baseline through the v3 facade |
| US2 — lifecycle ownership | Phase 2 implementation; T032 after T041–T046 | normal/failure/partial/repeat triggers publish actionable status and clean resources within five seconds |
| US3 — player parity | Phase 2 implementation; T037 after T041–T046 | 4–7 generated-player journeys converge and expose no private/Wails/legacy surface through the integrated host |
| US4 — bridge/events | Phase 2 | exact 25 methods/four events pass generation, shape, race, release, and forbidden-surface checks |
| US5 — reproducible development | integrated P1 checkpoint | isolated tools and one ordered root graph produce repeatable clean generations/builds |
| US6 — personal package | US5 | final pre-sign resource inventory and ad-hoc arm64 app pass offline smoke |
| US7 — rollback/cutover | US1–US6 | rollback drill and soak are recorded, then final source scans contain zero active v2/dual path |

## Parallel Opportunities

### Setup/foundation

- T003–T006 can create independent tool modules in parallel after T001 establishes the migration-contract preflight.
- T010, T012, and T014 touch independent build/test files after exact pins are known.

### User Story 1

- T018 window/assets tests and T019 platform adapter tests can run in parallel before T020–T022.

### User Story 2

- T025 host lifecycle tests and T026 core/tunnel tests can run in parallel before lifecycle implementation.

### User Story 3

- T033 Go/source isolation assertions and T034 Playwright expansion can run in parallel.

### User Story 4

- T038 binding checks, T039 browser facade tests, and T040 Go adapter tests can run in parallel before T041–T045.

### User Stories 5–7

- T047 build-graph tests can precede graph implementation while documentation work stays sequential with final commands.
- T055 package tests can run independently before Darwin task changes.
- T062 rollback assertions and T063 cutover scanner can run in parallel; T066 rollback and T067 soak precede T068 final removal, and T069 follows it.

### Final verification

- Final verification executes in file order as T070 → T071 → T072 → T073 → T074 → T075 → T076 → T081 → T079 → T080 → T077 → T078: tool/protobuf state → Wails bindings → locked frontend builds → Go tests → race tests → clean Playwright install/suite → final native journeys → bundle-hash preflight → package/offline smoke → local soak → conditional public soak/release → evidence reconciliation/final documentation scan. Only subcommands explicitly proven to use disjoint immutable inputs may run concurrently within one task.

## Implementation Strategy

### MVP first

1. Complete Phase 1 so tool isolation and design contracts are consistent.
2. Complete Phase 2 so the host, exact service surface, unchanged runtime-status contract, and build skeleton compile.
3. Complete the US1–US4 implementation slices, then run T024, T032, and T037 only at the integrated P1 native-acceptance checkpoint.
4. Do not designate the MVP production-ready; Wails v2 remains fallback until every P1, package, soak, and rollback gate passes.

### Incremental delivery

1. Prove master behavior (US1), lifecycle ownership (US2), player parity (US3), and bridge/events (US4) as attributable P1 checkpoints.
2. Freeze those behaviors before completing deterministic build orchestration (US5).
3. Package and inspect the personal-use app (US6) without invoking unavailable public trust gates.
4. Perform rollback/soak, remove v2, rebuild, and cut over only after the complete required matrix passes (US7).

## Notes

- Constitution 3.3.1 overrides every older `go install`, global binary, root `tool` block, or direct `wails3` example in feature artifacts.
- Internal lifecycle phase remains Go-only and test-observable; it MUST NOT add a protobuf field, change the detached native runtime-status shape, or alter domain, public player, session-v1, or player-config-v1 models.
- Do not opportunistically upgrade the Wails v2 fallback before it is recorded and retired.
- Do not edit generated protobuf or Wails binding output manually.
- Do not remove `production_resources_bindings.go` until clean v3 evidence proves it obsolete.
- Do not mutate an application bundle after its final signature.
- Do not claim formatting, vet, test, race, browser, CI, packaging, soak, rollback, ngrok, or public-release success unless the command or journey actually ran.
- Record unavailable credentials, connectivity, native UI access, or release services as `NOT RUN`, never `PASS`.
