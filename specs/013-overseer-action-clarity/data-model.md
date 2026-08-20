# Data Model: Overseer Action Clarity

This feature changes no persisted or transported data model. It clarifies how existing terminal and broadcast state is projected into UI actions and introduces only ephemeral dialog state.

## Existing entities

### Saved terminal

- **Identity**: Existing stable terminal ID.
- **Authored fields**: Name, hacking level, introduction, content tree, and durable command-state snapshots.
- **Relationship**: Belongs to one saved session; may be selected in the editor; may also be the active terminal.
- **Change**: None to shape or persistence. Creation now requires a non-blank name before this entity is appended.

### Selected terminal

- **Identity**: The saved terminal whose ID matches the editor selection.
- **Relationship**: Exactly zero or one selected terminal per open Overseer session.
- **Change**: Its contextual header owns activation and the additional settings menu.

### Active terminal

- **Identity**: The saved terminal ID referenced by current broadcast coordination state.
- **Relationship**: A broadcast has zero or one active terminal; active and selected may differ.
- **Change**: None to runtime semantics. The UI exposes status, publication, reapplication, and take-off-air according to this relationship.

### Broadcast

- **Identity**: Existing process-local broadcast epoch.
- **Relationship**: Retains connected sessions, character assignments, and controller authority with or without an active terminal.
- **Change**: None. Taking a terminal off air clears only its active-terminal reference.

## Derived action availability

| UI condition | Create terminal | Make active | Publish changes | Reapply settings | Take off air |
|---|---:|---:|---:|---:|---:|
| Session open, no broadcast | Enabled | Disabled | Hidden | Hidden | Hidden |
| Broadcast active, selected terminal inactive | Enabled | Enabled | Hidden | Hidden | Visible only if some other terminal is active |
| Broadcast active, selected terminal active | Enabled | Replaced by `В ЭФИРЕ` | Enabled | Available in additional menu | Enabled in broadcast panel |
| Any live command pending | Enabled unless its own save is pending | Disabled | Disabled | Disabled | Disabled |
| No selected terminal | Enabled | Disabled | Hidden | Hidden | Determined solely by broadcast active-terminal state |

The take-off-air action belongs to broadcast state, so it remains available when another terminal is selected in the editor.

## Ephemeral dialog states

### Create-terminal dialog

Fields:

- `open`: whether the modal is visible.
- `name`: user-entered terminal name, trimmed at confirmation.
- `error`: blank-name validation feedback.
- `submitting`: prevents duplicate confirmation.

Transitions:

1. **Closed → Open**: `+ СОЗДАТЬ ТЕРМИНАЛ`; clear previous name/error and focus the name input.
2. **Open → Open**: blank confirmation; show validation error and retain focus.
3. **Open → Closed**: cancel or Escape; create nothing and restore focus to the creation button.
4. **Open → Closed**: valid confirmation; append one draft, select it, autosave once, render it, then focus its editor context.

### Take-off-air confirmation

Fields:

- `open`: whether the modal is visible.
- `pending`: whether one clear command is in flight.
- `error`: safe command failure shown inside the dialog.

Transitions:

1. **Closed → Open**: `СНЯТЬ С ЭФИРА`; explain preserved state and focus cancel.
2. **Open → Closed**: cancel or Escape; issue no command and restore focus to take-off-air.
3. **Open → Pending**: confirm; disable both actions and issue exactly one existing clear request.
4. **Pending → Open**: rejection/failure; retain active state, show error, re-enable actions.
5. **Pending → Closed**: clear succeeds; render no active terminal and focus the surviving broadcast control.
6. **Pending → Existing decision dialog**: coordinator reports unfinished progress; close confirmation and open the existing preserve/discard/cancel decision without another clear request.

## Validation rules

- Terminal names are trimmed and must not be blank; existing length and content compatibility remain unchanged.
- Creation cancellation and validation failure must not mutate or autosave the session.
- Live action availability derives from the latest authoritative coordination revision, not from optimistic local toggles.
- A pending action prevents duplicate desktop calls and contradictory success/error messages.
- No ephemeral dialog state enters session JSON, player payloads, Wails DTOs, or browser storage.
