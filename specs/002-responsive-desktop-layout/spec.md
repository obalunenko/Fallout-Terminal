# Feature Specification: Responsive Desktop Player Layout

**Created**: 2026-08-09
**Scope**: Player-facing browser terminal presentation on desktop-class viewports
**Bugfix**: 2026-08-09 — BUG-001 Replaced localized scrolling with CRT-style pagination and clarified proportional font scaling
**Bugfix**: 2026-08-10 — BUG-002 Added readable single-screen hacking fit without scrolling or pagination
**Bugfix**: 2026-08-10 — BUG-003 Clarified production-rendered geometry verification for the password-game fit
**Bugfix**: 2026-08-10 — BUG-004 Added column-aware hacking-row font fitting

## User Scenarios & Testing

### User Story 1 - See the complete terminal at once (Priority: P1)

As a player using a desktop browser, I can see the terminal frame, current content, and available navigation controls within one browser viewport so that I can understand and operate the terminal without scrolling ~~the page~~ any browser or terminal region. The narrower page-only wording is superseded by BUG-001 because localized terminal scrolling is also prohibited.

**Why this priority**: This is the requested core value and prevents essential content or actions from being hidden below or beside the viewport.

**Independent Test**: Open each player presentation state at the supported minimum desktop viewport and confirm that neither the browser page nor any terminal region scrolls and every state-specific control is visible and operable.

**Acceptance Scenarios**:

1. **Given** a 1024×600 desktop viewport at 100% browser zoom, **when** the connection, idle, normal menu, record, command-output, hacking, or blocked state is displayed with representative content, **then** the complete interface frame and all available actions are visible without horizontal or vertical scrolling in the browser page or any terminal region.
2. **Given** a normal terminal menu or record with representative content, **when** it is displayed, **then** the header, active content, back control when applicable, and prompt remain simultaneously visible.
3. **Given** the smallest supported desktop viewport, **when** the player changes between terminal states, **then** no control is clipped, covered, or moved outside the viewport.

---

### User Story 2 - Use the terminal across desktop screen sizes (Priority: P1)

As a player, I can use the terminal on compact laptop screens, standard monitors, and large desktop displays so that the experience remains readable and coherent without manual resizing or zoom adjustments.

**Why this priority**: A responsive design must preserve the primary journey across the expected desktop range, not only at one reference resolution.

**Independent Test**: Exercise the same representative terminal at 1024×600, 1280×720, 1440×900, 1920×1080, and 2560×1440 and compare content visibility, readability, and control access.

**Acceptance Scenarios**:

1. **Given** any supported desktop viewport, **when** normal terminal content is displayed, **then** every font role derives from one responsive scale factor while preserving its defined hierarchy, and text, spacing, and the terminal frame use the available space without overlap, unintended truncation, or excessive empty space.
2. **Given** a hacking puzzle at any supported desktop viewport, **when** the board is displayed, **then** the board data, attempts, activity log, input preview, and available interactions remain visible and usable without browser-page scrolling.
3. **Given** an ultra-wide or high-resolution desktop viewport, **when** any state is displayed, **then** content remains bounded to a readable area and the Fallout terminal presentation does not become visually fragmented.
4. **Given** the bundled terminal font cannot be loaded, **when** the fallback font is used, **then** the layout still keeps all interface regions and controls within the viewport.
5. **Given** a hacking puzzle at any supported desktop viewport, **when** the password game is displayed, **then** its text is no smaller than primary terminal text at the same viewport and its board geometry adapts to keep the complete interaction on one screen without scrolling or pagination.
6. **Given** a hacking puzzle whose memory-dump columns have spare horizontal capacity, **when** the board is rendered or its viewport, orientation, or active font changes, **then** every hacking row uses one common fitted font size that expands its address and 12-character data to the maximum useful column width permitted by the complete single-screen layout.

---

### User Story 3 - Reach exceptional and enlarged content (Priority: P2)

As a player with long terminal content or an enlarged browser view, I can still reach every piece of information and every action so that responsive fitting never causes content loss.

**Why this priority**: ~~Ordinary desktop content should require no scrolling, while unavoidable overflow and accessibility enlargement must remain safe and usable.~~ Superseded by BUG-001 because no player-terminal state may rely on scrolling; overflow and accessibility enlargement must remain safe and usable through fitting or terminal-native pagination, with the hacking game restricted to fitting on one screen by BUG-002.

