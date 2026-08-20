# Задачи: команды, изменяющие состояние пунктов меню

**Bugfix**: 2026-08-17 — BUG-001 Updated from bugfix patch
**Bugfix**: 2026-08-17 — BUG-002 Updated from bugfix patch
**Bugfix**: 2026-08-17 — BUG-003 Updated from bugfix patch
**Bugfix**: 2026-08-18 — BUG-004 Updated from bugfix patch
**Bugfix**: 2026-08-18 — BUG-005 Updated from bugfix patch
**Bugfix**: 2026-08-18 — BUG-006 Updated from bugfix patch
**Cross-feature bugfix**: 2026-08-20 — Approval-free ordinary/completed expectations in Phase 6, T035, T037, T047, T071 and T072 are superseded by `specs/010-terminal-navigation/bugs/BUG-005.md`; completed implementation-task status remains unchanged because no implementation gap was found by this reconciliation.

**Input**: [spec.md](./spec.md), [plan.md](./plan.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: Конституция и критерии SC-001–SC-018 требуют автоматических domain, persistence, concurrency, contract, production-faithful desktop integration и multi-client browser проверок. В каждой user-story фазе тестовые задачи выполняются до реализации и должны сначала зафиксировать ожидаемое падение.

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

- [x] **T011** [P] [US1] ⚠️ Reopened — Написать падающие App/private-contract тесты для reset-one/reset-terminal, ID validation, idempotent no-op, active-terminal publication и session-state revision result (reopened — BUG-002; reopened — BUG-003) · `app_test.go`, `app_contract_test.go`
- [x] **T012** [P] [US1] ⚠️ Reopened — Написать падающий browser-тест мастерского editor flow: toggle, четыре поля, whitespace errors, frozen display, reset confirm/cancel и reset-all (reopened — BUG-001; reopened — BUG-002; reopened — BUG-003) · `tests/browser/state-changing-command-authoring.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T013** [P] [US1] ⚠️ Reopened — Реализовать master property editor для state-change toggle/полей, completed snapshot-индикации и reset-one/reset-terminal controls (reopened — BUG-001) · `frontend/src/master.js`, `frontend/src/master.css`
- [x] **T014** [P] [US1] ⚠️ Reopened — Реализовать private reset façades, повторную backend-валидацию, atomic store gate и active-runtime refresh (reopened — BUG-001; reopened — BUG-002; reopened — BUG-003) · `app.go`, `app_contract.go`, `desktop_service.go`, `wails_host.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — desktop integration join:**

- [x] **T015** [US1] ⚠️ Reopened — Подключить reset/session-state методы и события к generated Wails bindings, desktop API allowlist и browser fixture contract (reopened — BUG-001; reopened — BUG-002; reopened — BUG-003) · `frontend/src/desktop-api.js`, `frontend/bindings/`, `scripts/wails-bindings-check.sh`, `tests/browser/fixtures/desktop-bindings.js`

**Checkpoint**: User Story 1 независимо демонстрируется в master UI и после reopen; reset меняет ровно выбранный durable scope и синхронизирует активную проекцию.

---

## Phase 4: User Story 2 — запрос игрока и решение мастера (P1, MVP)

**Goal**: Выбор initial stateful-команды создаёт один общий pending-запрос, мастер approve/reject/close разрешает его, а только durable approve показывает completed result.

**Independent Test**: Контроллер выбирает команду, все игроки вместо меню видят полноэкранное «Выполняется запрос», master dialog появляется один раз; approve выполняет и сохраняет один раз и показывает полноэкранный frozen result, ~~reject/close показывает «Запрос отклонён»~~ reject/close показывает полноэкранную «Ошибка доступа», `Back`/`Enter` заблокированы до решения и возвращают всех после него, persistence failure не публикует успех (уточнено BUG-004).

### Tests

**Wave 1 — independent (different files):**

- [x] **T016** [P] [US2] Написать падающие coordinator-тесты для single pending, exact request resolution, approve persist-before-success, reject/close, duplicate/stale decisions и 100 concurrent selections · `internal/control/service_test.go`
- [x] **T017** [P] [US2] ⚠️ Reopened — Написать падающие live-тесты для effective initial/completed tree, pending action blocking, rejected Back и frozen repeat result без второго write (reopened — BUG-004) · `internal/live/service_test.go`, `internal/nav/nav_test.go`
- [x] **T018** [P] [US2] Написать падающие public adapter/handler тесты для pending/rejected projection, controller-only failure notice, ordinary path и отсутствия private prompt/resolve symbols · `internal/player/adapter_test.go`, `internal/player/handler_test.go`
- [x] **T019** [P] [US2] ⚠️ Reopened — Написать падающий browser journey для player request, единственного master dialog с раздельными точным command name и отличающимся confirmation text, approve, reject, close и persistence failure (reopened — BUG-004; reopened — BUG-005; reopened — BUG-006) · `tests/browser/state-changing-command-approval.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T020** [P] [US2] Реализовать coordinator single-pending state machine, server request ID, private resolve validation и approve/reject revisions · `internal/control/service.go`
- [x] **T021** [P] [US2] ⚠️ Reopened — Реализовать effective command projection, pending/rejected runtime presentation, shared-action conflict и completed repeat navigation (reopened — BUG-004) · `internal/live/service.go`, `internal/nav/nav.go`
- [x] **T022** [P] [US2] Реализовать public protobuf adapters/handler mapping для command execution presentation и персонального безопасного persistence notice · `internal/player/adapter.go`, `internal/player/handler.go`
- [x] **T023** [P] [US2] Реализовать private resolve method, coordination projection, session-state emission и безопасные master errors · `app.go`, `app_contract.go`, `desktop_service.go`, `wails_host.go`
- [x] **T024** [P] [US2] Реализовать master approval dialog с дедупликацией по request ID и отображением approve/reject/persistence outcomes · `frontend/src/master.js`
- [x] **T025** [P] [US2] ⚠️ Reopened — ~~Реализовать player screens «Выполняется запрос»/«Запрос отклонён», блокировку действий, Back и controller notice без optimistic completion~~ Реализовать единый полноэкранный player screen для «Выполняется запрос»/«Ошибка доступа»/completed result, блокировку `Back`/`Enter` до решения, оба acknowledgement после решения и controller notice без optimistic completion (reopened — BUG-004; reopened — BUG-005) · `client/client.js`, `client/client.css`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — end-to-end integration join:**

- [x] **T026** [US2] ⚠️ Reopened — Связать session store, coordinator effects, private desktop API и parity канонического Go/JavaScript approval fixture в полный player-request/master-decision flow (reopened — BUG-004; reopened — BUG-006) · `main.go`, `frontend/src/desktop-api.js`, `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js`

**Checkpoint**: User Story 2 независимо работает от player click до master decision; reject не пишет session, approve публикуется только после durability, repeat не выполняется снова.

---

## Phase 5: User Story 3 — единое состояние, lifecycle и reconnect (P1)

**Goal**: Controller, observers и master сходятся на pending/rejected/completed состоянии; disconnect инициатора сохраняет запрос, а terminal/broadcast/app lifecycle отменяет только transient request.

**Independent Test**: Подключить controller и двух observers, создать pending, отключить инициатора и одобрить мастером; затем проверить reconnect snapshot, navigation, end/switch/start и app reopen. Повторить с lifecycle cancellation до решения.

### Tests

**Wave 1 — independent (different files):**

- [x] **T027** [P] [US3] Написать падающие coordinator race/lifecycle тесты для controller disconnect retention, end broadcast/switch/shutdown cancellation и stale dialog callback · `internal/control/service_test.go`
- [x] **T028** [P] [US3] Написать падающие snapshot/stream тесты для первого pending/rejected/completed состояния, monotonic revisions и reconnect после overflow · `internal/player/public_stream_test.go`, `internal/player/stream_test.go`
- [x] **T029** [P] [US3] ⚠️ Reopened — Написать падающий multi-client browser journey с controller + двумя observers, disconnect/reconnect, navigation, terminal switch и broadcast restart (reopened — BUG-004; reopened — BUG-005) · `tests/browser/state-changing-command-sync.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T030** [P] [US3] Реализовать сохранение pending при controller disconnect и атомарную отмену transient request/rejection на end broadcast, terminal switch и shutdown · `internal/control/service.go`
- [x] **T031** [P] [US3] Добавить pending в private coordination bootstrap/event, защитить master dialog от пропущенных событий и поздних callback, применить document revision ordering · `app.go`, `frontend/src/master.js`, `frontend/src/desktop-api.js`
- [x] **T032** [P] [US3] ⚠️ Reopened — Обеспечить полные public snapshots и восстановление pending/rejected/completed UI без browser persistence (reopened — BUG-004; reopened — BUG-005) · `internal/player/adapter.go`, `internal/player/stream.go`, `client/client.js`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — lifecycle fixture join:**

- [x] **T033** [US3] ⚠️ Reopened — Расширить integration fixture для управляемых disconnect, master resolution, lifecycle transitions и reopen одного session file (reopened — BUG-004) · `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js`

**Checkpoint**: User Story 3 независимо доказывает единый server-authoritative экран и корректное разделение transient pending от durable completed state на всех lifecycle-границах.

---

## Phase 6: User Story 4 — совместимость обычных команд и session JSON v1 (P2)

**Goal**: Старые v1-документы открываются без миграции, ~~ordinary-команды идут прежним approval-free путём~~ ordinary-команды сохраняют только non-durable approved-result semantics по `specs/010-terminal-navigation/bugs/BUG-005.md`, а unknown fields и исходный формат остаются совместимыми.

**Independent Test**: Открыть неизменённый legacy fixture, ~~выполнить ordinary-команды без pending/dialog/store write~~ выбрать ordinary-команды, получить pending/private master dialog и после approve проверить авторский результат без state-store write, сохранить и переоткрыть документ; отдельно round-trip новый optional-field fixture и проверить unknown extras. Approval-free часть теста заменена `specs/010-terminal-navigation/bugs/BUG-005.md`.

### Tests

**Wave 1 — independent (different files):**

- [x] **T034** [P] [US4] Написать падающие legacy/forward-compat тесты для отсутствующих optional fields, version 1 round-trip, unknown extras и сохранения ordinary content · `internal/domain/model_test.go`, `internal/session/contract_test.go`, `internal/session/service_test.go`
- [x] **T035** [P] [US4] ~~Написать падающие ordinary-command regression тесты: нет pending/master dialog/state-store write, прежние navigation/result/replay semantics.~~ Approval-free expectation заменено `specs/010-terminal-navigation/bugs/BUG-005.md`: ordinary selection требует pending/private master dialog, а approve сохраняет прежний авторский result/replay outcome без state-store write · `internal/control/service_test.go`, `internal/live/service_test.go`, `internal/player/handler_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent (different files):**

- [x] **T036** [P] [US4] Закрепить absent-field defaults, unknown-field preservation и pruning только удалённых stable IDs в domain/session adapters · `internal/domain/json.go`, `internal/session/contract.go`, `internal/session/service.go`
- [x] **T037** [P] [US4] ~~Сохранить отдельный прежний branch для ordinary commands без pending, master effect или command-state persistence.~~ Approval-free branch заменён `specs/010-terminal-navigation/bugs/BUG-005.md`: ordinary commands входят в универсальный approval lifecycle, но их одобренный outcome не создаёт command-state persistence · `internal/control/service.go`, `internal/live/service.go`, `internal/player/adapter.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — compatibility fixtures:**

- [x] **T038** [US4] Зафиксировать неизменённый legacy fixture и добавить отдельный state-changing v1 fixture для reopen/frozen/reset тестов · `internal/testutil/testdata/session-v1.json`, `internal/testutil/testdata/session-v1-state-changing.json`

**Checkpoint**: User Story 4 независимо доказывает открытие/сохранение legacy v1 и ~~byte/behavior-compatible approval-free ordinary command flow~~ JSON/authoring-compatible ordinary flow с обязательным master approval и non-durable approved result по `specs/010-terminal-navigation/bugs/BUG-005.md`, без ручной конвертации.

---

## Phase 7: Polish и сквозная валидация

**Wave 1 — independent (different files):**

- [x] **T039** [P] ⚠️ Reopened — Добавить понятный state-changing пример без выполненного snapshot и обновить пользовательское описание настройки/мастерского одобрения; закрепить в каноническом approval input явные folder/entry/command и валидный initial `stateChange` у каждой его команды (reopened — BUG-006) · `sessions/demo.json`, `README.md`

**⟶ Wait for Wave 1 and all story checkpoints to finish, then:**

**Wave 2 — single-owner validation:**

- [x] **T040** ⚠️ Reopened — Выполнить single-owner Success Criteria validation: форматирование, vet, unit/race, protobuf format/generation/breaking/drift, Wails binding allowlist, frontend/client builds, полный Playwright suite и обязательный native master-click reset gate; задокументировать только реально выполненные результаты (reopened — BUG-001; reopened — BUG-002; reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006) · `Makefile`, `scripts/proto-check.sh`, `scripts/proto-breaking.sh`, `scripts/proto-drift-test.sh`, `scripts/wails-bindings-check.sh`, `scripts/state-changing-reset-native-smoke.sh`, `frontend/package.json`, `client/package.json`, `tests/browser/package.json`

**Checkpoint**: SC-001–SC-018 подтверждены автоматическими проверками либо явно отмечены как недоступные; generated drift отсутствует, public/private capability boundary сохранена.

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

BUG-001 overlay
  T009/T011 audit + reopened T012
    → T049 regression reproduction
        → reopened T013/T014/T015 + T050 correction
            → T051 focused verification
                → reopened T040 full-suite validation

BUG-002 overlay
  T009 audit + reopened T011/T012/T049/T054 + T056 production-faithful reproduction
    → reopened T014/T015/T050 + T057 correction
        → reopened T051 + T058 focused verification
            → reopened T040 full-suite validation

BUG-003 overlay
  T009 audit + reopened T011/T012/T049/T054/T056/T058 + T059 native-click reproduction
    → reopened T014/T015/T050/T057 + T060 correction
        → reopened T051 + T061 native focused verification
            → reopened T040 full-suite validation

BUG-004 overlay
  reopened T017/T019 + T062 full-screen regression reproduction
    → reopened T021/T025/T026 + T063 presentation correction
        → reopened T029/T032/T033/T043/T044 + T064 multi-client verification
            → reopened T040 full-suite validation

BUG-005 overlay
  reopened T019/T062 + T065 record-renderer parity reproduction
    → reopened T025/T063 + T066 presentation correction
        → reopened T029/T032/T043/T044/T064 + T067 multi-client verification
            → reopened T040 full-suite validation

BUG-006 overlay
  reopened T019/T039 + T070 canonical-input/master-identity reproduction
    → reopened T026 + T071 fixture parity correction
        → T072 every-command focused verification
            → T072 + independent T069 native-gate stabilization
                → reopened T040 full-suite validation
```

- Phase 1: T001/T002/T003 parallel → T004 generation join.
- Phase 2: T005/T006 parallel → T007 → T008 → T009 → T010.
- Phase 3: T011/T012 parallel → T013/T014 parallel → T015.
- Phase 4: T016/T017/T018/T019 parallel → T020/T021/T022/T023/T024/T025 parallel by owned files → T026 integration.
- Phase 5: T027/T028/T029 parallel → T030/T031/T032 parallel by owned files → T033 integration.
- Phase 6: T034/T035 parallel → T036/T037 parallel by owned packages → T038 fixtures.
- Phase 7: T039 after story checkpoints → T040 is the sole full-suite validation owner.
- Phase 10 (BUG-001 overlay): audit T009/T011 and complete reopened T012 → T049 → reopened T013/T014/T015 with T050 → T051 → reopened T040.
- Phase 13 (BUG-002 overlay): audit T009 and complete reopened T011/T012/T049/T054 with T056 → reopened T014/T015/T050 with T057 → reopened T051 with T058 → reopened T040.
- Phase 14 (BUG-003 overlay): audit T009 and complete reopened T011/T012/T049/T054/T056/T058 with T059 → reopened T014/T015/T050/T057 with T060 → reopened T051 with T061 → reopened T040.
- Phase 15 (BUG-004 overlay): complete reopened T017/T019 with T062 → reopened T021/T025/T026 with T063 → reopened T029/T032/T033/T043/T044 with T064 → reopened T040.
- Phase 16 (BUG-005 overlay): complete reopened T019/T062 with T065 → reopened T025/T063 with T066 → reopened T029/T032/T043/T044/T064 with T067 → reopened T040.
- Phase 19 (BUG-006 overlay): complete reopened T019/T039 with T070 → reopened T026 with T071 → T072; complete independent T069 and T072 before reopened T040.

### Parallel opportunities

- Protobuf source areas are independent until generation; generated trees always have one owner in T004.
- Test waves intentionally touch separate files from one another and can be authored concurrently before implementation.
- Within US2 and US3, coordinator, live, transport, master UI and player UI tasks use frozen contracts and can proceed independently, but integration waits for the full wave.
- Tasks that revisit `internal/control/service.go`, `app.go`, `master.js`, `client.js` or shared browser fixtures are placed in later phases/waves and must not overlap earlier owners.
- BUG-004 сохраняет существующую protobuf-модель: T063 использует `command_execution` для pending/rejected и `Nav.CommandNodeID` для completed, поэтому T002/T004/T018/T020/T022/T028 не переоткрываются; T064 повторно проверяет их границы через snapshots и browser journeys.
- BUG-005 не меняет protobuf или server-authoritative state machine: T065–T067 локализованы в player rendering/browser verification, а T017/T021/T026/T033 требуют только аудита при обнаружении transport/runtime расхождения.
- BUG-006 не меняет глобальную семантику ordinary-команд и не переиспользует completed reset fixture как approval input: T070–T072 закрепляют fixture-scoped contract и command identity на browser/master границе, а T024/T033/T038 остаются закрытыми до конкретного focused-test расхождения.

### MVP boundary

Phase 4 завершает минимальный вертикальный продукт: authored stateful-команда создаёт player request, показывает общее ожидание, разрешается мастером и сохраняет completed result ровно один раз. Phase 5 добавляет полную lifecycle/reconnect устойчивость, Phase 6 закрепляет регрессионную совместимость.

## Phase 8: Convergence

- [x] T041 Расширить автоматические coordinator/live проверки до 100 последовательных pending-проверок и 100 повторных/конкурентных обращений к completed-команде, доказывая ровно одно выполнение и одну durable write · `internal/control/service_test.go`, `internal/live/service_test.go` per SC-002, SC-004, SC-011 (partial)
- [x] T042 Добавить 100 детерминированных persistence-failure прогонов выполнения state-changing команды с проверкой неизменности durable/runtime state, revision/effects и восстановления прежнего состояния после reopen · `internal/control/service_test.go`, `internal/session/service_test.go`, `internal/session/storage_test.go` per SC-008 (partial)
- [x] **T043** ⚠️ Reopened — Довести матрицу explicit reject, dialog close и controller disconnect/approve до 100 случаев с единственным pending, точным resolution и отсутствием лишних store writes (reopened — BUG-004; reopened — BUG-005) · `internal/control/service_test.go`, `tests/browser/state-changing-command-approval.spec.mjs` per SC-013, SC-015, SC-017 (partial)
- [x] **T044** ⚠️ Reopened — Добавить явную проверку сходимости completed name/result у контроллера и минимум двух наблюдателей не позднее одной секунды после durable master decision (reopened — BUG-004; reopened — BUG-005) · `tests/browser/state-changing-command-sync.spec.mjs` per SC-003, SC-017 (partial)
- [x] T045 Добавить endurance-сценарий минимум из 20 переходов меню и проверку восстановления durable command state после остановки/старта broadcast, переключения терминалов и полного перезапуска desktop process с повторным открытием того же session file · `tests/browser/state-changing-command-sync.spec.mjs`, `tests/browser/fixture-server/main.go` per SC-005 (partial)
- [x] T046 Добавить выборку из 100 completed-команд для проверки сохранения snapshots при rename/move внутри терминала, удаления без наследования новым ID, frozen authored edits и reset/re-execute с новыми значениями · `internal/domain/model_test.go`, `internal/session/service_test.go` per SC-010, SC-012 (partial)

## Phase 9: Convergence

- [x] T047 Добавить 100 одновременных distinct requests к уже completed-команде и доказать, что ~~все обращения получают frozen result без нового pending/master effect~~ каждый distinct selection создаёт ровно один private master approval request, а после approve возвращает frozen result без повторного execution или durable write; счётчики execution и durable write остаются равны одному · `internal/control/service_test.go` per SC-004, T041 (partial)
- [x] T048 Провести полную 100-command матрицу стадиями над каждой командой: rename/move и authored edit сохраняют frozen snapshot, reset/re-execute применяет новые тексты, а последующее удаление всех команд и создание 100 новых ID не наследует ни одного состояния · `internal/session/service_test.go` per SC-010, SC-012, T046 (partial)

## Phase 10: Bugfix BUG-001 — восстановление индивидуального и общего сброса

**Goal**: Доказать и восстановить полный путь подтверждённого сброса от мастерского интерфейса до долговечного состояния и активной runtime-проекции.

**Wave 1 — regression reproduction:**

- [x] **T049** [US1] ⚠️ Reopened — Добавить падающее сквозное воспроизведение BUG-001 на реально выполненной stateful-команде для «Сбросить состояние» и «Сбросить все состояния» с проверкой удаления нужных `commandStates`, новой document revision и обновления master/player runtime-проекций (reopened — BUG-002; reopened — BUG-003) · `tests/browser/state-changing-command-authoring.spec.mjs`, `app_test.go`, `app_contract_test.go`

**⟶ T049 and the T009/T011 audit must finish before Wave 2:**

**Wave 2 — correction:**

- [x] **T050** [US1] ⚠️ Reopened — Локализовать и исправить общий reset-путь `master UI → confirmation → desktop API/Wails binding → private handler → session mutation → session-state/active-runtime refresh`, сохранив ID-валидацию, idempotent no-op, атомарность и точную область reset-one/reset-terminal (reopened — BUG-002; reopened — BUG-003) · `frontend/src/master.js`, `frontend/src/desktop-api.js`, `app.go`, `app_contract.go`, `desktop_service.go`, `wails_host.go`, `internal/session/service.go`

**⟶ T050 and reopened T012–T015 must finish before Wave 3:**

**Wave 3 — focused verification:**

- [x] **T051** [US1] ⚠️ Reopened — Проверить индивидуальный и общий подтверждённый сброс, отмену без мутации, синхронизацию активных представлений и сохранение исходного состояния после повторного открытия того же session JSON; затем передать результат в переоткрытую T040 (reopened — BUG-002; reopened — BUG-003) · `tests/browser/state-changing-command-authoring.spec.mjs`, `internal/session/service_test.go`, `app_test.go`

## Phase 11: Convergence

- [x] T052 CRITICAL: Заменить `context.Background()`/`context.TODO()` в feature-scoped session tests на `t.Context()` и корректно производные test-scoped cleanup/timeout contexts · `internal/session/service_test.go` per Constitution: Testing and Quality Gates (contradicts)
- [x] T053 CRITICAL: Перевести handwritten `t.Fatal`/`t.Error` и `reflect.DeepEqual` assertions в feature-scoped domain/session/storage tests на `testify/assert` или `testify/require`, сохранив table-driven cases и protobuf-aware сравнения там, где применимо · `internal/domain/model_test.go`, `internal/domain/validate_test.go`, `internal/session/service_test.go`, `internal/session/storage_test.go` per Constitution: Testing and Quality Gates (contradicts)
- [x] T054 ⚠️ Reopened — Добавить исполняемый `npm test --prefix frontend` suite для master validation, единственного approval dialog, approve/reject/close, production-faithful reset и revision ordering и зарегистрировать `test` script в `frontend/package.json` (reopened — BUG-002; reopened — BUG-003) per plan: Verification Plan / Master frontend, Native reset integration (partial)

## Phase 12: Convergence

- [x] T055 CRITICAL: Перевести три handwritten `t.Fatalf`/`reflect.DeepEqual` проверки feature-scoped navigation tests на `testify/assert` или `testify/require`, удалить импорт `reflect` и сохранить существующие table-driven cases · `internal/nav/nav_test.go` per Constitution: Testing and Quality Gates (contradicts)

## Phase 13: Bugfix BUG-002 — production-faithful общий сброс активного терминала

**Goal**: Воспроизвести и устранить повторную регрессию общего сброса через реальный desktop/backend path, доказав каноническое долговечное и runtime-состояние без fixture-only мутации.

**Wave 1 — regression reproduction:**

- [x] **T056** [US1] ⚠️ Reopened — Добавить падающее production-faithful воспроизведение общего сброса активного терминала в writable-сессии: реальный `App` + session service + coordinator, вызов `ResetTerminalCommandStates` через desktop contract, новая document revision, отсутствие прежних `commandStates` и обновление master/controller/observer; browser binding MUST проксировать вызов к Go fixture/backend, а не очищать JS fixture самостоятельно (reopened — BUG-003) · `app_test.go`, `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js`, `tests/browser/state-changing-command-authoring.spec.mjs` per FR-023, SC-009, SC-016

**⟶ T056 and reopened T011/T012/T049/T054 must fail for the observed regression before Wave 2:**

**Wave 2 — correction:**

- [x] **T057** [US1] ⚠️ Reopened — По падающему T056 локализовать и исправить ответственный production-слой общего reset path, принимать успех только из канонического backend result с более новой revision и пустым terminal `commandStates`, затем синхронизировать active runtime и master/player projections без optimistic state (reopened — BUG-003) · `frontend/src/master.js`, `frontend/src/desktop-api.js`, `app.go`, `app_contract.go`, `desktop_service.go`, `wails_host.go`, `internal/session/service.go`, `internal/control/service.go` per FR-023, SC-016

**⟶ T057 and reopened T014/T015/T050 must finish before Wave 3:**

**Wave 3 — focused verification:**

- [x] **T058** [US1] ⚠️ Reopened — Проверить общий reset активного writable-терминала для нескольких completed-команд: точный terminal scope, одна durable revision, master/controller/observer initial ≤1 s, отсутствие stale completed result после navigation/frontend reload и сохранение initial после reopen того же JSON; затем передать доказательство в переоткрытую T040 (reopened — BUG-003) · `app_test.go`, `internal/session/service_test.go`, `internal/control/service_test.go`, `tests/browser/state-changing-command-authoring.spec.mjs`, `tests/browser/state-changing-command-sync.spec.mjs` per SC-009, SC-016

## Phase 14: Bugfix BUG-003 — реальный native-клик общего сброса

**Goal**: Воспроизвести и устранить BUG-003 через ту же пользовательскую последовательность, которая остаётся сломанной: фактический клик и подтверждение reset-all в собранном приложении мастера с наблюдением канонического backend-состояния и активных player-представлений.

**Wave 1 — native regression reproduction:**

- [x] **T059** [US1] Добавить падающий автоматизируемый native smoke: запустить собранное приложение с реальным writable session file и completed-командой, подключить controller и observer, нажать и подтвердить «Сбросить все состояния» в master UI и зафиксировать для одного terminal ID факт generated Wails-вызова, backend result/error, document revision, канонические `commandStates`, `session-state` event, runtime revision и master/player DOM; direct App invocation и browser fixture не засчитываются вместо клика · `scripts/state-changing-reset-native-smoke.sh`, `app_test.go`, `frontend/src/master.js` per FR-023/BUG-003, SC-009, SC-016

**⟶ T059 and the T009 audit plus reopened T011/T012/T049/T054/T056/T058 must reproduce the observed failure before Wave 2:**

**Wave 2 — correction:**

- [x] **T060** [US1] По доказательствам T059 локализовать первое расходящееся звено и исправить только ответственный production path `master control → confirmation → generated Wails binding → desktop façade → App/private contract → session mutation → session-state/runtime publication → master/player rendering`; отсутствие вызова, backend error/no-op и непринятая revision MUST быть видимыми ошибками, а успех допустим только для канонического `INITIAL` · `frontend/src/master.js`, `frontend/src/desktop-api.js`, `frontend/bindings/`, `desktop_service.go`, `app.go`, `app_contract.go`, `internal/session/service.go`, `internal/control/service.go` per FR-023/BUG-003, SC-016

**⟶ T060 and reopened T014/T015/T050/T057 must finish before Wave 3:**

**Wave 3 — native focused verification:**

- [x] **T061** [US1] Повторить native master-click gate для нескольких completed-команд выбранного терминала и отдельного нетронутого терминала; доказать одну новую durable revision, master/controller/observer `INITIAL` ≤1 s без reload, сохранение другого terminal scope, отсутствие stale result после navigation и сохранение результата после полного закрытия приложения и reopen того же JSON, затем передать доказательство в T040 · `scripts/state-changing-reset-native-smoke.sh`, `app_test.go`, `tests/browser/state-changing-command-sync.spec.mjs` per SC-009, SC-016

## Phase 15: Bugfix BUG-004 — полноэкранный lifecycle выбранной команды

**Goal**: После выбора state-changing команды controller и observers видят единый полноэкранный экран записи; pending не принимает `Back`/`Enter`, а rejected/completed сохраняются до равнозначного server-authoritative acknowledgement этими действиями контроллера.

**Wave 1 — regression reproduction:**

- [x] **T062** [US2] ⚠️ Reopened — Добавить падающее воспроизведение BUG-004 для полной матрицы `PENDING|REJECTED|COMPLETED × Back|Enter`: экран занимает представление записи без списка меню, pending не меняет nav/revision, reject показывает «Ошибка доступа», approve показывает frozen result, а после решения оба действия возвращают controller и observers в прежнее меню (reopened — BUG-005) · `internal/live/service_test.go`, `internal/nav/nav_test.go`, `tests/browser/state-changing-command-approval.spec.mjs` per FR-006, FR-008, FR-030, FR-033, SC-002, SC-003, SC-013, SC-017

**⟶ T062 and reopened T017/T019 must reproduce the incomplete presentation before Wave 2:**

**Wave 2 — correction:**

- [x] **T063** [US2] ⚠️ Reopened — По падающему T062 реализовать единое полноэкранное представление записи для pending, «Ошибка доступа» и completed result; скрывать список меню, сохранять pagination/reveal длинного результата, блокировать `Back`/`Enter` в pending и нормализовать оба ввода после решения в один accepted shared `Navigate back` без browser-owned или optimistic state (reopened — BUG-005) · `client/client.js`, `client/client.css`, `internal/live/service.go`, `internal/nav/nav.go` per FR-006, FR-008, FR-030, FR-033

**⟶ T063 and reopened T021/T025/T026 must finish before Wave 3:**

**Wave 3 — focused verification:**

- [x] **T064** [US2] [US3] ⚠️ Reopened — Проверить controller + минимум двух observers для approve/reject/close, `Back`/`Enter`, длинного paginated результата, reconnect до acknowledgement, controller disconnect, terminal switch и broadcast restart; доказать одинаковый экран ≤1 s, read-only observers, отсутствие локального расхождения и возврат всех представлений только по принятой runtime revision, затем передать доказательство в переоткрытую T040 (reopened — BUG-005) · `internal/player/public_stream_test.go`, `internal/player/stream_test.go`, `tests/browser/state-changing-command-approval.spec.mjs`, `tests/browser/state-changing-command-sync.spec.mjs`, `tests/browser/fixture-server/main.go` per SC-003, SC-013, SC-015, SC-017

## Phase 16: Bugfix BUG-005 — паритет с рендером описания записи

**Goal**: `PENDING`, `REJECTED`, первый и повторный `COMPLETED` используют тот же presentation contract, что описание обычной записи, выбранной в меню, при сохранении server-authoritative lifecycle и общей навигации.

**Wave 1 — renderer-parity regression reproduction:**

- [x] **T065** [US2] Добавить падающее browser-воспроизведение BUG-005: выбрать обычную запись как эталон и сравнить с `PENDING`, `REJECTED`, первым `COMPLETED` и повторным просмотром completed-команды область контента, типографику, переносы, reveal, pagination/page controls и repagination при resize для короткого, многострочного и длинного текста; существующий отдельный `#termOutput.command-screen` MUST провалить проверку · `tests/browser/state-changing-command-approval.spec.mjs`, `tests/browser/state-changing-command-sync.spec.mjs`, `tests/browser/fixture-server/main.go` per FR-006, FR-008, FR-015, FR-030, FR-033/BUG-005, SC-002, SC-003, SC-013, SC-017/BUG-005

**⟶ T065 and reopened T019/T062 must fail on the separate command-output renderer before Wave 2:**

**Wave 2 — shared record renderer correction:**

- [x] **T066** [US2] По падающему T065 заменить отдельные pending/rejected/completed command-output branches общим record-description presentation primitive либо доказуемо эквивалентным компонентом с одинаковыми layout/reveal/pagination/resize semantics; сохранить pending action blocking, controller-only acknowledgement, observer read-only и текущие server/protobuf contracts · `client/client.js`, `client/client.css` per FR-006, FR-008, FR-015, FR-030, FR-033/BUG-005

**⟶ T066 and reopened T025/T063 must finish before Wave 3:**

**Wave 3 — multi-client renderer verification:**

- [x] **T067** [US2] [US3] Проверить record-renderer parity для controller + минимум двух observers во всех исходах approve/reject/close, при reconnect до acknowledgement, повторном выборе completed-команды, `Back`/`Enter` и resize на каждой странице длинного текста; доказать одинаковый экран ≤1 s и возврат всех клиентов только по принятой runtime revision, затем передать результат в переоткрытую T040 · `tests/browser/state-changing-command-approval.spec.mjs`, `tests/browser/state-changing-command-sync.spec.mjs`, `tests/browser/fixture-server/main.go` per SC-002, SC-003, SC-013, SC-017/BUG-005

## Phase 17: Convergence

- [x] T068 HIGH: Расширить обязательный native master-click reset gate до непрерывной проверки собранного приложения с writable session, controller и observer: зафиксировать generated Wails result/error, совпадающий terminal ID, одну новую document revision, канонически пустые `commandStates`, `session-state` event, новую runtime revision и `INITIAL` в master/controller/observer DOM не позднее одной секунды без reload; сохранить completed-состояния другого терминала, исключить stale result после navigation, полностью закрыть приложение и доказать тот же результат после reopen исходного JSON · `scripts/state-changing-reset-native-smoke.sh` per FR-023/BUG-003, SC-016, plan: Native master click regression, T059/T061 (partial)

## Phase 18: Convergence

- [x] T069 HIGH: Стабилизировать обязательный native master-click reset gate для повторных запусков на актуальном packaged app: надёжно синхронизировать lifecycle native process и появление master Wails/session-state accessibility evidence, сохранять диагностические данные при каждом раннем выходе и доказать последовательными успешными прогонами полный reset/reopen сценарий с controller и observer без ослабления ограничений по revision, terminal scope, `INITIAL` ≤1 s и отсутствию stale result · `scripts/state-changing-reset-native-smoke.sh`, `scripts/state-changing-reset-native-player-smoke.mjs`, `frontend/src/master.js` per FR-023/BUG-003, SC-016, plan: Native master click regression (partial)

## Phase 19: Bugfix BUG-006 — канонический approval input и имя команды у мастера

**Goal**: Канонический approval input содержит минимум одну явную папку, запись и команду, каждая его команда начинает в `INITIAL` с валидным `stateChange`, а master dialog отдельно показывает точное имя выбранной команды и отличный авторский prompt без изменения ordinary-command compatibility вне fixture.

**Wave 1 — fixture/master regression reproduction:**

- [x] **T070** [US2] Добавить падающую структурную и browser-проверку BUG-006: канонический approval input имеет явные folder/entry/command, пустые initial `commandStates`, непустые `name`/`text`/`completedName`/`confirmationText` у каждой команды; выбрать каждую команду и доказать `INITIAL → PENDING`, отдельные видимые `КОМАНДА: <точное имя>` и намеренно отличный `confirmationText`, а также отсутствие ordinary bypass (reopened T019/T039) · `sessions/demo.json`, `tests/browser/state-changing-command-authoring.spec.mjs`, `tests/browser/state-changing-command-approval.spec.mjs`, `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js` per FR-001–FR-006/BUG-006, SC-001, SC-002, SC-018

**⟶ T070 and reopened T019/T039 must fail on the incomplete or divergent approval input before Wave 2:**

**Wave 2 — canonical fixture parity correction:**

- [x] **T071** [US2] По падающему T070 определить один канонический approval fixture contract либо точную автоматическую parity-проекцию и выровнять его Go/JavaScript/JSON представления: добавить явные folder/entry/command, обеспечить initial `stateChange` каждой команды и раздельный `commandName`/`confirmationText` в master dialog; ~~не менять ordinary semantics~~ сохранить только non-durable semantics одобренного ordinary-результата, тогда как approval-free selection expectation заменено `specs/010-terminal-navigation/bugs/BUG-005.md`; legacy fixture и отдельный completed fixture для reset/reopen остаются исторической областью задачи (reopened T026; T024/T033/T038 только audit) · `sessions/demo.json`, `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js`, `frontend/src/master.js`, `README.md` per FR-004, FR-005/BUG-006, FR-017, FR-028, SC-007, SC-018

**⟶ T071 and reopened T026 must finish before Wave 3:**

**Wave 3 — every-command focused verification:**

- [x] **T072** [US2] [US4] Пройти каждую команду канонического approval input через approve, reject и close, проверяя точное имя и отличный prompt, единственный pending/request ID, отсутствие initial completed snapshot и ordinary bypass; ~~отдельно подтвердить неизменный approval-free ordinary flow legacy fixture~~ это ожидание заменено `specs/010-terminal-navigation/bugs/BUG-005.md`: каждый выбор требует master approval, ordinary approval остаётся non-durable, а completed state-changing approval не вызывает второго выполнения или записи; parity fixture sources и полный frontend/Playwright suite остаются исторической областью задачи, затем передать результат в переоткрытую T040 · `tests/browser/state-changing-command-approval.spec.mjs`, `tests/browser/state-changing-command-authoring.spec.mjs`, `tests/browser/state-changing-command-sync.spec.mjs`, `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js` per SC-002, SC-007, SC-013, SC-018/BUG-006
