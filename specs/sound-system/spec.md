---
status: migrated
feature: Sound System
source: existing implementation
---

# Feature Specification: Sound System

**Migration status**: Reverse-engineered from the existing implementation on 2026-08-13
**Scope**: Player-browser sound discovery, playback, terminal event integration, ambient lifecycle, and optional-audio failure behavior

**Bugfix**: 2026-08-13 — BUG-001 Clarified live-only ambient state, retryable user activation, and lifecycle verification
**Bugfix adjustment**: 2026-08-13 — BUG-001 Attempt live ambience immediately when autoplay is permitted and retain gesture fallback when blocked
**Bugfix reconciliation**: 2026-08-13 — BUG-001 Recorded completed implementation and separated resolved lifecycle defects from remaining sound-test follow-ups
**Analysis remediation**: 2026-08-13 — BUG-001 Defined activation events and readiness, and added focused behavioral and packaged-application evidence

## Relationship to Existing Specifications

Sound behavior is also described as a supporting concern in `specs/player-terminal-presentation/` and `specs/hacking-game/`. This focused specification records the current implementation after the Wails/Go migration and is the detailed source for future sound-system changes. It does not change the server-authoritative navigation or hacking contracts owned by those features.

For BUG-001, this focused specification supersedes the older `player-terminal-presentation` wording that a user gesture is always required before an ambient start attempt. The shared live-only and clear-time pause requirements remain compatible; only the ability to attempt playback immediately when browser policy already permits autoplay is newly clarified.

## Purpose

The sound system adds optional Fallout-style feedback to the player terminal. The embedded Go player server exposes a constrained manifest of packaged sound assets, while browser code prefetches and decodes those assets and maps them to local presentation events and authoritative hacking outcomes. Audio failure never blocks terminal rendering, navigation, hacking, or synchronization.

## User Scenarios and Acceptance

### User Story 1 — Discover packaged sound assets safely (Priority: P1)

As a player browser, I can discover the supported packaged sounds without gaining access to arbitrary embedded files so that audio is available through the same origin as the terminal.

**Independent verification**: Request each supported manifest, an unsupported manifest, and representative asset paths from the player HTTP handler.

**Acceptance scenarios**:

1. **Given** one of the eight supported category names, **when** `GET /api/sounds/:folder` is requested, **then** the server returns a JSON array containing only supported, non-directory filenames from that embedded folder in lexical order.
2. **Given** a file ending in `.mp3`, `.wav`, `.ogg`, `.m4a`, or `.webm`, **when** its category is listed, **then** the extension comparison is case-insensitive and the response contains only the basename.
3. **Given** an unknown category or an unreadable/missing category directory, **when** its manifest is requested, **then** the server returns HTTP 200 with an empty JSON array.
4. **Given** a valid packaged filename, **when** `/sounds/:folder/:filename` is requested, **then** the embedded static asset is returned under the player page's same-origin and `media-src 'self'` policy.
5. **Given** a path containing traversal segments or a backslash, **when** it reaches the player HTTP boundary, **then** it is rejected by the existing unsafe-path guard and cannot escape the embedded client root.

---

### User Story 2 — Receive contextual one-shot feedback (Priority: P1)

As a player, I hear distinct feedback for terminal navigation, content reveal, hacking previews, selections, and outcomes so that interactions feel responsive and atmospheric.

~~**Independent verification**: Enable audio with a click, navigate a terminal, reveal content, hover and select each hacking target class, and publish wrong, solved, and failed authoritative hacking transitions.~~ **Clarified after analysis**: Activate audio through both a document-level pointer gesture and a keyboard gesture, then navigate, reveal content, exercise every hacking target class, and publish wrong, solved, and failed authoritative transitions.

**Acceptance scenarios**:

