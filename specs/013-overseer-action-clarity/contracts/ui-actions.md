# UI Contract: Overseer Terminal Actions

This contract defines the visible and testable Overseer action surface. It introduces no new application or network API.

## Canonical visible copy

The following accepted strings are exact:

- `+ СОЗДАТЬ ТЕРМИНАЛ`
- `СДЕЛАТЬ АКТИВНЫМ`
- `ОПУБЛИКОВАТЬ ИЗМЕНЕНИЯ`
- `СНЯТЬ С ЭФИРА`
- `ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ`
- `В ЭФИРЕ`

Superseded primary labels `+ НОВЫЙ ТЕРМИНАЛ`, `ОБНОВИТЬ АКТИВНЫЙ`, `ОБНОВИТЬ У ИГРОКОВ`, and `УБРАТЬ АКТИВНЫЙ ТЕРМИНАЛ` must not remain on the active action surface.

## Context ownership

| Context | Required surface | Existing command boundary |
|---|---|---|
| Saved-terminal list | `+ СОЗДАТЬ ТЕРМИНАЛ` | Existing session autosave after confirmed local creation |
| Selected inactive terminal header | `СДЕЛАТЬ АКТИВНЫМ` | `RequestTerminalActivation` with the complete selected target |
| Selected active terminal header | `В ЭФИРЕ` and an additional menu | No command for status; menu owns reapplication |
| Content editor header | `ОПУБЛИКОВАТЬ ИЗМЕНЕНИЯ` | `UpdateLiveTerminal` with tree and introduction |
| Additional selected-terminal menu | `ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ` plus effect explanation | Existing `RequestTerminalActivation` for the already-active terminal |
| Current broadcast panel | `СНЯТЬ С ЭФИРА` | `RequestTerminalClear` only after confirmation |
| Saved-terminal row | Existing terminal deletion action | Existing durable deletion flow; never reused for live clearing |

## Stable and new UI identifiers

Stable identifiers remain where their command mapping is unchanged:

- `btnAddTerminal`
- `btnMakeLive`
- `liveFlag`
- `btnPublish`
- `btnStopBroadcast`

New local interaction identifiers:

- `createTerminalDialog`
- `createTerminalName`
- `createTerminalError`
- `btnCancelCreateTerminal`
- `btnConfirmCreateTerminal`
- `terminalSettingsMenu`
- `btnReapplySettings`
- `takeOffAirDialog`
- `takeOffAirError`
- `btnCancelTakeOffAir`
- `btnConfirmTakeOffAir`

These identifiers are private UI/test hooks, not Wails or player contracts.

## Create-terminal dialog

- Accessible name/title: `СОЗДАТЬ ТЕРМИНАЛ`.
- Required field label: `НАЗВАНИЕ ТЕРМИНАЛА`.
- Actions: `ОТМЕНА` and `СОЗДАТЬ ТЕРМИНАЛ`.
- Blank-name error: `УКАЖИТЕ НАЗВАНИЕ ТЕРМИНАЛА`.
- Open focuses the name field; cancel/Escape creates nothing and returns focus to `btnAddTerminal`.
- Confirm trims the name, creates exactly one default empty terminal, selects it, and initiates one autosave.

## Additional settings menu

- The menu is available only when selected and active terminal IDs match.
- Its disclosure is labelled `ДОПОЛНИТЕЛЬНО`.
- It explains that reapplication sends complete selected-terminal settings and may clear an unfinished request; it is broader than publishing content.
- `ПЕРЕПРИМЕНИТЬ НАСТРОЙКИ` invokes the existing activation command once and then closes the menu.
- When the selected terminal is inactive, the same header position exposes `СДЕЛАТЬ АКТИВНЫМ` instead of the active status/menu combination.

## Publication

- `ОПУБЛИКОВАТЬ ИЗМЕНЕНИЯ` appears only when selected and active IDs match.
- It invokes `UpdateLiveTerminal` with the authored tree and introduction, never the broader activation call.
- Pending disables publication and all competing live actions.
- Success uses the existing coordination status region and temporary positive button acknowledgement; failure leaves authoritative live state unchanged and exposes an alert.

## Take-off-air confirmation

- Accessible name/title: `СНЯТЬ ТЕРМИНАЛ С ЭФИРА?`.
- Description: `Игроки перестанут видеть активный терминал. Трансляция, подключения, роли, назначения и сохранённый терминал останутся без изменений.`
- Actions: `ОТМЕНА` and `СНЯТЬ С ЭФИРА`.
- Opening the dialog makes no desktop call and focuses cancel.
- Escape is equivalent to cancel.
- Confirm invokes `RequestTerminalClear` exactly once and disables both dialog actions while pending.
- On a normal clear, the dialog closes, no terminal remains active, and focus moves to the surviving broadcast control.
- On command failure, the dialog remains open with `takeOffAirError`, both actions re-enable, and the active terminal remains visible.
- On decision-required, the dialog closes and the existing unfinished-progress decision dialog opens using the returned switch ID; the UI does not issue another clear request.

## Accessibility and responsive behavior

- Every dialog uses native modal semantics, explicit `aria-labelledby`/`aria-describedby`, and an alert region for validation or command errors.
- Keyboard users can open, submit, cancel, and leave each interaction without losing focus to a hidden control.
- Additional-menu content is reachable and operable by keyboard.
- At narrow widths, action groups may wrap vertically but must remain adjacent to their owning list, editor, selected-terminal, or broadcast context.

## Compatibility boundary

- No new Wails methods, events, protobuf messages, session fields, player messages, routes, or generated bindings.
- Existing backend validation and authoritative coordination revisions remain decisive.
- Existing player reconnection and multi-tab convergence behavior remains unchanged.
