<!-- generated-by: gsd-doc-writer -->
# Getting Started

This guide walks a new developer (or restaurant owner) through cloning FoodBI and running it locally on macOS from scratch.

---

## Prerequisites

Install all of these before proceeding.

| Tool | Required version | Install |
|------|-----------------|---------|
| macOS | 13 Ventura or later | — |
| Go | `>= 1.26` | `brew install go` |
| Node.js | `>= 20` | `brew install node` |
| PostgreSQL | `16` (Homebrew native) | `brew install postgresql@16` |
| Xcode | Latest stable (iOS only) | Mac App Store |

**Go path note:** Homebrew installs Go at `/opt/homebrew/bin/go`. If you have a stale system Go at `/usr/local/bin/go` (commonly Go 1.19), it will be picked up first and the build will fail. Always verify:

```bash
which go          # should be /opt/homebrew/bin/go
go version        # should print go1.26.x or later
```

If it prints the wrong path, prepend Homebrew to your shell PATH:

```bash
echo 'export PATH="/opt/homebrew/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

**PostgreSQL note:** Use the Homebrew-native PostgreSQL 16 binary directly — do **not** use Docker for Postgres. A Docker postgres container also binds port 5432 and will silently conflict, causing connection errors that are hard to diagnose. If you have Docker postgres running, stop it before proceeding:

```bash
docker stop $(docker ps -q --filter ancestor=postgres) 2>/dev/null || true
```

---

## 1. Clone the repository

```bash
git clone https://github.com/iamyelyam/foodbi
cd foodbi
```

---

## 2. PostgreSQL setup

Start the native Homebrew PostgreSQL 16 server:

```bash
LC_ALL=C /opt/homebrew/opt/postgresql@16/bin/pg_ctl \
  -D /opt/homebrew/var/postgresql@16 \
  -l /opt/homebrew/var/log/postgresql@16.log \
  start
```

Create the database, user, and grant privileges:

```bash
/opt/homebrew/opt/postgresql@16/bin/psql postgres <<'SQL'
CREATE USER foodbi WITH PASSWORD 'foodbi' BYPASSRLS;
CREATE DATABASE foodbi OWNER foodbi;
GRANT ALL PRIVILEGES ON DATABASE foodbi TO foodbi;
SQL
```

Verify the connection:

```bash
psql postgres://foodbi:foodbi@localhost:5432/foodbi -c "SELECT 1;"
```

---

## 3. Backend environment

```bash
cd backend
cp .env.example .env
```

Open `.env` and set at minimum:

```dotenv
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=foodbi
DB_PASSWORD=foodbi
DB_NAME=foodbi

# JWT — replace with a long random string
JWT_SECRET=replace-me-with-a-secure-random-secret

# Redis (required for session/cache)
REDIS_URL=redis://localhost:6379

# Server
PORT=8080
ENV=development
```

See [docs/CONFIGURATION.md](CONFIGURATION.md) for the full variable reference and descriptions.

---

## 4. First run — backend API

Kill any stale Go processes from previous sessions before building (stale `go run` processes from `/var/folders` can survive reboots and overwrite DB state with old code):

```bash
pkill -9 -f foodbi 2>/dev/null || true
```

Build and start the API server. Database migrations apply automatically on startup:

```bash
cd backend
/opt/homebrew/bin/go build -o foodbi-api ./cmd/api
./foodbi-api
```

You should see log output ending with something like `listening on :8080`.

---

## 5. First run — sync service

The sync service pulls data from iiko into PostgreSQL. Run it once after the API is up:

```bash
cd backend
/opt/homebrew/bin/go build -o foodbi-sync ./cmd/sync
./foodbi-sync
```

**Note:** The sync service requires iiko credentials to be configured per-company in the database. Without valid iiko credentials the sync will log errors but will not crash the API. You can skip this step and add credentials later via the admin UI.

---

## 6. Frontend

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:3000` in your browser. The Vite dev server proxies all `/api` requests to the backend at `http://localhost:8080`.

---

## 7. Log in

**Production seed account** (if you restored a production DB dump):

- Email: `yelyam@choco.kz`
- Password: `Smart123`

**Fresh database** — no seed users exist. Register a new owner account:

1. Navigate to `http://localhost:3000/register`
2. Complete the registration flow (email + OTP verification)
3. The first user registered becomes the company owner

---

## 8. Verify it works

After logging in, check that these pages load without errors:

- `/` — Dashboard with metric cards
- `/revenue` — Revenue chart and table
- `/purchases` — Purchase line items
- `/stock` — Stock levels
- `/ai-suggestions` — AI-generated cost suggestions

If the Dashboard shows zeros or empty states, the sync service has not run yet (or iiko credentials are not configured). This is expected on a fresh install.

---

## 9. iOS development (optional)

Requires Xcode installed from the Mac App Store.

1. Build the frontend for Capacitor:
   ```bash
   cd frontend
   npm run build
   npx cap sync ios
   ```
2. Open the Xcode project:
   ```bash
   open ios/App/App.xcodeproj
   ```
3. Select a simulator target and press Run (`Cmd+R`).

For TestFlight archive and upload, see [DEPLOY_TESTFLIGHT.md](../DEPLOY_TESTFLIGHT.md) at the project root.

---

## Common first-run issues

**Port 5432 already in use**

Docker postgres is running and has taken port 5432. Stop it:

```bash
docker stop $(docker ps -q --filter ancestor=postgres) 2>/dev/null || true
```

Then restart the Homebrew postgres server (step 2).

---

**`connection refused` on psql after pg_ctl start**

The data directory path may differ if you installed postgresql@16 via a different method. Find the correct data dir:

```bash
/opt/homebrew/opt/postgresql@16/bin/pg_lsclusters 2>/dev/null \
  || ls /opt/homebrew/var/ | grep postgresql
```

---

**Migrations fail: `already exists` or duplicate migration error**

The `schema_migrations` table may be empty while migration files are already applied (this happens after restoring a DB dump without the migrations table). Backfill it from filenames:

```bash
psql postgres://foodbi:foodbi@localhost:5432/foodbi <<'SQL'
INSERT INTO schema_migrations (version, dirty)
SELECT regexp_replace(filename, '^(\d+)_.+\.up\.sql$', '\1')::bigint, false
FROM (SELECT unnest(ARRAY[
  '000001_init_schema.up.sql',
  '000002_fact_tables.up.sql'
  -- add all migration filenames that are already applied
]) AS filename) f
ON CONFLICT DO NOTHING;
SQL
```

---

**Stale `go run` binary from `/var/folders` overwrites DB data**

Old processes started with `go run` (rather than a compiled binary) can survive in `/var/folders` and keep running against the database. Always kill them before starting a new build:

```bash
pkill -9 -f foodbi 2>/dev/null || true
```

---

**iiko sync errors on first run**

The sync requires valid iiko Server credentials stored per-company in the `companies` table. Without them, the sync logs `401 Unauthorized` or similar and skips that company. This is expected on a fresh install — configure credentials via the admin UI or directly in the DB, then re-run `./foodbi-sync`.

---

## Next steps

- [docs/CONFIGURATION.md](CONFIGURATION.md) — full environment variable reference
- [docs/DEVELOPMENT.md](DEVELOPMENT.md) — build commands, code style, branch conventions
- [docs/ARCHITECTURE.md](ARCHITECTURE.md) — system overview and component diagram
