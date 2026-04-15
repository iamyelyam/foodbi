# FoodBI → TestFlight Deploy Guide

## Phase 1: Railway (backend + PostgreSQL)

1. Go to [railway.app](https://railway.app) → **New Project** → **Deploy from GitHub repo** → pick your FoodBI repo.
2. Railway detects the `backend/` directory via Dockerfile — select **Root Directory = `backend`**.
3. Add **PostgreSQL** to the project (**New → Database → Add PostgreSQL**). Railway creates a `DATABASE_URL` variable automatically.
4. In the **backend service** → **Variables**, paste:
   - Railway's `DATABASE_URL` is auto-injected (reference it from postgres plugin if not).
   - `JWT_SECRET` — generate with `openssl rand -hex 32`
   - `ENV=production`
   - `PORT` — Railway auto-sets; don't override.
   - iiko credentials live in DB (`companies.iiko_server_url`, `iiko_login`, `iiko_password`), not env — add them via SQL after migrations run.
5. Deploy. Watch logs for `migration: done`. The app auto-runs `./migrations/*.up.sql` on every boot (via `internal/database/migrate.go`).
6. Once deployed, copy the public URL (e.g. `https://foodbi-api-production.up.railway.app`) → paste into `frontend/.env.production` as `VITE_API_URL`.
7. Seed your company + user via SQL (use Railway's `Connect` → `psql`):
   ```sql
   INSERT INTO companies (id, name, iiko_server_url, iiko_login, iiko_password)
   VALUES (uuid_generate_v4(), 'Палаушы', 'https://...iiko-server...', 'login', 'password')
   RETURNING id;
   -- copy id, then create owner user + location
   ```

## Phase 2: Capacitor iOS build (local)

Prereqs: macOS, Xcode installed, cocoapods (`sudo gem install cocoapods`).

```bash
cd frontend
npm run build          # Vite build → dist/
npx cap add ios        # Creates ios/ folder with Xcode project
npx cap sync           # Copies dist/ into the iOS app bundle
npx cap open ios       # Opens Xcode
```

In Xcode:
1. Left sidebar → **App** target → **Signing & Capabilities**:
   - Team: your Apple Developer team
   - Bundle ID: `kz.foodbi`
   - Check "Automatically manage signing"
2. Top bar → select **"Any iOS Device (arm64)"**.
3. Menu: **Product → Archive**. Wait for build.
4. When Archive window opens: **Distribute App → App Store Connect → Upload**.
5. After upload (~5–10 min processing), go to [appstoreconnect.apple.com](https://appstoreconnect.apple.com) → **TestFlight** tab.
6. Add internal testers (yourself) → they get invite email → install **TestFlight** app on iPhone → install FoodBI.

## Updates
After any frontend change: `npm run build && npx cap sync` then re-Archive in Xcode.
After any backend change: push to GitHub — Railway redeploys automatically.
