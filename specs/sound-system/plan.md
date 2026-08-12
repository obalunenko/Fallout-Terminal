---
status: migrated
feature: Sound System
source: existing implementation
---

# Implementation Plan: Sound System

**Migration status**: Reconstructed from the implementation on 2026-08-13
**Spec**: `specs/sound-system/spec.md`

**Bugfix**: 2026-08-13 — BUG-001 Updated from bugfix patch
**Bugfix adjustment**: 2026-08-13 — BUG-001 Added immediate policy-permitted ambient attempt with retryable gesture fallback
**Bugfix reconciliation**: 2026-08-13 — BUG-001 Updated implementation and verification status after all corrective tasks completed
**Analysis remediation**: 2026-08-13 — BUG-001 Added focused sound-contract, dependency-clean, interactive, and packaged-application verification work
**Verification reconciliation**: 2026-08-13 — BUG-001 Corrected the final browser-suite and packaged-application verification status

## Summary

The implemented sound system spans the embedded Go player HTTP boundary and dependency-free browser JavaScript. Go lists a fixed set of packaged sound categories without exposing filesystem paths. ~~The browser eagerly discovers and prefetches files, unlocks and decodes one-shot Web Audio buffers after a click, uses a looping `HTMLAudioElement` for ambient sound, and receives event triggers from the player presentation.~~ **Adjusted by BUG-001**: The browser eagerly discovers and prefetches files, unlocks and decodes one-shot Web Audio buffers after a qualifying pointer or keyboard gesture, attempts live ambience immediately through a looping `HTMLAudioElement` when browser policy permits, and retains blocked ambience for a later gesture retry. Every audio failure is intentionally non-fatal.

## Technical Context

| Concern | Implemented choice |
|---|---|
| Runtime | Go 1.26 modular Wails application with an embedded browser player |
| Server APIs | Go `net/http`, `io/fs`, and embedded client assets |
| Browser APIs | Fetch, Web Audio API, and `HTMLAudioElement` |
| Third-party sound dependencies | None |
| Formats | MP3, WAV, OGG, M4A, and WebM manifests; current package contains WAV files |
| State | ~~Page-local filename, load-promise, raw-buffer, decoded-buffer, eligibility, and ambient-element state~~ **Analysis remediation**: page-local filename/raw/decoded caches, folder/raw load promises, AudioContext eligibility and in-flight activation, ambient requested state and lifecycle revision, in-flight ambient playback, and the ambient element |
| Persistence | None |
| Automated verification | Colocated Go tests plus Playwright browser journeys |
| Target | macOS 13+ arm64 Wails package and modern same-origin player browsers |

## Project Structure and Ownership

```text
client/
├── index.html                 # Loads sound.js before client.js
├── sound.js                  # Discovery, caching, Web Audio, ambient lifecycle
├── client.js                 # Presentation and authoritative-state sound triggers
└── sounds/
    ├── README.txt             # Folder convention and supported formats
    ├── ambient/
    ├── charscroll/
    ├── enter/
    ├── hack-bad/
    ├── hack-good/
    ├── menu-focus/
    ├── multiple/
    └── single/
internal/
├── player/http.go            # Manifest API, static assets, CSP, path protection
├── player/http_test.go       # Manifest, format, degradation, and asset tests
└── platform/assets_test.go   # Packaged-category and source-contract checks
tests/browser/
├── hacking-camouflage.spec.mjs
└── player-sessions-control.spec.mjs
```

`internal/player/` owns the public HTTP boundary. `client/` owns optional playback and event mapping. Canonical navigation and hacking state remain in the existing Go services and are merely consumed through accepted player messages.

## Contract and State Design

### HTTP manifests and assets

1. The browser requests `/api/sounds/{category}`.
2. `internal/player/http.go` checks the category against an eight-name allowlist.
3. The handler reads `sounds/{category}` from the embedded player filesystem, skips non-regular entries and unsupported extensions, sorts basenames, and emits JSON.
4. Unknown or unreadable categories emit `[]` rather than exposing an operational error.
5. The browser constructs `/sounds/{category}/{encoded-filename}` URLs. The common player asset handler and unsafe-path guard keep access inside the embedded client root.
6. The player CSP permits media only from the same origin.

### One-shot runtime lifecycle

1. `sound.js` starts manifest discovery for seven one-shot categories at page boot.
2. `folderLoads` and `rawLoads` coalesce concurrent work; `folderFiles`, `rawBufs`, and `decodedBufs` retain successful results for the page lifetime.
3. Asset fetches populate raw `ArrayBuffer` values before an audio context is required.
4. ~~The first document click calls `enableWebAudio()`, creates or resumes the browser AudioContext, and decodes prefetched buffers.~~ **Superseded by BUG-001**: Qualifying pointer and keyboard gestures call retryable, coalesced activation; successful eligibility triggers decoding while failed eligibility leaves a later retry possible.
5. An eligible event chooses the first or a random category file, creates a buffer source and gain node, connects them to the destination, and starts the source.
6. The optional test/diagnostic observer is notified only after `src.start()` succeeds.

