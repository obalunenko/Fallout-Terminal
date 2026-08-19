# Feature Specification: CRT Rendering and Motion Effects

**Feature Branch**: `008-crt-rendering`

**Created**: 2026-08-16

**Status**: Draft

**Input**: Use the CRT rendering, animation, and effects behavior present before commit `b84f450e87035862c4333135c4d9baec167936d9` as the experiential baseline for an active feature on the current branch.

**Bugfix**: 2026-08-16 — BUG-001 Added progressive first-render animation for hacking-puzzle code rows.

**Bugfix**: 2026-08-17 — BUG-002 Distinguished same-generation dud removal from puzzle replacement so the hacking board is reconciled without a full rerender.

**Bugfix**: 2026-08-19 — BUG-003 Required a complete-board font fit before hacking reveal and stable code-row typography throughout ordinary row insertion.

## Clarifications

### Session 2026-08-16

- Q: Should players have a reduced-motion mode or only a keyboard shortcut to skip progressive reveal? → A: Do not provide or honor a reduced-motion mode; while the current screen is revealing line by line, pressing any key renders the full current page immediately, while the other CRT effects remain active.
- Q: When a key is pressed during an active reveal, should that key only complete the page, or also perform its normal action? → A: Complete the current page and consume the key; only a subsequent key press may perform its normal action.
- Q: What contrast rule should selected, focused, active, and hacking-hover content meet against the CRT background? → A: Preserve the exact historical colors and validate those states with visual snapshots rather than a numeric contrast threshold.
- Q: How should the subtle intermittent screen flicker be defined for acceptance? → A: Preserve the exact historical six-second cycle: opacity remains 1 through 96%, changes to .92 at 97%, returns to 1 at 98%, changes to .96 at 99%, and returns to 1 at 100%.

## User Scenarios & Testing

### User Story 1 - Experience a Cohesive CRT Display (Priority: P1)

As a player, I see every player-terminal state through one consistent Fallout-style CRT presentation so that connecting, waiting, browsing, reading, and hacking all feel like the same in-world terminal.

**Why this priority**: The CRT identity is the feature's primary player value and must remain recognizable before secondary motion or polish matters.

**Independent Test**: Open the player experience and exercise connection, idle, character selection, assigned waiting, terminal list, record, command output, active hacking, and blocked hacking states; confirm the same frame, phosphor treatment, scanlines, and edge shading remain present without hiding state-specific content.

**Acceptance Scenarios**:

1. **Given** any supported player-terminal state is visible, **When** the state finishes rendering, **Then** it appears inside a dark rounded screen with a green phosphor treatment, subtle glow, scanlines, and edge vignette.
2. **Given** interactive content is visible beneath the CRT treatment, **When** a player points, clicks, taps, or focuses a control, **Then** decorative layers never block the interaction.
3. **Given** a selectable terminal row, character option, page control, or hacking target is active, **When** its selection or focus changes, **Then** the player sees the exact historical state-specific color treatment represented by the approved visual snapshot baseline.
4. **Given** a connection message covers the player experience, **When** the message is displayed, **Then** it remains legible and visually compatible with the terminal while clearly taking precedence over the underlying state.

---

### User Story 2 - Receive Purposeful Terminal Motion (Priority: P2)

As a player, I see restrained flicker, cursor activity, and progressive content reveal so that the terminal feels active without repeatedly delaying reading or interaction.

**Why this priority**: Motion establishes atmosphere, but it must build on a complete and readable static presentation.

**Independent Test**: Enter a new folder, record, and command output; trigger unchanged rerenders and rapid state replacement; then observe flicker, cursor, pending selection, and reveal behavior.

**BUG-001 extension**: Enter a newly generated hacking puzzle, observe its code rows reveal, then trigger unchanged hacking updates, fitting work, reconnect, and puzzle replacement.

**BUG-002 extension**: Activate a delimiter pattern that removes a dud within the current puzzle generation and verify that only the affected board content changes while existing rows and reveal progress remain in place.

**BUG-003 extension**: Enter a newly generated hacking puzzle and verify that the code-row font is fitted from the complete board before the first row appears and does not zoom out as later rows reveal.

**Acceptance Scenarios**:

