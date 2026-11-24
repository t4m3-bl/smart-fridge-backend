# SmartFridge: API и отдельный worker уведомлений

## Приложения и контейнеры

- `cmd/smart-fridge-backend` — HTTP API, авторизация, продукты, настройки и подписки. Планировщик в API не запускается.
- `cmd/notification-worker` — отдельное приложение: cron-планировщик и синхронная отправка Web Push.
- `cmd/vapid-keygen` — однократная генерация VAPID-ключей.
- `cmd/push-debug` — отдельный локальный стенд проверки, не включённый в production-запуск.

API и worker используют общий Docker-образ с двумя бинарниками, но работают в разных контейнерах с разными entrypoint.
Контейнер `notification-worker` не публикует порты, не запускает HTTP API и не требует JWT, OAuth или S3. Ему нужны PostgreSQL и настройки Web Push.
Приватный VAPID-ключ передаётся только worker; API получает публичный ключ.
Основной фронтенд не изменён. Для него ещё нужна интеграция PushManager и обработчиков Service Worker. Для ручной проверки сервера используйте `push-debug`.

## Как работает расписание

У каждого подписанного пользователя есть cron-расписание из `settings.notification_time` и `settings.timezone`.
Например, для 09:00 в Екатеринбурге worker строит `CRON_TZ=Asia/Yekaterinburg 0 0 9 * * *` (с первым полем секунд).
`robfig/cron/v3` вычисляет ближайший запуск; один цикл worker ждёт его таймером и последовательно отправляет сводки пользователям и всем их активным подпискам.
Отдельные goroutine на получателя не создаются.

`SMART_FRIDGE_PUSH_POLL_INTERVAL` теперь задаёт интервал перечитывания настроек (30 секунд по умолчанию), а не период проверки «пора ли отправлять».
Неизменённое расписание сохраняет ближайший запуск. Изменение времени/timezone пересчитывает его; отключение уведомлений или отсутствие подписок убирает расписание.
После изменения настроек нужно оставить достаточно времени до запуска, чтобы worker успел перечитать их.
При отправке дополнительно проверяются настройки и отзыв подписки.

Очереди, таблиц заданий, дат последних запусков, отложенных повторов и догоняющей рассылки нет.
После перезапуска cron рассчитывается от текущего времени; пропущенное во время простоя срабатывание не выполняется.
Ошибки одной подписки записываются в лог, затем worker продолжает отправку остальным. 404/410 отзывают подписку.
Для этой реализации запускайте **один экземпляр worker**: независимые экземпляры cron будут дублировать отправку.
Смена времени на другое будущее время может привести к дополнительной сводке в тот же день. Поведение переходов часового пояса следует cron-расписанию.

## Содержимое сводки

- Непросроченные продукты, попадающие в первые два индикатора пользователя после сортировки по `maxDays` (безлимитный `NULL` в конце). Цвет не имеет фиксированной роли.
- Дни считаются по календарю в timezone аккаунта, по `effective_expires_at` с учётом вскрытия.
- Дата истечения сегодня включается до конца локального дня; уже просроченные исключаются.
- Пустой список не отправляется. В payload помещается до 8 названий по 40 символов и число остальных продуктов.
- По умолчанию пороги — 1 и 3 дня, затем зона без верхней границы (`maxDays: null`). Индикаторы определяются на лету: среди первых двух выбирается минимальный конечный порог, покрывающий оставшееся число дней. Третий и последующие индикаторы пропускаются, даже если их пороги конечные. Зона без верхней границы тоже пропускается. Например, при порогах 1 и 2 дни 0–1 входят в первую зону, ровно 2 — во вторую, больше 2 — в последнюю и не включаются в сводку.
- Сводка идёт на все активные подписки. Logout отключает текущий браузер через HttpOnly cookie `push_binding`; остальные устройства не затрагиваются.
- Push-сервис должен принять сообщение по сети; время показа ОС не гарантируется. TTL ограничен часом и концом локального дня.

## Запуск в Docker (PowerShell)

Команды выполнять из `repo/smart-fridge-backend`.
Если `.env` ещё нет, скопируйте `.env.example`. Существующий `.env` не перезаписывайте.
Для API остаются обязательными JWT, PostgreSQL и существующие параметры S3; отдельный worker их не требует.

