# Публичный player-контракт

## Совместимые protobuf-добавления

Существующий ConnectRPC-метод `PlayerService.Navigate` остаётся единственным player-маршрутом выбора команды. `NavigateCommand` не получает признака одобрения: player не обладает такой capability.

```proto
enum CommandExecutionPhase {
  COMMAND_EXECUTION_PHASE_UNSPECIFIED = 0;
  COMMAND_EXECUTION_PHASE_PENDING = 1;
  COMMAND_EXECUTION_PHASE_REJECTED = 2;
}

message CommandExecutionPresentation {
  CommandExecutionPhase phase = 1;
  string command_node_id = 2;
}

message LiveTerminal {
  // existing fields 1..7 unchanged
  optional CommandExecutionPresentation command_execution = 8;
}

enum PlayerNoticeKind {
  PLAYER_NOTICE_KIND_UNSPECIFIED = 0;
  PLAYER_NOTICE_KIND_COMMAND_PERSISTENCE_FAILED = 1;
}

message PlayerNotice {
  PlayerNoticeKind kind = 1;
}

message PlayerState {
  // existing fields 1..8 unchanged
  optional PlayerNotice notice = 9;
}
```

Добавления не меняют существующие fields, service methods и enum values. Presence `command_execution` отличает обычную terminal presentation от временного общего экрана. Авторский confirmation prompt, master request ID, durable `commandStates`, session document и file path отсутствуют в public descriptor.

## Эффективная проекция дерева

| Durable состояние | `ContentNode.name` | `ContentCommand.text` |
|---|---|---|
| Обычная команда | авторское исходное имя | прежний текст |
| Stateful, исходная | авторское исходное имя | не используется как показанный результат |
| Stateful, выполненная | frozen completed name | frozen result text |

До approve будущие completed name/result не отображаются. Наличие state-change конфигурации не требует передавать private prompt игроку: выбор определяется сервером по node ID.

## Поток выбора и ожидания

1. Active controller выбирает строку и отправляет обычный `NavigateRequest` с `command.node_id`.
2. Сервер выполняет существующие session/broadcast/terminal/controller/fingerprint проверки.
3. Ordinary или completed-команда следует прежнему navigation path.
4. Initial stateful-команда создаёт один server request и возвращает accepted revision, которая публикует `COMMAND_EXECUTION_PHASE_PENDING` всем назначенным игрокам.
5. Client отображает ровно «Выполняется запрос» вместо terminal navigation. Пока phase pending, UI не инициирует navigation/command actions; сервер независимо отклоняет такие запросы как `ACTION_REASON_CONFLICT`.
6. Повторные или разные concurrent request ID при текущем pending не создают второй master request и не меняют accepted state.

## Решение мастера

### Approve и durable success

Одна более новая `CompoundUpdate` очищает `command_execution`, содержит effective completed tree и `NavigationState.command_node_id` выбранной команды. Все players одновременно переходят с ожидания на frozen result; строка меню уже имеет frozen completed name. Player не отправляет второй RPC.

### Reject или close dialog

Одна более новая update меняет phase на `COMMAND_EXECUTION_PHASE_REJECTED`; все players показывают ровно «Запрос отклонён». Только active controller может отправить существующий Back/navigation action для возврата в сохранённый menu path; accepted update очищает presentation у всех.

### Persistence failure после approve

Pending очищается, команда остаётся initial, а shared terminal возвращается в прежний menu path. Текущий controller получает безопасный `PlayerNotice` без storage details; master получает private error result. Completed name/result не публикуются. Notice очищается следующей принятой player action либо новой snapshot policy, определённой adapter tests.

## Lifecycle и reconnect

- Disconnect инициатора не меняет pending. Оставшиеся players продолжают видеть ожидание, а reconnect получает pending в первом полном snapshot.
- End broadcast, terminal switch и shutdown очищают pending/rejected без выполнения и persistence. После нового broadcast они не восстанавливаются.
- Если approve уже прошёл durable boundary до disconnect/shutdown, completed snapshot остаётся в session и появится при следующей активации.
- Client не применяет optimistic name/result и не сохраняет request/state в browser storage; local storage остаётся ограниченным recognition handle.

## Авторизация, ordering и ошибки

- Только active controller может создать запрос обычным `Navigate`; observers/unassigned сохраняют прежние typed rejections.
- Только private desktop capability может решить request; в player service нет approve/reject метода или поля.
- Accepted unary revision и следующая streamed update остаются монотонными. Master decision имеет собственную последующую coordinator revision.
- Replay того же Navigate request ID использует текущую fingerprint cache; разные IDs дополнительно дедуплицируются каноническим single-pending invariant.
- Existing origin, message size, deterministic fingerprint и public-access protections сохраняются.