**Independent Test**: Display unusually long introductions, labels, records, command output, ~~and hacking logs~~ confirm excess information text is available through terminal-native pages without any scrollable region, confirm the complete hacking game instead remains fitted to one non-paginated screen, then repeat the journey at 200% browser zoom using only the keyboard. The hacking exception is clarified by BUG-002.

**Acceptance Scenarios**:

1. **Given** authored content is too long to fit within the available content region, **when** it is displayed, **then** ~~overflow is confined to the relevant content region~~ it is fitted or divided into terminal-native pages without scrolling, and the terminal header and navigation controls remain visible. The localized-overflow behavior is superseded by BUG-001.
2. **Given** a long label or unbroken text value, **when** it is displayed, **then** it does not widen the browser page or cover another control.
3. **Given** the browser is enlarged to 200% zoom, **when** the player navigates every presentation state, **then** no information or action is lost and all controls remain reachable ~~with localized scrolling permitted when fitting is no longer possible~~ without browser-page or localized scrolling, ~~using terminal-native pagination where fitting is no longer possible~~ using terminal-native pagination for information views where fitting is no longer possible while keeping the hacking game fitted to one screen. The scrolling allowance is superseded by BUG-001 and the hacking exception is clarified by BUG-002.
4. **Given** the player uses only the keyboard, **when** focus moves through available controls, **then** the focused control remains visible and is not hidden behind another region.
5. **Given** selecting a menu item opens information text that spans multiple pages, **when** the view opens or the player changes pages, **then** it starts on page one, shows the current and total page count, exposes previous and next controls only when those directions are available, preserves the header and back control, and supports keyboard operation without scrolling.
6. **Given** a paginated information view is open, **when** its content or available viewport size changes, **then** page boundaries and the page count are recalculated, the selected page is clamped to the valid range, and no content is lost.

## Edge Cases

- The viewport is desktop-width but unusually short, such as 1024×600.
- The viewport is ultra-wide or high-resolution and would otherwise stretch line lengths excessively.
- The introduction, entry body, command output, or hacking log contains many lines.
- A menu label or authored text contains a single long, unbroken value.
- The most information-dense hacking board is displayed at the minimum supported viewport.
- The hacking columns have substantial spare horizontal capacity while their fixed 16-row height remains the limiting vertical constraint.
- The browser uses a fallback font with different character metrics.
- The browser zoom changes while a terminal state is already visible.
- Content changes after initial render and is larger than the content it replaces.

## Requirements

### Functional Requirements

