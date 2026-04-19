// HTTP fetch bridge — replaces Wails IPC

function token() { return localStorage.getItem("authToken") || ""; }

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

export function EstadoAuth() { return api("GET", "/api/gmail/auth"); }
export function IniciarOAuth2() { return api("POST", "/api/gmail/auth/iniciar").then(r => r.url); }
export function RevocarAuth() { return api("DELETE", "/api/gmail/auth"); }
export function ConfigDir() { return api("GET", "/api/gmail/config-dir").then(r => r.configDir); }
export function ObtenerCredenciales() { return api("GET", "/api/gmail/credenciales"); }
export function GuardarCredenciales(jsonContent) {
  const t = token();
  return fetch("/api/gmail/credenciales", {
    method: "PUT",
    headers: { "Content-Type": "application/json", ...(t ? { Authorization: "Bearer " + t } : {}) },
    body: jsonContent,
  }).then(r => { if (!r.ok) throw new Error("Error guardando credenciales"); return r.json(); });
}
export function GetGmailSyncProgress() { return api("GET", "/api/gmail/sync/progress"); }
export function SincronizarFacturas() { return api("POST", "/api/gmail/sync"); }
export function SincronizarConOpciones(opts) { return api("POST", "/api/gmail/sync/opciones", opts); }
export function ObtenerFacturasCompra(page, pageSize, q, sortBy, sortOrder) {
  return api("GET", `/api/gmail/facturas-compra?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}&sortBy=${sortBy||""}&sortOrder=${sortOrder||""}`);
}
export function ObtenerDetalleFacturaCompra(uuid) { return api("GET", `/api/gmail/facturas-compra/${uuid}`); }
export function ActualizarEstadoFacturaCompra(uuid, estado) {
  return api("PUT", `/api/gmail/facturas-compra/${uuid}/estado`, { estado });
}
export function ObtenerProveedoresConEstadisticas(page, pageSize, q) {
  return api("GET", `/api/gmail/proveedores/estadisticas?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(q||"")}`);
}
export function ObtenerTopProductosDeProveedor(nit, limit) {
  return api("GET", `/api/gmail/proveedores/${nit}/top-productos?limit=${limit}`);
}
export function ObtenerResumenCompras(desde, hasta) {
  return api("GET", `/api/gmail/proveedores/resumen-compras?desde=${desde||""}&hasta=${hasta||""}`);
}
export function SincronizarProveedoresDesdeFacturas() { return api("POST", "/api/gmail/proveedores/sincronizar"); }
export function GetAutoSync() { return api("GET", "/api/gmail/auto-sync"); }
export function SetAutoSync(enabled) { return api("PUT", "/api/gmail/auto-sync", { enabled }); }
export function EnriquecerDescripciones() { return Promise.resolve({ Success: false, Message: "No implementado" }); }
export function Startup() { return Promise.resolve(); }
export function BuscarFacturasVenta(q) {
  return api("GET", `/api/transferencias/buscar-facturas?q=${encodeURIComponent(q||"")}`);
}
