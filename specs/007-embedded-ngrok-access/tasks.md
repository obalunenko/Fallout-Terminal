# Tasks: Embedded ngrok Public Access

**Input artifacts**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, and
`quickstart.md` in `specs/007-embedded-ngrok-access/`  
**Required profile**: macOS 13+ Apple Silicon (`arm64`)  
**Method**: tests are authored to fail before the production task they govern. Do not weaken a
failing security, protobuf, race, reproducibility, or packaging gate to make it pass.

**Bugfix**: 2026-08-15 — ANALYZE-S1/U1 adds test-first source-bound public-ingress work and an
executable user-approved checkpoint/worktree rollback gate.

**Bugfix**: 2026-08-15 — BUG-001 supersedes the source/Host portions of completed T010–T012,
T029–T033, and T035–T036 after the target Darwin host rejected `127.0.0.2`; T034 remains active.
Corrective pending tasks use ngrok endpoint Basic Auth and the ordinary direct upstream to the
existing player server.

**Bugfix**: 2026-08-15 — BUG-001 follow-up assigns separate RED/GREEN ownership to T037–T039 and
makes the exact final constitution command sequence explicit without adding or reopening tasks.

**Bugfix**: 2026-08-15 — BUG-001 analysis follow-up makes final gate order clean-checkout safe,
adds canonical dev-smoke ownership, and removes historical evidence from active coverage mappings
and current-guidance prose.

**Bugfix**: 2026-08-16 — BUG-002 adds a test-first correction for the embedded SDK forwarder's
context lifetime and safe unexpected-disconnect diagnostics. The corrective tasks preserve the
completed historical task record and are checked only from actual RED/GREEN/gate evidence.

Every completed task must be journaled through the Companion writer. Do not edit completed feature
artifacts under `specs/005-connectrpc-protobuf-migration/` or `specs/006-wails-v3-migration/`;
their records remain historical. Direct `go run ./cmd/build dev|build|package` commands remain
canonical throughout.

## Requested Increment Mapping

- **P1 — secure packaged setup**: Phase 3 / US1 (UI, Application Support, Keychain, no readback).
- **P2 — embedded endpoint and local continuity**: Phase 4 / US2 establishes start/stop and
  random/reserved URLs; Phase 7 / US5 proves local/LAN survival across provider and storage failures.
- **P3 — protected player parity**: Phase 5 / US3 proves fail-closed Basic Auth plus streaming,
  reconnect, multi-client, and gameplay behavior.
- **P4 — safe lifecycle**: Phase 6 / US4 and Phase 8 / US6 cover reconfigure, failures, restart,
  Quit/`Cmd+Q`, stale completions, and bounded cleanup.
- **P5 — single-runtime cutover**: Phase 9 removes the CLI path only after embedded parity and then
  qualifies documentation, reproducibility, packaging, and conditional real-service evidence.

## Phase 1: Setup — dependency and license ownership

**Purpose**: Establish exact module/license expectations before adding runtime code.

**Wave 1 — test first:**

- [x] **T001** Add a deterministic dependency/license drift gate that expects exact ngrok SDK and Keychain pins, enumerates the resolved module graph, rejects floating versions, and reports missing notices without downloading at packaged runtime · `scripts/dependency-license-check.sh`

**⟶ Wait for T001 to fail on the absent pins, then:**

- [x] **T002** Pin `golang.ngrok.com/ngrok/v2 v2.1.4` and `github.com/keybase/go-keychain v0.0.1`, accept only reviewed exact transitive versions/checksums, and avoid upgrading existing Wails/ConnectRPC/protobuf pins · `go.mod`, `go.sum`

**⟶ Wait for T002, then:**

- [x] **T003** Record verified MIT notices for ngrok-go and go-keychain plus the actual detected license and required notice for every resolved new transitive runtime module so T001 passes with a reviewable offline inventory · `THIRD_PARTY_NOTICES.md`

---

## Phase 2: Foundational — protobuf contracts, native model, and deterministic fakes

**Purpose**: Create shared contracts and test seams that block every user story.

**Wave 1 — test first:**

- [x] **T004** Extend descriptor/adapter tests to require the exact secret-free public-access preferences, private lifecycle/presence/error enums, mutation `oneof`s, one-time generated-password result, `public-access-status` event, public-player isolation, and reserved legacy tunnel fields · `app_contract_test.go`

**⟶ Wait for T004 to fail, then implement schemas sequentially:**

- [x] **T005** Add version-1 non-secret `PublicAccessPreferences` with enabled preference, optional reserved domain, username, presence hints, and revision exactly as designed · `proto/fallout/terminal/config/v1/public_access.proto`
- [x] **T006** Add the exact private public-access snapshot, status, commands, error categories, secret mutation inputs, and one-time generated-password result · `proto/fallout/terminal/private/v1/public_access.proto`
- [x] **T007** Remove active plaintext/process tunnel configuration by reserving old field numbers/names and documenting the internal config cutover without touching session/player-config v1 contracts · `proto/fallout/terminal/config/v1/config.proto`
- [x] **T008** Regenerate exact Go descriptors, advance the schema revision, update the reviewed compatibility baseline once, and prove format/lint/drift/breaking gates remain enabled · `internal/gen/fallout/terminal/config/v1/public_access.pb.go`, `internal/gen/fallout/terminal/config/v1/config.pb.go`, `internal/gen/fallout/terminal/private/v1/public_access.pb.go`, `proto/schema-revision.txt`, `proto/compatibility-baseline.binpb`

**⟶ Wait for generated contracts and T004 to pass, then test native boundaries:**

- [x] **T009** Add failing table tests for native preferences/status validation, lifecycle states, error taxonomy, URL/domain normalization, secret refs/presence, revision/generation monotonicity, and absence of secret-capable string formatting · `internal/tunnel/model_test.go`

**⟶ Wait for T009 to fail, then:**

