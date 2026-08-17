# План реализации: команды, изменяющие состояние пунктов меню

**Feature**: `009-state-changing-menu-commands` | **Date**: 2026-08-17 | **Spec**: [spec.md](./spec.md)

## Summary

Функция добавляет однонаправленные команды: выбор активного игрока создаёт общий заблокированный запрос, а выполнение начинается только после решения мастера в приватном диалоге. Pending/rejected состояния принадлежат текущему broadcast и публикуются всем игрокам, тогда как только одобренные и успешно записанные frozen name/result становятся долговечным миром по стабильным ID терминала и команды в session JSON version 1. Существующий player `Navigate` создаёт запрос, новый private desktop command разрешает его, а approve проходит общий session pipeline до публикации успеха.

## Project Structure

```text
specs/009-state-changing-menu-commands/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
└── contracts/
    ├── session-v1.md
    ├── public-player.md
    └── private-desktop.md

proto/fallout/terminal/
├── persistence/v1/session.proto        # Авторская настройка и durable commandStates
├── player/v1/terminal.proto            # Публичные pending/rejected presentation phases
├── player/v1/player.proto              # Персональная безопасная ошибка persistence
└── private/v1/
    ├── coordination.proto              # Pending request и master decision enum
    ├── desktop.proto                   # Resolve/reset requests и results
    └── runtime.proto                   # Приватный bootstrap/session-state event

internal/
├── domain/
│   ├── model.go                        # StateChangeConfig и CommandExecutionState
│   ├── json.go                         # Известные v1 JSON-поля и unknown-field preservation
│   ├── validate.go
│   └── *_test.go
├── session/
│   ├── contract.go                     # Явное domain ↔ persistence protobuf отображение
│   ├── service.go                      # Общая ревизия, ID-мутации и stale-save merge
│   ├── storage.go                      # Существующая атомарная замена файла
│   └── *_test.go
├── control/
│   ├── service.go                      # Single pending, private resolve, persist-before-success
│   └── service_test.go
├── live/
│   ├── service.go                      # Эффективное дерево и повторный просмотр
│   └── service_test.go
├── nav/
│   ├── nav.go
│   └── nav_test.go
├── player/
│   ├── adapter.go                      # Pending/rejected/effective public projection
│   ├── handler.go
│   ├── stream.go
│   └── *_test.go
└── gen/fallout/terminal/{persistence,player,private}/v1/
                                         # Только сгенерированные Go contracts

main.go                                  # Композиция session store перед coordinator
app.go                                   # Приватные reset-команды и session-state emission
app_contract.go                          # Private protobuf adapters
desktop_service.go                       # Узкий Wails façade
wails_host.go                            # Регистрация типизированного события

frontend/
├── src/
│   ├── master.js                       # Редактор, master approval dialog и reset UI
│   ├── master.css
│   └── desktop-api.js                  # Reset façade и revision-aware session-state subscription
└── bindings/                           # Только Wails-generated bindings

client/
├── client.js                            # Общие pending/rejected screens и action blocking
└── gen/fallout/terminal/player/v1/      # Только сгенерированные ECMAScript contracts

tests/browser/
├── fixture-server/main.go
├── connectrpc-player.spec.mjs
├── player-sessions-control.spec.mjs
├── desktop-api.spec.mjs
└── fixtures/desktop-bindings.js

internal/testutil/testdata/session-v1.json
sessions/demo.json
proto/schema-revision.txt
proto/compatibility-baseline.binpb
scripts/wails-bindings-check.sh
```

**Structure Decision**: Доменные правила и долговечное состояние остаются в `internal/domain`/`internal/session`, сериализованный переход и публикация — в `internal/control`, public/private protobuf-адаптеры — на соответствующих границах, а оба браузерных интерфейса только отображают авторитетные проекции и отправляют узкие команды.

## Constitution Check

