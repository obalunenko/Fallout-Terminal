---
status: migrated
feature: Player Terminal Presentation
source: existing implementation
---

# Feature Specification: Player Terminal Presentation

**Migration status**: Reverse-engineered from the existing implementation on 2026-08-09  
**Scope**: Browser-rendered Fallout terminal shell, presentation states, local input affordances, reveal effects, and audio feedback

**Current runtime**: The static player behavior is unchanged. The same
HTML/CSS/JavaScript and assets are now embedded and served by the in-process Go
HTTP/WebSocket server rather than the removed Electron/Node runtime.

## Purpose

The player terminal turns server-owned live state into a themed browser experience for tabletop players. It presents connection and idle states, terminal folders, entries, command output, and the hacking gate with a green RobCo-style CRT treatment. Players can operate the shared terminal with pointer or keyboard input, while visual and audio feedback reinforces state changes without owning canonical navigation or hacking state.

## User Scenarios and Acceptance

### User Story 1 — Recognize connection and broadcast state (Priority: P1)

As a player, I can tell whether my browser is connecting, disconnected, waiting for content, or displaying a live terminal so that I understand whether the shared experience is available.

**Independent verification**: Open the player page before a broadcast, start and stop a broadcast, interrupt the server connection, and restore it.

**Acceptance scenarios**:

1. **Given** the page is establishing its WebSocket connection, **when** it first renders, **then** a full-screen connection overlay displays `УСТАНОВКА СВЯЗИ...`.
2. **Given** the WebSocket opens and no terminal is live, **when** the overlay clears, **then** the CRT shell shows the idle waiting message.
3. **Given** an established WebSocket closes, **when** the client detects the loss, **then** the connection overlay reports the loss and a reconnect attempt is scheduled after approximately three seconds.
4. **Given** a terminal is live, **when** `TERMINAL_CLEAR` arrives, **then** live content is hidden, ambient audio stops, and the idle state returns.

---

### User Story 2 — Read a Fallout-style live terminal (Priority: P1)

As a player, I can read the terminal identity, introduction, directories, records, and command output in a consistent Fallout-inspired display so that the game master's content feels like an in-world computer.

**Independent verification**: Broadcast a terminal containing nested folders, an entry, a command, multiline text, and an empty directory, then inspect every display state.

**Acceptance scenarios**:

1. **Given** a normal live terminal, **when** the player view renders, **then** it shows the RobCo header, a per-broadcast server number, optional introduction text, and terminal prompt.
2. **Given** the current folder contains child nodes, **when** list mode renders, **then** every child appears as a selectable row and the locally selected row is highlighted.
3. **Given** the current folder has no children, **when** list mode renders, **then** the player sees `[ ДИРЕКТОРИЯ ПУСТА ]`.
4. **Given** an entry is active, **when** its server-owned navigation state arrives, **then** the title and multiline description appear in the record view with a back control.
5. **Given** a command is active, **when** its server-owned navigation state arrives, **then** its multiline output appears in a distinct output pane.
6. **Given** terminal content contains markup characters, **when** it is rendered, **then** user-authored text is displayed as text rather than executed as HTML.

---

### User Story 3 — Operate the display with pointer or keyboard (Priority: P1)

As a player, I can select terminal rows and return from content views using familiar pointer and keyboard controls so that I can participate from a laptop or browser device.

**Independent verification**: Navigate the same terminal using clicks and then Arrow Up, Arrow Down, Enter, Escape, and Backspace.

**Acceptance scenarios**:

1. **Given** list mode has rows, **when** the pointer moves to another row, **then** the local highlight moves to that row and focus audio plays once for the transition.
2. **Given** list mode has rows, **when** Arrow Up or Arrow Down is pressed, **then** the local selection moves within the list bounds and does not wrap.
3. **Given** a row is selected, **when** it is clicked or Enter is pressed, **then** the client sends the corresponding navigation request without changing canonical navigation locally.
4. **Given** a player is inside a nested folder or record, **when** the back control, Escape, or Backspace is used, **then** the client requests the shared back action.
5. **Given** record mode is active, **when** Enter is pressed, **then** the client also requests the shared back action.

---

### User Story 4 — See hacking state in the same terminal language (Priority: P1)

As a player, I can see and interact with the hacking gate through a dense RobCo-style board so that the puzzle remains visually integrated with the terminal.

