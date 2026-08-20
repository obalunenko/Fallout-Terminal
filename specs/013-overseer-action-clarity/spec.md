# Feature Specification: Overseer Action Clarity

**Feature Branch**: `[013-overseer-action-clarity]`

**Created**: 2026-08-20

**Status**: Draft

**Input**: Make the Overseer controls for creating, activating, publishing, and removing a terminal from the air unambiguous through clearer names, placement, and interaction behavior.

## Clarifications

### Session 2026-08-20

- Q: What should happen to the “Make active” action when the selected terminal is already on air? → A: Show “On air” as status and move “Reapply settings” into an additional menu with an explanation.
- Q: When should the application ask for confirmation after the Overseer chooses “Take off air”? → A: Always confirm and explain that the broadcast, players, and saved terminal remain intact.
- Q: How should the “Create terminal” journey begin? → A: Open a dialog with a required name and create the non-live draft only after confirmation.
- Q: How should actions be distributed so their ownership is clear? → A: Put creation by the terminal list, activation and settings by the selected terminal, publication by the editor, and take-off-air by the live broadcast status.
- Q: Which final label set should the terminal actions use? → A: Use “+ СОЗДАТЬ ТЕРМИНАЛ”, “СДЕЛАТЬ АКТИВНЫМ”, “ОПУБЛИКОВАТЬ ИЗМЕНЕНИЯ”, “СНЯТЬ С ЭФИРА”, and “ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ”.

## User Scenarios & Testing

### User Story 1 - Understand terminal actions at a glance (Priority: P1)

As an Overseer, I can distinguish actions that edit the session from actions that immediately affect connected players, so I do not accidentally change the live experience.

**Why this priority**: Ambiguous neighboring controls can cause a live-session mistake even when every underlying operation works correctly.

**Independent Test**: Open a session with one active and one inactive terminal and verify that each visible action has a unique purpose, contextual location, and resulting status.

**Acceptance Scenarios**:

1. **Given** an inactive terminal is selected during a broadcast, **When** the Overseer views its controls, **Then** the interface offers an unambiguous action to put that terminal on air and does not offer publication for a terminal players are not viewing.
2. **Given** the active terminal is selected, **When** the Overseer views its controls, **Then** the interface identifies it as on air, clearly separates content publication from taking the terminal off air, and keeps full settings reapplication in an explained additional menu.
3. **Given** no broadcast is active, **When** the Overseer edits a session, **Then** session-editing actions remain distinguishable from unavailable live-session actions.
4. **Given** the full Overseer workspace is visible, **When** the Overseer scans available actions, **Then** creation is located with the terminal list, activation and settings with the selected terminal, publication with the editor, and taking off air with the current broadcast status.

---

### User Story 2 - Publish edits without changing live-session identity (Priority: P1)

As an Overseer, I can publish edited terminal content to players using one clearly named primary action while preserving the active terminal and its current live progress.

**Why this priority**: Content publication is the common live-editing operation and must not be confused with selecting or removing a terminal.

**Independent Test**: Edit content in the active terminal, publish it, and verify that connected players receive the change without switching terminal identity or restarting live progress.

**Acceptance Scenarios**:

1. **Given** the active terminal has saved content edits, **When** the Overseer publishes changes, **Then** all connected players receive the updated content and the interface confirms publication.
2. **Given** the active terminal has live navigation or hacking progress, **When** content is published, **Then** that progress is preserved according to the existing live-update behavior.
3. **Given** a different terminal is selected in the editor, **When** the Overseer views the editor actions, **Then** publication is unavailable until that terminal is active.

---

### User Story 3 - Create and remove terminal states safely (Priority: P2)

As an Overseer, I can create a terminal draft and take the current terminal off air without mistaking either operation for deleting content or ending the broadcast.

**Why this priority**: These are less frequent operations, but their present wording does not reveal whether they affect saved data, players, or the broadcast itself.

