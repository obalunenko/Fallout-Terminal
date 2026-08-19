# План реализации: переходы между терминалами

**Feature**: `010-terminal-navigation` | **Date**: 2026-08-18 | **Spec**: [spec.md](./spec.md)

**Bugfix**: 2026-08-19 — BUG-001 Updated from bugfix patch
**Bugfix**: 2026-08-19 — BUG-002 Updated from bugfix patch

## Summary

Функция добавляет к обычной авторской команде необязательную ссылку на другой терминал того же session JSON version 1. Выбор такой команды и возврат из корня создают один server-authoritative approve/reject request; только approve атомарно меняет активный терминал и LIFO-маршрут.

Существующий coordinator остаётся единым владельцем broadcast state, per-terminal checkpoints, revisions, replay protection и stream publication. Переход использует отдельный runtime lifecycle вместо ручного `PendingTerminalSwitch`, поэтому сохраняет исходный checkpoint без второго диалога, повторно проверяет актуальную session перед approve и рассылает всем игрокам одну полную ревизионную проекцию.

По BUG-002 отдельный protocol lifecycle перехода сохраняется, но его ожидающая public projection переиспользует в player UI полноэкранный record-description renderer изменяющей состояние команды: меню скрыто, показан точный текст «Выполняется запрос», а server-authoritative active terminal и маршрут не меняются до решения мастера.

## Project Structure

```text
specs/010-terminal-navigation/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
└── contracts/
    ├── session-v1.md
    ├── public-player.md
    └── private-desktop.md

proto/fallout/terminal/
├── persistence/v1/session.proto       # authored targetTerminalId
├── player/v1/terminal.proto           # route/pending player projection
└── private/v1/
    ├── coordination.proto             # exact master-only pending request
    └── desktop.proto                  # approve/reject request and result

internal/
├── domain/{model.go,json.go,validate.go} # durable and runtime aggregates
├── session/{contract.go,service.go}      # v1 adapter and trusted terminal catalog
├── nav/nav.go                           # moved-folder and ancestor fallback
├── live/service.go                      # checkpoint activation with explicit nav placement
├── control/service.go                   # pending request, LIFO route, atomic resolution
├── player/adapter.go                    # public protobuf boundary
└── gen/fallout/terminal/{persistence,player,private}/v1/
                                                    # generated Go only

main.go                                             # session-catalog composition
app.go                                              # private resolve operation and lifecycle clearing
app_contract.go                                     # explicit private protobuf adapters
desktop_service.go                                  # narrow Wails method

frontend/src/{master.js,master.css,desktop-api.js}  # authoring and master decision dialog
frontend/bindings/                                  # Wails-generated bindings only
client/{client.js,client.css}                       # pending lock and root return control
client/gen/fallout/terminal/player/v1/              # generated ECMAScript only

tests/browser/
├── terminal-navigation.spec.mjs
├── fixture-server/main.go
└── fixtures/desktop-bindings.js
```

**Structure Decision**: Долговечная ссылка остаётся в `domain`/`session`, маршрут и checkpoint-ы — в `control`/`live`, а public/private браузеры получают только свои типизированные protobuf-проекции через существующие узкие границы.

## Constitution Check

| Principle | До исследования | После дизайна | Обоснование |
|---|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | PASS | Wails остаётся в root/private bridge; `domain`, `session`, `nav`, `live`, `control` и `player` не зависят от desktop runtime. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | PASS | Все known persistence, public player и private desktop structures зафиксированы в versioned protobuf до реализации и имеют явные адаптеры. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | PASS | Контроллер по-прежнему отправляет unary `PlayerService.Navigate`; coordinator один меняет route/checkpoints и рассылает revisioned stream updates. |
| IV. Separate Public and Private Capabilities | PASS | PASS | Public projection содержит route depth/top target и secret-free pending; decision ID, full prompt, notice и approve/reject остаются в private desktop service. |
| V. Evolve Schemas Safely and Reproducibly | PASS | PASS | Дизайн добавляет только new fields/messages/enums с `UNSPECIFIED = 0`; published numbers не меняются, generation остаётся pinned. |
| VI. Preserve Portable Session JSON Version 1 | PASS | PASS | JSON `version` остаётся `1`; optional `terminalTransition` имеет legacy default, unknown fields round-trip, runtime route/hack state в файл не попадают. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | PASS | Новый transport или dual runtime не вводится; existing Navigate/Subscribe/coordination paths расширяются целиком, а manual switch остаётся отдельной действующей семантикой. |

Нарушений конституции и оснований для Complexity Tracking нет.

## Implementation Strategy

### Phase 1 — Схемы и чистые доменные правила