- [x] **T010** **Superseded in part — BUG-001; corrected by T037.** Implement provider-neutral native models plus `SecretStore`, `TunnelService`, `TunnelEndpoint`, ~~public-policy~~, clock, filesystem, and event callback interfaces without importing Wails, Keychain, or ngrok SDK types; the former policy interface is historical · `internal/tunnel/model.go`, `internal/tunnel/secret.go`, `internal/tunnel/service.go`

**⟶ Wait for T010, then test deterministic seams:**

- [x] **T011** **Superseded in part — BUG-001; corrected by T037.** Add failing self-tests for controllable secret, endpoint, ~~policy~~, clock, filesystem, and event fakes, including delayed completion, `Done`, close failure, active endpoint count, and secret-buffer clearing observations; the former policy-fake coverage is historical · `internal/testutil/public_access_fakes_test.go`

**⟶ Wait for T011 to fail, then:**

- [x] **T012** **Superseded in part — BUG-001; corrected by T037.** Implement the deterministic public-access fakes used by all ordinary unit/race/integration tests so they never contact Keychain or the network; the former player-policy fake is historical · `internal/testutil/public_access_fakes.go`

**Foundational checkpoint**: protobuf contracts generate deterministically, public player descriptors
remain secret-free, and all later stories can use provider/store/~~policy~~ **BUG-001** lifecycle
fakes without network access.

---

## Phase 3: User Story 1 — Save public-access settings securely (Priority P1)

**Goal**: A packaged user configures non-secret preferences and two independent Keychain secrets
through the UI, sees presence only, and can generate/copy a password once without Terminal or leaks.

**Independent Test**: Save, replace, delete, quit, and relaunch with canary credentials; preferences
and presence return, secret values never do, and locked/denied/unavailable Keychain creates no
plaintext fallback.

### Tests — Wave 1, independent and written before production

- [x] **T013** [P] [US1] Add failing tests for distinct token/password refs, scoped use, replacement/deletion, manual password minimum 8 with no composition rules, cryptographic generation with at least 128 bits, one-time result, buffer cleanup, and redacted failures · `internal/tunnel/secret_test.go`
- [x] **T014** [P] [US1] Add failing tests for version-1 Application Support JSON, presentation-only enabled preference with no auto-start effect, `players` default, optional domain, presence hints, `0700`/`0600`, exclusive temp+sync+rename, failure cleanup, unknown/corrupt-version quarantine, and zero session/player-config writes · `internal/tunnel/settings_test.go`, `internal/platform/paths_test.go`
- [x] **T015** [P] [US1] Add failing Keychain adapter and opt-in Darwin integration tests for stable dev/prod services, fixed token/password accounts, attribute-only presence, update/add/delete/not-found, locked/denied/unavailable/user-cancelled mappings, signing namespace isolation, and secret-free errors · `internal/platform/keychain_test.go`, `internal/platform/keychain_darwin_integration_test.go`
- [x] **T016** [P] [US1] Add failing private bridge/facade tests for exact five methods, typed event, event-before-snapshot ordering, mutation non-echo, `UNKNOWN` presence, one-time generated result, no saved-secret readback, and exact disposal · `app_contract_test.go`, `tests/browser/desktop-api.spec.mjs`
- [x] **T017** [P] [US1] Add failing UI/accessibility tests for labelled password inputs, default `players`, optional domain, presentation-only enabled preference, presence-only display, Generate/Copy once, replace/delete, no Reveal, input/result clearing, live regions, keyboard focus, relaunch restoration, and zero endpoint auto-start · `internal/platform/assets_test.go`, `tests/browser/public-access-settings.spec.mjs`
- [x] **T018** [P] [US1] Add a canary leak gate covering protobuf descriptors/results, ordinary config, Application Support, session/player-config fixtures, named events, frontend storage/bundle, errors/logs/diagnostics, process arguments, and packaged resources · `scripts/secret-leak-check.sh`

**⟶ Wait for all US1 tests to fail for the intended missing behavior, then implement independent stores:**

### Implementation — Wave 2, independent (different files)

- [x] **T019** [P] [US1] Implement secret validation, fixed refs, scoped-use helpers, constant-time-safe byte handling, crypto/rand password generation, and zero/minimal-lifetime cleanup without a general read/export API · `internal/tunnel/secret.go`
- [x] **T020** [P] [US1] Implement the atomic explicit protobuf↔versioned-JSON settings adapter, safe defaults, private corruption quarantine, presence-hint reconciliation inputs, and no session/player-config coupling · `internal/tunnel/settings.go`, `internal/platform/paths.go`
- [x] **T021** [P] [US1] Implement the native Security.framework Keychain adapter using pinned go-keychain, isolated `com.vaulttec.fallout-terminal[.dev].public-access` services, fixed accounts, metadata-only presence, scoped reads, replace/delete, and redacted OSStatus categories · `internal/platform/keychain.go`, `internal/platform/keychain_darwin.go`

**⟶ Wait for Wave 2, then integrate the trusted desktop boundary sequentially:**

- [x] **T022** [US1] Implement explicit protobuf/native adapters, secret-free snapshots/events/results, five trusted master methods, generated-password direct result, and settings-only lifecycle behavior without an endpoint · `app_contract.go`, `app.go`, `desktop_service.go`, `wails_host.go`
- [x] **T023** [US1] Compose the Application Support settings store and production Keychain adapter while loading saved preferences into disabled state only—never auto-starting public access · `main.go`
- [x] **T024** [US1] Regenerate Wails v3 method/event bindings and update the exact allowlist without exposing lifecycle, provider, environment, process, Keychain, or generic dispatch operations · `frontend/bindings/github.com/obalunenko/Fallout-Terminal/desktopservice.js`, `frontend/bindings/github.com/wailsapp/wails/v3/internal/eventdata.d.ts`, `scripts/wails-bindings-check.sh`

**⟶ Wait for T024, then implement independent frontend surfaces:**

