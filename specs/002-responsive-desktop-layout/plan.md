# Implementation Plan: Responsive Desktop Player Layout

> The implementation approach is defined in [spec.md § Approach](./spec.md#approach); execution is tracked in [tasks.md](./tasks.md).

**Bugfix**: 2026-08-09 — BUG-001 Updated from bugfix patch

## BUG-001 Implementation Notes

- Replace the independent font-role clamps in `client/client.css` with one responsive scale factor and fixed role multipliers so all text scales proportionally while retaining the existing hierarchy and 16-pixel primary-text minimum.
- Remove browser and player-region scrollbar behavior. Fit bounded terminal states within the viewport and route information text that exceeds its content region through discrete CRT-style pages.
- Implement page calculation and state in `client/client.js`; expose a current/total indicator plus boundary-aware previous and next controls through `client/index.html` and `client/client.css`; preserve the existing header, back navigation, focus visibility, and terminal behavior.
- Recalculate pagination when content or viewport dimensions change. Open information views on page one, clamp the current page after recalculation, and support the same actions by keyboard.
- Extend `internal/platform/assets_test.go` and browser verification to cover common font scaling, absence of page and localized scrollbars, one-page/exact-boundary/multi-page text, page boundaries, and 200% zoom.

## BUG-001 Complexity Notes

- Page capacity depends on rendered font metrics, the shared responsive scale, viewport height, persistent header/footer space, and fallback-font variance; pagination must measure the actual available content region without introducing layout feedback loops.
- Responsive repagination must keep content reachable when a resize or zoom change reduces the page count, and must not disrupt menu, back, hacking, reveal, or audio behavior.