Один раз сгенерировать ключи:

```powershell
go run ./cmd/vapid-keygen
```

Без установленного Go:

```powershell
docker run --rm -v "${PWD}:/app" -w /app golang:1.27 go run ./cmd/vapid-keygen
```

Генератор намеренно выводит новый приватный ключ. Сохраните пару в секретах окружения, не коммитьте и не меняйте при каждом деплое.
В `.env`:

```dotenv
SMART_FRIDGE_PUSH_ENABLED=true
SMART_FRIDGE_PUSH_SUBJECT=mailto:your-real-contact@example.com
SMART_FRIDGE_PUSH_PUBLIC_KEY=<publicKey>
SMART_FRIDGE_PUSH_PRIVATE_KEY=<privateKey>
SMART_FRIDGE_PUSH_POLL_INTERVAL=5s
SMART_FRIDGE_PUSH_REQUEST_TIMEOUT=10s
```

Subject — контактный URI `mailto:` или `https:`, а не просто email. API проверяет публичный ключ, worker — соответствие всей пары. При `PUSH_ENABLED=false` рассылка отключена, API подписки отвечает 503.

```powershell
docker compose up --build -d
docker compose logs -f notification-worker
```

PostgreSQL и Goose-миграции запускаются до API и worker. Compose использует единый образ, но разные процессы.
Остановить или перезапустить только рассылку:

```powershell
docker compose stop notification-worker
docker compose up -d notification-worker
docker compose restart notification-worker
```

Для worker отдельно от API (например, при отладке API на хосте):

```powershell
docker compose up --build -d postgres migrate notification-worker
```

У production Compose сервис называется так же: `notification-worker`. Workflow деплоя обновляет его вместе с backend.
Не выполняйте `docker compose down -v` для перезапуска — это удаляет данные PostgreSQL.

## Нативная отладка

Остановить контейнер worker, запустить только БД/миграции, затем поставить breakpoint и запустить команду:

```powershell
docker compose stop notification-worker
docker compose build smart-fridge-backend
docker compose up -d postgres migrate
go run ./cmd/notification-worker
```

Нативному worker нужен `SMART_FRIDGE_DATABASE_URL` на опубликованный порт `localhost:5433`, а не Docker-hostname `postgres`.
Точка входа — `cmd/notification-worker/main.go`. Планировщик — `internal/usecase/service/notifications.go`: `refresh`, `runDue`, `sendSummary`, `sendSubscription`.
Worker корректно отменяет текущий HTTP-запрос и останавливается по SIGTERM/Ctrl+C.
Изменения настроек подхватываются раз в `PUSH_POLL_INTERVAL`. Изменения env/ключей требуют перезапуска.

## Проверка реальной доставки без основного фронтенда

Запустить локальный стенд:

```powershell
go run ./cmd/push-debug
```

Без Go его можно собрать для Windows:

```powershell
docker run --rm -e GOOS=windows -e GOARCH=amd64 -e CGO_ENABLED=0 -v "${PWD}:/app" -w /app golang:1.27 go build -o push-debug.exe ./cmd/push-debug
.\push-debug.exe
```

Открыть `http://localhost:8081` (стенд слушает только loopback и проксирует API на `localhost:8080`). Для другого API: `go run ./cmd/push-debug -api http://localhost:8082`.

1. Войти или создать тестовый аккаунт.
2. Подписать браузер и разрешить уведомления.
3. Кнопка локального показа проверяет только разрешения и ОС, без сервера/push-провайдера.
4. Добавить тестовые продукты. Проверить каждый элемент `results` batch-ответа: просроченный и свежий при стандартных порогах не попадут в сводку, истекающий сегодня — попадёт.
5. Назначить сводку через 2 минуты. Worker должен успеть перечитать настройки до назначенного времени.
6. Закрыть вкладку и дождаться уведомления. Для повторной проверки назначить другое будущее время.
7. Для проверки нескольких устройств подписать оба браузера до запуска. Logout в одном должен отключить только его подписку.