- **FR-001**: The player interface MUST provide a responsive layout for viewport sizes from 1024×600 through 2560×1440 at 100% browser zoom.
- **FR-002**: ~~Every player presentation state MUST fit within the browser viewport without horizontal or vertical browser-page scrolling when populated with representative content.~~ Superseded by BUG-001: Every player presentation state MUST fit within the browser viewport without horizontal or vertical scrolling in either the browser page or any player-terminal region.
- **FR-003**: The terminal header, active content region, and available navigation controls MUST remain simultaneously visible at every supported desktop viewport.
- **FR-004**: Responsive changes MUST prevent interface text and controls from overlapping, being clipped, or extending outside the viewport.
- **FR-005**: The hacking state MUST keep the board data, attempts, activity log, input preview, and available interactions visible and usable at every supported desktop viewport.
- **FR-006**: ~~Text size and spacing MUST adapt across the supported desktop range while keeping primary terminal text at least 16 pixels at 100% zoom.~~ Clarified by BUG-001: Every terminal font role MUST derive from one viewport-responsive scale factor, preserve a fixed role hierarchy across the supported desktop range, and keep primary terminal text at least 16 pixels at 100% zoom. Clarified by BUG-004: the shared scale defines the base font roles; the rendered hacking rows MAY apply the bounded fit multiplier required by FR-017, provided they never shrink below the base hacking role.
- **FR-007**: The terminal frame and decorative presentation MUST remain wholly contained within the viewport without reducing the usable content region below what the active state requires.
- **FR-008**: ~~Content that exceeds the available content region MUST use localized overflow without causing browser-page scrolling or hiding the terminal header or available navigation controls.~~ ~~Superseded by BUG-001: Content that exceeds its available region MUST be fitted or divided into terminal-native pages; browser-page and localized scrolling MUST NOT be available, and the terminal header and available navigation controls MUST remain visible.~~ Clarified by BUG-002: Content that exceeds its available region MUST be fitted or, for information views only, divided into terminal-native pages; the hacking game MUST use fitting only, browser-page and localized scrolling MUST NOT be available, and the terminal header and available navigation controls MUST remain visible.
- **FR-009**: ~~Long or unbroken authored values MUST remain fully accessible without widening the browser page or covering another control.~~ Clarified by BUG-001: Long or unbroken authored values MUST remain fully accessible through wrapping, fitting, or terminal-native pagination without widening the browser page, covering another control, or creating a scrollable region.
- **FR-010**: ~~At 200% browser zoom, all information and controls MUST remain reachable and usable without overlap or content loss, although localized scrolling MAY be used.~~ ~~Superseded by BUG-001: At 200% browser zoom, all information and controls MUST remain reachable and usable without overlap, content loss, browser-page scrolling, or localized scrolling; terminal-native pagination MUST be used where fitting is no longer possible.~~ Clarified by BUG-002: At 200% browser zoom, all information and controls MUST remain reachable and usable without overlap, content loss, browser-page scrolling, or localized scrolling; terminal-native pagination MUST be used for information views where fitting is no longer possible, while the hacking game MUST remain fitted to one screen.
- **FR-011**: Keyboard focus MUST remain visible whenever a player moves among interactive terminal controls.
- **FR-012**: Responsive presentation changes MUST preserve existing connection, navigation, hacking, reveal, and audio behavior.
- **FR-013**: When a selected menu item opens information text that does not fit on one screen, the player MUST present discrete CRT-style pages with a current/total page indicator and keyboard-operable previous and next controls, while keeping the terminal header and back control visible.
- **FR-014**: A paginated information view MUST open on page one, prevent navigation beyond its first and last pages, recalculate page boundaries after content or viewport changes, and keep the current page within the resulting valid range.
- **FR-015**: At every supported desktop viewport, hacking-game text MUST be no smaller than primary terminal text at the same viewport, and the hacking board MUST use the available content region without appearing disproportionately reduced relative to other terminal states.
- **FR-016**: The hacking game MUST remain a single-screen interaction with its board data, attempts, activity log, input preview, and interactions simultaneously visible; it MUST NOT expose browser-page scrolling, localized scrolling, pagination controls, or paginated board content, and MUST adapt board geometry, gaps, padding, log allocation, or orientation rather than reducing text below the FR-015 floor. Clarified by BUG-003: “simultaneously visible” MUST be verified from the production-rendered player layout, with every named region wholly inside the viewport and no positive document or player-region overflow beyond subpixel rounding.
- **FR-017**: All rendered hacking rows MUST share one column-aware font size that is no smaller than the base hacking role and is the largest size, within 0.5 CSS pixels, at which each address, address-to-data gap, and 12-character board row fits the narrower useful hacking-column width while all 16 rows and every other required hacking region remain within the single-screen layout. The fit MUST be recalculated after puzzle rendering and after viewport, layout-orientation, or active-font changes.

## Success Criteria

### Measurable Outcomes

- **SC-001**: ~~All seven player presentation states pass at 1024×600, 1280×720, 1440×900, 1920×1080, and 2560×1440 with zero horizontal or vertical browser-page scrollbars at 100% zoom using representative content.~~ Superseded by BUG-001: All seven player presentation states pass the same viewport matrix with zero horizontal or vertical scrollbars on the browser page and in every player-terminal region at 100% zoom using representative content.
- **SC-002**: Across the desktop viewport matrix, 100% of visible labels and interactive controls remain inside the viewport without overlap or unintended clipping.
- **SC-003**: ~~In long-content tests, 100% of authored content remains reachable while the terminal header and available navigation controls remain visible.~~ ~~Clarified by BUG-001: In long-content tests, 100% of authored content remains reachable through terminal-native pages, with zero scrollable regions and with the terminal header and available navigation controls visible.~~ Clarified by BUG-002: In long-information-content tests, 100% of authored content remains reachable through terminal-native pages, with zero scrollable regions and with the terminal header and available navigation controls visible; the hacking game instead exposes 100% of its supported content on one fitted screen.
- **SC-004**: ~~At 200% browser zoom, every presentation state retains access to 100% of its information and actions with no overlapping controls.~~ Clarified by BUG-001: At 200% browser zoom, every presentation state retains access to 100% of its information and actions with no overlap, content loss, browser-page scrolling, or localized scrolling.
- **SC-005**: A keyboard-only pass can reach and operate every available player control at the minimum supported viewport while keeping focus visible.
- **SC-006**: Existing player connection, navigation, hacking, reveal, and audio acceptance journeys continue to pass after the responsive layout change.
- **SC-007**: Across the desktop viewport matrix, computed sizes for all terminal font roles use the same responsive scale factor and preserve their defined inter-role ratios within one percent. Clarified by BUG-004: this ratio check applies to the base role sizes; a rendered hacking row may grow above its base role only through the bounded FR-017 fit.
- **SC-008**: One-page, exact-boundary, and multi-page information fixtures expose 100% of their content; report the correct current/total page count; enforce first/last-page boundaries; and remain operable by keyboard with zero page or localized scrolling.
- **SC-009**: Across the desktop viewport matrix and at 200% zoom, the most information-dense supported hacking fixture displays 100% of its board data, attempts, activity log, input preview, and interactions on one screen with hacking text at least as large as primary terminal text, zero scrollable regions, and no page indicator or pagination controls. Clarified by BUG-003: the pass MUST use production player assets and inspect rendered region bounds and scroll/client extents; DOM presence or stylesheet-source assertions alone do not establish a fit.
- **SC-010**: Across the supported viewport matrix, the 200%-zoom case, and bundled- and fallback-font rendering, every hacking row uses the same fitted font size, remains wholly contained, and is within 0.5 CSS pixels of the largest font size allowed simultaneously by the narrower useful column width and the complete 16-row single-screen geometry.

