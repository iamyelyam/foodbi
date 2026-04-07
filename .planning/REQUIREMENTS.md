# Requirements: FoodBI

**Defined:** 2026-04-07
**Core Value:** Владелец ресторана видит актуальную аналитику по выручке, закупкам и складу всех своих локаций в одном мобильном приложении, данные берутся напрямую из iiko.

## v1 Requirements

### Authentication

- [ ] **AUTH-01**: User can sign up as Owner with email and password
- [ ] **AUTH-02**: User can sign up as Employee via invite link
- [ ] **AUTH-03**: User can log in with email and password
- [ ] **AUTH-04**: User receives OTP code for account activation
- [ ] **AUTH-05**: User session persists via JWT across browser refresh
- [ ] **AUTH-06**: User can log out from any screen
- [ ] **AUTH-07**: User can enable Face ID / biometric auth (WebView)
- [ ] **AUTH-08**: User can enable/disable push notifications during onboarding

### Multi-tenancy

- [ ] **TENANT-01**: Each company is isolated — users cannot see other companies' data
- [ ] **TENANT-02**: Company can have multiple locations (restaurants)
- [ ] **TENANT-03**: Owner can add new locations to their company
- [ ] **TENANT-04**: Owner can switch between locations — all data views update accordingly
- [ ] **TENANT-05**: PostgreSQL RLS enforces tenant isolation at database level

### iiko Integration

- [ ] **IIKO-01**: System connects to iiko Cloud API v2 using company-specific API key
- [ ] **IIKO-02**: Sync Worker pulls revenue/orders data every 15 minutes per location
- [ ] **IIKO-03**: Sync Worker pulls stock data every 30 minutes per location
- [ ] **IIKO-04**: Sync Worker pulls purchase invoices hourly per location
- [ ] **IIKO-05**: iiko token auto-refreshes before 1h expiry (at 45min)
- [ ] **IIKO-06**: Sync failures are logged and retried with exponential backoff
- [ ] **IIKO-07**: Owner can see sync status (last successful sync timestamp)

### Dashboard

- [ ] **DASH-01**: Owner sees daily/weekly/monthly revenue summary on main screen
- [ ] **DASH-02**: Owner sees purchase cost summary on main screen
- [ ] **DASH-03**: Dashboard shows period-over-period comparison (vs previous period)
- [ ] **DASH-04**: Dashboard displays data for currently selected location
- [ ] **DASH-05**: Dashboard loads in under 2 seconds (materialized views)

### Revenue

- [ ] **REV-01**: User can view list of orders with filters (date, status, location)
- [ ] **REV-02**: User can view order details with line items
- [ ] **REV-03**: User can view revenue breakdown by product/category
- [ ] **REV-04**: User can view product details (sales volume, revenue, trends)
- [ ] **REV-05**: User can filter revenue data by date range
- [ ] **REV-06**: User can see order status (open, closed, review)

### Purchases

- [ ] **PURCH-01**: User can view list of purchases by supplier and date
- [ ] **PURCH-02**: User can view purchase detail with line items
- [ ] **PURCH-03**: User can view supplier directory with contact info
- [ ] **PURCH-04**: User can view supplier profile and purchase history
- [ ] **PURCH-05**: User can filter purchases by date, supplier, amount

### Stock Management

- [ ] **STOCK-01**: User can view current stock levels per location
- [ ] **STOCK-02**: User receives low-stock alerts (threshold-based)

### Supplying

- [ ] **SUPPLY-01**: User can create supply request (select supplier, category, products)
- [ ] **SUPPLY-02**: User can set quantities for each product in request
- [ ] **SUPPLY-03**: User can review and confirm supply request
- [ ] **SUPPLY-04**: Owner can approve/reject supply requests
- [ ] **SUPPLY-05**: User can view list of supply requests with status

### Transfers

- [ ] **TRANS-01**: User can create transfer request between locations
- [ ] **TRANS-02**: User can select category and products for transfer
- [ ] **TRANS-03**: User can set quantities and confirm transfer
- [ ] **TRANS-04**: User can view transfer history with filters (date, location, status)
- [ ] **TRANS-05**: Transfer log shows source → destination, quantity, timestamp

