// SSE-based event bridge — replaces Wails runtime

// ─── SSE singleton ────────────────────────────────────────────────────────────

let _es = null;
const _listeners = {}; // eventName → Set<{cb, remaining}>

function getEventSource() {
  if (typeof EventSource === "undefined") return null; // non-browser (SSR/tests)
  if (_es && _es.readyState !== EventSource.CLOSED) return _es;
  const t = localStorage.getItem("authToken");
  if (!t) return null; // not authenticated — connect after login via resetEventSource()
  _es = new EventSource("/api/events?token=" + encodeURIComponent(t));
  _es.onmessage = (e) => {
    try {
      const { event, data } = JSON.parse(e.data);
      const handlers = _listeners[event];
      if (!handlers) return;
      const dead = [];
      handlers.forEach((h) => {
        h.cb(data);
        if (h.remaining > 0) { h.remaining--; if (h.remaining === 0) dead.push(h); }
      });
      dead.forEach((h) => handlers.delete(h));
    } catch {}
  };
  _es.onerror = () => {
    // Auto-reconnect: EventSource handles it natively
  };
  return _es;
}

// ─── Public API ───────────────────────────────────────────────────────────────

// resetEventSource closes any existing stream and reconnects using the current
// auth token. Call it after login (to start the authenticated stream) and after
// logout (drops the stream — no token means no reconnect). Registered listeners
// survive because they live in the module-level _listeners map.
export function resetEventSource() {
  if (_es) {
    try { _es.close(); } catch {}
    _es = null;
  }
  getEventSource();
}

export function EventsOnMultiple(eventName, callback, maxCallbacks) {
  getEventSource();
  if (!_listeners[eventName]) _listeners[eventName] = new Set();
  const h = { cb: callback, remaining: maxCallbacks < 0 ? -1 : maxCallbacks };
  _listeners[eventName].add(h);
  return () => _listeners[eventName]?.delete(h);
}

export function EventsOn(eventName, callback) {
  return EventsOnMultiple(eventName, callback, -1);
}

export function EventsOff(eventName, ...additionalEventNames) {
  [eventName, ...additionalEventNames].forEach((n) => { delete _listeners[n]; });
}

export function EventsOffAll() {
  Object.keys(_listeners).forEach((k) => delete _listeners[k]);
}

export function EventsOnce(eventName, callback) {
  return EventsOnMultiple(eventName, callback, 1);
}

export function EventsEmit(eventName, ...args) {
  // No-op in HTTP mode — events flow server→client only via SSE
}

// ─── Browser / Window stubs ──────────────────────────────────────────────────

export function BrowserOpenURL(url) { window.open(url, "_blank"); }
export function WindowReload() { window.location.reload(); }
export function WindowReloadApp() { window.location.reload(); }
export function WindowSetTitle(title) { document.title = title; }
export function Quit() {}
export function Hide() {}
export function Show() {}

// Stubs for window/screen operations (no-op in browser)
export function WindowSetAlwaysOnTop() {}
export function WindowSetSystemDefaultTheme() {}
export function WindowSetLightTheme() {}
export function WindowSetDarkTheme() {}
export function WindowCenter() {}
export function WindowFullscreen() {}
export function WindowUnfullscreen() {}
export function WindowIsFullscreen() { return false; }
export function WindowGetSize() { return { w: window.innerWidth, h: window.innerHeight }; }
export function WindowSetSize() {}
export function WindowSetMaxSize() {}
export function WindowSetMinSize() {}
export function WindowSetPosition() {}
export function WindowGetPosition() { return { x: 0, y: 0 }; }
export function WindowHide() {}
export function WindowShow() {}
export function WindowMaximise() {}
export function WindowToggleMaximise() {}
export function WindowUnmaximise() {}
export function WindowIsMaximised() { return false; }
export function WindowMinimise() {}
export function WindowUnminimise() {}
export function WindowIsMinimised() { return false; }
export function WindowIsNormal() { return true; }
export function WindowSetBackgroundColour() {}
export function ScreenGetAll() { return []; }
export function Environment() { return { buildType: "production", platform: "web", arch: "" }; }

// Log stubs
export function LogPrint(msg) { console.log(msg); }
export function LogTrace(msg) { console.trace(msg); }
export function LogDebug(msg) { console.debug(msg); }
export function LogInfo(msg) { console.info(msg); }
export function LogWarning(msg) { console.warn(msg); }
export function LogError(msg) { console.error(msg); }
export function LogFatal(msg) { console.error("FATAL:", msg); }

// Clipboard
export function ClipboardGetText() { return navigator.clipboard?.readText() ?? Promise.resolve(""); }
export function ClipboardSetText(text) { navigator.clipboard?.writeText(text); }

// File drop (no-op in browser)
export function OnFileDrop() {}
export function OnFileDropOff() {}
export function CanResolveFilePaths() { return false; }
export function ResolveFilePaths(files) { return files; }
