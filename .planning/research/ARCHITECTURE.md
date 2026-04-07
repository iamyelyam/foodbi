# Architecture Patterns: FoodBI

**Domain:** Multi-tenant restaurant BI SaaS with POS integration
**Researched:** 2026-04-07
**Overall confidence:** HIGH (Go + PostgreSQL RLS patterns well-documented; iiko API sync patterns MEDIUM — iiko webhook support is limited, polling is primary confirmed method)

---

## Recommended Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    CLIENT LAYER                         │
│   React + TypeScript (WebView, 375px mobile-first)      │
│   TanStack Query · Zustand · React Router               │
└───────────────────┬────────────────────────────────────┘
                    │ HTTPS / JWT Bearer
┌───────────────────▼────────────────────────────────────┐
│                   API GATEWAY                           │
│   Go · Chi or Echo · JWT validation · tenant injection  │
│   Rate limiting · Security headers · Request logging    │
└──┬──────────────────────┬────────────────────────────┬──┘
   │                      │                            │
┌──▼──────┐      ┌────────▼──────┐         ┌──────────▼──┐
│  Auth   │      │  Core API     │         │  Sync       │
│ Service │      │  Service      │         │  Worker     │
│         │      │               │         │             │
│ Login   │      │ Revenue       │         │ iiko poller │
│ OTP     │      │ Purchases     │         │ Job queue   │
│ Roles   │      │ Stock         │         │ Token cache │
│ Session │      │ Transfers     │         │ Error retry │
│ Invites │      │ Employees     │         │             │
└──┬──────┘      │ Notifications │         └──────┬──────┘
   │             │ AI Suggest    │                │
   │             └───────┬───────┘                │
   │                     │                        │
   └─────────────────────▼────────────────────────▼──────┐
                   PostgreSQL (single cluster)            │
                   Shared schema + RLS per tenant_id      │
                   Tables: companies, locations,          │
                   users, roles, revenue_facts,           │
                   purchase_facts, stock_snapshots,       │
                   transfers, iiko_sync_log               │
                                                         │
                   ┌─────────────────────────────────────┘
                   │
                   ▼
          iiko Cloud API v2
          api-ru.iiko.services
          (pull-only, token TTL 1h, refresh at 45min)
```

---

## Component Boundaries

| Component | Responsibility | Communicates With | Technology |
|-----------|---------------|-------------------|------------|
| React Frontend | UI, local state, auth token storage, mobile WebView rendering | API Gateway (HTTPS) | React + TS, TanStack Query, React Router, Zustand |
| API Gateway | Auth middleware, JWT validation, tenant context injection into every request, rate limiting, routing to service handlers | Frontend, Auth Service, Core API Service, PostgreSQL | Go, Chi or Echo, golang-jwt |
| Auth Service | Registration, login, OTP verification, session management, role assignment, invite flows | API Gateway, PostgreSQL | Go, bcrypt/argon2, UUID sessions |
| Core API Service | All business domain reads and writes: revenue, purchases, stock, transfers, employees, notifications, AI suggestions | API Gateway, PostgreSQL | Go, pgx v5, sqlx |
| Sync Worker | Periodic pull from iiko Cloud API, token management, data normalization, write to PostgreSQL | iiko Cloud API, PostgreSQL, Job Queue | Go, cron scheduler, iiko HTTP client |
| PostgreSQL | Single source of truth, multi-tenant isolation via RLS, analytics queries | All Go services | PostgreSQL 15+, Row Level Security |
| iiko Cloud API | External POS data source (read-only from FoodBI perspective) | Sync Worker only | REST, token-based auth, 1h TTL |

---

## Multi-Tenancy Model

**Pattern: Shared schema, shared database, RLS isolation**

This is the right choice for FoodBI v1. It minimizes operational overhead (one database to manage), costs less to run, and PostgreSQL RLS enforces tenant boundaries at the database level — even if application code has a bug and forgets to filter, the database will not leak cross-tenant data.

**Tenant hierarchy:**
```
Company (tenant)
  └── Locations (1..N)
        └── Users with roles (Owner / Employee)
              └── Data: revenue, purchases, stock, transfers
