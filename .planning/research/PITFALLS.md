# Domain Pitfalls: FoodBI (Restaurant BI SaaS + iiko Integration)

**Domain:** Multi-tenant restaurant BI SaaS with POS integration (iiko Cloud API v2)
**Researched:** 2026-04-07
**Overall confidence:** MEDIUM — iiko-specific rate limits lack official public documentation; general multi-tenant and WebView patterns are HIGH confidence from multiple verified sources.

---

## Critical Pitfalls

Mistakes that cause rewrites, data breaches, or total system rebuilds.

---

### Pitfall 1: Missing Tenant ID in Every DB Query (Cross-Tenant Data Leak)

**What goes wrong:** A query fetches `orders`, `stock`, or `transfers` using only a resource ID without also filtering by `company_id` (tenant). One tenant can read or mutate another tenant's data by guessing integer IDs.

**Why it happens:** Developers add `company_id` to tables but forget to include it in WHERE clauses, especially in sub-queries, aggregation pipelines, or background jobs. ORM convenience methods make it easy to omit the second filter.

**Consequences:** Complete cross-tenant data exposure. GDPR/data breach liability. Rewrite of all data access layer. Trust destruction with customers.

**Prevention:**
- Enforce `company_id` in PostgreSQL Row-Level Security (RLS) policies as a second defense layer — not a replacement for application-level checks, but a backstop.
- Set `app.current_tenant` session variable on every connection before query execution in Go middleware.
- Code review rule: every SELECT/UPDATE/DELETE on tenant-scoped tables must have `AND company_id = $1` explicitly, no exceptions.
- Integration test: create two tenants, authenticate as Tenant A, attempt to read Tenant B's resource IDs — assert 404/403.

**Warning signs:**
- Any query that filters only by `id` (UUID/int) without `company_id`
- Background sync jobs that iterate over all rows without a tenant filter
- Raw SQL strings in Go code that are not parameterized

**Phase mapping:** Phase 1 (Auth + Multi-tenancy foundation). Must be solved before any data endpoints exist.

---

### Pitfall 2: iiko Cloud API Token Expiry Causing Silent Data Sync Failures

**What goes wrong:** iiko API tokens expire after ~60 minutes. If the backend does not proactively refresh tokens and a sync job runs with an expired token, the sync silently fails (or returns a 401 that is swallowed) — the dashboard shows stale data with no user-visible error.

**Why it happens:** Token refresh is often treated as "handled by the library," but background goroutines and scheduled sync jobs need their own token lifecycle management separate from HTTP request handlers. The token refresh interval in Go iiko libraries defaults to 45 minutes, but this only works if the client instance is long-lived and not re-created per sync cycle.

**Consequences:** Revenue/stock data goes stale. Users make procurement decisions on wrong numbers. Hard to debug because the system looks healthy — no crashes, just wrong data.

**Prevention:**
- Store iiko credentials (api-login, api-key) per tenant in the database, encrypted at rest.
- Run a dedicated token refresh goroutine per tenant that proactively refreshes at 45-minute intervals (before the 60-minute expiry).
- Log every token refresh event with tenant ID and timestamp. Log failures with an alert.
- After every iiko API call, check for 401 response — if received, force token refresh and retry once before marking sync as failed.
- Store `last_successful_sync_at` per tenant per data domain (orders, stock, purchases). Surface a staleness indicator in the UI if data is more than 30 minutes old.

**Warning signs:**
- Dashboard shows data frozen at a specific timestamp
- `last_sync_at` not updating for one tenant while others are current
- 401 errors in logs that do not trigger any retry

**Phase mapping:** Phase 2 (iiko integration + data sync). Design the token lifecycle before writing any iiko API calls.

---

### Pitfall 3: iiko Rate Limits Triggered by Naive Per-Tenant Polling

**What goes wrong:** With 5+ tenants each polling iiko every 5 minutes across revenue, stock, purchases, and transfers endpoints, the system sends dozens of concurrent requests to `api-ru.iiko.services`. iiko's undocumented rate limits (per api-login) get hit, resulting in 429 or timeout errors.

