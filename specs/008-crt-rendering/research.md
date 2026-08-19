# Research: CRT Rendering and Motion Effects

**Bugfix**: 2026-08-16 — BUG-001 Extended reveal identity and row construction to new hacking generations.

**Bugfix**: 2026-08-17 — BUG-002 Separated puzzle-generation identity from same-generation dud-removal reconciliation.

**Bugfix**: 2026-08-19 — BUG-003 Added complete-board pre-fit and reveal-time hacking typography stability.

## Decision 1: Retain the historical DOM/CSS composition

**Decision**: Keep the existing HTML/CSS screen composition and protect the exact historical palette, glow, scanlines, vignette, six-second flicker checkpoints, and hard-step blink/pulse behavior.

**Rationale**: The parent of `b84f450e87035862c4333135c4d9baec167936d9` and the current client use the same browser-native treatment. It preserves semantic DOM text and controls, ordinary hit testing, responsive layout, and the current ConnectRPC player architecture without a rendering dependency.

**Alternatives considered**:

- Canvas or WebGL: rejected because it would complicate semantic content, responsive fitting, and browser testing without improving the requested historical result.
- Randomized or tunable effects: rejected because the clarified acceptance contract pins exact colors and flicker checkpoints.
- A redesigned accessible palette: rejected for this feature because the user selected exact historical-color snapshots rather than a numeric contrast target.

## Decision 2: Keep persistent CRT motion always active

**Decision**: Do not implement a player setting, persisted preference, or `prefers-reduced-motion` branch. Screen flicker, cursor blink, assigned-waiting blink, pending-selection pulse, scanlines, and vignette continue independently of reveal skipping.

**Rationale**: The clarified product requirement explicitly replaces motion reduction with a one-shot way to finish the currently revealing page. A motion preference would create a second presentation mode that contradicts `FR-016` and could silently disable the historical effects.

**Alternatives considered**:

- Honor the operating-system reduced-motion preference: rejected because the specification explicitly says not to expose or honor that mode.
- Add an application toggle: rejected because no configurable or persisted motion mode is allowed.
- Disable every effect when reveal is skipped: rejected because reveal completion must not affect persistent CRT effects.

## Decision 3: Represent reveal as an explicit controller

**Decision**: Replace timer-only bookkeeping with an idempotent, container-scoped reveal controller holding ordered nodes, next index, generation, pending timer, and state, with explicit complete and cancel operations.

**Rationale**: A timer alone cannot safely complete all remaining nodes on demand or prove that stale callbacks are invalidated. A controller can synchronously append the remainder within the 100ms target, cancel replacement work, unregister itself, and preserve the existing 40ms uninterrupted cadence.

**Alternatives considered**:

- Query hidden source content from the DOM on keydown: rejected because unrevealed nodes are not present and reconstructing them risks order or escaping drift.
- Set all timer delays to zero after a key: rejected because queued callbacks are harder to cancel and do not guarantee one synchronous completion.
- Persist a global “skip animations” flag: rejected because later content identities must reveal normally.

## Decision 4: Consume the physical key before normal input handling

**Decision**: Use a capture-phase reveal-skip guard. When an active visible-page reveal exists, the guard completes it, prevents the event’s default action, stops further handling for that event, and suppresses auto-repeat from the same physical key until key release.

**Rationale**: The first key must not also navigate, activate, change page, go back, submit hacking input, or trigger a focused control. Treating held-key repeats as part of the consumed physical press satisfies “only a subsequent key press may perform its normal action,” while a genuinely later press can use the unchanged keyboard handler.

**Alternatives considered**:

- Put the check only at the top of the existing bubble-phase handler: rejected because target listeners or native control behavior may already have observed the event.
- Complete the reveal and continue dispatch: rejected because one key could both skip and mutate authoritative state.
- Swallow all later keys for a fixed timeout: rejected because it would make legitimate subsequent input feel unresponsive.

## Decision 5: Keep content identity distinct from pagination layout

**Decision**: ~~A newly opened folder, record, or command identity may reveal once.~~ Under BUG-001, ~~a newly opened folder, record, command, or authoritative hacking-generation-plus-board identity may reveal once.~~ BUG-002 narrows hacking reveal identity to the authoritative puzzle generation; rendered board content is a mutable generation-local snapshot. Pagination within already-opened text, viewport repagination, font readiness, hacking fitting, and same-generation dud removal do not create a new reveal identity.

**Rationale**: This preserves `FR-010` and `FR-011` and reconciles “complete the current page” with the existing immediate-pagination contract. The current page is the completion boundary, not a persisted preference; a later new content identity gets an independent reveal.

~~For hacking, attempts, log lines, hover, reconnect, and fitting can update without changing the generation or rendered board. Keying the reveal to generation plus board text preserves partially or fully revealed rows through those updates while still making a replacement puzzle visibly new.~~ BUG-002 records that dud removal changes rendered board text and word metadata inside the same generation. Generation identity therefore controls replacement, while a separate board snapshot drives incremental reconciliation and preserves partially or fully revealed rows.