**Independent verification**: Broadcast a hacking-enabled terminal and inspect the active, solved, and failed presentation while hovering and selecting board targets.

**Acceptance scenarios**:

1. **Given** an unsolved hacking gate, **when** the live state renders, **then** normal terminal chrome is replaced by the hacking header, attempts, two board columns, log, and input preview.
2. **Given** the pointer is over a candidate or filler target, **when** hover changes, **then** all cells sharing that target are highlighted and their text appears in the input preview.
3. **Given** an active target is clicked, **when** the client accepts the interaction, **then** entry audio plays and a hacking request is sent to the server.
4. **Given** a failed puzzle state arrives, **when** it renders, **then** the board is replaced by the blocked-access message.
5. **Given** a newly solved puzzle state arrives, **when** it renders, **then** success audio plays and normal terminal presentation appears after approximately 2.6 seconds.

---

### User Story 5 — Receive atmospheric visual and audio feedback (Priority: P2)

As a player, I receive CRT styling, progressive text reveal, terminal effects, and contextual sounds so that interactions feel responsive and atmospheric.

**Independent verification**: Visit new folders, entries, and command output; repeat renders without changing content; trigger interaction and hacking feedback; then clear the broadcast.

**Acceptance scenarios**:

1. **Given** player content is visible, **when** the screen renders, **then** the bundled terminal font, green glow, scanlines, vignette, blinking cursor, and periodic flicker styling are applied.
2. **Given** a folder, entry, or command output is newly shown, **when** it renders, **then** its lines or rows reveal progressively with character-scroll audio.
3. **Given** the same content rerenders because unrelated state changed, **when** its identity is unchanged, **then** it appears immediately without replaying the reveal sequence.
4. **Given** supported sound files are available, **when** menu, board, entry, failure, or success events occur, **then** the mapped sound category plays at its configured volume.
5. **Given** the user has clicked the document and a live terminal exists, **when** ambient audio is ready, **then** it loops until the live terminal is cleared.
6. **Given** a sound folder, file, fetch, or decode operation fails, **when** playback is attempted, **then** the visual terminal remains usable without surfacing a blocking error.

## Functional Requirements

- **FR-001**: The player experience MUST run as static browser HTML, CSS, and JavaScript without desktop-runtime or Node.js APIs.
- **FR-002**: The page MUST expose distinct connection, idle, normal list, record, hacking, and blocked presentation states.
- **FR-003**: WebSocket connection loss MUST display a reconnecting overlay and MUST trigger repeated reconnect attempts using the existing three-second delay.
- **FR-004**: Normal terminal presentation MUST show the RobCo heading, a display-only server number from 1 through 9, optional introduction text, content body, and prompt.
- **FR-005**: Folder rows, empty directories, records, and command output MUST render from the current tree and server-provided navigation state.
- **FR-006**: User-authored terminal names and content MUST be assigned as text or escaped before any HTML insertion.
- **FR-007**: Pointer and keyboard selection MUST remain local presentation state; navigation and hacking transitions MUST occur only after corresponding server state is received.
- **FR-008**: List-mode keyboard controls MUST support Arrow Up, Arrow Down, Enter, Escape, and Backspace without moving the selection beyond list bounds.
- **FR-009**: Record-mode keyboard controls MUST support Enter, Escape, and Backspace as back requests, and visible nested views MUST provide a pointer-operated back control.
- **FR-010**: The hacking presentation MUST display the public board, addresses, selectable targets, remaining attempts, shared log, input preview, and failed state without requiring private puzzle data.
- **FR-011**: New folders, records, and command outputs MUST reveal their child rows or lines at the existing 40 millisecond interval; unchanged content MUST NOT replay the reveal solely because it rerenders.
- **FR-012**: The visual shell MUST use the bundled Fixedsys font with a monospace fallback and MUST apply the existing green CRT frame, glow, scanlines, vignette, cursor blink, and screen flicker effects.
- **FR-013**: Text sizing MUST use the existing viewport-responsive `clamp()` values, and the terminal body and bounded output areas MUST remain scrollable when content exceeds their available height.
- **FR-014**: The client MUST discover audio only through the allowlisted `/api/sounds/:folder` endpoint and MUST support the existing ambient, success, failure, focus, single-character, multi-character, entry, and reveal categories.
- **FR-015**: Sound discovery, prefetch, decoding, and playback failures MUST degrade silently without preventing visual rendering or player input.
- **FR-016**: Ambient audio MUST require the existing browser user-gesture gate, loop while active, and pause when the live terminal is cleared.

