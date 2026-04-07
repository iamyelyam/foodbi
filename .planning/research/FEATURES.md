# Feature Landscape: Restaurant BI SaaS

**Domain:** Multi-tenant restaurant business intelligence platform (iiko-integrated)
**Researched:** 2026-04-07
**Confidence:** MEDIUM-HIGH (primary sources: G2 reviews, vendor documentation, industry analysis)

---

## Context

FoodBI is a mobile-first BI layer on top of iiko Cloud API v2. It is NOT a POS, NOT a CRM, NOT an accounting system. The feature set is analytics-and-operations-visibility — owners and employees see what's happening across locations without manually pulling iiko reports.

Competitive reference points: MarketMan, Restaurant365, MarginEdge, Lightspeed Advanced Insights, Supy, iiko built-in analytics.

---

## Table Stakes

Features users expect. Missing = product feels incomplete or users stay in iiko's native reports.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Revenue dashboard (daily/weekly/monthly) | Every BI tool has this; operators check revenue daily | Low | Sales totals, trends, period comparison |
| Revenue breakdown by location | Multi-location owners compare sites constantly | Low | Per-location P&L view is baseline expectation |
| Revenue breakdown by product/category | Menu engineering requires this | Medium | Needs iiko order data sync |
| Orders list with filters (date, status, location) | Operators investigate anomalies order-by-order | Medium | Pagination, status filters, detail drilldown |
| Food cost / COGS visibility | Protects margins; labor + food = prime cost | Medium | Requires purchase + sales data correlation |
| Purchases list (by supplier, date, amount) | Owners want to audit what was ordered and at what cost | Low | Direct from iiko purchase orders |
| Purchase detail view | Needed to verify invoices, spot overcharges | Low | Line-item breakdown per order |
| Stock / inventory current levels | Stockouts are daily operational pain | Medium | Requires iiko stock sync |
| Low-stock alerts | Prevents revenue loss from 86'd items | Medium | Threshold-based push notifications |
| Supplier directory | Needed before any ordering workflow | Low | Name, contact, catalog linkage |
| Transfer log between locations | Multi-location chains move stock constantly; without this, balances drift | Medium | Source → destination, quantity, timestamp |
| Role-based access (Owner vs Employee) | Sensitive financial data must be access-controlled | Medium | Owner sees all; Employee sees their location only |
| Multi-location switcher | Owners operate 2–10+ locations | Low | Location context affects all data views |
| Mobile-optimized UI (375px) | Operators check metrics from phone between service | Medium | Touch targets, scroll-safe charts |
| Date range filtering on all data | Standard analytics expectation | Low | Today, week, month, custom range |
| Notifications center | Alerts on stock, anomalies, approvals | Medium | In-app + push; role-filtered |

---

## Differentiators

Features that set the product apart. Not universally expected, but create lock-in and word-of-mouth.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| AI purchase recommendations | Tells the owner what to order based on sales velocity + current stock | High | MarketMan and Restaurant365 both doing this; early movers win; requires ML or LLM layer on top of iiko data |
| AI menu optimization suggestions | Identifies underperforming dishes, recommends price adjustments or 86ing items | High | Lightspeed Advanced Insights does this; high perceived value for owners |
| Invoice scanning (camera → line items) | Eliminates manual invoice entry; MarginEdge built their brand on this | High | OCR + AI classification; MarginEdge processes 10M+/yr at 99% accuracy; FoodBI can start with photo upload + manual verification |
| Anomaly detection / variance alerts | Flags when actual COGS deviates from theoretical (e.g. theft, waste, shrinkage) | High | Actual vs. theoretical cost comparison is a Restaurant365 differentiator |
| Supply request workflow (Supplying module) | Structured ordering request from location → supplier, with approval | Medium | Replaces WhatsApp/phone ordering; creates audit trail |
| Cross-location performance benchmarking | Shows which location has best food cost %, best revenue/cover, etc. | Medium | Requires normalized metrics across locations |
| Revenue vs. purchase trend overlay | One chart showing if purchase costs are growing faster than revenue | Medium | Simple to display, high insight value |
| Profit / margin view (not just revenue) | Owners care about margin, not just top-line; rare in mobile BI tools | Medium | Needs purchase data correlated with revenue |
| Export to PDF/CSV | Finance team and accountants need data out | Low | Date-range reports per location or consolidated |
| Employee performance metrics | Revenue-per-employee, orders-per-shift — not scheduling | Medium | Sensitive; requires careful RBAC; out of scope for MVP |

---

## Anti-Features