### Ambient runtime lifecycle

1. Ambient discovery runs separately during boot.
2. The first sorted ambient filename becomes a looping `Audio` element at volume `0.25`.
3. ~~Setup completion and the first document click both call `tryStartAmbient()`.~~ **Superseded by BUG-001**: Setup readiness and generic gestures only reconcile state and cannot start ambience without an active live request.
4. ~~An accepted `TERMINAL_LIVE` also calls `tryStartAmbient()`.~~ **Clarified by BUG-001**: Accepted current live state sets the ambient request active through one sound-module state API.
5. ~~broadcast-mirror clearing and accepted `TERMINAL_CLEAR` messages call `stopAmbient()`.~~ **Clarified by BUG-001**: Every transition without an accepted current live terminal revokes the ambient request and pauses the element through the same state API.

~~The current implementation has no explicit `ambientActive` or `hasLive` input in the sound module. Consequently, an idle click may start ambience; this is recorded as a gap rather than treated as an intended new design.~~ **Resolved by BUG-001**: The sound module now owns explicit requested ambient state through `setAmbientActive(active)`, and reconciliation cannot start ambience while that state is inactive.

**BUG-001 lifecycle correction**:

1. Browser playback eligibility and live ambient request state are independent conditions.
2. `client/sound.js` exposes one ambient state transition such as `setAmbientActive(active)`; its reconciliation starts playback only when the request is active, the asset is ready, the element is paused, and browser policy permits playback.
3. Accepted current `TERMINAL_LIVE` state sets the ambient request active. Clear, mirror reset, broadcast replacement, and reconnect paths with no accepted current terminal set it inactive and pause the element.
4. Qualifying pointer and keyboard user gestures attempt AudioContext activation and reconcile requested ambience. In-flight activation is coalesced, successful eligibility is retained, and an ineligible or failed attempt clears its retry gate for a later qualifying gesture.
5. Setup readiness and generic gestures never grant ambient permission by themselves. Repeated gestures or state reconciliation while already playing do not start another loop.
6. Browser rejection remains silent and non-blocking; no browser is required to override its autoplay policy.

**BUG-001 autoplay adjustment**: Reconciliation MUST call `ambientAudio.play()` as soon as the live request and asset are both ready, without assuming that a prior gesture is always necessary. A fulfilled attempt starts the loop immediately. A thrown or rejected attempt leaves the live request active and the element paused so the next qualifying pointer or keyboard gesture can retry. This opportunistic attempt never weakens the non-live gate.

### Authoritative outcome boundary

Local hover cues are presentation feedback. A hacking selection `enter` cue plays only when `beginSharedAction()` accepts the local dispatch, so observer and already-pending clicks remain silent while accepted requests still receive immediate feedback before the server response. In contrast, `hack-good` and `hack-bad` are produced only by `playHackOutcomeTransition(previousHack, nextHack)` after an accepted current `TERMINAL_LIVE` continuation or `HACK_STATE` update. Revision guards, baseline detection, and transition comparison prevent optimistic, duplicate, stale, pending, and reconnect replay.

## Reconstructed Implementation Phases

### Phase 1 — Package and expose sound assets

- Established the eight-folder asset convention and bundled initial WAV files.
- Loaded the dedicated sound adapter before the player interaction script.
- Added Go manifest discovery with category and extension allowlists, lexical ordering, basename-only responses, and empty-array degradation.
- Served sound files through the existing embedded static client boundary and restrictive CSP.

### Phase 2 — Build optional browser playback

- Added coalesced manifest and raw-file prefetching plus page-lifetime caches.
- ~~Added click-gated AudioContext creation/resume and buffer decoding.~~ **Superseded by BUG-001**: Added retryable pointer/keyboard AudioContext activation with coalesced in-flight initialization and retained successful eligibility.
- ~~BUG-001 reopens activation so qualifying pointer and keyboard gestures can retry after an ineligible attempt without duplicating in-flight work.~~ **Completed by BUG-001** through T008, T026, and T028.
- Added first-file and random-file playback helpers with category-specific gains.
- Added a separate looping `HTMLAudioElement` for ambience.
- Wrapped every media boundary so failure remains non-blocking.

### Phase 3 — Integrate terminal and hacking events

