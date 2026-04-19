// HTTP fetch bridge — Outlook (Microsoft Graph) endpoints

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

export function EstadoAuth() { return api("GET", "/api/outlook/auth"); }
export function IniciarOAuth2() { return api("POST", "/api/outlook/auth/iniciar").then(r => r.url); }
export function RevocarAuth() { return api("DELETE", "/api/outlook/auth"); }
export function ObtenerCredenciales() { return api("GET", "/api/outlook/credenciales"); }
export function GuardarCredenciales(jsonContent) {
  const t = token();
  return fetch("/api/outlook/credenciales", {
    method: "PUT",
    headers: { "Content-Type": "application/json", ...(t ? { Authorization: "Bearer " + t } : {}) },
    body: jsonContent,
  }).then(r => { if (!r.ok) throw new Error("Error guardando credenciales"); return r.json(); });
}
export function GetOutlookSyncProgress() { return api("GET", "/api/outlook/sync/progress"); }
export function SincronizarConOpciones(opts) { return api("POST", "/api/outlook/sync/opciones", opts); }
export function GetAutoSync() { return api("GET", "/api/outlook/auto-sync"); }
export function SetAutoSync(enabled) { return api("PUT", "/api/outlook/auto-sync", { enabled }); }
