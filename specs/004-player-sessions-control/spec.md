# Feature Specification: Player Sessions, Character Assignment, and Shared Terminal Control

**Feature directory**: `004-player-sessions-control`  
**Scope**: Process-local browser sessions, broadcast-scoped character assignments, exclusive player control, observer behavior, game-master coordination, and active-terminal switching

## User Scenarios & Testing

### User Story 1 - Join as a Character (Priority: P1)

As a player opening the shared link, I select one available game-master-defined character and enter the current broadcast with a stable identity, so the table knows whom my device represents.

**Why this priority**: Character assignment is the eligibility boundary for every other player-facing capability in this feature.

**Independent Test**: Start a broadcast with a two-character roster, connect one new browser profile, select a character, and verify that the same character becomes claimed and the player proceeds to the current terminal.

**Acceptance Scenarios**:

1. **Given** a browser profile is unknown to the running server, **when** it opens the shared player link, **then** the server establishes one logical session with a unique fallback name.
2. **Given** the logical session has no valid assignment for the current broadcast, **when** the player connects, **then** an immersive terminal-styled selection screen shows the current roster with available and claimed characters distinguishable.
3. **Given** an available character, **when** the player selects it, **then** the server claims it for that session and the player proceeds to the active terminal or the waiting state when no terminal is active.
4. **Given** two sessions select the same available character concurrently, **when** the claims are ordered, **then** exactly one succeeds and the other remains on selection with the updated roster.
5. **Given** two unassigned sessions select different characters concurrently while no controller exists, **when** both claims complete, **then** exactly one becomes active controller and the other becomes an observer.
6. **Given** all roster characters are claimed, **when** another unassigned session connects, **then** it remains on selection or waiting for game-master intervention and cannot enter the terminal.

---

### User Story 2 - Control One Shared Terminal (Priority: P1)

As the active character, I can operate the terminal while every other assigned character observes the same authoritative screen, so one physical-table participant leads without splitting the shared experience.

**Why this priority**: Exclusive control is the central gameplay value and security boundary of the feature.

**Independent Test**: Connect one active session and two observers, exercise every existing player action from each, and verify that only active-session actions can change the canonical terminal.

**Acceptance Scenarios**:

1. **Given** a connected, character-assigned active session, **when** it submits a valid player terminal action, **then** the action is evaluated under all existing terminal and hacking rules and its authoritative result is shown to every connected assigned session.
2. **Given** an observer, **when** it submits any navigation, password, filler, or special-pattern action, **then** the action is rejected without changing canonical state, attempts, randomness, logs, or outcome.
3. **Given** an unassigned, unknown, expired, stale, or invalid session, **when** it submits any player action, **then** the action is rejected without canonical mutation.
4. **Given** an observer is viewing the shared terminal, **when** it hovers, focuses, or uses a client-local control, **then** passive local feedback may change but canonical state does not.
5. **Given** the active session submits an action, **when** the result is pending, **then** shared player input remains pending until an authoritative outcome is applied and no optimistic shared mutation appears.
6. **Given** a player surface or crafted player request, **when** it attempts to invoke `ForceHackSuccess`, **then** no player operation is available and the game-master operation remains private.

---

### User Story 3 - Reuse One Device Session (Priority: P1)

As a player, I can refresh, reconnect, reopen the browser, or use multiple tabs from the same browser profile without losing my logical identity, character, or current control status during their defined lifetimes.

**Why this priority**: Reliable same-device continuity prevents ordinary browser behavior from disrupting a live tabletop session.

**Independent Test**: Assign a character to one browser profile, open multiple tabs, disconnect and reconnect them in varying order, and verify that one logical session and one character claim remain.

**Acceptance Scenarios**:

