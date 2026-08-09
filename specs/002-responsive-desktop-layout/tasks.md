# Tasks: Responsive Desktop Player Layout

- [x] **T001** ⚠️ Reopened — Define responsive shell dimensions and spacing plus one shared responsive scale factor with fixed font-role multipliers for the supported viewport range (reopened — BUG-001) + `client/client.css`
- [x] **T002** ⚠️ Reopened — Adapt normal terminal header, menu, entry, output, prompt, pagination, and navigation regions to remain simultaneously visible without browser or localized scrolling (reopened — BUG-001) + `client/client.css`
- [x] **T003** Adapt the hacking board, columns, log, attempts, and input preview for compact desktop heights and widths + `client/client.css`
- [x] **T004** ⚠️ Reopened — Replace exceptional-content scrolling with wrapping, fitting, or terminal-native pagination while covering long values, fallback-font variance, 200% zoom, and visible keyboard focus (reopened — BUG-001) + `client/client.css`, `client/client.js`
- [x] **T005** [P] ⚠️ Reopened — Adjust the player document structure to expose the CRT page indicator and boundary-aware previous/next controls without displacing the header or back control (reopened — BUG-001) + `client/index.html`
- [x] **T006** ⚠️ Reopened — Add asset-level regression assertions for the shared font scale, zero browser/local scrollbars, pagination contracts, and persistent navigation visibility (reopened — BUG-001) + `internal/platform/assets_test.go`
- [x] **T007** ⚠️ Reopened — Run the production build and verify all player states at the specified desktop viewport and zoom matrix using one-page, exact-boundary, and multi-page fixtures (reopened — BUG-001) + `specs/002-responsive-desktop-layout/spec.md`
- [x] **T008** [US3] Implement information-view page calculation, page state, resize/content repagination, first/last-page boundaries, and keyboard navigation; depends on T001, T002, and T005 + `client/client.js`
- [x] **T009** [US3] Add and style the current/total page indicator and previous/next controls, preserving the terminal header, back control, focus visibility, and CRT presentation; depends on T005 and T008 + `client/index.html`, `client/client.css`

**Bugfix**: 2026-08-09 — BUG-001 Updated from bugfix patch
