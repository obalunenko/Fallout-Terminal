# UI Contract: CRT Rendering and Motion Effects

**Bugfix**: 2026-08-16 — BUG-001 Added progressive reveal for new hacking-generation code rows.

**Bugfix**: 2026-08-17 — BUG-002 Added same-generation dud-removal reconciliation without board replay.

**Bugfix**: 2026-08-19 — BUG-003 Added complete-board pre-fit and stable hacking-row typography during reveal.

## Scope

This contract governs observable player-browser presentation. It does not alter the player RPC, authoritative state, session JSON, player configuration, Wails bridge, static routes, public-access boundary, or private capabilities.

## Visual Shell Contract

Every supported player state renders inside `#screen` with:

- dark green-black background `#020a02` and border `#0c2e0c`;
- green phosphor foreground `#57ff6e` with the existing restrained glow;
- the existing rounded frame and responsive bounded layout;
- `.scanlines` and `.vignette` covering the same screen bounds;
- a connection overlay that remains readable and dominant when visible.

`.scanlines` and `.vignette` MUST be explicitly decorative, excluded from keyboard focus and the accessibility tree, and use `pointer-events: none`. A hit test over either layer and an interactive target MUST reach the target.

## Historical Color Snapshot Contract

Acceptance uses approved visual snapshots rather than a numeric contrast threshold. The snapshots protect these current historical states:

| State | Existing selector/state | Historical treatment |
|---|---|---|
| Selected terminal row | `.term-row.sel` | `#57ff6e` background, `#021002` foreground, no text shadow |
| Focused character option | `.character-option:focus-visible` | `#d8ffb8` border and `rgba(87, 255, 110, .16)` background |
| Active character option | `.character-option:active` | `#57ff6e` background, `#021002` foreground, no text shadow |
| Focused navigation control | `.back-btn:focus-visible`, `.page-btn:focus-visible` | `#d8ffb8` outline and `rgba(87, 255, 110, .12)` background |
| Hacking hover/focus | `.hcell.hi` | `#57ff6e` background, `#021002` foreground, no text shadow |

Snapshots are captured at a defined stable animation phase and reviewed as repository artifacts. They MUST NOT be replaced by a new palette or a numeric WCAG threshold within this feature.

## Persistent Motion Contract

The player has one CRT motion mode. It MUST NOT expose or honor an application or operating-system reduced-motion path that disables persistent effects.

### Flicker

`#screen` uses one infinite `flicker` animation with a six-second duration and these exact opacity checkpoints:

| Offset | Opacity |
|---:|---:|
| 0% through 96% | 1 |
| 97% | .92 |
| 98% | 1 |
| 99% | .96 |
| 100% | 1 |

### Hard-step indicators

- `.blink` and the assigned-waiting indicator use the existing one-second hard on/off step.
- Pending character selection uses the existing one-second stepped border change.
- These effects remain independent of flicker and reveal completion.

## Progressive Reveal Contract

~~The revealable containers are `#termList`, `#entryBody`, and `#termOutput`.~~ Under BUG-001, the active hacking-code container is also revealable.

- A newly opened folder, record, or command identity reveals rows or lines in source order.
- ~~A new authoritative hacking-generation-plus-board identity reveals complete address-and-code rows at the same cadence in deterministic DOM source order.~~ Under BUG-002, a new authoritative hacking generation reveals complete address-and-code rows at that cadence in deterministic DOM source order; a row and its targets enter the interaction surface atomically.
- The first element may appear immediately; each following element is scheduled at the historical `REVEAL_DELAY_MS` value of 40ms.
- Before the first hacking row is painted, one shared row font size MUST be calculated from the complete generation, including queued rows, under the current viewport, orientation, activity-log allocation, and active font metrics.
- Ordinary hacking row appends and uninterrupted or skipped completion MUST reuse that size with zero visible grow, shrink, or zoom; a genuine viewport, orientation, or active-font change may recalculate it only from the complete generation and MUST retain the responsive font floor, containment, and maximal-fit tolerance.
- An uninterrupted 25-row list MUST be complete within 1.2 seconds.
- An unchanged identity renders immediately and MUST NOT replay because of an unrelated authoritative update.
- Replacement content cancels and invalidates the prior sequence before clearing the container; stale callbacks append nothing.
- Pagination within already-opened text, viewport repagination, font readiness, and hacking layout fitting render immediately and do not create a new reveal identity.
- Attempts, activity-log content, hover state, reconnect, viewport, font readiness, fit work, and same-generation dud removal retain active or completed row progress; a replacement generation cancels stale work before starting one new reveal.
- Dud removal reconciles only affected content inside an existing revealed row or its queued descriptor, removes no unrelated row, preserves unaffected row node identity, and never exposes a queued row before its original reveal turn.
- Empty content resolves to a visible stable empty state with no live timer.

