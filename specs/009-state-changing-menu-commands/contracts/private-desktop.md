# Приватный desktop-контракт

## Назначение

Master frontend получает pending-запрос через существующий `CoordinationState`, показывает диалог с авторским prompt и решает его одним новым allowlisted методом. Два reset-метода и revision-aware session event обслуживают долговечное состояние. Ни одна из этих capabilities не регистрируется в публичном ConnectRPC service.

## Protobuf-сообщения

```proto
enum CommandExecutionDecision {
  COMMAND_EXECUTION_DECISION_UNSPECIFIED = 0;
  COMMAND_EXECUTION_DECISION_APPROVE = 1;
  COMMAND_EXECUTION_DECISION_REJECT = 2;
}

message PendingCommandExecution {
  string request_id = 1;
  string broadcast_id = 2;
  string terminal_id = 3;
  string command_id = 4;
  string command_name = 5;
  string confirmation_text = 6;
}

message ResolveCommandExecutionRequest {
  string request_id = 1;
  CommandExecutionDecision decision = 2;
}

message ResolveCommandExecutionResult {
  bool ok = 1;
  optional string error = 2;
  CoordinationState state = 3;
}

message CoordinationState {
  // existing fields 1..6 unchanged
  optional PendingCommandExecution pending_command_execution = 7;
}

message ResetCommandStateRequest {
  string terminal_id = 1;
  string command_id = 2;
}

message ResetTerminalCommandStatesRequest {
  string terminal_id = 1;
}

message SessionStateResult {
  bool ok = 1;
  optional string error = 2;
  uint64 revision = 3;
  fallout.terminal.persistence.v1.Session session = 4;
}

message SessionStateEvent {
  uint64 revision = 1;
  fallout.terminal.persistence.v1.Session session = 2;
}
```

Имена Wails-методов:

- `ResolveCommandExecution(ResolveCommandExecutionRequest) ResolveCommandExecutionResult`;
- `ResetCommandState(ResetCommandStateRequest) SessionStateResult`;
- `ResetTerminalCommandStates(ResetTerminalCommandStatesRequest) SessionStateResult`.

Существующее приватное событие `coordination-state` переносит pending-запрос. Новое событие `session-state` переносит канонический durable document после successful execute/reset.

## Master approval flow

Producer pending — coordinator после принятого player Navigate; consumer — `frontend/src/desktop-api.js`/`master.js` через существующий coordination snapshot/event.

1. Frontend сравнивает `pending_command_execution.request_id` с последним показанным request и открывает ровно один диалог с `confirmation_text`.
2. Positive action вызывает `ResolveCommandExecution` с `APPROVE`; cancel, close или отрицательная action — с `REJECT`.
3. Backend проверяет enum presence, точный текущий request ID и актуальные broadcast/terminal/command IDs. Stale/duplicate decision возвращает typed private error/no-op и ничего не меняет.
4. Dialog result не считается выполнением. При approve success приходит более новый coordination state без pending и durable session event; при reject — coordination state без pending и player rejection presentation.
5. Reload master frontend получает pending из `GetRuntimeStatus.coordination_state` даже если исходное event было пропущено и вновь открывает тот же request, не создавая новый.

### Approve failure

Если session mutation не записана атомарно, `ResolveCommandExecutionResult.ok=false`, pending закрывается, completed snapshot отсутствует, master показывает безопасную ошибку. Внутренний file path/error chain не попадает ни в reusable status, ни к players.

### Lifecycle cancellation

End broadcast, terminal switch и App shutdown очищают pending coordinator state без вызова session store. Frontend закрывает/игнорирует dialog, если более новая coordination revision больше не содержит тот же request ID. Поздний dialog callback получает stale request rejection. Disconnect инициировавшего controller не очищает pending.

## Reset-one

Producer — master frontend после отдельного `window.confirm`; consumer — `desktopService`/`App`, затем coordinator/session mutation service.

- Frontend передаёт terminal ID и command ID из открытой канонической session.
- Backend проверяет открытый файл, оба ID, принадлежность команды и state-change config.
- Success атомарно удаляет ровно один snapshot. Уже initial state — idempotent no-op без write.
- Для active terminal successful change обновляет player projection одной live revision.
- Ошибка не меняет session, runtime или player stream.

## Reset-terminal

- Backend проверяет terminal ID текущей открытой session.
- Одна atomic mutation удаляет все и только `commandStates` этого terminal.
- Отсутствие snapshots — idempotent no-op без write.
- Для active terminal публикуется одно полное update, не последовательность изменений.

## `session-state` event

Producer — App/effect router после успешной долговечной ID-адресованной мутации; consumer — desktop API/master state.

- Event содержит полный канонический session document и document revision, но не file path.
- Publication происходит только после atomic write.
- Frontend принимает только revision новее последней применённой, затем заменяет рабочий document и save status.
- Open-session/runtime bootstrap остаётся начальным источником. Session-side merge независимо защищает уже queued stale autosaves.

## Ошибки и capability separation

Master получает стабильный пользовательский error; diagnostic cause остаётся в backend chain. Методы добавляются в explicit Wails binding allowlist. Descriptor tests доказывают отсутствие `PendingCommandExecution`, `ResolveCommandExecution`, reset/session document и dialog capabilities в public player service. Player public projection содержит только pending/rejected phase и command node ID.

## Ordering и shutdown

Coordination revision сериализует create/resolve/cancel request. Document revision сериализует только durable execute/reset/save. Session write не удерживает callback в coordinator: lock order остаётся `control → session`, а store никогда не вызывает effect router. После начала shutdown новые player/private decisions отклоняются; уже достигший durability approve дренируется по текущим session worker rules.