1. ~~**Given** the page has loaded, **when** sound initialization runs, **then** the client requests the seven one-shot manifests and prefetches every returned supported asset without blocking page readiness.~~ **Clarified after analysis**: **Given** the page has loaded, **when** sound initialization runs, **then** manifest discovery and asset prefetch run asynchronously without delaying WebSocket connection setup or the first terminal render.
2. ~~**Given** the first document click creates or resumes a running Web Audio context, **when** initialization completes, **then** prefetched one-shot assets are decoded and become eligible for playback.~~ **Superseded by BUG-001**: **Given** a qualifying pointer or keyboard user gesture, **when** Web Audio activation reaches a running context, **then** prefetched one-shot assets are decoded and become eligible; if that attempt remains ineligible, a later qualifying gesture can retry without creating concurrent initialization work.
3. **Given** the pointer moves to a different normal menu row, **when** the local highlight changes, **then** the first `menu-focus` file plays at gain `0.50` once for that transition.
4. **Given** a selectable individual hacking symbol is newly hovered or focused, **when** its local preview changes, **then** one random `single` file plays at gain `0.55`.
5. **Given** a hacking word or unused valid pattern is newly hovered or focused, **when** its grouped preview changes, **then** one random `multiple` file plays at gain `0.55`; moving inside the same group does not replay it.
6. **Given** normal menu selection changes with Arrow Up or Arrow Down, **when** the index changes, **then** one random `multiple` file plays at gain `0.55`.
7. **Given** a valid hacking target is clicked or an information page changes, **when** the local action is accepted for dispatch, **then** one random `enter` file plays at gain `0.65`.
8. **Given** a folder, entry, or command begins an animated reveal, **when** each new row or line is appended, **then** one random `charscroll` file plays at gain `0.40`; a non-animated rerender remains silent.
9. **Given** an established puzzle changes authoritatively from unsolved to solved, **when** the accepted current revision renders, **then** the first `hack-good` file plays at gain `0.80` exactly once in each eligible player view.
10. **Given** an established puzzle authoritatively loses an attempt or reaches lockout, **when** the accepted current revision renders, **then** the first `hack-bad` file plays at gain `0.70` exactly once in each eligible player view.
11. **Given** only an action acknowledgement, rejected action, duplicate snapshot, stale revision, reconnect baseline, or pending-state rerender occurs, **when** it is processed, **then** no hacking outcome sound is replayed.

---

### User Story 3 — Hear the terminal ambience (Priority: P2)

As a player, I hear a low looping terminal hum after browser autoplay requirements are satisfied, and the loop stops when the current terminal presentation is cleared.

~~**Independent verification**: Load the page with an ambient asset, click before and after publication, publish a terminal, and clear or replace its broadcast mirrors while observing the `HTMLAudioElement` lifecycle.~~ **Adjusted by BUG-001**: Exercise autoplay-allowed and autoplay-blocked browser doubles, publish before and after a qualifying pointer or keyboard gesture, and clear or replace broadcast mirrors while observing the `HTMLAudioElement` lifecycle.

**Acceptance scenarios**:

1. **Given** the ambient manifest contains files and `window.Audio` is available, **when** setup completes, **then** the first lexically sorted ambient file is assigned to a looping audio element at volume `0.25`.
2. ~~**Given** the ambient element is ready, **when** the first document click occurs, **then** the client records the gesture and attempts playback without surfacing a rejected autoplay promise.~~ **Superseded by BUG-001**: **Given** the ambient element is ready and an accepted live terminal requests ambience, **when** a qualifying pointer or keyboard user gesture satisfies browser playback policy, **then** the client starts the loop without surfacing a rejected autoplay promise.
3. **Given** a live terminal is accepted after the gesture, **when** `TERMINAL_LIVE` renders, **then** the client attempts to start a paused ambient element.
4. **Given** terminal mirrors are cleared or an accepted `TERMINAL_CLEAR` arrives, **when** the player returns to a non-live state, **then** the ambient element is paused.
5. ~~**Observed behavior**: If ambient setup has completed, the first document click can start the loop even when no live terminal exists because the sound module does not track live-state eligibility.~~ **Superseded by BUG-001**: Connecting, character-selection, assigned-waiting, idle, cleared, and other non-live states MUST remain ambient-silent regardless of gestures or asset readiness.
6. ~~**Given** an accepted live terminal requests ambience before any eligible gesture, **when** the terminal renders, **then** the ambient element remains paused until a later qualifying gesture permits playback.~~ **Adjusted by BUG-001**: **Given** an accepted live terminal requests ambience and the ambient asset is ready, **when** either condition becomes true, **then** the client immediately attempts playback; if browser policy allows autoplay the loop starts, otherwise it remains requested and paused until a later qualifying gesture retries it.
7. **Given** the first activation attempt remains suspended or fails, **when** a later qualifying gesture occurs, **then** activation is retried and eligible playback can recover without a page reload.
8. **Given** ambience is already playing for the active terminal, **when** additional qualifying gestures or duplicate live-state reconciliation occurs, **then** no duplicate ambient playback is started.

---

### User Story 4 — Continue without usable audio (Priority: P1)

As a player, I can continue using every visible terminal and hacking interaction when audio APIs, files, decoding, or playback are unavailable.

**Independent verification**: Exercise the terminal with missing manifests, invalid responses, no AudioContext, an ineligible context, decode failures, and source-start failures.

**Acceptance scenarios**:

