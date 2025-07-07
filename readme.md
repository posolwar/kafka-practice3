# Система обработки сообщений с фильтрацией и блокировкой

## Описание проекта

Этот проект представляет собой систему обработки сообщений, реализованную с использованием Kafka и Goka. Система выполняет следующие задачи:
1. **Фильтрация сообщений**:
   - Проверяет сообщения на наличие запрещённых слов и заменяет их на указанные слова-заменители.
   - Фильтрует сообщения от заблокированных пользователей, чтобы они не доходили до получателя.
2. **Управление блокировками**:
   - Позволяет пользователям блокировать или разблокировать других пользователей.
   - Хранит списки заблокированных пользователей в персистентном хранилище.
3. **Отправка и получение сообщений**:
   - Пользователи могут отправлять сообщения через CLI-инструмент.
   - Полученные сообщения доступны через HTTP API.

### Структура проекта

#### Основные пакеты и файлы
- **Пакет `user`**:
  - `user.go`: Определяет структуру `User` для представления пользователя (`ID`, `Name`).
  - `block-user-emitter.go`: Отправляет запросы на блокировку/разблокировку в топик `blocked-users-stream`.
  - `block-user-processor.go`: Обрабатывает запросы на блокировку/разблокировку, сохраняет списки заблокированных пользователей в хранилище `blocked-users`. Предоставляет HTTP API для получения списка блокировок (`/block-list/{user_id}`).
- **Пакет `message`**:
  - `fitler-word.go`: Обрабатывает сообщения из топика `pre-filtered-messages-stream`, заменяет запрещённые слова и отправляет результат в `filtered-messages-stream`.
  - `message-sender.go`: Отправляет сообщения в топик `messages-stream`.
  - `message-receiver.go`: Сохраняет отфильтрованные сообщения из `filtered-messages-stream` в персистентное хранилище и предоставляет HTTP API для получения сообщений (`/messages/{user_id}`).
  - `RunBlockFilter` (в `fitler-word.go`): Фильтрует сообщения из `messages-stream`, пропуская только сообщения от незаблокированных пользователей в `pre-filtered-messages-stream`.
- **Пакет `censore`**:
  - `new-filter-word.go`: Отправляет запросы на добавление запрещённых слов в топик `filtered-messages`.
- **Командные утилиты**:
  - `block.go`: CLI-инструмент для отправки запросов на блокировку/разблокировку.
  - `send-message.go`: CLI-инструмент для отправки сообщений.
- **Инфраструктура**:
  - `docker-compose.yaml`: Разворачивает кластер Kafka с тремя брокерами, Kafka UI и Schema Registry.
  - `Taskfile.yml`: Содержит команды для управления инфраструктурой (создание/удаление топиков, запуск/остановка контейнеров, добавление слов в фильтр, блокировка пользователей).

#### Логика работы
1. **Поток сообщений**:
   - Сообщения отправляются в топик `messages-stream` через `send-message.go`.
   - Процессор `RunBlockFilter` проверяет, не заблокирован ли отправитель (`FromUserID`) получателем (`ToUserID`), используя данные из хранилища `blocked-users`. Если отправитель не заблокирован, сообщение передаётся в `pre-filtered-messages-stream`.
   - Процессор `RunWordFilter` обрабатывает сообщения из `pre-filtered-messages-stream`, заменяет запрещённые слова и отправляет результат в `filtered-messages-stream`.
   - Процессор `RunMessageReceiver` сохраняет сообщения из `filtered-messages-stream` в персистентное хранилище и делает их доступными через HTTP API (`/messages/{user_id}`).

2. **Блокировка пользователей**:
   - Запросы на блокировку/разблокировку отправляются через `block.go` в топик `blocked-users-stream`.
   - Процессор `RunBlockProcess` обновляет список заблокированных пользователей в хранилище `blocked-users`.
   - Список блокировок доступен через HTTP API (`/block-list/{user_id}`).

3. **Фильтрация слов**:
   - Запрещённые слова добавляются через CLI-команду (`task filter-add:badword:goodword`) в топик `filtered-messages`.
   - Процессор `RunWordFilter` использует хранилище `filtered-messages` для замены запрещённых слов в сообщениях.

#### Топики Kafka
- **Потоки**:
  - `messages-stream`: Входящие сообщения.
  - `pre-filtered-messages-stream`: Сообщения от незаблокированных юзеров.
  - `filtered-messages-stream`: Сообщения после фильтрации слов.
  - `blocked-users-stream`: Запросы на блокировку/разблокировку.
  - `filter-words-stream`: Запросы на добавление запрещённых слов.
- **Таблицы**:
  - `blocked-users-table`: Хранит списки заблокированных пользователей.
  - `filtered-messages-table`: Хранит списки запрещённых слов.

## Инструкция по запуску проекта

### Требования
- Установлены **Docker** и **Docker Compose**.
- Установлен **Go** (версия 1.18 или выше).
- Установлен **Task** (для выполнения команд из `Taskfile.yml`): `go install github.com/go-task/task/v3/cmd/task@latest`.