### Employees

- [ ] **EMP-01**: Owner can view list of employees
- [ ] **EMP-02**: Owner can add new employee (name, email, phone, role)
- [ ] **EMP-03**: Owner can assign role (Owner/Employee) to user
- [ ] **EMP-04**: Owner can assign employee to specific location(s)
- [ ] **EMP-05**: Employee sees only their assigned location's data
- [ ] **EMP-06**: Owner can view employee profile

### Profile

- [ ] **PROF-01**: User can view their personal profile
- [ ] **PROF-02**: User can edit their profile information

### Notifications

- [ ] **NOTIF-01**: User can view notification center with all alerts
- [ ] **NOTIF-02**: User receives notifications for low-stock, approvals, sync failures
- [ ] **NOTIF-03**: Notifications are role-filtered (Owner sees all, Employee sees relevant)

### Statistics

- [ ] **STAT-01**: User can view revenue statistics with charts (line, bar)
- [ ] **STAT-02**: User can view profit/margin statistics
- [ ] **STAT-03**: User can filter statistics by custom date range
- [ ] **STAT-04**: User can compare revenue vs purchase trends

### File Upload

- [ ] **FILE-01**: User can scan invoice using camera
- [ ] **FILE-02**: User can upload invoice file from device
- [ ] **FILE-03**: User can edit scanned invoice data
- [ ] **FILE-04**: User can share invoice to other apps

### AI Suggestions

- [ ] **AI-01**: User can view AI-generated recommendations for menu optimization
- [ ] **AI-02**: User can view AI purchase recommendations based on sales velocity + stock
- [ ] **AI-03**: User can create task from AI suggestion
- [ ] **AI-04**: AI suggestions are based on company's historical iiko data

### Location Management

- [ ] **LOC-01**: Owner can add new location with details
- [ ] **LOC-02**: User can switch active location from any screen
- [ ] **LOC-03**: Location change updates all data views immediately

## v2 Requirements

### Advanced Analytics
- **ADV-01**: Cross-location performance benchmarking
- **ADV-02**: Anomaly detection / variance alerts (actual vs theoretical cost)
- **ADV-03**: Export reports to PDF/CSV
- **ADV-04**: Employee performance metrics (revenue-per-employee)
- **ADV-05**: Custom report builder

### Enhanced Auth
- **EAUTH-01**: Magic link login
- **EAUTH-02**: OAuth (Google)
- **EAUTH-03**: 2FA via authenticator app

## Out of Scope

| Feature | Reason |
|---------|--------|
| Native iOS/Android apps | WebView is faster to ship, one codebase |
| Staff scheduling | 7shifts/HotSchedules own this; iiko has native scheduling |
| Payroll processing | Financial compliance complexity |
| Full accounting / GL | Restaurant365 territory |
| Customer/Guest CRM | Guest 360 is separate product in SR ecosystem |
| Online ordering / delivery | Delivery SR owns this |
| Recipe/ingredient costing engine | High maintenance; pull theoretical cost from iiko |
| Loyalty/rewards program | CRM-adjacent; separate product domain |
| Food safety/temperature logs | Different buyer persona |
| Other POS integrations | Focus on iiko in v1 |

## Traceability

| Requirement | Phase |
|-------------|-------|
| AUTH-01..08 | Phase 1 |
| TENANT-01..05 | Phase 1 |
| IIKO-01..07 | Phase 2 |
| LOC-01..03 | Phase 2 |
| DASH-01..05 | Phase 3 |
| REV-01..06 | Phase 3 |
| PURCH-01..05 | Phase 3 |
| STAT-01..04 | Phase 3 |
| STOCK-01..02 | Phase 4 |
| SUPPLY-01..05 | Phase 4 |
| TRANS-01..05 | Phase 4 |
| EMP-01..06 | Phase 5 |
| PROF-01..02 | Phase 5 |
| NOTIF-01..03 | Phase 5 |
| FILE-01..04 | Phase 6 |
| AI-01..04 | Phase 6 |

---
*Last updated: 2026-04-07 after initialization*
