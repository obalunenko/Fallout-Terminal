# Задачи: команды, изменяющие состояние пунктов меню

**Input**: [spec.md](./spec.md), [plan.md](./plan.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: Конституция и критерии SC-001–SC-015 требуют автоматических domain, persistence, concurrency, contract и multi-client browser проверок. В каждой user-story фазе тестовые задачи выполняются до реализации и должны сначала зафиксировать ожидаемое падение.

## Phase 1: Setup — контрактные источники и генерация

**Wave 1 — independent (different files):**

- [x] **T001** [P] Добавить в persistence v1 protobuf необязательную state-change конфигурацию команды и terminal `command_states`, сохранив номера существующих полей · `proto/fallout/terminal/persistence/v1/session.proto`
- [x] **T002** [P] Добавить в public player protobuf pending/rejected presentation и безопасное controller notice без master prompt или decision capability · `proto/fallout/terminal/player/v1/terminal.proto`, `proto/fallout/terminal/player/v1/player.proto`
- [x] **T003** [P] Добавить в private protobuf pending request, approve/reject decision, resolve/reset DTO и session-state event · `proto/fallout/terminal/private/v1/coordination.proto`, `proto/fallout/terminal/private/v1/desktop.proto`, `proto/fallout/terminal/private/v1/runtime.proto`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — generation join:**

- [x] **T004** Перегенерировать Go/ECMAScript protobuf artifacts, синхронизировать schema revision и проверить additive compatibility baseline без ручного редактирования generated files · `internal/gen/fallout/terminal/`, `client/gen/fallout/terminal/player/v1/`, `proto/schema-revision.txt`, `proto/compatibility-baseline.binpb`

---

## Phase 2: Foundational — durable model и единый session pipeline

**Purpose**: Создать общие типы, валидацию и persistence-гарантии, от которых зависят все четыре user stories.

### Tests

**Wave 1 — independent (different files):**

- [x] **T005** [P] Написать падающие domain-тесты для optional config, frozen snapshots, stable-ID references, invalid variants, unknown JSON fields и legacy defaults · `internal/domain/model_test.go`, `internal/domain/validate_test.go`
- [x] **T006** [P] Написать падающие persistence-тесты для protobuf round-trip, monotonic document revisions, execute/reset mutations, stale whole-document merge, deletion pruning и atomic failures · `internal/session/contract_test.go`, `internal/session/service_test.go`, `internal/session/storage_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — domain model:**

- [x] **T007** Реализовать durable `StateChangeConfig`, `CommandExecutionState`, terminal command-state map, detached cloning, JSON known-field preservation и validation invariants · `internal/domain/model.go`, `internal/domain/json.go`, `internal/domain/validate.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — persistence contract:**

- [x] **T008** Реализовать явное domain ↔ persistence protobuf отображение новых известных session v1 полей без ProtoJSON и потери unknown JSON extras · `internal/session/contract.go`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — ordered session mutations:**

- [x] **T009** Сделать session service единым владельцем document revision; добавить ID-адресованные execute/reset mutations и merge server-owned snapshots при полном autosave · `internal/session/service.go`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5 — coordinator store seam:**

- [x] **T010** Добавить узкий `CommandStateStore` seam в coordinator и подключить его к session service с lock order `control → session` без обратных callback · `internal/control/service.go`, `main.go`

**Checkpoint**: Session v1 умеет безопасно хранить конфигурацию и completed snapshots; stale autosave и storage failure не могут стереть или частично установить состояние.

---

## Phase 3: User Story 1 — настройка и сброс мастером (P1)

**Goal**: Мастер настраивает четыре обязательных текста, видит frozen completed state и атомарно сбрасывает одну команду либо весь терминал.

**Independent Test**: Создать stateful-команду, сохранить/переоткрыть её, выполнить backend fixture mutation, изменить authored тексты, проверить frozen snapshot, затем подтвердить reset-one и reset-terminal; отмена reset ничего не меняет.

### Tests

**Wave 1 — independent (different files):**

- [x] **T011** [P] [US1] Написать падающие App/private-contract тесты для reset-one/reset-terminal, ID validation, idempotent no-op, active-terminal publication и session-state revision result · `app_test.go`, `app_contract_test.go`
- [x] **T012** [P] [US1] Написать падающий browser-тест мастерского editor flow: toggle, четыре поля, whitespace errors, frozen display, reset confirm/cancel и reset-all · `tests/browser/state-changing-command-authoring.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T013** [P] [US1] Реализовать master property editor для state-change toggle/полей, completed snapshot-индикации и reset-one/reset-terminal controls · `frontend/src/master.js`, `frontend/src/master.css`
- [x] **T014** [P] [US1] Реализовать private reset façades, повторную backend-валидацию, atomic store gate и active-runtime refresh · `app.go`, `app_contract.go`, `desktop_service.go`, `wails_host.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — desktop integration join:**

- [x] **T015** [US1] Подключить reset/session-state методы и события к generated Wails bindings, desktop API allowlist и browser fixture contract · `frontend/src/desktop-api.js`, `frontend/bindings/`, `scripts/wails-bindings-check.sh`, `tests/browser/fixtures/desktop-bindings.js`

**Checkpoint**: User Story 1 независимо демонстрируется в master UI и после reopen; reset меняет ровно выбранный durable scope и синхронизирует активную проекцию.

---

## Phase 4: User Story 2 — запрос игрока и решение мастера (P1, MVP)

**Goal**: Выбор initial stateful-команды создаёт один общий pending-запрос, мастер approve/reject/close разрешает его, а только durable approve показывает completed result.

**Independent Test**: Контроллер выбирает команду, все игроки видят «Выполняется запрос», master dialog появляется один раз; approve выполняет и сохраняет один раз, reject/close показывает «Запрос отклонён», persistence failure не публикует успех.

### Tests

**Wave 1 — independent (different files):**

- [x] **T016** [P] [US2] Написать падающие coordinator-тесты для single pending, exact request resolution, approve persist-before-success, reject/close, duplicate/stale decisions и 100 concurrent selections · `internal/control/service_test.go`
- [x] **T017** [P] [US2] Написать падающие live-тесты для effective initial/completed tree, pending action blocking, rejected Back и frozen repeat result без второго write · `internal/live/service_test.go`, `internal/nav/nav_test.go`
- [x] **T018** [P] [US2] Написать падающие public adapter/handler тесты для pending/rejected projection, controller-only failure notice, ordinary path и отсутствия private prompt/resolve symbols · `internal/player/adapter_test.go`, `internal/player/handler_test.go`
- [x] **T019** [P] [US2] Написать падающий browser journey для player request, единственного master dialog, approve, reject, close и persistence failure · `tests/browser/state-changing-command-approval.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T020** [P] [US2] Реализовать coordinator single-pending state machine, server request ID, private resolve validation и approve/reject revisions · `internal/control/service.go`
- [x] **T021** [P] [US2] Реализовать effective command projection, pending/rejected runtime presentation, shared-action conflict и completed repeat navigation · `internal/live/service.go`, `internal/nav/nav.go`
- [x] **T022** [P] [US2] Реализовать public protobuf adapters/handler mapping для command execution presentation и персонального безопасного persistence notice · `internal/player/adapter.go`, `internal/player/handler.go`
- [x] **T023** [P] [US2] Реализовать private resolve method, coordination projection, session-state emission и безопасные master errors · `app.go`, `app_contract.go`, `desktop_service.go`, `wails_host.go`
- [x] **T024** [P] [US2] Реализовать master approval dialog с дедупликацией по request ID и отображением approve/reject/persistence outcomes · `frontend/src/master.js`
- [x] **T025** [P] [US2] Реализовать player screens «Выполняется запрос»/«Запрос отклонён», блокировку действий, Back и controller notice без optimistic completion · `client/client.js`, `client/client.css`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — end-to-end integration join:**

- [x] **T026** [US2] Связать session store, coordinator effects, private desktop API и browser fixtures в полный player-request/master-decision flow · `main.go`, `frontend/src/desktop-api.js`, `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js`

**Checkpoint**: User Story 2 независимо работает от player click до master decision; reject не пишет session, approve публикуется только после durability, repeat не выполняется снова.

---

## Phase 5: User Story 3 — единое состояние, lifecycle и reconnect (P1)

**Goal**: Controller, observers и master сходятся на pending/rejected/completed состоянии; disconnect инициатора сохраняет запрос, а terminal/broadcast/app lifecycle отменяет только transient request.

**Independent Test**: Подключить controller и двух observers, создать pending, отключить инициатора и одобрить мастером; затем проверить reconnect snapshot, navigation, end/switch/start и app reopen. Повторить с lifecycle cancellation до решения.

### Tests

**Wave 1 — independent (different files):**

- [x] **T027** [P] [US3] Написать падающие coordinator race/lifecycle тесты для controller disconnect retention, end broadcast/switch/shutdown cancellation и stale dialog callback · `internal/control/service_test.go`
- [x] **T028** [P] [US3] Написать падающие snapshot/stream тесты для первого pending/rejected/completed состояния, monotonic revisions и reconnect после overflow · `internal/player/public_stream_test.go`, `internal/player/stream_test.go`
- [x] **T029** [P] [US3] Написать падающий multi-client browser journey с controller + двумя observers, disconnect/reconnect, navigation, terminal switch и broadcast restart · `tests/browser/state-changing-command-sync.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T030** [P] [US3] Реализовать сохранение pending при controller disconnect и атомарную отмену transient request/rejection на end broadcast, terminal switch и shutdown · `internal/control/service.go`
- [x] **T031** [P] [US3] Добавить pending в private coordination bootstrap/event, защитить master dialog от пропущенных событий и поздних callback, применить document revision ordering · `app.go`, `frontend/src/master.js`, `frontend/src/desktop-api.js`
- [x] **T032** [P] [US3] Обеспечить полные public snapshots и восстановление pending/rejected/completed UI без browser persistence · `internal/player/adapter.go`, `internal/player/stream.go`, `client/client.js`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — lifecycle fixture join:**

- [x] **T033** [US3] Расширить integration fixture для управляемых disconnect, master resolution, lifecycle transitions и reopen одного session file · `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js`

**Checkpoint**: User Story 3 независимо доказывает единый server-authoritative экран и корректное разделение transient pending от durable completed state на всех lifecycle-границах.

---

## Phase 6: User Story 4 — совместимость обычных команд и session JSON v1 (P2)

**Goal**: Старые v1-документы открываются без миграции, ordinary-команды идут прежним путём, а unknown fields и исходный формат остаются совместимыми.

**Independent Test**: Открыть неизменённый legacy fixture, выполнить ordinary-команды без pending/dialog/store write, сохранить и переоткрыть документ; отдельно round-trip новый optional-field fixture и проверить unknown extras.

### Tests

**Wave 1 — independent (different files):**

- [x] **T034** [P] [US4] Написать падающие legacy/forward-compat тесты для отсутствующих optional fields, version 1 round-trip, unknown extras и сохранения ordinary content · `internal/domain/model_test.go`, `internal/session/contract_test.go`, `internal/session/service_test.go`
- [x] **T035** [P] [US4] Написать падающие ordinary-command regression тесты: нет pending/master dialog/state-store write, прежние navigation/result/replay semantics · `internal/control/service_test.go`, `internal/live/service_test.go`, `internal/player/handler_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T036** [P] [US4] Закрепить absent-field defaults, unknown-field preservation и pruning только удалённых stable IDs в domain/session adapters · `internal/domain/json.go`, `internal/session/contract.go`, `internal/session/service.go`
- [x] **T037** [P] [US4] Сохранить отдельный прежний branch для ordinary commands без pending, master effect или command-state persistence · `internal/control/service.go`, `internal/live/service.go`, `internal/player/adapter.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — compatibility fixtures:**

- [x] **T038** [US4] Зафиксировать неизменённый legacy fixture и добавить отдельный state-changing v1 fixture для reopen/frozen/reset тестов · `internal/testutil/testdata/session-v1.json`, `internal/testutil/testdata/session-v1-state-changing.json`

**Checkpoint**: User Story 4 независимо доказывает открытие/сохранение legacy v1 и byte/behavior-compatible ordinary command flow без ручной конвертации.

---

## Phase 7: Polish и сквозная валидация

**Wave 1 — independent (different files):**

- [x] **T039** [P] Добавить понятный state-changing пример без выполненного snapshot и обновить пользовательское описание настройки/мастерского одобрения · `sessions/demo.json`, `README.md`

**⟶ Wait for Wave 1 and all story checkpoints to finish, then:**

**Wave 2 — single-owner validation:**

- [x] **T040** Выполнить single-owner Success Criteria validation: форматирование, vet, unit/race, protobuf format/generation/breaking/drift, Wails binding allowlist, frontend/client builds и полный Playwright suite; задокументировать только реально выполненные результаты · `Makefile`, `scripts/proto-check.sh`, `scripts/proto-breaking.sh`, `scripts/proto-drift-test.sh`, `scripts/wails-bindings-check.sh`, `frontend/package.json`, `client/package.json`, `tests/browser/package.json`

**Checkpoint**: SC-001–SC-015 подтверждены автоматическими проверками либо явно отмечены как недоступные; generated drift отсутствует, public/private capability boundary сохранена.

---

## Dependencies & Execution Order

### Phase dependencies

```text
Phase 1 Setup
  → Phase 2 Foundational
      → Phase 3 US1 Authoring/reset
          → Phase 4 US2 Request/approval (MVP)
              → Phase 5 US3 Sync/lifecycle
                  → Phase 6 US4 Compatibility
                      → Phase 7 Polish/validation
```

- Phase 1: T001/T002/T003 parallel → T004 generation join.
- Phase 2: T005/T006 parallel → T007 → T008 → T009 → T010.
- Phase 3: T011/T012 parallel → T013/T014 parallel → T015.
- Phase 4: T016/T017/T018/T019 parallel → T020/T021/T022/T023/T024/T025 parallel by owned files → T026 integration.
- Phase 5: T027/T028/T029 parallel → T030/T031/T032 parallel by owned files → T033 integration.
- Phase 6: T034/T035 parallel → T036/T037 parallel by owned packages → T038 fixtures.
- Phase 7: T039 after story checkpoints → T040 is the sole full-suite validation owner.

### Parallel opportunities

- Protobuf source areas are independent until generation; generated trees always have one owner in T004.
- Test waves intentionally touch separate files from one another and can be authored concurrently before implementation.
- Within US2 and US3, coordinator, live, transport, master UI and player UI tasks use frozen contracts and can proceed independently, but integration waits for the full wave.
- Tasks that revisit `internal/control/service.go`, `app.go`, `master.js`, `client.js` or shared browser fixtures are placed in later phases/waves and must not overlap earlier owners.

### MVP boundary

Phase 4 завершает минимальный вертикальный продукт: authored stateful-команда создаёт player request, показывает общее ожидание, разрешается мастером и сохраняет completed result ровно один раз. Phase 5 добавляет полную lifecycle/reconnect устойчивость, Phase 6 закрепляет регрессионную совместимость.

## Phase 8: Convergence

- [x] T041 Расширить автоматические coordinator/live проверки до 100 последовательных pending-проверок и 100 повторных/конкурентных обращений к completed-команде, доказывая ровно одно выполнение и одну durable write · `internal/control/service_test.go`, `internal/live/service_test.go` per SC-002, SC-004, SC-011 (partial)
- [x] T042 Добавить 100 детерминированных persistence-failure прогонов выполнения state-changing команды с проверкой неизменности durable/runtime state, revision/effects и восстановления прежнего состояния после reopen · `internal/control/service_test.go`, `internal/session/service_test.go`, `internal/session/storage_test.go` per SC-008 (partial)
- [x] T043 Довести матрицу explicit reject, dialog close и controller disconnect/approve до 100 случаев с единственным pending, точным resolution и отсутствием лишних store writes · `internal/control/service_test.go`, `tests/browser/state-changing-command-approval.spec.mjs` per SC-013, SC-015 (partial)
- [x] T044 Добавить явную проверку сходимости completed name/result у контроллера и минимум двух наблюдателей не позднее одной секунды после durable master decision · `tests/browser/state-changing-command-sync.spec.mjs` per SC-003 (partial)
- [x] T045 Добавить endurance-сценарий минимум из 20 переходов меню и проверку восстановления durable command state после остановки/старта broadcast, переключения терминалов и полного перезапуска desktop process с повторным открытием того же session file · `tests/browser/state-changing-command-sync.spec.mjs`, `tests/browser/fixture-server/main.go` per SC-005 (partial)
- [x] T046 Добавить выборку из 100 completed-команд для проверки сохранения snapshots при rename/move внутри терминала, удаления без наследования новым ID, frozen authored edits и reset/re-execute с новыми значениями · `internal/domain/model_test.go`, `internal/session/service_test.go` per SC-010, SC-012 (partial)

## Phase 9: Convergence

- [x] T047 Добавить 100 одновременных distinct requests к уже completed-команде и доказать, что все обращения получают frozen result без нового pending/master effect, а счётчики execution и durable write остаются равны одному · `internal/control/service_test.go` per SC-004, T041 (partial)
- [x] T048 Провести полную 100-command матрицу стадиями над каждой командой: rename/move и authored edit сохраняют frozen snapshot, reset/re-execute применяет новые тексты, а последующее удаление всех команд и создание 100 новых ID не наследует ни одного состояния · `internal/session/service_test.go` per SC-010, SC-012, T046 (partial)