1. **Given** the player screen is active, **When** each six-second flicker cycle runs, **Then** opacity remains 1 through 96%, changes to .92 at 97%, returns to 1 at 98%, changes to .96 at 99%, and returns to 1 at 100%.
2. **Given** a cursor or pending terminal action is visible, **When** its indicator animates, **Then** it alternates in a deliberate hard step that remains distinguishable from the screen flicker.
3. **Given** a folder, record, or command output is shown for the first time, **When** its content renders, **Then** its rows or lines appear progressively in source order.
4. **Given** the same content rerenders because unrelated player state changed, **When** its identity and text are unchanged, **Then** the content appears immediately without replaying its reveal.
5. **Given** a reveal is still running, **When** another state or content identity replaces it, **Then** the obsolete reveal stops and no stale rows or lines are appended.
6. **Given** a player changes pages within already opened content, **When** the next or previous page appears, **Then** navigation remains immediate and does not replay the initial reveal unnecessarily.
7. **Given** a new hacking-puzzle generation is shown for the first time, **When** its code grid renders, **Then** complete code rows appear one at a time every 40 milliseconds in deterministic DOM source order instead of the full grid appearing at once.
8. **Given** a hacking-puzzle reveal is active, **When** attempts, activity-log content, hover state, viewport size, or font metrics change without replacing the puzzle generation and board text, **Then** the visible rows remain stable and the reveal neither restarts nor duplicates code.
9. **Given** a hacking-puzzle reveal is active, **When** a different puzzle generation replaces it, **Then** the obsolete reveal is cancelled before the new puzzle begins and no stale code row is appended.
10. **Given** a delimiter pattern removes a dud without changing the hacking generation, **When** the authoritative board update arrives, **Then** the affected candidate changes to its removed presentation without clearing, replacing, or replaying the rest of the hacking board.
11. **Given** a new hacking-puzzle generation and unchanged viewport, orientation, and active font metrics, **When** its code rows progressively reveal, **Then** the first visible row already uses the complete board's fitted font size and every later row append preserves exactly that size instead of producing a grow, shrink, or zoom effect.

---

### User Story 3 - Skip Progressive Reveal and Retain Readability (Priority: P3)

As a player who wants to read the current content immediately or uses a compact display, I can skip the current line-by-line reveal with the keyboard and continue using the terminal without clipping or effect-related interference.

**Why this priority**: The atmospheric reveal should remain part of each newly opened screen without forcing a player to wait for it to finish.

**Independent Test**: Start the progressive reveal on a folder, record, and command page; press a key before each reveal completes; then exercise the same player states at representative compact, medium, and large viewports.

**BUG-001 extension**: Start a new hacking-puzzle code-row reveal, press a hacking input key before completion, and verify the board completes without submitting that input; then use a later key normally.

**Acceptance Scenarios**:

1. **Given** the current folder, record, or command page is still revealing rows or lines, **When** the player presses any key, **Then** the active reveal stops, the complete current page appears immediately, and that key performs no navigation, activation, page change, back action, or hacking action.
2. **Given** a reveal was completed by a consumed key press, **When** the player presses another key after the page is complete, **Then** that subsequent key performs its normal action.
3. **Given** the player skipped the reveal on one screen, **When** a different folder, record, command, or page is opened, **Then** that new screen begins its normal reveal and may be skipped independently.
4. **Given** a compact supported viewport, **When** any supported terminal state is displayed, **Then** essential text and controls remain inside the visible screen and available without page-level scrolling.
5. **Given** a viewport or font-metric change causes content to be repaginated or a hacking board to be refitted, **When** the layout settles, **Then** the CRT layers remain aligned with the screen and do not trigger a new content reveal.
6. **Given** hacking code rows are still revealing, **When** the player presses any key, **Then** all remaining rows appear within 100 milliseconds and that physical press and its repeats perform no hacking action; a later physical key press follows the normal hacking contract.

### Edge Cases