**Alternatives considered**:

- Reveal every pagination page: rejected because page navigation and repagination would repeatedly delay already-opened content.
- Key identity only by DOM container: rejected because a reused container displays different authoritative content over time.
- Replay on every stream update: rejected because unrelated authoritative revisions must not delay reading.
- Treat every board-text change as replacement: rejected by BUG-002 because dud removal is an ordinary same-generation gameplay mutation.

## Decision 6: Split animation contracts from stable visual snapshots

**Decision**: Verify duration, iteration, keyframe offsets/opacities, reveal timing, and event behavior with browser animation/DOM APIs. Capture approved historical-color screenshots only after freezing animations at a defined stable phase.

**Rationale**: A live flicker or blink phase makes screenshots nondeterministic. Separating temporal assertions from frozen selection, focus, active, and hacking-hover snapshots provides exact evidence without replacing the user-selected visual baseline with a numeric contrast rule.

**Alternatives considered**:

- Screenshot live animations: rejected because the sampled phase varies.
- Assert only CSS source strings: rejected because source checks cannot prove rendered geometry, hit testing, or keyboard behavior.
- Use only computed color values: rejected because the clarified acceptance method is an approved visual snapshot baseline.

## Decision 7: Exercise authored-content and audio failure paths in the focused journey

**Decision**: Give the browser fixture literal markup-like authored strings and simulate unavailable/rejected browser audio while reveal and skip behavior runs.

**Rationale**: Refactoring insertion and keyboard timing must not regress text-only rendering or allow optional audio setup/playback failures to interrupt visual completion and input. These are explicit requirements and need observable browser evidence rather than assumptions about the existing sound module.

**Alternatives considered**:

- Rely only on existing asset-string checks: rejected because they do not execute the new reveal path.
- Assert a specific number of reveal sound calls: rejected because sound discovery, autoplay, and playback lifecycle remain outside this feature.
- Remove reveal audio integration: rejected because audio is optional but existing behavior need not be redesigned.

## Decision 8: Reveal complete hacking code rows

**Decision**: Build the hacking grid through safe DOM boundaries and reveal each complete code row, including its address and targets, at the existing 40ms cadence in deterministic DOM source order. Do not reveal individual characters or expose incomplete rows to interaction.

**Rationale**: Whole-row units reuse the established row/line motion language, keep password and pattern targets structurally intact, avoid partially focusable targets, and provide a visible initial animation without changing authoritative hacking rules.

**Alternatives considered**:

- Render the full grid atomically: rejected by BUG-001 because it provides no hacking-screen rendering animation.
- Reveal individual characters: rejected because it creates transient partial targets and substantially complicates focus, hover, and pattern-span behavior.
- Animate only opacity on an already interactive grid: rejected because unrevealed targets would remain reachable and completion/cancellation would not share the reveal controller contract.

## Decision 9: Reconcile dud removal by stable row coordinates

**Decision**: Retain a generation-local board snapshot and reconcile authoritative dud-removal changes by stable column and row coordinates. Update affected content inside an existing revealed row, or update its queued descriptor if it has not yet reached the DOM; preserve every unaffected row node and the active controller's schedule.

**Rationale**: Dud removal replaces one candidate with dots and removes its word metadata without changing the puzzle generation. Coordinate reconciliation reflects that authoritative mutation while preventing a whole-board teardown, reveal replay, focus loss, and premature interaction with a queued row.

**Alternatives considered**:

- Use rendered board text as reveal identity: rejected because it conflates same-generation gameplay with puzzle replacement.
- Rebuild every row immediately without animation: rejected because it still destroys node identity, focus, hover, and active reveal progress.
- Defer every dud update until the initial reveal completes: rejected because already revealed content would temporarily disagree with authoritative state.

## Decision 10: Fit the complete hacking board before progressive painting

**Decision**: Calculate the shared hacking-row font from the complete generation-local board snapshot before the reveal controller paints its first row. Retain that fit through ordinary row appends and uninterrupted or skipped completion. Recalculate only when a genuine viewport, layout-orientation, or active-font input changes, and always measure the complete snapshot, including queued rows.

**Rationale**: The responsive fitter's height constraint is meaningful only for the complete fixed-row board. Measuring the interaction DOM during progressive reveal sees only the rows already appended, temporarily permits an oversized font, and then shrinks it as more rows arrive. A generation-local pre-fit preserves stable typography while retaining the responsive base-size floor, one-size rule, complete-board containment, and maximal-fit tolerance. The snapshot can participate in measurement without placing unrevealed targets in the interaction surface.

**Alternatives considered**:

- Refit after every appended row: rejected by BUG-003 because partial-board measurements produce the visible zoom-out effect.
- Render every row invisibly in the interactive columns before reveal: rejected because unrevealed targets must remain absent from pointer, focus, and keyboard interaction and hidden live DOM complicates that guarantee.
- Hold the base hacking role until reveal completes and enlarge once: rejected because it creates a second size jump and fails to use the required maximal complete-board fit from the first visible row.
