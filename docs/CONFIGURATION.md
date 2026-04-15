<!-- generated-by: gsd-doc-writer -->
# Configuration

This document covers all environment variables, database setup, frontend configuration, per-tenant settings, and client-side preferences for FoodBI.

---

## Backend Environment Variables

The backend reads configuration from environment variables (loaded via `godotenv` from a `.env` file in development). The canonical example file is `backend/.env.example`.

### Database connection

The backend accepts either a single `DATABASE_URL` or individual component variables. When `DATABASE_URL` is set it takes precedence; individual variables are used as a fallback with the defaults shown below.

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Optional | _(built from components below)_ | Full PostgreSQL DSN, e.g. `postgres://foodbi:foodbi@localhost:5432/foodbi?sslmode=disable`. When set, all `DB_*` variables are ignored. |
| `DB_HOST` | Optional | `localhost` | PostgreSQL host |
| `DB_PORT` | Optional | `5432` | PostgreSQL port |
| `DB_USER` | Optional | `foodbi` | PostgreSQL user |
| `DB_PASSWORD` | Optional | `foodbi` | PostgreSQL password |
| `DB_NAME` | Optional | `foodbi` | PostgreSQL database name |

Source: `backend/internal/database/pool.go`

### Authentication

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | **Required** | _(none)_ | Secret key used to sign and verify JWT access tokens. Must be a long, randomly generated string in production. |

Source: `backend/internal/auth/service.go`, `backend/internal/middleware/auth.go`

### Server

| Variable | Required | Default | Description |
|---|---|---|---|
| `PORT` | Optional | `8080` | TCP port the HTTP API server listens on |
| `ENV` | Optional | _(unset)_ | Set to `production` to disable pretty-print console logging and enable structured JSON log output |
| `MIGRATIONS_DIR` | Optional | `./migrations` | Path to the SQL migration files directory. The API server applies pending migrations on startup. |

Source: `backend/cmd/api/main.go`

### Telegram bot

| Variable | Required | Default | Description |
|---|---|---|---|
| `TELEGRAM_BOT_TOKEN` | Optional | _(none)_ | Telegram Bot API token. When set, the payment notification bot starts automatically. When absent, the bot is disabled and a warning is logged. |

Source: `backend/cmd/api/main.go`

### File uploads

| Variable | Required | Default | Description |
|---|---|---|---|
| `UPLOAD_DIR` | Optional | `./uploads` | Directory where uploaded invoice and document files are stored. Created automatically if it does not exist. |

Source: `backend/internal/files/handler.go`

### Redis

| Variable | Required | Default | Description |
|---|---|---|---|
| `REDIS_URL` | Optional | `redis://localhost:6379` | Redis connection URL. Listed in `.env.example`; verify which features currently depend on Redis. <!-- VERIFY: confirm which backend features require Redis and whether startup fails without it --> |

Source: `backend/.env.example`

### OTP / email delivery

OTP codes are generated on registration and password-reset flows. The auth service contains a comment: "In production, send OTP via email, never return in response." No SMTP or email-provider environment variables are currently wired in the source — email sending is a **pending implementation**.

<!-- VERIFY: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD or equivalent email provider vars are not present in backend source as of this writing. Add them here once the email delivery feature is implemented. -->

---

## Frontend Environment Variables

The frontend is a Vite + React application. Vite exposes environment variables prefixed with `VITE_` at build time via `import.meta.env`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `VITE_API_URL` | Optional | _(empty — uses Vite proxy in dev)_ | Absolute base URL of the backend API. Example: `https://foodbi-production.up.railway.app`. When unset, all `/api/*` requests go through the Vite dev proxy to `http://localhost:8080`. |

Set in `frontend/.env.production`:

```
VITE_API_URL=https://foodbi-production.up.railway.app
```

Source: `frontend/src/lib/api.ts`, `frontend/.env.production`

The resolved API base is:

```
// dev:        /api/v1  (proxied by Vite to localhost:8080)
// production: https://foodbi-production.up.railway.app/api/v1
```

---

## Vite Dev Proxy

In development, the Vite dev server (`localhost:3000`) proxies all requests matching `/api/*` to the local Go backend at `http://localhost:8080`. No manual CORS or token changes are needed.

```ts
// frontend/vite.config.ts
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
},
```

---

## PostgreSQL Setup

### Local development

The default local database matches the `.env.example` values:

| Setting | Value |
|---|---|
| Host | `localhost` |
| Port | `5432` |
| Database | `foodbi` |
| User | `foodbi` |
| Password | `foodbi` |

**Important:** If you have Homebrew `postgresql@16` installed it binds to `localhost:5432` by default and will conflict with any Docker-based Postgres. Stop the Homebrew service before starting a Docker container:

```bash
brew services stop postgresql@16
```

### Row Level Security (RLS)

All tenant-scoped tables use PostgreSQL Row Level Security. The Go API sets the active tenant for every request via:

```sql
SET LOCAL app.current_tenant = '<company_uuid>';
```

This must run inside a transaction. A `BYPASSRLS` superuser account is needed for migration runs and administrative queries that must operate across all tenants.

<!-- VERIFY: confirm the exact BYPASSRLS role name used in production and whether it is a separate DB user or the same foodbi user with elevated privileges -->