Features to explicitly NOT build in v1. These expand scope without proportional value for FoodBI's core use case.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Staff scheduling | Complex HR product; 7shifts and HotSchedules own this; iiko has native scheduling | If labor data is needed, pull from iiko read-only; don't build scheduling UI |
| Payroll processing | Requires financial compliance, integrations with payment rails | Out of scope; R365 took years to build this correctly |
| Full accounting / GL | Restaurant365 is the right tool for this; not a BI differentiator | Show cost summaries; don't recreate chart of accounts |
| Customer / guest CRM | Guest 360 is a separate product in the SR ecosystem; FoodBI is operator-facing | Keep FoodBI strictly B2B operator analytics |
| Online ordering / delivery management | Entirely different product; Delivery SR owns this in the ecosystem | Not in scope |
| Recipe / ingredient costing engine | Requires maintaining ingredient databases, recipes, yields — high maintenance | Pull theoretical cost from iiko if available; don't build a recipe DB |
| Loyalty / rewards program | CRM adjacent; high complexity; separate product domain | Not in scope |
| Food safety / temperature logs | Niche compliance feature; different buyer persona (ops managers, not owners) | Not in scope for v1 |
| Custom report builder | High development cost; most operators want pre-built views | Provide curated reports with filters; custom builder in v2+ |
| Native iOS/Android apps | Already out of scope per PROJECT.md; WebView is faster to ship | WebView with mobile-first CSS |

---

## Feature Dependencies

```
iiko Cloud API v2 sync
  └── Revenue module (orders, products, statuses)
  └── Purchases module (purchase orders, suppliers)
  └── Stock module (inventory levels, movements)
       └── Transfer module (stock → transfer records)
       └── Low-stock alerts (thresholds on stock levels)
  └── Supplier directory (vendor data from iiko)

Auth + RBAC (Owner / Employee roles)
  └── Multi-location context (location switcher)
       └── All data modules (filtered by location access)
       └── Cross-location benchmarking (requires multi-location access)

Revenue module
  └── Statistics / trend charts
  └── Revenue vs. purchase overlay (depends on Purchases too)
  └── AI menu recommendations (depends on revenue + product breakdown)

Purchases module
  └── Invoice scanning (photo → purchase record)
  └── Supplier management (supplier linked to purchase)
  └── Supplying (purchase request workflow)
  └── AI purchase recommendations (depends on stock + sales velocity)

Notifications
  └── Low-stock alerts (depends on Stock)
  └── Approval notifications (depends on Supplying workflow)
```

---

## MVP Recommendation

**Ship these to validate core value:**

1. Auth (Owner + Employee, multi-tenant, RBAC)
2. iiko sync (revenue + purchases — the two highest-value data domains)
3. Revenue dashboard (daily/weekly/monthly, by location, by product)
4. Purchases module (list, filters, detail, by supplier)
5. Stock overview (current levels, low-stock threshold alerts)
6. Transfer log (history with filters)
7. Notifications center (low-stock, basic alerts)
8. Location management (switcher, add location)

**Defer to phase 2:**
- Supplying workflow (ordering requests): medium complexity; validates after MVP
- Invoice scanning: high complexity OCR; ship photo upload first, smart extraction later
- AI recommendations: needs 30+ days of synced data before it can generate useful signals; build after data pipeline is stable
- Employee performance metrics: requires careful RBAC design; lower priority than financial visibility
- Cross-location benchmarking: requires normalized data model; build after per-location views are proven

**Never build (scope boundary):**
- Staff scheduling, payroll, full accounting, CRM, delivery management, native apps

---

## Competitive Positioning

| Capability | MarketMan | Restaurant365 | MarginEdge | FoodBI Target |
|------------|-----------|---------------|------------|---------------|
| Revenue analytics | Via POS integration | Strong | Basic | Strong (iiko-native) |
| Purchase management | Core feature | Strong | Strong | Strong (iiko-native) |
| Stock management | Core feature | Strong | Basic | Strong (iiko-native) |
| Invoice scanning | Yes (AI) | Yes | Yes (market leader) | Phase 2 (OCR) |
| AI recommendations | Yes (ordering) | Growing | Growing | Phase 2 |
| Multi-location | Yes | Yes | Yes | Yes (core) |
| Mobile-first UX | Partial | Limited | Limited | Core differentiator |
| iiko-native integration | No | No | No | Exclusive advantage |
| Supplier management | Yes | Yes | Yes | Yes (basic v1) |
| Accounting / GL | No | Yes | Partial | Never |
| Staff scheduling | No | Yes | No | Never |

FoodBI's exclusive advantage is deep iiko integration. No competitor is iiko-native. This means FoodBI can offer zero-configuration setup for iiko users and richer data fidelity than generic integrations.

---

## Sources

- G2 Restaurant Business Intelligence category — https://www.g2.com/categories/restaurant-business-intelligence-analytics
- MarketMan AI inventory features — https://www.marketman.com/platform/restaurant-inventory-management-software
- MarginEdge invoice automation — https://www.marginedge.com/automated-invoice
- Restaurant365 features overview — https://www.restaurant365.com/
- Lightspeed Advanced Insights — https://forcked.com/lightspeed-restaurant-pos-review
- Multi-location inventory management patterns — https://supy.io/blog/multi-location-restaurant-inventory-management/
- AI in restaurant operations 2025 — https://pos.toasttab.com/blog/on-the-line/ai-restaurant-data
- Restroworks BI software guide — https://www.restroworks.com/blog/best-restaurant-business-intelligence-software/
- xenia.team restaurant analytics roundup — https://www.xenia.team/articles/restaurant-analytics-software
