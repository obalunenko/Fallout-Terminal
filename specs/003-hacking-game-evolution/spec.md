# Feature Specification: Immersive Hacking Game

## Clarifications

### Session 2026-08-11

- Q: How many valid special-pattern bracket sets must each newly generated board contain? → A: 3–6 inclusive.

## User Scenarios & Testing

### User Story 1 - Solve Without Player Cheats (Priority: P1)

As a player, I face the hacking puzzle using password guesses and authentic terminal interactions only, so success feels earned and follows the rules of the Fallout games.

**Why this priority**: Removing the existing shortcuts is the core immersion goal and establishes the fair-play boundary for every other story.

**Independent Test**: Start a hacking puzzle, try every previously available player shortcut, and confirm that none can directly expose or force the answer while normal password guessing still works.

**Acceptance Scenarios**:

1. **Given** an active puzzle, **When** a player enters the former administrator command, **Then** the puzzle state does not change and no incorrect passwords are removed.
2. **Given** a newly generated board, **When** the player reviews its selectable content, **Then** no administrator entry or other player-selectable control can directly force success or bulk-remove incorrect passwords.
3. **Given** an active puzzle, **When** the player selects a candidate password, **Then** the existing password-match and attempt rules continue to determine the result.

### User Story 2 - Use Special Patterns (Priority: P1)

As a player, I can discover and activate bracketed symbol patterns on the board for a chance to remove an incorrect password or recover my attempts, giving me an immersive way to improve my odds without bypassing the puzzle.

**Why this priority**: Special patterns replace the removed shortcut with the intended risk-and-reward mechanic.

**Independent Test**: Generate boards containing each allowed bracket type, activate unused patterns repeatedly across controlled outcomes, and verify the 80/20 effect split, one-use rule, and shared board updates.

**Acceptance Scenarios**:

1. **Given** an unused valid pattern, **When** the player points to its opening bracket, **Then** the complete pattern from opening bracket through matching closing bracket is highlighted as one selectable target.
2. **Given** an unused valid pattern and at least one incorrect password remains, **When** the pattern produces the dud-removal outcome, **Then** exactly one selectable incorrect password is replaced by periods and the correct password remains unchanged.
3. **Given** an unused valid pattern, **When** the pattern produces the attempt-restoration outcome, **Then** the remaining attempts return to the puzzle maximum without exceeding it.
4. **Given** a pattern has already been activated, **When** the player points to or selects the same opening-and-closing pair again, **Then** it cannot be highlighted or produce another effect.
5. **Given** multiple players share a puzzle, **When** any player activates a pattern, **Then** all players see the same effect and the same used-pattern state.

### User Story 3 - Discover Stacked and Dynamic Patterns (Priority: P1)

As a player, I can find overlapping patterns and newly formed patterns after a dud disappears, so careful board inspection remains rewarding as the puzzle changes.

**Why this priority**: Stacked and dynamically revealed patterns are explicit parts of the authentic interaction and must work in the first complete release.

**Independent Test**: Exercise a board with multiple compatible opening brackets sharing one closing bracket, then remove a dud whose replacement periods create another valid pair and verify every distinct pattern becomes usable once.

**Acceptance Scenarios**:

1. **Given** two compatible opening brackets can reach the same closing bracket on one row, **When** the player points to each opening bracket, **Then** each opening identifies a separate selectable pattern ending at that shared closing bracket.
2. **Given** one of two stacked patterns has been used, **When** the player points to the other unused opening bracket, **Then** the other pattern remains selectable.
3. **Given** an incorrect password separates a compatible opening and closing bracket, **When** a dud-removal effect replaces that password with periods and the resulting span meets the pattern rules, **Then** the newly valid pattern becomes immediately selectable.
4. **Given** a board change creates a pattern, **When** players inspect the updated board, **Then** every connected player sees the same newly available pattern without restarting the puzzle.

### User Story 4 - Let the Game Master Resolve the Puzzle (Priority: P1)

As a game master, I can solve an active puzzle from my application so I can keep the session moving when the story or table situation requires intervention.

**Why this priority**: Removing player cheats must not remove the game master's trusted recovery control.