1. **Given** a recognized browser profile reconnects during the same server process and broadcast, **when** its session is still valid, **then** it resumes the same fallback name, character assignment, and control status without selecting again.
2. **Given** several tabs share one recognized browser identity, **when** all are open, **then** they represent one logical session and receive the same assignment, control status, and canonical terminal state.
3. **Given** several tabs share one logical session, **when** one tab closes, **then** the session remains connected while at least one other tab is open and its character stays claimed.
4. **Given** the last connection for an observer session closes, **when** presence is updated, **then** the session becomes disconnected and retains its character claim.
5. **Given** browser storage is cleared or a different profile or private context opens the link, **when** the server cannot recognize the prior identity, **then** a new logical session is established and the old character claim remains reserved.
6. **Given** a browser presents an identifier from a previous server process, **when** it connects after restart, **then** the identifier restores no old name, assignment, presence, or control state.

---

### User Story 4 - Manage Roster and Assignments (Priority: P1)

As the game master, I can maintain the character roster and correct, release, assign, or move character claims without changing terminal or puzzle progress.

**Why this priority**: Tabletop setup mistakes and device changes must be recoverable without restarting play.

**Independent Test**: Add and rename characters, create claims, release and transfer them between sessions, and compare terminal and puzzle state before and after every operation.

**Acceptance Scenarios**:

1. **Given** a live server process, **when** the game master adds or renames a roster character, **then** all affected master and player views update and existing logical sessions and terminal state remain unchanged.
2. **Given** an unclaimed roster character, **when** the game master assigns it to an unassigned session, **then** the claim becomes exclusive and the session becomes eligible for terminal control.
3. **Given** a claimed character, **when** deletion is attempted, **then** deletion is refused until the claim is released or transferred.
4. **Given** a connected session with a character, **when** the game master releases the claim, **then** the character becomes available and that session returns to character selection without terminal or puzzle mutation.
5. **Given** a character is moved from one session to another, **when** the move completes, **then** the old session loses it, the new session receives it, and neither terminal state nor puzzle progress changes.
6. **Given** two tabs of one session submit different character selections concurrently, **when** the requests are ordered, **then** the session receives no more than one assignment.
7. **Given** a player has completed selection, **when** the player attempts to choose another character independently, **then** the existing assignment remains until the game master changes it.

---

### User Story 5 - Reassign Terminal Control (Priority: P1)

As the game master, I can make a connected, character-assigned observer the active controller at any time, so control follows the table's decisions without disturbing play.

**Why this priority**: Explicit reassignment is required for turn-taking and recovery from an unavailable controller.

**Independent Test**: Reassign control between two connected characters during navigation and during an unfinished puzzle, then verify authorization and state continuity from every tab.

**Acceptance Scenarios**:

1. **Given** one active session and one connected assigned observer, **when** the game master selects the observer as controller, **then** the observer becomes active, the former controller becomes an observer, and every tab receives the new status.
2. **Given** a controller reassignment, **when** it completes, **then** character claims, current terminal, navigation, puzzle generation, attempts, board, patterns, logs, and outcome remain unchanged.
3. **Given** an action and reassignment race, **when** the action is ordered before reassignment, **then** it may complete under the former controller's authority.
4. **Given** an action and reassignment race, **when** the action is ordered after reassignment, **then** the former controller's action is rejected without mutation.
5. **Given** no eligible controller exists, **when** an assigned observer remains connected, **then** the terminal stays read-only until the game master assigns control or a later eligible first assignment establishes it.
6. **Given** an active character claim is released or moved away without explicit controller reassignment, **when** the assignment change completes, **then** control is cleared and no existing observer is promoted automatically.

---

### User Story 6 - Handle Controller Disconnects (Priority: P1)

As a group, we keep the same shared state when the active player's connection drops, while the game master decides whether to wait or hand control to someone else.

**Why this priority**: Network and device interruptions must not elect an unintended controller or alter a puzzle.

**Independent Test**: Disconnect an active multi-tab session, reconnect it before and after game-master reassignment, and verify control and puzzle continuity.

