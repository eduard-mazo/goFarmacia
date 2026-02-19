# goFarmacia

Sistema de gestión farmacéutica de escritorio — **POS + ERP** para droguerías y farmacias.

Construido con **Wails v2** (Go + Vue 3), empaquetado como aplicación nativa de escritorio para Linux, Windows y macOS.

---

## Características

### Punto de Venta (POS)
- Búsqueda de productos por nombre o código de barras con dropdown de resultados
- Carrito de compra con edición inline de cantidad y precio unitario
- Múltiples carritos en espera (F11 para guardar, F12 para finalizar)
- Selección de cliente y método de pago
- Cálculo de cambio en efectivo en tiempo real
- Generación de factura numerada con IVA configurable

### ERP — Gestión administrativa (rol Admin)
- **Productos:** catálogo con stock, precio de venta y código único
- **Clientes:** base de datos con tipo/número de ID, teléfono, dirección
- **Vendedores:** gestión de usuarios con roles (Administrador / Cajero)
- **Proveedores:** directorio con contacto
- **Inventario:** control de stock con historial de operaciones (VENTA, COMPRA, AJUSTE)
- **Reportes:** ventas e inventario con gráficos
- **Configuración:** nombre de tienda, IVA, umbral de stock bajo

### Seguridad
- Autenticación JWT con expiración
- Autenticación de dos factores (TOTP — Google Authenticator compatible)
- Protección de rutas por rol (`admin` / `cajero`)
- Cambio de contraseña con verificación de contraseña actual

### Dashboard
- KPIs del día: ventas, ingresos, stock bajo, top vendedor
- Gráfico de ingresos de los últimos 7 días
- Tabla de ventas recientes
- Alertas de productos con stock crítico

---

## Stack tecnológico

| Capa | Tecnología |
|------|-----------|
| Desktop runtime | Wails v2 |
| Backend | Go 1.21+, `lib/pq`, `golang-migrate`, `golang-jwt` |
| Base de datos | PostgreSQL 15+ |
| Frontend | Vue 3 + TypeScript |
| UI Components | shadcn-vue + Reka UI |
| Estilos | TailwindCSS v4 (`@tailwindcss/vite`) |
| Animaciones | tw-animate-css |
| Iconos | Lucide Vue Next |
| Estado | Pinia |
| Router | Vue Router 4 |
| Gráficos | Chart.js + vue-chartjs |
| Toasts | vue-sonner |
| Package manager | pnpm |

---

## Requisitos previos

- **Go** 1.21 o superior
- **Node.js** 18+ y **pnpm**
- **Wails CLI** v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **PostgreSQL** 15+
- **webkit2gtk-4.1** (Linux) — *esta máquina tiene 4.1, NO 4.0*

---

## Configuración

### 1. Base de datos

```sql
-- Crear usuario y base de datos
CREATE USER luna WITH PASSWORD 'tu_password_seguro';
CREATE DATABASE farmacia_db OWNER luna;
GRANT ALL PRIVILEGES ON DATABASE farmacia_db TO luna;
```

### 2. Variables de entorno

Crear `.env` en la raíz del proyecto:

```env
DATABASE_URL=postgresql://luna:tu_password_seguro@localhost:5432/farmacia_db?sslmode=disable
JWT_SECRET=tu_clave_secreta_jwt
```

### 3. Migraciones

Las migraciones se ejecutan automáticamente al arrancar la app. Para restaurar desde backup de Supabase:

```bash
cd backend/python
bash reset_and_import.sh
```

---

## Desarrollo

```bash
# Linux (webkit2gtk-4.1)
wails dev -tags webkit2_41

# Windows / macOS
wails dev
```

El servidor de desarrollo Vite corre en `http://localhost:34115` para inspección en navegador.

---

## Build

```bash
# Linux
wails build -tags webkit2_41

# Windows / macOS
wails build
```

El binario se genera en `build/bin/`.

---

## Roles de usuario

| Rol | Acceso |
|-----|--------|
| `admin` | Dashboard completo — POS + ERP, reportes, configuración, gestión de usuarios |
| `cajero` | Solo POS — ventas, facturas y perfil personal |

Los roles se gestionan desde **ERP → Vendedores → Editar** (requiere rol `admin`).

---

## Estructura del proyecto

```
goFarmacia/
├── backend/                  # Lógica Go (bound a Wails)
│   ├── auth.go               # Login, registro, JWT, MFA
│   ├── database.go           # Conexión PostgreSQL, migraciones
│   ├── vendedor_logic.go     # CRUD vendedores + roles
│   ├── producto_logic.go     # CRUD productos
│   ├── cliente_logic.go      # CRUD clientes
│   ├── transaccion_logic.go  # Ventas, compras, stock
│   ├── dashboard_logic.go    # KPIs y analytics
│   └── db/migrations/        # Archivos golang-migrate
│       └── postgres/
│           ├── 000001_initial_schema.up.sql
│           └── ...
├── frontend/
│   └── src/
│       ├── views/Dashboard/  # Vistas por módulo
│       │   ├── Home.vue
│       │   ├── Facturacion/POS.vue
│       │   ├── Personas/Vendedores.vue
│       │   ├── Perfil/MiPerfil.vue
│       │   └── ...
│       ├── components/
│       │   ├── auth/         # Login, Register, MFASetup
│       │   ├── layout/       # AppSidebar, NavUser, NavMain
│       │   ├── modals/       # BuscarClienteModal, etc.
│       │   └── ui/           # shadcn-vue components
│       ├── stores/           # Pinia: auth, mode
│       └── router/index.ts   # Rutas con guards de rol
├── backend/python/           # Herramientas de migración de BD
│   ├── final_schema.sql      # Schema completo post-migraciones
│   ├── reset_and_import.sh   # Script de restauración completa
│   └── preprocess.py         # Procesador de exports Supabase
├── main.go                   # Entrada Wails, configuración de ventana
├── app.go                    # App struct, startup/shutdown
├── wails.json                # Configuración del proyecto Wails
└── CHANGELOG.md              # Historial de cambios
```

---

## Usuarios por defecto

Tras restaurar desde backup o en instalación fresca, los usuarios deben crearse desde la pantalla de registro. El primer usuario `admin` puede asignarse directamente en BD:

```sql
UPDATE vendedors SET role = 'admin' WHERE email = 'tu@email.com';
```

O desde la app: **ERP → Vendedores → (menú del usuario) → Editar → Rol: Administrador**.

---

## Notas de build en Linux

Esta máquina tiene `webkit2gtk-4.1` instalado (**no** `webkit2gtk-4.0`). Todos los comandos de build y desarrollo **deben** incluir el tag:

```bash
-tags webkit2_41
```

Sin este tag, la compilación falla con errores de CGO relacionados con webkit.