**Independent Test**: Create a terminal, take another terminal off air, and verify the saved session, broadcast, player assignments, and terminal list each retain the expected state.

**Acceptance Scenarios**:

1. **Given** a session is open, **When** the Overseer enters a valid name and confirms terminal creation, **Then** a new editable terminal draft is added to the session without appearing for players automatically.
2. **Given** a terminal is active, **When** the Overseer chooses to take it off air, **Then** a confirmation explains what remains intact and players lose the active terminal only after explicit confirmation.
3. **Given** a terminal exists in the session, **When** the Overseer chooses its delete action, **Then** deletion remains visibly and behaviorally distinct from taking it off air.

### Edge Cases

- A live action must not appear usable when no broadcast is active.
- A publication action must not target an inactive terminal merely because it is selected in the editor.
- Taking a terminal off air always requires confirmation; unfinished live progress may additionally require the existing preserve-or-discard decision before the terminal can leave the air.
- A failed live command must leave the previous active state visible and present an actionable error without implying success.
- Rapid repeated activation, publication, or take-off-air input must not produce duplicate requests or contradictory status messages.
- Creating a terminal when the session has no terminals must leave one clear editing selection.
- Cancelling terminal creation or submitting a blank name must not add or save a new terminal.

## Requirements

### Functional Requirements

- **FR-001**: The interface MUST place terminal creation with the terminal list, activation and additional settings with the selected terminal, content publication with the editor, and taking off air with the current broadcast status.
- **FR-002**: The interface MUST use the exact visible action labels `+ СОЗДАТЬ ТЕРМИНАЛ`, `СДЕЛАТЬ АКТИВНЫМ`, `ОПУБЛИКОВАТЬ ИЗМЕНЕНИЯ`, `СНЯТЬ С ЭФИРА`, and `ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ`, each only for its corresponding result.
- **FR-003**: Creating a terminal MUST first open a dialog with a required non-blank name and MUST add the non-live editable draft only after explicit confirmation, without publishing it to players.
- **FR-004**: An inactive selected terminal MUST offer a clear action to make it active during an existing broadcast.
- **FR-005**: The active selected terminal MUST show the exact status `В ЭФИРЕ` instead of a primary activation button, while full reapplication remains available as `ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ` in an additional menu with an explanation of its broader effect.
- **FR-006**: The active selected terminal MUST offer one primary publication action that sends its current content to players while preserving existing live-update semantics.
- **FR-007**: Taking the active terminal off air MUST always require confirmation that explicitly states the broadcast, player assignments, saved terminal, and session remain intact.
- **FR-008**: Terminal deletion MUST remain a separate destructive action associated with one saved terminal and MUST NOT share wording with taking a terminal off air.
- **FR-009**: Live controls MUST expose disabled, pending, success, and failure states that accurately reflect whether an action can run and whether it completed.
- **FR-010**: The revised controls MUST remain keyboard accessible and MUST restore focus to a relevant surviving control after dialogs or state transitions.
- **FR-011**: Existing persistence, terminal runtime, player synchronization, and broadcast semantics MUST remain unchanged except where an accepted clarification explicitly changes the interaction policy.

### Impacted Application Surfaces

- **Composition and Wails bridge (`main.go`, `app.go`)**: Not expected to change; existing trusted commands remain the action boundary unless clarification reveals a missing operation.
- **Domain and canonical state (`internal/domain/`, `internal/nav/`, `internal/hack/`, `internal/live/`, `internal/control/`)**: Existing lifecycle semantics remain authoritative and are not redesigned.
- **Persistence (`internal/session/`, `internal/playerconfig/`, `sessions/`)**: Session format is unchanged; terminal creation continues through the existing saved session.
- **Player transport (`internal/player/`)**: No contract change; players continue receiving authoritative live projections.
- **Platform and public access (`internal/platform/`, `internal/tunnel/`)**: Not affected except focused static UI verification under platform tests.
- **Overseer UI (`frontend/overseer/src/`)**: Affected; labels, visibility, grouping, helper text, and confirmation behavior are the primary scope.
- **Player UI (`frontend/client/`)**: Not affected; observable live outcomes must remain compatible.
- **Tests and fixtures (`internal/**/*_test.go`, `tests/browser/`, `internal/testutil/`)**: Affected; focused resource assertions and browser journeys must cover the clarified control model.
- **Build and packaging (`go.mod`, `frontend/`, `build/`, `scripts/`)**: No dependency or packaging design change; normal Overseer and native build gates still apply.

