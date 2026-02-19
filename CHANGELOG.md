# Changelog

All notable changes to **goFarmacia** are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased] — 2026-02-19

### Features

#### Perfil de usuario — Página dedicada (`/dashboard/perfil`)
- Nueva vista accesible para todos los usuarios autenticados (sin restricción de rol).
- **Tarjeta de identidad:** avatar con iniciales, nombre completo, email y badge de rol (Administrador / Cajero).
- **Información personal:** edición de nombre, apellido, email y cédula con guardado independiente via `ActualizarPerfilVendedor`.
- **Cambio de contraseña:** campos con toggle de visibilidad (Eye/EyeOff) para contraseña actual, nueva y confirmación.
- **Sección 2FA:** componente `MFASetup` embebido inline en la página.
- El ítem "Ajustes de Perfil" del dropdown de `NavUser` ahora navega a esta página en lugar de abrir un dialog inline (menos código duplicado, mejor UX).

#### Gestión de roles desde el panel Vendedores
- **Backend (`vendedor_logic.go`):** `ActualizarVendedor` ahora incluye `role` en el `UPDATE`. Valida que el valor sea `admin` o `cajero` antes de persistir.
- **`DataTableVendedorDropDown.vue`:** Dialog de edición incluye un `<Select>` para cambiar el rol (Cajero / Administrador) de cualquier usuario.
- **`Vendedores.vue`:** Nueva columna **Rol** con badge de color — azul para Administrador, gris para Cajero.

### Bug Fixes

#### Migraciones — Idempotencia completa (000003–000007)
- **Problema:** Los `UPDATE` en migraciones 003 y 004 referenciaban columnas enteras (`factura_id`) eliminadas en migración 006. Al correr sobre un schema UUID-only, fallaban con `column does not exist`, dejando la BD sucia (`version=N, dirty=true`) en cada arranque.
- **Fix (000003, 000004):** Los `UPDATE` envueltos en bloques `DO $$ BEGIN IF EXISTS(column) THEN ... END IF; END $$` para saltar la sentencia si la columna ya no existe.
- **Fix (000006):** Todos los 8 `UPDATE` de migración de datos envueltos en guards de existencia de columna; todos los `DROP COLUMN` cambiados a `DROP COLUMN IF EXISTS`; `ADD PRIMARY KEY` y `ADD CONSTRAINT FK` protegidos con `DO $$` con checks de `pg_constraint`.
- **Fix (000007):** `ADD COLUMN role` cambiado a `ADD COLUMN IF NOT EXISTS role`.
- **BD corregida directamente:** `schema_migrations` actualizado a `version=7, dirty=false` para reflejar el estado real del schema restaurado.

#### POS / ERP — Pestañas no visibles para rol `cajero`
- El contenedor completo de tabs tenía `v-if="modeStore.isAdmin"`, ocultando incluso la pestaña POS para usuarios cajero.
- **Fix (`AppSidebar.vue`):** `v-if` movido solo al `<TabsTrigger value="erp">`. El contenedor usa `:class` dinámico para grid-cols-1 o grid-cols-2 según el rol. Cajeros ven POS, admins ven POS + ERP.

#### Layout — App no ocupaba pantalla completa
- `html`, `body` y `#app` carecían de `height: 100%`, impidiendo que el `SidebarProvider` de shadcn-vue se expandiera al tamaño completo de la ventana Wails.
- **Fix (`index.html`):** `class="h-full"` en `<html>` y `<body>`; `style="height:100%;display:flex;flex-direction:column"` en `#app`.
- **Fix (`style.css`):** Eliminada regla duplicada `body { @apply bg-gray-100 }` que conflictuaba con `@layer base`.
- **Fix (`DashboardLayout.vue`):** Eliminados exports muertos `iframeHeight` y `description` del template shadcn; añadidos `h-full` y `min-h-0` al `SidebarProvider` y `SidebarInset`.

#### Log — Triple archivo de log en cada arranque
- `wails dev` arranca el binario Go 3 veces (introspección de bindings, hot-reload probe, ejecución real). `sync.Once` se ejecutaba en cada proceso → 3 archivos de log por sesión.
- **Fix (`database.go`):** La creación del archivo de log se movió de `GetDbInstance()` (ejecutado en cada proceso via `once.Do`) a `initDB()`, que solo corre durante el arranque real de la aplicación.

#### Migraciones — Auto-recuperación de estado sucio
- `runMigrations` hacía `log.Fatal` al encontrar `ErrDirtyDatabase`, bloqueando la app permanentemente.
- **Fix (`database.go`):** Detección de `migrate.ErrDirty` con `errors.As`, llamada a `m.Force(version)` para limpiar el estado, y reintento de `m.Up()` automáticamente.

### Changed