## Assumptions

- “Desktop browser” means a viewport of at least 1024×600 at 100% browser zoom.
- ~~“Without scrolling” applies to the browser page and to representative terminal content that fits the product's ordinary usage; unbounded user-authored content may use a localized content-region scrollbar so it is not lost.~~ Superseded by BUG-001: “Without scrolling” applies to the browser page and every player-terminal region; content that does not fit must be wrapped, fitted, or paginated so it is not lost. Clarified by BUG-002: pagination applies to information views only; the hacking game must remain fitted to one screen.
- The player-facing browser terminal is in scope; the game-master Wails desktop window and mobile or tablet layouts are outside this feature.
- The existing Fallout CRT visual identity and player behavior remain unchanged apart from layout and content-fit improvements.
- Representative content includes every presentation state, a populated normal menu, a multiline record, command output, and the most information-dense supported hacking board.

## Approach

- Refine the player layout and responsive sizing in `client/client.css`, preserving the existing state model and CRT presentation.
- Define one responsive typography scale in `client/client.css`, derive every font role from it, and remove scrollbar-based player-region overflow behavior.
- Keep the hacking role at least as large as primary terminal text and fit the complete password game by adapting board columns, gaps, padding, log allocation, and compact orientation; never route hacking content through the information-view paginator.
- Fit all hacking rows with one rendered column-aware font multiplier above the shared base hacking role, choosing the largest size allowed by both the narrower useful column width and the complete 16-row height, and recalculate it after content, viewport, orientation, or font changes.
- Add information-view page calculation and navigation state in `client/client.js`; add or adjust page indicators and controls in `client/index.html` and `client/client.css` while preserving existing back and menu navigation.
- Keep the current viewport declaration in `client/index.html` unless verification exposes a desktop fitting issue that requires a markup adjustment.
- Extend the existing embedded-asset presentation checks in `internal/platform/assets_test.go` with shared-type-scale, zero-scroll-region, and pagination contracts, and verify the full desktop viewport matrix in a browser with one-page, exact-boundary, and multi-page fixtures.
- No server, protocol, persistence, or game-master authoring changes are required.

## Verification (2026-08-09)

- The production frontend build and packaged Wails production build completed successfully.
- Browser checks covered 1024×600, 1280×720, 1440×900, 1920×1080, and 2560×1440. At every size, the multi-page information view, header, back control, page controls, and prompt remained inside the viewport with zero document or localized scrollable regions; primary text was 16 pixels at 1024×600 and every sampled font-role ratio remained fixed.
- Connection, idle, menu, one-page entry, command-output, hacking, and blocked states all passed at the minimum supported viewport. The same content states passed at the 512×300 effective layout produced by a 1024×600 desktop viewport at 200% zoom, including the compact hacking board and log.
- One-page and exact-boundary fixtures reported `1 / 1` with both direction controls unavailable. A five-page fixture exposed the exact 4,529-character source across pages `1 / 5` through `5 / 5`, hid unavailable first/last controls, and preserved all content through mouse and keyboard navigation.
- Resizing a five-page view to a three-page capacity recalculated page boundaries and clamped the selected page to `3 / 3`. A missing bundled font fell back to the configured monospace stack with zero scrolling and all persistent controls visible.
- `node --check client/client.js`, `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...` completed successfully.