### State and Contract Requirements

- **Session/player-config compatibility**: Portable session version 1 and player configuration behavior remain unchanged.
- **Wails bridge and event contract**: Existing explicit activation, live-update, and clear commands remain private to Overseer; no generic action dispatcher is introduced.
- **WebSocket contract**: Existing authoritative player updates and reconnect behavior remain unchanged.
- **Reconnect and multi-tab behavior**: Reconnected players receive the same authoritative active-or-no-active-terminal state as before the UI clarification.
- **HTTP/static contract**: No route or origin-policy change.
- **Runtime-state lifecycle**: Creation affects durable authored state; activation selects a live terminal; publication refreshes live content; taking off air clears only the live selection; deletion removes authored state only through its separate existing action.

### Security and Privacy Requirements

- The revised controls MUST NOT expose privileged Overseer commands to player-facing surfaces.
- Existing validation at the trusted desktop boundary MUST remain in force for every live action.

### Verification Requirements

- **Go tests**: Focused static resource tests must verify canonical labels, contextual visibility, and separation of destructive and live actions.
- **Race testing**: Not required unless implementation changes concurrent runtime behavior rather than UI orchestration only.
- **Browser tests**: An Overseer journey must cover inactive, active, publication, terminal creation, take-off-air, confirmation, error, and keyboard-focus states.
- **Interactive verification**: Exercise the revised controls with one active terminal and at least one connected player.
- **Packaging/release verification**: The Overseer production build and native application build must pass; signing and notarization are not required for this UI-only change.

### Key Entities

- **Saved terminal**: An authored terminal in the session, whether or not players currently see it.
- **Selected terminal**: The saved terminal currently open in the Overseer editor.
- **Active terminal**: The one terminal currently presented to players during a broadcast.
- **Broadcast**: The live session epoch that retains players, roles, and controller state even when no terminal is active.
- **Terminal action**: A control whose scope is session editing, live selection, content publication, taking off air, or deletion.

## Success Criteria

### Measurable Outcomes

- **SC-001**: In every supported selected-terminal and broadcast state, each visible primary action maps to exactly one of creation, activation, publication, taking off air, or deletion.
- **SC-002**: No screen presents two neighboring primary controls whose labels both communicate an unspecified form of terminal update.
- **SC-003**: Publishing an edited active terminal updates all connected players while preserving active identity and existing live progress in all automated acceptance scenarios.
- **SC-004**: Taking a terminal off air leaves the broadcast, player assignments, session file, and saved terminal intact in all automated acceptance scenarios.
- **SC-005**: Terminal creation never publishes the new draft before an explicit activation action.
- **SC-006**: All affected keyboard, browser, resource, production frontend, and native build checks pass without changing player-facing or persistence contracts.

## Assumptions

- Overseer remains the only role authorized to create, activate, publish, take off air, or delete terminals.
- Editing continues to autosave the explicitly opened session file independently of live publication.
- One broadcast can have either one active terminal or no active terminal.
- This feature clarifies the existing action model rather than redesigning terminal runtime rules.

## Out of Scope

- Changing the portable session format or player configuration format.
- Adding new player-facing controls or network operations.
- Redesigning terminal content editing, hacking rules, player assignment, or broadcast start/end behavior.
- Renaming stable backend contract types that are not visible in the Overseer interface.