| Principle | До исследования | После дизайна | Обоснование |
|---|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | PASS | Wails остаётся только в root composition/private bridge; domain, session, control и player не зависят от Wails. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | PASS | Все новые известные persistence, public player и private desktop/event структуры сначала определяются в protobuf и проходят явные адаптеры. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | PASS | Browser отправляет unary `Navigate`; сервер сохраняет и только затем рассылает полную ревизионную проекцию через существующий stream. |
| IV. Separate Public and Private Capabilities | PASS | PASS | Player получает только pending/rejected phase и effective command; master prompt/decision, reset, документ и storage details остаются в private Wails boundary. |
| V. Evolve Schemas Safely and Reproducibly | PASS | PASS | Используются только добавочные поля/messages/enum value; прежние номера неизменны, generated code обновляется штатным pinned toolchain. |
| VI. Preserve Portable Session JSON Version 1 | PASS | PASS | Version остаётся 1, отсутствие полей задаёт прежние defaults, unknown fields сохраняются, запись остаётся атомарной в выбранный файл. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | PASS | Новый параллельный протокол не вводится: существующие Save/Navigate/Subscribe пути расширяются, временные обходы не остаются. |

Нарушений конституции и оснований для Complexity Tracking нет.

## Technical Context

**Runtime**: Go 1.26.6; browser JavaScript modules с Node.js 20.19+; Wails v3.0.0-beta.8.

**Contracts**: Protobuf v1 packages, ConnectRPC unary/stream service, приватные Wails bindings/events, явный адаптер session JSON version 1.

**Storage**: Выбранный пользователем JSON-файл с атомарной same-directory temp-write/fsync/rename заменой; одна упорядоченная очередь document revisions.

**Scale**: Один desktop process, один активный broadcast, до 1000 терминалов и 100000 узлов по существующим лимитам; критерии функции требуют одного контроллера, минимум двух наблюдателей и стресс до 100 конкурентных/повторных запросов.

**Performance target**: При нормальном соединении контроллер и наблюдатели сходятся на новом названии и результате не позднее одной секунды; persistence остаётся синхронным барьером принятия, а не фоновой best-effort записью.

**Dependencies**: Новых runtime или test dependencies нет.

## Contract and State Design

### Persistent JSON

- `ContentNode.stateChange` присутствует только у stateful-команды и содержит `completedName`/`confirmationText`; существующие `name`/`text` задают исходное название и успешный результат.
- `Terminal.commandStates[commandID]` содержит замороженные `completedName`/`resultText` и вместе с `Terminal.id` образует устойчивую идентичность.
- Pending request и rejection presentation в JSON не попадают и отменяются при завершении текущего broadcast/terminal context.
- Legacy v1 без полей остаётся валидным и исходным; unknown fields продолжают round-trip.
- Полный author Save не владеет `commandStates`: session service сохраняет канонические снимки для существующих ID и очищает только удалённые команды.
- Выполнение и reset получают новую document revision из того же владельца и завершаются успехом только после атомарной записи.

Подробный shape и protobuf mapping: [contracts/session-v1.md](./contracts/session-v1.md).

### Public ConnectRPC

- Обычный `NavigateCommand{node_id}` для initial stateful-команды создаёт ровно один server-owned pending request; player не получает поля approve/confirm.
- `LiveTerminal.command_execution` с phase pending/rejected обеспечивает одинаковый общий экран и reconnect snapshot без раскрытия приватного prompt.
- Пока pending, UI и сервер блокируют shared navigation/commands. Disconnect инициатора не отменяет запрос.
- Approve success очищает pending и публикует effective frozen tree/result одной следующей revision; reject показывает общий rejection screen до возврата контроллера.
- Storage failure возвращает терминал в прежнее menu state, отправляет безопасную персональную ошибку контроллеру и не публикует completed name/result.
- Уже выполненная команда остаётся selectable; повтор показывает frozen result без dialog и записи.

Полная семантика: [contracts/public-player.md](./contracts/public-player.md).

### Private Wails Bridge

- `CoordinationState.pending_command_execution` переносит master-only request ID, terminal/command identity и авторский prompt как в bootstrap, так и в existing `coordination-state` event.
- `ResolveCommandExecution` принимает только exact current request ID и enum approve/reject; закрытие/cancel диалога отправляет reject, stale callback является no-op/error.
- `ResetCommandState` и `ResetTerminalCommandStates` являются отдельными allowlisted desktop methods; frontend выполняет confirm, backend повторно валидирует ID.
- `session-state` несёт канонический session document и document revision только master frontend после durable mutation.
- Frontend дедуплицирует approval dialog по request ID и принимает только более новые coordination/document revisions.

Полная семантика: [contracts/private-desktop.md](./contracts/private-desktop.md).