**Acceptance Scenarios**:

1. **Given** the active session loses its last open connection, **when** it becomes disconnected, **then** it retains its character and active designation and no observer is promoted.
2. **Given** the disconnected active session has not been replaced, **when** it reconnects, **then** it automatically resumes active control.
3. **Given** the game master reassigned control while the former controller was disconnected, **when** the former controller reconnects, **then** it returns as an observer with its original character.
4. **Given** any session disconnects or reconnects, **when** presence changes, **then** puzzle generation, attempts, candidates, removed duds, pattern state, logs, navigation, and outcome remain unchanged.
5. **Given** the active session is disconnected, **when** the game master views session status, **then** that session remains visibly identified as the disconnected active session until it reconnects or control is reassigned.

---

### User Story 7 - Follow the Active Terminal (Priority: P1)

As a connected player, I automatically follow whichever configured terminal the game master presents, without reopening the link or selecting my character again.

**Why this priority**: A broadcast may span several terminals while retaining the same table participants and controller.

**Independent Test**: With active and observer sessions connected, switch among configured terminals and verify automatic transition, stable assignments, and identical terminal state.

**Acceptance Scenarios**:

1. **Given** assigned sessions are connected, **when** the game master activates another terminal, **then** every player transitions through the existing loading presentation to the same newly active terminal.
2. **Given** a terminal switch completes, **when** player status is compared, **then** logical sessions, character assignments, active controller, and observer statuses are unchanged.
3. **Given** no terminal is active during a broadcast, **when** assigned sessions are connected, **then** they retain identity and assignment while seeing an immersive waiting state.
4. **Given** a session joins after a terminal is active, **when** it completes character selection, **then** it joins the currently active terminal's canonical state.
5. **Given** a terminal is inactive, **when** a player sends an action intended for it, **then** the action cannot change that terminal or the active terminal.

---

### User Story 8 - Decide an Unfinished Puzzle's Fate (Priority: P1)

As the game master, I explicitly preserve, discard, or keep playing an unfinished puzzle before switching terminals, so progress is never silently lost or altered.

**Why this priority**: Terminal switching introduces a destructive boundary for runtime puzzle progress that requires an explicit table decision.

**Independent Test**: Attempt to switch away from an unfinished puzzle and exercise preserve, discard, and cancel, then return to the original terminal where applicable.

**Acceptance Scenarios**:

1. **Given** the active terminal has an unfinished puzzle, **when** the game master requests another terminal, **then** the switch pauses and offers preserve, discard, or cancel.
2. **Given** preserve is chosen, **when** the old terminal becomes inactive, **then** its puzzle is suspended and cannot receive player actions.
3. **Given** a preserved puzzle, **when** its terminal is activated again, **then** the same board, attempts, removed duds, pattern state, progress log, and outcome are restored.
4. **Given** discard is chosen, **when** the terminal is later activated and hacking begins, **then** a fresh puzzle is created under the existing generation rules.
5. **Given** cancel is chosen, **when** the decision completes, **then** the original terminal and puzzle remain active and unchanged.
6. **Given** the active terminal has no unfinished puzzle, **when** another terminal is activated, **then** no unfinished-puzzle decision is required.

---

### User Story 9 - End and Restart Broadcast Lifetimes (Priority: P2)

As the game master, I can end one live broadcast and start another with clean character and control assignments while recognized devices remain convenient within the same server process.

**Why this priority**: Clear lifetime boundaries prevent stale ownership from leaking into a later tabletop scene.

**Independent Test**: Assign characters and control, end the broadcast, start another without restarting, and then repeat after a server restart.

**Acceptance Scenarios**:

