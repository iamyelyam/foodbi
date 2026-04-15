<!-- generated-by: gsd-doc-writer -->
# FoodBI API Reference

## Base URL

| Environment | Base URL |
|-------------|----------|
| Local | `http://localhost:8080/api/v1` |
| Production | `https://foodbi-production.up.railway.app/api/v1` |

Health check (no auth): `GET /health` → `{"status":"ok","time":"..."}`

---

## Authentication

All endpoints under `/api/v1` except the public auth routes below require a JWT access token:

```
Authorization: Bearer <access_token>
```

The JWT carries `user_id`, `company_id`, and `role` claims. The `JWTAuth` middleware validates the token and the `TenantContext` middleware sets `app.current_tenant` on the PostgreSQL connection for row-level security (RLS). Every query is automatically scoped to the authenticated company — there is no way to access another tenant's data.

**Owner-only endpoints** check `role == "owner"` and return `403 Forbidden` for non-owners.

---

## Common Patterns

### Date filtering

All date-range parameters follow the exclusive upper-bound pattern:

```
order_date >= date_from AND order_date < (date_to::date + 1)
```

Pass dates as `YYYY-MM-DD` strings. `date_to` is inclusive from the caller's perspective (the backend adds 1 day internally).

### Pagination

Most list endpoints return the envelope below. Purchases `/purchases` is special: without `?page=N` it returns **all rows** in the date window (no LIMIT/OFFSET applied). Pass `?page=1` to activate 20-per-page pagination.

```json
{
  "items": [...],
  "total": 142,
  "page": 1,
  "per_page": 20,
  "total_pages": 8
}
```

Revenue `/revenue/orders` always returns all rows for the period (no server-side pagination).

### Error format

All errors return JSON with an `error` field:

```json
{"error": "description of what went wrong"}
```

| HTTP Status | Meaning |
|-------------|---------|
| 400 | Bad request / validation failure |
| 401 | Missing or invalid JWT / invalid OTP or refresh token |
| 403 | Authenticated but insufficient role (non-owner hitting owner-only route) |
| 404 | Resource not found within the tenant |
| 409 | Conflict (duplicate email on register/employee create) |
| 429 | Rate limit exceeded (webhooks only: 600 req/min per company) |
| 500 | Internal server error |

---

## Modules

### Auth

Public routes (no JWT required):

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register a new owner account; sends OTP to email |
| POST | `/auth/login` | Login with email + password; returns tokens |
| POST | `/auth/verify-otp` | Verify 6-digit OTP code; activates account and returns tokens |
| POST | `/auth/refresh` | Exchange refresh token for new access + refresh tokens |
| POST | `/auth/accept-invite` | Set password and name for an invited user; returns tokens |
| POST | `/auth/forgot-password` | Send password reset link (always returns 200 to prevent enumeration) |
| POST | `/auth/reset-password` | Reset password using token from email |

Authenticated routes (JWT required):

| Method | Path | Auth | Owner only | Description |
|--------|------|------|------------|-------------|
| POST | `/auth/logout` | Yes | No | Invalidate current session |
| GET | `/auth/me` | Yes | No | Get current user record |
| POST | `/auth/invite` | Yes | Yes | Create invitation link for a new employee |

**Register request:**
```json
{
  "email": "owner@example.com",
  "password": "minlength8",
  "first_name": "Ali",
  "last_name": "Bekov",
  "role": "owner",
  "company_name": "My Restaurant Group"
}
```