## Verification (2026-08-10)

- The BUG-002 production frontend build and packaged Wails production build completed successfully.
- Browser checks repeated the 1024×600, 1280×720, 1440×900, 1920×1080, and 2560×1440 matrix plus the 512×300 effective layout used to model 200% zoom. Normal menu, information-entry, command-output, hacking, and blocked regions remained inside the viewport with zero document or localized scrollable regions.
- The densest hacking fixture displayed all 32 board rows and nine activity-log lines at every size. Hacking text exactly matched primary terminal text from 16 pixels at 1024×600 through 24 pixels at the largest viewports, the 512×300 layout selected compact stacked geometry, and pagination controls stayed absent.
- A 7,560-character information fixture remained byte-for-byte complete across five pages at 512×300. Mouse and keyboard navigation enforced the first and last page boundaries, and resizing from the fifth 512×300 page to 1024×600 recalculated and clamped the view to page `4 / 4`; one-page and command-output fixtures retained their expected pagination behavior.
- `node --check client/client.js`, `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...` completed successfully.

## BUG-003 Reproduction (2026-08-10)

- The failure reproduced from the production `client/` assets at 200% browser zoom on the minimum 1024×600 desktop viewport, yielding a 512×300 effective CSS viewport with `devicePixelRatio: 1` in the verification browser.
- The active font stack was `Fixedsys, Consolas, monospace`; primary and hacking text both computed to 8 CSS pixels (16 physical pixels at 200% zoom) with an 8.16-pixel compact row line height, preserving the FR-015 text floor.
- The fixture used both 16-row memory-dump columns and the supported 13-line activity history produced by administrator activation plus four rejected guesses. The header occupied `y=10.50..36.64`, the hacking board `y=40.64..287.50`, the input `y=278.70..287.50`, and the page controls remained absent.
- In the resulting `hack-stacked hack-compact` layout, the columns ended at `y=168.26` while each final board row ended at `y=171.14`, clipping 2.88 pixels from both final rows. The activity log and input remained inside the board.
- The document remained `512×300` with matching client and scroll extents, and the board and columns likewise reported matching scroll/client extents because `overflow: hidden` masked the displaced child rows. This establishes the BUG-003 verification escape: scroll-extent equality alone did not prove that every required descendant was within its allocated region.

## BUG-003 Verification (2026-08-10)

- Production `client/` assets were exercised with the 13-line dense hacking history at 1024×600, 1280×720, 1440×900, 1920×1080, and 2560×1440, plus the 512×300 effective viewport for 200% zoom. The complete matrix was repeated with the bundled Fixedsys face replaced by `Courier New, monospace` to vary glyph metrics.
- Every bundled-font and fallback-font case rendered all 32 board rows and all 13 activity lines. The header, board columns, log panel, log, and input were wholly inside their required parent regions; document, body, screen, terminal body, hacking board, columns, and log scroll extents stayed within the one-pixel subpixel tolerance of their client extents.
- Hacking and primary terminal text remained equal at every size: 16 pixels at 1024×600, approximately 19.2 pixels at 1280×720, 22.5 pixels at 1440×900, 24 pixels at 1920×1080 and 2560×1440, and 8 CSS pixels (16 physical pixels) in the 200%-zoom model.
- The exact former failure selected `hack-stacked hack-compact hack-tight`; both final rows ended at the columns boundary, all 13 log lines and the input remained visible, document overflow was zero, and information-view pagination controls remained absent. Larger viewports retained the normal side-by-side board and log composition.
- `go test ./internal/platform` passed the asset-level contracts for the shared type scale, zero-scroll policy, required hacking regions, descendant-bound geometry detection, tight compact fallback, and exclusion of information pagination from hacking mode.

## BUG-004 Reproduction (2026-08-10)