1. **Given** a broadcast has assigned characters and an active controller, **when** the game master ends it, **then** the active terminal is removed from player control and all claims and controller assignment are cleared.
2. **Given** the broadcast ended without a server restart, **when** connected devices remain or reconnect, **then** their logical sessions and fallback names remain recognized but their former characters do not.
3. **Given** a new broadcast starts in the same process, **when** recognized sessions join, **then** every session must select a character again and the first eligible completed assignment becomes the initial controller.
4. **Given** the server application restarts, **when** players reconnect, **then** all prior logical sessions, fallback-name changes, presence, character claims, and controller assignment are gone.
5. **Given** a broadcast ends or the server restarts, **when** durable terminal data is inspected, **then** configured terminals and existing unlocked-terminal behavior remain unchanged.

## Edge Cases

- An empty roster leaves every unassigned session on the immersive selection or waiting surface without terminal access.
- Duplicate character names may be confusing but do not merge distinct roster identities or claims.
- A rename arriving while selection is open updates the same roster entry and does not invalidate a valid claim already being processed.
- Releasing a disconnected session's claim makes the character available while leaving the old logical session recognized.
- A stale selection from an earlier broadcast cannot claim a character in the current broadcast.
- A stale action from a former controller cannot mutate state after reassignment.
- Closing one of several tabs cannot mark the shared logical session disconnected.
- Refreshing the active browser cannot trigger observer promotion.
- Clearing local browser state cannot release the old session's claim.
- A session with an invalidated assignment returns to selection even if another tab still displays the terminal.
- Moving the active character without an explicit control reassignment clears control rather than transferring it implicitly.
- Switching terminals while the unfinished-puzzle decision is pending leaves the current terminal authoritative and actionable only according to the existing game-master flow.
- Rapid duplicate one-use actions from one or several tabs can produce no more than one accepted mutation.
- If an authoritative result rejects an action, pending input still resolves without requiring an animation timeout.
- The private game-master application remains operational when no player is connected, assigned, or active.

## Requirements

### Functional Requirements

#### Logical sessions and presence

- **FR-001**: The system MUST resolve every player connection to exactly one server-recognized logical session.
- **FR-002**: A logical session MUST have an opaque stable identity for the lifetime of the current server process.
- **FR-003**: A newly established logical session MUST receive a unique automatically generated fallback display name.
- **FR-004**: The game master MUST be able to rename a logical session's fallback display name without changing its identity, character assignment, control status, or terminal state.
- **FR-005**: Reopening, refreshing, reconnecting, navigating away and returning, or reopening the browser from the same recognized browser profile MUST reuse the same logical session while the server process remains active.
- **FR-006**: Multiple tabs sharing one recognized browser identity MUST represent one logical session.
- **FR-007**: A logical session MUST be connected while at least one of its player connections remains open.
- **FR-008**: Closing one of several connections for a logical session MUST NOT disconnect the session or release its character.
- **FR-009**: A different browser, browser profile, private-browsing context, cleared local identity, or otherwise unrecognized device context MUST establish a separate logical session.
- **FR-010**: Establishing a new logical session MUST NOT release any claim held by an older disconnected session.

#### Broadcast and assignment lifetimes

- **FR-011**: Logical session identity and fallback-name changes MUST remain process-local and MUST expire when the server process restarts.
- **FR-012**: Character assignments MUST belong to the current live broadcast rather than to an individual configured terminal.
- **FR-013**: Switching active terminals, starting or finishing a puzzle, and disconnecting or reconnecting MUST NOT clear a valid current-broadcast character assignment.
- **FR-014**: Ending a broadcast MUST clear every character claim and the active-controller assignment.
- **FR-015**: Ending a broadcast MUST retain recognized logical sessions until server restart.
- **FR-016**: Starting a new broadcast MUST require every logical session to obtain a new character assignment.
- **FR-017**: A session identifier issued by a previous server process MUST restore none of its previous name, assignment, presence, or control state.
- **FR-018**: Runtime session, fallback-name, presence, character-claim, and controller data MUST NOT be added to the persisted version-1 terminal/session schema.

#### Character roster and claims

