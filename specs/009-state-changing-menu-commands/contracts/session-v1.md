# Контракт session JSON version 1

## Совместимое расширение

Поле верхнего уровня `version` остаётся `1`. Новые поля необязательны; существующий документ без них открывается без миграции, и все команды имеют прежнее поведение.

### Изменяющая состояние команда

```json
{
  "id": "n_doors",
  "type": "command",
  "name": "Открыть двери",
  "text": "Двери открыты. Доступ в сектор разрешён.",
  "stateChange": {
    "completedName": "Двери открыты",
    "confirmationText": "Открыть двери? Это действие нельзя повторить."
  }
}
```

Потребитель трактует четыре авторских текста так:

| Назначение | JSON-путь |
|---|---|
| Исходное название | `terminal.root…node.name` |
| Название после выполнения | `terminal.root…node.stateChange.completedName` |
| Запрос мастеру на одобрение | `terminal.root…node.stateChange.confirmationText` |
| Успешный результат | `terminal.root…node.text` |

`stateChange` отсутствует у обычной команды. Оно запрещено у folder и entry.

### Выполненные снимки терминала

```json
{
  "id": "t_security",
  "name": "Терминал охраны",
  "hackLevel": 0,
  "introText": "",
  "root": { "id": "root", "type": "folder", "name": "ROOT", "children": [] },
  "commandStates": {
    "n_doors": {
      "completedName": "Двери открыты",
      "resultText": "Двери открыты. Доступ в сектор разрешён."
    }
  }
}
```

Ключ карты — ID команды только внутри содержащего терминала. Снимок является server-owned: полное frontend-сохранение не может заменить или удалить его, пока команда с тем же ID существует в том же терминале. Он появляется только после approve мастера и успешной атомарной записи. Pending/rejected запросы не сериализуются. Удаление команды удаляет запись в том же атомарном сохранении; явный reset удаляет её через доверенную backend-команду.

## Persistence protobuf

Источник истины известных полей расширяется совместимо:

```proto
message StateChangeConfig {
  string completed_name = 1;
  string confirmation_text = 2;
}

message CommandExecutionState {
  string completed_name = 1;
  string result_text = 2;
}

message CommandContent {
  string text = 1;
  optional StateChangeConfig state_change = 2;
}

message Terminal {
  // existing fields 1..5 unchanged
  map<string, CommandExecutionState> command_states = 6;
}
```

Номера существующих полей не меняются. Сгенерированные типы не редактируются вручную. `internal/session/contract.go` явно отображает protobuf в установленный JSON shape, а `internal/domain/json.go` добавляет `stateChange` и `commandStates` в наборы известных полей, продолжая сохранять остальные unknown fields.

## Значения по умолчанию и ошибки

- Нет `stateChange` — обычная команда.
- Есть `stateChange`, нет ключа в `commandStates` — исходное состояние.
- Есть ключ — выполненное состояние с зафиксированным названием и результатом.
- Пустые/whitespace-only обязательные тексты, превышение действующих лимитов, неизвестный command ID, неверный тип узла или снимок без настройки делают документ невалидным.
- Неизвестные совместимые JSON-поля сохраняются при чтении/записи.
- Ошибка validation, contract roundtrip, temp write, sync, close или rename не устанавливает новую активную session revision; прежний файл остаётся источником истины.

## Порядок и конкурентность

Все полные сохранения и ID-адресованные мутации используют один session epoch/revision pipeline. Каждая принятая мутация получает более новую document revision; вызвавшая сторона завершает успехом только когда эта либо более новая совместимая ревизия долговечно записана. Merge полного сохранения сохраняет канонические снимки живых команд и очищает снимки удалённых узлов.
