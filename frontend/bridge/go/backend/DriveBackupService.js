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

export function EstadoAuthDrive() { return api("GET", "/api/drive/auth"); }
export function IniciarOAuth2Drive() { return api("POST", "/api/drive/auth/iniciar").then(r => r.url); }
export function RevocarAuthDrive() { return api("DELETE", "/api/drive/auth"); }
export function GetAutoBackup() { return api("GET", "/api/drive/auto-backup"); }
export function SetAutoBackup(enabled) { return api("PUT", "/api/drive/auto-backup", { enabled }); }
export function EjecutarBackupAhora() { return api("POST", "/api/drive/backup"); }
export function ListarBackups() { return api("GET", "/api/drive/backups"); }
export function EliminarBackup(fileID) { return api("DELETE", `/api/drive/backups/${fileID}`); }
export function RestaurarBackup(fileID) { return api("POST", `/api/drive/backups/${fileID}/restore`); }
export function Startup() { return Promise.resolve(); }
export function Shutdown() { return Promise.resolve(); }
