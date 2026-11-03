# Data management

Файловая база данных на Go: однотабличные БД в собственных файлах `.db`,
файловые индексы (`hash` и `B+Tree`), сервисный слой, REST API и
веб-интерфейс.

## Требования и запуск

- Go 1.26+.
- Свободный порт 8080.

```bash
go mod tidy
go run ./cmd/app          # открывает http://localhost:8080
go run ./cmd/seed -reset  # демо-данные (опционально)
```

Файлы БД создаются в `./data/` при первом запуске.

## Описание предметной области

Предметная область - онлайн-платформа поиска работы.
Она охватывает публикацию вакансий, создание резюме, отклики, переписку,
отзывы, модерацию вакансий и верификацию работодателей. Модель
нормализована до 3 нормальной формы и включает 11 сущностей, каждая в отдельном `.db`-файле.

- `seekers` - соискатели: id, email (unique), password_hash, phone,
  full_name, birth_date, city, status.
- `employers` - работодатели: id, email (unique), password_hash, phone,
  company_name, inn (unique), website, verification_status, city,
  moderator_id.
- `moderators` - модераторы: id, email (unique), full_name.
- `resumes` - резюме: id, seeker_id (FK), position, desired_salary,
  category, employment_type, schedule, status, work_experience, education.
- `vacancies` - вакансии: id, employer_id (FK), title, city, description,
  salary_from, salary_to, required_experience, category, employment_type,
  schedule, status, moderator_id.
- `skills` - справочник навыков: id, name (unique).
- `resume_skills`, `vacancy_skills` - связи M:N резюме ↔ навыки и
  вакансии ↔ навыки: id, resume_id/vacancy_id (FK), skill_id (FK).
- `applications` - отклики: id, seeker_id, resume_id, vacancy_id (FK),
  initiator, cover_letter, status, created_at.
- `messages` - сообщения: id, author_type, author_id, text, sent_at, is_read.
- `reviews` - отзывы: id, seeker_id, employer_id (FK), rating, text,
  is_published, created_at, moderator_id.

Связи: `seekers 1:N resumes`, `employers 1:N vacancies`,
`resumes N:M vacancies` через `applications`, `resumes N:M skills` через
`resume_skills`, `vacancies N:M skills` через `vacancy_skills`,
`moderators 1:N` с `vacancies`, `employers`, `reviews`.
`seekers N:M employers` через `messages` и `reviews`.

Типы полей: `int64`, `str`, `date` (unix), `bool`. В каждой таблице не
менее 4 полей разных типов и ровно 1 PK `id` с автоинкрементом. Уникальные
и индексируемые поля помечены `FlagHash` или `FlagBTree`.

Реализованы каскадные удаления (`internal/service/cascade.go`). Удаление
соискателя уносит его резюме, отклики, связи навыков, сообщения и отзывы;
удаление работодателя - его вакансии, отклики, связи, отзывы и сообщения.

## Возможности

- CRUD: добавление с проверкой уникальности по PK и unique-полям,
  редактирование, удаление по PK и по значению поля (все совпадения),
  поиск по значению поля (все совпадения), поиск по диапазону для
  BTree-полей.
- Каскадные удаления, backup/restore в zip, экспорт в `.xlsx`,
  очистка и удаление всех БД, веб-интерфейс.

## Архитектура

| Слой | Пакет | Назначение |
|---|---|---|
| Web | `internal/web` | REST API и GUI |
| Service | `internal/service` | Бизнес-логика, FK, каскады |
| Engine | `internal/engine` | CRUD, согласованность индексов |
| Storage | `internal/storage` | Слоты, buffer pool |
| Index | `internal/index` | Файловые `hash` и `btree` |
| Schema | `internal/schema` | Схемы, кодирование |
| Backup, Export | `internal/backup`, `internal/export` | zip и xlsx |

Работа с БД ведётся через файлы (`pread`/`pwrite`); массивы в памяти
используются только для отрисовки таблицы на экране.

## Формат `.db` и индексы

Файл - заголовок (`MYDB`, версия, recordSize, recordCount, freeHead,
описание полей) и слоты по `recordSize` байт: `used(1) + nextFree(8) +
payload`. Свободные слоты в LIFO-списке, при удалении переиспользуются;
PK монотонно растёт и не переиспользуется. Buffer pool - LRU, 64 страницы
по 16 КБ.

| Индекс | Файл | Сложность |
|---|---|---|
| PK hash | `<db>.pk.hash` | O(1) |
| Field hash (`FlagHash`) | `<db>.<field>.hash` | O(1) + O(k) |
| Field B+Tree (`FlagBTree`) | `<db>.<field>.btree` | O(log n) / O(log n + k) |

Hash - FNV-1a, бакеты с удвоением при переполнении; служебные значения
`0`, `""`, `false` не индексируются. B+Tree - страницы, big-endian ключи,
составной ключ `(key, slotID)`, forward-scan по `leaf.extra` для поиска
дубликатов и диапазонов.