- **FR-019**: The game master MUST be able to define a roster of player characters before or during a live broadcast.
- **FR-020**: Each roster entry MUST have a stable identity for the current server process, a player-facing name, and an available-or-claimed state.
- **FR-021**: The game master MUST be able to add and rename roster entries without mutating logical sessions, controller status, or terminal state.
- **FR-022**: Renaming a roster entry MUST update every affected game-master and player view without creating a new assignment.
- **FR-023**: The system MUST refuse to delete a claimed roster entry until its claim is released or transferred.
- **FR-024**: One logical session MUST have no more than one character assignment during a broadcast.
- **FR-025**: One roster character MUST be claimed by no more than one logical session during a broadcast.
- **FR-026**: Character availability and claim decisions MUST be authoritative at the server when each selection is processed.
- **FR-027**: Concurrent claims for one character MUST result in exactly one successful claim.
- **FR-028**: Concurrent different selections from one logical session MUST result in no more than one character assignment.
- **FR-029**: A rejected claimant MUST remain unassigned and receive the current roster state.
- **FR-030**: A disconnected session's character MUST remain claimed until release, transfer, broadcast end, or server restart.
- **FR-031**: A player MUST NOT independently replace a completed character assignment.
- **FR-032**: The game master MUST be able to assign an available character to an unassigned logical session.
- **FR-033**: Releasing a character MUST make it available and return its connected former session to character selection without canonical terminal or puzzle mutation.
- **FR-034**: Moving a character MUST remove it from the old session and assign it to the new session as one authoritative operation.
- **FR-035**: Character release, correction, and transfer MUST NOT mutate terminal navigation, puzzle state, logs, attempts, randomness, or outcome.
- **FR-036**: Player-facing roster state MUST distinguish available from claimed characters without exposing private connection information.

#### Exclusive controller assignment

- **FR-037**: No more than one character-assigned logical session MUST be designated active controller at any time.
- **FR-038**: The first eligible character assignment processed while the broadcast has no established controller MUST atomically designate exactly one initial active controller.
- **FR-039**: Character-assigned sessions completing selection while a controller exists MUST begin as observers.
- **FR-040**: Raw connection order MUST NOT establish controller status before successful character assignment.
- **FR-041**: Concurrent first-time assignments MUST establish exactly one active controller.
- **FR-042**: Controller assignment MUST apply globally across the live broadcast and all configured terminals.
- **FR-043**: The game master MUST be able to designate a connected, character-assigned observer as the active controller.
- **FR-044**: Reassignment MUST make the selected session active and the previous active session an observer as one authoritative change.
- **FR-045**: Controller reassignment MUST preserve character assignments and all canonical terminal and puzzle state.
- **FR-046**: Releasing or moving the active session's character without explicit reassignment MUST clear controller assignment and MUST NOT promote an observer.
- **FR-047**: When no eligible controller is designated, player terminal actions MUST remain read-only until game-master reassignment or a later eligible initial assignment establishes control.

#### Disconnect behavior

- **FR-048**: Disconnecting the active session MUST retain its active designation and character claim.
- **FR-049**: Disconnecting the active session MUST NOT automatically promote or elect another session.
- **FR-050**: Reconnecting the unchanged active session before reassignment MUST restore its ability to control the terminal.
- **FR-051**: Reconnecting a former controller after game-master reassignment MUST restore it as an observer with its existing character.
- **FR-052**: Disconnecting or reconnecting any session MUST NOT mutate terminal or puzzle state.

#### Player authorization and shared state