1. **Given** a manifest request fails, returns a non-success response, returns invalid JSON, or returns a non-array value, **when** a category loads, **then** it behaves as an empty category without blocking the player UI.
2. **Given** an asset fetch or decode fails, **when** its event occurs, **then** playback is skipped and terminal state remains usable.
3. **Given** Web Audio is absent, cannot be created, cannot be resumed to `running`, or rejects a source start, **when** one-shot playback is requested, **then** no exception escapes into player interaction handling.
4. **Given** ambient construction, autoplay, playback, or pause fails, **when** its lifecycle runs, **then** the failure remains non-fatal.
5. **Given** a diagnostic sound observer is absent or throws, **when** a sound starts, **then** production playback and terminal behavior continue unaffected.

## Functional Requirements

- **FR-001**: The player HTTP handler MUST allowlist exactly `ambient`, `charscroll`, `enter`, `hack-bad`, `hack-good`, `menu-focus`, `multiple`, and `single` for manifest discovery.
- **FR-002**: Manifest discovery MUST include only regular files with case-insensitive `.mp3`, `.wav`, `.ogg`, `.m4a`, or `.webm` extensions and MUST return sorted basenames rather than paths.
- **FR-003**: Unknown or unavailable sound categories MUST degrade to HTTP 200 with the exact JSON shape `[]`.
- **FR-004**: Sound assets MUST remain inside the embedded `client/` filesystem and under the player page's same-origin media policy.
- **FR-005**: The browser MUST coalesce concurrent folder and raw-asset loads and cache discovered filenames, raw buffers, and decoded buffers for the page lifetime.
- **FR-006**: ~~One-shot files MUST be prefetched at boot, while Web Audio context activation and decoding MUST wait for an eligible document click.~~ **Superseded by BUG-001**: One-shot files MUST be prefetched at boot, while Web Audio activation and decoding MUST wait for a qualifying pointer or keyboard user gesture and follow FR-014's coalescing and retry rules.
- **FR-007**: One-shot playback MUST create a buffer source and gain node, connect them to the current AudioContext destination, and use the implemented category volume.
- **FR-008**: `single`, `multiple`, `enter`, and `charscroll` MUST select randomly from the discovered category; `menu-focus`, `hack-good`, and `hack-bad` MUST select the first sorted file.
- **FR-009**: Local preview and navigation sounds MUST follow the event mappings and replay guards described in User Story 2.
- **FR-010**: Hacking outcome sounds MUST derive from accepted authoritative state transitions, not optimistic requests or action acknowledgements.
- **FR-011**: Duplicate, stale, reconnect-baseline, and pending presentation updates MUST NOT replay hacking outcome audio.
- **FR-012**: Eligible active-controller and observer views MUST independently play the same authoritative outcome transition exactly once.
- **FR-013**: ~~Ambient playback MUST use the first sorted ambient file, an `HTMLAudioElement`, looping enabled, and volume `0.25`.~~ **Clarified by BUG-001**: Ambient playback MUST use the first sorted ambient file, an `HTMLAudioElement`, looping enabled, and volume `0.25`; playback MUST be permitted only when an accepted live terminal requests ambience. **Adjusted by BUG-001**: Once both the live request and asset are ready, the client MUST immediately attempt playback even without a prior gesture; rejection MUST preserve the request for a later qualifying-gesture retry.
- **FR-014**: ~~The current gesture gate MUST be the first document click; keyboard-only interaction does not activate audio.~~ **Superseded by BUG-001**: The client MUST attempt Web Audio activation and blocked-ambient recovery from qualifying pointer and keyboard user gestures, MUST coalesce an in-flight attempt, MUST retain successful eligibility, and MUST allow a later qualifying gesture to retry after an ineligible or failed attempt. Ambient's immediate `HTMLAudioElement.play()` attempt under FR-013 does not depend on prior Web Audio eligibility. **Clarified after analysis**: The qualifying activation opportunities are document-level `pointerdown` and `keydown` events; browser rejection remains non-fatal and does not make the event intrinsically eligible.
- **FR-015**: ~~Clearing live terminal state MUST pause ambient playback.~~ **Clarified by BUG-001**: Clearing live terminal state MUST revoke the ambient request and pause playback; connecting, selection, waiting, cleared, broadcast-replacement, mirror-reset, and reconnect-baseline paths without an accepted live terminal MUST remain ambient-inactive.
- **FR-016**: ~~Fetch, response, decode, AudioContext, source, observer, autoplay, and media-device failures MUST remain optional and non-blocking.~~ **Clarified by BUG-001**: These failures MUST remain optional and non-blocking, and a failed activation attempt MUST NOT prevent a later qualifying retry.
- **FR-017**: Sound behavior MUST remain presentation-only and MUST NOT mutate canonical navigation, hacking, player-session, or persistent session state.