**Why it happens:** Rate limits for iiko Cloud API v2 are not publicly documented. Developers assume "no documented limit = no limit." Each data module independently schedules polling without a global rate coordinator.

**Consequences:** All sync jobs for a tenant fail simultaneously. iiko may throttle or block the api-login. Data outages for paying customers.

**Prevention:**
- Implement a centralized iiko API client pool in Go with a per-tenant rate limiter (e.g., `golang.org/x/time/rate`). All iiko calls go through this pool.
- Stagger sync schedules: do not poll all tenants at minute-0. Spread across the polling window using tenant-hash-based offsets.
- Respect 429 responses with exponential backoff and jitter. Minimum backoff: 30 seconds. Maximum: 10 minutes.
- Track API call count per tenant per hour. Alert if approaching a conservative self-imposed limit (e.g., more than 100 req/hour per api-login).
- Contact iiko support at api@iiko.ru early in development to request documentation of actual rate limits.

**Warning signs:**
- Spike in 429/503 responses from iiko in logs
- Multiple tenants failing sync at the same clock minute
- Sync jobs queuing up faster than they complete

**Phase mapping:** Phase 2 (iiko integration). Architecture decision — must be designed in from the start.

---

### Pitfall 4: Role Check at UI Layer Only (Owner/Employee Privilege Escalation)

**What goes wrong:** The frontend hides certain menu items for Employee role users. The backend endpoints for those features exist and are reachable — any Employee with a valid JWT can call them directly via API client or modified browser session.

**Why it happens:** "The UI won't show it" is treated as security. Backend role checks are deferred as "we can add them later."

**Consequences:** Employees can access revenue figures, salary data, or management actions they should not see. Owner-only actions (location management, employee CRUD) can be triggered by any user in the tenant.

**Prevention:**
- Every backend handler must extract `role` from the JWT and enforce it server-side before processing any request.
- Create a Go middleware `RequireRole(RoleOwner)` that returns 403 if the JWT role does not match. Apply it declaratively on routes.
- The JWT `role` claim must be set at login time and is non-mutable by the client. Never read role from a request body or query parameter.
- For Employee access to location-scoped data: validate that the employee's assigned `location_ids` includes the requested location.
- Test: log in as Employee, call Owner-only endpoints directly — assert 403.

**Warning signs:**
- Route definitions that have no auth middleware applied
- Any endpoint that reads role from the request body instead of the JWT
- Frontend code that passes role as a URL parameter

**Phase mapping:** Phase 1 (Auth). Middleware must exist before any protected endpoints are built.

---

### Pitfall 5: iiko API Key Stored in Frontend or Hardcoded in Backend Config

**What goes wrong:** The iiko api-login and api-key (credentials for `api-ru.iiko.services`) get committed to source control, stored in frontend environment variables, or hardcoded in Go config structs.

**Why it happens:** Fast prototyping. "I'll move it to env vars later." WebView apps tempt developers to make iiko calls directly from the browser to avoid building a proxy.

**Consequences:** Exposed iiko credentials allow anyone to query or manipulate the restaurant's POS data. As of November 2024, iiko disabled all API logins of 8 characters or fewer — a reminder that iiko controls credential lifecycle and credentials can be revoked unpredictably, breaking any tenant whose stored credentials are no longer valid.

**Prevention:**
- iiko credentials are stored in the database per tenant, encrypted at rest (AES-256 or equivalent).
- All iiko API calls are made server-side (Go backend) only. The frontend never touches `api-ru.iiko.services`.
- Use environment variables for service-level secrets (not tenant credentials). Add `.env`, `*.key`, `secrets.*` to `.gitignore` immediately at project init.
- Document a credential rotation procedure in the runbook before v1 launch.

**Warning signs:**
- `IIKO_API_KEY` in frontend `.env` files
- Any iiko API call visible in the browser network tab
- Credentials in Go struct literals or config files committed to git

**Phase mapping:** Phase 1 (project setup) and Phase 2 (iiko integration). Non-negotiable from day one.