## Reveal-Skip Keyboard Contract

While any reveal controller belongs to the visible page:

1. Any keyboard key completes every remaining row or line for that page in source order within 100ms.
2. The triggering event is consumed before normal terminal handling and performs no navigation, activation, page change, back action, focused-control action, or hacking action.
3. Auto-repeat keydown events from that same physical press remain consumed until its matching key release.
4. A different later physical key press follows the existing normal keyboard behavior.
5. No skip preference or completed state carries into a newly opened folder, record, command, or hacking-generation identity.
6. Flicker, blink, pulse, scanlines, vignette, palette, glow, and optional audio lifecycle remain independent of reveal completion.

A key pressed when no reveal is active does not set a future skip state.

## State/Effect Matrix

| State | Frame/glow | Scanlines/vignette | Flicker | Blink/pulse | Progressive reveal |
|---|---:|---:|---:|---:|---:|
| Connecting/reconnecting | Under status layer | Yes | Exact six-second cycle | No | No |
| Idle | Yes | Yes | Exact six-second cycle | Waiting cursor | No |
| Character selection | Yes | Yes | Exact six-second cycle | Pending selection only | No |
| Assigned waiting | Yes | Yes | Exact six-second cycle | Waiting indicator | No |
| Folder list | Yes | Yes | Exact six-second cycle | Prompt cursor | New folder identity only |
| Record | Yes | Yes | Exact six-second cycle | Prompt cursor | New record identity only |
| Command output | Yes | Yes | Exact six-second cycle | Prompt cursor | New command identity only |
| Hacking | Yes | Yes | Exact six-second cycle | Input cursor | ~~No~~ ~~New hacking-generation-plus-board identity only (BUG-001)~~ New hacking generation only; dud removal reconciles in place (BUG-002) |
| Blocked | Yes | Yes | Exact six-second cycle | No | No |

## Responsive Contract

At 360×640, 768×720, and 1440×900:

- the screen remains inside the viewport without page-level scrolling;
- scanlines and vignette share the screen bounds;
- essential text and controls remain visible and reachable;
- connection status remains readable above the screen;
- pagination and same-generation hacking fitting may adjust layout but do not replay already-opened content or code rows.
- progressive hacking row insertion does not itself trigger fitting from the partial interaction DOM; any legitimate refit includes both revealed and queued rows.

## Content, Capability, and Audio Contract

- Authored names, descriptions, command text, and log content continue to render literally through `textContent` or the existing escaping boundary; markup-like strings do not create executable or interactive DOM.
- The restrictive CSP, same-origin assets, and current public/private capability separation remain unchanged.
- No CRT or reveal code gains Wails, filesystem, credential, tunnel, or private game-master access.
- Sound discovery, decode, autoplay, device, or playback failure is non-fatal and cannot prevent visual rendering, reveal completion, or keyboard use.
- This feature defines no count or ordering contract for reveal sound calls.

## Browser-Test Observables

Tests may code against these existing identifiers: `#screen`, `.scanlines`, `.vignette`, `.blink`, `#termList`, `.term-row`, `.term-row.sel`, `#termEntry`, `#entryBody`, `#termOutput`, `#backBtn`, `#pagePrev`, `#pageNext`, `#hackBoard`, `#hackColumns`, `.hack-row`, `.hcell.hi`, `#hackBlocked`, and `#connOverlay`.

Implementation-only timer, controller, registry, board-snapshot, board-fit, and consumed-key fields remain private and MUST be verified through visible content, computed row styles, row-node identity, mutation records, event outcomes, geometry, and cancellation behavior rather than a new public browser API.
