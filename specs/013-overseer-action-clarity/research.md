# Research: Overseer Action Clarity

## Decision 1: Organize controls by state ownership

**Decision**: Place terminal creation with the saved-terminal list, activation and secondary settings with the selected terminal, publication with the content editor, and take-off-air with live broadcast status.

**Rationale**: The existing layout mixes durable authoring and immediate player effects in one footer/header pair. Contextual placement lets location explain scope before the label is read and directly implements the accepted clarification.

**Alternatives considered**: One grouped action toolbar was rejected because it still requires users to infer scope; putting most actions into per-terminal menus was rejected because publication and broadcast state are not properties of a saved list item alone.

## Decision 2: Keep one primary action for live content publication

**Decision**: When the selected terminal is active, replace the activation control with the `В ЭФИРЕ` status and expose `ОПУБЛИКОВАТЬ ИЗМЕНЕНИЯ` as the only primary update action. Retain the broader existing activation request as `ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ` inside an explained additional menu.

**Rationale**: The live-update path preserves active identity, puzzle, navigation, and runtime command state and therefore matches the common authoring intent. Full reapplication remains available for metadata/settings without competing visually with publication.

**Alternatives considered**: Removing full reapplication entirely would strand existing metadata behavior; keeping it as a neighboring primary button would preserve the ambiguity; renaming it without relocation would not clarify its lower frequency and broader effect.

## Decision 3: Use accessible local dialogs for deliberate actions

**Decision**: Terminal creation opens a labelled dialog with a required non-blank name. Taking a terminal off air always opens a labelled confirmation explaining that the broadcast, players, assignments, saved terminal, and session remain intact.

**Rationale**: Existing Overseer dialogs already establish modal, labelling, cancellation, pending-state, and focus conventions. Creation confirmation prevents accidental autosaved placeholders; take-off-air confirmation protects a live player-facing interruption without changing backend behavior.

**Alternatives considered**: Immediate draft creation was rejected because an accidental click changes durable state; native browser confirmation was rejected because it is harder to test and style and has weaker explicit focus behavior; confirmation only for unfinished progress was rejected by the accepted clarification.

## Decision 4: Chain confirmation with the existing unfinished-progress decision

**Decision**: Confirmation authorizes one clear request. If the coordinator returns decision-required for unfinished progress, close the confirmation and open the existing preserve/discard/cancel dialog; do not issue a second clear request.

**Rationale**: The two dialogs answer different questions: whether to remove the terminal from player view, then what to do with preserved runtime progress. Keeping the coordinator response authoritative avoids duplicating puzzle rules in the UI.

**Alternatives considered**: Combining both decisions into one speculative dialog was rejected because the UI cannot authoritatively know whether a runtime decision is required; silently choosing preserve or discard would violate existing control semantics.

## Decision 5: Preserve all application contracts

**Decision**: Reuse terminal creation through the existing session save and reuse `RequestTerminalActivation`, `UpdateLiveTerminal`, and `RequestTerminalClear` without schema, binding, or backend changes.

**Rationale**: The reported defect is terminology and action placement. Existing commands already implement the required outcomes, so expanding the contract would add risk without user value.

**Alternatives considered**: A generic UI-action dispatcher and new create/confirm desktop methods were rejected because they duplicate existing boundaries and weaken the explicit private service contract.