- Mapped menu hover, keyboard movement, pagination, animated reveal, hacking hover/focus, and hacking click events to one-shot categories.
- Mapped accepted authoritative wrong, solved, and lockout transitions to exact-once outcome cues.
- Started ambience during live publication and paused it when broadcast mirrors clear.
- ~~BUG-001 reopens this integration so every accepted live/non-live transition uses one explicit ambient-request state boundary and idle gestures remain silent.~~ **Completed by BUG-001** through T017, T024, T025, and T027.
- Preserved presentation-only ownership: no sound event mutates server or persistent state.

### Phase 4 — Harden verification

- Added Go HTTP tests for allowlisting, sorting, basename exposure, supported formats, static retrieval, and empty-array degradation.
- Added packaged-asset checks for all required categories and source-level outcome contract guards.
- Added browser Web Audio doubles and playback observation at the source-start boundary.
- Covered local preview/entry cues, authoritative active/observer outcomes, replay guards, ineligible contexts, and start failures.

## Technical Decisions

1. **Folder names form the sound contract** so assets can vary without changing source mappings.
2. **The Go server owns discovery allowlists** so browser-controlled paths never become arbitrary filesystem access.
3. **Manifest responses contain basenames only** and browser URLs encode filenames before retrieval.
4. **Unknown categories return `[]`** because sound is optional and the visual client must remain operational.
5. **One-shot bytes are prefetched before user gesture** to reduce event latency without violating autoplay rules.
6. **Decode and playback use Web Audio** for inexpensive per-event gain control and overlapping effects.
7. **Ambient uses `HTMLAudioElement`** because native loop and pause behavior fit a single long-running track.
8. **Multi-file interaction categories choose randomly**, while semantic outcome, focus, and ambient categories use the first sorted file.
9. **Outcome cues follow authoritative state transitions** so all clients converge and acknowledgements do not imply game results.
10. **Audio errors are swallowed at every media boundary** because sound is a progressive enhancement rather than a correctness dependency.

## Constitution Check

| Principle | Assessment |
|---|---|
| Preserve runtime boundaries | Pass: Go owns HTTP/embedded assets; `client/` owns browser playback; no Wails or filesystem API reaches the player page. |
| Keep shared state server-authoritative | Pass: local sounds do not mutate state, and hacking outcomes wait for authoritative snapshots. |
| Protect public boundaries | Pass: category/extension allowlists, shared path traversal rejection, basename responses, and `media-src 'self'` constrain access. |
| Preserve session compatibility | Not affected: no session or player-configuration data is read or written. |
| Match code conventions | Pass: lowercase browser filenames, camelCase JavaScript, small Go HTTP helpers, and colocated Go/browser tests. |
| Testing and packaging | Pass for discovered implementation: Go, packaged-asset, and Playwright coverage exists; remaining gaps are documented below. |

No constitution violation or complexity exception is required.

## Complexity Assessment

| Measure | Existing scope |
|---|---|
| Core playback | ~~211 lines in `client/sound.js`~~ One dependency-free browser module owning discovery, Web Audio activation, one-shot playback, and ambient reconciliation; exact line count is intentionally not treated as a stable design metric. |
| Integration | ~~13 direct playback/lifecycle call sites in `client/client.js`, plus script loading in `client/index.html`~~ Presentation event mappings and explicit accepted live/non-live lifecycle transitions in `client/client.js`, with script ordering owned by `client/index.html`. |
| Server boundary | One manifest branch and helper set within the 155-line `internal/player/http.go` |
| Packaged assets | 20 WAV files in eight required folders plus one README |
| Automated surfaces | Four test files: HTTP, packaged assets/source contract, and two Playwright suites |
| Dependency depth | Low: browser-native audio/fetch APIs and Go standard-library HTTP/filesystem APIs |
| State complexity | Moderate: asynchronous coalesced loads, autoplay eligibility, caches, ambient state, and authoritative replay guards |
| BUG-001 lifecycle edges | Idle gesture before publication, live publication before gesture, failed-first/later-successful activation, repeated gestures, clear/reset/replacement, and reconnect without a current live terminal |

## Verification Plan

~~The following commands are the applicable current gates; this documentation migration does not claim they were rerun while reverse-engineering the feature.~~ ~~**Post-implementation reconciliation**: BUG-001 validation recorded `go test ./... -count=1` passing across all packages and the complete Playwright suite passing 37/37 tests. Wails packaging and interactive native-audio checks remain unclaimed.~~ **Current verification status**: Formatting, vet, Go tests, 39/39 Playwright journeys, `wails dev`, and the clean macOS arm64 packaged-asset verification passed. Manual audible-output confirmation remains unclaimed because the agent environment cannot reliably observe system speakers or browser autoplay policy.