- [x] **T025** [P] [US1] Add explicit facade normalization and fixture bindings for the five methods plus `public-access-status`, preserving event-wins-over-stale-snapshot and clearing native secret arguments/results · `frontend/src/desktop-api.js`, `tests/browser/fixtures/desktop-bindings.js`
- [x] **T026** [P] [US1] Add accessible Fallout-style public-access form, presence indicators, secret inputs, one-time password dialog, live status/error regions, and responsive styles without a Reveal control · `frontend/src/index.html`, `frontend/src/master.css`

**⟶ Wait for T025–T026, then:**

- [x] **T027** [US1] Implement settings load/save, presentation-only enabled preference, replace/delete, Generate, transient manual/generated Copy flows, revision conflict handling, transition disabling, focus restoration, zero auto-start behavior, and immediate DOM/module reference clearing · `frontend/src/master.js`
- [x] **T028** [US1] Run the focused Go, browser, leak, restart, and opt-in Keychain tests and record honest US1 evidence without credential values · `specs/007-embedded-ngrok-access/quickstart.md`

**Checkpoint US1**: Settings and secrets are independently usable from packaged UI; both production
secrets exist only in Keychain, stored values cannot be revealed, and the one-time generated result
is the sole bounded exception.

---

## Phase 4: User Story 2 — Start and share protected embedded access (Priority P1)

**Goal**: Start/stop one embedded ngrok endpoint from the running UI, obtain random or exact reserved
HTTPS URL, and publish it only after the endpoint's Basic Auth Traffic Policy is active.

**Independent Test**: With valid settings, observe disabled→starting→ready, verify no request succeeds
before protected endpoint readiness, copy URL/username, stop, and keep local mode working.

### Tests — sequential because they govern shared lifecycle/player-auth surfaces

**Wave 1 — test first:**

- [x] **T029** [US2] **Superseded in part — BUG-001; corrected by T038.** Add state-machine tests for disabled/starting/ready/stopping/failed, generation/revision guards, 15s target/30s bound, repeated Start/Stop, cancellation, private URL until readiness, stale completion close, maximum one endpoint, and redacted status/events; former player-policy activation expectations are historical · `internal/tunnel/manager_test.go`
- [x] **T030** [US2] **Superseded historical evidence — BUG-001; corrected by T037.** Add failing SDK adapter and real-loopback dialer tests for the former dedicated-source design · `internal/tunnel/ngrok_test.go`, `internal/tunnel/upstream_dialer_test.go`
- [x] **T031** [US2] **Superseded historical evidence — BUG-001; corrected by T037/T041/T046.** Add the former source/Host-oriented opt-in SDK harness · `internal/tunnel/ngrok_integration_test.go`, `specs/007-embedded-ngrok-access/research.md`
- [x] **T032** [US2] **Superseded historical evidence — BUG-001; removed by T037.** Add the former player source/Host admission tests · `internal/player/public_access_test.go`
- [x] **T033** [US2] **Superseded in part — BUG-001; corrected by T039.** Add application lifecycle tests; source sharing and player-policy ordering assertions are historical, while local-readiness/private-publication behavior remains required · `app_test.go`
- [x] **T034** [US2] Extend browser tests first for UI Start/Stop, exact `disabled→stopped` and `failed→error` presentation, random/reserved URL state, redacted domain errors, URL/username Copy without saved password, disabled transition actions, and no pre-ready URL · `tests/browser/public-access-settings.spec.mjs`

**⟶ Wait for all US2 tests to fail for intended gaps, then implement security boundary before provider publication:**

### Implementation

- [x] **T035** [US2] **Superseded historical implementation — BUG-001; removed by T037.** Implement the former player source/Host policy · `internal/player/public_access.go`
- [x] **T036** [US2] **Superseded historical implementation — BUG-001; removed by T037.** Inject the former policy into player routing · `internal/player/http.go`, `internal/player/server.go`
- [x] **T037** [US2] [BUG-001] Test-first replace the source-bound/player-policy design with the pinned ngrok Agent/Forward adapter. First rewrite every former policy-dependent adapter, model, lifecycle-fake, manager, composition, and affected player test to the endpoint contract. Register `internal/tunnel/model_test.go`, `internal/tunnel/manager_test.go`, `internal/tunnel/ngrok_test.go`, `internal/tunnel/ngrok_integration_test.go`, `internal/testutil/public_access_fakes_test.go`, and affected `internal/player/*_test.go` files in the convention inventory with Testify, `t.Context()`, and protobuf-aware comparisons where applicable. Confirm the T037-owned adapter/model/fake/player subset is RED for the intended endpoint-policy gaps; then add scoped token/username/password endpoint input, direct upstream `http://127.0.0.1:3690`, and in-memory Basic Auth Traffic Policy and make that subset GREEN. Corrected manager and root-composition suites remain explicitly expected RED for T038 and T039 and do not block T037. Remove `ForwarderSource`, `PublicAccessPolicy`, `PublicAccessGrant`, `upstream_dialer*`, and player-policy injection only after the T037-owned subset passes and active tests contain no old policy references. · `internal/tunnel/service.go`, `internal/tunnel/model.go`, `internal/tunnel/model_test.go`, `internal/tunnel/manager_test.go`, `internal/tunnel/ngrok.go`, `internal/tunnel/ngrok_test.go`, `internal/tunnel/ngrok_integration_test.go`, `internal/tunnel/upstream_dialer.go`, `internal/tunnel/upstream_dialer_test.go`, `internal/testutil/public_access_fakes.go`, `internal/testutil/public_access_fakes_test.go`, `internal/player/public_access.go`, `internal/player/public_access_test.go`, `internal/player/http.go`, `internal/player/server.go`, `app_test.go`, `internal/platform/test_conventions_test.go`
- [x] **T038** [US2] Take the corrected manager suite produced by T037 from expected RED to GREEN by implementing the generation-aware manager and exact acquire-protected-endpoint→validate→publish / withdraw→close sequences without locks across store/network/event calls or a separate player-policy mutation · `internal/tunnel/manager.go`, `internal/tunnel/manager_test.go`
- [x] **T039** [US2] Take the corrected root-composition suite produced by T037 from expected RED to GREEN by routing Start/Stop/status exclusively through the embedded core and composition root, making legacy env/argument/process helpers and superseded player-policy composition unreachable from production while retaining CLI source only for bounded rollback until T075, using the existing port 3690 player server, keeping startup disabled until explicit UI intent, and preserving safe `server-info` local/public compatibility · `app.go`, `app_test.go`, `main.go`
- [x] **T040** [US2] Complete secret-free status/event adapters and UI rendering with explicit `disabled→stopped` and `failed→error` mappings plus starting/ready/stopping, random/reserved outcomes, URL/username Copy, redacted errors, and current generation/revision · `app_contract.go`, `frontend/src/desktop-api.js`, `frontend/src/master.js`
- [x] **T041** [US2] Run focused manager/SDK/Traffic-Policy/app/browser tests plus opt-in real startup where available; record direct upstream, policy-before-publication, timing, random/reserved behavior, local fallback, and separate `PASS`/`FAIL`/`NOT RUN`, never presenting conditional `NOT RUN` as real endpoint proof · `specs/007-embedded-ngrok-access/quickstart.md`