---

## Moderate Pitfalls

### Pitfall 6: No Data Freshness Indicator — Users Trust Stale Analytics

**What goes wrong:** The dashboard shows revenue and stock numbers without indicating when they were last synced from iiko. Sync fails silently. An Owner makes a purchasing decision based on 3-hour-old stock data.

**Why it happens:** "We'll add last updated time later" — it gets cut from scope as a cosmetic detail.

**Prevention:**
- Every data domain (revenue, purchases, stock, transfers) has a `synced_at` timestamp stored per tenant per location.
- Dashboard cards display "Updated X minutes ago." If `synced_at` is more than 30 minutes old, show a warning indicator.
- Background sync jobs update `synced_at` only on a successful iiko response, never on error.

**Phase mapping:** Phase 2 (iiko data sync) and Phase 3 (dashboard UI).

---

### Pitfall 7: PostgreSQL N+1 Queries for Multi-Location Aggregates

**What goes wrong:** The revenue dashboard queries each of 10 locations separately in a loop (N+1), resulting in 10 sequential DB calls per page load. At 50 tenants with 10 locations each, this degrades to hundreds of queries per dashboard load.

**Why it happens:** Repository patterns fetch one location at a time. Works fine in dev with one location, breaks in staging with realistic multi-location data.

**Prevention:**
- All multi-location aggregation queries must use SQL GROUP BY with `location_id` in a single query.
- Add composite indexes: `(company_id, location_id, date)` on orders/purchases tables — the most common filter pattern.
- Run `EXPLAIN ANALYZE` on all analytics queries during development, not only after performance complaints.
- Benchmark with realistic data: seed 10 locations, 90 days of orders before testing dashboard performance.

**Phase mapping:** Phase 3 (dashboard/analytics modules). Design queries with multi-location in mind from the start.

---

### Pitfall 8: WebView Gesture Conflicts — Scrollable Content Inside Bottom Sheets

**What goes wrong:** FoodBI uses bottom sheets (date pickers, filters, confirmation flows). On Android, a scrollable list or chart inside a bottom sheet causes gesture conflicts — the inner scroll and the sheet-dismiss gesture compete. The bottom sheet becomes non-dismissible or the inner list becomes non-scrollable.

**Why it happens:** This is a documented, persistent issue in WebView gesture handling on Android. The native container and web scroll events compete for touch ownership. Confirmed in react-native-webview issue tracker.

**Consequences:** Core UX flows (date range selection, filter sheets, stock detail views) become unusable on Android — the most common platform for restaurant staff.

**Prevention:**
- Implement bottom sheets as pure CSS/JS overlay within the WebView (not as native wrappers). This avoids the native/web gesture boundary entirely.
- Use `touch-action: pan-y` and `overscroll-behavior: contain` on scrollable elements inside sheets.
- Test every bottom sheet flow on real Android hardware (not emulator) early in development.
- Avoid putting large scrollable data tables inside bottom sheets. Prefer navigation to a full-page detail view.

**Phase mapping:** Phase 3-4 (UI component library, Revenue/Stock/Purchases modules).

---

### Pitfall 9: Real-Time Analytics Illusion — Polling iiko Too Frequently

**What goes wrong:** The team polls iiko every 30 seconds to show "live" revenue. iiko's POS data has a sync delay of 1-5 minutes from terminal to iiko cloud. The aggressive polling gives no freshness benefit but multiplies API call volume by 10x compared to 5-minute polling.

**Why it happens:** Product requirement says "real-time" without defining what that means for a POS-backed system.

**Prevention:**
- Define "real-time" explicitly in product requirements: iiko Cloud is not a WebSocket stream. Terminal-to-cloud data lag is 1-5 minutes. "Near real-time" for this system means 5-minute sync intervals.
- Use 5-minute sync intervals for transactional data (orders, revenue). Use 15-minute intervals for stock. Use 1-hour intervals for purchases — match frequency to data change velocity.
- AI recommendations module should run on a scheduled batch (nightly or every few hours), not on-demand per page load.
- Set user expectations in UI: "Data updated every 5 minutes from iiko."