### Runtime-State Lifecycle

1. При открытии session service загружает optional configs/states; отсутствие означает исходное состояние.
2. Активация терминала гидратирует runtime из backend session и строит effective tree, не доверяя server-owned полям frontend payload.
3. Initial player selection создаёт runtime pending, публикует ожидание players и private coordination request мастеру без session write.
4. Reject/close очищает pending и публикует rejection; Back возвращает прежний menu path. Disconnect инициатора pending сохраняет.
5. Approve проходит exact request validation и durable store gate; после save одна coordinator revision доставляет всем одинаковые effective name/result/navigation.
6. Reconnect player получает pending/rejected/completed в первом snapshot; reload master получает pending в coordination bootstrap.
7. End broadcast, terminal switch и shutdown отменяют pending/rejected без сохранения; durable `commandStates` остаются и восстанавливаются при активации.
8. Reset изменяет durable state тем же save-before-publication порядком; shutdown дренирует уже достигшие store boundary writes.

### Security and Capability Boundaries

- Выполнение разрешено только текущему connected controller текущего broadcast/terminal; observers остаются read-only.
- Player `Navigate` только запрашивает выполнение; approve/reject существует исключительно в private desktop service.
- Public projection не раскрывает master prompt, request ID, будущий результат, полный session document, file path или внутреннюю storage error.
- Resolve/reset доступны только через явно зарегистрированный private desktop service; public descriptor test продолжает запрещать desktop capabilities.
- Существующие origin, request-size, fingerprint и public-access protections остаются без ослабления.

## Implementation Phases

### Phase 0: Research and Decisions

- Зафиксировать разделение authored `stateChange` и server-owned `commandStates`.
- Зафиксировать единого владельца document revision и merge-защиту от stale whole-document Save.
- Переиспользовать `Navigate` для создания запроса, existing coordination snapshot/event для master prompt и отдельный private resolve command.
- Не добавлять зависимости или новую JSON-версию.

Результат: [research.md](./research.md).

### Phase 1: Schemas and Pure Domain Rules

1. Добавить совместимые persistence/player/private protobuf-поля для config/snapshot, pending/rejected presentation, private pending/decision и session event без изменения опубликованных номеров.
2. Обновить schema revision, compatibility baseline и generated Go/ECMAScript artifacts только генератором.
3. Добавить domain types, deep-clone/JSON known-field mapping и валидацию config/state/reference invariants.
4. Покрыть legacy v1 defaults, новые round-trips, unknown extras, invalid variants и frozen snapshots.

Независимый срез: открыть, сохранить и снова открыть настроенную невыполненную команду; обычные команды и старый fixture остаются прежними.

### Phase 2: Ordered Persistence and Authoring Safety

1. Перенести allocation document revision в session service для всех видов записи.
2. Добавить ID-адресованные execute/reset-one/reset-terminal mutations в общий epoch/revision worker.
3. Реализовать merge server-owned snapshots при полнодокументном Save и pruning удалённых command ID.
4. Проверить write/rename failure, racing autosave, concurrent revision ordering, delete/rename/move и reopen.

Независимый срез: backend mutation переживает повторное открытие файла и не стирается параллельным устаревшим Save.

### Phase 3: Server-Authoritative Execution and Projection

1. Внедрить в coordinator узкий `CommandStateStore` с односторонним lock order `control → session` и без callback в coordinator.
2. Распознавать ordinary, initial-stateful и completed команды; для initial создавать один pending request без persistence.
3. Добавить private resolve: reject/close публикует rejection, approve сохраняет snapshot до completed publication.
4. Строить effective public tree и отдельную pending/rejected presentation без утечки master prompt.
5. Обрабатывать повторные/concurrent selections, controller disconnect и stale master decisions без второго dialog/write.
6. Гидратировать activation/reconnect из backend session/runtime и типизировать persistence failure для master/controller.

Независимый срез: ConnectRPC controller создаёт запрос, master approve выполняет команду один раз, два observers и reconnect получают pending и frozen result; ordinary path не вызывает request/store.

### Phase 4: Master and Player Interfaces

