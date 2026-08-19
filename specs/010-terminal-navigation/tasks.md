# Задачи: переходы между терминалами

**Bugfix**: 2026-08-19 — BUG-001 Updated from bugfix patch

**Input**: design artifacts from `specs/010-terminal-navigation/`

**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/session-v1.md`, `contracts/public-player.md`, `contracts/private-desktop.md`

**Testing**: Изменение затрагивает versioned protobuf, portable session JSON, конкурентный coordinator, private Wails bridge и два браузерных UI. Для каждой пользовательской истории сначала добавляются падающие focused Go/Playwright tests; финальная фаза выполняет schema, binding, race, build и browser gates из плана.

**Organization**: Задачи сгруппированы по приоритетным пользовательским историям. `[P]` означает независимую задачу текущей волны в других файлах; `[US#]` связывает работу с историей из `spec.md`.

## Phase 1: Setup — protobuf contracts and generated baseline

**Purpose**: Зафиксировать совместимые persistent, public и private контракты до изменения их producers/consumers.

**Wave 1 — independent (different files):**

- [x] **T001** [P] Добавить optional `TerminalTransitionConfig` в persistence v1, сохранив номера существующих полей и JSON version 1 · `proto/fallout/terminal/persistence/v1/session.proto`
- [x] **T002** [P] Добавить public direction/route/pending presentation и optional field 9 без новой player RPC · `proto/fallout/terminal/player/v1/terminal.proto`
- [x] **T003** [P] Добавить private pending/notice/decision contracts и exact resolve request/result с `UNSPECIFIED = 0` · `proto/fallout/terminal/private/v1/coordination.proto`, `proto/fallout/terminal/private/v1/desktop.proto`

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T004** Регенерировать pinned Go/ECMAScript protobuf outputs и reviewed schema revision штатным `make proto-generate` без ручного редактирования generated code · `internal/gen/fallout/terminal/persistence/v1/session.pb.go`, `internal/gen/fallout/terminal/player/v1/terminal.pb.go`, `internal/gen/fallout/terminal/private/v1/coordination.pb.go`, `internal/gen/fallout/terminal/private/v1/desktop.pb.go`, `client/gen/fallout/terminal/player/v1/terminal_pb.js`, `proto/schema-revision.txt`

---

## Phase 2: Foundational — durable models, trusted lookup, restoration, and adapters

**Purpose**: Построить общие модели, compatible persistence и detached boundary projections, блокирующие все пользовательские истории.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [x] **T005** [P] Покрыть deep clone, known/unknown JSON fields, two-pass validation, forward references, missing/self target и конфликт со `stateChange` · `internal/domain/model_test.go`, `internal/domain/validate_test.go`
- [x] **T006** [P] Покрыть legacy/new v1 round-trip, terminal-order independence, atomic invalid-save rejection и detached trusted catalog lookup · `internal/session/contract_test.go`, `internal/session/service_test.go`
- [x] **T007** [P] Покрыть поиск folder по stable ID, восстановление новой ancestry и nearest-ancestor/root fallback после move/delete · `internal/nav/nav_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T008** Добавить durable transition config, broadcast route/pending/notice/presentation aggregates, deep clones, JSON known fields и two-pass cross-terminal validation · `internal/domain/model.go`, `internal/domain/json.go`, `internal/domain/validate.go`

**⟶ Wait for T008 to finish, then Wave 3 — independent (different files):**

- [x] **T009** [P] Реализовать explicit protobuf mapping и current-session `TerminalCatalog`, возвращающий detached target/command snapshot без изменения storage/revision pipeline · `internal/session/contract.go`, `internal/session/service.go`
- [x] **T010** [P] Реализовать pure stable-folder lookup и детерминированное восстановление current ancestry с parent/root fallback · `internal/nav/nav.go`

**⟶ Wait for Wave 3 to finish, then:**

- [x] **T011** Подключить public/private protobuf projections, clone-safe adapters и catalog seam в composition, не раскрывая decision ID/full route публичному клиенту · `internal/player/adapter.go`, `app_contract.go`, `internal/control/service.go`, `main.go`

**Checkpoint**: Version-1 link, runtime navigation types, folder restoration, trusted lookup and typed boundary projections are independently testable foundations.

---

## Phase 3: User Story 1 — перейти по команде в другой терминал (Priority: P1) 🎯 MVP

**Goal**: Контролирующий игрок выбирает authored link, мастер видит один exact approve/reject dialog, а approve атомарно активирует target root без второго switch dialog.

**Independent Test**: Настроить A → B, сохранить/переоткрыть session, выбрать link контроллером и проверить pending, approve, reject, stale target, replay и блокировку конкурирующих действий.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [x] **T012** [P] [US1] Добавить coordinator tests для controller-only linked command, exact-one pending, 20 replayed requests, competing-action conflict, approve/reject, stale/missing/self target и atomic route/active revision · `internal/control/service_test.go`
- [x] **T013** [P] [US1] ⚠️ Reopened — добавить Playwright tests для mutually-exclusive authoring, completed-state guard, save/reopen, inbound-delete guard и master forward decision dialog; согласовать fixture round-trip с полной session, не теряющей второй terminal `(reopened — BUG-001)` · `tests/browser/terminal-navigation.spec.mjs`, `tests/browser/fixtures/desktop-bindings.js`

### Implementation

**⟶ Wait for Wave 1 to finish, then Wave 2 — independent (different files):**

- [x] **T014** [P] [US1] Перехватывать linked `NavigateCommand` после authority/replay checks, создавать один `PendingTerminalNavigation`, блокировать gameplay и атомарно approve/reject со свежим catalog lookup · `internal/control/service.go`
- [x] **T015** [P] [US1] Добавить command-editor toggle/select, mutual exclusion, local validation, inbound-reference delete guard и deduplicated master dialog с close-as-reject/stale callback guard · `frontend/src/master.js`, `frontend/src/master.css`, `frontend/src/desktop-api.js`
- [x] **T016** [P] [US1] Отображать authoritative pending status и делать shared controls inert без optimistic terminal switch · `client/client.js`, `client/client.css`

**⟶ Wait for Wave 2 to finish, then:**

- [x] **T017** [US1] Провести exact resolve через App и единственный private desktop method, валидировать enum/request ID, публиковать только newer coordination state и очищать pending/notice по contract · `app.go`, `desktop_service.go`, `app_contract.go`

**⟶ Wait for T017 to finish, then:**

- [x] **T018** [US1] Регенерировать allowlisted Wails binding и завершить browser fixture/server wiring для полного authored-link → pending → approve/reject journey · `frontend/bindings/github.com/obalunenko/Fallout-Terminal/desktopservice.js`, `frontend/bindings/github.com/obalunenko/Fallout-Terminal/models.js`, `tests/browser/fixture-server/main.go`, `tests/browser/fixtures/desktop-bindings.js`

**Checkpoint**: User Story 1 independently persists and executes one master-approved forward transition, while rejection and invalid/stale links preserve the source state.

---

## Phase 4: User Story 2 — не взламывать повторно уже открытый терминал (Priority: P1)

**Goal**: Broadcast-scoped terminal checkpoints retain solved, unfinished and failed hack state; forward approve places the destination at root without recreating its hack runtime.

**Independent Test**: Впервые открыть защищённый B, решить или частично пройти hack, уйти и вернуться не менее 10 раз; убедиться в сохранении состояния и в отсутствии второго preserve/discard dialog.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [x] **T019** [P] [US2] Добавить tests для first-entry hack, solved/unfinished/failed reactivation, explicit reset/discard, root placement и single-decision source checkpoint preservation · `internal/live/service_test.go`, `internal/control/service_test.go`
- [x] **T020** [P] [US2] Расширить browser journey проверками первого взлома, повторного входа без взлома и retained blocked/progress state · `tests/browser/terminal-navigation.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T021** [US2] Переиспользовать `TerminalRuntimes`/`SuspendRuntime`/`ReactivateRuntime`, сохранять private hack checkpoint и задавать destination root при approved transition без `PendingTerminalSwitch` · `internal/live/service.go`, `internal/control/service.go`

**Checkpoint**: User Story 2 independently preserves hack continuity for every revisited terminal within the current broadcast.

---

## Phase 5: User Story 3 — вернуться в предыдущий терминал (Priority: P1)

**Goal**: Root back creates one return approval; approve pops exactly one LIFO point and restores the source folder, its moved location, nearest surviving ancestor or root.

**Independent Test**: Пройти A/nested → B → C, отклонить и затем одобрить возвраты; проверить B затем A, точный saved folder, moved-folder recovery и delete fallback.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [x] **T022** [P] [US3] Добавить coordinator/nav tests для root-only return, unchanged-top validation, reject/close immutability, LIFO/cycles, pop-after-approve и moved/deleted folder restoration · `internal/control/service_test.go`, `internal/nav/nav_test.go`
- [x] **T023** [P] [US3] Расширить Playwright journey root return control, pending lock, approve/reject, nested restore, A → B → C unwind и no-route absence · `tests/browser/terminal-navigation.spec.mjs`

### Implementation

**⟶ Wait for Wave 1 to finish, then Wave 2 — independent (different files):**

- [x] **T024** [P] [US3] Интерпретировать root `NavigateBack` как return request, хранить immutable top copy и при approve restore folder/fallback и pop ровно одну route point · `internal/control/service.go`
- [x] **T025** [P] [US3] Показывать authoritative return target только в root list и отправлять existing `NavigateBack`, сохраняя прежний intra-terminal back · `client/client.js`, `client/client.css`

**Checkpoint**: User Story 3 independently unwinds direct transitions one approved LIFO step at a time and restores the intended source menu.

---

## Phase 6: User Story 4 — сохранить общий и устойчивый маршрут (Priority: P2)

**Goal**: Все игроки и переподключившиеся клиенты сходятся к одной revisioned projection; observer, stale edits and lifecycle resets cannot move to a wrong terminal or retain an old route.

**Independent Test**: Использовать controller и двух observers, reconnect во время pending, изменить/delete source/target до approve/return, затем проверить monotonic convergence и empty route после manual switch/end/shutdown/new broadcast.

### Tests

**Wave 1 — independent (different files), write these tests to fail first:**

- [x] **T026** [P] [US4] Добавить authority, concurrent request, snapshot/reconnect, monotonic stream, stale edit, manual/end/shutdown clearing и public/private capability-separation tests · `internal/control/service_test.go`, `internal/player/adapter_test.go`, `internal/player/public_stream_test.go`, `app_test.go`, `app_contract_test.go`, `wails_host_test.go`
- [x] **T027** [P] [US4] Расширить journey controller + two observers проверками pending reconnect, convergence ≤2s, stale target safe failure, moved/deleted return path и new-broadcast cleanup · `tests/browser/terminal-navigation.spec.mjs`, `tests/browser/fixture-server/main.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T028** [US4] Завершить single-revision public/private publication, reconnect projection, typed master notice и route/pending/notice clearing при manual activation/clear, end broadcast и shutdown · `internal/control/service.go`, `internal/player/adapter.go`, `app.go`, `frontend/src/master.js`, `client/client.js`

**Checkpoint**: User Story 4 independently proves authoritative multi-client convergence, safe stale handling and broadcast-scoped cleanup.

---

## Final Phase: Polish and Success-Criteria validation

**Purpose**: Проверить совместимость, отсутствие generated drift/private leakage и все измеримые outcomes одной полной реализацией.

**Wave 1:**

- [x] **T029** Проверить protobuf format/lint/generation/breaking и exact Wails allowlist командами `make proto-check proto-breaking bindings-check`; исправить только governed schema/adapters, если найден drift или public leakage · `proto/`, `internal/gen/`, `client/gen/`, `frontend/bindings/`, `app_contract.go`

**⟶ Wait for T029 to finish, then:**

- [x] **T030** ⚠️ Reopened — выполнить единственную полную Success Criteria validation: `go test ./internal/domain ./internal/session ./internal/nav`, `go test -race ./internal/control ./internal/live ./internal/player`, `go test ./...`, frontend/client builds, focused `terminal-navigation.spec.mjs`, затем `make check`, включая SC-009 `(reopened — BUG-001)` · `specs/010-terminal-navigation/spec.md`, `Makefile`, `frontend/package.json`, `client/package.json`, `tests/browser/package.json`

**⟶ Wait for T030 to finish, then:**

- [x] **T031** ⚠️ Reopened — собрать accepted Wails runtime и пройти native master + controller + two observers smoke для approve/reject, 10 revisits, three-terminal unwind, reconnect and shutdown cleanup; дополнительно открыть реальный `sessions/demo.json`, применить `t_demo1` → `t_demo2`, дождаться durable revision и проверить цель после reopen; зафиксировать любой недоступный manual gate без ложного PASS `(reopened — BUG-001)` · `main.go`, `app.go`, `wails.json`, `sessions/demo.json`, `specs/010-terminal-navigation/spec.md`

---

## Dependencies & Execution Order

- Phase order: Setup → Foundational → US1 (MVP) → US2 → US3 → US4 → Polish. US2 и US3 используют forward lifecycle US1; US4 проверяет и завершает общий маршрут после всех P1 slices.
- Phase 1: Wave 1 (`T001–T003`) → `T004`.
- Phase 2: test Wave 1 (`T005–T007`) → `T008` → implementation Wave 3 (`T009–T010`) → `T011`.
- Phase 3: test Wave 1 (`T012–T013`) → implementation Wave 2 (`T014–T016`) → `T017` → `T018`.
- Phase 4: test Wave 1 (`T019–T020`) → `T021`.
- Phase 5: test Wave 1 (`T022–T023`) → implementation Wave 2 (`T024–T025`).
- Phase 6: test Wave 1 (`T026–T027`) → `T028`.
- Polish: `T029` → `T030` → `T031`.
- BUG-001 corrective DAG overrides the earlier completion state: `T034` failing regression → `T035` correction → reopened `T013` and `T033` → reopened `T030` → reopened `T031` → `T036` accepted packaged-runtime gate.

## Parallel Opportunities

- Contract files `T001–T003`, foundation test files `T005–T007`, and foundation implementations `T009–T010` are independent inside their declared waves.
- After the foundation, each story's Go tests and Playwright tests can be authored together; implementation work marked `[P]` separates coordinator, master UI and player UI files.
- Tasks that revisit `internal/control/service.go`, `internal/control/service_test.go`, `client/client.js`, `frontend/src/master.js`, or `tests/browser/terminal-navigation.spec.mjs` stay ordered by phase even when they cover different stories.

## Phase 7: Convergence

- [x] T032 **CRITICAL** Сделать снятие блокировки обычной folder/entry/back-навигации независимым от порядка authoritative stream и unary result и добавить regression coverage для replay, multi-player, public-fallback и shared-lifecycle сценариев per FR-028 / Constitution Testing and Quality Gates (contradicts) · `client/client.js`, `tests/browser/connectrpc-player.spec.mjs`, `tests/browser/player-sessions-control.spec.mjs`, `tests/browser/public-access-fallback.spec.mjs`, `tests/browser/state-changing-command-sync.spec.mjs`
- [x] T033 **CRITICAL** ⚠️ Reopened — разделить browser-assertions bundled demo для state-changing и terminal-transition команд, не требуя `text`/`stateChange` от валидной команды перехода, восстановить полный Playwright gate и доказать, что fixture round-trip сохраняет полный terminal set per plan: Verification Strategy / Constitution Testing and Quality Gates `(reopened — BUG-001)` · `tests/browser/state-changing-command-authoring.spec.mjs`, `tests/browser/terminal-navigation.spec.mjs`, `tests/browser/fixtures/desktop-bindings.js`, `sessions/demo.json`

## Phase 8: BUG-001 — complete demo session across the native save boundary

**Purpose**: Воспроизвести и устранить ложное отклонение существующей цели `t_demo2`, не ослабляя missing/self-target validation.

### Tests

- [x] **T034** [US1] Добавить падающий regression для полного `sessions/demo.json` через production App/desktop `SaveSession` boundary: на входе и после contract/Wails round-trip присутствуют `t_demo1` и `t_demo2`, ссылка сохраняется и переоткрывается; отдельно сохранить negative assertions для missing/self target · `app_test.go`, `wails_host_test.go`, `internal/session/service_test.go`, `sessions/demo.json`

### Implementation

**⟶ Wait for T034 to fail for the reported reason, then:**

- [x] **T035** [US1] Зафиксировать payload по обе стороны Wails `SaveSession`, определить и исправить подтверждённый master/Wails/native projection либо stale-resource selection, теряющий `t_demo2`; передавать полный candidate document и не ослаблять `domain.ValidateSession` · `frontend/src/master.js`, `frontend/src/desktop-api.js`, `frontend/bindings/`, `desktop_service.go`, `app.go`, `internal/session/service.go`

**Checkpoint**: `T034` проходит после `T035`; затем выполнить переоткрытые `T013` и `T033`, полный gate `T030` и native acceptance `T031`.

## Phase 9: Convergence

- [x] T036 **HIGH** Пройти и зафиксировать на accepted packaged Wails runtime полный native master + controller + two observers journey для approve/reject, 10 повторных посещений без повторного взлома, трёхтерминального LIFO unwind, reconnect во время pending и shutdown/new-broadcast cleanup; не считать browser-fixture coverage или отдельный `sessions/demo.json` save/reopen smoke заменой этому acceptance gate per SC-001–SC-008 / plan: Verification Strategy and interactive acceptance / T031 (partial) · `build/bin/Fallout Terminal.app`, `sessions/demo.json`, `specs/010-terminal-navigation/.spec-context.json`