## Presentation Inputs and Outputs

| Direction | Contract | Presentation behavior |
|---|---|---|
| Server → player | `TERMINAL_LIVE` | Initializes terminal identity, content tree, introduction, navigation, optional hack state, and a new display-only server number. |
| Server → player | `TERMINAL_UPDATE` | Replaces tree/introduction data, applies optional navigation, and rerenders the visible state. |
| Server → player | `NAV_STATE` | Applies authoritative folder, entry, or command position and resets the local row selection. |
| Server → player | `HACK_STATE` | Renders attempts, board, log, success, or failure and triggers outcome audio. |
| Server → player | `TERMINAL_CLEAR` | Returns to idle presentation and stops ambient audio. |
| Player → server | `NAV_ACTION` | Requests `enter`, `entry`, `command`, or `back`; the client waits for server state before transitioning. |
| Player → server | `HACK_GUESS` / `HACK_ADMIN` | Requests a puzzle action; canonical puzzle behavior is outside this feature's scope. |
| Player → server | `GET /api/sounds/:folder` | Retrieves filenames only for a server-allowlisted sound category. |

## Edge Cases Observed

- A malformed incoming WebSocket frame is ignored after a console warning and does not replace the current view.
- Selection resets to the first row whenever authoritative navigation state is applied.
- A missing entry ID renders an empty record title and body rather than an explicit error state.
- Ambient playback begins only after a document click; keyboard interaction does not set the current user-gesture flag.
- Reveal timers are cancelled before a container starts a replacement render, preventing overlapping append sequences.
- Hacking words split at a 12-character row boundary render as separate spans but share one target ID and highlight together.
- The sound endpoint returns an empty list for unknown folders, unreadable directories, and directories without supported file extensions.
- The viewport prevents body scrolling; overflow is delegated to terminal body, command output, hacking columns, and hacking log containers.

## Success Criteria

- **SC-001**: A player can distinguish connecting, disconnected/reconnecting, idle, live terminal, active hacking, and blocked states by visible screen content.
- **SC-002**: A terminal containing nested folders, an empty folder, a multiline entry, and multiline command output can be completely read and operated with both pointer and supported keyboard controls.
- **SC-003**: Repeated rendering of an unchanged folder, entry, or command output does not replay its progressive reveal, while newly selected content does.
- **SC-004**: Text containing `<`, `>`, `&`, or HTML-like input is displayed as literal content and does not create executable markup.
- **SC-005**: Navigation and hacking input does not change shared content locally before a corresponding authoritative server message arrives.
- **SC-006**: Removing or corrupting an optional sound asset leaves all visible states and input paths usable.
- **SC-007**: Clearing a live broadcast returns every connected player client to the idle view and pauses its ambient loop.

## Assumptions

- The player client is intended for modern browsers with WebSocket, Fetch, Web Audio, CSS `clamp()`, and standard DOM support.
- The green CRT treatment and Russian player-facing copy are intentional parts of the current experience.
- Audio is atmospheric and optional; the browser's autoplay policy takes precedence over automatic playback.
- The display number is cosmetic and is not a stable server identity.
- The server remains responsible for validating actions and publishing canonical navigation and hacking state.
- Presentation of the hacking board is included here, while puzzle generation, attempt rules, secrets, and game-master controls remain specified by the migrated Hacking Game feature.

## Known Gaps

- No automated browser, DOM, accessibility, visual-regression, or sound-endpoint tests exist.
- There is no explicit narrow-screen media query; the hacking board's two-column layout and fixed proportional log panel may become cramped on phones.
- Selectable terminal rows and hacking cells are non-semantic elements without roles, ARIA labels, or native focus behavior.
- CRT flicker, blinking, and reveal animation do not honor `prefers-reduced-motion`.
- Ambient audio is unlocked only by a click, so keyboard-only users may never start it.
- Sound failures are intentionally silent and provide no diagnostic or user-visible muted state.
- Player-facing strings are hardcoded and no localization mechanism exists.