1. Добавить в master property editor state-change config, четыре обязательных текста, frozen state и field errors.
2. Показывать один approval dialog на pending request ID; approve/reject/close вызывать private resolve и игнорировать stale callback после lifecycle cancel.
3. Добавить reset-one/reset-terminal, bindings и revision-aware `session-state` subscription.
4. Добавить в player client общие «Выполняется запрос»/«Запрос отклонён», action blocking, Back и безопасную controller-only persistence error.
5. Не сохранять pending/command state в browser storage и не делать optimistic completed transition.

Независимый срез: player выбирает, все ожидают, master approve/reject/close разрешает запрос, а все интерфейсы сходятся только по авторитетным ревизиям.

### Phase 5: Integration, Compatibility and Regression Gates

1. Расширить fixture server и Playwright journeys для controller + минимум двух observers, master approve/reject/close, repeat, disconnect/reconnect, navigation, broadcast stop/switch/start.
2. Проверить полный app reopen на том же session file и frozen-config edit/reset/re-execute.
3. Проверить 100 конкурентных/повторных запросов и 100 persistence failures детерминированными fakes/race tests.
4. Выполнить protobuf compatibility/drift, Wails binding allowlist, Go race/vet/test и оба frontend build/test контура.

## Verification Plan

| Поверхность | Автоматическая проверка | Ключевое доказательство |
|---|---|---|
| Domain/JSON | `go test ./internal/domain ./internal/session` | Legacy v1 defaults; config/state round-trip; unknown fields; invalid orphan state; frozen snapshots |
| Atomic persistence | Focused session tests with storage fakes | Ошибка write/sync/rename оставляет старый файл/revision; stale Save не стирает состояние; delete prune атомарен |
| Coordinator concurrency | `go test -race ./internal/control ./internal/live ./internal/player` | Один pending/dialog; disconnect сохраняет его; approve пишет один раз; reject/lifecycle cancel не пишут; ordinary path прежний |
| Protobuf governance | `scripts/proto-generate.sh --sync-revision`, `scripts/proto-check.sh`, `scripts/proto-breaking.sh --all-fixtures`, `scripts/proto-drift-test.sh` | Добавления совместимы, generated artifacts детерминированы и не редактировались вручную |
| Wails boundary | `scripts/wails-bindings-check.sh` и app/contract tests | Только два reset-метода и одно private event добавлены; public descriptor не содержит desktop capability |
| Go repository | `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...` | Формат, статическая проверка и все регрессии проходят |
| Master frontend | `npm ci --prefix frontend`, `npm test --prefix frontend`, `npm run build --prefix frontend` | Валидация полей, один dialog/request ID, approve/reject/close, stale callback, reset и revision ordering |
| Player/browser | `npm ci --prefix tests/browser`, `npm test --prefix tests/browser` | Controller + 2 observers видят pending/rejected/result ≤1 s; actions блокируются; disconnect/reconnect/lifecycle корректны |
| Desktop lifecycle | Целевой app integration test и ручной smoke того же файла | End broadcast, terminal switch, app close/open сохраняют выполненное имя; reset активного терминала синхронен |

Если signed packaging или внешний public provider не затрагиваются изменениями, соответствующие credential-dependent проверки отмечаются `N/A`, а не заявляются выполненными.

## Implementation Risks and Controls

| Риск | Контроль |
|---|---|
| Stale JS autosave перезаписывает player mutation | Единый revision owner + server-owned merge по стабильным ID + race test с gated writer |
| Control/session deadlock | Только `control → session`; store не вызывает coordinator/effect router; тесты под race detector |
| Успех опубликован до durability | Синхронный store gate внутри candidate transaction; failure сохраняет byte-identical canonical state/revision/effects |
| Player получает или подделывает master approval | Public schema не содержит decision capability; resolve существует только в private Wails service |
| Потерянное coordination event не открывает dialog | Pending входит в runtime bootstrap snapshot; frontend дедуплицирует по request ID |
| Поздний dialog callback выполняет отменённый запрос | Exact current request/broadcast/terminal validation превращает stale decision в no-op/error |
| Config edit меняет уже показанный результат | Публичная проекция читает frozen snapshot до явного reset |
| Master frontend остаётся на старом документе | Private `session-state` с монотонной document revision и session-side merge как защита очереди |
| Состояние теряется при activation/reconnect | Runtime гидратируется из backend session; snapshot строится из effective tree |
| Публичная схема раскрывает private state | Public contract содержит только effective value/prompt; descriptor separation test расширяется |