- **FR-053**: A player action MUST be eligible for canonical processing only when its connection resolves to a current logical session with a valid current-broadcast character assignment.
- **FR-054**: A terminal-mutating player action MUST be eligible only when the owning logical session is the active controller at processing time and is currently connected.
- **FR-055**: Controller authorization MUST be enforced authoritatively for every player action rather than relying on disabled player controls.
- **FR-056**: Observer, unassigned, unknown, expired, invalid, and stale-controller actions MUST leave all canonical state unchanged.
- **FR-057**: Rejected player actions MUST NOT consume attempts, activate patterns, advance randomness, navigate content, alter logs, trigger outcomes, or otherwise mutate puzzle or terminal state.
- **FR-058**: An unassigned session MUST NOT submit or cause terminal actions before character selection completes.
- **FR-059**: The system MUST accept existing player-side navigation, menu, password-candidate, filler-character, special-pattern, and other terminal actions for canonical processing only from the active controller.
- **FR-060**: Observer hover and focus feedback MUST remain local without submitting a shared action or changing canonical state.
- **FR-061**: Any control available to an observer MUST be limited to client-local behavior that does not affect shared state.
- **FR-062**: The player surface MUST present observer controls as visibly read-only within the existing terminal aesthetic.
- **FR-063**: The player surface MUST expose no operation that invokes `ForceHackSuccess`.
- **FR-064**: Active-controller status MUST NOT grant access to any private game-master operation.
- **FR-065**: Accepted active-controller actions MUST continue to follow all existing password, likeness, attempt, special-pattern, dud-removal, restoration, lockout, navigation, and content rules.

#### Ordering, pending input, and convergence

- **FR-066**: Character claims, player actions, controller changes, roster changes, and active-terminal changes MUST have one unambiguous authoritative order.
- **FR-067**: An action ordered before controller reassignment MUST be evaluated using the former controller's authority at that processing point.
- **FR-068**: An action ordered after controller reassignment MUST be rejected when submitted by the former controller.
- **FR-069**: A race between action processing and reassignment MUST NOT produce duplicate or unauthorized canonical mutation.
- **FR-070**: After submitting a shared action, player input MUST enter a pending state until an authoritative outcome is applied.
- **FR-071**: Player clients MUST NOT optimistically mutate canonical shared state.
- **FR-072**: The authoritative outcome of an accepted or rejected action MUST end its pending state without relying only on an arbitrary animation delay.
- **FR-073**: Rapid duplicate submissions for a one-use action MUST produce no more than one accepted mutation.
- **FR-074**: Every connected assigned session MUST receive accepted terminal navigation, hacking state, logs, loading transitions, and outcomes from the same canonical state.
- **FR-075**: All tabs belonging to one logical session MUST receive the same assignment and controller status changes.

#### Active terminal and unfinished puzzles

- **FR-076**: The game master MUST determine which single configured terminal, if any, is currently presented to players.
- **FR-077**: Activating another terminal MUST transition every connected player automatically through the existing terminal loading presentation.
- **FR-078**: Active-terminal switching MUST preserve logical sessions, character assignments, and controller assignment.
- **FR-079**: A newly assigned player MUST join the terminal currently selected by the game master.
- **FR-080**: When no terminal is active during a broadcast, assigned sessions MUST retain identity and assignment while seeing an immersive waiting state.
- **FR-081**: An inactive terminal MUST reject player actions until it becomes active again.
- **FR-082**: Requesting a switch away from an unfinished puzzle MUST pause the switch for an explicit preserve, discard, or cancel decision by the game master.
- **FR-083**: Choosing preserve MUST suspend the unfinished puzzle in process-local state and make it unable to receive player actions while inactive.
- **FR-084**: Reactivating a terminal with a preserved puzzle MUST restore its exact board, remaining attempts, removed duds, available and used patterns, progress log, and outcome.
- **FR-085**: Choosing discard MUST cause the terminal's next hacking attempt to create a fresh puzzle under the existing generation rules.
- **FR-086**: Choosing cancel MUST leave the current terminal and unfinished puzzle active and unchanged.
- **FR-087**: The system MUST NOT silently discard, solve, fail, restart, or otherwise alter an unfinished puzzle during a terminal-switch request.

#### Game-master coordination and scope boundaries

