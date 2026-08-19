# Feature Specification: Player Intelligence and Hacker Perk Management

**Feature directory**: `011-player-intelligence-hacker`  
**Scope**: Store Intelligence and Hacker perk availability for each configured player, and provide inactive-session roster management in a dedicated pop-up window.

## User Scenarios & Testing

### User Story 1 - Add Players with Gameplay Attributes (Priority: P1)

As the game master preparing a session, I can add each player with a name, an Intelligence level from 1 through 10, and whether the Hacker perk is available, so the complete player profile is ready before play begins.

**Why this priority**: Capturing valid player attributes is the core value of the feature and provides the data needed by every later roster workflow.

**Independent Test**: Open an inactive session with a selected player configuration, add one player with valid values in the player pop-up, reopen the configuration, and verify that the same name, Intelligence level, and Hacker perk availability are shown.

**Acceptance Scenarios**:

1. **Given** a player configuration is selected and no broadcast is active, **when** the game master opens the player pop-up and adds a player with a non-blank name, an Intelligence level from 1 through 10, and a Hacker perk choice, **then** the player appears in the detailed list with all three values.
2. **Given** the add-player form is open, **when** the game master enters an Intelligence value below 1, above 10, or not a whole number, **then** the player is not added and the invalid Intelligence value is identified.
3. **Given** the add-player form is open, **when** the game master omits the name, Intelligence level, or Hacker perk choice, **then** the player is not added and the missing value is identified.
4. **Given** a player was added successfully, **when** the player configuration is closed and reopened, **then** the player's name, Intelligence level, and Hacker perk availability are unchanged.

---

### User Story 2 - Edit the Inactive Player Roster (Priority: P1)

As the game master between active sessions, I can update or remove players and correct their stored attributes in the player pop-up, so the roster remains accurate without disrupting live play.

**Why this priority**: Preparation changes and corrections are unavoidable, but they must not alter the roster while players are using it.

**Independent Test**: With no broadcast active, edit a player's name, Intelligence level, and Hacker perk availability, remove another player, reload the configuration, and verify that only the requested changes were persisted.

**Acceptance Scenarios**:

1. **Given** no broadcast is active and the detailed player list contains a player, **when** the game master changes the player's name, Intelligence level, or Hacker perk availability to valid values, **then** the updated values are displayed and retained after reload.
2. **Given** no broadcast is active and the detailed player list contains a player, **when** the game master confirms removal, **then** that player is removed from the stored roster and the remaining players are unchanged.
3. **Given** a broadcast is active, **when** the game master opens the player pop-up, **then** the current roster and attributes remain visible but all roster-changing actions are unavailable.
4. **Given** a broadcast becomes active while the player pop-up is open, **when** the game master attempts a roster change using stale controls or another request path, **then** the change is rejected and the stored roster remains unchanged.
5. **Given** an otherwise valid roster change cannot be stored, **when** the operation fails, **then** the detailed list continues to show the last stored values and the game master sees an actionable error.

---

### User Story 3 - Review Player Details in a Dedicated Window (Priority: P2)

As the game master, I can open a dedicated pop-up window containing the detailed player list, so roster details and editing controls do not crowd the main session screen.

**Why this priority**: A focused detail view makes the new attributes usable while preserving the main screen for session and broadcast control.

**Independent Test**: From the master screen, open the player pop-up for both a populated and an empty roster, verify its detailed or empty presentation, and close it without changing session state.

**Acceptance Scenarios**:

1. **Given** a player configuration is selected, **when** the game master invokes player management from the main screen, **then** a new pop-up window opens with one detailed entry per configured player.
2. **Given** the player pop-up contains roster entries, **when** the game master reviews the list, **then** each entry clearly shows the player's name, Intelligence level, and Hacker perk availability.
3. **Given** the selected player configuration has no players, **when** the game master opens the pop-up, **then** the window shows a clear empty-roster state and, when no broadcast is active, offers the add-player controls.
4. **Given** the player pop-up is open, **when** the game master closes it without submitting a change, **then** the player configuration and session state remain unchanged.

## Edge Cases

