# Модель данных: команды, изменяющие состояние пунктов меню

## Обзор владения

```text
Session (долговечный документ)
└── Terminal[id]
    ├── Root → ContentNode[id]
    │   └── command: authored name/text + optional StateChangeConfig
    └── CommandStates[commandID] → CommandExecutionState

MasterCoordinationState / LiveBroadcast (только runtime)
├── PendingCommandExecution?
└── CommandExecutionPresentation? → pending | rejected

PublicLiveState
├── Tree → эффективные authored/frozen названия и результаты
└── CommandExecutionPresentation? → общий экран игроков
```

Авторская конфигурация и выполненные снимки являются полями session JSON version 1. Ожидающий запрос, решение диалога, экран отказа, навигация, подключения, hacking и broadcast lifecycle остаются процессным состоянием и в JSON не попадают.

## Долговечные сущности

### `StateChangeConfig`

Необязательная авторская настройка только для узла типа `command`.

| Поле | JSON | Назначение |
|---|---|---|
| CompletedName | `completedName` | Название пункта после первого успешного выполнения |
| ConfirmationText | `confirmationText` | Приватный текст диалогового запроса мастеру |

Два остальных обязательных авторских текста уже принадлежат команде:

- `ContentNode.Name` / `name` — исходное название;
- `ContentNode.Text` / `text` — текст успешного выполнения.

Отсутствие `stateChange` означает обычную команду. Пустой объект не считается валидной настройкой. `ConfirmationText` никогда не входит в публичную player-проекцию.

### `CommandExecutionState`

Долговечный неизменяемый снимок первого одобренного и успешно сохранённого выполнения.

| Поле | JSON | Назначение |
|---|---|---|
| CompletedName | `completedName` | Зафиксированное отображаемое название |
| ResultText | `resultText` | Зафиксированный успешный результат |

Снимок создаётся из актуальных `StateChangeConfig.CompletedName` и `ContentNode.Text` после approve мастера. Последующее редактирование конфигурации не изменяет снимок.

### `Terminal.CommandStates`

Необязательное отображение `commandStates`, где ключ — стабильный ID команды внутри данного терминала, а значение — `CommandExecutionState`. Полная идентичность записи равна `(Terminal.ID, commandID)`.

Отсутствующая карта или отсутствующий ключ означает исходное состояние. Пустая карта при сериализации опускается. Терминал не может содержать запись для команды другого терминала, отсутствующего узла, папки, записи или обычной команды.

## Runtime-сущности

### `PendingCommandExecution`

Единственный ожидающий запрос текущего broadcast.

| Поле | Назначение |
|---|---|
| RequestID | Сгенерированный сервером корреляционный ID для решения мастера |
| BroadcastID | Контекст, в котором выбор был принят |
| TerminalID | Активный терминал запроса |
| CommandID | Стабильный ID выбранной команды |
| ConfirmationText | Приватный снимок prompt для текущего диалога мастера |

Запрос создаётся только после проверки active controller и текущих broadcast/terminal ID. Он не хранит обязательную ссылку на живое соединение инициатора: disconnect контроллера его не отменяет. End broadcast, terminal switch и shutdown удаляют запрос без выполнения и persistence.

### `CommandExecutionPresentation`

Общая публичная runtime-проекция текущего запроса.

| Phase | Отображение | Допустимые shared player actions |
|---|---|---|
| отсутствует | Обычный терминал | По прежним правилам |
| `PENDING` | «Выполняется запрос» | Навигация и остальные terminal-команды отклоняются |
| `REJECTED` | «Запрос отклонён» | Контроллеру доступен возврат в сохранённое прежнее меню |

Проекция содержит command ID только для корреляции отображения и не содержит `ConfirmationText`, completed title либо success text до approve.

### `EffectiveCommand`

Вычисляемая часть публичного дерева, не отдельная долговечная сущность.

| Durable состояние | Название узла | Текст команды |
|---|---|---|
| Обычная | `name` | `text` |
| Stateful, исходная | `name` | не показывается как результат до approve |
| Stateful, выполненная | снимок `completedName` | снимок `resultText` |

## Связи и инварианты

1. `Session.Version` остаётся равным `1`.
2. ID терминала уникален в сессии; ID узла уникален в дереве терминала — существующие проверки формируют устойчивую пару идентичности.
3. `stateChange` допустим только у `type: "command"`.
4. Для stateful-команды `name`, `text`, `completedName` и `confirmationText` после проверки на whitespace содержат хотя бы один непробельный символ.
5. `name` и `completedName` соблюдают существующий предел имени; `text`, `confirmationText` и `resultText` — предел содержимого команды.
6. Каждый ключ `commandStates` разрешается ровно в один существующий узел команды с `stateChange` в том же терминале.
7. Поля долговечного снимка непусты и не вычисляются заново при чтении.
8. Неизвестные совместимые JSON-поля сохраняются на уровнях session, terminal и content node; новые известные поля не могут одновременно существовать в `Extra`.
9. Две команды с одинаковыми названиями независимы благодаря разным ID.
10. Rename/reorder в пределах терминала не меняют ключ состояния. Удаление узла удаляет ключ. Узел с новым ID не наследует старую запись.
11. Снятие state-change настройки с выполненной команды требует предварительного reset либо отклоняется; completed snapshot не может стать сиротой.
12. На один broadcast существует не более одного `PendingCommandExecution`; повторные выборы при pending не создают новый request/dialog.
13. Private decision принимается только для точного текущего RequestID и совпадающих broadcast/terminal/command; stale decision ничего не меняет.
14. `PENDING` существует тогда и только тогда, когда существует pending request. `REJECTED` не содержит pending request и очищается возвратом контроллера или завершением контекста.

