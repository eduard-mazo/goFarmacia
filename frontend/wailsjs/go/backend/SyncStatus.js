// HTTP fetch bridge — unified sync status dashboard

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

export function GetSyncStatus() { return api("GET", "/api/sync/status"); }
export function SetSyncEnabled(id, enabled) { return api("PUT", `/api/sync/${id}/enabled`, { enabled }); }