**Phase mapping:** Phase 2 (iiko sync architecture). This is a product + architecture decision, not just a backend detail.

---

### Pitfall 10: Heavy SVG Charts Blocking Mobile WebView Main Thread

**What goes wrong:** The Statistics module renders complex SVG charts with 90-day revenue trends across multiple locations. On mid-range Android devices, the main thread blocks for 2-4 seconds on render, causing the UI to freeze.

**Why it happens:** SVG chart libraries render all data points as DOM nodes. On a 375px WebView in a budget Android device, rendering 270 bars (3 locations x 90 days) as SVG elements is expensive.

**Prevention:**
- Use Canvas-based charting (Chart.js or ECharts) rather than SVG-based libraries (Recharts, Victory) for datasets with more than 50 data points.
- Implement data windowing: never render more than 30 data points on screen at once. Use a time range selector to limit visible data.
- Lazy-load the Statistics module — do not render charts until the user navigates to the Statistics tab.
- Test on a real mid-range Android device (not a high-end developer phone) during chart development.

**Warning signs:**
- Chrome DevTools shows main thread blocked more than 100ms on chart render
- Frame drops below 30fps during chart animation
- UI lag on scroll when charts are in viewport

**Phase mapping:** Phase 5 (Statistics module).

---

## Minor Pitfalls

### Pitfall 11: JWT Stored in localStorage (XSS Vulnerability)

**What goes wrong:** The React frontend stores access and refresh tokens in `localStorage`. An XSS vulnerability in any page — even a single unsanitized user-rendered string — allows a script to exfiltrate both tokens. Setting `innerHTML` with unsanitized content is a common source of this.

**Prevention:** Store tokens in `httpOnly` cookies set by the Go backend. Never store tokens in `localStorage` or `sessionStorage`. Mark cookies as `Secure`, `HttpOnly`, `SameSite=Strict`. Sanitize all user-generated content before rendering in the DOM.

**Phase mapping:** Phase 1 (Auth).

---

### Pitfall 12: Employee Sees Other Employees' Salaries or Personal Data

**What goes wrong:** The Employees module shows staff for a location. An employee assigned to Location A requests the employee list for Location B (different location, same company) — the backend returns it because `company_id` matches but `location_id` is not checked.

**Prevention:** Location-scoped data endpoints (employee lists, transfers, stock) must validate that the requesting user's `location_ids` includes the target location. Owner role bypasses this restriction (they can see all locations). Employee role enforces it strictly.

**Phase mapping:** Phase 6 (Employees module).

---

### Pitfall 13: File Upload Accepts Arbitrary File Types

**What goes wrong:** The invoice scan / file upload feature accepts any file type. A malicious user uploads a server-executable or polyglot file. If the server mishandles it, this can lead to code execution or data exposure.

**Prevention:** Validate MIME type server-side (not just file extension). Accept only `image/*` and `application/pdf`. Store uploads in object storage (S3/MinIO), never on the web server filesystem. Never serve uploaded files with executable content-type headers. Generate a new random filename on upload — never use the user-supplied filename.

**Phase mapping:** Phase 7 (File Upload feature).

---

### Pitfall 14: iiko Credential Policy Changes Breaking Existing Tenants

**What goes wrong:** As of November 2024, iiko disabled all API logins of 8 characters or fewer that were created before May 2024. Tenants who had not updated credentials lost all iiko sync silently.

**Why it happens:** iiko changes credential policies without prominent advance notice. Systems that do not monitor credential health proactively will fail silently when iiko revokes or modifies access.

**Prevention:**
- On every successful iiko token fetch, record the timestamp. If token fetch has failed for more than 30 minutes, trigger an in-app notification to the Owner to re-check iiko credentials.
- Build a credential health-check endpoint that tenant owners can trigger from the Settings screen to verify iiko connectivity.
- When onboarding a tenant, validate that their iiko api-login credentials work before saving them.

**Phase mapping:** Phase 2 (iiko integration) and Settings/onboarding flow.

