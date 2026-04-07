# Technology Stack

**Project:** FoodBI — Restaurant BI SaaS with iiko Cloud API v2
**Researched:** 2026-04-07
**Overall confidence:** HIGH (core stack verified via multiple 2026 sources)

---

## Recommended Stack

### Backend: Go

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Chi** | v5 | HTTP router + middleware | Stays on net/http (stdlib-compatible), zero external deps, composable middleware, idiomatic Go. Fiber is faster but breaks net/http contract — that matters for long-term maintainability. Echo is close second but Chi wins on minimalism. |
| **golang-jwt/jwt** | v5 | JWT authentication | Battle-tested, v5 supports RFC 7519 fully. Never roll your own JWT. |
| **pgx** | v5 | PostgreSQL driver | Lib/pq is in maintenance mode. pgx/v5 is faster, better maintained, native COPY protocol, named parameters. |
| **sqlc** | v1.27+ | SQL → type-safe Go | Write SQL, get type-safe Go code generated. Eliminates ORM magic while keeping compile-time safety. Superior to GORM for a BI platform where you need precise query control. |
| **golang-migrate** | v4 | Database migrations | Standard tool, SQL files checked into repo, up/down migrations, dirty-state tracking. |
| **River** | v0.x (latest) | Background job queue | Postgres-native (no Redis dependency), transaction-safe job enqueueing, generics-based typed workers. Perfect for iiko sync jobs — enqueue sync task in same DB transaction as tenant creation. |
| **go-redis/redis** | v9 | Redis client + caching | Session store, cache for iiko API responses (rate limit compliance), real-time dashboard caching. go-redis v9 supports Redis 7+, generics. |
| **go-playground/validator** | v10 | Request validation | Struct-tag based validation, widely used with Chi, comprehensive rule set. Use for incoming HTTP request validation. |
| **slog** (stdlib) | Go 1.21+ | Structured logging | Standard library slog added in Go 1.21. Use slog with zerolog as handler backend for high-throughput paths. Zero external dep for most logging needs. |
| **zerolog** | v1 | High-perf log handler | Use as slog handler backend for production. 50K+ logs/sec with minimal allocations. |
| **golang.org/x/crypto** | latest | bcrypt for passwords | Use bcrypt (cost 12) for password hashing. Part of x/crypto — standard Go extension library. |
| **sashabaranov/go-openai** | v1 | OpenAI API client | For AI Suggestions feature. Well-maintained, supports GPT-4o and streaming. |

**Go version target:** 1.22+ (for enhanced routing in stdlib net/http, range-over-func, slog stability)

---

### Frontend: React + TypeScript

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Vite** | v5+ | Build tool | 40x faster builds vs CRA. Rolldown (Rust bundler) in v6 cuts prod builds 8.5x further. First-class TS support. Standard for SPA/dashboard apps in 2026. |
| **React** | 19 | UI framework | Already decided. React 19 ships with native Suspense for data fetching. |
| **TypeScript** | 5.x | Type safety | Non-negotiable for production dashboard. Strict mode. |
| **shadcn/ui** | latest | UI component system | Copy-paste components (you own the code), Tailwind + Radix UI base, built for Tailwind v4 + React 19. Ideal for custom design systems — matches FoodBI's Atomic Design approach. NOT a traditional library (no version to install as a package). |
| **Tailwind CSS** | v4 | Styling | shadcn/ui requires it. v4 is a complete rewrite — faster, CSS-first config. Mobile-first utilities work perfectly for 375px WebView target. |
| **TanStack Query** | v5 (5.96+) | Server state management | Caching, background refetch, loading/error states. Standard for API-heavy dashboards. v5 has full Suspense support and cleaner API. Replaces manual useEffect+useState for all API calls. |
| **Zustand** | v5 | Client state management | 1.2KB, ~20M weekly downloads, no Provider wrapper needed. For UI state (active location, selected date ranges, sidebar state). NOT for server data — TanStack Query handles that. |
| **Recharts** | v2 | Charts | Built on React + D3. Declarative, composable. Tremor wraps Recharts — using Recharts directly gives more control for custom mobile chart layouts. |
| **React Hook Form** | v7 | Form management | Performance-first (uncontrolled inputs), integrates with Zod for validation. For all forms: login, employee add, supply request. |
| **Zod** | v3 | Schema validation | TypeScript-first runtime validation. Use at API boundaries + form validation. Pair with React Hook Form. |
| **React Router** | v6/v7 | Client routing | Standard SPA routing. v7 (formerly Remix) adds loaders/actions but stay on v6 unless SSR needed — WebView SPA doesn't need it. |
| **Axios** | v1 | HTTP client | Interceptors for auth token injection + refresh logic. More ergonomic than fetch for complex auth flows in a SaaS app. |

