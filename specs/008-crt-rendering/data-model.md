# Data Model: CRT Rendering and Motion Effects

**Bugfix**: 2026-08-16 — BUG-001 Added ephemeral hacking-generation reveal state.

**Bugfix**: 2026-08-17 — BUG-002 Added same-generation hacking-board reconciliation state.

**Bugfix**: 2026-08-19 — BUG-003 Added generation-local complete-board font-fit state.

This feature adds no persistent entity, Protobuf message, RPC payload, session field, or player-configuration field. The model below describes ephemeral presentation state owned independently by each browser tab.

## Presentation Mode

The visible player surface remains derived from existing connection state and authoritative player projections.

| Value | Meaning | CRT/reveal expectation |
|---|---|---|
| Connecting | Initial connection or reconnect feedback | Connection overlay is dominant; underlying CRT shell stays intact |
| Idle | Connected with no active terminal | CRT frame and hard-step waiting cursor; no reveal |
| Character selection | Player identity requires assignment | CRT frame, historical focus/active colors, and pending pulse; no reveal |
| Assigned waiting | Character assigned but no terminal live | CRT frame and hard-step waiting indicator; no reveal |
| List | Folder contents visible | A new folder identity may reveal ordered rows |
| Record | Record title/body visible | A new record identity may reveal ordered lines |
| Command output | Command result visible with list state | A new command identity may reveal ordered lines |
| Hacking | Active public hacking board visible | ~~CRT frame, historical hover highlight, log, and cursor; no reveal~~ BUG-001: a new generation reveals complete code rows once; BUG-002: same-generation dud removal reconciles affected content without replay; BUG-003: the complete board is fitted before reveal and retains stable row typography during insertion |
| Blocked | Failed hacking state visible | CRT frame and blocked message; no reveal |

Presentation mode is not persisted, published, or used as an optimistic gameplay state.

## Reveal Sequence

One sequence owns one revealable container for the current visible content identity.

| Field | Meaning | Validation |
|---|---|---|
| `container` | `#termList`, `#entryBody`, `#termOutput`, or the active hacking-code container | Must be connected and belong to the current visible page |
| `contentIdentity` | Existing folder path, record ID, command ID, or ~~authoritative hacking generation plus rendered board text~~ authoritative hacking generation under BUG-002 | Unchanged generation does not replay even when dud removal mutates board text |
| `boardSnapshot` | Generation-local hacking column text, addresses, and word metadata | Compared only to reconcile same-generation changed rows; never creates a new reveal identity |
| `boardFit` | Shared hacking-row font size plus the viewport, orientation, activity-log allocation, and active-font inputs used to calculate it from the complete `boardSnapshot` | Exists before the first hacking row is painted; ordinary row append or completion never invalidates it |
| `pageIndex` | Visible pagination index when applicable | Pagination of already-opened text renders immediately |
| `elements` | Ordered row or line nodes built through safe text boundaries; a hacking element is one complete address-and-code row | Order is immutable for the sequence; partial hacking rows are invalid |
| `nextIndex` | Next node to append | Integer from 0 through `elements.length` |
| `generation` | Container-local render generation | A stale callback cannot match a replacement generation |
| `timer` | At most one pending 40ms continuation | Cleared on complete, cancel, or replacement |
| `state` | Inactive, Revealing, Completing, Complete, or Cancelled | Complete and cancel operations are idempotent |

### State transitions

```text
Inactive
  ├─ new folder/record/command identity with elements → Revealing
  ├─ new hacking-generation identity with complete boardSnapshot → calculate boardFit from every row → Revealing
  └─ unchanged identity or layout-only render → Complete (all nodes immediate)

Revealing
  ├─ 40ms continuation + current generation → Revealing
  ├─ genuine viewport/orientation/active-font change → recalculate boardFit from complete boardSnapshot → Revealing
  ├─ same-generation dud removal → Reconcile affected revealed or queued row → Revealing
  ├─ final node appended → Complete
  ├─ skip key for visible page → Completing → Complete
  └─ replacement/teardown → Cancelled

Complete
  ├─ unchanged update/pagination/resize/font fit → Complete (no replay)
  ├─ genuine hacking viewport/orientation/active-font change → recalculate boardFit from complete boardSnapshot → Complete
  ├─ same-generation dud removal → Reconcile affected revealed row → Complete
  └─ new content identity or hacking generation → Inactive/new sequence

Cancelled
  └─ stale callback → no operation
```