- **FR-088**: The private game-master application MUST show every currently connected logical session with its fallback name, character assignment if any, presence, and controller-or-observer status.
- **FR-089**: The private game-master application MUST keep a disconnected active session visibly identified until it reconnects or control is reassigned.
- **FR-090**: The game-master session view MUST show both fallback session name and character name when needed to resolve device or assignment problems.
- **FR-091**: Character name MUST remain the primary player-facing identity after assignment, while fallback session name remains a separate technical label.
- **FR-092**: Ending a broadcast MUST remove the active terminal from player control and return connected clients to an immersive waiting or selection state.
- **FR-093**: Ending a broadcast MUST NOT delete configured terminals or silently alter durable unlocked-terminal state.
- **FR-094**: Server restart MUST discard logical sessions, fallback-name changes, presence, character claims, controller assignment, and any active runtime puzzle allowed to expire by the existing persistence boundary.
- **FR-095**: The game-master application MUST retain its existing trusted operations regardless of player connection, assignment, or controller state.
- **FR-096**: This feature MUST NOT redesign existing terminal or hacking logs to add historical character ownership.
- **FR-097**: This feature MUST NOT add accounts, authentication, persistent player profiles, individual invitation links, unassigned spectators, multiple simultaneous controllers, or automatic controller election after disconnect.
- **FR-098**: This feature MUST NOT import or manage character sheets, attributes, skills, perks, eligibility, rules tests, inventory, or campaign history.
- **FR-099**: This feature MUST NOT change existing password-guessing, likeness, attempt, special-pattern, dud-removal, attempt-restoration, lockout, terminal-content, or game-master success rules.

## Key Entities

- **Logical Connection Session**: A temporary identity for one recognized device and browser profile during one server process. It has an opaque identity, unique fallback name, aggregate connected presence, optional current-broadcast character assignment, and controller-or-observer status. It is not an account or campaign profile.
- **Browser Connection**: One currently open player connection belonging to a logical session. Several connections may belong to the same session, and aggregate presence remains connected until the final one closes.
- **Character Roster Entry**: A game-master-defined, player-facing character identity available within the current server process. It has a stable process-local identity, a mutable display name, and current available-or-claimed state, but no character-sheet data.
- **Character Assignment**: The exclusive relationship between one logical session and one roster entry for the current live broadcast. It is independent from controller status.
- **Live Terminal Broadcast**: The runtime period in which the game master may present one configured terminal at a time to connected players. It owns the current character-assignment lifetime and controller assignment.
- **Active Controller Assignment**: The exclusive designation of at most one character-assigned logical session as authorized to submit player terminal actions for the current broadcast.
- **Observer Session**: Any character-assigned logical session that is not the active controller. It mirrors canonical state and may use only passive or client-local interactions.
- **Active Terminal**: The single configured terminal currently presented to players, or no terminal while assigned players wait.
- **Suspended Puzzle**: An unfinished process-local puzzle preserved for an inactive terminal, unavailable for actions until that terminal is active again.
- **Authoritative Action Outcome**: The accepted or rejected result that determines whether canonical state changes and releases the player's pending input state.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Across at least 100 concurrent two-session trials for one available character, every trial produces exactly one successful claim and one rejection.
- **SC-002**: Across at least 100 concurrent first-assignment trials with different characters, every trial finishes with exactly one active controller and all other assigned sessions as observers.
- **SC-003**: A suite covering refresh, reopen, transient disconnect, navigation away and back, and at least three simultaneous tabs produces one logical session and one character claim for the same recognized browser profile.
- **SC-004**: Reconnecting a recognized session during the same broadcast restores its character without showing selection again in every tested reconnect path.
- **SC-005**: A different browser profile, private context, or cleared local identity creates a distinct logical session in every tested case.
- **SC-006**: A claimed character remains unavailable through at least 100 disconnect and competing-claim trials until the game master releases or transfers it.
- **SC-007**: One hundred observer attempts spanning every player action category produce zero canonical navigation, puzzle, attempt, randomness, log, or outcome mutations.
- **SC-008**: Only a connected, character-assigned active session produces accepted player mutations in authorization tests covering active, observer, unassigned, disconnected, stale, unknown, and expired sessions.
- **SC-009**: Disconnecting the active session produces zero automatic observer promotions across all tested single-tab and multi-tab disconnect sequences.
- **SC-010**: Reconnecting the unchanged active session restores control in every trial, while reconnecting after reassignment restores observer status in every trial.
- **SC-011**: Game-master reassignment changes authorization and all affected tab statuses while producing zero changes to assignments, terminal navigation, board, attempts, patterns, logs, or outcome.
- **SC-012**: Character rename, session rename, claim correction, release, and transfer each produce zero terminal or puzzle state changes in before-and-after comparisons.
- **SC-013**: At least 100 deliberately interleaved action-and-reassignment trials follow one authoritative order and produce no duplicate or unauthorized mutation.
- **SC-014**: All connected player views converge on identical canonical terminal state after every accepted action in multi-client navigation and hacking scenarios.
- **SC-015**: Every connected client follows at least ten active-terminal switches without reconnecting or selecting its character again.
- **SC-016**: A preserved unfinished puzzle returns with an exact match for board, attempts, removed duds, pattern state, progress log, and outcome after switching away and back.
- **SC-017**: Preserve, discard, and cancel tests show no silent puzzle solve, failure, restart, or loss during terminal-switch decisions.
- **SC-018**: Ending a broadcast clears all character and controller assignments while retaining all recognized logical sessions until restart.
- **SC-019**: Starting a second broadcast requires new character selection from every recognized session, and restarting the server restores no prior logical session or claim.
- **SC-020**: Persistence comparison before and after the feature shows no logical-session, fallback-name, presence, character-claim, or controller data in the version-1 saved schema.
- **SC-021**: Player-surface and crafted-player-action checks find zero path to `ForceHackSuccess`, while the existing game-master operation remains available.
- **SC-022**: Existing regression suites for password guesses, likeness, attempts, special patterns, dud removal, attempt restoration, lockout, terminal content, and game-master forced success pass unchanged.