- Production `client/` assets were measured with the dense 32-row, 13-log-line hacking fixture before adding a row-specific fit. Both columns had the same geometry in every case and every row inherited the base hacking role without a fitted override.
- With bundled Fixedsys, the 1024×600 case used a 16-pixel row font and 17.59-pixel row height; its 16-row block occupied 281.50 pixels, while the widest address-plus-12-character row used only 164.19 of 319.73 useful column pixels and left 155.54 pixels unused. At 1280×720 the values were 19.20-pixel font, 21.12-pixel row height, 337.88-pixel block, 195.05/400.71 pixels used, and 205.66 pixels residual. At 1440×900 they were 22.50, 24.75, 396.00, 226.02/441.46, and 215.45 pixels. At both bounded large-screen layouts the base role capped at 24 pixels, each row was 26.40 pixels high, the block was 422.38 pixels high, and only 240.00 of 470.24 pixels was used, leaving 230.24 pixels residual.
- With `Courier New, monospace`, the same cases retained the same base and computed row sizes and 16-row heights but used 193.02/319.73 pixels at 1024×600, 229.65/400.71 at 1280×720, 266.63/441.46 at 1440×900, and 283.29/470.24 at the bounded large-screen layouts. Residual width remained substantial at 126.71, 171.06, 174.84, and 186.95 pixels respectively.
- The 512×300 effective 200%-zoom layout selected `hack-stacked hack-compact hack-tight`. Its base and computed row font were 8 CSS pixels, its row height was 8.16 pixels, and its 16-row block occupied 130.50 pixels. Fixedsys used 74.00 of 241.90 useful pixels and left 167.90 pixels residual; Courier New used 88.46 pixels and left 153.44 pixels residual.
- Every baseline case retained all 16 rows per column and zero document overflow, proving the issue was unused horizontal capacity rather than a containment failure. The measured gap also establishes enough room for a bounded row-font increase, subject to the unchanged complete-screen height and descendant-containment limits.

## Final Verification (2026-08-10)

- The frontend Vite production build and packaged Wails production build completed successfully with the current embedded player assets.
- Forty-two rendered state/viewport cases passed: connection, idle, menu, information entry, command output, active hacking, and blocked hacking at 1024×600, 1280×720, 1440×900, 1920×1080, 2560×1440, and the 512×300 effective 200%-zoom viewport. Every required region remained visible and inside the viewport with zero document or localized overflow; pagination appeared only for the information entry and command-output states.
- At 512×300, a 200-character one-page fixture and the measured 1,815-character exact-boundary fixture both reported `1 / 1`, preserved their complete source, hid both unavailable directions, and had zero content overflow. A 7,560-character fixture remained byte-for-byte complete across pages `1 / 5` through `5 / 5`; keyboard navigation held at both boundaries, and every page stayed within its content region.
- Resizing the five-page fixture from 512×300 while on its final page to 1024×600 recalculated the capacity and clamped the selection to `4 / 4`, with the next direction unavailable and no content overflow.
- The dense BUG-003 hacking fixture and alternate-font matrix remained green as recorded above. `node --check client/client.js`, `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...` all completed successfully.

## BUG-004 Verification (2026-08-10)

- The production player passed 42/42 rendered state cases: connection, idle, menu, information entry, command output, active hacking, and blocked hacking at 1024×600, 1280×720, 1440×900, 1920×1080, 2560×1440, and the 512×300 effective 200%-zoom viewport. Every required region and control remained inside the viewport with zero document or localized overflow, and hacking continued to expose no pagination controls.
- The dense hacking fixture passed 12/12 bundled- and fallback-font geometry cases with 32 visible rows, 13 visible activity lines, one common row size, complete descendant containment, zero overflow, and a row size no smaller than the base hacking role. Fixedsys fitted from base sizes `16, 19.20, 22.50, 24, 24, 8` to `28.75, 34.66, 42.55, 44.48, 44.48, 8` CSS pixels across the matrix. Courier New fitted to `27.72, 34.66, 38.66, 41.21, 41.21, 8` CSS pixels.
- The limiting width-or-height headroom inferred from the rendered column and 16-row bounds was `0.000..0.181` font pixels in every bundled and fallback case, within the 0.5 CSS-pixel maximal-fit requirement. Width limited the Courier New cases at 1024×600, 1440×900, 1920×1080, and 2560×1440; the complete 16-row height limited the remaining cases, including the 512×300 compact stacked layout where the base 8-pixel role was already maximal.
- Pagination revalidation at 512×300 kept the 200- and 1,815-character fixtures on `1 / 1`, preserved the 7,560-character fixture byte-for-byte across five pages, enforced both keyboard boundaries, and recalculated and clamped the final page to `4 / 4` after resizing to 1024×600.
- `npm run build` completed the Vite production bundle and `wails build` produced the packaged self-signed darwin/arm64 application. `node --check client/client.js`, `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...` all completed successfully.