### Шаги для запуска
1. **Клонируйте репозиторий** 
2. **Запустите инфраструктуру**:
   ```bash
   task start
   ```
   Это развернёт Kafka, Kafka UI (доступно на `http://localhost:8080`) и Schema Registry.
3. **Создайте топики**:
   ```bash
   task create-topics
   ```
   Это создаст все необходимые топики с 3 партициями и коэффициентом репликации 2.
4. **Запустите основной процесс**:
   ```bash
   go run ./cmd/chat/main.go
   ```
   Это запустит все процессоры и HTTP API:
   - `http://localhost:9091/block-list/{user_id}`: список заблокированных пользователей. Замените user_id на идентификатор юзера.
   - `http://localhost:9092/messages/{user_id}`: список сообщений для пользователя. Замените user_id на идентификатор юзера.

### Остановка проекта
```bash
task stop
```
Это остановит и удалит все контейнеры и тома.

### Удаление топиков (при необходимости)
```bash
task remove-topics
```

## Инструкция по тестированию

### Подготовка
1. Убедитесь, что инфраструктура запущена (`task start`) и топики созданы (`task create-topics`).
2. Запустите процессоры (`go run ./cmd/chat/main.go`).
3. Откройте Kafka UI (`http://localhost:8080`) для мониторинга топиков.

### Тестовые сценарии

#### 1. Добавление запрещённых слов
Добавьте слово `badword`, которое будет заменяться на `goodword`:
```bash
task filter-add:badword:goodword
```
Проверьте в Kafka UI, что сообщение появилось в топике `filtered-messages`.

#### 2. Блокировка пользователя
Заблокируйте пользователя `client1` для `client2`:
```bash
task user-block:client2:client1:true
```
Проверьте список блокировок:
```bash
curl http://localhost:9091/block-list/client2
```
Ожидаемый результат:
```json
{"BlockedUsers":["client1"]}
```

#### 3. Отправка сообщения от заблокированного пользователя
Отправьте сообщение от `client1` к `client2`:
```bash
go run ./cmd/message/message-sender.go -from_user_id=client1 -to_user_id=client2 -content="тестирование badword"
```
- В Kafka UI проверьте, что сообщение появилось в `messages-stream`
- Убедитесь, что сообщения нет в ответе API, юзер заблокирован:
  ```bash
  curl http://localhost:9092/messages/client2
  ```
  Ожидаемый результат:
  ```json
  {"messages":[]}
  ```

#### 4. Отправка сообщения от незаблокированного пользователя
Разблокируйте `client1` для `client2`:
```bash
task user-block:client2:client1:false
```
Отправьте сообщение от `client1` к `client2`:
```bash
go run ./cmd/message/message-sender.go -from_user_id=client1 -to_user_id=client2 -content="тестирование badword"
```
- В Kafka UI проверьте, что сообщение появилось в `messages-stream`, затем в `pre-filtered-messages-stream`, и в `filtered-messages-stream` с заменённым словом (`badword` → `****`).
- Проверьте сообщения для `client2`:
  ```bash
  curl http://localhost:9092/messages/client2
  ```
  Ожидаемый результат:
  ```json
  {"messages":[{"from_user_id":"client1","to_user_id":"client2","content":"This is a **** test"}]}
  ```

#### 5. Отправка сообщения без запрещённых слов
Отправьте сообщение от `client3` к `client2`:
```bash
go run ./cmd/send-message/main.go -from_user_id=client3 -to_user_id=client2 -content="Hello, this is a clean message"
```
- В Kafka UI проверьте, что сообщение прошло через все топики без изменений.
- Проверьте сообщения:
  ```bash
  curl http://localhost:9092/messages/client2
  ```
  Ожидаемый результат (включая предыдущее сообщение):
  ```json
  {"messages":[{"from_user_id":"client1","to_user_id":"client2","content":"This is a **** test"},{"from_user_id":"client3","to_user_id":"client2","content":"Hello, this is a clean message"}]}
  ```

### Тестовые данные
1. **Добавление запрещённых слов**:
   ```bash
   task filter-add:badword:****
   task filter-add:curse:****
   ```
2. **Блокировка пользователей**:
   ```bash
   task user-block:client2:client1:true
   task user-block:client2:client3:false
   ```
3. **Отправка сообщений**:
   ```bash
   go run ./cmd/send-message/main.go -from_user_id=client1 -to_user_id=client2 -content="This is a badword test"
   go run ./cmd/send-message/main.go -from_user_id=client3 -to_user_id=client2 -content="Hello, clean message"
   ```

## Замечания
- **Очистка сообщений**: Текущая реализация накапливает сообщения в хранилище. Для продакшен-системы рекомендуется добавить механизм удаления старых сообщений.
- **Безопасность**: HTTP API не включает аутентификацию. В продакшене добавьте проверку прав доступа.
- **Логирование**: Все действия логируются с использованием `slog` для упрощения отладки.