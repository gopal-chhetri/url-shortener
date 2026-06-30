# Implementation Plan - User & Admin Dashboards

This plan covers extending the database queries, building new API endpoints (analytics, status toggles, user/role management), and developing a premium frontend Single Page Application (SPA) under `web/`.

## User Review Required
> [!IMPORTANT]
> The frontend will be implemented as a Single Page Application using Vanilla HTML, CSS (Vanilla CSS with a premium glassmorphic dark theme), and Javascript. We will serve this frontend from the backend (static files serving) or directly as an independent static site.

---

## Proposed Changes

### 1. Database Queries & Regeneration

We will add queries for statistics aggregation and entity status/role updates, then run `sqlc generate`.

#### [MODIFY] [urls.sql](file:///home/soy/stuffs/url-shortener/internal/db/queries/urls.sql)
Add queries:
* `UpdateURLStatus`: Toggle URL `is_active` status.

#### [MODIFY] [users.sql](file:///home/soy/stuffs/url-shortener/internal/db/queries/users.sql)
Add queries:
* `UpdateUserRole`: Assign role to user.
* `UpdateUserStatus`: Toggle user active status.

#### [MODIFY] [clicks.sql](file:///home/soy/stuffs/url-shortener/internal/db/queries/clicks.sql)
Add queries for browser and device segmentation:
* `GetDeviceStatsByURLID`: Count clicks grouped by device type.
* `GetBrowserStatsByURLID`: Count clicks grouped by browser.

---

### 2. Backend API Extensions

#### [NEW] Admin Handlers & Service
Create admin logic under `internal/admin/` following the feature-first layout:
* `GET /api/v1/admin/stats`: Get total count of users, active/inactive URLs, total clicks, and top 5 URLs by click volume.
* `GET /api/v1/admin/users`: Paginated list of all users.
* `PUT /api/v1/admin/users/:id/role`: Change user role.
* `PUT /api/v1/admin/users/:id/status`: Toggle user status (active/inactive).

#### [MODIFY] URL Handlers & Service
* `PATCH /api/v1/urls/:id/status`: Handler to toggle URL active status.
* `GET /api/v1/urls/:id/analytics`: Returns daily clicks (past 7 days), device distribution, and browser distribution.

---

### 3. Frontend Web App (Vanilla HTML/CSS/JS)

Create a beautiful, modern user interface served under `web/`.

#### [NEW] [index.html](file:///home/soy/stuffs/url-shortener/web/index.html)
* Core structure for login, user dashboard, URL creation panel, interactive click statistics view, and admin control panel.
* Integrates Chart.js (via CDN) to render beautiful charts.

#### [NEW] [style.css](file:///home/soy/stuffs/url-shortener/web/style.css)
* Premium design: Deep indigo and violet dark background gradients, frosted glass glassmorphism cards (`backdrop-filter: blur()`), smooth hover transitions, custom toggle switches, and responsive grids.

#### [NEW] [app.js](file:///home/soy/stuffs/url-shortener/web/app.js)
* Manages JWT authentication (store tokens, token refresh).
* Handles state transitions between views (Login $\leftrightarrow$ User Dashboard $\leftrightarrow$ Admin Panel).
* Pulls statistics, analytics, and lists URLs/users dynamically, displaying them on charts and interactive tables.

---

## Verification Plan

### Automated & Manual Verification
* **API Compilation**: Compile via `go build`.
* **Flow validation**:
  1. Register an admin user and a standard user.
  2. Create shortened URLs.
  3. Deactivate/Activate URLs and test redirection (confirming 404 on inactive URLs).
  4. Perform multiple clicks using different user agents and verify the generated analytics charts.
  5. Login as admin, promote/demote user roles, and deactivate a user.
