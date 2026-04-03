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

export function EstadoAuth() { return api("GET", "/api/bancolombia/auth"); }
export function IniciarOAuth2() { return api("POST", "/api/bancolombia/auth/iniciar").then(r => r.url); }
export function RevocarAuth() { return api("DELETE", "/api/bancolombia/auth"); }
export function GetSyncProgress() { return api("GET", "/api/bancolombia/sync/progress"); }
export function VerificarAhora() { return api("POST", "/api/bancolombia/sync"); }
export function SincronizarConPeriodo(opts) { return api("POST", "/api/bancolombia/sync/periodo", opts); }
export function ObtenerTransferencias(page, pageSize, soloNoLeidas, busqueda, estado) {
  return api("GET", `/api/bancolombia/transferencias?page=${page}&pageSize=${pageSize}&soloNoLeidas=${soloNoLeidas||false}&busqueda=${encodeURIComponent(busqueda||"")}&estado=${estado||""}`);
}
export function MarcarLeida(uuid) { return api("PUT", `/api/bancolombia/transferencias/${uuid}/leida`); }
export function MarcarTodasLeidas() { return api("PUT", "/api/bancolombia/transferencias/todas-leidas"); }
export function ContarNoLeidas() { return api("GET", "/api/bancolombia/transferencias/no-leidas").then(r => r.count); }
export function EliminarTransferencia(uuid) { return api("DELETE", `/api/bancolombia/transferencias/${uuid}`); }
export function EliminarTransferencias(uuids) { return api("DELETE", "/api/bancolombia/transferencias", { uuids }); }
export function VincularFactura(transferUUID, facturaUUID, facturaNumero) {
  return api("PUT", `/api/bancolombia/transferencias/${transferUUID}/vincular`, { facturaUUID, facturaNumero });
}
export function DesvincularFactura(transferUUID) {
  return api("PUT", `/api/bancolombia/transferencias/${transferUUID}/desvincular`);
}
export function BuscarFacturasVenta(q) {
  return api("GET", `/api/bancolombia/buscar-facturas?q=${encodeURIComponent(q||"")}`);
}
export function GetAutoPolling() { return api("GET", "/api/bancolombia/auto-polling"); }
export function SetAutoPolling(enabled) { return api("PUT", "/api/bancolombia/auto-polling", { enabled }); }
export function Startup() { return Promise.resolve(); }
export function Shutdown() { return Promise.resolve(); }