**Independent Test**: Start an unsolved puzzle, use the game-master solve control, and verify that all players receive the solved result and proceed to the terminal content.

**Acceptance Scenarios**:

1. **Given** an active unsolved puzzle, **When** the game master presses the solve control, **Then** the puzzle is marked solved without consuming an attempt.
2. **Given** the game master solves the puzzle, **When** the result is shared, **Then** all connected players receive the solved state and transition to normal terminal access through the existing success flow.
3. **Given** there is no active eligible puzzle, **When** the game master views the hacking controls, **Then** the solve control is unavailable and cannot alter session state.

### User Story 5 - Preserve Shared Progress Across Connections (Priority: P2)

As a group of players, we see one authoritative puzzle state, so pattern availability, removed duds, attempts, and outcomes remain consistent when people act concurrently or reconnect.

**Why this priority**: A shared terminal experience is only trustworthy if special-pattern progress cannot diverge between players.

**Independent Test**: Connect two players, activate and reveal patterns from both clients, reconnect one client, and compare the board, attempts, used patterns, and outcome after every action.

**Acceptance Scenarios**:

1. **Given** multiple players are connected, **When** two players attempt to activate the same unused pattern, **Then** the pattern produces at most one effect.
2. **Given** a pattern has been used or dynamically created, **When** a player reconnects, **Then** the player receives its current availability and the current board.
3. **Given** a puzzle has ended, **When** a player submits a pattern action, **Then** attempts, board contents, logs, and outcome remain unchanged.
4. **Given** a new hacking attempt begins, **When** its board is generated, **Then** pattern availability is recalculated for the new board and no used-pattern state carries over.

## Edge Cases

- What happens when an 80% dud-removal outcome is selected after no incorrect password remains? The pattern restores attempts instead so every valid activation has a useful result.
- Attempt restoration returns attempts to the configured maximum and never adds attempts above that maximum.
- A valid pattern must use matching bracket types, remain within one horizontal row, and contain no alphabetic characters between its endpoints.
- Each opening bracket pairs with the first compatible closing bracket to its right on the same row; this allows multiple openings to share a closing bracket while keeping the target deterministic.
- A closing bracket before its opening bracket, a mismatched bracket type, a cross-row span, or a span containing alphabetic characters is not selectable.
- Replacing a dud with periods may create, remove, or change overlapping pattern targets; availability must be recalculated from the current board immediately after the replacement.
- Activating one stacked pattern does not consume another pattern that has a different opening position, even when they share a closing position.
- Repeated, stale, malformed, or tampered pattern selections do not consume attempts and do not change the puzzle.
- Pattern actions received after success or failure have no effect.
- If simultaneous player actions target the same pattern, only the first accepted action applies its random outcome.

## Requirements

### Functional Requirements

- **FR-001**: The player experience MUST provide no command, board entry, or other selectable action that directly forces puzzle success.
- **FR-002**: The player experience MUST remove the former administrator shortcut that bulk-removes incorrect passwords.
- **FR-003**: The game master MUST retain a solve control that completes an active unsolved puzzle without consuming an attempt.
- **FR-004**: A game-master solve action MUST publish the solved result to every connected player through the normal shared success flow.
- **FR-005**: A special pattern MUST begin and end with one matching pair from the four allowed bracket types.
- **FR-006**: A special pattern MUST have its opening and closing bracket on the same horizontal board row.
- **FR-007**: A special pattern MUST contain no alphabetic characters between its opening and closing bracket.
- **FR-008**: Pointing to the opening bracket of an unused valid pattern MUST highlight the entire pattern through its matching closing bracket.
- **FR-009**: Selecting an unused valid pattern MUST produce exactly one effect for the shared puzzle.
- **FR-010**: When at least one incorrect password remains, each valid pattern activation MUST have an 80% chance to remove one incorrect password and a 20% chance to restore attempts.
- **FR-011**: A dud-removal effect MUST replace exactly one selectable incorrect password with periods without removing the correct password.
- **FR-012**: An attempt-restoration effect MUST return remaining attempts to the puzzle maximum without exceeding that maximum.
- **FR-013**: Each newly generated board MUST contain a random number of valid special patterns from 3 through 6 inclusive.
- **FR-014**: The special-pattern count range MUST NOT vary by hacking difficulty.
- **FR-015**: Each distinct opening-position and closing-position pair MUST be usable at most once during a puzzle.
- **FR-016**: Multiple compatible opening brackets MUST each form a separate pattern when they share a compatible closing bracket on the same row.
- **FR-017**: Using one stacked pattern MUST NOT consume another pattern with a different opening position.
- **FR-018**: Replacing a dud with periods MUST immediately make any newly valid pattern selectable during the same puzzle.
- **FR-019**: When no incorrect password remains, an otherwise valid dud-removal outcome MUST restore attempts instead.
- **FR-020**: Pattern activation, pattern availability, used-pattern state, removed duds, attempts, and puzzle outcome MUST be shared consistently across all connected players.
- **FR-021**: Concurrent attempts to use the same pattern MUST produce no more than one effect.
- **FR-022**: A reconnecting player MUST receive the current board and current pattern availability without regenerating the puzzle.
- **FR-023**: Malformed, stale, tampered, repeated, or terminal-state pattern actions MUST leave puzzle state unchanged.
- **FR-024**: Starting a fresh hacking attempt MUST discard the previous puzzle's pattern availability and used-pattern state.

