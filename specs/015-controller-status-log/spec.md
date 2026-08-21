# Feature Specification: Immersive Controller Status Log

**Feature Branch**: `015-controller-status-log`
**Created**: 2026-08-20
**Status**: Draft

## User Scenarios & Testing

### User Story 1 - See controller ownership without breaking immersion (Priority: P1)

As a player using the terminal, I can identify my character, input channel, and active-controller state from a restrained ROBCO-style system line near the command prompt, so the information remains useful without looking like an external game lobby overlay.

**Why this priority**: This is the selected presentation and the primary reason for the change.

**Independent Test**: Open a normal terminal screen as the active controller assigned to Nick Valentine and confirm that the bottom system line communicates input channel, character, and active state while the former framed identity panel is absent.

**Acceptance Scenarios**:

1. **Given** the active session is the default first player and Nick Valentine is assigned, **When** a normal terminal menu is displayed, **Then** the bottom-left system line reads `[СИСТЕМА] ВВОД P1 // НИК ВАЛЕНТАЙН // АКТИВЕН` immediately above the command prompt.
2. **Given** the system line is visible, **When** the player scans the menu, **Then** the selected menu item remains the dominant visual element and no framed, coloured, or badge-like identity panel appears above it.
3. **Given** the terminal content changes between menus, entries, and command results, **When** the active session remains connected, **Then** the system line remains anchored to the lower terminal chrome and does not consume content space.

---

### User Story 2 - Retain clear status for every player role (Priority: P2)

As a player who is observing, selecting a character, waiting for a broadcast, or not yet assigned, I still receive an unambiguous session-role message in the same terminal-native format.

**Why this priority**: Moving the active-controller presentation must not remove the role clarity and accessibility already available to non-active sessions.

**Independent Test**: Exercise active, observer, and unassigned sessions and verify that each role has distinct text, updates when authority changes, and is announced as status information.

**Acceptance Scenarios**:

1. **Given** an observer session, **When** the player view renders, **Then** the bottom system line identifies that session as `НАБЛЮДАТЕЛЬ` without blue colouring, dashed borders, or a modern badge.
2. **Given** a session has no assigned character, **When** the status line renders, **Then** it communicates the fallback session identity without a blank separator segment.
3. **Given** controller authority or character assignment changes, **When** the updated session state arrives, **Then** the existing system line updates in place and assistive technology is notified without creating duplicate messages.

---

### User Story 3 - Preserve the terminal across viewport sizes (Priority: P3)

As a player on a compact or wide display, I can read the system line without it overlapping the prompt, navigation controls, hacking interface, or terminal content.

**Why this priority**: The chosen bottom placement is only successful if it remains reliable on every supported layout.

**Independent Test**: Render representative wide, narrow, and short viewports across normal and hacking surfaces and verify that the line remains legible or follows the established surface visibility rules without overlap.

**Acceptance Scenarios**:

1. **Given** a narrow viewport, **When** the system line exceeds the available width, **Then** it wraps or truncates within its own lower-chrome area without causing horizontal scrolling.
2. **Given** a short viewport, **When** terminal content fills the screen, **Then** the system line and prompt remain visible without covering selectable content.
3. **Given** a terminal surface where the normal command prompt is intentionally hidden, **When** that surface renders, **Then** the status line follows the established visibility behaviour and does not obstruct the specialised interface.

## Edge Cases

- The session has a custom fallback name rather than a default `PLAYER N` label.
- No character is assigned, or the assigned character is removed during the session.
- Authority changes from active to observer and back without a page reload.
- A character or session name is too long for one line on a narrow display.
- Session state is temporarily unavailable during connection or reconnection.
- The command prompt or normal terminal chrome is intentionally hidden for a specialised screen.

## Requirements

### Functional Requirements

- **FR-001**: The player view MUST replace the framed identity and role panel with a single ROBCO-style system status line in the lower terminal chrome.
- **FR-002**: The status line MUST present the current input-channel identity, assigned character when present, and current session role in that order without empty separator segments.
- **FR-003**: For the default first-player session assigned to Nick Valentine with active authority, the visible line MUST match `[СИСТЕМА] ВВОД P1 // НИК ВАЛЕНТАЙН // АКТИВЕН`.
- **FR-004**: Default session labels in the form `PLAYER N` MUST be displayed as the compact input identifier `PN`, while custom session labels remain recognisable.
- **FR-005**: Active, observer, and unassigned roles MUST remain distinguishable through terminal-native wording without yellow or blue role colours, outlined badges, or a framed background.
- **FR-006**: The status line MUST update in place when the assigned character, fallback session identity, or role changes.
- **FR-007**: Status changes MUST remain available to assistive technology without announcing duplicate or stale identity information.
- **FR-008**: The lower status presentation MUST avoid overlapping terminal content, prompt, navigation controls, and specialised terminal surfaces at supported viewport sizes.
- **FR-009**: The change MUST preserve controller-authority behaviour, character assignment behaviour, and overseer controls; only the player-facing presentation and wording are in scope.

## Success Criteria

### Measurable Outcomes

- **SC-001**: In the selected active-session scenario, the displayed status matches the approved system-line wording exactly and the previous framed identity panel is absent.
- **SC-002**: All active, observer, and unassigned role scenarios communicate the correct current role after initial load, reassignment, and reconnection.
- **SC-003**: At representative wide, narrow, and short supported viewports, zero status-line elements overlap selectable terminal content, navigation, or the command prompt.
- **SC-004**: Existing player navigation and state-changing command journeys continue to pass with the updated role wording and placement.
- **SC-005**: A keyboard or screen-reader user receives no more than one current identity/role status announcement per rendered state change.

## Assumptions

- The selected third concept is a persistent low-priority system-log line while session state is available, rather than a timed notification that disappears and removes role visibility.
- The line remains part of the player terminal only; the overseer interface retains its existing presentation.
- Existing session state is sufficient to derive the displayed input identifier, character, and role; no new player data is required.
- Existing specialised-screen visibility rules remain authoritative when the normal prompt is hidden.

## Out of Scope

- Changing how active-controller authority is assigned or transferred.
- Changing character selection, session persistence, or broadcast coordination.
- Redesigning overseer session controls.
- Adding a notification timer, animation, sound, or new colour palette.

## Verbatim Constraints

- `[СИСТЕМА] ВВОД P1 // НИК ВАЛЕНТАЙН // АКТИВЕН`
- `[СИСТЕМА]`
- `ВВОД`
- `АКТИВЕН`
- `НАБЛЮДАТЕЛЬ`