**Node version target:** 20 LTS (for Vite v5 compatibility and stable ESM)

---

### Database: PostgreSQL

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **PostgreSQL** | 15+ | Primary data store | Already decided. Use JSON/JSONB for iiko webhook payloads, partial indexes for tenant queries. |
| **Row-Level Security (RLS)** | PostgreSQL native | Multi-tenant isolation | Database-enforced tenant isolation. Every table gets `company_id` column + RLS policy. Defense-in-depth: even if application code has a bug, cross-tenant data leaks are blocked at DB layer. |
| **Redis** | 7+ | Cache + session store | iiko API response caching (iiko has rate limits), JWT session invalidation list, background job deduplication. |

**Multi-tenancy pattern:** Shared database, shared schema, RLS + `company_id` column on all tenant-scoped tables. Schema-per-tenant adds operational complexity (migrations must run N times) that isn't worth it at this scale.

---

### Infrastructure

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Docker + Docker Compose** | latest | Local dev + deployment | Standard containerization. Compose for local: postgres, redis, backend, frontend dev server. |
| **Air** | v1 | Go hot reload (dev) | Live reload for Go during development. Standard tool in Go ecosystem. |

---

### iiko Cloud API Integration

| Aspect | Decision | Why |
|--------|----------|-----|
| **HTTP client** | `net/http` with custom wrapper | Standard library is sufficient. Add retry logic, rate limit handling, and circuit breaker manually. No external dependency needed. |
| **Auth** | iiko API key per organization | Stored encrypted in DB per company tenant. Never in code or env for all tenants. |
| **Sync strategy** | Scheduled River jobs per location | Pull data on schedule (e.g., every 15min for revenue, hourly for stock). Store normalized in PostgreSQL. Don't call iiko on every dashboard load — too slow and rate-limited. |
| **Base URL** | `https://api-ru.iiko.services/api/1/` | Official iiko Cloud API v2 endpoint for Russian region |

---

## Alternatives Considered

| Category | Recommended | Rejected | Why Rejected |
|----------|-------------|----------|--------------|
| HTTP Router | Chi | Gin | Gin uses custom context (breaks net/http compatibility), more magic. Chi is closer to stdlib. |
| HTTP Router | Chi | Fiber | Fiber uses fasthttp (incompatible with net/http middleware ecosystem). Excellent benchmarks but ecosystem lock-in. |
| DB Access | sqlc + pgx | GORM | GORM uses reflection, generates inefficient queries, hides SQL from BI-critical paths. BI platforms need query control. |
| DB Access | sqlc + pgx | ent | ent is code-first schema, good for simple CRUD, poor for complex analytical queries. |
| Job Queue | River | Asynq | Asynq requires Redis. River uses existing Postgres — one fewer infrastructure dependency. Transaction-safe job enqueueing is a significant advantage. |
| Job Queue | River | Machinery | Machinery is largely unmaintained as of 2024. |
| Frontend UI | shadcn/ui | MUI | MUI imposes Google's design system, difficult to theme to match FoodBI green/dark design. shadcn/ui gives full ownership. |
| Frontend UI | shadcn/ui | Ant Design | Ant Design is heavy (hundreds of KB), opinionated Chinese enterprise aesthetic, poor mobile WebView performance at 375px. |
| Frontend UI | shadcn/ui + Recharts | Tremor | Tremor wraps Recharts + shadcn — adding an extra abstraction layer. Use the building blocks directly for more control over mobile chart behavior. |
| State | Zustand | Redux Toolkit | Redux adds boilerplate (actions, reducers, slices) that's unnecessary for a dashboard with TanStack Query handling server state. |
| Build | Vite | Next.js | FoodBI is a WebView SPA — no SEO, no SSR needed. Next.js adds complexity without benefit. Vite SPA is simpler and faster. |
| Logging | slog + zerolog | logrus | logrus is in maintenance mode since 2022. slog is now the stdlib standard. |

---

## Installation

### Backend (Go)

```bash
# Initialize module
go mod init github.com/yourorg/foodbi-backend

# Core dependencies
go get github.com/go-chi/chi/v5
go get github.com/golang-jwt/jwt/v5
go get github.com/jackc/pgx/v5
go get github.com/riverqueue/river
go get github.com/redis/go-redis/v9
go get github.com/go-playground/validator/v10
go get github.com/rs/zerolog
go get golang.org/x/crypto
go get github.com/sashabaranov/go-openai

# Dev tools
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/air-verse/air@latest
```

### Frontend (React + TypeScript)