- An empty folder still receives the CRT treatment and displays its empty-state message without waiting on a multi-row reveal.
- Empty record or command text remains a visible, stable state rather than leaving a reveal timer active.
- Rapid navigation cancels the outgoing reveal before the replacement view renders.
- A reconnect or authoritative update that preserves the visible content identity does not restart atmospheric motion tied to content entry.
- Long terminal text, zoom changes, fallback fonts, and late font loading may alter pagination but must not displace the decorative layers or leave controls unreachable.
- A key press when no progressive reveal is active does not create or persist a reveal-skip state for later screens.
- A key press consumed to complete an active reveal cannot also move selection, activate content, go back, change pages, or submit a hacking action; normal keyboard handling resumes with the next key press.
- Skipping the current reveal does not disable screen flicker, cursor blinking, pending-state pulsing, scanlines, vignette, or other CRT effects.
- Decorative effects remain noninteractive even when the underlying content uses pointer, keyboard, or touch input.
- A hacking-board row becomes interactive only after that complete row is visible; code targets in unrevealed rows cannot receive pointer, focus, or keyboard interaction.
- Hacking attempt, log, hover, reconnect, viewport, and font-fit updates for the same puzzle generation and board text do not replay completed or in-progress code rows.
- Replacing a hacking generation during its reveal cancels the old generation before any new row is inserted.
- Dud removal in an already revealed row updates that row in place while every unaffected revealed row retains its DOM identity and interaction state.
- Dud removal in a not-yet-revealed row updates the pending row content without revealing it early or restarting the active sequence.
- Hacking row insertion alone never triggers a partial-board refit; a genuine viewport, orientation, or active-font change may refit during reveal only from the complete generation, including queued rows.

## Requirements

### Functional Requirements

- **FR-001**: The player experience MUST present every supported state within one consistent CRT visual shell.
- **FR-002**: The CRT shell MUST provide a dark screen, green phosphor foreground, restrained glow, scanline texture, and edge darkening while preserving text legibility.
- **FR-003**: Decorative CRT layers MUST cover the visible screen without receiving pointer or keyboard interaction.
- **FR-004**: Connection feedback MUST remain visually dominant and readable without permanently replacing or mutating the underlying player state.
- **FR-005**: Selection, focus, active, and hacking-hover feedback MUST preserve the exact historical color treatment and MUST be validated through approved visual snapshot baselines rather than a numeric contrast threshold.
- **FR-006**: Screen flicker MUST use the historical six-second cycle with opacity 1 through 96%, .92 at 97%, 1 at 98%, .96 at 99%, and 1 at 100%.
- **FR-007**: Cursor and pending-action indicators MUST use a discrete on/off presentation that remains independent of screen flicker.
- **FR-008**: Newly opened folder rows, record lines, and command-output lines MUST reveal progressively in source order under normal motion settings.
- **FR-009**: A replacement render MUST cancel any unfinished reveal associated with the replaced content.
- **FR-010**: An unchanged content identity MUST NOT replay progressive reveal solely because another player-state field changed.
- **FR-011**: Pagination, viewport recalculation, and font-fit recalculation MUST NOT restart progressive reveal for already visible content.
- **FR-012**: While a folder, record, or command page is progressively revealing, pressing any keyboard key MUST stop the active reveal, render the complete current page immediately, and consume that key without performing its normal action; normal keyboard handling MUST resume on the next key press.
- **FR-013**: The visual shell and essential controls MUST remain usable without page-level scrolling across the supported compact, medium, and large viewport checks.
- **FR-014**: CRT presentation behavior MUST remain local to the player experience and MUST NOT alter authoritative navigation, hacking, identity, broadcast, or persistence state.
- **FR-015**: Optional audio availability or playback failure MUST NOT prevent the visual shell, animation fallback, or player interaction from working.
- **FR-016**: The player experience MUST NOT expose or honor a reduced-motion mode that disables screen flicker, cursor blinking, pending-state pulsing, or other persistent CRT effects; reveal skipping applies only to the currently rendering page.
- **FR-017**: On the first presentation of a new hacking-puzzle generation, the player experience MUST reveal complete hacking code rows at the existing 40-millisecond reveal cadence in deterministic DOM source order, with the first row appearing immediately and each row's address and code targets becoming visible atomically.
- **FR-018**: ~~Hacking reveal identity MUST include the authoritative puzzle generation and rendered board text; attempts, activity-log content, hover state, reconnect, viewport recalculation, font readiness, and hacking-fit work that preserve that identity MUST NOT restart, replace, or duplicate the reveal.~~ Superseded by BUG-002 because dud removal legitimately changes rendered board text within the same generation. Hacking reveal identity MUST be anchored to the authoritative puzzle generation; attempts, activity-log content, hover state, reconnect, viewport recalculation, font readiness, hacking-fit work, and same-generation authoritative board mutations MUST NOT restart, replace, or duplicate the reveal.
- **FR-019**: ~~Replacing an active hacking reveal with a different puzzle generation or board text MUST cancel the obsolete generation before clearing or appending code rows, and stale callbacks MUST append zero elements.~~ Superseded by BUG-002 because changed board text alone does not identify a replacement puzzle. Replacing an active hacking reveal with a different authoritative puzzle generation MUST cancel the obsolete generation before clearing or appending code rows, and stale callbacks MUST append zero elements.
- **FR-020**: While hacking code rows are progressively revealing, pressing any keyboard key MUST render every remaining row within 100 milliseconds and consume that physical press and its repeats without submitting a guess, pattern, typed character, back action, or other hacking action; normal hacking input MUST resume with a later physical key press.
- **FR-021**: An unrevealed hacking code row and all of its targets MUST be absent from the interaction surface until the complete row is revealed, and reveal bookkeeping MUST remain browser-local presentation state that never mutates authoritative hacking state.
- **FR-022**: Within an unchanged hacking generation, authoritative dud removal MUST reconcile only the affected board content and related pattern/log presentation without clearing or rebuilding the hacking board, replacing unaffected revealed row nodes, or restarting active reveal timing. If the affected row is unrevealed, its updated content MUST appear at its existing reveal position and MUST remain outside the interaction surface until then.
- **FR-023**: Before the first code row of a new hacking generation becomes visible, the player experience MUST calculate one shared hacking-row font size from the complete generation, including every queued row, under the current viewport, layout orientation, activity-log allocation, and active font metrics. Ordinary reveal appends, reveal completion, skip completion, reconnect, hover, attempts, log updates, and same-generation reconciliation MUST reuse that size without visible grow, shrink, or zoom. A genuine viewport, orientation, or active-font change MAY recalculate the size, but the recalculation MUST still use the complete generation and preserve the responsive single-screen font floor, containment, and maximal-fit rules.