**Checkpoint US2**: One embedded endpoint can start/stop from UI, but ready URL is impossible before
endpoint Basic Auth protection; no SDK/Keychain/network failure breaks local play.

---

## Phase 5: User Story 3 — Full authenticated remote player parity (Priority P1)

**Goal**: Public players authenticate for every resource/RPC and retain non-empty ConnectRPC
streaming, multi-client convergence, gameplay, sound, and reconnect behavior.

**Independent Test**: Through the protected origin, exercise static assets, all unary procedures,
non-empty `Subscribe`, four-to-seven clients, gameplay updates, disconnection, and ≤5s reconnect
convergence while wrong/missing credentials are rejected by the ngrok endpoint.

### Tests — sequential shared player-auth surface

**Wave 1 — test first:**

- [x] **T042** [US3] Extend provider-adapter and unchanged player HTTP/Connect tests for policy-protected static/unary/streaming intent, exact direct upstream, missing/wrong/correct edge-auth outcomes in the external seam, same-Origin/limits ordering after forwarding, local/LAN no-challenge continuity, and snapshot plus later update before stream completion · `internal/tunnel/ngrok_test.go`, `internal/player/http_test.go`, `internal/player/public_stream_test.go`
- [x] **T043** [US3] Extend Playwright first for protected static/unary/streaming paths, character selection, navigation, hacking, sound, four-to-seven client convergence, non-empty updates, reconnect, stale URL, and explicit real-endpoint `NOT RUN` semantics · `tests/browser/connectrpc-player.spec.mjs`

**⟶ Wait for T042–T043 to fail for intended gaps, then:**

### Implementation

- [x] **T044** [US3] Make the deterministic browser fixture emulate endpoint Basic Auth before its unchanged player routes and expose controlled stream/update/reconnect probes without claiming real endpoint evidence · `tests/browser/fixture-server/main.go`
- [x] **T045** [US3] Fix only demonstrated stream/reconnect gaps while keeping current Connect request limits, server-authoritative state, same-origin client, and three-second reconnect loop; do not add application Host/source authentication · `internal/player/http.go`, `client/client.js`
- [x] **T046** [US3] Run focused Go/browser multi-client journeys and one credential-gated real missing/wrong/correct Basic Auth plus non-empty incremental stream check when available; record fake versus real evidence separately and use `NOT RUN` otherwise · `specs/007-embedded-ngrok-access/quickstart.md`

**Checkpoint US3**: The complete remote player experience is protected and behaviorally equivalent
to local play; the deterministic fixture is never reported as external reachability proof.

---

## Phase 6: User Story 4 — Stop and reconfigure safely (Priority P2)

**Goal**: Replace/delete credentials or change settings while active with old acceptance disabled
before one replacement starts; concurrent intents end consistently with no stale URL or duplicate.

**Independent Test**: Rotate each field/secret during starting and ready states across 100 schedules,
deliver late completions, and prove latest valid intent, zero overlap, and safe recovery from partial
durable mutation failures.

### Tests — sequential shared lifecycle/UI files

**Wave 1 — test first:**

- [x] **T047** [US4] Add 100-schedule race tests for concurrent/repeated Start/Stop/Reconfigure, settings revisions, old endpoint withdrawal/close before replacement publication, no old/new overlap, late success/failure rejection, close failure/retry, and latest-valid-intent convergence · `internal/tunnel/manager_test.go`
- [x] **T048** [US4] Add core integration tests for active token/password/domain/username changes, generated-password rotation, delete blocking Start, partial Keychain/file mutation reconciliation, event order, and no mixed revision restart · `app_test.go`
- [x] **T049** [US4] Extend browser tests first for active edits, confirmation/error states, stopped→starting replacement, stale event/result rejection, replacement/delete presence, transition-disabled actions, and no saved-secret reconstruction · `tests/browser/public-access-settings.spec.mjs`
- [x] **T050** [US4] Add long-diagnostic and canary redaction tests covering SDK codes, token/password/username, account/domain data, truncation, concurrent errors, direct results, status/events, and retry paths · `internal/tunnel/redaction_test.go`

**⟶ Wait for US4 tests to fail, then:**

### Implementation

- [x] **T051** [US4] Implement protected reconfigure and durable-mutation reconciliation in the manager: advance generation, ~~deny/withdraw/close old endpoint~~ **BUG-001** withdraw the URL and close the old endpoint/Agent, apply Keychain/file changes, re-query presence, validate one revision, and start at most one replacement · `internal/tunnel/manager.go`
- [x] **T052** [US4] Implement stable redacted provider/domain/auth/network/timeout/close error mapping and bounded diagnostics without retaining SDK error text or credentials · `internal/tunnel/ngrok.go`, `internal/tunnel/redaction.go`
- [x] **T053** [US4] Integrate active settings mutation, revision conflicts, stale UI completion rejection, replace/delete/generate actions, and protected restart status without persisting secrets in frontend state · `app.go`, `frontend/src/desktop-api.js`, `frontend/src/master.js`
- [x] **T054** [US4] Run focused race/core/browser/leak reconfiguration tests and record 100-run latest-intent, zero-overlap, stale URL, partial failure, and recovery evidence · `specs/007-embedded-ngrok-access/quickstart.md`

