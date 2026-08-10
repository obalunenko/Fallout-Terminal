# Implementation Plan: Responsive Desktop Player Layout

> The implementation approach is defined in [spec.md § Approach](./spec.md#approach); execution is tracked in [tasks.md](./tasks.md).

**Bugfix**: 2026-08-09 — BUG-001 Updated from bugfix patch
**Bugfix**: 2026-08-10 — BUG-002 Updated from bugfix patch
**Bugfix**: 2026-08-10 — BUG-003 Updated from bugfix patch
**Bugfix**: 2026-08-10 — BUG-004 Updated from bugfix patch

## BUG-001 Implementation Notes

- Replace the independent font-role clamps in `client/client.css` with one responsive scale factor and fixed role multipliers so all text scales proportionally while retaining the existing hierarchy and 16-pixel primary-text minimum.
- Remove browser and player-region scrollbar behavior. Fit bounded terminal states within the viewport and route information text that exceeds its content region through discrete CRT-style pages.
- Implement page calculation and state in `client/client.js`; expose a current/total indicator plus boundary-aware previous and next controls through `client/index.html` and `client/client.css`; preserve the existing header, back navigation, focus visibility, and terminal behavior.
- Recalculate pagination when content or viewport dimensions change. Open information views on page one, clamp the current page after recalculation, and support the same actions by keyboard.
- Extend `internal/platform/assets_test.go` and browser verification to cover common font scaling, absence of page and localized scrollbars, one-page/exact-boundary/multi-page text, page boundaries, and 200% zoom.

## BUG-001 Complexity Notes

- Page capacity depends on rendered font metrics, the shared responsive scale, viewport height, persistent header/footer space, and fallback-font variance; pagination must measure the actual available content region without introducing layout feedback loops.
- Responsive repagination must keep content reachable when a resize or zoom change reduces the page count, and must not disrupt menu, back, hacking, reveal, or audio behavior.

## BUG-002 Implementation Notes

- Raise the hacking font role to at least the primary terminal-text size at every supported viewport while retaining the shared responsive scale factor.
- Fit the complete hacking interaction on one screen by adapting column and row gaps, board padding, activity-log allocation, and compact orientation before considering any text reduction.
- Keep pagination scoped to information views. Hacking mode must hide page controls and must never pass board or log content through the information-view paginator.
- Extend asset-level checks and browser verification with the densest supported hacking fixture, asserting primary-text parity, full board/log/input visibility, zero scrollable regions, and absent pagination controls across the viewport matrix and at 200% zoom.

## BUG-002 Complexity Notes

- Larger hacking text increases pressure on both the fixed row count and the activity log at short viewports; responsive geometry must reclaim space without clipping addresses, selectable cells, attempts, log entries, or the input preview.
- Compact orientation and fallback-font metrics can change usable row and column capacity, so verification must cover every required viewport and the effective 512×300 layout produced by 200% zoom.

## BUG-003 Revalidation Notes

- The BUG-002 design remains authoritative. Reproduce the reported production-rendered failure before changing layout rules, recording the viewport, zoom, effective dimensions, active font metrics, region bounds, and scroll/client extents that distinguish the failing state from the previously passing fixture.
- Treat the supplied Fallout 4 screenshot as a spatial-composition guide, not an exact pixel target: keep the header and attempts above two dense board columns, allocate a narrow activity log alongside them when space permits, and retain the input preview within the same frame without reducing text below the FR-015 floor.
- Close the verification escape with production-asset browser checks that measure actual rendered bounds and overflow for the exact reproduced case, the full viewport matrix, 200% zoom, fallback-font rendering, and the densest supported hacking fixture.

## BUG-004 Implementation Notes

- Preserve completed T001, T003, and T010 as the responsive-layout baseline. Add a follow-up row-fitting pass that measures the rendered hacking columns and fixed 16-row allocation, then exposes one common fitted font size to every `.hack-row` without changing the base terminal font roles.
- Start from the base hacking role and select the largest size, within the FR-017 tolerance, that fits the address, row gap, and 12 board characters inside the narrower useful column while retaining the attempts display, both complete columns, activity log, and input preview on one screen.
- Recalculate after puzzle rendering, viewport resizing, compact/stacked orientation changes, and font readiness. Stabilize measurement before committing the fitted size so the resize observer cannot create a grow/shrink feedback loop.
- Extend the production-rendered geometry matrix to record useful column width, rendered row width, computed row font size, row height, and residual right-side space for bundled and fallback fonts at every supported viewport and at 200% zoom.

## BUG-004 Complexity Notes

- Width and height can impose different maxima: wide, short layouts may be limited by 16-row height before the glyphs reach the column edge, while narrower layouts may be limited by the address-plus-data width. The implementation must choose the lower bound and treat only that bounded capacity as useful space.
- Fallback-font glyph advances, fractional CSS pixels, word-span markup, orientation changes, and the existing compact/tight classes can alter the measured maximum. Tests must allow only the specified subpixel tolerance while retaining the BUG-003 descendant-containment checks.