### Impacted Application Surfaces

- **Composition and Wails bridge (`main.go`, `app.go`)**: Not affected; the feature does not add desktop capabilities or lifecycle operations.
- **Domain and canonical state (`internal/domain/`, `internal/nav/`, `internal/hack/`, `internal/live/`, `internal/control/`)**: Not affected; all authoritative state and rules remain unchanged.
- **Persistence (`internal/session/`, `internal/playerconfig/`, `sessions/`)**: Not affected; no stored fields or document versions change.
- **Player transport (`internal/player/`)**: Not affected at the protocol level; existing static delivery and player streams continue supplying the same states.
- **Platform and public access (`internal/platform/`, `internal/tunnel/`)**: Asset verification is affected; runtime, tunnel, credential, and public-access behavior are not.
- **Master UI (`frontend/src/`)**: Not affected.
- **Player UI (`client/`)**: Affected; this surface owns the visual layers, presentation motion, reveal lifecycle, and keyboard reveal-skip behavior.
- **Tests and fixtures (`internal/**/*_test.go`, `tests/browser/`, `internal/testutil/`)**: Affected; focused asset and browser coverage is required for presentation states, key-driven reveal completion, and reveal behavior.
- **Build and packaging (`go.mod`, `frontend/`, `wails.json`, `build/`, `scripts/`)**: Existing player-client build and embedded-asset gates apply; no dependency or packaging design change is required.

### State and Contract Requirements

- **Session/player-config compatibility**: No stored schema or compatibility change.
- **Wails bridge and event contract**: No change.
- **Player service contract**: No message, method, field, ordering, or authorization change; the feature consumes existing authoritative snapshots and updates.
- **Reconnect and multi-tab behavior**: Existing convergence behavior remains authoritative; each tab may skip only its own active reveal without changing shared state.
- **HTTP/static contract**: Existing player asset delivery continues to serve the visual resources; no new route is required.
- **Runtime-state lifecycle**: Effect, reveal, and reveal-skip bookkeeping remains transient per browser tab and is cleared or replaced with the visible presentation state.

### Security and Privacy Requirements

- The feature MUST NOT add filesystem, native desktop, tunnel, credential, or private game-master capabilities to the player experience.
- User-authored content MUST retain its existing literal-text or escaped rendering behavior while reveal effects are applied.
- The restrictive content policy and same-origin asset policy MUST remain intact.
- Effect and reveal-skip bookkeeping MUST remain local browser presentation state and MUST NOT be persisted in session data or published to other players.

### Verification Requirements

