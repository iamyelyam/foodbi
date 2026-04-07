# Roadmap: FoodBI

**Created:** 2026-04-07
**Phases:** 6
**Requirements:** 63 v1 requirements mapped
**Coverage:** 100% — all v1 requirements assigned to a phase

---

## Overview

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | Foundation | Auth + DB + Project Scaffold | AUTH-01..08, TENANT-01..05 | 5 |
| 2 | iiko Integration | Sync Worker + Location Management | IIKO-01..07, LOC-01..03 | 4 |
| 3 | Core Analytics | Dashboard + Revenue + Purchases + Statistics | DASH-01..05, REV-01..06, PURCH-01..05, STAT-01..04 | 5 |
| 4 | Operations | Stock + Supplying + Transfers | STOCK-01..02, SUPPLY-01..05, TRANS-01..05 | 4 |
| 5 | People + Notifications | Employees + Profile + Notifications | EMP-01..06, PROF-01..02, NOTIF-01..03 | 4 |
| 6 | Intelligence | AI Suggestions + File Upload | AI-01..04, FILE-01..04 | 3 |

---

## Phase Details

### Phase 1: Foundation
**Goal:** Рабочая авторизация (email/password, OTP), мультитенантная схема БД с RLS, скелет Go API и React фронтенда с дизайн-системой из Figma.

**Requirements:** AUTH-01, AUTH-02, AUTH-03, AUTH-04, AUTH-05, AUTH-06, AUTH-07, AUTH-08, TENANT-01, TENANT-02, TENANT-03, TENANT-04, TENANT-05

**UI hint**: yes

**Success Criteria:**
1. User can register as Owner, log in, receive OTP, and access protected dashboard stub
2. Employee can register via invite and see restricted view
3. RLS prevents cross-tenant data access (verified by test)
4. Frontend renders all auth screens matching Figma design (Sign in, Password, OTP, Sign up)
5. Docker Compose starts all services (postgres, redis, backend, frontend)

**Depends on:** nothing (first phase)

---

### Phase 2: iiko Integration
**Goal:** Стабильная синхронизация данных из iiko Cloud API v2 в PostgreSQL. Управление локациями.

**Requirements:** IIKO-01, IIKO-02, IIKO-03, IIKO-04, IIKO-05, IIKO-06, IIKO-07, LOC-01, LOC-02, LOC-03

**UI hint**: yes

**Success Criteria:**
1. Sync Worker connects to iiko API, authenticates, and pulls organization data
2. Revenue, stock, and purchase data appear in PostgreSQL fact tables after sync
3. Sync failures are retried with backoff and logged to iiko_sync_log
4. Owner can add location, switch location, and see sync status

**Depends on:** Phase 1 (DB schema, auth, API skeleton)

---

### Phase 3: Core Analytics
**Goal:** Главный экран с метриками, модули Revenue и Purchases с реальными данными из iiko, статистика.

**Requirements:** DASH-01, DASH-02, DASH-03, DASH-04, DASH-05, REV-01, REV-02, REV-03, REV-04, REV-05, REV-06, PURCH-01, PURCH-02, PURCH-03, PURCH-04, PURCH-05, STAT-01, STAT-02, STAT-03, STAT-04

**UI hint**: yes

**Success Criteria:**
1. Dashboard loads with real aggregated revenue and purchase data in under 2 seconds
2. User can browse orders, filter by date/status, view product details
3. User can browse purchases by supplier, view supplier profile
4. Statistics page shows revenue/profit charts with custom date ranges
5. All screens match Figma design for Revenue, Purchases, Statistics sections

**Depends on:** Phase 2 (iiko sync providing data)

---

### Phase 4: Operations
**Goal:** Управление складом, заявки на поставку, трансферы между локациями.

**Requirements:** STOCK-01, STOCK-02, SUPPLY-01, SUPPLY-02, SUPPLY-03, SUPPLY-04, SUPPLY-05, TRANS-01, TRANS-02, TRANS-03, TRANS-04, TRANS-05

**UI hint**: yes

**Success Criteria:**
1. User sees current stock levels synced from iiko
2. User can create, review, and confirm supply request end-to-end
3. User can create transfer between locations with quantity selection
4. Transfer history shows all past transfers with working filters

**Depends on:** Phase 3 (fact tables, product/category data established)

---

### Phase 5: People + Notifications
**Goal:** Управление сотрудниками, профили, уведомления, RBAC на всех endpoints.

**Requirements:** EMP-01, EMP-02, EMP-03, EMP-04, EMP-05, EMP-06, PROF-01, PROF-02, NOTIF-01, NOTIF-02, NOTIF-03

**UI hint**: yes

**Success Criteria:**
1. Owner can add employee, assign role and location
2. Employee sees only their location's data (RBAC verified)
3. Notification center shows alerts for stock, approvals, sync failures
4. Profile view/edit works for both Owner and Employee

**Depends on:** Phase 1 (auth/roles), Phase 3 (tenant structure confirmed)

---

### Phase 6: Intelligence
**Goal:** AI-аналитика продаж, сканирование инвойсов.

**Requirements:** AI-01, AI-02, AI-03, AI-04, FILE-01, FILE-02, FILE-03, FILE-04

**UI hint**: yes

**Success Criteria:**
1. AI suggestions page shows recommendations based on historical data
2. User can create task from AI suggestion
3. User can scan/upload invoice and edit extracted data

**Depends on:** Phase 3-4 (accumulated data volume for AI analysis)

---

## Milestone

All 6 phases constitute **Milestone 1: FoodBI v1 MVP**.

---
*Last updated: 2026-04-07 after initialization*