Completing appends every remaining node synchronously in source order and does not require optional audio playback to succeed.

Reconciliation retains the current sequence and schedule. A revealed hacking row is updated inside its existing row node; an unrevealed row remains queued and has its descriptor updated before its original append turn. No reconciliation operation registers a new controller or makes pending targets interactive.

For hacking, `boardFit` is established from the complete snapshot before the first append. The reveal controller applies the retained value to every row, including rows appended during synchronous skip completion. Appending or reconciling a row does not invalidate the value. Only changed viewport, orientation, or active-font inputs may request recalculation, and recalculation still evaluates all snapshot rows without adding queued targets to the interaction surface.

## Active Page Reveal Registry

The registry answers whether the currently visible page has reveal work that a key may complete.

| Field | Meaning |
|---|---|
| `pageToken` | Local render-generation token for the current visible page |
| `controllers` | Set of active Reveal Sequences owned by that page |

The registry is cleared when all sequences complete, the visible page is replaced, the terminal is cleared, or the tab is closed. A skip completes every active controller registered to the visible page; it never affects a hidden, cancelled, previous, or future page.

## Consumed Key Guard

The guard prevents the physical key used to skip a reveal from also performing a normal action.

| Field | Meaning | Lifecycle |
|---|---|---|
| `physicalKey` | `KeyboardEvent.code` with a stable local fallback when unavailable | Set when a key completes an active page reveal |
| `active` | Whether repeats from the consumed press are still suppressed | True until matching key release or terminal teardown |

### State transitions

```text
Idle
  └─ keydown while page reveal active
       → complete visible page
       → prevent default and stop further handling
       → Consuming(physicalKey)

Consuming(physicalKey)
  ├─ repeated keydown for same physical key → consume; no terminal action
  ├─ keyup for same physical key → Idle
  └─ different later physical keydown → ordinary existing keyboard handling
```

The guard is not a preference and does not disable later reveals or persistent CRT effects.

## CRT Decorative and Motion Layers

| Layer/effect | Visual responsibility | Interaction/lifecycle responsibility |
|---|---|---|
| Screen frame | Dark green-black background, border, rounded clipping, glow, phosphor text | Contains all player states |
| Scanlines | Historical repeating horizontal texture | Decorative, `aria-hidden`, pointer-transparent |
| Vignette | Historical edge darkening | Decorative, `aria-hidden`, pointer-transparent |
| Flicker | Exact historical six-second opacity sequence | Always active while the screen is active |
| Blink/pulse | One-second hard-step cursor/waiting/pending indication | Always active when the related indicator is visible |
| Connection overlay | Connection/reconnect status above the CRT screen | Does not mutate underlying authoritative state |

## Validation Invariants

- At most one current reveal sequence exists per reveal container and generation.
- A replacement invalidates the old generation before clearing or appending nodes.
- A stale or cancelled timer callback appends zero nodes.
- Completion is idempotent and leaves no reveal timer or active registry entry.
- The consumed key and its same-press repeats produce zero navigation, activation, page, back, or hacking actions.
- A later physical key press uses the existing normal keyboard contract.
- Unchanged content, pagination, same-generation hacking attempts/log/hover/reconnect/dud-removal updates, viewport recalculation, font readiness, and hacking fitting do not replay a reveal.
- Each revealed hacking element contains one complete address-and-code row; unrevealed rows contribute no pointer, focus, or keyboard target.
- A new hacking generation has a complete-board `boardFit` before its first row is visible, and every ordinary reveal append through uninterrupted or skipped completion uses exactly that value.
- Row append, hover, attempts, log updates, reconnect, and same-generation reconciliation do not invalidate `boardFit`; a genuine viewport, orientation, or active-font change recalculates it from all revealed and queued rows while preserving the responsive font floor, containment, and maximal-fit tolerance.
- ~~A changed hacking generation or rendered board invalidates the previous reveal before any replacement row is appended.~~ Under BUG-002, only a changed hacking generation invalidates the previous reveal; changed board content within the same generation is reconciled.
- Same-generation dud removal removes zero unrelated rows, preserves every unaffected revealed row node, and updates a pending row without appending it early.
- Screen flicker, blink, pulse, scanlines, vignette, palette, and glow have no reduced-motion branch.
- Authored rows and lines remain literal text or pass through the existing escaping boundary.
- Optional audio failure cannot change reveal state, suppress completion, or block input.
- No presentation state enters RPC payloads, authoritative projections, session JSON, or player configuration.