## HTTP Contract

| Direction | Contract | Behavior |
|---|---|---|
| Player → server | `GET /api/sounds/:folder` | Returns a JSON array of sorted supported basenames for an allowlisted category, otherwise `[]`. |
| Player → server | `GET /sounds/:folder/:filename` | Returns an existing embedded client asset after the common path-safety boundary. |

No WebSocket, Wails bridge, JSON persistence, database, or external-service contract is introduced by the sound system. WebSocket snapshots merely provide the authoritative transitions consumed by the presentation layer.

## Assumptions and Constraints

- Player browsers provide Fetch and may provide Web Audio and `HTMLAudioElement`; missing media APIs are supported through silent degradation.
- Browser autoplay policy takes precedence over automatic audio playback.
- Audio state and caches are page-local and reset on reload.
- Packaged categories currently contain at least one non-empty WAV asset, although the runtime tolerates empty categories.
- The test-only `window.__falloutTerminalSoundObserver` hook is diagnostic and is not a product API.

## Success Criteria

- **SC-001**: Every allowlisted manifest returns only sorted supported basenames, and an unsupported category returns `[]` without exposing another embedded path.
- **SC-002**: ~~After one eligible click, every mapped one-shot interaction can cross the Web Audio source-start boundary at its configured gain.~~ **Clarified by BUG-001**: After a qualifying eligible pointer or keyboard gesture, every mapped one-shot interaction can cross the Web Audio source-start boundary at its configured gain; an earlier ineligible attempt does not prevent later recovery.
- **SC-003**: A wrong, solved, or failed authoritative hacking transition produces exactly one matching outcome cue per eligible active or observer client, with no replay from duplicate or stale snapshots.
- **SC-004**: Hovering within one grouped hacking target or rerendering pending state does not replay its preview cue.
- **SC-005**: ~~Clearing terminal mirrors pauses any existing ambient element.~~ **Clarified by BUG-001**: Ambient remains paused throughout every non-live phase, attempts to start immediately when live ambience and the asset are present, starts at once if autoplay is permitted or after a qualifying gesture if initially blocked, and pauses on clear, mirror reset, broadcast replacement, or reconnect without a current live terminal.
- **SC-006**: ~~With manifests, assets, Web Audio, decoding, or playback unavailable, all visible terminal, navigation, and hacking paths remain usable.~~ **Clarified by BUG-001**: All visible terminal, navigation, and hacking paths remain usable, and a later qualifying activation attempt remains possible.
- **SC-007**: A clean packaged application retains `client/sound.js` and at least one non-empty supported asset in every required category.
- **SC-008**: Repeated or concurrent reconciliation and qualifying gestures create no duplicate AudioContext initialization or ambient start, while an autoplay-blocked immediate attempt or failed first activation can recover on a later browser-eligible gesture without reloading the page.

## Identified Gaps

~~Gaps 1–4 are tracked by BUG-001 and remain open until the reopened and corrective tasks are implemented and verified.~~ **Resolved by BUG-001**: The live-state gate, pointer/keyboard activation retry, failed-attempt recovery, and ambient lifecycle coverage were implemented and verified by T008, T011, T017, T020, and T023–T029.

1. ~~Ambient playback can begin on the first click while the player is idle; there is no live-state flag inside `sound.js`.~~ **Resolved by BUG-001**: `setAmbientActive(active)` gates reconciliation on accepted live-terminal state.
2. ~~The gesture gate listens only for `click`, so keyboard-only users may never enable either Web Audio or ambient playback.~~ **Resolved by BUG-001**: Qualifying pointer and keyboard gestures both drive retryable activation and blocked-ambient recovery.
3. ~~A first eligibility attempt that resolves false is cached in `webAudioReady`; the page provides no retry path or muted-status indication.~~ **Partially resolved by BUG-001**: Failed activation state is cleared for later retry; a user-visible muted/readiness indication remains out of scope.
4. ~~Automated coverage does not directly assert ambient loop/start/pause behavior, category gain values at the Web Audio node, random selection, cache coalescing, or character-scroll playback.~~ **Resolved by BUG-001 and analysis remediation**: Ambient lifecycle and activation coalescing are covered; T031 verifies every one-shot gain including character scroll, and T032 verifies deterministic selection plus folder/raw load coalescing.
5. ~~The broader migrated presentation plan still describes the removed Express server, while the current manifest endpoint is implemented in Go.~~ **Resolved during post-verification consistency cleanup**: the presentation artifacts now defer to this sound-system contract and describe the embedded Go server.