---

## Phase-Specific Warnings Summary

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Auth + Multi-tenancy | Cross-tenant data leak via missing `company_id` filter | RLS policies + mandatory middleware tenant context |
| Auth + Multi-tenancy | JWT tokens exposed to script injection | httpOnly cookies only, no localStorage |
| Auth + Multi-tenancy | Employee bypasses Owner-only endpoints | Server-side role enforcement middleware on every route |
| iiko Integration | Token expiry causing silent sync failures | Proactive 45-min refresh goroutine + 401 retry logic |
| iiko Integration | Rate limit hit by naive concurrent polling | Centralized rate-limited API pool, staggered schedules |
| iiko Integration | iiko credentials in source code or frontend | Server-side only calls, encrypted DB storage |
| Dashboard/Analytics | N+1 queries for multi-location data | Composite indexes + single GROUP BY query per aggregate |
| Dashboard/Analytics | Stale data shown without indicator | `synced_at` timestamp on every data domain, UI staleness warning |
| Statistics Module | SVG chart freezing WebView on Android | Canvas-based charts (Chart.js/ECharts), 30-point window max |
| All UI with bottom sheets | Gesture conflict on Android WebView | Pure CSS/JS overlay sheets, `overscroll-behavior: contain` |
| Employees Module | Employee reads other locations' staff data | Location scope check on every request beyond company_id |
| File Upload | Arbitrary file type upload | Server-side MIME validation, object storage isolation |
| iiko Credential lifecycle | Policy changes silently breaking tenant sync | Credential health monitor + onboarding validation |

---

## Sources

- iiko Cloud API token lifetime and Go client defaults: [pkg.go.dev/github.com/wollzy/iiko-go](https://pkg.go.dev/github.com/wollzy/iiko-go)
- iiko API login deprecation (8-char logins, Nov 2024): [arbus.biz/blog/api_iiko_19](https://www.arbus.biz/blog/api_iiko_19/)
- iiko WebHook events (StopListUpdate, DeliveryOrderUpdate): [iikoWeb Public API - Postman](https://documenter.getpostman.com/view/2896430/TVemBpmn)
- PostgreSQL RLS pitfalls — PgBouncer session leaks, superuser bypass, policy caching: [permit.io/blog/postgres-rls-implementation-guide](https://www.permit.io/blog/postgres-rls-implementation-guide)
- Multi-tenant RLS tenant context fragility: [thenile.dev/blog/multi-tenant-rls](https://www.thenile.dev/blog/multi-tenant-rls)
- Cross-tenant data leakage patterns in JWT SaaS: [OWASP Multi-Tenant Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Multi_Tenant_Security_Cheat_Sheet.html)
- WebView gesture conflicts Android bottom sheet: [github.com/react-native-webview/react-native-webview/issues/3840](https://github.com/react-native-webview/react-native-webview/issues/3840)
- WebView non-interactive on Android inside bottom sheet: [github.com/gorhom/react-native-bottom-sheet/issues/499](https://github.com/gorhom/react-native-bottom-sheet/issues/499)
- SVG vs Canvas chart performance on mobile: [digitaladblog.com — Canvas vs WebGL for JavaScript Chart Performance](https://digitaladblog.com/2025/05/21/comparing-canvas-vs-webgl-for-javascript-chart-performance/)
- POS data sync stale data and inconsistency patterns: [trykitchenhub.com — Data Standardization](https://www.trykitchenhub.com/post/data-standardization-ensuring-format-compatibility-between-your-pos-and-external-apis)
- RBAC multi-tenant role scoping pitfalls: [workos.com/blog/how-to-design-multi-tenant-rbac-saas](https://workos.com/blog/how-to-design-multi-tenant-rbac-saas)
- iiko organization model and credential structure: [pyiikocloudapi PyPI](https://pypi.org/project/pyiikocloudapi/0.0.7/)
- Multi-tenant JWT security best practices: [OWASP Multi-Tenant Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Multi_Tenant_Security_Cheat_Sheet.html)
