// HTTP fetch bridge — replaces Wails IPC

function token() {
  return localStorage.getItem("authToken") || "";
}

async function api(method, path, body) {
  const opts = { method, headers: { "Content-Type": "application/json" } };
  const t = token();
  if (t) opts.headers["Authorization"] = "Bearer " + t;
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (!res.ok) {
    let msg = res.statusText;
    try { const j = await res.json(); msg = j.message || j.error || msg; } catch {}
    throw new Error(msg);
  }
  return res.json();
}

async function apiBlob(path, filename) {
  const t = token();
  const res = await fetch(path, { headers: t ? { Authorization: "Bearer " + t } : {} });
  if (!res.ok) throw new Error("Error descargando archivo");
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url; a.download = filename; a.click();
  URL.revokeObjectURL(url);
  return { Success: true, Message: "Archivo descargado: " + filename };
}

// ─── Auth / Setup (public) ───────────────────────────────────────────────────

export function GetDBStatus() { return api("GET", "/api/config/db-status"); }
export async function IsSetupMode() { const r = await api("GET", "/api/config/setup-mode"); return r.setupMode === true; }
export function ConfigurarDB(cfg) { return api("POST", "/api/config/configurar-db", cfg); }
export function TestDBConnection(cfg) { return api("POST", "/api/config/test-connection", cfg); }
export function LoginVendedor(creds) { return api("POST", "/api/auth/login", creds); }
export function VerificarLoginMFA(tokenStr, code) { return api("POST", "/api/auth/verify-mfa", { token: tokenStr, code }); }
export function RegistrarVendedor(v) { return api("POST", "/api/auth/register", v); }
export function GenerarMFA(email) { return api("POST", "/api/auth/setup-mfa", { Email: email }); }
export function HabilitarMFA(email, code) { return api("POST", "/api/auth/enable-mfa", { Email: email, Code: code }); }

// Alias used in auth store
export const GenerateJWT = () => Promise.resolve("");
export const AuthMiddleware = () => Promise.resolve(null);

// ─── Productos ───────────────────────────────────────────────────────────────

export function ObtenerProductosPaginado(page, pageSize, q, sortBy, sortOrder) {
  return api("GET", `/api/productos?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}&sortBy=${sortBy||""}&sortOrder=${sortOrder||""}`);
}
export function ObtenerProductoPorUUID(uuid) { return api("GET", `/api/productos/${uuid}`); }
export function RegistrarProducto(p) { return api("POST", "/api/productos", p); }
export function ActualizarProducto(p) { return api("PUT", `/api/productos/${p.UUID}`, p); }
export function EliminarProducto(uuid) { return api("DELETE", `/api/productos/${uuid}`); }
export function ObtenerHistorialStock(uuid) { return api("GET", `/api/productos/${uuid}/historial-stock`); }
export function ActualizarStockMasivo(updates) { return api("PUT", "/api/productos/stock/masivo", updates); }
export function NormalizarStock() { return api("POST", "/api/admin/normalizar-stock"); }
export function NormalizarStockTodosLosProductos() { return api("POST", "/api/admin/normalizar-stock"); }

// ─── Clientes ────────────────────────────────────────────────────────────────

export function ObtenerClientesPaginado(page, pageSize, q, sortBy, sortOrder) {
  return api("GET", `/api/clientes?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}&sortBy=${sortBy||""}&sortOrder=${sortOrder||""}`);
}
export function ObtenerClientePorID(uuid) { return api("GET", `/api/clientes/${uuid}`); }
export function RegistrarCliente(c) { return api("POST", "/api/clientes", c); }
export function ActualizarCliente(c) { return api("PUT", `/api/clientes/${c.UUID}`, c); }
export function EliminarCliente(uuid) { return api("DELETE", `/api/clientes/${uuid}`); }

// ─── Vendedores ──────────────────────────────────────────────────────────────

export function ObtenerVendedoresPaginado(page, pageSize, q, sortBy, sortOrder) {
  return api("GET", `/api/vendedores?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}&sortBy=${sortBy||""}&sortOrder=${sortOrder||""}`);
}
export function ActualizarVendedor(v) { return api("PUT", `/api/vendedores/${v.UUID}`, v); }
export function ActualizarPerfilVendedor(v) { return api("PUT", `/api/vendedores/${v.UUID}/perfil`, v); }
export function EliminarVendedor(uuid) { return api("DELETE", `/api/vendedores/${uuid}`); }

// ─── Proveedores ─────────────────────────────────────────────────────────────

