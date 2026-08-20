# UI Contract: Player Status System Line

## Scope

This contract governs the player-facing identity and controller-role status line. It changes presentation only and does not change the player RPC schema, server authority, session persistence, or Overseer UI.

## Visible Contract

For the default first input channel assigned to Nick Valentine with active authority, the rendered line is exactly:

```text
[СИСТЕМА] ВВОД P1 // НИК ВАЛЕНТАЙН // АКТИВЕН
```

Role labels are:

| Authoritative role | Visible label |
|---|---|
| active | `АКТИВЕН` |
| observer | `НАБЛЮДАТЕЛЬ` |
| unassigned | `НЕ НАЗНАЧЕН` |

Default fallback labels matching `PLAYER N` are presented as `PN`. Custom labels remain recognisable and are not replaced by a fabricated player number.

## Stable DOM Identifiers

| Identifier | Contract |
|---|---|
| `playerIdentity` | Single status-line container; hidden when no session projection exists. |
| `playerFallbackName` | Current compact input-channel identity. |
| `playerCharacterName` | Current assigned character; hidden when absent. |
| `roleBadge` | Current role text. The identifier remains stable although badge styling is removed. |

No second hidden legacy identity surface is permitted.

## Accessibility Contract

- `playerIdentity` retains the accessible label `Идентификация игрока`.
- The status surface exposes one current reading order: system prefix, input label, optional character, role.
- Role changes remain available through the existing polite live-status semantics.
- Hidden character segments and their separators must not be announced.
- Colour is not the sole role discriminator; role wording remains explicit.

## Layout Contract

- The status line is a lower-chrome flex item after terminal content/navigation and immediately before `termPrompt` in document order.
- It has no enclosing border, filled panel background, yellow active colour, blue observer colour, or outlined badge.
- It uses the existing green phosphor palette at lower emphasis than the selected menu row.
- It may wrap within its own area on narrow screens but must not create horizontal scrolling.
- Its bounding box must not intersect visible terminal content, navigation controls, specialised hacking controls, or the command prompt.
- When normal terminal content changes, the line remains anchored by layout rather than absolute positioning.

## Verification Contract

- Browser coverage asserts the approved active text, observer and unassigned role text, in-place authority updates, absence of framed/badge presentation, and vertical ordering above the prompt.
- Existing player journeys update their intentional active-role assertion from `АКТИВНЫЙ КОНТРОЛЛЕР` or `АКТИВНЫЙ` to `АКТИВЕН`.
- The player-client production build remains required.