- **Go tests**: Extend focused embedded-player asset checks to protect the required CRT layer, interaction, and keyboard reveal-skip contracts.
- **Race testing**: Not required specifically for browser-only presentation behavior; the normal repository race gate remains unchanged.
- **Browser tests**: Add a focused player journey covering representative states, folder/record/command and BUG-001 hacking-code reveal, replay/cancellation, key-driven reveal completion, BUG-002 same-generation dud reconciliation, BUG-003 complete-board pre-fit and reveal-time font stability, hit testing, and supported viewport checks.
- **Interactive verification**: Run the current development entrypoint and inspect reveal and reveal-skip journeys with list, record, command, pagination, and first-entry/replacement hacking states.
- **Packaging/release verification**: Run the normal player build and unsigned application build gates because embedded player assets change; signing and public-provider checks are not feature-specific.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All nine defined player presentation states use the same recognizable CRT shell without losing their state-specific text or controls.
- **SC-002**: Every decorative overlay hit-test performed over an interactive target reaches the intended underlying control, with zero overlay-intercepted interactions.
- **SC-003**: A newly opened list of 25 rows completes its ordered reveal within 1.2 seconds when uninterrupted; pressing a key during that reveal displays all remaining rows within 100 milliseconds and produces zero navigation, activation, page-change, back, or hacking actions from the consumed key.
- **SC-004**: Repeated updates that preserve visible content identity cause zero additional reveal sequences, and replacing content mid-reveal produces zero stale appended elements.
- **SC-005**: Skipping one active reveal completes only the current page, leaves all other CRT animations active, and causes the next newly opened screen to begin a fresh reveal.
- **SC-006**: Connection, idle, selection, list, record, command, paginated content, active hacking, and blocked hacking journeys remain readable and operable at 360×640, 768×720, and 1440×900 viewport checks without page-level scrolling.
- **SC-007**: Selection, focus, active, and hacking-hover states match their approved historical-color visual snapshots at every viewport where each state is exercised.
- **SC-008**: The player screen completes exactly one historical flicker sequence every six seconds and matches all five specified opacity checkpoints during each observed cycle.
- **SC-009**: A controlled new hacking puzzle shows fewer than all code rows on its first render, reveals complete rows in deterministic DOM source order at the 40-millisecond cadence, and displays all remaining rows within 100 milliseconds after a key press while producing zero actions from that consumed press and its repeats.
- **SC-010**: Attempts, log, hover, reconnect, viewport, font-readiness, and fit updates preserving the hacking generation and board text produce zero additional row-reveal sequences, while replacing the generation mid-reveal produces zero stale appended rows and exactly one reveal for the replacement puzzle.
- **SC-011**: A controlled same-generation dud-removal update removes zero hacking rows, starts zero additional reveal sequences, preserves the exact DOM identity of every unaffected revealed row, updates the affected candidate presentation, and leaves an affected unrevealed row unavailable until its original reveal turn.
- **SC-012**: At every supported normal and compact/stacked viewport, the 200%-zoom case, and bundled- and fallback-font rendering, a controlled new hacking generation uses one computed row font size from its first visible row through uninterrupted or skipped completion, with zero append-driven size changes; after a genuine viewport, orientation, or font change, any new size is calculated from all rows and still satisfies the inherited complete-board font-floor, containment, and maximal-fit tolerance.

## Assumptions

- The historical green Fixedsys-inspired terminal treatment remains the intended visual baseline.
- Historical selection, focus, active, and hacking-hover colors take precedence over adopting a new numeric contrast target in this feature.
- The historical six-second flicker timing and opacity checkpoints are intentional product behavior rather than tunable presentation defaults.
- The supported player environment is a modern browser with keyboard input for reveal skipping.
- Screen flicker, scanlines, vignette, cursor blink, and progressive folder, text-line, and hacking-code-row reveal are atmospheric enhancements rather than authoritative state.
- Existing pagination and hacking-layout fitting remain part of the current player presentation and must coexist with the CRT treatment.
- Hacking board text and word metadata may change authoritatively within one puzzle generation when a dud is removed; that mutation does not identify a replacement puzzle.
- Audio feedback may accompany reveal and interaction, but its discovery, autoplay, volume, and lifecycle behavior remain governed by the sound-system feature.

## Out of Scope

- Changes to player RPC schemas, server streaming, navigation rules, hacking rules, character identity, or controller authority.
- Changes to the game-master interface, session persistence, public access, credentials, or packaging architecture.
- New WebGL shaders, canvas rendering, video effects, 3D tube curvature, chromatic aberration, randomized noise, or additional media dependencies.
- A player-facing or operating-system-driven reduced-motion mode for disabling persistent CRT effects.
- Redesigning player-facing copy or adding localization.
- Redesigning the sound system beyond preserving visual behavior when audio is unavailable.
