# FoodBI Autopilot Spec — Full Implementation

## Current State

**Built:** 6 phases scaffolded — 16 frontend pages, 14 backend modules, 5 DB migrations, all compiling.
**Gap:** Pages are single-view scaffolds. Figma has ~100 screens across 14 sections with multi-state flows (overlays, filters, modals, errors, empty states). Backend handlers are basic CRUD — need pagination hardening, proper error responses, and missing endpoints.

## Execution Plan — 8 Work Streams

### WS1: Auth Flow Completion (7 Figma screens)
**Screens:** Enter email (4 states), Password (2 states), OTP (2 states), Sign up Owner, Sign up Employee, Enable Notifications, Enable Face ID
**Backend gaps:**
- Employee invite flow (generate invite link, accept invite)
- Password reset endpoint
- Onboarding progress tracking (notifications/faceid preferences)
**Frontend gaps:**
- Split LoginPage into email-first → password step (matches Figma 2-step flow)
- Sign up Employee screen (separate from Owner)
- Onboarding wizard: Enable Notifications → Enable Face ID → Done (with ProgressBar)
- Keyboard-aware layouts (Figma shows iOS keyboard overlays)
- Error states on all input fields

### WS2: Home + Dashboard (8 Figma screens)
**Screens:** Main page Revenue, Main page Purchase, Location Change overlay, Window status/Review, Statistics Revenue, Statistics Profit, Statistics Profit with filters, Date range picker
**Backend gaps:**
- Dashboard should return location name alongside data
- Location change should return available locations with last sync time
**Frontend gaps:**
- Segmented control on dashboard: Revenue / Purchase view toggle
- Location change BottomSheet with search
- Revenue trend chart needs proper date axis formatting
- Period comparison badge (% vs last period)
- Skeleton loading states for all cards

### WS3: Revenue Module (8 Figma screens)
**Screens:** Orders list, Orders with status overlay, Window status/Review, Products list, Product details, Filters BottomSheet, Window/Filters, Transfer view + DatePicker
**Backend gaps:**
- Order status update endpoint (review → approved/rejected)
- Product detail: daily sales trend for last 30 days
- Revenue by category aggregation endpoint
**Frontend gaps:**
- Order status overlay (BottomSheet showing order details + status change)
- Product detail page (chart + metrics)
- Full filter BottomSheet: date range, status, order type
- Transfer view with integrated DatePicker organism

### WS4: Purchases Module (6 Figma screens)
**Screens:** Purchases list, Purchases with filters overlay, Purchases with filters applied, Supplier detail (Coca Cola LLC), Supplier detail with purchases, Supplier details full
**Backend gaps:**
- Supplier contact info storage (phone, email, address)
- Purchase line items (not just totals)
**Frontend gaps:**
- Supplier detail page (full page, not just BottomSheet)
- Purchase detail page with line items
- Multi-filter BottomSheet (date + supplier + amount range)

### WS5: Operations Module (28 Figma screens)
**Screens:**
- Stock Management (2): inventory list, low-stock view
- Supplying (11): list, create flow (supplier → category → products → quantities → confirm), success snackbar
- Transfer (15): list, create flow (location → category → products → quantities → confirm), history with filters, segmented control
**Backend gaps:**
- Categories endpoint (for product selection in supply/transfer flows)
- Product catalog from iiko (for selection UI)
- Transfer complete/cancel status update
- Supply request with multi-step validation
**Frontend gaps:**
- Multi-step creation wizards (Supplying: 5 steps, Transfer: 5 steps)
- Category selection screen with search
- Product selection with quantity input (keyboard overlay)
- Confirmation screen with review of all items
- Success Snackbar component
- Segmented control (Requests / History tabs)
- History page with filter BottomSheet

### WS6: People Module (16 Figma screens)
**Screens:**
- Employees (8): list, list with snackbar, adding form, role selection (BottomSheet), location selection (BottomSheet), filled form, error form, designer comments
- Profile (4): personal profile, employee profile view, employee profile edit
- Employee Home (1): restricted dashboard view
**Backend gaps:**
- Employee invite via email (send invite link)
- Employee deactivation
- Role-based dashboard data filtering
**Frontend gaps:**
- Role selection BottomSheet (list of roles)
- Location selection BottomSheet (list of locations with checkboxes)
- Form validation with error states (red borders, error messages per field)
- Success Snackbar after employee added
- Employee restricted home screen (fewer tabs, limited data)

### WS7: Intelligence Module (17 Figma screens)
**Screens:**
- AI Suggestions (7): recommendations list, drill-down detail, action cards, create task, task list, task detail, task assignment
- File Upload (10): scanning camera view, scanning with preview, edit invoice, share sheet, upload screen, upload with action sheet, file/camera access modals, upload progress
**Backend gaps:**
- AI suggestion detail endpoint with actionable data
- Task assignment to employee
- Invoice OCR processing (placeholder/mock for now)
- File download/serve endpoint
**Frontend gaps:**
- Camera scanning UI (camera viewfinder, shutter button)
- Invoice edit form (extracted fields)
- Share sheet BottomSheet
- File/camera permission modal dialogs
- Upload progress indicator
- AI suggestion drill-down page
- Task creation form with assignee selection

### WS8: Notifications + Location Management (6 Figma screens)
**Screens:**
- Notifications (2): badge on header, notification list with read/unread
- Add Location (4): location change overlay, add location form, filled form, designer comment
**Backend gaps:**
- Push notification token registration
- Notification badge count in header response
**Frontend gaps:**
- Header notification badge (red dot with count)
- Location form with proper validation and address autocomplete placeholder
- Notification grouping by date

## Component Library Gaps (Design System)

Missing from Figma atoms/molecules/organisms:
- **SegmentedControl** — used in Revenue, Purchases, Supplying, Transfers
- **Snackbar/Toast** — success/error notifications
- **DatePicker** — calendar organism
- **SearchBar** — used in Supplying category/product selection
- **Checkbox** — used in multi-select (locations, products)
- **Toggle** — used in settings
- **ProgressBar** — used in onboarding
- **FilterChip** — active filter pills
- **Skeleton** — loading placeholder
- **EmptyState** — consistent empty views
- **Modal/Alert** — camera/file access permissions

## Priority Order

1. **WS8 + Component Library** — shared components needed by all other streams
2. **WS1** — auth is the entry point
3. **WS2** — dashboard is where users land
4. **WS3 + WS4** — core analytics value
5. **WS5** — operations workflows
6. **WS6** — people management
7. **WS7** — intelligence layer (highest complexity, lowest priority for MVP)
