# Контракт session JSON version 1

**Bugfix**: 2026-08-19 — BUG-004 Уточнён общий `CommandContent.oneof behavior` без изменения JSON v1 или field numbers.

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

~~Предыдущий контракт представлял оба взаимоисключающих config как независимые `optional` fields; BUG-004 заменяет только эту generated-type семантику.~~

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

Исправленный контракт:

```proto
message CommandContent {
  string text = 1;
  oneof behavior {
    StateChangeConfig state_change = 2;
    TerminalTransitionConfig terminal_transition = 3;
  }
}
```

Номера `CommandContent.text = 1`, `state_change = 2` и `terminal_transition = 3` не меняются. JSON names остаются `stateChange` и `terminalTransition`, а unset `behavior` означает ordinary command. Новые generated Go-типы создаются pinned generator; `internal/session/contract.go` явно отображает ровно один variant в оба направления.

## Validation

- `targetTerminalId` не пуст и проходит те же stable-ID лимиты, что terminal ID.
- Target существует в полной session и не равен source terminal ID.
- Config разрешён только на command leaf; folder/entry с config невалидны.
- Общий protobuf `oneof behavior` структурно не позволяет `terminal_transition` и `state_change` присутствовать одновременно; JSON/import validation продолжает отклонять malformed input с обоими полями.
- Удаление target при сохранившейся inbound-ссылке отклоняет весь candidate document. Master UI до мутации показывает список inbound-команд и не оставляет local session в заведомо невалидном состоянии.

## Compatibility and unknown fields

- Legacy v1 без field открывается без migration и повторно сохраняется без field.
- Валидные legacy v1 ordinary/state-change/transition документы сохраняют wire field numbers и JSON shape; изменение generated API на shared oneof намеренно и покрывается descriptor/adapter tests.
- `nodeFields` включает `terminalTransition`; все остальные unknown node/session/terminal fields по-прежнему round-trip.
- Наличие known config не даёт extras затенить `terminalTransition`.
- Save остаётся атомарным в explicitly selected path и использует existing revision/coalescing pipeline.
- `commandStates` merge не меняется. Completed state-changing command нельзя превратить в transition, пока snapshot не сброшен.

## Authoring contract

- ~~Command editor даёт отдельные checkbox toggle «ИЗМЕНЯЕТ СОСТОЯНИЕ» и «ПЕРЕХОД В ДРУГОЙ ТЕРМИНАЛ», которые программно снимают друг друга.~~ По BUG-004 editor даёт один command-mode selector с вариантами ordinary, state-change и terminal-transition; transition mode показывает select терминалов текущей session, исключая source.
- Select value — terminal ID, label — текущее terminal name.
- Selector допускает ровно один mode, скрывает и удаляет inactive config; completed state-change блокирует смену mode до reset.
- Apply не мутирует node до прохождения local validation; backend validation остаётся окончательной.
