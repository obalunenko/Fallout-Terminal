---
status: migrated
feature: Hacking Game
source: existing implementation
---

# Feature Specification: Hacking Game

**Migration status**: Reverse-engineered from the existing implementation on 2026-08-09  
**Scope**: Server-authoritative hacking gate, player experience, game-master controls, and persisted difficulty setting

**Current runtime**: The behavior contract is unchanged under Wails v2. The
mutex-protected Go live/hack services own private puzzle state; the master uses
an explicitly bound desktop method and receives only the sanitized `hack-state`
event.

## Purpose

The hacking game optionally gates access to a live terminal behind a shared Fallout-style word puzzle. The game master configures the difficulty for each terminal, the server creates and owns the active puzzle, and every connected player sees and acts on the same public state. The secret answer remains on the server.

## User Scenarios and Acceptance

### User Story 1 — Configure a terminal's hacking gate (Priority: P1)

As a game master, I can choose whether a terminal requires hacking and select a difficulty so that access matches the encounter I am running.

**Independent verification**: Set a terminal to each available value, save it, rebroadcast it, and inspect the player screen.

**Acceptance scenarios**:

1. **Given** a terminal has hacking level 0, **when** it becomes live, **then** players enter normal terminal navigation without seeing a hacking board.
2. **Given** a terminal has hacking level 1 through 5, **when** it becomes live, **then** players see a fresh hacking board before terminal navigation.
3. **Given** an active puzzle is in progress, **when** the game master changes the configured level and applies the settings, **then** the current puzzle is not interrupted and the new level takes effect on the next rebroadcast.
4. **Given** a configured terminal is saved and reopened, **when** its settings render, **then** the persisted `hackLevel` is selected.

---

### User Story 2 — Attempt the shared word puzzle (Priority: P1)

As a player, I can select candidate words or filler characters and receive immediate feedback so that I can discover the password before exhausting the shared attempts.

**Independent verification**: Broadcast a hacking-enabled terminal, select wrong words, filler characters, and the correct word, and compare the attempt counter and log after each action.

**Acceptance scenarios**:

1. **Given** an unsolved puzzle with attempts remaining, **when** a player selects a wrong candidate word, **then** one attempt is consumed and the log reports the number of characters matching the secret in the same positions.
2. **Given** an unsolved puzzle with attempts remaining, **when** a player selects filler outside a candidate word, **then** one attempt is consumed and the log reports zero positional matches.
3. **Given** an unsolved puzzle, **when** a player selects the secret word, **then** the puzzle becomes solved without consuming another attempt and the success sequence is shown.
4. **Given** the final attempt is consumed by an incorrect selection, **when** the updated state arrives, **then** the puzzle becomes failed and terminal access remains blocked.
5. **Given** a puzzle is already solved or failed, **when** another guess is received, **then** game state does not change.

---

### User Story 3 — Use the administrator aid (Priority: P2)

As a player, I can discover and activate the hidden administrator entry so that most decoy words are removed while the correct answer remains available.

**Independent verification**: Enter `1` from the hacking screen, inspect the broadcast board, and confirm that the answer and at most one decoy remain selectable.

**Acceptance scenarios**:

1. **Given** an active puzzle, **when** the player types `1` and presses Enter, **then** the client requests administrator mode.
2. **Given** administrator mode has not been used, **when** the server applies it, **then** the secret word, the administrator entry, and at most one ordinary decoy remain selectable while other candidate words are replaced with dots.
3. **Given** administrator mode has already mutated the board, **when** it is requested again, **then** no additional candidate words are removed.
4. **Given** the puzzle is solved or failed, **when** administrator mode is requested, **then** game state does not change.

---

### User Story 4 — Share one authoritative puzzle (Priority: P1)

As a group of players, we see the same attempts, board, log, and outcome so that actions from any browser affect the shared terminal consistently.

**Independent verification**: Connect two browsers, make guesses from each, disconnect and reconnect one browser, and compare both views after every action.

**Acceptance scenarios**:

1. **Given** multiple connected players, **when** any player submits a valid hacking action, **then** the server mutates one canonical puzzle and broadcasts the resulting `HACK_STATE` to all players.
2. **Given** a hacking-enabled terminal is already live, **when** a player connects or reconnects, **then** the initial `TERMINAL_LIVE` message contains the current public puzzle state.
3. **Given** the game master restarts or changes the live terminal, **when** `TERMINAL_LIVE` is broadcast, **then** the server creates a fresh puzzle for the configured level.
4. **Given** a public puzzle payload, **when** a client inspects it, **then** it does not contain `secretWord` or the private `wordsById` lookup.

---

### User Story 5 — Monitor and override the puzzle as game master (Priority: P2)

As a game master, I can monitor shared progress and force success so that the session can continue if the encounter requires intervention.

**Independent verification**: Watch the master status while a player guesses, invoke forced success, and verify the master and all players receive the solved state.

**Acceptance scenarios**:

1. **Given** a live hacking-enabled terminal, **when** attempts or outcome change, **then** the master interface displays remaining attempts, success, or blocked status.
2. **Given** an active unsolved puzzle, **when** the game master selects forced success, **then** the server marks it solved and broadcasts the sanitized solved state.
3. **Given** a solved or failed puzzle, **when** the master status renders, **then** the forced-success control is disabled.
4. **Given** a puzzle is solved on a player client, **when** the success state arrives, **then** success audio plays and normal navigation appears after approximately 2.6 seconds.

## Functional Requirements