**Checkpoint US4**: Reconfigure is a safe stop/change/start transaction at the public boundary even
though Keychain and filesystem writes are individually durable; incomplete revisions never restart.

---

## Phase 7: User Story 5 — Preserve local/LAN play through public failures (Priority P2)

**Goal**: Provider, account, domain, network, timeout, Keychain, policy, and unexpected endpoint
failures affect only public status; local gameplay remains live and recovery needs no app restart.

**Independent Test**: Keep local clients active while injecting every public failure, including
post-ready `Done`; verify local streaming/gameplay/reconnect and successful later public restart.

### Tests — written before failure handling

**Wave 1 — test first:**

- [x] **T055** [US5] Add failure-matrix tests for invalid/revoked token, no network, DNS/timeout, domain conflict, Keychain locked/denied/unavailable, endpoint-policy construction/provider failure, unexpected `Done`, close failure, local server status, and retry without restart · `app_test.go`, `internal/tunnel/ngrok_test.go`
- [x] **T056** [US5] Add browser journeys that keep local/LAN streams, selection, navigation, hacking, sound, and reconnect active during each public failure and after recovery · `tests/browser/public-access-fallback.spec.mjs`

**⟶ Wait for T055–T056 to fail, then:**

### Implementation

- [x] **T057** [US5] Handle current unexpected endpoint completion and provider/store failures by ~~atomically denying public Host~~ **BUG-001** withdrawing the URL and closing the endpoint/Agent, bounded cleanup, redacted failed state, and restartable new generation · `internal/tunnel/manager.go`
- [x] **T058** [US5] Preserve local `server-info`, player lifecycle, and master recovery presentation independently of tunnel status while keeping public failures nonfatal · `app.go`, `frontend/src/master.js`
- [x] **T059** [US5] Add deterministic fixture controls for provider disconnect, network timeout, store failure, stale completion, and local-client continuity without introducing a second runtime path · `tests/browser/fixture-server/main.go`
- [x] **T060** [US5] Run the full deterministic failure matrix plus available real invalid-token/domain/offline cases and record local continuity, redaction, recovery, and conditional `NOT RUN` outcomes · `specs/007-embedded-ngrok-access/quickstart.md`

**Checkpoint US5**: Local/LAN mode is demonstrably independent of every optional public-access
dependency and can recover public mode without restarting Fallout Terminal.

---

## Phase 8: User Story 6 — Exit the packaged application cleanly (Priority P2)

**Goal**: Quit, `Cmd+Q`, repeated shutdown, partial startup, and crash leave no usable public URL or
owned endpoint/resource; graceful cleanup fits the existing single five-second budget.

**Independent Test**: Exit from every lifecycle state, repeat 100 schedules, force a process loss,
and verify ~~deny-before-close~~ **BUG-001** URL-withdrawal-before-endpoint-close, bounded cleanup,
old URL failure, and stopped next launch.

### Tests — sequential shared lifecycle surface

**Wave 1 — test first:**

- [x] **T061** [US6] Add manager race tests for shutdown during every state/boundary, repeated shutdown joining one deadline, stale post-cancel acquisition, blocked/erroring Close, `Done` races, goroutine/secret cleanup, and zero active endpoint after 100 schedules · `internal/tunnel/manager_test.go`
- [x] **T062** [US6] Extend core/Wails tests for URL withdrawal first, endpoint/Agent close second, then player/session/desktop cleanup; fresh five-second context, partial startup, normal close, `Cmd+Q`, repeated shutdown, and stopped relaunch · `app_test.go`, `wails_host_test.go`
- [x] **T063** [US6] Extend SDK adapter tests for concurrent idempotent Close, Close-before-Start-completion, Close-after-`Done`, cancellation not counting as cleanup, Agent disconnect, retained ownership on failure, and no goroutine/reference leak · `internal/tunnel/ngrok_test.go`
- [x] **T064** [US6] Add a redaction-safe macOS packaged lifecycle harness for double-click launch, close, `Cmd+Q`, partial startup, forced owner loss, stale URL probe, next-launch stopped state, and five-second timing · `scripts/public-access-macos-smoke.sh`

**⟶ Wait for US6 tests/harness assertions to fail, then:**

### Implementation

- [x] **T065** [US6] Implement manager/SDK shutdown joining, ~~deny/withdraw-before-Close~~ **BUG-001** URL-withdrawal-before-endpoint-Close, explicit bounded Close+Disconnect, stale acquisition disposal, retry ownership, monitor cancellation, and secret/reference cleanup · `internal/tunnel/manager.go`, `internal/tunnel/ngrok.go`
- [x] **T066** [US6] Put public manager shutdown before existing player/session/desktop release while preserving fresh five-second Wails ownership, idempotence, error joining, and later cleanup after earlier failure · `app.go`, `wails_host.go`
- [x] **T067** [US6] Run unit/race lifecycle schedules and packaged close/`Cmd+Q`/owner-loss/relaunch journeys, then record exact timing, resource counts, stale URL outcome, and any external `NOT RUN` evidence · `specs/007-embedded-ngrok-access/quickstart.md`

**Checkpoint US6**: Graceful cleanup is within five seconds in every state, repeated shutdown does
not extend the budget, and unexpected process loss leaves the prior URL unusable and next launch
disabled.

---

## Phase 9: P5 Cutover, packaging, documentation, and final qualification

**Purpose**: Build and pass the full embedded-only package and rollback gates before deleting CLI
source, then rerun the gates on the final single-runtime tree without weakening conditional evidence.

### Pre-removal gates — tests and gate implementations must precede deletion

**Wave 1 — sequential gate construction:**

