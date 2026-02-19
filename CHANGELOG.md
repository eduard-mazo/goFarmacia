# Changelog

All notable changes to **goFarmacia** are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased] — 2026-02-18

### Bug Fixes

#### Critical — Stock calculation incorrect after sales
- **Root cause:** `CrearOperacionStock` was storing `cantidad_cambio` as a **positive** integer for `VENTA` operations. Because `calcularStockRealLocal` derives the current stock via `SUM(cantidad_cambio)`, every sale was *adding* to the stock total instead of subtracting.
- **Fix (`backend/transaccion_logic.go`):** Reduction operations (`VENTA`, `AJUSTE_NEGATIVO`, `DEVOLUCION_CLIENTE`) now persist `cantidad_cambio` as a **negative** value (`dbCambio = -cambio`), aligning with the historical Supabase export format and making the `SUM` function the single, correct source of truth.

#### Critical — Entity creation failures (Producto, Cliente, Proveedor)
- **Root cause:** `operacion_stocks.vendedor_uuid` is a PostgreSQL `uuid` column with a FK constraint. The code was passing literal strings `""` and `"SYSTEM-ADMIN"` — both invalid UUIDs — causing `pq: invalid input syntax for type uuid` on every product registration and bulk stock update.
- **Fix (`backend/transaccion_logic.go`):** `CrearOperacionStock` now converts an empty `vendedorUUID` to `nil` (SQL `NULL`) via a typed `interface{}` variable before executing the insert.
- **Fix (`backend/producto_logic.go`):** `RegistrarProducto` passes `nuevo.VendedorUUID` (the actual caller UUID) instead of a hardcoded empty string. `ActualizarStockMasivo` passes `nil` instead of `"SYSTEM-ADMIN"`.
- **Fix (`backend/transaccion_logic.go`):** `RegistrarCompra` passes `nil` for `vendedor_uuid` in the direct stock operation insert, removing the same `"SYSTEM-ADMIN"` placeholder.

#### POS — `clienteUUID` fallback was an invalid UUID
- The POS store initialized `clienteUUID` to `"SYSTEM-ADMIN"`. If `cargarClienteGeneralPorDefecto` failed to find the default client, this value was sent to `RegistrarVenta`, causing a FK violation on `facturas.cliente_uuid`.
- **Fix (`frontend/src/views/Dashboard/Facturacion/POS.vue`):** Initial value and all fallbacks changed to `""`. `finalizarVenta` now guards against an empty `clienteUUID` and shows a descriptive toast error instead of crashing.

#### POS — Product search blocked for single-character queries
- Both the POS product search and the client search modal had a `length < 2` guard, making it impossible to find products or clients whose name or code is a single character (e.g., code `"1"`, name `"1"`).
- **Fix (`POS.vue`):** Threshold lowered to `length < 1` (fires on any non-empty input). Template `v-else-if` condition updated to match.
- **Fix (`BuscarClienteModal.vue`):** Guard changed to `!nuevoValor.trim()`.

#### POS — Silent auto-add when product name equals product code
- The search watcher contained logic that silently added a product to the cart (without showing the dropdown) when exactly one result was returned and `Codigo === typedValue`. If a product's name and code were identical, typing the full name triggered an invisible auto-add with no user feedback.
- **Fix (`POS.vue`):** Removed all auto-add logic from the watcher. The watcher now only populates `productosEncontrados` and always shows the dropdown. Auto-add on a single unambiguous match is delegated exclusively to `manejarBusquedaConEnter` (Enter key / barcode scanner flow).

#### POS — Search dropdown clipped by Radix `TabsContent` overflow
- Shadcn-vue's `<TabsContent>` applies `overflow: hidden` internally, clipping the absolutely-positioned search results dropdown so it never appeared below the input.
- **Fix (`POS.vue`):** Replaced `<TabsContent>` wrappers with plain `v-show` divs that carry no overflow constraints. Dropdown z-index raised to `z-[200]`. Removed `backdrop-blur-sm` from the sticky table header (it was creating an independent stacking context that also clipped the dropdown).

---

### Features