### Migrations

Migrations are plain SQL files in `backend/migrations/`. The API binary applies all pending migrations automatically on startup using the `database.RunMigrations` helper. Migration failures are logged as warnings — the server continues to start.

---

## Per-Tenant Configuration

iiko credentials and locale settings are stored in the database per company, not in environment variables.

### `companies` table columns

| Column | Description |
|---|---|
| `iiko_server_url` | URL of the company's iiko Server instance (e.g. `https://iiko.company.kz`) |
| `iiko_login` | iiko Server API login |
| `iiko_password` | iiko Server API password |
| `country` | ISO country code, default `KZ` |
| `currency_code` | ISO currency code, default `KZT` |
| `currency_symbol` | Display symbol, default `₸` |
| `locale` | Number/date locale, default `ru-KZ` |

The sync worker reads `iiko_server_url`, `iiko_login`, and `iiko_password` for each company at every sync cycle. These are configured by the company owner through the locations/settings UI, not via environment variables.

Source: `backend/migrations/000007_iiko_server.up.sql`, `backend/migrations/000014_company_i18n.up.sql`

### `locations` table columns

| Column | Description |
|---|---|
| `iiko_org_id` | iiko organization UUID used in OLAP API calls |
| `city` | City name for display |
| `pos_system` | POS back-office system: `iiko`, `r_keeper`, `Poster`, or `manual` |
| `address` | Physical address |

Source: `backend/migrations/000001_init_schema.up.sql`, `backend/migrations/000019_location_city_pos.up.sql`

---

## Client-Side Storage

The frontend persists several user preferences in `localStorage`. These are per-device and are never synced to the backend.

### i18n locale

| Key | Type | Default | Description |
|---|---|---|---|
| `foodbi_locale` | `string` | `'ru'` | Active UI locale. Valid values: `en`, `ru`, `kk`, `es`. Set via the profile page language selector. |

Source: `frontend/src/i18n/index.ts`

### UI preferences

| Key | Type | Description |
|---|---|---|
| `foodbi-ui-prefs-v1` | JSON object | User-level UI preferences. Currently contains one key: `showUploadInvoicesBanner` (boolean, default `true`). Parsed with a `DEFAULT_UI_PREFS` fallback so missing keys never cause errors. |

The `showUploadInvoicesBanner` field controls whether the invoice upload prompt banner is shown. It is set to `false` when the user dismisses the banner.

Source: `frontend/src/stores/app.ts`

### JWT tokens

| Key | Description |
|---|---|
| `access_token` | Short-lived JWT used as `Authorization: Bearer <token>` on every API request |
| `refresh_token` | Long-lived token used by the Axios interceptor to silently refresh the access token on 401 responses |

---

## Notifications Feature Flag

The push notification bell icon is globally disabled in the frontend until the feature is complete. The flag lives in `frontend/src/components/layout/Header.tsx`:

```ts
// TEMP: notifications hidden across the project until the feature is ready.
const NOTIFICATIONS_ENABLED = false
```

To re-enable: set `NOTIFICATIONS_ENABLED = true` and uncomment the bell icon JSX in the same file.

---

## Currency Formatting

Currency display is hardcoded to Kazakhstan Tenge (KZT) with the `ru-KZ` locale. There are no configurable currency environment variables.

- **Frontend**: use `toLocaleString('ru-KZ', { maximumFractionDigits: 0 })` or the `useCurrency()` hook from `frontend/src/stores/app.ts`. Never use `toFixed(2)` or the `'en'` locale.
- **Symbol**: `₸` — never hardcode `€`, `$`, or `₽`.
- **Backend**: all monetary values from iiko are in KZT with no subunits. No division or multiplication is applied anywhere in the pipeline.

The company-level `currency_code`, `currency_symbol`, and `locale` columns in the `companies` table allow future multi-currency support, but the formatter is currently hardcoded to `ru-KZ`.

---

## Production Deployment (Railway)

FoodBI is deployed on Railway using the `railway.json` at the repository root. The deployment builds from a `Dockerfile` and starts the API binary:

```json
{
  "build": { "builder": "DOCKERFILE", "dockerfilePath": "Dockerfile" },
  "deploy": {
    "startCommand": "./api",
    "healthcheckPath": "/health",
    "healthcheckTimeout": 30,
    "restartPolicyType": "ON_FAILURE"
  }
}
```

The following environment variables must be set in the Railway project dashboard:

| Variable | Notes |
|---|---|
| `DATABASE_URL` | Railway PostgreSQL plugin provides this automatically when a Postgres addon is attached <!-- VERIFY: confirm Railway auto-injects DATABASE_URL from the attached Postgres plugin --> |
| `JWT_SECRET` | Set a long random string in Railway's Variables panel |
| `TELEGRAM_BOT_TOKEN` | Set to enable payment notification bot; omit to disable |
| `ENV` | Set to `production` |
| `PORT` | Railway injects this automatically <!-- VERIFY: confirm Railway injects PORT automatically for the web service --> |

<!-- VERIFY: confirm whether REDIS_URL must be set in Railway or if Redis is not required in production -->
<!-- VERIFY: confirm whether UPLOAD_DIR needs to point to a mounted volume in Railway or if file uploads use an external object store in production -->