- [x] **T068** Add a diagnostic active-runtime scan that rejects `NGROK_BIN`, external ngrok execution/PATH lookup, guardian/process runner, log URL parser, hard-coded default domain, env/argument packaged launch, bundled provider binary, or a second production mechanism, including Makefile and active docs; before removal it may identify known legacy files, but after removal it MUST pass · `scripts/legacy-public-access-check.sh`
- [x] **T069** Extend package-plan tests first to require third-party notices, exact SDK graph inputs, no provider executable/resource, offline local launch assets, arm64/macOS13 identity, and unchanged protobuf→player→bindings→master→native/package ownership · `internal/buildtool/buildtool_test.go`
- [x] **T070** Implement package notice inclusion and verification of no external ngrok binary/PATH/runtime download while preserving arm64, minimum OS, native framework linkage, entitlements, signing, offline resources, canonical hash, and buildtool ownership · `internal/buildtool/buildtool.go`, `THIRD_PARTY_NOTICES.md`, `scripts/verify-macos-app.sh`
- [x] **T071** Extend reproducibility and CI gates for exact module/license drift, secret scan, legacy diagnostics, race/browser suites, two-build equality, unsigned package inspection, and credential-free deterministic CI; do not make real ngrok or signing credentials mandatory · `scripts/reproducible-build-check.sh`, `.github/workflows/wails-macos.yml`
- [x] **T072** Expand the leak gate with long canaries across errors/logs/events/protobuf/config/Application Support/session/player-config/args/fixtures/frontend/package and prove generated password occurs only in its direct one-time result/presentation · `scripts/secret-leak-check.sh`

**⟶ Wait for T068–T072, then prove the embedded-only candidate before any deletion:**

- [x] **T073** Run and record contracts/fakes/unit/race/HTTP/non-empty-stream/browser/lifecycle/dependency/license/leak/reproducible-build/full-package/offline-double-click parity against embedded-only composition; block removal unless tests prove the fixed direct `http://127.0.0.1:3690` upstream, Basic Auth Traffic Policy attached before URL publication, secrets Keychain-only with only scoped SDK policy use, local fallback, no provider binary/PATH/download, and cleanup within five seconds · `specs/007-embedded-ngrok-access/quickstart.md`
- [x] **T074** After T073 PASS and only after a user-approved checkpoint commit exists, require a clean candidate, record its full 40-hex SHA and T073 package digest, create a detached task-owned temporary worktree at exactly that SHA, run canonical `go run ./cmd/build build` and `go run ./cmd/build package`, verify and hash the packaged app, require its digest to equal the T073 digest byte-for-byte, then remove only the validated worktree and record cleanup; otherwise mark `BLOCKED` and forbid T075 without creating a dual-runtime switch · `specs/007-embedded-ngrok-access/quickstart.md`

**⟶ CLI deletion is blocked until T073 and T074 PASS, then cut over sequentially:**

- [x] **T075** Delete the already unreachable startup env/argument helpers, inline credentials, hard-coded domain, external process selection, and configuration-error shim plus CLI config/process/guardian/log-parser implementations and their process-only tests only after the full embedded-only package and rollback gates pass · `main.go`, `internal/tunnel/config.go`, `internal/tunnel/process.go`, `internal/tunnel/process_darwin.go`, `internal/tunnel/process_other.go`, `internal/tunnel/process_test.go`, `internal/tunnel/process_darwin_integration_test.go`
- [x] **T076** Rewrite surviving tunnel tests around SDK/fake timeout, cancellation, `Done`, diagnostics redaction, concurrent close, and lifecycle parity; reconcile the complete surviving-test inventory after process deletion, remove deleted process-test entries, and rerun Testify, `t.Context()`, and protobuf-aware convention enforcement so no process assertion remains · `internal/tunnel/service_test.go`, `internal/platform/test_conventions_test.go`
- [x] **T077** Replace active CLI/PATH/env/default-domain instructions with packaged UI, personal authtoken, Keychain presence/non-readback, random/reserved URL, Start/Stop/sharing/recovery guidance; provider-neutralize active templates and ensure Makefile contains at most thin canonical aliases · `README.md`, `.specify/templates/plan-template.md`, `.specify/templates/spec-template.md`, `.specify/templates/tasks-template.md`, `Makefile`

**⟶ Wait for T075–T077, then require the legacy scan and every deterministic gate to pass again:**

- [x] **T078** On the final post-CLI-removal tree, run the exact clean-checkout-safe deterministic command sequence in `quickstart.md`, including locked npm installs before dependent protobuf generation, empty `gofmt -l .` output, protobuf/binding gates, vet, unit/race, clean locked `frontend/` and `client/` builds, full Playwright, reproducibility, direct `go run ./cmd/build build`, package build/verification, offline double-click local fallback, architecture/signature/resource, and five-second lifecycle gates; then run the separate bounded canonical `go run ./cmd/build dev` master/player smoke without a separately started frontend or player server. Require zero CLI source/runtime/documentation path and attach the final deterministic candidate digest. T074's pre-removal package does not satisfy this task · `specs/007-embedded-ngrok-access/quickstart.md`

**⟶ Only after T078 PASS, run conditional external evidence sequentially:**

- [x] **T079** Run real credential-gated random URL, reserved domain, invalid/revoked token, missing/wrong/correct Basic Auth, static/unary/non-empty incremental `Subscribe`, multi-client/reconnect, stop/reconfigure, failure, and stale URL journeys; record each unavailable prerequisite as `NOT RUN` and never call it deterministic endpoint proof · `specs/007-embedded-ngrok-access/quickstart.md`
- [x] **T080** Run packaged `.app` double-click configuration/Keychain/Start/authenticated-player/Stop/Quit smoke with no Terminal, env/args, installed ngrok executable, or PATH dependency, plus unconditional offline local fallback/architecture/signature/resource checks; record public portion `NOT RUN` without credentials · `specs/007-embedded-ngrok-access/quickstart.md`
- [x] **T081** Run Developer ID, hardened runtime, notarization, stapling, DMG, Gatekeeper, and provider-plan gates only with real prerequisites and record independent `PASS`/`FAIL`/`NOT RUN` results · `specs/007-embedded-ngrok-access/quickstart.md`