```

**Isolation mechanism:**

Every table that holds tenant-owned data carries a `company_id UUID NOT NULL` column. RLS policies are enabled on all such tables. At the start of every request, the Go service calls:

```sql
SET LOCAL app.current_tenant = '<company_id>';
```

RLS policies use:
```sql
CREATE POLICY tenant_isolation ON revenue_facts
  USING (company_id = current_setting('app.current_tenant')::uuid);
```

**Tenant context flow:**
1. User authenticates → JWT issued containing `company_id`, `user_id`, `role`
2. API Gateway validates JWT signature and expiry
3. Gateway extracts `company_id` from token claims
4. Gateway injects `company_id` into request context (Go context.Context)
5. Every database call opens a transaction and sets `app.current_tenant` before any query
6. RLS silently filters all rows — no application-level WHERE clauses needed on tenant ID (defense in depth: still use them, but RLS is the backstop)

**Role enforcement:**
- Owner: full access to all locations in their company
- Employee: access limited to their assigned location(s), read-only on most modules
- RBAC is enforced in the Core API Service handler layer, after tenant isolation is already established by the gateway

---

## Data Flow: iiko to User Dashboard

```
iiko Cloud API v2
        │
        │  PULL (polling every N minutes per company)
        │  GET /api/1/organizations
        │  GET /api/1/reports/olap  (revenue/orders)
        │  GET /api/1/store/MOVEMENTS (stock movements)
        │  GET /api/1/documents/purchase_invoice (purchases)
        │
        ▼
   Sync Worker (Go)
        │
        ├─ Authenticate: POST /api/1/access_token
        │  Response: { token, correlationId }
        │  Token TTL: 1 hour → refresh at 45min intervals
        │
        ├─ Per-company, per-location sync loop
        │  ├─ Fetch raw iiko data
        │  ├─ Normalize to FoodBI schema
        │  ├─ Upsert into PostgreSQL fact tables
        │  └─ Write sync_log entry (success/fail/timestamp)
        │
        │  Error handling: exponential backoff, dead letter log,
        │  alert if company sync fails > 3 consecutive times
        │
        ▼
   PostgreSQL (fact tables)
   revenue_facts, purchase_facts, stock_snapshots, transfer_records
        │
        │  QUERY (on user request)
        │
        ▼
   Core API Service (Go)
        │
        ├─ Aggregation queries (GROUP BY date, location, category)
        ├─ Pre-computed materialized views for dashboard KPIs
        │  (refreshed after each sync cycle)
        └─ JSON response with typed DTOs
        │
        ▼
   API Gateway (applies RLS context, validates JWT)
        │
        ▼
   React Frontend
        │
        ├─ TanStack Query caches responses (staleTime: 5min for dashboards)
        ├─ Background refetch on window focus
        ├─ Skeleton loading states (mobile WebView, slow connections)
        └─ Rendered dashboard with revenue, purchases, stock charts
```

**Sync frequency recommendation:**
- Dashboard KPIs: sync every 15 minutes (acceptable lag for BI)
- Stock levels: sync every 30 minutes
- Historical reports: sync once per hour or on-demand
- Do NOT attempt real-time sync — iiko Cloud API is not designed for it and rate limits apply

---

## Component Build Order (Phase Dependencies)

Build in this order — each phase unblocks the next:

```
Phase 1: Foundation
├─ PostgreSQL schema (companies, locations, users, roles)
├─ RLS policies on all tenant tables
├─ Auth Service (register, login, JWT issuance, OTP)
└─ API Gateway skeleton (JWT middleware, tenant injection, routing)
    [Frontend can stub with mock data]

Phase 2: iiko Integration
├─ Sync Worker foundation (iiko token management, retry logic)
├─ iiko organizations + locations sync
└─ Initial fact table schema (revenue_facts, purchase_facts)
    [Depends on: Phase 1 DB schema]