- An Intelligence value of exactly 1 or exactly 10 is valid; fractional, non-numeric, and out-of-range values are invalid.
- A player name containing only whitespace is treated as missing; names otherwise retain the roster's established length and character rules.
- If a broadcast starts while the pop-up is open, the view must immediately become read-only and reject any already-prepared change.
- If the selected player configuration is removed, unreadable, or replaced while the pop-up is open, no partial update may be applied and the game master must be prompted to reopen or reselect it.
- Repeated submission of the same add, update, or removal action must not create duplicate players or apply the change more than once.
- Existing player configurations that predate these attributes must remain openable and receive consistent default attribute values.
- Reloading or restarting after a successful change must reproduce the same roster order, player identities, and stored attributes.

## Requirements

### Functional Requirements

- **FR-001**: The master screen MUST provide a control that opens player management in a dedicated pop-up window.
- **FR-002**: The player-management pop-up MUST display every configured player's name, Intelligence level, and Hacker perk availability.
- **FR-003**: The player-management pop-up MUST show a clear empty-roster state when the selected player configuration contains no players.
- **FR-004**: The system MUST allow the game master to add a player only when a player configuration is selected and no broadcast is active.
- **FR-005**: Adding a player MUST require a non-blank name, a whole-number Intelligence level from 1 through 10 inclusive, and an explicit Hacker perk availability choice.
- **FR-006**: The system MUST store each player's Intelligence level and Hacker perk availability as part of that player's persistent roster entry.
- **FR-007**: The system MUST allow the game master to change a player's name, Intelligence level, and Hacker perk availability when no broadcast is active.
- **FR-008**: The system MUST allow the game master to remove a player from the roster when no broadcast is active.
- **FR-009**: The system MUST reject every add, edit, or removal attempt while a broadcast is active without changing the stored roster.
- **FR-010**: The player-management pop-up MUST remain available as a read-only detailed list while a broadcast is active.
- **FR-011**: Editing a player's attributes MUST preserve that player's stable identity and roster position.
- **FR-012**: A successful roster mutation MUST be stored before the interface reports the operation as complete.
- **FR-013**: A failed roster mutation MUST leave the last stored roster unchanged and present an actionable error to the game master.
- **FR-014**: Reopening a player configuration MUST restore every player's stored name, Intelligence level, Hacker perk availability, identity, and roster order.
- **FR-015**: A player configuration created before this feature MUST load without manual repair by assigning Intelligence 1 and Hacker perk unavailable to entries that do not contain those attributes.
- **FR-016**: Closing the player-management pop-up without submitting a roster mutation MUST leave the player configuration and current session state unchanged.

## Key Entities

- **Player roster entry**: A persistent game-master-defined player identity containing a stable identity, display name, Intelligence level from 1 through 10, and Hacker perk availability.
- **Player configuration**: The reusable, ordered collection of player roster entries selected for a session and stored independently of live connections and assignments.
- **Session activity state**: Whether a broadcast is active; it determines whether the roster pop-up permits changes or presents player details read-only.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A game master can add a valid player with all required attributes from the pop-up in under 60 seconds.
- **SC-002**: In acceptance testing, 100% of valid Intelligence boundary values and Hacker perk choices are retained after closing and reopening the player configuration.
- **SC-003**: In acceptance testing, 100% of roster mutation attempts made during an active broadcast are rejected without changing any stored player entry.
- **SC-004**: For populated rosters, the game master can identify every player's name, Intelligence level, and Hacker perk availability from the pop-up without navigating away from it.
- **SC-005**: Existing player configurations without the new attributes open successfully without manual file changes in all compatibility tests.
- **SC-006**: Failed storage operations produce no partial roster changes in all tested add, edit, and removal cases.

## Assumptions

- “Player” refers to the existing persistent character roster entry managed by the game master, not a transient connected browser session.
- “Session is active” means the session currently has an active broadcast; an open session document without a broadcast is considered inactive for roster editing.
- Hacker perk availability is a yes-or-no player attribute.
- This feature stores and displays Intelligence and Hacker perk availability but does not change hacking rules or automatically grant gameplay effects from those values.
- Existing rules for player-name length, roster size, stable identities, and player-configuration selection continue to apply.