**⟶ Final single-owner validation (no post-implement hook owns it):**

- [x] **T082** Re-run the exact final clean-checkout-safe `quickstart.md` sequence on the post-CLI-removal tree: install locked npm dependencies before dependent protobuf generation; require empty `gofmt -l .` output; run tool-module, protobuf format/lint/drift/breaking, Wails binding allowlist, dependency/license/legacy/leak, vet, full unit and `go test -race ./...` gates; run locked builds in both `frontend/` and `client/`; run the full Playwright gate; run reproducibility, direct `go run ./cmd/build build`, package build/verification, rollback-reference validation, the separate bounded canonical `go run ./cmd/build dev` master/player smoke, and the dated vulnerability review with explicit `PASS`/`FAIL`/`NOT RUN`; then reconcile every SC without claiming skipped evidence · `specs/007-embedded-ngrok-access/quickstart.md`
- [x] **T083** Perform the final source/generated/package/documentation scan, verify only one embedded production path remains, attach final candidate digests and FR/SC evidence, and mark every gap explicitly rather than declaring incomplete external gates passed · `specs/007-embedded-ngrok-access/quickstart.md`

**Final checkpoint**: Feature 007 has exactly one embedded production runtime, no CLI/PATH
dependency, Keychain-only secrets, no unprotected public window, complete player streaming parity,
bounded shutdown, immutable rollback evidence, deterministic pre/post-removal package evidence, and
honest conditional results.

---

## Phase 10: BUG-002 — Preserve the committed embedded endpoint lifetime

**Purpose**: Correct the implementation drift that coupled the acquired ngrok forwarder's ongoing
lifetime to the manager's bounded startup-operation context, while retaining redacted diagnostics
and optional local-first behavior.

- [x] **T084** [US2] [BUG-002] Add the RED SDK-adapter regression whose fake forwarder closes
  `Done` when the exact `Forward` context is canceled; prove a committed endpoint survives normal
  startup-context completion, and extend disconnect/redaction tests so only validated provider codes
  can reach manager status · `internal/tunnel/ngrok_test.go`, `internal/tunnel/manager_test.go`,
  `internal/tunnel/redaction_test.go`
- [x] **T085** [US2] [BUG-002] Implement an endpoint-owned `Forward` context with pre-commit startup
  cancellation, post-validation commit, and Close-owned cancellation; capture provider disconnects
  through a safe failure surface, discard raw SDK diagnostic text, and synchronize the internal
  lifecycle contract · `internal/tunnel/ngrok.go`, `internal/tunnel/manager.go`,
  `internal/tunnel/redaction.go`, `specs/007-embedded-ngrok-access/contracts/tunnel-service.md`
- [x] **T086** [US2] [BUG-002] Run the focused RED/GREEN test, tunnel unit and race suites, tunnel
  vet, canonical `go run ./cmd/build build`, and the secret-leak gate. Record the real credentialed
  provider retry as `NOT RUN` because no explicit opt-in credentials were supplied ·
  `specs/007-embedded-ngrok-access/bugs/BUG-002.md`

**Checkpoint BUG-002**: A successful startup no longer tears down its committed endpoint; early
cancellation still aborts acquisition, explicit lifecycle cleanup remains the sole endpoint owner,
and unexpected disconnect status contains no raw provider or secret-bearing diagnostic.

---

## Coverage Mapping: Functional Requirements → Tasks

| Requirement | Task IDs |
|---|---|
| FR-001 | T016–T028, T034, T039–T041, T077, T080 |
| FR-002 | T013, T016–T019, T022, T027 |
| FR-003 | T014, T029, T034, T037–T041, T079 |
| FR-004 | T013, T017, T019, T026–T027 |
| FR-005 | T013, T019, T027 |
| FR-006 | T009, T013, T015, T019, T021–T022 |
| FR-007 | T015, T021, T028, T072, T080 |
| FR-008 | T005, T014, T020, T023, T028 |
| FR-009 | T004, T006, T013–T018, T022, T024–T028, T050, T072 |
| FR-010 | T001–T003, T018, T068–T072, T075–T078 |
| FR-011 | T016–T017, T022, T025, T027 |
| FR-012 | T013, T016–T017, T019, T022, T027 |
| FR-013 | T013, T017, T019, T027 |
| FR-014 | T013, T016–T019, T025, T027 |
| FR-015 | T014–T017, T020–T023, T027–T028 |
| FR-016 | T034, T038–T041, T084–T086 |
| FR-017 | T006, T009, T016–T017, T027, T029, T034, T040, T049, T053, T058 |
| FR-018 | T034, T037, T041, T079 |
| FR-019 | T034, T037, T040–T041, T079 |
| FR-020 | T029, T037–T041, T084–T086 |
| FR-021 | T037, T041–T046, T073, T079 |
| FR-022 | T029, T037–T041, T073, T084–T086 |
| FR-023 | T037, T039, T042, T073, T083 |
| FR-024 | T037, T042–T046, T073, T079 |
| FR-025 | T042–T046, T055–T060, T079 |
| FR-026 | T033, T037–T039 |
| FR-027 | T037, T042, T055–T060, T080 |
| FR-028 | T033, T041, T055–T060, T080 |
| FR-029 | T009, T029, T038, T047, T051, T061, T065, T084–T086 |
| FR-030 | T047–T054 |
| FR-031 | T047–T054 |
| FR-032 | T047–T054, T061–T067, T084–T086 |
| FR-033 | T061–T067, T073, T078, T080 |
| FR-034 | T009, T029, T038, T047, T051, T061, T065, T084–T086 |
| FR-035 | T004–T005, T007–T008, T014, T020, T082 |
| FR-036 | T042–T046, T055–T060, T082 |
| FR-037 | T068–T078, T083 |
| FR-038 | T068, T070–T078, T080, T083 |
| FR-039 | T010–T012, T037, T068–T069, T075 |
| FR-040 | T064, T069–T071, T073, T078, T080 |
| FR-041 | T079–T083 |
| FR-042 | T013–T018, T029, T033–T034, T042–T043, T047–T050, T055–T056, T061–T064, T082 |
| FR-043 | T018, T050, T068, T072–T073, T078, T082–T083 |
| FR-044 | T064, T069–T070, T073, T078, T080 |
| FR-045 | T043, T079–T082 |
| FR-046 | T041, T043, T046, T079–T082 |
| FR-047 | T061–T067, T080 |
| FR-048 | T062, T064, T066–T067, T080 |
| FR-049 | T068–T071, T073, T075–T080, T083 |
| FR-050 | T013, T017, T019, T027 |
| FR-051 | T037, T042, T046, T079 |
| FR-052 | T004, T006, T013, T016, T019, T022, T025, T027, T050, T072 |
| FR-053 | T004, T006, T013, T016–T019, T022, T025, T027, T072 |
| FR-054 | T013, T018–T019, T037, T050, T072–T073, T083 |