- **FR-001**: Each terminal MUST persist a numeric `hackLevel`; level 0 disables the gate and levels 1 through 5 enable it.
- **FR-002**: Starting or restarting a live hacking-enabled terminal MUST generate a fresh server-owned puzzle.
- **FR-003**: Levels 1 through 5 MUST use candidate word lengths 4 through 8 respectively.
- **FR-004**: A generated puzzle MUST start with four attempts and contain 12 through 16 ordinary candidate words according to level, subject to the available word bank and board placement.
- **FR-005**: The puzzle MUST render two columns of 16 rows, with 12 characters per row and generated hexadecimal row addresses.
- **FR-006**: The server MUST retain the secret word and private candidate lookup and MUST omit both from public live and hacking-state payloads.
- **FR-007**: A correct candidate selection MUST mark the puzzle solved and append success messages to the shared log.
- **FR-008**: An incorrect candidate selection MUST consume one attempt and report positional character matches as `matches/wordLength`.
- **FR-009**: A valid filler selection outside a candidate word MUST consume one attempt and report zero positional matches.
- **FR-010**: Attempts MUST NOT fall below zero; reaching zero after an incorrect selection MUST mark the puzzle failed.
- **FR-011**: Guess and administrator actions MUST be ignored after the puzzle is solved or failed.
- **FR-012**: The server MUST reject malformed JSON, unsupported message types, non-string guess targets, invalid coordinates, and stale or tampered filler references without mutating the puzzle.
- **FR-013**: Administrator mode MUST preserve the secret word, preserve at most one ordinary decoy, replace removed words with dots, and operate at most once per puzzle.
- **FR-014**: Player hacking actions MUST be requests to the server; clients MUST NOT mutate canonical attempts, board contents, logs, or outcome locally.
- **FR-015**: Every accepted hacking action MUST result in a sanitized `HACK_STATE` broadcast to all connected clients and a master-status notification.
- **FR-016**: A newly connected client MUST receive the current sanitized puzzle in `TERMINAL_LIVE` when a terminal is live.
- **FR-017**: Solving the puzzle MUST transition the player from hacking mode to normal navigation after the existing success delay; failure MUST keep navigation blocked until the live terminal is restarted.
- **FR-018**: The game master MUST be able to force success through the sandboxed bound-desktop boundary without exposing Node.js APIs to the frontend.
- **FR-019**: Applying a difficulty change to an already-live terminal MUST NOT reset the active puzzle; the changed difficulty MUST apply on the next start or restart of the broadcast.
- **FR-020**: Stopping the live terminal MUST clear the active puzzle and notify player and master interfaces.

## Public Protocol Contract

| Direction | Message | Required behavior |
|---|---|---|
| Player → server | `HACK_GUESS` | Carries a string `targetId` identifying a word or `columnIndex:characterIndex` filler position. |
| Player → server | `HACK_ADMIN` | Requests the one-time administrator aid; no payload is required. |
| Server → players | `HACK_STATE` | Carries `hack`, the current sanitized public puzzle state. |
| Server → player | `TERMINAL_LIVE` | Includes `hackLevel` and current sanitized `hack` state during initial live publication or reconnection. |
| Go runtime → master frontend | `hack-state` | Carries sanitized progress for the status panel. |
| Master frontend → Go runtime | `ForceHackSuccess` | Requests privileged game-master success through the narrow compatibility facade. |

The public `hack` object contains `level`, `wordLength`, `attemptsMax`, `attemptsLeft`, `solved`, `failed`, `log`, and `columns`. Each column contains public addresses, board text, and selectable word locations. It excludes the secret and the private ID-to-word lookup.

## Edge Cases Observed

- Invalid or missing levels passed to board generation fall back to four-letter words, although persisted level values are not currently validated.
- A candidate may be skipped if no valid board position is found after 300 placement attempts.
- Repeated administrator requests do not remove more words, but currently append the activation log line again.
- Previously guessed words remain selectable and may consume further attempts.
- Concurrent browser messages are serialized by the mutex-protected Go live service against shared in-memory state.
- Runtime puzzle state is not stored in session JSON; rebroadcasting generates a new puzzle.

## Success Criteria

- **SC-001**: For each enabled level 1 through 5, a fresh broadcast presents a board whose candidate length matches 4 through 8 characters respectively.
- **SC-002**: A newly generated puzzle exposes exactly four attempts to every connected player and the master status.
- **SC-003**: After an action from any connected browser, all connected browsers display the same attempts, outcome, log, and board from the next server broadcast.
- **SC-004**: Inspecting `TERMINAL_LIVE` and `HACK_STATE` payloads reveals neither `secretWord` nor `wordsById`.
- **SC-005**: Correct guesses and game-master overrides unlock navigation; four incorrect selections produce the blocked screen.
- **SC-006**: Reconnecting during an active, solved, or failed puzzle restores the current public state without generating a new board.
- **SC-007**: A configured difficulty survives session save and reopen, while runtime puzzle state remains absent from the saved session.

## Assumptions

- The game is deliberately shared: any connected player can spend the group's attempts.
- The word bank is English while interface feedback is primarily Russian.
- The `SUCCESS` board entry is the implemented administrator trigger, while the player's keyboard command `1` sends the same server action without selecting that entry.
- The game master is trusted and can force success from the local Wails desktop interface.
- Existing randomness is acceptable for play; deterministic generation is not part of the current contract.

## Known Gaps

- No automated tests cover pure game logic, public-state redaction, protocol validation, persistence, or multi-client convergence.
- Loaded session files do not validate `hackLevel` as an integer from 0 through 5.
- Repeated administrator requests can add duplicate activation messages.
- Previously guessed candidates are not disabled or recorded as consumed.
- Protocol payloads have no executable schema validation or version field.
- Random generation has no injectable seed or deterministic test seam.
- Accessibility expectations beyond the existing mouse and keyboard interaction are undocumented.
