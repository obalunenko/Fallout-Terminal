---
status: migrated
feature: Sound System
source: existing implementation
---

# Tasks: Sound System

**Migration status**: Reconstructed from completed implementation on 2026-08-13
**Input**: `specs/sound-system/spec.md` and `specs/sound-system/plan.md`

**Bugfix**: 2026-08-13 — BUG-001 Updated from bugfix patch
**Bugfix adjustment**: 2026-08-13 — BUG-001 Added immediate policy-permitted ambient attempt and blocked-playback fallback coverage
**Bugfix reconciliation**: 2026-08-13 — BUG-001 Recorded completion of all reopened and corrective tasks and retained only unresolved follow-ups
**Analysis remediation**: 2026-08-13 — BUG-001 Added constitution-gate, packaged-output, and focused one-shot contract verification
**Verification reconciliation**: 2026-08-13 — BUG-001 Corrected the final task count and verification status

~~All implementation tasks are marked complete because this feature predates its focused Spec Kit artifacts.~~ **BUG-001 correction**: Five historical completions were reopened and six corrective tasks were added. ~~**Post-implementation reconciliation**: All 29 tasks are now complete with BUG-001 verification evidence recorded below.~~ **Verification reconciliation**: T030–T032 brought the BUG-001 milestone to 32 completed tasks. Phase 7 later added and completed T033–T034, bringing the current total to 34 tasks.

## Phase 1 — Asset and HTTP Contract

**Goal**: Package categorized sound files and expose them through a constrained same-origin server boundary.

- [x] T001 [US1] Define the eight sound folder roles and supported media formats in `client/sounds/README.txt`.
- [x] T002 [US1] Add non-empty ambient, reveal, focus, hacking preview, selection, success, and failure assets under `client/sounds/`.
- [x] T003 [US1] Load `client/sound.js` before `client/client.js` in `client/index.html` so playback functions exist before interaction handlers run.
- [x] T004 [US1] Implement the `/api/sounds/:folder` category allowlist, extension filter, lexical ordering, basename-only JSON response, and empty-array degradation in `internal/player/http.go`.
- [x] T005 [US1] Serve sound assets through the existing embedded player asset handler while preserving traversal rejection and `media-src 'self'` in `internal/player/http.go`.
- [x] T006 [US1] Add manifest, supported-file, asset-serving, and degradation tests in `internal/player/http_test.go`.

**Checkpoint**: Player browsers can discover and fetch only supported sound assets from the embedded client root.

---

## Phase 2 — Browser Playback Foundation

**Goal**: Make one-shot and ambient audio fast when available and harmless when unavailable.

- [x] T007 [US2] Implement folder discovery plus coalesced folder/raw-load promises and page-lifetime filename, raw-buffer, and decoded-buffer caches in `client/sound.js`.
- [x] T008 [US2] ⚠️ Reopened: Prefetch all discovered one-shot files at boot and create/resume the AudioContext from retryable qualifying pointer or keyboard gestures without duplicate in-flight initialization in `client/sound.js`. (reopened — BUG-001)
- [x] T009 [US2] Decode cached buffers, create Web Audio source/gain graphs, and report successful source starts through the optional observer in `client/sound.js`.
- [x] T010 [US2] Implement random selection for `single`, `multiple`, `enter`, and `charscroll`, first-file selection for semantic categories, and the existing category gains in `client/sound.js`.
- [x] T011 [US3] ⚠️ Reopened: Implement first-file ambient discovery, looping `HTMLAudioElement` playback at volume `0.25`, immediate live-state-gated start attempt when policy permits, retained request after autoplay rejection, and non-fatal pause in `client/sound.js`. (reopened — BUG-001)
- [x] T012 [US4] Contain manifest, fetch, JSON, decode, AudioContext, source-start, observer, autoplay, and pause failures inside optional-audio boundaries in `client/sound.js`.

**Checkpoint**: Audio initializes after an eligible gesture and every failure path leaves the player interface usable.

---

## Phase 3 — Player Presentation Integration

**Goal**: Connect local terminal feedback and authoritative outcomes without transferring state ownership to the browser.

- [x] T013 [US2] Map normal-menu pointer transitions and keyboard row movement to focus/selection feedback in `client/client.js`.
- [x] T014 [US2] Map animated row/line reveal and information-page changes to `charscroll` and `enter` feedback in `client/client.js`.
- [x] T015 [US2] Map hacking symbol, word, and valid-pattern hover/focus transitions plus accepted local target clicks to `single`, `multiple`, and `enter` feedback in `client/client.js`.
- [x] T016 [US2] Derive `hack-good` and `hack-bad` only from accepted current authoritative state transitions, with baseline, duplicate, stale-revision, and pending-rerender replay guards in `client/client.js`.
- [x] T017 [US3] ⚠️ Reopened: Drive one explicit ambient-request API from every accepted live, clear, mirror-reset, broadcast-replacement, and reconnect-without-live transition in `client/client.js`. (reopened — BUG-001)

