# Future Work

This document outlines features and improvements planned for future development, organized by priority and category.

---

## 🔴 High Priority

### 1. User Access Control (`internal/auth/`)
The `internal/auth/` directory is currently empty. No authentication or authorization exists.

**Required:**
- Login / registration endpoints (JWT or session-based)
- Role-based access control (admin, manager, staff)
- Auth middleware to protect API routes
- Login page on the frontend
- Protected routes and conditional UI elements based on roles

### 2. Search & Filter on Products & Inventories Pages
Only the Stock page has search/filter capabilities. Products and Inventories need the same.

**Required:**
- Search input (by name) on Products page
- Search input (by name) on Inventories page
- Backend query params for name-based search on both endpoints

### 3. Configurable Low Stock Thresholds
The low stock threshold is currently hardcoded (`quantity < 5`). Products need a configurable minimum stock level.

**Required:**
- `min_stock` column on the Product model
- Backend API to set/update min_stock per product
- Dashboard and alerts use `product.min_stock` instead of hardcoded value
- Frontend form field for min_stock in product create/edit modal

---

## 🟠 Medium Priority

### 4. ✅ Barcode / SKU Support (implemented)
Products carry a required `sku` and an optional `barcode`, both unique **within a
company** (not globally) — two tenants may legitimately share an identifier.

**Shipped:**
- `sku` / `barcode` columns on the Product model, enforced by partial unique
  indexes over `deleted_at IS NULL`, so a soft-deleted product does not permanently
  reserve an identifier and an omitted barcode never collides.
- Products created before SKUs existed were backfilled with `LEGACY-<id>` SKUs
  during migration.
- `GET /api/products/scan?code=…` — resolves a scanned code (SKU first, then
  barcode) to a single product.
- `GET /api/products?search=…` — matches name, SKU or barcode; the Stock list
  search matches them too.
- Frontend: SKU/barcode columns, product form fields and details view; debounced
  search box; scan input on the Products page and on the Stock page (the latter
  opens the pre-filled "Add Stock" dialog).

**Still open:** camera-based scanning. A hardware barcode scanner works today —
those emit keystrokes ending in Enter, which the scan inputs already handle.

### 5. Stock Aging & Reporting
The dashboard only shows basic counts. No inventory aging or advanced reporting exists.

**Required:**
- Backend endpoint for stock aging report (items grouped by days in stock: 0–30, 31–90, 90+)
- Inventory turnover rate calculation (COGS / average inventory)
- Frontend reporting tab or page with aging table and turnover metrics
- Export to CSV/Excel

### 6. Batch / Serial Number Tracking
No lot tracking or serial number support exists on the Stock model.

**Required:**
- `batch_number` and/or `serial_number` columns on Stock
- Backend filtering by batch/serial
- Frontend batch input on stock add forms
- Traceability view showing entry/exit history for a batch

### 7. Audit Log / Activity History
No record of who did what and when.

**Required:**
- `audit_logs` table recording all stock movements (action, user, inventory, product, old_qty, new_qty, timestamp)
- Backend endpoints to query audit logs
- Frontend activity log page or tab

---

## 🟡 Lower Priority

### 8. Order Management
No purchase orders, sales orders, backorders, or returns processing.

**Required:**
- `purchase_orders` and `sales_orders` models
- Order lifecycle: draft → confirmed → shipped → received (or fulfilled)
- Backorder tracking
- Returns processing (RMA)
- Frontend order management pages

### 9. Automated Reordering & Notifications
No automated PO generation or alert system.

**Required:**
- Background job checking stock levels against thresholds
- Auto-generate purchase orders when stock falls below min_stock
- Email or in-app notifications for low stock
- Configurable notification preferences

### 10. Demand Forecasting
No historical sales data or predictive analytics.

**Required:**
- Sales history tracking (could tie into order management)
- Basic forecasting models (moving average, trend analysis)
- Frontend forecast visualization (charts)

### 11. Multi-Channel Integration
No e-commerce or external platform connectivity.

**Required:**
- Webhook system for real-time sync
- API connectors for Shopify, Amazon, WooCommerce, etc.
- Channel-specific inventory allocation

### 12. Integration Capabilities
No API key system or third-party connectors.

**Required:**
- API key management for external integrations
- Webhook endpoints for event notifications (stock updated, low stock, etc.)
- Connectors for accounting software (QuickBooks, Xero) and ERPs

### 13. Mobile Accessibility
The frontend is responsive but not optimized for mobile use.

**Required:**
- Progressive Web App (PWA) support (service worker, manifest, offline cache)
- Touch-optimized UI components for warehouse floor use
- Optional: dedicated mobile app (React Native)

---

## 🔧 Technical Improvements

### 14. Loading Skeletons
The app currently shows a raw `<Spin />` component while loading. Replace with Ant Design `Skeleton` components for a smoother UX.

### 15. Error Boundaries
No React error boundaries exist. A single API crash can blank the entire page.

### 16. Pagination Enhancements
- Add page size changer to all tables (currently stuck at 20)
- Show total record count consistently

### 17. Bulk Operations
- Select multiple stock rows and perform batch add/deduct
- Select multiple products/inventories for batch delete

### 18. Dark Mode
Add dark mode toggle using Ant Design's `ConfigProvider` `theme` prop.

### 19. Database Migrations
Currently using GORM AutoMigrate. For production, versioned migrations would be safer.

### 20. Testing
- Unit tests for service and repository layers
- Integration tests for API endpoints
- Frontend component tests
- E2E tests for critical flows

### 21. CI/CD
- GitHub Actions (or similar) for linting, testing, and building
- Docker image publishing
- Automated deployment pipeline

---

## 📊 Feature Coverage Summary

| Category | Current | Planned |
|----------|---------|---------|
| Multi-Location Management | ✅ Complete | — |
| Real-Time Tracking | ⚠️ Basic | Audit log (#7) |
| Dashboard & Analytics | ⚠️ Basic | Aging & turnover (#5) |
| User Access Control | ❌ Missing | Auth system (#1) |
| Search & Filters | ⚠️ Partial | Products & Inventories (#2) |
| Low Stock Alerts | ⚠️ Hardcoded | Configurable thresholds (#3) |
| Barcode/SKU | ✅ Complete | Camera scanning (#4) |
| Batch/Serial Tracking | ❌ Missing | Lot tracking (#6) |
| Order Management | ❌ Missing | PO/SO system (#8) |
| Automated Reordering | ❌ Missing | Notifications (#9) |
| Demand Forecasting | ❌ Missing | Analytics (#10) |
| Multi-Channel | ❌ Missing | Integrations (#11) |
| Mobile | ⚠️ Basic | PWA / mobile app (#13) |
| Testing | ❌ Missing | Test suite (#20) |
| CI/CD | ❌ Missing | Pipelines (#21) |
