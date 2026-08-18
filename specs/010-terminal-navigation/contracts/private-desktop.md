# Приватный desktop-контракт

## Coordination projection

```proto
enum TerminalNavigationDecision {
  TERMINAL_NAVIGATION_DECISION_UNSPECIFIED = 0;
  TERMINAL_NAVIGATION_DECISION_APPROVE = 1;
  TERMINAL_NAVIGATION_DECISION_REJECT = 2;
}

enum TerminalNavigationNoticeReason {
  TERMINAL_NAVIGATION_NOTICE_REASON_UNSPECIFIED = 0;
  TERMINAL_NAVIGATION_NOTICE_REASON_TARGET_MISSING = 1;
  TERMINAL_NAVIGATION_NOTICE_REASON_SELF_TARGET = 2;
  TERMINAL_NAVIGATION_NOTICE_REASON_COMMAND_STALE = 3;
  TERMINAL_NAVIGATION_NOTICE_REASON_TARGET_CHANGED = 4;
}

message PendingTerminalNavigation {
  string request_id = 1;
  string broadcast_id = 2;
  fallout.terminal.player.v1.TerminalNavigationDirection direction = 3;
  string source_terminal_id = 4;
  string source_terminal_name = 5;
  string command_id = 6;
  string command_name = 7;
  string target_terminal_id = 8;
  string target_terminal_name = 9;
  uint32 route_depth = 10;
}

message TerminalNavigationNotice {
  TerminalNavigationNoticeReason reason = 1;
  string source_terminal_id = 2;
  string command_id = 3;
  optional string target_terminal_id = 4;
}

message CoordinationState {
  // existing fields 1..7 unchanged
  optional PendingTerminalNavigation pending_terminal_navigation = 8;
  optional TerminalNavigationNotice terminal_navigation_notice = 9;
}
```

`coordination.proto` переиспользует public enum direction, но private pending не попадает в public descriptor. Existing `coordination-state` named event и `GetRuntimeStatus.coordination_state` доставляют один и тот же current pending/notice для live UI и master reload.

## Resolve operation

```proto
message ResolveTerminalNavigationRequest {
  string request_id = 1;
  TerminalNavigationDecision decision = 2;
}

message ResolveTerminalNavigationResult {
  bool ok = 1;
  optional string error = 2;
  CoordinationState state = 3;
}
```

Узкий Wails-метод:

```text
ResolveTerminalNavigation(ResolveTerminalNavigationRequest) ResolveTerminalNavigationResult
```

Метод регистрируется только на private desktop service. Он отсутствует в `PlayerService`, public descriptors, player assets и public HTTP routes.

## Master dialog flow

1. `frontend/src/master.js` получает `pendingTerminalNavigation` из bootstrap/event и deduplicate-ит dialog по exact `requestId`.
2. Dialog отдельно показывает direction, source terminal, command name и target terminal.
3. Positive action посылает `APPROVE`; negative action, Escape и dialog close посылают `REJECT`.
4. Пока resolve в полёте, buttons disabled; dialog epoch не позволяет stale callback закрыть новый request.
5. Frontend применяет только coordination state с revision новее current; пропущенный event восстанавливается bootstrap-ом.

## Backend resolution

`ResolveTerminalNavigation` проверяет:

- nonblank exact request ID и allowlisted enum decision;
- current broadcast ID и active source terminal;
- для forward: source command с тем же stable ID всё ещё ссылается на тот же target ID;
- для return: top route point всё ещё равна pending copy;
- latest target terminal существует в trusted session catalog и не равен source.

Approve одной coordinator transaction сохраняет source checkpoint, меняет route/active target, очищает pending/notice и публикует одну revision. Он никогда не создаёт `PendingTerminalSwitch` и не открывает preserve/discard dialog.

Reject очищает только exact pending. Stale/duplicate decision возвращает `ok=false` и не меняет current state. Если target/command устарели после создания pending, coordinator очищает pending, сохраняет active/nav/route, ставит typed notice и возвращает safe private error.

## Invalid target notice

`terminal_navigation_notice` используется, когда player выбрал команду с missing/self/stale target или approve обнаружил изменившуюся ссылку. Master frontend отображает localized reason и IDs/name, разрешённые текущей session. Notice не содержит filesystem/storage error и очищается следующим valid transition request, manual switch, end broadcast или shutdown.

## Lifecycle and ordering

- Manual `RequestTerminalActivation`/`RequestTerminalClear` очищает terminal-navigation pending, notice и route перед своим existing switch lifecycle; поздний dialog callback становится stale.
- `EndBroadcast` и `Shutdown` очищают route/pending/notice вместе с broadcast runtimes. `StartBroadcast` начинает с пустыми значениями.
- Coordinator может синхронно читать detached session target по существующему lock order `control → session`; catalog не вызывает coordinator callbacks.
- Effect publication остаётся detached и revision ordered. Master result/event и player updates относятся к одной committed revision.

## Capability verification

- Generated Wails binding inventory содержит ровно один новый allowlisted method и не содержит generic dispatch.
- Public protobuf descriptor не содержит `PendingTerminalNavigation`, `TerminalNavigationDecision`, `ResolveTerminalNavigationRequest`, notice и private coordination fields.
- Private adapters round-trip exact enum/presence/field numbering; generated files не редактируются вручную.
