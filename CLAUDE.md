# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

goFarmacia is a desktop pharmacy POS (Point of Sale) application built with **Wails v2** (Go backend + Vue 3 frontend). It manages vendors, clients, products, inventory, invoices, and stock operations. The codebase uses Spanish for domain names, comments, and user-facing strings.

## Commands

### Development
```bash
wails dev              # Run with hot reload (Vite dev server + Go backend)
```

### Build
```bash
wails build            # Create production binary (frontend embedded via go:embed)
```

### Frontend (from frontend/ directory)
```bash
pnpm install           # Install dependencies
pnpm run build         # TypeScript check (vue-tsc) + Vite build
pnpm run dev           # Vite dev server only
```

### Tests
```bash
go test ./backend/tests/ -v          # Run all tests (requires Docker for PostgreSQL)
go test ./backend/tests/ -v -run TestVenta_RegistrarVenta   # Run a single test
```

Tests use `ory/dockertest` to spin up a PostgreSQL 15 container automatically.

## Architecture

### Wails Desktop App Pattern
- `main.go` — Entry point. Creates `App` and `Db` instances, binds them to Wails runtime.
- `app.go` — App lifecycle hooks (`startup`, `shutdown`).
- Frontend calls Go methods directly via Wails auto-generated TypeScript bindings in `frontend/wailsjs/`.
- The `Bind: []any{app, db}` in main.go exposes all public methods on `App` and `Db` structs as callable functions from the frontend.

### Backend (`backend/`)
- **`database.go`** — Singleton `Db` struct via `GetDbInstance()` with `sync.Once`. Holds `*sql.DB`, logger, and JWT secret. Runs PostgreSQL migrations on startup using `golang-migrate`.
- **`auth.go`** — JWT token generation/validation (24h expiry), bcrypt password hashing (cost 14), TOTP-based MFA.
- **`*_logic.go`** files — Domain logic for each entity (vendedor, cliente, producto, proveedor, transaccion, admin, dashboard). Each file contains methods on the `*Db` receiver.
- **`import.go`** — CSV import with worker pool pattern (4 workers, 500-item batches).
- **`db/migrations/postgres/`** — Sequential migration files (currently 6). Auto-applied on startup.

All database operations use **raw SQL with parameterized queries** (`$1`, `$2` PostgreSQL placeholders). Data modifications use `BeginTx()` with explicit commit/rollback. Queries use `Context`-aware methods (`QueryRowContext`, `ExecContext`).

### Frontend (`frontend/`)
- **Vue 3** with `<script setup>` + TypeScript + Composition API
- **Pinia** stores: `auth.ts` (JWT token, user state, localStorage persistence), `cart.ts` (POS shopping cart)
- **Vue Router** with auth guards checking token expiration
- **UI**: shadcn-vue + Reka UI (headless components) + TailwindCSS
- **Data tables**: @tanstack/vue-table
- **Charts**: chart.js + vue-chartjs

### Database
PostgreSQL with UUID primary keys. Key tables: `vendedors`, `clientes`, `productos`, `proveedors`, `facturas`, `detalle_facturas`, `operacion_stocks`. All tables support soft deletes (`deleted_at`) and timestamps (`created_at`, `updated_at`).

## Environment

Requires a `.env` file in the project root:
```
DATABASE_URL=postgresql://user:password@localhost:5432/farmacia_db
JWT_SECRET_KEY=your-secret-key
```

## Conventions

- Git commit prefixes: `[FEAT]`, `[FIX]`, `[TODO]`, `[PENDING]`
- Backend methods are on the `*Db` receiver, making them auto-exposed to the frontend via Wails binding
- Logging uses `logrus` with file output (`logs/app_TIMESTAMP.log`) + stdout
- Stock changes are always tracked via `operacion_stocks` audit trail
