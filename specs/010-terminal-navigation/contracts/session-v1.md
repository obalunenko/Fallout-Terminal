# Контракт session JSON version 1

## JSON shape

Поле `version` остаётся `1`. На command-узле появляется одно optional known field:

```json
{
  "id": "cmd_open_security",
  "type": "command",
  "name": "Открыть терминал охраны",
  "text": "",
  "terminalTransition": {
    "targetTerminalId": "terminal_security"
  }
}
```

| JSON-путь | Семантика |
|---|---|
| `terminals[].root…terminalTransition.targetTerminalId` | Stable ID другого терминала той же session. |

Нет `terminalTransition` — команда сохраняет прежнюю семантику. Runtime route, pending request, active terminal, navigation и hack checkpoints в session JSON не записываются.

## Persistence protobuf

```proto
message TerminalTransitionConfig {
  string target_terminal_id = 1;
}

message CommandContent {
  string text = 1;
  optional StateChangeConfig state_change = 2;
  optional TerminalTransitionConfig terminal_transition = 3;
}
```

Номера `CommandContent.text = 1` и `state_change = 2` не меняются. Новые generated Go-типы создаются pinned generator; `internal/session/contract.go` явно отображает config в оба направления.

## Validation

- `targetTerminalId` не пуст и проходит те же stable-ID лимиты, что terminal ID.
- Target существует в полной session и не равен source terminal ID.
- Config разрешён только на command leaf; folder/entry с config невалидны.
- `terminalTransition` и `stateChange` не могут присутствовать одновременно.
- Удаление target при сохранившейся inbound-ссылке отклоняет весь candidate document. Master UI до мутации показывает список inbound-команд и не оставляет local session в заведомо невалидном состоянии.

## Compatibility and unknown fields

- Legacy v1 без field открывается без migration и повторно сохраняется без field.
- `nodeFields` включает `terminalTransition`; все остальные unknown node/session/terminal fields по-прежнему round-trip.
- Наличие known config не даёт extras затенить `terminalTransition`.
- Save остаётся атомарным в explicitly selected path и использует existing revision/coalescing pipeline.
- `commandStates` merge не меняется. Completed state-changing command нельзя превратить в transition, пока snapshot не сброшен.

## Authoring contract

- Command editor даёт toggle «ПЕРЕХОД В ДРУГОЙ ТЕРМИНАЛ» и select терминалов текущей session, исключая source.
- Select value — terminal ID, label — текущее terminal name.
- Toggle transition и state-change toggle взаимоисключающи; completed state-change блокирует включение transition до reset.
- Apply не мутирует node до прохождения local validation; backend validation остаётся окончательной.

