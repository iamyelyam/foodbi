<!-- generated-by: gsd-doc-writer -->
# Deployment

This document describes how FoodBI is built, deployed, and operated in production.

---

## Production Stack

| Component | Platform | Notes |
|---|---|---|
| Backend API (`./api`) | Railway web service | Deployed from `Dockerfile` via `railway.json` |
| PostgreSQL database | Railway PostgreSQL plugin | Attached to the backend service |
| iOS / TestFlight | Apple App Store Connect | Capacitor-wrapped Vite build; see [iOS / TestFlight](#ios--testflight) |

Production API base URL: `https://foodbi-production.up.railway.app`

The frontend is a Capacitor iOS application — the compiled Vite assets are bundled into the iOS app via `npx cap sync` and distributed through TestFlight. There is no separate web hosting for the frontend.

---

## CI/CD Pipeline

### GitHub Actions (`.github/workflows/ci.yml`)

Every push to `main` and every pull request against `main` triggers two parallel jobs:

| Job | Steps |
|---|---|
| `backend` | `go vet ./...` then `go test ./...` (uses Go version from `backend/go.mod`) |
| `frontend` | `npm ci` → `tsc --noEmit` → `bash scripts/lint-money.sh` → `npm run build` |

CI does **not** deploy — it only validates. Deployment is handled by Railway's GitHub integration.

### Railway auto-deploy

When the `main` branch passes CI and is pushed to `github.com/iamyelyam/foodbi`, Railway detects the push and redeploys the backend service automatically. <!-- VERIFY: confirm Railway is connected to the iamyelyam/foodbi repo and that auto-deploy on push to main is enabled in the Railway project settings -->

---

## Build Process

The `Dockerfile` at the repository root uses a two-stage build:

**Stage 1 — builder (`golang:alpine`)**

```dockerfile
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api/
RUN CGO_ENABLED=0 GOOS=linux go build -o /sync ./cmd/sync/
```

Two static binaries are produced: `/api` (the HTTP server) and `/sync` (the iiko sync worker).

**Stage 2 — runtime (`alpine:3.19`)**

```dockerfile
COPY --from=builder /api .
COPY --from=builder /sync .
COPY backend/migrations ./migrations
EXPOSE 8080
CMD ["./api"]
```

The final image contains only the two binaries, the `migrations/` directory, and CA certificates. No Node.js toolchain or frontend assets are included — the frontend is distributed via iOS only.

`railway.json` instructs Railway to build from this Dockerfile and start with `./api`:

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

The health check endpoint returns `{"status":"ok","time":"..."}`. Railway polls `/health` after every deploy; the service is considered healthy when it responds with HTTP 200.

---

## Database Migrations

Migrations are plain SQL files in `backend/migrations/`. The `./api` binary applies all pending migrations automatically on every startup before the HTTP server begins accepting connections:

```go
// backend/cmd/api/main.go
if err := database.RunMigrations(ctx, db, migrationsDir); err != nil {
    log.Warn().Err(err).Msg("migration: runner failed — continuing to serve")
}
```

Migration failures are logged as warnings — the server continues to start. The migrations directory is copied into the Docker image at `./migrations` (resolved via the `MIGRATIONS_DIR` env var, defaulting to `./migrations`).

**On a fresh Railway deploy**, the first startup creates all tables from scratch. No manual `migrate` command is needed.

---

## Environment Variables on Railway

Set these in the Railway project dashboard under the backend service → **Variables**.

| Variable | Source | Required | Notes |
|---|---|---|---|
| `DATABASE_URL` | Railway auto-injected | Yes | Provided automatically when the PostgreSQL plugin is attached to the project <!-- VERIFY: confirm Railway auto-injects DATABASE_URL from the attached Postgres plugin --> |
| `PORT` | Railway auto-injected | Yes | Railway sets this to the port the service must listen on; do not override <!-- VERIFY: confirm Railway injects PORT automatically for the web service --> |
| `JWT_SECRET` | Manual | Yes | Generate with `openssl rand -hex 32`; must be set before any user can log in |
| `ENV` | Manual | Recommended | Set to `production` to enable structured JSON logging instead of console output |
| `TELEGRAM_BOT_TOKEN` | Manual | No | Enables the payment notification Telegram bot; omit to disable the bot |
| `MIGRATIONS_DIR` | Manual | No | Defaults to `./migrations`; only override if the image layout changes |
| `UPLOAD_DIR` | Manual | No | Defaults to `./uploads`; uploaded invoice files are stored here <!-- VERIFY: confirm whether file uploads persist across Railway redeployments (ephemeral filesystem) or require an external volume/object store in production --> |
| `REDIS_URL` | Manual | No | Defaults to `redis://localhost:6379`; <!-- VERIFY: confirm which backend features require Redis and whether a Redis instance is attached in the Railway project --> |

iiko credentials (`iiko_server_url`, `iiko_login`, `iiko_password`) are **not** environment variables. They are stored per-company in the `companies` table and configured by the owner through the app's settings UI.

See [docs/CONFIGURATION.md](CONFIGURATION.md) for the full variable reference including defaults and source locations.

---

## Frontend Deployment (iOS / Capacitor)

The React/Vite frontend is not hosted as a web service. It is packaged as an iOS application using Capacitor and distributed via TestFlight.

The production API URL is baked into the frontend at build time via `frontend/.env.production`:

```
VITE_API_URL=https://foodbi-production.up.railway.app
```

### Full TestFlight workflow

See **[DEPLOY_TESTFLIGHT.md](../DEPLOY_TESTFLIGHT.md)** at the repository root for the complete step-by-step guide. Summary:

1. `cd frontend && npm run build` — Vite build to `dist/`
2. `npx cap sync` — copies `dist/` into the `ios/` Xcode project bundle
3. `npx cap open ios` — opens Xcode
4. In Xcode: set Bundle ID `app.foodbi.kz`, select team, choose **Any iOS Device (arm64)**
5. **Product → Archive**, then **Distribute App → App Store Connect → Upload**
6. In App Store Connect → **TestFlight**: add internal testers

**After a backend-only change**: push to `main`; Railway redeploys automatically. No iOS rebuild needed.

**After a frontend change**: run `npm run build && npx cap sync`, re-Archive in Xcode, upload a new build to TestFlight.

---

## Production Seed

The production database is seeded with real restaurant data. The initial owner account is:

- **Login**: `yelyam@choco.kz`
- **Password**: `Smart123`
- **Company**: Палаушы (restaurant in Almaty, Kazakhstan)

iiko credentials for each location are stored in `companies.iiko_server_url`, `companies.iiko_login`, and `companies.iiko_password`. To add a new company on a fresh deployment, connect via Railway's `psql` console and insert a row into `companies`, then create the owner user and locations.

---

## Rollback

Railway retains previous deployment builds. To roll back:

1. Open the Railway project dashboard.
2. Navigate to the backend service → **Deployments** tab.
3. Find the last known-good deployment and click **Redeploy**.

<!-- VERIFY: confirm Railway's exact UI path for redeploying a previous build (dashboard labels may change) -->

Database migrations are not automatically reversed on rollback. If a migration introduced a breaking schema change, apply the corresponding `.down.sql` file manually via Railway's `psql` console before redeploying the older binary.

---

## Monitoring

| Method | Details |
|---|---|
| Railway logs | Real-time structured JSON logs from `./api` and `./sync`; accessible in the Railway dashboard under the service → **Logs** tab <!-- VERIFY: confirm Railway log retention period --> |
| Health endpoint | `GET https://foodbi-production.up.railway.app/health` returns `{"status":"ok","time":"..."}` |
| Sync status | `GET /api/v1/locations/sync-status` (authenticated) returns the last sync timestamp and status per location |
| Post-sync validation | After each iiko sync cycle the service asserts `SELECT MAX(revenue) FROM revenue_facts` is > 10,000 per day; failures are logged as warnings |

No external APM (Sentry, Datadog, New Relic) is currently configured. <!-- VERIFY: confirm no external error tracking or APM is attached to the Railway project -->
