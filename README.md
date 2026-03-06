# Gophkeeper — Менеджер паролей

## Обзор

Gophkeeper — клиент-серверная система для безопасного хранения приватных данных:
логинов/паролей, текстовых заметок, бинарных файлов и данных банковских карт.

## Архитектурные решения

### Протокол взаимодействия
- gRPC с Opaque API (Protocol Buffers v3).
- TLS-шифрование канала (сертификаты в `./cert`).
- Максимальный размер сообщения: 64 МБ.
- Два gRPC-сервиса: `AuthService` (регистрация/вход) и `DataService` (CRUD данных).

### Аутентификация и авторизация
- JWT-токены с временем жизни 30 минут (настраивается).
- Токен передаётся в gRPC metadata (`authorization: Bearer <token>`).
- При каждом успешном запросе к `DataService` токен продлевается — новый токен
  возвращается в response metadata.
- Токен хранится в зашифрованном файле сессии на клиенте.

### Шифрование данных
- Данные шифруются на клиенте перед отправкой на сервер.
- Используется AES-256-GCM.
- Ключ шифрования выводится из мастер-пароля через Argon2id с уникальным salt.
- Мастер-пароль никогда не сохраняется на диск — запрашивается интерактивно
  непосредственно перед отправкой запроса.

### Хранение данных (сервер)
- В качестве хранения используется PostgreSQL
- Миграции через `golang-migrate`.
- Миграции запускаются при запуске сервера, но можно запустить отдельно (`make migrate-up`)

### Клиент (CLI)
- Cobra + интерактивный ввод
- Пароли вводятся через `golang.org/x/term`.
- Команды: `register`, `login`, `add`, `get`, `list`, `delete`, `version`.
- `add` — полностьвю интерактивный режим.
- `get` / `list` / `delete` — через флаги (`--id`, `--title`, `--type`).
- Retry при сетевых ошибках (количество настраивается в конфиге).
- Конфигурация через Viper (YAML-файл).
- Логирование через slog (stderr/stdout или файл).

### Типы данных
| Код           | Описание                | Ввод                              | Вывод               |
|---------------|-------------------------|-----------------------------------|---------------------|
| `credentials` | Логин/пароль            | Интерактивный                     | В терминал          |
| `card`        | Банковская карта        | Интерактивный                     | В терминал          |
| `text`        | Текстовые данные        | Интерактивный или из файла (путь) | В терминал или файл |
| `binary`      | Бинарные данные         | Из файла (путь)                   | В файл              |

### Метаданные
Произвольный текст, передаётся как строка, хранится в БД как `TEXT`.

## Поток данных

### Регистрация

User → CLI (register) → gRPC AuthService.Register → hash password → DB insert → JWT → Client session file

### Вход

User → CLI (login) → gRPC AuthService.Login → verify password → JWT → Client session file

### Добавление данных

User → CLI (add, интерактивный) → ввод полей → ввод мастер-пароля →
→ Argon2id(master-password) → AES-GCM encrypt → gRPC DataService.Add →
→ Auth interceptor (JWT) → DB insert → response

### Получение данных

User → CLI (get --id/--title) → ввод мастер-пароля →
→ gRPC DataService.Get → Auth interceptor → DB select → encrypted blob →
→ AES-GCM decrypt(master-password) → display / save to file

## Сборка и запуск

```bash
# Генерация сертификатов (self-signed, поэтому только для разработки)
make cert

# Генерация proto
make proto

# Сборка
make build

# Запуск миграций (опционально или накатятся при запуске сервера)
make migrate-up DSN="postgres://user:pass@localhost:5432/gophkeeper?sslmode=disable"

# Запуск сервера
./bin/gophkeeper-server

# Запуск клиента
./bin/gophkeeper-client register
./bin/gophkeeper-client login
./bin/gophkeeper-client add
./bin/gophkeeper-client get --title="my secret" # или --id=1 для выдачи по id, один из флагов должен быть заполнен
./bin/gophkeeper-client list # или --type=credentials для вывода только определенного типа 
./bin/gophkeeper-client delete --title="my secret" # или --id=1 для выдачи по id, один из флагов должен быть заполнен
./bin/gophkeeper-client version
```

## Примеры использования

### Регистрация

```markdown
$ ./bin/gophkeeper-client register
Enter username: demo
Enter user password: ******
User registered successfully. Session saved.
```

### Добавление логина/пароля

```markdown
$ ./bin/gophkeeper-client add     
Enter data type (credentials/card/text/binary): credentials
Enter entry title: demo creds      
Enter metadata (optional): foo=bar
Enter data login: test_user
Enter data password: ******
Enter master password to encrypt data: *******
Data added successfully (ID: 4).
```

### Получение списка данных для пользователя
```markdown
$ ./bin/gophkeeper-client list
ID         Type            Title                          Created_at                     Metadata
------------------------------------------------------------------------------------------------------------
4          credentials     demo creds                     2026-03-04 23:52:07            foo=bar
```

### Получение данных

#### Пара логин/пароль

```markdown
$ ./bin/gophkeeper-client get --id=4
Enter master password to decrypt the data: *******

--- RESULT ---
Type: credentials
Title: demo creds
Metadata: foo=bar
Created at: 2026-03-04 23:52:07

--- CREDENTIALS DATA ---
Login: test_user
Password: test_pass
```

#### Банковская карта

```markdown
$ ./bin/gophkeeper-client get --id=6
Enter master password to decrypt the data: ******

--- RESULT ---
Type: card
Title: demo card
Metadata: card metadata
Created at: 2026-03-05 00:09:35

--- CARD DATA ---
Number: 1111 2222 3333 4444
Holder: John Doe
Expiry date: 01/02
CVV: 123
```

#### Произвольный текст

Вывод текста в терминал:
```markdown
% ./bin/gophkeeper-client get --id=5                 
Enter master password to decrypt the data: 
Save to file (enter path or press `Enter` to show text in console):

--- RESULT ---
Type: text
Title: demo text
Metadata: text for demo
Created at: 2026-03-05 00:04:09

--- TEXT DATA ---
Lorem ipsum dolor sit amet
```

Сохранение текста в файл:
```markdown
% ./bin/gophkeeper-client get --title="demo text"
Enter master password to decrypt the data: *******
Save to file (enter path or press `Enter` to show text in console): demo.txt

--- RESULT ---
Type: text
Title: demo text
Metadata: text for demo
Created at: 2026-03-05 00:04:09

Text data saved to file: demo.txt
```

#### Бинарные данные

Аналогично текстовым, но заполнение возможно только из файла и получение обратно тоже только в файл.

### Удаление данных

```markdown
$ ./bin/gophkeeper-client delete --title="demo creds"
Are you sure to delete entry? (y/n): y
Entry deleted successfully.
```