## Coverage Mapping: Success Criteria → Tasks

| Success criterion | Task IDs |
|---|---|
| SC-001 | T029, T038, T041, T079, T084–T086 |
| SC-002 | T037, T042–T046, T047–T054, T073, T079 |
| SC-003 | T042–T046, T073, T079 |
| SC-004 | T055–T060, T080 |
| SC-005 | T061–T067, T073, T078, T080 |
| SC-006 | T047–T054 |
| SC-007 | T018, T050, T072–T073, T078, T082–T083 |
| SC-008 | T064, T069–T070, T073, T078, T080 |
| SC-009 | T014–T015, T020–T021, T028 |
| SC-010 | T004–T005, T007–T008, T042–T046, T055–T060, T082 |
| SC-011 | T079–T083 |
| SC-012 | T013, T017, T019, T027 |
| SC-013 | T037, T042, T046, T079 |

---

## User-Story Dependency Graph

```text
Setup T001–T003
  ↓
Foundational T004–T012
  ↓
US1 Secure settings/Keychain T013–T028
  ↓
US2 Embedded start/stop + protected endpoint publication T029–T041
  ├──────────────→ US3 Authenticated player parity T042–T046
  ├──────────────→ US4 Safe reconfigure T047–T054
  └──────────────→ US5 Local fallback T055–T060
                         ↓
              US6 Bounded shutdown T061–T067
                         ↓
              P5 pre-gate/rollback/cutover/requalification T068–T083
                         ↓
              BUG-002 lifetime correction T084–T086
```

- US3 depends on US2's protected endpoint, but not on US4 UI rotation.
- US4 depends on US1 durable mutations and US2 endpoint lifecycle ordering.
- US5 depends on US2 endpoint monitoring but can be tested independently of US3 gameplay
  breadth; final acceptance still reruns the complete player journey.
- US6 depends on the latest manager behavior from US4/US5 so every state and failure path is owned.
- BUG-002 follows the completed embedded cutover and executes strictly T084 → T085 → T086.
- CLI deletion T075 is forbidden until the full embedded-only package/security gate T073 and
  user-approved immutable checkpoint/worktree rollback drill T074 pass. If T074 is `BLOCKED`, T075
  cannot start. T078 reruns every deterministic gate on the final tree before conditional external
  evidence.

## Phase and Wave Execution Order

1. Phase 1: T001 → T002 → T003.
2. Phase 2: T004 → T005 → T006 → T007 → T008 → T009 → T010 → T011 → T012.
3. US1: parallel tests T013–T018 → parallel implementations T019–T021 → T022 → T023 →
   T024 → parallel frontend T025–T026 → T027 → T028.
4. US2: ~~historical completed T029–T036 remain recorded~~ **BUG-001 follow-up**: completed
   T029–T036 remain recorded; T030–T032 and T035–T036 are non-normative history, while the retained
   portions of T029/T033 and active T034 remain valid. Corrective T037 performs its rewritten RED
   tests before implementation → T038 → T039 → T040 → T041.
5. US3: T042 → T043 → T044 → T045 → T046.
6. US4: T047 → T048 → T049 → T050 → T051 → T052 → T053 → T054.
7. US5: T055 → T056 → T057 → T058 → T059 → T060.
8. US6: T061 → T062 → T063 → T064 → T065 → T066 → T067.
9. P5: T068 → T069 → T070 → T071 → T072 → T073 → T074 → T075 → T076 → T077 → T078 →
   T079 → T080 → T081 → T082 → T083.
10. BUG-002: T084 → T085 → T086.

## Parallel Examples

- After foundational completion, T013–T018 can run concurrently because they author tests/gates in
  disjoint secret, settings, platform, bridge/browser, UI, and scan files.
- After those tests fail as expected, T019–T021 can run concurrently: secret rules, atomic settings,
  and the platform Keychain adapter have separate files and depend only on Phase 2 interfaces.
- After Wails binding generation T024, T025 and T026 can run concurrently because the facade/fixture
  and HTML/CSS surfaces are disjoint; T027 joins them.
- Endpoint auth, protobuf/generated/bindings, lifecycle, and documentation tasks are intentionally not
  marked `[P]`, even where parts look separable, because they share security/order or generated
  ownership surfaces.

## MVP Recommendation

The smallest user-valued demonstration is Phases 1–5 (US1–US3): UI/Keychain setup, one embedded
endpoint, and complete protected player streaming. It is an MVP candidate only if T015/T021 prove
Keychain-only storage and T037/T042/T073 prove the endpoint policy is attached before publication
while direct local/LAN play remains unchanged. ~~Conditional T031/T046 remains real-provider
evidence~~ **BUG-001 follow-up**: Conditional T046/T079 remains real-provider evidence, never a
substitute for deterministic policy/lifecycle tests. A production-ready MVP must also complete US4–US6 and the P5
cutover through T083, because leaving unsafe
reconfigure/shutdown behavior or a parallel CLI production path would violate the constitution.