#### BuscarClienteModal (POS) — Comportamiento de carga
- **Antes:** Lista vacía al abrir; requería escribir para ver cualquier resultado.
- **Ahora:** Carga los primeros 50 clientes al abrir el modal; la barra actúa como filtro con debounce 300ms. Al borrar el texto, recarga la lista completa. Spinner en la barra de búsqueda (no bloquea la lista). Input con `autofocus`.

---

## [v0.2.0] — 2026-02-18

### Bug Fixes

#### Critical — Cálculo de stock incorrecto tras ventas
- **Causa:** `CrearOperacionStock` guardaba `cantidad_cambio` como entero **positivo** para operaciones `VENTA`. `calcularStockRealLocal` usa `SUM(cantidad_cambio)` como fuente de verdad, por lo que cada venta sumaba stock en lugar de restar.
- **Fix (`backend/transaccion_logic.go`):** Las operaciones de reducción (`VENTA`, `AJUSTE_NEGATIVO`, `DEVOLUCION_CLIENTE`) ahora persisten `dbCambio = -cambio`.

#### Critical — Fallo al crear entidades (Producto, Cliente, Proveedor)
- **Causa:** `operacion_stocks.vendedor_uuid` es columna `uuid` con FK. El código pasaba strings literales `""` y `"SYSTEM-ADMIN"` — UUIDs inválidos — causando `pq: invalid input syntax for type uuid`.
- **Fix:** `CrearOperacionStock`, `RegistrarProducto`, `ActualizarStockMasivo` y `RegistrarCompra` pasan `nil` (SQL `NULL`) en lugar de strings inválidos.

#### POS — `clienteUUID` inicializado como UUID inválido
- **Fix (`POS.vue`):** Valor inicial y fallbacks cambiados a `""`. `finalizarVenta` valida y muestra toast descriptivo si no hay cliente asignado.

#### POS — Búsqueda bloqueada para queries de un carácter
- Umbral `length < 2` impedía encontrar productos con código `"1"` o nombre `"1"`.
- **Fix:** Umbral cambiado a `length < 1` en POS y a `!nuevoValor.trim()` en BuscarClienteModal.

#### POS — Dropdown de resultados recortado por Radix TabsContent
- `<TabsContent>` aplica `overflow: hidden` internamente, cortando el dropdown absoluto.
- **Fix (`POS.vue`):** `<TabsContent>` reemplazados por divs con `v-show`. Z-index del dropdown subido a `z-[200]`. Eliminado `backdrop-blur-sm` del header sticky.

### Features

#### Dashboard — Rediseño completo UI
- **Home:** KPI cards (ventas del día, stock bajo, top vendedor), gráfico de ingresos 7 días, tabla de ventas recientes, panel de alertas de stock bajo.
- **Sidebar:** Navegación colapsable con modo icono, visibilidad por rol, resaltado de ruta activa.
- **Todas las vistas de gestión:** Patrón DataTable consistente — paginación server-side, búsqueda con debounce, columnas ordenables, acciones inline, modales de creación/edición/eliminación.

#### POS — Rediseño completo
- Header compacto con tabs, selector de cliente y método de pago.
- Búsqueda con dropdown `z-[200]`, navegación teclado (↑/↓/Enter) y shortcut "Crear Producto".
- Tabla de carrito con edición inline de cantidad y precio.
- Footer con total en vivo, cambio de efectivo, atajos F11/F12.

#### Login & Register — Rediseño split-panel
- Panel izquierdo (40%): gradiente azul oscuro, icono `Building2`, nombre de la farmacia, lista de características.
- Panel derecho (60%): formulario limpio, toggle de contraseña, paso MFA con transición animada, `Alert` destructivo para errores.

#### Herramientas de migración de BD (`backend/python/`)
- **`preprocess.py`:** Convierte exportaciones SQL de Supabase a INSERTs compatibles con el schema local.
- **`final_schema.sql`:** Schema único y autoritativo tras las 7 migraciones. Siembra `schema_migrations` para que golang-migrate reconozca todas las versiones como aplicadas.
- **`reset_and_import.sh`:** Restauración completa: drop schema → recrear → preprocesar → importar en orden FK → recalcular stock.

#### Backend — Endpoint de analytics del dashboard
- `ObtenerResumenDashboard`: ventas del día, productos con stock bajo, top vendedor del mes, ingresos últimos 7 días.

### Changed
- Umbral de búsqueda mínimo reducido de 2 a 1 carácter en POS y BuscarClienteModal.
- `vendedor_logic.go`: limpieza de referencias ORM obsoletas.
- `database.go`: registro de `ObtenerResumenDashboard` como método Wails.

---

## [v0.1.0] — 2026-01-01

### Features
- Backend en Go con raw SQL (PostgreSQL via `lib/pq`).
- Autenticación JWT + MFA (TOTP) con `golang-migrate` para migraciones.
- Frontend Vue 3 + TypeScript + shadcn-vue + TailwindCSS v4.
- Módulos: Ventas POS, Facturación, Inventario, Clientes, Vendedores, Proveedores, Reportes.
- Empaquetado como app de escritorio con Wails v2.