Phase 3: Core Data Modules
├─ Revenue sync + API endpoints + frontend module
├─ Purchases sync + API endpoints + frontend module
└─ Dashboard with real aggregated data
    [Depends on: Phase 2 sync working]

Phase 4: Operations Modules
├─ Stock Management (sync + CRUD + frontend)
├─ Transfers (sync + CRUD + frontend)
└─ Supplying / purchase requests
    [Depends on: Phase 3 fact tables established]

Phase 5: People + Admin
├─ Employee management (invites, roles, locations)
├─ Notifications center
└─ Profile module
    [Depends on: Phase 1 auth, Phase 3 tenant structure confirmed]

Phase 6: Intelligence Layer
├─ File Upload (invoice scanning)
├─ AI Suggestions (analytics on top of existing fact tables)
└─ Statistics (deep reports, custom date ranges, exports)
    [Depends on: Phase 3-4 data volume accumulated]
```

**Critical dependency:** The Sync Worker must be stable before any data module can be validated with real data. Build and harden it in Phase 2 before building any UI that depends on it.

---

## Patterns to Follow

### Pattern 1: RLS Tenant Injection via SET LOCAL

Set tenant context per transaction, not per connection. Using SET LOCAL scopes the variable to the current transaction, which is safe with connection pooling (PgBouncer, pgx pool).

```go
func (r *repo) withTenant(ctx context.Context, tx pgx.Tx, companyID uuid.UUID) error {
    _, err := tx.Exec(ctx,
        "SET LOCAL app.current_tenant = $1", companyID.String())
    return err
}
```

Never trust a connection-level setting when using a pool — always use SET LOCAL inside the transaction.

### Pattern 2: Sync Worker as Isolated Go Service

The sync worker should be a separate binary (or at minimum a separate goroutine group) from the API service. It runs on a cron schedule per company, manages its own iiko token cache, and writes to the database independently of user requests.

This prevents a slow or failing sync from impacting API latency for end users.

### Pattern 3: Materialized Views for Dashboard KPIs

Pre-compute aggregated metrics (daily revenue by location, top products) as PostgreSQL materialized views. Refresh them after each sync cycle completes. Dashboard endpoint queries the view, not raw fact tables.

This keeps dashboard loads fast (< 200ms target) even as fact tables grow to millions of rows.

### Pattern 4: TanStack Query for Frontend State

Use TanStack Query (React Query v5) for all server state. Do not use Redux or Zustand for server-fetched data. Use Zustand only for pure UI state (selected date range, active tab, modal open state).

```typescript
// Dashboard KPIs — 5 min stale, background refetch
const { data, isLoading } = useQuery({
  queryKey: ['dashboard', companyId, locationId, dateRange],
  queryFn: () => api.getDashboard(locationId, dateRange),
  staleTime: 5 * 60 * 1000,
})
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Passing tenant_id as a URL or body parameter

Never let the client send their own `company_id` in a request body or URL path that the backend trusts without verification. The company_id must always be derived from the validated JWT on the server side.

**Consequence:** Any user could query another tenant's data by changing a parameter.
**Instead:** Extract company_id exclusively from JWT claims in the API Gateway middleware.

### Anti-Pattern 2: Synchronous iiko data fetch on user request

Never call iiko Cloud API synchronously when a user loads a dashboard. iiko API latency is unpredictable (external dependency), and their rate limits would be hit immediately at any real scale.

**Consequence:** Dashboard load times of 2-10 seconds, timeouts, cascading failures.
**Instead:** All iiko data is pre-fetched by the Sync Worker and stored in PostgreSQL. The API serves only from the local database.

### Anti-Pattern 3: Connection-level SET for tenant context

Using `SET app.current_tenant` (without LOCAL) persists for the entire connection session. With connection pooling, the next request reusing the connection will inherit the previous tenant's context.

**Consequence:** Cross-tenant data leak.
**Instead:** Always use `SET LOCAL` inside an explicit transaction.

