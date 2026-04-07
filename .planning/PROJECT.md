# FoodBI

## What This Is

FoodBI — multi-tenant SaaS BI-платформа для ресторанного бизнеса. Интегрируется с кассовой системой iiko Cloud API v2, предоставляя владельцам и сотрудникам ресторанных сетей аналитику по выручке, закупкам, складу, трансферам между локациями. WebView мобильное приложение с AI-рекомендациями по оптимизации бизнеса.

## Core Value

Владелец ресторана видит актуальную аналитику по выручке, закупкам и складу всех своих локаций в одном мобильном приложении, данные берутся напрямую из iiko.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Авторизация (email/password, OTP, регистрация Owner/Employee)
- [ ] Мультитенантность (компании с несколькими локациями)
- [ ] Интеграция с iiko Cloud API v2 (синхронизация данных)
- [ ] Дашборд — главный экран с ключевыми метриками (выручка, закупки)
- [ ] Revenue модуль — заказы, продукты, фильтры, детали, статусы
- [ ] Purchases модуль — закупки по поставщикам, фильтры, детали
- [ ] Stock Management — управление складом
- [ ] Supplying — заявки на поставку (категории, продукты, подтверждение)
- [ ] Transfer — трансферы между локациями
- [ ] History — история трансферов с фильтрами
- [ ] Employees — управление сотрудниками (добавление, роли, локации)
- [ ] Profile — личный профиль и профиль сотрудника
- [ ] Notifications — центр уведомлений
- [ ] File Upload — сканирование инвойсов камерой, загрузка файлов
- [ ] AI Suggestions — AI-аналитика продаж (рекомендации по меню, ценам, закупкам)
- [ ] Statistics — детальная статистика Revenue/Profit с фильтрами и графиками
- [ ] Location Management — добавление/смена локации

### Out of Scope

- Нативное мобильное приложение (iOS/Android) — используем WebView
- Собственная платежная система — работаем через данные iiko
- Интеграция с другими POS-системами кроме iiko — фокус на iiko в v1
- CRM функционал — FoodBI это BI/аналитика, не CRM

## Context

- **Figma дизайн:** FoodBI design system v1 (Atomic Design: atoms, molecules, organisms)
- **Целевая платформа:** Mobile WebView (375px ширина)
- **Дизайн-система:** Primary green (#6ADEBF), dark theme elements, shadows
- **Компоненты:** Main Header, Tabbar, Bottom Sheets, Date Picker, Segmented Controls
- **Роли пользователей:** Owner (полный доступ) и Employee (ограниченный)
- **iiko Cloud API:** api-ru.iiko.services — выручка, заказы, продукты, склад, поставщики
- **14 секций в дизайне**, ~100 экранов total

## Constraints

- **Tech Stack (Backend):** Go — выбор заказчика
- **Tech Stack (Frontend):** React + TypeScript — WebView mobile-first
- **Tech Stack (Database):** PostgreSQL — аналитические запросы, надёжность
- **Integration:** iiko Cloud API v2 — единственный источник данных POS
- **Multi-tenant:** Каждая компания изолирована, несколько локаций на компанию
- **Mobile-first:** 375px viewport, touch-optimized UI

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go для backend | Выбор заказчика, производительность | — Pending |
| React + TS для frontend | Экосистема, WebView совместимость | — Pending |
| PostgreSQL | Аналитические запросы, JSON поддержка | — Pending |
| iiko Cloud API v2 | Облачная версия, REST API, документация | — Pending |
| Multi-tenant SaaS | Масштабируемость для нескольких компаний | — Pending |
| WebView вместо native | Быстрее разработка, один кодбейс | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-07 after initialization*