1. Добавить additive persistence/public/private protobuf messages и field numbers из [contracts/](./contracts/), затем регенерировать Go/ECMAScript artifacts и schema revision только штатными scripts.
2. Расширить domain models, deep clones, JSON known fields и двухпроходную session validation.
3. Добавить в `internal/nav` pure helper, который находит folder по stable ID во всём дереве, восстанавливает current ancestry и применяет nearest-ancestor/root fallback.

### Phase 2 — Authoring и trusted catalog

1. Обновить explicit session protobuf adapter и clone/round-trip tests, не меняя atomic storage и revision pipeline.
2. Добавить в master command editor target toggle/select, mutual exclusion с `stateChange`, completed-state guard и inbound-reference guard при удалении терминала.
3. Скомпоновать узкий `TerminalCatalog`, который строит detached `TerminalTarget` только из current validated session snapshot.

### Phase 3 — Атомарный coordinator lifecycle

1. Добавить route, pending и notice в broadcast/process aggregate и все clone/snapshot paths.
2. Перехватывать linked `NavigateCommand` и root `NavigateBack` после authorization/replay checks, но до ordinary live action. Создавать только один pending и блокировать competing gameplay.
3. Реализовать exact private resolve: re-read source/target, approve как single commit с checkpoint preservation и explicit nav placement; reject/stale без route/active mutation.
4. Очищать route/pending/notice при manual activation/clear, end broadcast и shutdown. Не изменять existing manual unfinished-hack switch semantics.

### Phase 4 — Границы и UI

1. Обновить public/private adapters, App result, Wails method, desktop API normalization, generated bindings и exact allowlist/descriptor tests.
2. Добавить master dialog с direction/source/command/target, request-ID deduplication, close-as-reject и stale callback guards; typed notice показывать отдельно.
3. Добавить player route/pending mapping, root return control и input lock, сохранив current unary+stream acknowledgement discipline и observer read-only behavior. По BUG-002 прямой pending MUST скрывать меню и переиспользовать общий полноэкранный record-description renderer с точным текстом «Выполняется запрос», не объединяя `TerminalNavigationPresentation` с `CommandExecutionPresentation` и не выполняя optimistic terminal switch.

### Phase 5 — Верификация и cutover

1. Покрыть legacy/new JSON, cross-reference validation, moved/deleted folder restoration, coordinator atomicity/replay/authority, hack continuity, stream reconnect и private capability separation.
2. Добавить Playwright journey master + controller + observers для approve/reject, LIFO/cycle, pending reconnect, stale target и moved/deleted return location. Для BUG-002 отдельно сравнить полноэкранную поверхность прямого pending с renderer ожидающей state-changing команды, проверить точный текст, отсутствие меню, блокировку `Back`/`Enter` и восстановление того же экрана после reconnect.
3. Пройти protobuf format/lint/generation/breaking, Wails binding inventory, Go race, frontend/client builds и browser suite; generated drift и временные protocol paths не оставлять.

## Verification Strategy

| Surface | Ключевые проверки |
|---|---|
| Session/domain | Legacy absence, new config round-trip, unknown fields, forward-reference order, missing/self target, state-change conflict, inbound delete guard. |
| Navigation/live | Rename/move/delete folder, nearest parent/root fallback, destination root placement, solved/unfinished/failed hack retention, explicit reset/discard. |
| Coordinator | Approve/reject/stale, exact-one pending, 20 replays, concurrent distinct IDs, LIFO A→B→C, cycle A→B→A, pop-after-approve, manual/end/shutdown clearing. |
| Player stream | Controller-only mutation, observers read-only, monotonic complete update, pending snapshot/reconnect с тем же полноэкранным экраном «Выполняется запрос», active-terminal convergence без optimistic switch, overflow resubscribe. |
| Private desktop | Exact prompt fields, dialog dedup/close/stale callback, typed notice, one new allowlisted Wails method, no public decision capability. |
| Browser | Master, controller and at least two observers; approve/reject/return, nested source folder, stale target, reconnect while pending, retained hacking; для прямого pending — общий record-description renderer, точный текст «Выполняется запрос», скрытое меню и заблокированные `Back`/`Enter`. |
| Master/native persistence | Реальный `sessions/demo.json`: полный terminal set до и после Wails `SaveSession`, успешная durable revision для `t_demo1` → `t_demo2`, reopen с сохранённой целью; missing/self target по-прежнему отклоняются. |

Applicable commands:

```bash
make proto-generate
make proto-check proto-breaking bindings-check
go test ./internal/domain ./internal/session ./internal/nav
go test -race ./internal/control ./internal/live ./internal/player
go test ./...
npm ci --prefix frontend && npm run build --prefix frontend
npm ci --prefix client && npm run build --prefix client
npm ci --prefix tests/browser
npm test --prefix tests/browser -- terminal-navigation.spec.mjs
make check
```

`make check` не заменяет Playwright journey. Packaging и interactive `go run ./cmd/build dev` проводятся перед acceptance, поскольку изменяются embedded frontend/client assets и Wails bindings.
