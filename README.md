<!-- generated-by: gsd-doc-writer -->
# FoodBI

Restaurant analytics SaaS for Kazakhstan restaurants, built on top of the iiko POS system.

FoodBI pulls sales, stock, and purchasing data from iiko via OLAP reports and REST endpoints, stores it in a multi-tenant PostgreSQL database, and serves it through a mobile-first web app with revenue dashboards, stock tracking, AI-powered cost suggestions, and role-based employee management.

---

## Key Features

- **Revenue analytics** — Daily, weekly, and monthly revenue charts with per-dish breakdown, order counts, and average check. All monetary values in KZT with no subunits.
- **Stock management** — Current stock levels with recipe component resolution. Override support for non-standard units.
- **AI suggestions** — Cost and pricing recommendations generated per location, surfaced as actionable cards.
- **Purchases tracking** — Purchase line items synced from iiko with supplier alias resolution.
- **Multi-location support** — Switch between restaurant locations via the location switcher; each location has its own iiko credentials and sync schedule.
- **Role-based employees** — Owner, general manager, manager, bartender, waiter, cashier, and accountant roles with permission gating on sensitive views.
- **Multi-language UI** — English, Russian (default), Kazakh, and Spanish via a key-based i18n store.
- **iOS app** — Capacitor wrapper ships to TestFlight / App Store alongside the PWA.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend API | Go 1.26, Chi v5, pgx/v5 |
| Database | PostgreSQL with RLS tenant isolation via `app.current_tenant` |
| Sync worker | Go, iiko Server API (OLAP + REST) |
| Frontend | React 19, TypeScript 6, Vite 8, TailwindCSS 4, Recharts 3 |
| State | Zustand 5, TanStack Query 5 |
| Mobile | Capacitor 8 (iOS) |
| Auth | JWT (golang-jwt/jwt v5) |
| Deployment | Railway (backend + PostgreSQL), Vite static build (frontend) |

---

## Project Structure

```
FoodBI/
├── backend/
│   ├── cmd/
│   │   ├── api/          # HTTP API server entry point
│   │   └── sync/         # iiko sync worker entry point
│   ├── internal/
│   │   ├── ai/           # AI suggestion generation
│   │   ├── auth/         # JWT auth middleware and handlers
│   │   ├── dashboard/    # Dashboard aggregation endpoints
│   │   ├── employees/    # Employee CRUD and role management
│   │   ├── iiko/         # iiko API client (OLAP + REST)
│   │   ├── locations/    # Location management
│   │   ├── purchases/    # Purchase line items
│   │   ├── revenue/      # Revenue facts
│   │   ├── stock/        # Stock levels and overrides
│   │   ├── sync/         # Sync orchestration service
│   │   └── ...           # profiles, payments, notifications, etc.
│   └── migrations/       # PostgreSQL migration files (000001–000020+)
├── frontend/
│   ├── src/
│   │   ├── components/   # Shared UI components (charts, layout, ui primitives)
│   │   ├── pages/        # Route-level pages (Dashboard, Revenue, Stock, AI, etc.)
│   │   ├── stores/       # Zustand global stores
│   │   ├── i18n/         # Translation files: en, ru, kk, es
│   │   └── lib/          # Utilities (format, currency, API client)
│   └── ios/              # Capacitor iOS project (App.xcodeproj)
├── Dockerfile            # Multi-stage build for backend API
├── docker-compose.yml    # Local development stack
├── railway.json          # Railway deployment configuration
└── CLAUDE.md             # Project-specific rules for AI-assisted development
```

---

## Quick Start

### Prerequisites

- Go >= 1.26 (`/opt/homebrew/bin/go` on macOS via Homebrew)
- Node.js >= 20
- PostgreSQL 16
- Docker (optional, for local DB via `docker-compose`)

### Backend

```bash
# Start local database
docker-compose up -d

# Copy env and configure
cp backend/.env.example backend/.env   # edit DATABASE_URL, JWT_SECRET, IIKO_* vars

# Run migrations
cd backend && go run cmd/api/main.go --migrate

# Start API server
/opt/homebrew/bin/go run cmd/api/main.go
```

The API listens on `http://localhost:8080` by default.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The dev server starts at `http://localhost:5173`.

---

## Documentation

| Doc | Description |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture, component diagram, data flow |
| [docs/GETTING-STARTED.md](docs/GETTING-STARTED.md) | Prerequisites and first-run walkthrough |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Local dev setup, build commands, code style |
| [docs/TESTING.md](docs/TESTING.md) | Test framework, running tests, CI integration |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | All environment variables and config file reference |
| [docs/API.md](docs/API.md) | HTTP endpoint reference, auth, request/response shapes |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Railway deployment, build pipeline, rollback |
| [DEPLOY_TESTFLIGHT.md](DEPLOY_TESTFLIGHT.md) | iOS TestFlight / App Store submission guide |
| [CLAUDE.md](CLAUDE.md) | Project rules for AI-assisted development (iiko rules, monetary formatting, UI conventions) |

---

## Production

<!-- VERIFY: production URL and auth details -->
The backend API is deployed to Railway at `https://foodbi-production.up.railway.app`.

Access requires a valid account — contact the project owner for credentials. Self-registration is not open to the public.

---

## iOS App

The iOS target lives at `frontend/ios/App/App.xcodeproj` and is built with Capacitor 8.

See [DEPLOY_TESTFLIGHT.md](DEPLOY_TESTFLIGHT.md) for the full TestFlight and App Store submission process, including Railway backend setup, environment variable injection, and Xcode signing configuration.

---

## License

Private and proprietary. All rights reserved. Not licensed for redistribution or commercial use outside of the FoodBI project.