## Key Entities

- **Special Pattern**: A selectable span on one board row, identified by its bracket type, opening position, closing position, current availability, and whether it has been used.
- **Hacking Puzzle**: The shared challenge containing the board, candidate passwords, correct password, attempt limit, remaining attempts, special patterns, progress log, and outcome.
- **Candidate Password**: A selectable word that is either the correct password or an incorrect dud and may be replaced by periods when a pattern removes it.
- **Pattern Effect**: The single outcome of activating a pattern: either one dud is removed or attempts are restored.
- **Player Action**: A request to guess a password or activate a special pattern against the current shared puzzle.
- **Game-Master Action**: A trusted request to solve the active puzzle for all players.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Across 100 controlled valid pattern activations with an available dud, observed outcomes match 80 dud removals and 20 attempt restorations.
- **SC-002**: Every allowed bracket type can be discovered and activated, while 100% of cross-row, mismatched, or alphabetic-content spans remain unavailable.
- **SC-003**: In all tested stacked-pattern boards, each distinct opening can be activated once even when multiple openings share a closing bracket.
- **SC-004**: In all tested dud removals that create a valid bracket span, the new pattern is selectable immediately without reloading or restarting the puzzle.
- **SC-005**: Repeated or simultaneous activation attempts against one pattern produce exactly one effect in 100% of tested cases.
- **SC-006**: Two connected players and one reconnecting player display identical attempts, board text, pattern availability, used-pattern state, and outcome after every accepted action.
- **SC-007**: Players have zero available actions that directly force success or trigger the removed administrator aid.
- **SC-008**: The game master can solve an eligible puzzle with one control action, and every connected player receives normal terminal access through the existing success sequence.
- **SC-009**: Fresh puzzles contain 3–6 valid special patterns in 100% of 1,000 generated-board checks, with no count dependency on hacking difficulty.

## Assumptions

- The existing password candidate, likeness, attempt-spending, shared-session, and success-transition rules remain unchanged except where this specification explicitly modifies them.
- A pattern's closing bracket is the first matching closing bracket to the right of its opening bracket on the same row.
- “No words inside” is interpreted as no alphabetic characters between the brackets; periods and other non-alphabetic symbols are allowed.
- Pattern positions are evaluated against the current visible board, so replacing a dud with periods can change the available set during play.
- Random outcomes are evaluated for each first-time valid activation and can be controlled in verification so the required distribution is tested exactly.
- If a dud-removal result cannot remove a dud, restoring attempts is the fallback so a valid one-use pattern is never wasted.
- Runtime hacking progress remains temporary and resets when a fresh hacking attempt begins.

## Verbatim Constraints

- Allowed pattern pairs: `()`, `[]`, `{}`, `<>`.
- Valid special-pattern count per newly generated board: `3–6` inclusive.
- Dud-removal probability: `80%`.
- Attempt-restoration probability: `20%`.