```bash
# Create project
npm create vite@latest foodbi-frontend -- --template react-ts

# UI + styling
npx shadcn@latest init
npm install tailwindcss @tailwindcss/vite

# Data fetching + state
npm install @tanstack/react-query zustand axios

# Charts + forms
npm install recharts react-hook-form zod @hookform/resolvers

# Routing
npm install react-router-dom

# Dev
npm install -D typescript @types/react @types/react-dom
```

---

## Security Notes

- JWT secrets: environment variable only, never hardcoded. Use 256-bit random secret.
- iiko API keys: stored encrypted (AES-256) in PostgreSQL per tenant. Key management via env-provided master key.
- Passwords: bcrypt cost factor 12 via `golang.org/x/crypto/bcrypt`.
- RLS policies: set `app.current_company_id` PostgreSQL session variable from JWT claim in every request, never from user input.
- Rate limiting: apply per-IP and per-user middleware at Chi router level for auth endpoints (login, register, OTP).
- CORS: whitelist specific WebView origins only, not `*`.
- Redis sessions: set TTL, use `SETNX` for session tokens to prevent duplication.

---

## Confidence Assessment

| Area | Confidence | Basis |
|------|------------|-------|
| Go HTTP (Chi) | HIGH | Multiple 2026 framework comparisons confirm Chi for minimalist/stdlib-compatible use cases |
| Go DB (sqlc + pgx) | HIGH | Official sqlc docs + multiple verified 2026 sources; pgx v5 is current official recommendation |
| Go jobs (River) | MEDIUM-HIGH | Active project, Postgres-native design well-documented; younger than Asynq, but architectural fit is superior |
| Frontend (Vite + React 19) | HIGH | Multiple 2026 production guides; Vite is ecosystem standard for SPA |
| shadcn/ui + Tailwind v4 | HIGH | Official shadcn site confirms Tailwind v4 + React 19 support as of 2026 |
| TanStack Query v5 | HIGH | v5 stable release confirmed; 5.96.2 latest as of April 2026 |
| PostgreSQL RLS | HIGH | AWS docs + multiple 2026 sources confirm RLS as standard multi-tenant pattern |
| iiko API v2 | MEDIUM | Base URL confirmed; full v2 endpoint coverage needs verification against official iiko docs at api-ru.iiko.services. No official Go SDK exists — custom HTTP wrapper required. |
| AI (go-openai) | MEDIUM | Library is well-maintained; AI feature requirements are loosely defined in PROJECT.md — deeper research needed at AI Suggestions phase |

---

## Sources

- [Choosing a Go Web Framework in 2026 (Medium)](https://medium.com/@samayun_pathan/choosing-a-go-web-framework-in-2026-a-minimalists-guide-to-gin-fiber-chi-echo-and-beego-c79b31b8474d)
- [Best Go Backend Frameworks in 2026 (Encore)](https://encore.dev/articles/best-go-backend-frameworks)
- [sqlc + pgx in Go (brandur.org)](https://brandur.org/sqlc)
- [sqlc official docs — Using Go and pgx](https://docs.sqlc.dev/en/stable/guides/using-go-and-pgx.html)
- [pgx GitHub](https://github.com/jackc/pgx)
- [River — Fast background jobs in Go](https://riverqueue.com/)
- [golang-migrate GitHub](https://github.com/golang-migrate/migrate)
- [go-redis official Redis docs](https://redis.io/docs/latest/develop/clients/go/)
- [TanStack Query v5 overview](https://tanstack.com/query/v5/docs/framework/react/overview)
- [shadcn/ui official site](https://ui.shadcn.com/)
- [Tailwind v4 support in shadcn/ui](https://ui.shadcn.com/docs/tailwind-v4)
- [Zustand GitHub](https://github.com/pmndrs/zustand)
- [Top 5 React chart libraries 2026 (Syncfusion)](https://www.syncfusion.com/blogs/post/top-5-react-chart-libraries)
- [PostgreSQL Row-Level Security for Multi-Tenant SaaS (AWS)](https://aws.amazon.com/blogs/database/multi-tenant-data-isolation-with-postgresql-row-level-security/)
- [Shipping multi-tenant SaaS using Postgres RLS (Nile)](https://www.thenile.dev/blog/multi-tenant-rls)
- [High-Performance Structured Logging in Go with slog and zerolog (Leapcell)](https://leapcell.io/blog/high-performance-structured-logging-in-go-with-slog-and-zerolog)
- [iiko Cloud API Postman collection](https://www.postman.com/avatariya/iiko-cloud-api/overview)
- [salesduck/iiko-cloud-api GitHub](https://github.com/salesduck/iiko-cloud-api)
- [any-llm-go Mozilla AI](https://blog.mozilla.ai/run-openai-claude-mistral-llamafile-and-more-from-one-interface-now-in-go/)
- [Vite production setup guide 2026](https://oneuptime.com/blog/post/2026-01-08-react-typescript-vite-production-setup/view)