export function ObtenerProveedoresPaginado(page, pageSize, q) {
  return api("GET", `/api/proveedores?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}`);
}
export function ObtenerProveedorPorUUID(uuid) { return api("GET", `/api/proveedores/${uuid}`); }
export function ObtenerProveedoresConEstadisticas(page, pageSize, q) {
  return api("GET", `/api/proveedores/estadisticas?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}`);
}
export function CrearProveedor(p) { return api("POST", "/api/proveedores", p); }
export function ActualizarProveedor(p) { return api("PUT", `/api/proveedores/${p.uuid || p.UUID}`, p); }
export function EliminarProveedor(uuid) { return api("DELETE", `/api/proveedores/${uuid}`); }
export function ObtenerResumenCompras(desde, hasta) {
  return api("GET", `/api/proveedores/resumen-compras?desde=${desde||""}&hasta=${hasta||""}`);
}
export function ObtenerTopProductosDeProveedor(nit, limit) {
  return api("GET", `/api/proveedores/${nit}/top-productos?limit=${limit}`);
}
export function SincronizarProveedoresDesdeFacturas() { return api("POST", "/api/proveedores/sincronizar"); }
export function UpsertProveedorPorNIT(nit, data) { return api("POST", "/api/proveedores", { ...data, NIT: nit }); }

// ─── Facturas venta ──────────────────────────────────────────────────────────

export function ObtenerFacturasPaginado(page, pageSize, q, sortBy, sortOrder) {
  return api("GET", `/api/facturas?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}&sortBy=${sortBy||""}&sortOrder=${sortOrder||""}`);
}
export function ObtenerDetalleFactura(uuid) { return api("GET", `/api/facturas/${uuid}`); }
export function RegistrarVenta(req) { return api("POST", "/api/facturas/venta", req); }
export function RegistrarCompra(req) { return api("POST", "/api/facturas/venta", req); }

// ─── Facturas compra (Db) ────────────────────────────────────────────────────

export function ObtenerFacturasCompraPaginado(page, pageSize, q, sortBy, sortOrder) {
  return api("GET", `/api/facturas-compra?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}&sortBy=${sortBy||""}&sortOrder=${sortOrder||""}`);
}
export function ObtenerDetalleFacturaCompra(uuid) { return api("GET", `/api/facturas-compra/${uuid}`); }
export function ActualizarEstadoFacturaCompra(uuid, estado) {
  return api("PUT", `/api/facturas-compra/${uuid}/estado`, { estado });
}
export function GuardarFacturaCompra(fc) { return api("POST", "/api/facturas-compra", fc); }
export function ExisteFacturaCompra(provNIT, numFact) {
  return api("GET", `/api/facturas-compra/existe?nit=${encodeURIComponent(provNIT)}&numero=${encodeURIComponent(numFact)}`);
}

// ─── Transferencias (Db) ────────────────────────────────────────────────────