**Checkpoint**: Local interaction cues remain presentation-only, while outcome cues follow canonical server state exactly once per eligible client.

---

## Phase 4 — Regression and Packaging Evidence

**Goal**: Protect the asset contract, interaction mappings, authoritative transition behavior, and graceful degradation.

- [x] T018 [US1] Verify all eight packaged categories contain a non-empty supported asset in `internal/platform/assets_test.go`.
- [x] T019 [US2] Guard the sound adapter mappings, Web Audio eligibility path, caches, gains, and authoritative outcome source boundary in `internal/platform/assets_test.go`.
- [x] T020 [US2] ⚠️ Reopened: Extend browser audio doubles to observe ambient `Audio` play/pause state, controllable autoplay fulfillment/rejection, and retryable AudioContext activation in `tests/browser/hacking-camouflage.spec.mjs` and `tests/browser/player-sessions-control.spec.mjs`. (reopened — BUG-001)
- [x] T021 [US2] Cover grouped and individual hacking preview sounds, exact one-shot entry behavior, and pending-rerender replay prevention in `tests/browser/hacking-camouflage.spec.mjs`.
- [x] T022 [US2] Cover menu focus plus exact-once wrong, solved, and lockout cues for active and observer clients in `tests/browser/player-sessions-control.spec.mjs`.
- [x] T023 [US4] ⚠️ Reopened: Cover reconnect baselines, rejected actions, ineligible audio, source-start failure, and later eligible activation recovery as silent, non-blocking cases in `tests/browser/player-sessions-control.spec.mjs`. (reopened — BUG-001)

**Checkpoint**: The implemented contract has automated evidence across the Go HTTP, packaged-asset, browser interaction, and authoritative synchronization boundaries.

---

## Phase 5 — BUG-001 Ambient Activation and Lifecycle Correction

**Goal**: Bind ambience to accepted live-terminal state and make browser activation retryable without duplicate playback or initialization.

- [x] T024 [US3] Add an explicit `setAmbientActive(active)`-style state API and require ambient reconciliation to attempt playback immediately when live request, asset readiness, and paused state align, retain the request after autoplay rejection, and retry on a later qualifying gesture in `client/sound.js`.
- [x] T025 [US3] Replace direct ambient start/stop integration with explicit accepted live/non-live state transitions covering publication, clear, mirror reset, broadcast replacement, and reconnect-without-live paths in `client/client.js`; depends on T024.
- [x] T026 [US2] Separate gesture observation, in-flight activation, successful Web Audio eligibility, and retryable failure state for qualifying pointer and keyboard gestures in `client/sound.js`; depends on reopened T008.
- [x] T027 [US3] Add deterministic Playwright coverage for idle silence, autoplay-allowed immediate live start, autoplay-blocked gesture fallback, live-before-gesture, gesture-before-live, loop/volume, repeated reconciliation, and clear/reset/reconnect pause behavior in `tests/browser/player-sessions-control.spec.mjs`; depends on T020, T024, and T025.
- [x] T028 [US4] Add deterministic Playwright coverage for failed-first/later-successful pointer and keyboard activation, coalesced concurrent attempts, and non-blocking browser rejection in `tests/browser/player-sessions-control.spec.mjs`; depends on T020, T023, and T026.
- [x] T029 [US3] Update `internal/platform/assets_test.go` source-contract guards to protect the explicit ambient state and retry boundaries without coupling to incidental implementation text; depends on T024–T026.

**Checkpoint**: Ambient sound is live-only, policy-permitted autoplay begins immediately, blocked autoplay recovers on a qualifying gesture, every non-live path is silent, duplicate work is prevented, and focused regression evidence passes.

---

## Phase 6 — Analysis Remediation and Completion Gates

**Goal**: Close constitution-gate evidence and the remaining packaged-output and focused one-shot verification gaps without changing runtime sound behavior.

- [x] T030 [US1] Build a clean macOS arm64 Wails application and verify the packaged player serves `client/sound.js`, all eight sound manifests, and at least one ambient asset; depends on T003, T018, and T029.
- [x] T031 [US2] Extend the Playwright AudioContext double to capture gain values and verify every mapped one-shot category, including character-scroll playback, crosses the source-start boundary at its specified gain; depends on T019–T023.
- [x] T032 [US2] Add deterministic browser coverage for random-versus-first-file selection and concurrent folder/raw-load coalescing using controlled randomness and fetch counters; depends on T007, T010, and T020.