### Anti-Pattern 4: Storing iiko API credentials per-request

The iiko token has a 1-hour TTL. Requesting a new token per API call will exhaust rate limits and add 200-400ms of latency to every sync operation.

**Instead:** The Sync Worker maintains an in-memory token cache per company, refreshes at 45-minute intervals, and reuses the cached token for all requests within the TTL window.

---

## Scalability Considerations

| Concern | At 10 restaurants | At 100 restaurants | At 1000 restaurants |
|---------|-------------------|--------------------|---------------------|
| Sync Worker | Single goroutine pool, sequential per company | Parallel goroutines per company, rate limit awareness | Distributed worker queue (Redis + worker pool), dedicated sync service |
| Database | Single PostgreSQL instance sufficient | Read replica for analytics queries | Partitioned fact tables by company_id + date range |
| API latency | Direct PostgreSQL queries | Materialized views + Redis cache for hot queries | Dedicated analytics database or OLAP layer (ClickHouse) |
| Token management | In-memory map in sync worker | Same, but with TTL management per company | Distributed cache (Redis) for tokens across worker instances |
| Multi-tenancy | RLS on all tables | Same, RLS handles scale transparently | Schema-per-tenant becomes viable at this scale if needed |

For FoodBI v1 (target: small team, initial customers), the single-instance model with RLS is correct. Do not over-engineer for 1000 restaurants on day one.

---

## Security Architecture Notes

(Per CLAUDE.md security requirements)

- JWT tokens contain: `user_id`, `company_id`, `role`, `location_ids[]`, `exp`. Signed with HS256 or RS256. Never embed sensitive PII.
- Passwords stored with bcrypt (cost factor 12 minimum). Never logged.
- iiko API credentials (api-login, api-key) stored in environment variables, never in code or database.
- All inter-service communication is internal (same binary or same private network). No public-facing sync worker.
- RLS policies are the second line of defense after application-level authorization checks — both must exist.
- Auth failures are logged with timestamp, IP, user_id attempt. Successful logins logged similarly. Rate limiting on `/auth/*` endpoints (10 attempts / minute / IP).
- File uploads (invoice scanning) validated server-side: MIME type, file size cap, stored with random UUID filenames not user-supplied names.

---

## Sources

- Multi-tenant RLS patterns: [AWS: Multi-tenant data isolation with PostgreSQL Row Level Security](https://aws.amazon.com/blogs/database/multi-tenant-data-isolation-with-postgresql-row-level-security/)
- PostgreSQL RLS for SaaS: [The Nile: Shipping multi-tenant SaaS using Postgres Row-Level Security](https://www.thenile.dev/blog/multi-tenant-rls)
- Multi-tenant architecture overview: [bix-tech: Multi-Tenant Architecture Complete Guide](https://bix-tech.com/multi-tenant-architecture-the-complete-guide-for-modern-saas-and-analytics-platforms-2/)
- Go clean architecture patterns: [AleksK1NG Go-Clean-Architecture-REST-API](https://github.com/AleksK1NG/Go-Clean-Architecture-REST-API)
- iiko Cloud API reference: [iiko Postman collection](https://www.postman.com/avatariya/iiko-cloud-api/overview), [iikoWeb Public API docs](https://documenter.getpostman.com/view/2896430/TVemBpmn)
- iiko Go client (token TTL/refresh behavior): [wollzy/iiko-go](https://pkg.go.dev/github.com/wollzy/iiko-go)
- JWT in microservices: [microservices.io authentication series](https://microservices.io/post/architecture/2025/05/28/microservices-authn-authz-part-2-authentication.html)
- TanStack Query for BI dashboards: [Mastering React Query 2025](https://dev.to/jdavissoftware/mastering-react-query-in-2025-a-deep-dive-into-data-fetching-for-modern-apps-22jf)
- Data pipeline patterns: [Real-Time Data Integration for Scalable Analytics](https://www.perceptive-analytics.com/real-time-data-integration-architecture-for-scalable-analytics/)
