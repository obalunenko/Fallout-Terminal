# Feature Specification: Responsive Desktop Player Layout

**Created**: 2026-08-09
**Scope**: Player-facing browser terminal presentation on desktop-class viewports

## User Scenarios & Testing

### User Story 1 - See the complete terminal at once (Priority: P1)

As a player using a desktop browser, I can see the terminal frame, current content, and available navigation controls within one browser viewport so that I can understand and operate the terminal without scrolling the page.

**Why this priority**: This is the requested core value and prevents essential content or actions from being hidden below or beside the viewport.

**Independent Test**: Open each player presentation state at the supported minimum desktop viewport and confirm that the browser page does not scroll and every state-specific control is visible and operable.

**Acceptance Scenarios**:

1. **Given** a 1024×600 desktop viewport at 100% browser zoom, **when** the connection, idle, normal menu, record, command-output, hacking, or blocked state is displayed with representative content, **then** the complete interface frame and all available actions are visible without horizontal or vertical browser-page scrolling.
2. **Given** a normal terminal menu or record with representative content, **when** it is displayed, **then** the header, active content, back control when applicable, and prompt remain simultaneously visible.
3. **Given** the smallest supported desktop viewport, **when** the player changes between terminal states, **then** no control is clipped, covered, or moved outside the viewport.

---

### User Story 2 - Use the terminal across desktop screen sizes (Priority: P1)

As a player, I can use the terminal on compact laptop screens, standard monitors, and large desktop displays so that the experience remains readable and coherent without manual resizing or zoom adjustments.

**Why this priority**: A responsive design must preserve the primary journey across the expected desktop range, not only at one reference resolution.

**Independent Test**: Exercise the same representative terminal at 1024×600, 1280×720, 1440×900, 1920×1080, and 2560×1440 and compare content visibility, readability, and control access.

**Acceptance Scenarios**:

1. **Given** any supported desktop viewport, **when** normal terminal content is displayed, **then** text, spacing, and the terminal frame scale to use the available space without overlap, unintended truncation, or excessive empty space.
2. **Given** a hacking puzzle at any supported desktop viewport, **when** the board is displayed, **then** the board data, attempts, activity log, input preview, and available interactions remain visible and usable without browser-page scrolling.
3. **Given** an ultra-wide or high-resolution desktop viewport, **when** any state is displayed, **then** content remains bounded to a readable area and the Fallout terminal presentation does not become visually fragmented.
4. **Given** the bundled terminal font cannot be loaded, **when** the fallback font is used, **then** the layout still keeps all interface regions and controls within the viewport.

---

### User Story 3 - Reach exceptional and enlarged content (Priority: P2)

As a player with long terminal content or an enlarged browser view, I can still reach every piece of information and every action so that responsive fitting never causes content loss.

**Why this priority**: Ordinary desktop content should require no scrolling, while unavoidable overflow and accessibility enlargement must remain safe and usable.

**Independent Test**: Display unusually long introductions, labels, records, command output, and hacking logs, then repeat the journey at 200% browser zoom using only the keyboard.

**Acceptance Scenarios**:

1. **Given** authored content is too long to fit within the available content region, **when** it is displayed, **then** overflow is confined to the relevant content region and the terminal header and navigation controls remain visible.
2. **Given** a long label or unbroken text value, **when** it is displayed, **then** it does not widen the browser page or cover another control.
3. **Given** the browser is enlarged to 200% zoom, **when** the player navigates every presentation state, **then** no information or action is lost and all controls remain reachable, with localized scrolling permitted when fitting is no longer possible.
4. **Given** the player uses only the keyboard, **when** focus moves through available controls, **then** the focused control remains visible and is not hidden behind another region.

## Edge Cases

- The viewport is desktop-width but unusually short, such as 1024×600.
- The viewport is ultra-wide or high-resolution and would otherwise stretch line lengths excessively.
- The introduction, entry body, command output, or hacking log contains many lines.
- A menu label or authored text contains a single long, unbroken value.
- The most information-dense hacking board is displayed at the minimum supported viewport.
- The browser uses a fallback font with different character metrics.
- The browser zoom changes while a terminal state is already visible.
- Content changes after initial render and is larger than the content it replaces.

## Requirements

### Functional Requirements

- **FR-001**: The player interface MUST provide a responsive layout for viewport sizes from 1024×600 through 2560×1440 at 100% browser zoom.
- **FR-002**: Every player presentation state MUST fit within the browser viewport without horizontal or vertical browser-page scrolling when populated with representative content.
- **FR-003**: The terminal header, active content region, and available navigation controls MUST remain simultaneously visible at every supported desktop viewport.
- **FR-004**: Responsive changes MUST prevent interface text and controls from overlapping, being clipped, or extending outside the viewport.
- **FR-005**: The hacking state MUST keep the board data, attempts, activity log, input preview, and available interactions visible and usable at every supported desktop viewport.
- **FR-006**: Text size and spacing MUST adapt across the supported desktop range while keeping primary terminal text at least 16 pixels at 100% zoom.
- **FR-007**: The terminal frame and decorative presentation MUST remain wholly contained within the viewport without reducing the usable content region below what the active state requires.
- **FR-008**: Content that exceeds the available content region MUST use localized overflow without causing browser-page scrolling or hiding the terminal header or available navigation controls.
- **FR-009**: Long or unbroken authored values MUST remain fully accessible without widening the browser page or covering another control.
- **FR-010**: At 200% browser zoom, all information and controls MUST remain reachable and usable without overlap or content loss, although localized scrolling MAY be used.
- **FR-011**: Keyboard focus MUST remain visible whenever a player moves among interactive terminal controls.
- **FR-012**: Responsive presentation changes MUST preserve existing connection, navigation, hacking, reveal, and audio behavior.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All seven player presentation states pass at 1024×600, 1280×720, 1440×900, 1920×1080, and 2560×1440 with zero horizontal or vertical browser-page scrollbars at 100% zoom using representative content.
- **SC-002**: Across the desktop viewport matrix, 100% of visible labels and interactive controls remain inside the viewport without overlap or unintended clipping.
- **SC-003**: In long-content tests, 100% of authored content remains reachable while the terminal header and available navigation controls remain visible.
- **SC-004**: At 200% browser zoom, every presentation state retains access to 100% of its information and actions with no overlapping controls.
- **SC-005**: A keyboard-only pass can reach and operate every available player control at the minimum supported viewport while keeping focus visible.
- **SC-006**: Existing player connection, navigation, hacking, reveal, and audio acceptance journeys continue to pass after the responsive layout change.

## Assumptions

- “Desktop browser” means a viewport of at least 1024×600 at 100% browser zoom.
- “Without scrolling” applies to the browser page and to representative terminal content that fits the product's ordinary usage; unbounded user-authored content may use a localized content-region scrollbar so it is not lost.
- The player-facing browser terminal is in scope; the game-master Wails desktop window and mobile or tablet layouts are outside this feature.
- The existing Fallout CRT visual identity and player behavior remain unchanged apart from layout and content-fit improvements.
- Representative content includes every presentation state, a populated normal menu, a multiline record, command output, and the most information-dense supported hacking board.

## Approach

- Refine the player layout and responsive sizing in `client/client.css`, preserving the existing state model and CRT presentation.
- Keep the current viewport declaration in `client/index.html` unless verification exposes a desktop fitting issue that requires a markup adjustment.
- Extend the existing embedded-asset presentation checks in `internal/platform/assets_test.go` and verify the full desktop viewport matrix in a browser.
- No server, protocol, persistence, or game-master authoring changes are required.