export function ObtenerTransferenciasPaginado(page, pageSize, soloNoLeidas, busqueda, estado) {
  return api("GET", `/api/transferencias?page=${page}&pageSize=${pageSize}&soloNoLeidas=${soloNoLeidas||false}&busqueda=${encodeURIComponent(busqueda||"")}&estado=${estado||""}`);
}
export function BuscarFacturasVenta(q, limit) {
  return api("GET", `/api/transferencias/buscar-facturas?q=${encodeURIComponent(q||"")}&limit=${limit||20}`);
}
export function VincularFactura(transferUUID, facturaUUID, facturaNumero) {
  return api("PUT", `/api/bancolombia/transferencias/${transferUUID}/vincular`, { facturaUUID, facturaNumero });
}
export function DesvincularFactura(transferUUID) {
  return api("PUT", `/api/bancolombia/transferencias/${transferUUID}/desvincular`);
}
export function EliminarTransferencia(uuid) { return api("DELETE", `/api/bancolombia/transferencias/${uuid}`); }
export function EliminarTransferencias(uuids) { return api("DELETE", "/api/bancolombia/transferencias", { uuids }); }
export function MarcarTodasLeidas() { return api("PUT", "/api/bancolombia/transferencias/todas-leidas"); }
export function MarcarTransferenciaLeida(uuid) { return api("PUT", `/api/bancolombia/transferencias/${uuid}/leida`); }
export function ContarTransferenciasNoLeidas() { return api("GET", "/api/bancolombia/transferencias/no-leidas"); }
export function GuardarTransferencia(t) { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function ExisteTransferencia(id) { return Promise.reject(new Error("no disponible en modo HTTP")); }

// ─── Dashboard ────────────────────────────────────────────────────────────────

export function ObtenerDatosDashboard(fecha) {
  return api("GET", `/api/dashboard?fecha=${encodeURIComponent(fecha||"")}`);
}
export function ObtenerFechasConVentas() { return api("GET", "/api/dashboard/fechas-ventas"); }
export function ObtenerResumenInventario() { return api("GET", "/api/dashboard/resumen-inventario"); }
export function ObtenerReporteVentasRango(desde, hasta) {
  return api("GET", `/api/dashboard/reporte-ventas?desde=${desde||""}&hasta=${hasta||""}`);
}

// ─── Admin ────────────────────────────────────────────────────────────────────

export function GetTablas() { return api("GET", "/api/admin/tablas"); }
export function GetDatosTabla(name, limit, offset, sortCol, sortDir) {
  return api("GET", `/api/admin/tablas/${name}?limit=${limit}&offset=${offset}&sortCol=${sortCol||""}&sortDir=${sortDir||""}`);
}
export function GetEsquemaTabla(name) { return api("GET", `/api/admin/tablas/${name}/esquema`); }
export function ExportarTablaCSV(name) { return apiBlob(`/api/admin/tablas/${name}/export/csv`, name + ".csv"); }
export function ExportarTablaSQL(name) { return apiBlob(`/api/admin/tablas/${name}/export/sql`, name + ".sql"); }
export function ExportarBDSQL() { return apiBlob("/api/admin/export/bd-sql", "backup.sql"); }
export function ActualizarFilaTabla(tableName, pkColumn, pkValue, updateColumn, newValue, isNull) {
  return api("PUT", `/api/admin/tablas/${tableName}/rows/${pkValue}`, { TableName: tableName, PKColumn: pkColumn, PKValue: pkValue, UpdateColumn: updateColumn, NewValue: newValue, IsNull: isNull });
}
export function EliminarFilaTabla(tableName, pkColumn, pkValue) {
  return api("DELETE", `/api/admin/tablas/${tableName}/rows/${pkValue}`, { TableName: tableName, PKColumn: pkColumn, PKValue: pkValue });
}
export function AnalizarTabla(name) { return api("POST", `/api/admin/tablas/${name}/action`, { action: "analizar" }).then(() => ({ Success: true, Message: "Tabla analizada" })); }
export function VacuumTabla(name) { return api("POST", `/api/admin/tablas/${name}/action`, { action: "vacuum" }).then(() => ({ Success: true, Message: "VACUUM ejecutado" })); }
export function TruncarTabla(name) { return api("POST", `/api/admin/tablas/${name}/action`, { action: "truncar" }).then(() => ({ Success: true, Message: "Tabla truncada" })); }
export function ResetearTodaLaData() { return api("POST", "/api/admin/reset"); }
export function GetSetting(key) { return api("GET", `/api/admin/settings/${key}`); }
export function SetSetting(key, value) { return api("PUT", `/api/admin/settings/${key}`, { value }); }
export function GetDbInstance() { return Promise.resolve(null); }

// ─── Import (triggers browser file picker) ───────────────────────────────────

export function ImportarTablaCSV(tableName) {
  return new Promise((resolve, reject) => {
    const input = document.createElement("input");
    input.type = "file"; input.accept = ".csv";
    input.onchange = async () => {
      const file = input.files && input.files[0];
      if (!file) return resolve({ Success: false, Message: "No se seleccionó archivo" });
      const form = new FormData();
      form.append("file", file);
      const t = token();
      const res = await fetch(`/api/admin/tablas/${tableName}/import/csv`, {
        method: "POST",
        headers: t ? { Authorization: "Bearer " + t } : {},
        body: form,
      });
      if (!res.ok) { const e = await res.json().catch(() => ({})); return reject(new Error(e.message || "Error importando")); }
      resolve(await res.json());
    };
    input.click();
  });
}

export function ImportarSQL() {
  return Promise.resolve({ Success: false, Message: "Importar SQL: usa el endpoint /api/admin/import/sql" });
}

export function SelectFile() {
  return Promise.resolve("");
}

// ─── Impresora / Recibo ──────────────────────────────────────────────────────

export async function ImprimirRecibo(data) {
  const r = await api("POST", "/api/pos/imprimir", data);
  return r;
}
export async function VerificarImpresora() {
  try {
    const r = await api("GET", "/api/pos/verificar");
    return r.available === true;
  } catch {
    return false;
  }
}

// ─── Stub: internal Go methods not for frontend ──────────────────────────────

export function Startup() { return Promise.resolve(); }
export function Close() { return Promise.resolve(); }
export function BeginTx() { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function Exec() { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function Query() { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function QueryRow() { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function CargarDesdeCSV() { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function ImportaCSV() { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function NewPostgresDB() { return Promise.reject(new Error("no disponible en modo HTTP")); }
export function ExistingEmailIDsSet() { return Promise.reject(new Error("no disponible en modo HTTP")); }