#### Dashboard — Full UI redesign
- Replaced the generic scaffolding across all dashboard views with a cohesive design system built on **shadcn-vue + TailwindCSS v4**.
- **Home (`Home.vue`):** New analytics dashboard with KPI cards (ventas del día, stock bajo, top vendedor), revenue chart (`PaymentMethodsChart`), recent sales table, and low-stock alert panel. Powered by new `ObtenerResumenDashboard` backend endpoint (`dashboard_logic.go`).
- **Sidebar (`AppSidebar.vue`):** Collapsible navigation with icon-only and expanded modes, role-based menu visibility, and active route highlighting.
- **Layout (`DashboardLayout.vue`):** Sticky top bar with breadcrumbs, user avatar, logout, and responsive sidebar toggle.
- **Productos, Clientes, Vendedores, Facturas, ControlStock:** All views rebuilt with consistent `DataTable` patterns: server-side pagination, debounced search, sortable columns, inline action menus, and modal-based create/edit/delete flows.

#### POS — Complete UI overhaul
- Full redesign of the Point-of-Sale view for usability and visual consistency:
  - **Header bar:** Tab switcher (Venta Actual / Carritos en Espera), client selector, and payment method selector all in a single compact row.
  - **Search:** Large, prominent search input with `z-[200]` dropdown overlay, keyboard navigation (↑/↓/Enter), and auto-scroll to highlighted item. Shows a "Crear Producto" shortcut when no results are found.
  - **Cart table:** Scrollable inner container with a sticky header (no `backdrop-blur` to avoid stacking context), inline quantity and price editing, and subtotal per line.
  - **Footer bar:** Live total, cash change display, "En Espera (F11)" and "Finalizar Venta (F12)" action buttons with keyboard shortcut badges.
  - **Saved carts panel:** Card list for carts on hold with load and delete actions; empty-state illustration.

#### Login & Register — Split-panel redesign
- Both auth views replaced with a full-screen split layout:
  - **Left panel (40%):** Dark blue-slate gradient, `Building2` brand icon, pharmacy name, and feature list with `CheckCircle2` bullets.
  - **Right panel (60%):** Clean form with password show/hide toggle (`Eye`/`EyeOff`), MFA step with animated transition and `ShieldCheck` icon, and inline `Alert` for error display.
  - Register mirrors the same brand panel with a 2-column grid (Nombre/Apellido, Email/Cédula) and `Loader2` spinner on submit.

#### Database migration tooling (`backend/python/`)
- **`preprocess.py`:** Transforms Supabase SQL row exports into schema-compatible INSERT statements. Strips columns absent from the local schema (`reset_password_token`, `reset_password_expires`), adds `ON CONFLICT (uuid) DO NOTHING` idempotency guards, and handles quoted identifiers, `null` literals, and escaped single-quote strings.
- **`final_schema.sql`:** Authoritative single-file schema representing the final state after all 7 migrations. Bypasses the incremental migration chain (which fails on a fresh database because intermediate migrations assume pre-existing integer-PK columns). Also seeds `schema_migrations` so golang-migrate considers all versions applied.
- **`reset_and_import.sh`:** One-command full restore: drops the public schema, recreates it from `final_schema.sql`, preprocesses all SQL backups, imports in FK-dependency order (vendedors → clientes → productos → facturas → operacion\_stocks → detalle\_facturas), and recalculates product stock from `operacion_stocks` as the source of truth.
- **`migrate.py`:** Legacy CSV-to-SQL converter for older integer-PK exports (kept for reference).

#### Backend — Dashboard analytics endpoint
- New `ObtenerResumenDashboard` function in `dashboard_logic.go` returns:
  - Today's sales count and revenue
  - Products with low stock (≤ 10 units)
  - Top seller by revenue this month
  - Last 7 days of daily revenue for the chart

---

### Changed

- **Search minimum threshold:** Both product search (POS) and client search modal now trigger on **1+ character** instead of 2, enabling single-character codes and names.
- **`vendedor_logic.go`:** Cleaned up unused ORM references; aligned pagination and search with raw SQL patterns used throughout the rest of the backend.
- **`auth.go`:** Minor session handling cleanup.
- **`database.go`:** Registered `ObtenerResumenDashboard` as a Wails-bound method.
- **`tsconfig.app.json` / `vite.config.ts`:** Path alias and build config adjusted for Wails frontend embedding.

---

### Internal / Tooling

- **Build tag:** All builds and dev runs require `-tags webkit2_41` on this machine (`webkit2gtk-4.1` installed, `webkit2gtk-4.0` absent).
- **`golang-migrate` CLI** installed at `~/go/bin/migrate` for manual migration management.