## REST API

| Метод | Endpoint | Операция |
|---|---|---|
| GET / POST | `/api/{entity}` | Список / добавить |
| GET / PUT / DELETE | `/api/{entity}/{id}` | По PK |
| GET | `/api/{entity}/search?field=&value=` | Поиск по значению |
| DELETE | `/api/{entity}/by-field?field=&value=` | Удалить все совпадения |
| GET | `/api/{entity}/range?field=&lo=&hi=` | Диапазон (BTree) |
| GET / POST | `/api/backup`, `/api/restore` | Резервная копия |
| GET | `/api/export/xlsx?entity=` | Экспорт в xlsx |
| POST | `/api/db/clear`, `/api/db/delete`, `/api/db/save` | Управление БД |

`{entity}` - `vacancies`, `resumes`, `applications`, `seekers`,
`employers`, `skills`, `reviews`, `messages`, `moderators`,
`resume_skills`, `vacancy_skills`. Поиск и удаление по значению
выполняются на сервере через `engine.FindBy` / `engine.DeleteBy`;
клиентская фильтрация массива не используется.

## Анализ сложности

| Операция | Сложность |
|---|---|
| `Add` | O(log n) амортизированно |
| `GetByPK` | O(1) |
| `FindBy` по hash-полю | O(1) + O(k) |
| `FindBy` по BTree-полю | O(log n) + O(k) |
| `FindBy` по неиндексированному | O(n) |
| `FindByRange` | O(log n + k) |
| `Update` | O(log n) |
| `DeleteByPK` | O(1) |
| `DeleteBy` по hash-полю | O(1) + O(k) |
| `DeleteBy` по BTree-полю | O(log n) + O(k) |
| `DeleteBy` по неиндексированному | O(n) + O(k) |

Добавление логарифмическое за счёт вставки в хэш-таблицы и B+Tree
индексированных полей. Поиск по PK и по hash-полям константный - одно
обращение к хэш-таблице и одно чтение слота. Поиск по BTree-полям
логарифмический - спуск от корня до листа. Поиск по неиндексированным
полям линейный - чтение всех слотов. Диапазонный поиск даёт
O(log n + k) благодаря спуску по дереву и последовательному обходу
листьев по связному списку; эта операция недостижима через hash-индекс.

Хэш-индекс при переполнении бакета перестраивается, но удваивает размер,
поэтому амортизированная сложность вставки остаётся O(1). 

## Временная статистика

Intel Core i5-7200U @ 2.50 GHz, Windows, Go 1.26, флаги
`-benchmem -benchtime=2s`. Подготовка данных исключена через
`b.ResetTimer()`.

**Add:**

| N | ns/op | B/op | allocs/op |
|---|---|---|---|
| 1 000 | 136 235 | 12 921 | 25 |
| 10 000 | 144 700 | 26 902 | 29 |
| 100 000 | 154 373 | 44 063 | 26 |

Рост N в 100 раз → рост времени на 13 %, кривая плоская.

**GetByPK:**

| N | ns/op | B/op | allocs/op |
|---|---|---|---|
| 1 000 | 8 172 | 2 590 | 12 |
| 10 000 | 8 645 | 2 599 | 13 |

Разница 6 % - шум и кэш-эффекты, сложность константная.

Полные замеры `FindBy`, `FindByRange`, `Update` и `DeleteBy` - в файле
`bench.txt`, полученном командой:

```bash
go test -bench=. -benchmem -benchtime=2s -run='^$' ./internal/engine/
```

В PowerShell:

```powershell
go test "-bench=." "-benchmem" "-benchtime=2s" "-run=NoSuchTest" "./internal/engine/"
```


## Тесты и бенчмарки

```bash
go test ./...
```

Тесты покрывают: автоинкремент, дубликаты PK и unique, обновление,
удаление по PK и по значению, восстановление `nextID`; FK и каскады;
alloc/write/read/free и переиспользование слота; кодирование и заголовок;
hash-индекс; B+Tree (2000 записей, дубликаты, диапазон, сохранение на диск).

`internal/engine/bench_test.go` содержит 11 бенчмарков: `Add`, `GetByPK`,
`FindByHash`, `FindByBTree`, `FindByScan`, `FindByRangeBTree`,
`UpdateByPK`, `DeleteByPK`, `DeleteByHash`, `DeleteByBTree`,
`DeleteByScan` - на N = 10³, 10⁴, 10⁵.

## Структура

```
file_db/
├── cmd/app, cmd/seed
├── internal/{schema, storage, index, engine, service, web, backup, export}
└── data/   *.db / *.hash / *.btree
```

## Ограничения

1. B+Tree не ребалансируется при удалении: пустые листья остаются, файл
   индекса может расти.
2. Буферный пул фиксирован - 64 × 16 КБ.
3. Все записи сериализуются мьютексом `Engine.mu`.