## Состояния и переходы

```text
ORDINARY
  player выбирает → обычный показ text
  мастер включает stateChange → INITIAL

INITIAL
  active controller выбирает → PENDING (без persistence)
  повторный/конкурентный выбор → PENDING (no-op/reject, без второго dialog)

PENDING
  controller disconnect → PENDING
  master reject/close dialog → REJECTED
  master approve + save success → COMPLETED
  master approve + save failure → INITIAL + безопасная ошибка мастеру/контроллеру
  end broadcast/switch/shutdown → INITIAL без восстановления запроса

REJECTED
  controller возвращается → INITIAL в прежнем menu path
  end broadcast/switch/shutdown → INITIAL

COMPLETED
  повторный выбор → показать frozen resultText без dialog/save
  редактировать authored config → COMPLETED со старым snapshot
  rename/reorder в терминале → COMPLETED
  master reset-one/reset-terminal → INITIAL
  удалить команду → удалена вместе со snapshot
```

Только `PENDING → COMPLETED`, `COMPLETED → INITIAL` и удаление затрагивают server-owned durable state. `INITIAL/PENDING/REJECTED` являются runtime-переходами и никогда не сериализуются в session JSON.

## Транзакция player request

1. Координатор проверяет recognition handle, request fingerprint, broadcast, terminal, active controller, node ID и актуальный durable snapshot.
2. Ordinary-команда использует прежний navigation path; completed-команда показывает frozen result без записи.
3. Для initial stateful-команды координатор создаёт один `PendingCommandExecution`, сохраняет прежний menu path и увеличивает live/coordination revision.
4. Одна принятая compound publication показывает всем игрокам `PENDING`, а private coordination projection предоставляет мастеру request ID и prompt.
5. Пока запрос существует, остальные navigation/command actions отклоняются как conflict; disconnect инициатора не меняет запрос.

## Транзакция решения мастера

### Approve

1. Private boundary проверяет RequestID и решение `APPROVE`; coordinator повторно сверяет broadcast, terminal, command и отсутствие durable snapshot.
2. На рабочей копии строится `CommandExecutionState`.
3. Session service выделяет следующую document revision, валидирует документ и атомарно заменяет выбранный файл.
4. Только после durability coordinator очищает pending, устанавливает effective completed tree/navigation и публикует одну accepted revision всем игрокам и master coordination.
5. Master получает новый session document через revision-aware private event.
6. Ошибка сохранения очищает ожидание, оставляет команду initial, не показывает completed title/result и доставляет безопасную ошибку мастеру и текущему контроллеру.

### Reject или close dialog

1. Private boundary разрешает текущий RequestID как `REJECT`.
2. Coordinator очищает pending без session write и публикует `REJECTED` всем игрокам.
3. Возврат контроллера очищает rejection presentation и восстанавливает сохранённый menu path.

## Транзакции сброса

- `reset-one` удаляет ровно snapshot указанной команды; отсутствие snapshot — идемпотентный no-op без записи.
- `reset-terminal` удаляет все и только `commandStates` выбранного терминала одним document save.
- Для активного терминала успешная запись сопровождается одной runtime-ревизией с полной новой проекцией; для неактивного новая проекция появится при следующей активации.
- Frontend cancel подтверждения reset не вызывает private method и ничего не меняет.

## Слияние полного авторского сохранения

Поступивший от master frontend полный документ не является источником истины для `commandStates`.

1. Сопоставить терминалы входящего документа с каноническими терминалами по `Terminal.ID`.
2. Для каждого сохранившегося command ID перенести канонический snapshot независимо от входящего `commandStates`.
3. Не переносить snapshots удалённых команд и команд, перемещённых в другой терминал.
4. Применить новые authored/unknown compatible поля входящего документа.
5. Валидировать итоговый документ и поставить его в общую очередь с новой revision.

Pending request при author Save не сериализуется и остаётся в coordinator, если его terminal/broadcast context не завершён. Изменение или удаление ожидающей команды должно либо отклоняться до решения, либо сначала отменять pending через ту же coordinator-транзакцию; stale approve никогда не применяется к изменённой идентичности.

## Клонирование и гидратация

- Все session/runtime/public значения, покидающие владельца состояния, являются detached copies.
- Активация терминала получает authored config и `commandStates` из текущей backend session, а не доверяет frontend payload как источнику server-owned state.
- Reconnect player получает текущую `CommandExecutionPresentation`; reload master получает текущий pending в private `CoordinationState` и может повторно открыть тот же диалог без второго запроса.
- End broadcast очищает pending/rejected runtime projection, но не `commandStates`; новый broadcast строит effective tree из долговечного документа.