## Assumptions

- The existing shared terminal, server-authoritative navigation, hacking puzzle, special-pattern behavior, player presentation, loading animation, and private game-master operation are stable dependencies of this feature.
- A browser identity can be recognized across ordinary reopen and reconnect behavior within a server process; the mechanism is deliberately deferred to planning.
- The roster definition is process-local, remains available across broadcast endings within that process, and is cleared on server restart; claims are still cleared at every broadcast end.
- When control has been cleared, the next newly completed eligible character assignment may establish control, but already assigned observers are never promoted automatically.
- Moving a character away from the active session without an explicit simultaneous controller reassignment clears control and does not transfer control with the character.
- Duplicate character display names are permitted because stable roster identity, not the visible name, determines claims; the game master is responsible for choosing distinguishable names.
- A rejected player action yields an authoritative outcome sufficient to clear local pending input even though its exact transport representation is deferred to planning.
- Computers, laptops, and tablets are supported; mobile-phone-specific layout work is outside this feature.
- The shared player link and existing network exposure model remain unchanged; this feature adds no access-control guarantee for the link itself.

## Scope Boundaries

This feature excludes user accounts and passwords, persistent player or campaign profiles, character-sheet and rules automation, hacking eligibility, individual invitation links, additional internet-access controls, unassigned spectators, simultaneous multi-session control, automatic controller election after disconnect, persistence of sessions or claims beyond their stated lifetimes, historical action attribution, per-terminal controller assignments, mobile-phone-specific presentation, localization, audio-system work, and any visual redesign.

## Verbatim Constraints

- The trusted game-master operation name MUST remain exactly `ForceHackSuccess`.