Стенд не публикуется в production. Для проверки с телефона нужен HTTPS origin; localhost телефона не указывает на компьютер. На iOS/iPadOS нужно добавить веб-приложение на домашний экран.
ОС может скрыть уведомление из-за разрешений или режима «Не беспокоить». Access token стенда хранится только в памяти: после истечения войдите снова.

## API и настройки

Общий префикс `/api/v1`:

- `GET /push/vapid` — публичный ключ, без авторизации.
- `PUT /push/subscriptions` — Bearer JWT, тело `PushSubscription.toJSON()` с endpoint и keys; возвращает id и защищённую cookie привязки для logout.
- `DELETE /push/subscriptions/{id}` — идемпотентный отзыв своей подписки.
- `PATCH /settings` — время, timezone, включение и индикаторы.
- `POST /auth/logout` — отзыв refresh-сессии и подписки текущего браузера. При офлайн-выходе клиент должен вызвать `unsubscribe()` и повторить logout после восстановления сети.

```json
{
  "hasNotifications": true,
  "notificationTime": "09:00:00",
  "timezone": "Asia/Yekaterinburg",
  "indicators": [
    { "color": "#EF4444", "maxDays": 1 },
    { "color": "#EAB308", "maxDays": 3 },
    { "color": "#22C55E", "maxDays": null }
  ]
}
```

В частичном PATCH индикаторы можно не передавать. Если переданы — список заменяется целиком. Пороги и цвета уникальны в пределах пользователя; зона без верхней границы может быть только одна.
Endpoint должен использовать HTTPS/443. Исходящие соединения на приватные/служебные IP и redirects блокируются. Чужой endpoint не переназначается другому аккаунту (409); при смене аккаунта нужно отписаться и подписаться заново.

## Логи и БД

```powershell
docker compose logs -f notification-worker
docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

```sql
SELECT user_id, has_notifications, notification_time, timezone,
       now() AT TIME ZONE timezone AS local_now FROM settings;
SELECT user_id, max_days, color FROM indicators;
SELECT id, user_id, created_at, revoked_at FROM push_subscriptions;
```

- `push cron worker started`: планировщик запущен.
- `push schedule updated`: выражение cron пользователя добавлено/обновлено.
- `push schedule refresh failed`: настройки не перечитались, ошибка БД.
- `push accepted`: провайдер принял сообщение; это не подтверждение показа.
- `push send failed`: ошибка отправки; HTTP-код провайдера есть в логе. Автоматического повтора нет.
- 404/410 отзывают подписку; 401/403 требуют проверки ключей/subject; 429/5xx требуют проверки провайдера.

Endpoint, auth/p256dh и приватные VAPID-ключи не следует выводить в диагностические логи.

## Миграции

После `00003_notifications.sql` есть одна новая миграция `00004_notification_settings.sql`: она добавляет только `push_subscriptions.logout_token_hash` для отключения подписки текущего браузера при logout. Индексов, изменений индикаторов, ролей, триггеров и функций в ней нет.
Индикаторы остаются в исходной схеме `color` + `max_days integer`; `NULL` — отсутствие верхней границы. Стандартные записи создаёт приложение при регистрации или ленивом создании настроек. Повторный вызов не сбрасывает персональные настройки.
Старые экспериментальные `00004_notification_delivery.sql` и `00005_synchronous_notifications.sql` удалены из исходников. Уже применённую экспериментальную схему нельзя исправить одной заменой файлов: её нужно отдельно откатить или пересоздать тестовую БД. Автоматического изменения существующей пользовательской БД не выполнялось.
Перед дальнейшим изменением миграций нужно согласовать с пользователем: создавать новую или править существующую.

## Проверки

```powershell
go test ./...
go test -tags integration -race -shuffle=on -count=1 -timeout=10m ./...
golangci-lint run
```

Интеграционные тесты используют отдельный PostgreSQL 18 через testcontainers. Проверяются cron/timezone, отсутствие догоняющих запусков, обновление/удаление расписания, продолжение после ошибки подписки, отмена отправки, фильтрация продуктов, независимость worker от API и отсутствие очереди в схеме.
Генерация: `sqlc generate` (1.31.1) и `go generate ./internal/transport/http/api`.
