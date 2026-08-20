# Data Model: Immersive Controller Status Log

This feature introduces no persisted or transported entity. It reshapes one derived browser view from the existing authoritative player projection.

## Derived View: PlayerStatusLine

| Field | Source | Rules |
|---|---|---|
| `inputLabel` | Session fallback name | Required while session state exists. `PLAYER N` becomes `PN`; another non-empty label remains recognisable. |
| `characterName` | Assigned character name | Optional. Omit the character segment and its preceding separator when no character is assigned. |
| `role` | Authoritative session role | One of active, observer, or unassigned as supplied by the existing player projection. |
| `roleLabel` | Derived from `role` | Active → `АКТИВЕН`; observer → `НАБЛЮДАТЕЛЬ`; unassigned → `НЕ НАЗНАЧЕН`. |
| `visible` | Session readiness | Visible when authoritative session state is available; hidden while no current session projection exists. |

## Rendered Shape

With an assigned character:

```text
[СИСТЕМА] ВВОД <inputLabel> // <characterName> // <roleLabel>
```

Without an assigned character:

```text
[СИСТЕМА] ВВОД <inputLabel> // <roleLabel>
```

The terminal presentation renders the line in uppercase while the canonical character and custom session names remain unchanged in authoritative state.

## Validation Rules

- `inputLabel` must never be empty when the line is visible.
- Separator tokens appear only between populated segments.
- The role label must reflect the current authoritative role and must not be inferred from local interaction state.
- The view must not retain a previous character or role after a newer projection removes or changes it.
- Long labels remain contained within the lower terminal chrome and cannot create horizontal page overflow.

## State Transitions

| Prior state | Event | Result |
|---|---|---|
| No session projection | Session projection received | Show the line using current input, character, and role fields. |
| Unassigned | Character assigned | Add the character segment; retain the same input label and current role. |
| Active | Authority transferred away | Replace `АКТИВЕН` with `НАБЛЮДАТЕЛЬ` or `НЕ НАЗНАЧЕН` from the new projection. |
| Observer | Authority transferred in | Replace the role label with `АКТИВЕН` in place. |
| Assigned | Character removed | Remove the character segment and its adjacent separator without leaving empty space. |
| Any visible state | Session projection unavailable | Hide the line until authoritative state returns. |