**Login / verify-otp response (tokens):**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 3600
}
```

**Invite response:**
```json
{
  "token": "abc123",
  "invite_url": "/accept-invite?token=abc123"
}
```

---

### Locations

Base path: `/locations` (JWT required)

| Method | Path | Owner only | Description |
|--------|------|------------|-------------|
| GET | `/locations` | No | List all locations for the company |
| POST | `/locations` | Yes | Create a new location |
| GET | `/locations/sync-status` | No | Get last iiko sync status per location and sync type |
| POST | `/locations/{id}/sync` | No | Trigger iiko sync for a location (queues all 4 sync types) |

**Create location request:**
```json
{
  "name": "Almaty Central",
  "city": "Almaty",
  "address": "Abay 10",
  "pos_system": "iiko",
  "iiko_org_id": "uuid-from-iiko"
}
```

**List locations response item:**
```json
{
  "id": "uuid",
  "company_id": "uuid",
  "name": "Almaty Central",
  "address": "Abay 10",
  "iiko_org_id": "uuid",
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Sync status response item:**
```json
{
  "location_id": "uuid",
  "sync_type": "revenue",
  "status": "completed",
  "records_synced": 1420,
  "started_at": "2024-01-01T06:00:00Z",
  "completed_at": "2024-01-01T06:01:12Z",
  "error": null
}
```

`sync_type` values: `revenue`, `product_sales`, `purchases`, `stock`.

**Trigger sync response:**
```json
{"status": "sync_queued", "location_id": "uuid"}
```

---

### Dashboard

Base path: `/dashboard` (JWT required)

| Method | Path | Query params | Description |
|--------|------|--------------|-------------|
| GET | `/dashboard/summary` | `location_id`, `date_from`, `date_to` | Aggregated KPIs: revenue, orders, purchases, profit, top suppliers |
| GET | `/dashboard/revenue-trend` | `location_id`, `days` (1–365, default 7) | Daily revenue + order count time series |

**Summary response:**
```json
{
  "today_revenue": 1250000,
  "today_orders": 87,
  "today_purchases": 340000,
  "today_purchase_count": 4,
  "week_revenue": 7800000,
  "week_orders": 512,
  "week_purchases": 1900000,
  "week_purchase_count": 22,
  "today_profit": 910000,
  "week_profit": 5600000,
  "prev_week_revenue": 7200000,
  "revenue_change_percent": 8.33,
  "top_suppliers": [
    {"supplier_name": "АгроПоставка", "total_sum": 620000}
  ]
}
```

`today_profit` = revenue for selected period − COGS (from `product_sales_facts.cost_price`). `week_profit` = week revenue − week COGS. All monetary values are in KZT with no decimal subunits.

**Revenue trend response (array):**
```json
[
  {"date": "2024-01-15", "revenue": 1250000, "orders": 87, "items": 340}
]
```

---

### Revenue

Base path: `/revenue` (JWT required)

| Method | Path | Owner only | Description |
|--------|------|------------|-------------|
| GET | `/revenue/orders` | No | List orders with optional filters |
| GET | `/revenue/orders/{id}` | No | Get order detail with line items |
| POST | `/revenue/orders/{id}/status` | Yes | Update order status (approved / rejected) |
| GET | `/revenue/products` | No | List products with aggregated sales data |
| GET | `/revenue/products/{id}` | No | Last 30 days of daily sales for a product |
| GET | `/revenue/products/{id}/trend` | No | Sales trend for a product with optional date range |
| GET | `/revenue/products/{id}/orders` | No | Recent orders containing a product (`?limit=` 1–50, default 10) |

**Order list query params:** `location_id`, `status`, `date_from`, `date_to`

**Order list response:**
```json
{
  "orders": [
    {
      "id": "uuid",
      "order_number": "12345",
      "order_date": "2024-01-15T13:22:00Z",
      "revenue": 15400,
      "discount": 0,
      "order_type": "dine_in",
      "status": "approved",
      "item_count": 4,
      "waiter_name": "Arman"
    }
  ],
  "total": 87,
  "page": 1,
  "per_page": 87,
  "total_pages": 1
}
```

Note: the orders endpoint always returns all rows for the period (no pagination).

**Order detail response** adds `total_cost`, `profit`, and `items` array:
```json
{
  "id": "uuid",
  "items": [
    {"product_name": "Плов классический", "quantity": 2, "revenue": 5000, "cost_price": 1400}
  ],
  "total_cost": 1400,
  "profit": 13600
}
```

**Product list query params:** `location_id`, `date_from`, `date_to`

**Product list response item:**
```json
{
  "product_id": "iiko-uuid",
  "product_name": "Плов классический",
  "category": "Горячие блюда",
  "total_quantity": 142.0,
  "total_revenue": 710000,
  "total_cost": 213000
}
```

---

### Purchases

Base path: `/purchases` (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/purchases` | List purchase invoices |
| GET | `/purchases/{id}` | Get invoice detail with line items |
| GET | `/purchases/suppliers` | List suppliers aggregated by total spend |
| GET | `/purchases/suppliers/{id}` | Get supplier detail with recent invoices |
| PUT | `/purchases/suppliers/{id}/alias` | Set or remove a display name override for a supplier |

**Purchase list query params:** `location_id`, `supplier_id`, `date_from`, `date_to`, `page`

Pagination behavior: without `?page=N` all rows in the date window are returned. With `?page=N`, 20 rows per page.

**Purchase list response:**
```json
{
  "purchases": [
    {
      "id": "uuid",
      "document_number": "INV-2024-001",
      "supplier_id": "iiko-supplier-uuid",
      "supplier_name": "АгроПоставка",
      "incoming_date": "2024-01-15T00:00:00Z",
      "status": "accepted",
      "total_sum": 340000
    }
  ],
  "total": 22,
  "page": 1,
  "per_page": 22,
  "total_pages": 1
}
```

**Purchase detail** adds `line_items`:
```json
{
  "line_items": [
    {
      "product_name": "Рис басмати",
      "product_code": "P-001",
      "unit": "кг",
      "quantity": 50.0,
      "price": 800,
      "subtotal": 40000
    }
  ]
}
```

**Set supplier alias request:** `{"display_name": "АгроПоставка (Алматы)"}` — pass empty string to remove the alias.

---

### Stock

Base path: `/stock` (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/stock` | Current stock snapshot (latest per product) |
| GET | `/stock/low-stock` | Products with amount <= 5 (default threshold) |
| PUT | `/stock/products/{id}/alias` | Set or remove display name for a stock product |
| PUT | `/stock/products/{id}/override` | Set manual amount and/or unit price override |
| GET | `/stock/products/{id}/used-in` | Dishes whose recipe uses the given ingredient |

`{id}` in all stock product paths is the `iiko_product_id` (iiko UUID).

**Query params:** `location_id` (optional, filters by location)

**Stock item response:**
```json
{
  "product_id": "iiko-uuid",
  "product_name": "Рис басмати",
  "amount": 48.5,
  "unit": "кг",
  "cost_sum": 38800,
  "price_per_unit": 800,
  "snapshot_at": "2024-01-15T06:00:00Z",
  "override_at": "2024-01-14T10:00:00Z"
}
```

`override_at` is present only when a manual override is active. When an override exists, `amount`, `cost_sum`, and `price_per_unit` reflect the overridden values rather than the iiko snapshot.

**Set override request:**
```json
{"manual_amount": 50.0, "manual_price_per_unit": 800}
```

Either field may be omitted (partial update). Sending both as `null` deletes the override and reverts to iiko values.

**Set alias request:** `{"display_name": "Рис басмати (Краснодарский)"}` — empty string removes alias.

**Used-in dishes response (array):**
```json
[
  {
    "dish_iiko_id": "iiko-uuid",
    "dish_name": "Плов классический",
    "amount": 0.15,
    "unit": "кг",
    "dish_unit": "порц."
  }
]
```

---

### AI Suggestions

Base path: `/ai` (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ai/suggestions` | Generate data-driven suggestions from current analytics |
| POST | `/ai/tasks` | Create a manual follow-up task |
| GET | `/ai/tasks` | List all tasks for the company |

**Query params for suggestions:** `location_id` (optional)

**Suggestions response:**
```json
{
  "summary": {
    "total_loss": 85000,
    "total_gain_with_ai": 127000,
    "date": "2024-01-15"
  },
  "suggestions": [
    {
      "id": "uuid",
      "type": "menu_optimization",
      "title_key": "ai.s.topSeller.title",
      "title_params": {"product": "Плов классический"},
      "description_key": "ai.s.topSeller.description",
      "solution_key": "ai.s.topSeller.solution",
      "solution_params": {"product": "Плов классический"},
      "impact": "high",
      "gain_amount": 71000,
      "loss_amount": 0
    }
  ]
}
```

**Important:** `title_key`, `description_key`, `solution_key` are i18n translation keys, not rendered strings. The frontend resolves them via `useT(key, params)` substituting `{placeholder}` values from the corresponding `*_params` map. Backend pre-formats param values (e.g. applies `formatProductName()`) so the frontend substitutes verbatim.

`impact` values: `high`, `medium`, `low`.

`type` values observed: `menu_optimization`, `price_adjustment`, `cost_configuration`, `stock_data_issue`, `purchase_recommendation`.

`loss_amount` — money currently slipping away (data errors, low margins). `gain_amount` — projected uplift if action is taken. Both are KZT integers, 0 when not applicable.

**Create task request:**
```json
{"title": "Review Rис basil pricing", "description": "Margin below 15%"}
```

---

### Statistics

Base path: `/statistics` (JWT required)

| Method | Path | Query params | Description |
|--------|------|--------------|-------------|
| GET | `/statistics/revenue` | `location_id`, `date_from`, `date_to` | Daily revenue + order count (defaults: last 30 days) |
| GET | `/statistics/profit` | `location_id`, `date_from`, `date_to` | Daily revenue vs purchase cost, profit per day |
| GET | `/statistics/top-products` | `location_id` | Top 20 products by revenue with margin % |

**Revenue stats response (array):**
```json
[{"date": "2024-01-15", "revenue": 1250000, "cost": 0, "profit": 0, "orders": 87}]
```

**Profit stats response (array):** same shape; `cost` = purchase invoice totals for that day, `profit` = revenue − cost.

**Top products response item:**
```json
{
  "product_name": "Плов классический",
  "category": "Горячие блюда",
  "quantity": 142.0,
  "revenue": 710000,
  "margin_percent": 70.0
}
```

---

### Supplying (Supply Requests)

Base path: `/supplying` (JWT required)

| Method | Path | Owner only | Description |
|--------|------|------------|-------------|
| GET | `/supplying` | No | List supply requests (`?status=pending\|approved\|rejected`) |
| POST | `/supplying` | No | Create a new supply request |
| GET | `/supplying/{id}` | No | Get supply request detail with items |
| POST | `/supplying/{id}/approve` | Yes | Approve a pending request |
| POST | `/supplying/{id}/reject` | Yes | Reject a pending request |

**Create supply request:**
```json
{
  "location_id": "uuid",
  "supplier_name": "АгроПоставка",
  "items": [
    {
      "product_name": "Рис басмати",
      "category": "Крупы",
      "quantity": 50.0,
      "unit": "кг",
      "price_per_unit": 800
    }
  ]
}
```

Response: `{"id": "uuid", "status": "pending", "total_sum": 40000}`

---

### Transfers

Base path: `/transfers` (JWT required)

| Method | Path | Owner only | Description |
|--------|------|------------|-------------|
| GET | `/transfers` | No | List transfers (`?location_id`, `?status`, `?date_from`, `?date_to`) |
| POST | `/transfers` | No | Create a stock transfer between locations |
| GET | `/transfers/{id}` | No | Get transfer detail with items |
| POST | `/transfers/{id}/complete` | Yes | Mark pending transfer as completed |
| POST | `/transfers/{id}/cancel` | Yes | Cancel a pending transfer |

**Create transfer request:**
```json
{
  "from_location_id": "uuid-a",
  "to_location_id": "uuid-b",
  "items": [
    {"product_name": "Масло подсолнечное", "category": "Масла", "quantity": 5.0, "unit": "л"}
  ]
}
```

`from_location_id` must differ from `to_location_id`.

---

### Employees

Base path: `/employees` (JWT required)

| Method | Path | Owner only | Description |
|--------|------|------------|-------------|
| GET | `/employees` | No | List all employees in the company |
| POST | `/employees` | Yes | Create a new employee with password |
| GET | `/employees/{id}` | No | Get employee detail with assigned location names |
| PUT | `/employees/{id}/role` | Yes | Update employee role (`owner` or `employee`) |
| PUT | `/employees/{id}/locations` | Yes | Replace employee's assigned locations |
| POST | `/employees/{id}/deactivate` | Yes | Deactivate an employee (cannot deactivate self) |

**Create employee request:**
```json
{
  "email": "arman@example.com",
  "first_name": "Arman",
  "last_name": "Seitkali",
  "phone": "+77001234567",
  "role": "employee",
  "password": "minlength8"
}
```

**Assign locations request:** `{"location_ids": ["uuid1", "uuid2"]}`

Replaces all existing assignments. Pass an empty array to unassign all locations.

---

### Profile

Base path: `/profile` (JWT required)

| Method | Path | Owner only | Description |
|--------|------|------------|-------------|
| GET | `/profile/me` | No | Get current user profile with company settings |
| PUT | `/profile/me` | No | Update first name, last name, phone |
| PUT | `/profile/company-settings` | Yes | Update company country/currency/locale preset |

**Profile response:**
```json
{
  "id": "uuid",
  "email": "owner@example.com",
  "first_name": "Ali",
  "last_name": "Bekova",
  "phone": "+77001234567",
  "role": "owner",
  "company_name": "My Restaurant Group",
  "company_settings": {
    "country": "KZ",
    "currency": "KZT",
    "currency_symbol": "₸",
    "locale": "ru-KZ"
  }
}
```

**Update company settings request:** `{"country": "KZ"}` — country code selects a preset. Supported codes: `KZ`, `RU`, `UZ`, `AE`, `US`, `EU`, `GB`, `TR`, `GE`.

---

### Notifications

Base path: `/notifications` (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/notifications` | List last 50 notifications for the user (or company-wide) |
| GET | `/notifications/unread-count` | Get count of unread notifications |
| POST | `/notifications/{id}/read` | Mark a notification as read |
| POST | `/notifications/read-all` | Mark all unread notifications as read |

**Notification object:**
```json
{
  "id": "uuid",
  "type": "sync_complete",
  "title": "Sync finished",
  "message": "Revenue sync completed for Almaty Central",
  "is_read": false,
  "created_at": "2024-01-15T06:01:12Z"
}
```

---

### Files

Base path: `/files` (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/files/upload` | Upload a file (multipart/form-data, field name `file`, max 10 MB) |
| GET | `/files` | List last 50 uploaded files |
| GET | `/files/{id}` | Get file metadata by ID |
| DELETE | `/files/{id}` | Delete a file record and the stored file |

**Upload response:**
```json
{"id": "uuid", "filename": "invoice.pdf", "size": 204800, "status": "uploaded"}
```

`status` lifecycle: `uploaded` → `processing` → `processed`.

---

### Payments Webhook

**This endpoint does not require JWT.** It uses HMAC-SHA256 signature verification instead.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhooks/payment/{companyID}` | Receive payment terminal events |

Note: this endpoint is under `/api/v1/webhooks/payment/{companyID}` — outside the JWT-protected group.

**Request headers:**
```
X-Webhook-Signature: <hmac-sha256-hex>
Content-Type: application/json
```

The signature is `HMAC-SHA256(request_body, webhook_secret)` where `webhook_secret` is stored per company in the `companies` table.

**Webhook payload:**
```json
{
  "terminal_id": "uuid-or-terminal-name",
  "order_id": "POS-ORDER-123",
  "table_number": "12",
  "guest_name": "Иван И.",
  "guest_phone": "+77001234567",
  "amount": 15400,
  "status": "failed",
  "timestamp": "2024-01-15T13:22:00Z"
}
```

`status` must be `failed` or `success`. `amount` is KZT integer (no subunits). `timestamp` is optional RFC3339; defaults to server time if omitted.

**Behavior:**
- Idempotent: duplicate `(company, order_id, status, attempt_at)` is silently ignored (`status: "duplicate"` returned).
- On `failed`: enqueues a Telegram notification to subscribed chats for the terminal.
- On `success`: sends Telegram notification only if prior `failed` attempts exist for the same `order_id`.
- Rate limit: 600 requests per minute per company; returns `429` if exceeded.

---

## iiko Sync Notes

Sync is triggered via `POST /locations/{id}/sync`. The sync worker (separate binary) processes the queue. The four sync types and what they populate:

| Sync type | Target table |
|-----------|-------------|
| `revenue` | `revenue_facts` |
| `product_sales` | `product_sales_facts` |
| `purchases` | `purchase_facts`, `purchase_line_items` |
| `stock` | `stock_snapshots` |

Check progress via `GET /locations/sync-status`. The `status` field values are: `queued`, `running`, `completed`, `failed`.

**Critical iiko data rules:**
- `DishSumInt` from iiko OLAP is already in KZT — never divide by 100.
- Revenue OLAP groups by `UniqOrderId.Id + DishName`; Go code sums `DishSumInt` per `UniqOrderId.Id` before upserting to avoid iiko returning a single arbitrary row instead of the order total.
- Cost price uses `ProductCostBase.ProductCost` from iiko for real COGS. Margin = `(1 - cost / revenue) × 100`.