**Checkpoint**: Clean dependency installation, formatting, vet, Go tests, focused and complete browser tests, interactive availability reporting, and packaged-player asset evidence satisfy the applicable completion gates.

## Dependencies and Execution Order

1. Asset folders and the Go manifest contract precede browser discovery.
2. Discovery, prefetching, gesture eligibility, and playback helpers precede event integration.
3. Local interaction mappings are independent of canonical state; hacking outcome mappings depend on existing revision and authoritative snapshot guards.
4. HTTP and packaged-asset tests verify the producer boundary; Playwright journeys verify the browser consumer and multi-client outcome boundary.
5. For BUG-001, T024 precedes T025 and T027; reopened T008 precedes T026, which precedes T028; T020 precedes both new browser verification tasks.
6. T024–T026 define the final state boundaries before T029 updates source-contract guards.
7. T030 depends on the packaged source and guard tasks; T031 and T032 depend on the completed playback foundation and browser doubles, so all three analysis-remediation tasks form the final verification wave.

## Identified Gaps

~~The following are findings, not completed migration tasks:~~ **Post-implementation reconciliation**: BUG-001 findings are retained as history and annotated with their resolution; unrelated follow-ups remain open without new task claims.

~~Items 1–4 are tracked by BUG-001 and have reopened or new tasks above.~~ **Resolved or narrowed by BUG-001** through the completed reopened tasks and T024–T029.

1. ~~Ambient playback is not gated on a live terminal and may begin from an idle-page click.~~ **Resolved by T011, T017, and T024–T027.**
2. ~~Keyboard-only interaction does not unlock audio.~~ **Resolved by T008, T026, and T028.**
3. ~~Failed first-click Web Audio eligibility has no later retry or visible readiness state.~~ **Partially resolved by T008, T026, and T028**: later retry is implemented; visible readiness remains out of scope.
4. ~~Ambient lifecycle, exact gain-node values, random selection, cache coalescing, and character-scroll playback lack focused behavioral assertions.~~ **Resolved after analysis**: ambient lifecycle and activation coalescing are covered; T031 covers gain-node and character-scroll behavior, and T032 covers deterministic selection and folder/raw load coalescing.
5. ~~The older migrated player-presentation plan describes an Express sound endpoint rather than the current Go handler.~~ **Resolved during post-verification consistency cleanup**: the presentation plan now records the embedded Go handler and current audio lifecycle.

## Verification Status

~~This brownfield migration inspected existing tests and implementation but did not modify or revalidate runtime code. No formatting, Go, Playwright, Wails, packaging, native-audio, or release result is claimed by creation of these documentation artifacts.~~ ~~**BUG-001 implementation verification**: `go test ./... -count=1` passed across all packages, and the complete Playwright suite passed 37/37 tests. Wails packaging, interactive native-audio, and release validation remain unclaimed.~~ **Current verification status**: All 34 tasks are complete. Formatting, vet, Go tests, race tests, 39/39 Playwright journeys, `wails dev`, and clean packaged-player verification passed. Manual audible-output confirmation remains unclaimed.

**Analysis remediation**: T030–T032 reopen verification completeness only. The applicable constitution gates and any unavailable interactive checks MUST be recorded before these tasks are completed.

**Analysis remediation evidence**: `npm ci --prefix tests/browser`, `gofmt -l .`, `go vet ./...`, `go test ./... -count=1`, 39/39 Playwright journeys, `wails dev`, and a clean macOS arm64 Wails build passed. The development player served its page and ambient manifest; the launched clean package served `sound.js`, all eight manifests, and a 296,320-byte ambient WAV. npm's optional blocked `fsevents` install-script warning was non-fatal. Manual audible-output confirmation is unclaimed because the agent environment cannot reliably observe system speakers or browser policy; deterministic Playwright journeys cover live/clear and pointer/keyboard audio behavior.

## Phase 7: Convergence

- [x] T033 Require manifest discovery in `internal/player/http.go` to exclude every non-regular filesystem entry with a supported extension, and add focused coverage in `internal/player/http_test.go`, per FR-002 (partial)
- [x] T034 Gate hacking `enter` feedback in `client/client.js` on `beginSharedAction()` accepting the local dispatch, and cover accepted, observer, and pending clicks in the Playwright sound journeys, per US2/AC7 (contradicts)
