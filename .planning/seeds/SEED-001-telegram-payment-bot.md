---
id: SEED-001
status: dormant
planted: 2026-04-10
planted_during: Phase 1 — Foundation (pre-execution)
trigger_when: При добавлении интеграции с платежной системой или Telegram-функционала
scope: Large
---

# SEED-001: Telegram-бот уведомлений о неуспешных оплатах за столом

## Why This Matters

**Контроль в реальном времени.** Менеджеры ресторанов должны мгновенно видеть проблемы с оплатой — не в отчётах через час, а сразу в Telegram. Неуспешная оплата за столом = потенциально потерянный заказ, если персонал не подойдёт помочь.

У нас уже есть:
- Инструмент оплаты за каждым столом, интегрированный с кассовой системой
- Данные об авторизованных гостях (имя, телефон)
- terminalID для фильтрации по локациям

## When to Surface

**Trigger:** Когда начнём работу над Telegram-интеграциями, платёжными webhook'ами, или добавим real-time уведомления за пределы in-app notifications (Phase 5+).

This seed should be presented during `/gsd-new-milestone` when the milestone scope matches any of these conditions:
- Milestone включает Telegram-бота или внешние мессенджеры
- Milestone включает интеграцию с платёжной системой / терминалами оплаты
- Milestone включает real-time оповещения для менеджеров
- Milestone расширяет систему уведомлений за пределы текущего NOTIF-01..03

## Scope Estimate

**Large** — полный milestone. Включает:

1. **Telegram Bot Service** (Go) — новый сервис/пакет в backend
   - Telegram Bot API интеграция (go-telegram-bot-api)
   - Webhook от платёжной системы для получения событий оплаты
   - Логика отслеживания попыток: failed → уведомление, success после failed → уведомление об успехе

2. **Фильтрация по локациям (terminalID)**
   - Пользователь бота выбирает один или несколько терминалов/локаций
   - Подписка сохраняется в БД (telegram_subscriptions)
   - Команды бота: /start, /locations, /subscribe, /unsubscribe

3. **Данные в уведомлении**
   - Имя гостя
   - Номер телефона
   - Номер стола
   - Время попытки
   - Статус: неуспешно / успешно (после неуспешной)

4. **БД расширения**
   - Таблица payment_attempts (terminal_id, table_number, guest_name, guest_phone, attempt_time, status, order_id)
   - Таблица telegram_subscriptions (telegram_chat_id, company_id, terminal_ids[], active)

5. **UI настройки** (опционально)
   - Страница в frontend для управления Telegram-подписками
   - Аналитика попыток оплаты

## Breadcrumbs

Related code and decisions found in the current codebase:

- `backend/internal/notifications/handler.go` — существующая система уведомлений (in-app), можно расширить паттерн
- `backend/internal/locations/handler.go` — управление локациями, iiko_org_id маппинг
- `backend/internal/models/models.go` — модель Location с IikoOrgID (terminalID маппинг)
- `backend/cmd/api/main.go` — роутинг, сюда добавить webhook endpoint
- `backend/internal/sync/service.go` — паттерн фоновых сервисов (тикеры, graceful shutdown)
- `migrations/000004_notifications.up.sql` — схема уведомлений для расширения
- `.planning/ROADMAP.md` Phase 5 — People + Notifications (NOTIF-01..03)

## Notes

- Платёжная система уже интегрирована с кассовой системой — нужен только webhook для получения событий
- Авторизованные гости уже идентифицированы — данные (имя, телефон) доступны
- terminalID — ключ фильтрации, маппится на локации в системе
- Рассмотреть: rate limiting на уведомления (чтобы не спамить при массовых сбоях терминала)
- Рассмотреть: группировка уведомлений по столу (если несколько попыток подряд — одно уведомление с счётчиком)