| Surface | Automated check | Interactive check | Expected result |
|---|---|---|---|
| Sound manifests/assets | `go test ./internal/player ./internal/platform` | Request each manifest and one sound asset | Only allowlisted sorted basenames are exposed and packaged categories are non-empty. |
| Browser syntax | `node --check client/sound.js` and `node --check client/client.js` | Load player page | Scripts load without a parse or initialization failure. |
| Player behavior | `npm test --prefix tests/browser` | Use menu, hacking preview/selection, success, failure, and lockout with audio enabled | Mapped cues play once and visual behavior remains authoritative. |
| Browser dependency integrity | `npm ci --prefix tests/browser` | N/A | The pinned Playwright test environment installs cleanly before browser verification. |
| Failure degradation | Relevant Playwright journey associated with `specs/004-player-sessions-control/bugs/BUG-006.md` | Disable or reject browser audio | Terminal interactions and authoritative rendering remain usable and silent. |
| BUG-001 ambient lifecycle | Focused Playwright journey with observable `Audio` and AudioContext doubles | Exercise idle gesture, live-before-gesture, gesture-before-live, clear/reset/reconnect, repeated gestures, and failed-first retry | Ambient plays only when both conditions are true, stops on every non-live transition, never duplicates, and activation can recover. |
| BUG-001 immediate autoplay fallback | Focused Playwright journey with autoplay-allowed and autoplay-blocked `Audio` doubles | Publish a live terminal before any gesture in both policy modes, then provide a qualifying pointer or keyboard gesture to the blocked case | Allowed autoplay starts immediately; blocked autoplay remains requested and starts on retry; idle state never plays. |
| Full Go quality | `gofmt -l .`, `go vet ./...`, `go test ./...` | N/A | No formatting paths; vet and tests succeed. |
| Interactive Wails journey | `wails dev` | Connect a player browser and exercise live, clear, pointer, and keyboard audio transitions | The player remains usable and audio follows the accepted lifecycle; unavailable audio checks are reported rather than claimed. |
| Package | `wails build -clean -platform darwin/arm64` | Launch the packaged app, connect a player browser, and request the sound script, manifests, and an ambient asset | The clean arm64 application serves `client/sound.js`, all eight manifests, and at least one ambient asset from its embedded player boundary. |

Completion evidence for the BUG-001 gates is recorded by T030–T032. T033–T034 close the later regular-file manifest and dispatch-gated selection-audio convergence findings. A command that cannot be exercised in the current environment remains explicitly unclaimed with its reason.

### Analysis remediation evidence

| Gate | Result |
|---|---|
| `npm ci --prefix tests/browser` | Passed; npm reported the optional blocked `fsevents` install-script warning. |
| `gofmt -l .` | Passed with no paths. |
| `go vet ./...` | Passed. |
| `go test ./... -count=1` | Passed across all packages. |
| `npm test --prefix tests/browser` | Passed 39/39 Playwright journeys. |
| `wails dev` | Built and launched successfully; the live player served its page and ambient manifest before clean shutdown. Manual audible-output confirmation is unclaimed because the agent environment cannot reliably observe system speakers or browser policy; deterministic Playwright journeys provide the live/clear and pointer/keyboard audio assertions. |
| `wails build -clean -platform darwin/arm64` | Passed; the launched clean package served `sound.js`, all eight manifests, and a 296,320-byte ambient WAV. |

## Identified Gaps and Follow-up Options

~~BUG-001 patches the intended behavior and reopens implementation work for items 1–4; they remain implementation gaps until the updated tasks pass verification.~~ **Resolved by BUG-001**: All reopened and corrective tasks passed their recorded verification; only the explicitly retained follow-ups below remain.

1. ~~Gate ambient start on explicit live presentation state and add start/clear/reconnect browser coverage.~~ **Completed by T024, T025, and T027.**
2. ~~Consider eligible keyboard gestures and an accessible muted/audio-readiness indication.~~ **Adjusted by BUG-001**: Qualifying pointer and keyboard fallback is required; an accessible muted/audio-readiness indication remains a separate follow-up.
3. ~~Decide whether a failed AudioContext activation should be retried on a later gesture.~~ **Completed by T008, T026, and T028** with later-gesture retry and concurrent-attempt coalescing.
4. ~~Add focused assertions for gain values, random selection boundaries, load coalescing, and character-scroll playback.~~ **Completed after analysis**: T031 covers gain-node and character-scroll behavior; T032 covers deterministic selection and folder/raw load coalescing.
5. ~~Reconcile the older `player-terminal-presentation` implementation narrative with the current Go server if that broader migrated artifact is refreshed.~~ **Completed during post-verification consistency cleanup**: the broader presentation artifacts now describe the embedded Go server and current ambient activation contract.
