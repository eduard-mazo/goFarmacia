package backend

// bancolombia_service.go — Sincronización de Gmail para notificaciones de transferencias Bancolombia.
//
// ═══════════════════════════════════════════════════════════════════════════════
// DESCRIPCIÓN GENERAL
// ═══════════════════════════════════════════════════════════════════════════════
//
// Este servicio detecta y almacena automáticamente las transferencias bancarias
// que Bancolombia notifica por correo electrónico. Es crítico para el flujo de
// caja de la droguería: el personal valida el pago recibido antes de entregar
// el pedido.
//
// ═══════════════════════════════════════════════════════════════════════════════
// ESTRATEGIA DE POLLING
// ═══════════════════════════════════════════════════════════════════════════════
//
//   • Ticker en segundo plano (cada 2 minutos, ver bancolombiaTickerPeriod):
//     revisa los últimos 30 correos del remitente Bancolombia. Mantiene el feed
//     actualizado sin intervención manual durante la jornada laboral.
//
//   • Sincronización manual ("Sincronizar" en la UI):
//     permite elegir el período (hoy / semana / mes / rango / completo) y
//     recuperar mensajes anteriores. Soporta paginación completa mediante
//     NextPageToken para no perder ningún mensaje aunque Gmail los agrupe en
//     hilos.
//
// ═══════════════════════════════════════════════════════════════════════════════
// AUTENTICACIÓN OAUTH2
// ═══════════════════════════════════════════════════════════════════════════════
//
//   • Reutiliza el mismo credentials.json que el servicio DIAN/Gmail (app de
//     Google Cloud configurada por el administrador).
//   • Guarda el token en un archivo separado (bancolombia_token.json), lo que
//     permite autenticar una cuenta Gmail diferente a la de facturas DIAN.
//   • El flujo de autorización abre un servidor HTTP local en el puerto 8095
//     y espera el callback de Google. La URL de autorización se devuelve al
//     frontend, que es responsable de abrir el navegador (BrowserOpenURL).
//   • Ruta de redirección: http://localhost:8095/bancolombia/oauth2/callback
//
// ═══════════════════════════════════════════════════════════════════════════════
// CONSULTA GMAIL
// ═══════════════════════════════════════════════════════════════════════════════
//
//   Filtro base: from:notificacionesbancolombia.com
//   Cubre ambos dominios de Bancolombia:
//     • alertasynotificaciones@notificacionesbancolombia.com
//     • alertasynotificaciones@an.notificacionesbancolombia.com
//
//   ZONA HORARIA: Los filtros de fecha usan timestamps Unix (not strings
//   YYYY/MM/DD). Esto es CRÍTICO. Las cadenas "after:2026/03/02" son
//   interpretadas por Gmail según la zona horaria de la CUENTA Gmail
//   (que puede ser UTC), no la del servidor. Al usar Unix timestamps se
//   garantiza que "hoy" corresponde al día correcto en Colombia (COT = UTC-5),
//   evitando que correos de ayer aparezcan como de hoy.
//
//   Ejemplo para modo "hoy" a las 21:00 COT del 02/03/2026:
//     after:1740891600  → medianoche COT 02/03/2026 = 05:00 UTC 02/03/2026
//     before:1740978000 → medianoche COT 03/03/2026 = 05:00 UTC 03/03/2026
//
// ═══════════════════════════════════════════════════════════════════════════════
// FORMATOS DE CORREO SOPORTADOS
// ═══════════════════════════════════════════════════════════════════════════════
//
//   Formato A — Transferencia clásica:
//     "Bancolombia: Recibiste una transferencia por $65,000 de DAVID SIERRA
//      en tu cuenta **8368, el 07/05/2025 a las 13:59."
//     Captura: [monto=$65000] [remitente=DAVID SIERRA]
//
//   Formato B — Llave Bancolombia / Transferencia dirigida:
//     "Bancolombia: RECEPTOR, recibiste una transferencia de SANDRA LORENA
//      VALENCIA CORREA por $70,000.00 en tu producto *8368 conectado a la
//      llave email@gmail.com el 27/05/25 a las 13:05."
//     Captura: [remitente=SANDRA LORENA VALENCIA CORREA] [monto=$70000.00]
//
//   Separador de miles: coma  |  Decimal: punto
//     $65,000 → 65000    |    $70,000.00 → 70000.00
//
//   NOTA UNICODE: Los patrones de nombre usan \p{L} (letra Unicode) en lugar
//   de [A-Z], lo que permite capturar nombres con Ñ, Á, É, etc.
//   (p.ej. "JHOFER LONDOÑO" fallaba con [A-Z] porque Ñ está fuera del rango ASCII).
//
// ═══════════════════════════════════════════════════════════════════════════════
// DETECCIÓN DEL TIPO DE TRANSFERENCIA
// ═══════════════════════════════════════════════════════════════════════════════
//
//   El campo concepto almacena el método de pago detectado del texto:
//     "llave"        → "Por llave Bancolombia"
//     "código qr"    → "Por código QR"
//     (otro)         → "Transferencia normal"
//
// ═══════════════════════════════════════════════════════════════════════════════
// MÉTODOS PÚBLICOS EXPUESTOS A WAILS
// ═══════════════════════════════════════════════════════════════════════════════
//
//   EstadoAuth()                          → BancolombiaAuthStatus
//     Verifica si credentials.json y bancolombia_token.json existen.
//
//   IniciarOAuth2()                       → (url string, err error)
//     Inicia el flujo OAuth2. Devuelve la URL de autorización de Google.
//     El frontend abre el navegador; el callback guarda el token automáticamente.
//
//   RevocarAuth()                         → error
//     Elimina el token guardado y detiene el ticker.
//
//   SincronizarConPeriodo(opts)           → (BancolombiaCheckResult, error)
//     Ejecuta una sincronización inmediata para el período indicado.
//     Modos: "hoy" | "semana" | "mes" | "rango" | "completo"
//     Emite eventos bancolombia:sync:log en tiempo real y bancolombia:sync:result al finalizar.
//
//   ObtenerTransferencias(page, pageSize, soloNoLeidas) → TransferenciasResponse
//     Devuelve transferencias paginadas desde la BD. (En bancolombia_db.go)
//
//   MarcarLeida(uuid)                     → error
//     Marca una transferencia específica como leída.
//
//   MarcarTodasLeidas()                   → error
//     Marca todas las transferencias no leídas como leídas y actualiza el badge.
//
//   EliminarTransferencia(uuid)           → error
//     Elimina permanentemente una transferencia de la BD.
//
//   GetAutoPolling()                      → BancolombiaAutoPollingState
//   SetAutoPolling(enabled bool)
//     Habilita/deshabilita el ticker de fondo.
//
// ═══════════════════════════════════════════════════════════════════════════════
// EVENTOS WAILS EMITIDOS
// ═══════════════════════════════════════════════════════════════════════════════
//
//   "bancolombia:nueva"       → TransferenciaBancolombia   (una por transferencia nueva)
//   "bancolombia:badge"       → int                        (cantidad de no leídas)
//   "bancolombia:sync:result" → BancolombiaCheckResult     (resumen al finalizar sync)
//   "bancolombia:sync:log"    → BancolombiaLogEntry        (línea de log en tiempo real)

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	bancolombiaCallbackPort = 8095
	bancolombiaCallbackPath = "/bancolombia/oauth2/callback"
	bancolombiaTickerPeriod = 2 * time.Minute
	bancolombiaCheckCount   = 30 // last N messages per ticker run
)

// ── Pre-compiled regexes ──────────────────────────────────────────────────────

var (
	// Date/time: "el 07/05/2025 a las 13:59" or "el 27/05/25 a las 13:05"
	reBancoFechaHora = regexp.MustCompile(
		`(?i)el (\d{2}/\d{2}/\d{2,4}) a las (\d{2}:\d{2})`,
	)

	// Format A: "transferencia por $65,000 de NOMBRE en tu"
	// Captures: [1]=monto  [2]=remitente
	// \p{L} matches any Unicode letter, covering names with Ñ, Á, É, etc.
	reBancoFormatoA = regexp.MustCompile(
		`(?i)transferencia por \$([\d,]+(?:\.[\d]{1,2})?) de ([\p{L}][\p{L} ]+?) (?:en tu|,)`,
	)

	// Format B: "transferencia de NOMBRE por $70,000.00"
	// Captures: [1]=remitente  [2]=monto
	reBancoFormatoB = regexp.MustCompile(
		`(?i)transferencia de ([\p{L}][\p{L} ]+?) por \$([\d,]+(?:\.[\d]{1,2})?)`,
	)

	// Account: "cuenta **8368", "cuenta *8368", "producto *8368"
	reBancoCuenta = regexp.MustCompile(
		`(?i)(?:cuenta|producto) \*+(\d+)`,
	)

	// HTML tag stripper
	reBancoHTMLTags   = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	reBancoHTMLStrip  = regexp.MustCompile(`<[^>]+>`)
	reBancoWhitespace = regexp.MustCompile(`\s+`)
)

// ── Service ───────────────────────────────────────────────────────────────────

// BancolombiaService handles Gmail polling for Bancolombia transfer notifications.
// It is bound to Wails and exposed to the frontend.
type BancolombiaService struct {
	ctx         context.Context
	db          *Db
	configDir   string
	ticker      *time.Ticker
	done        chan struct{}
	autoPolling bool
	syncMu      sync.Mutex // prevents concurrent sync runs that can deadlock the DB pool
}

// BancolombiaAuthStatus reports authentication state to the frontend.
type BancolombiaAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	CredPresent   bool   `json:"credPresent"`
	ConfigDir     string `json:"configDir"`
}

// BancolombiaCheckResult summarizes one sync run.
type BancolombiaCheckResult struct {
	Revisados int      `json:"revisados"`
	Nuevas    int      `json:"nuevas"`
	Errores   []string `json:"errores"`
	Ts        string   `json:"ts"`
}

// BancolombiaLogEntry is one line in the real-time sync log (same pattern as SyncLogEntry).
type BancolombiaLogEntry struct {
	Nivel   string `json:"nivel"`   // "info" | "ok" | "warn" | "error"
	Mensaje string `json:"mensaje"`
	Ts      string `json:"ts"`
}

// NewBancolombiaService creates the service. Shares configDir with GmailService.
func NewBancolombiaService(db *Db) *BancolombiaService {
	home, _ := os.UserHomeDir()
	return &BancolombiaService{
		db:          db,
		configDir:   filepath.Join(home, ".config", "goFarmacia"),
		done:        make(chan struct{}),
		autoPolling: true,
	}
}

// Startup is called by Wails when the app starts.
func (b *BancolombiaService) Startup(ctx context.Context) {
	b.ctx = ctx
	_ = os.MkdirAll(b.configDir, 0o700)

	go func() {
		time.Sleep(2 * time.Second)
		b.emitBadge()
	}()

	if b.EstadoAuth().Authenticated {
		b.startTicker()
	}
}

// Shutdown stops the background ticker gracefully.
func (b *BancolombiaService) Shutdown() {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
	if b.ticker != nil {
		b.ticker.Stop()
	}
}

func (b *BancolombiaService) startTicker() {
	if b.ticker != nil {
		return
	}
	b.ticker = time.NewTicker(bancolombiaTickerPeriod)
	go func() {
		for {
			select {
			case <-b.done:
				return
			case <-b.ticker.C:
				result, err := b.sincronizarRecientes()
				if err != nil {
					result.Errores = append(result.Errores, err.Error())
				}
				wailsruntime.EventsEmit(b.ctx, "bancolombia:sync:result", result)
				if result.Nuevas > 0 {
					b.emitBadge()
				}
			}
		}
	}()
}

// ── OAuth2 ────────────────────────────────────────────────────────────────────

func (b *BancolombiaService) credPath() string {
	return filepath.Join(b.configDir, "credentials.json")
}
func (b *BancolombiaService) tokenPath() string {
	return filepath.Join(b.configDir, "bancolombia_token.json")
}

// EstadoAuth checks whether credentials and a Bancolombia token are present.
func (b *BancolombiaService) EstadoAuth() BancolombiaAuthStatus {
	_, credErr := os.Stat(b.credPath())
	_, tokErr := os.Stat(b.tokenPath())
	return BancolombiaAuthStatus{
		Authenticated: credErr == nil && tokErr == nil,
		CredPresent:   credErr == nil,
		ConfigDir:     b.configDir,
	}
}

func (b *BancolombiaService) loadOAuth2Config() (*oauth2.Config, error) {
	data, err := os.ReadFile(b.credPath())
	if err != nil {
		return nil, fmt.Errorf("credentials.json no encontrado en %s", b.credPath())
	}
	cfg, err := google.ConfigFromJSON(data, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("credentials.json inválido: %w", err)
	}
	cfg.RedirectURL = fmt.Sprintf("http://localhost:%d%s", bancolombiaCallbackPort, bancolombiaCallbackPath)
	return cfg, nil
}

// IniciarOAuth2 starts the OAuth2 flow for the Bancolombia Gmail account.
// Returns the OAuth2 URL — the frontend is responsible for opening the browser.
func (b *BancolombiaService) IniciarOAuth2() (string, error) {
	cfg, err := b.loadOAuth2Config()
	if err != nil {
		return "", err
	}
	authURL := cfg.AuthCodeURL("bancolombia-state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	go b.startCallbackServer(cfg)
	return authURL, nil
}

// RevocarAuth deletes the saved token.
func (b *BancolombiaService) RevocarAuth() error {
	if b.ticker != nil {
		b.ticker.Stop()
		b.ticker = nil
	}
	return os.Remove(b.tokenPath())
}

func (b *BancolombiaService) startCallbackServer(cfg *oauth2.Config) {
	mux := http.NewServeMux()
	srv := &http.Server{Addr: fmt.Sprintf(":%d", bancolombiaCallbackPort), Handler: mux}

	mux.HandleFunc(bancolombiaCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Código no recibido", http.StatusBadRequest)
			return
		}
		tok, err := cfg.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "Error al intercambiar código: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := b.saveToken(tok); err != nil {
			http.Error(w, "Error al guardar token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html><head><meta charset="utf-8">
<style>body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f8fafc}
.card{background:#fff;border-radius:12px;padding:48px;text-align:center;box-shadow:0 4px 24px rgba(0,0,0,.08);max-width:400px}
h2{color:#f59e0b;margin-bottom:8px}p{color:#64748b}</style></head>
<body><div class="card"><h2>✅ Cuenta Bancolombia conectada</h2>
<p>Recibirás notificaciones de transferencias automáticamente. Puedes cerrar esta ventana.</p></div></body></html>`)
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = srv.Shutdown(context.Background())
			b.startTicker()
			b.emitBadge()
		}()
	})
	_ = srv.ListenAndServe()
}

func (b *BancolombiaService) saveToken(tok *oauth2.Token) error {
	f, err := os.OpenFile(b.tokenPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

// savingTokenSource wraps an oauth2.TokenSource and persists the token to disk
// every time the access token changes (i.e. after an automatic refresh).
// This prevents 401 errors after app restart when the stored access token has expired.
type savingTokenSource struct {
	mu    sync.Mutex
	inner oauth2.TokenSource
	last  string
	save  func(*oauth2.Token) error
}

func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, err := s.inner.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last {
		s.last = tok.AccessToken
		_ = s.save(tok)
	}
	return tok, nil
}

func (b *BancolombiaService) newGmailSvc() (*gmail.Service, error) {
	cfg, err := b.loadOAuth2Config()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(b.tokenPath())
	if err != nil {
		return nil, fmt.Errorf("no autenticado — ejecuta IniciarOAuth2 primero")
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("token inválido: %w", err)
	}
	src := &savingTokenSource{
		inner: cfg.TokenSource(context.Background(), tok),
		last:  tok.AccessToken,
		save:  b.saveToken,
	}
	httpClient := oauth2.NewClient(context.Background(), src)
	httpClient.Timeout = 90 * time.Second // prevent indefinite hangs that freeze the UI
	return gmail.NewService(context.Background(), option.WithHTTPClient(httpClient))
}

// ── Public API ────────────────────────────────────────────────────────────────

// VerificarAhora triggers an immediate sync and returns the result.
func (b *BancolombiaService) VerificarAhora() (BancolombiaCheckResult, error) {
	result, err := b.sincronizarRecientes()
	if err != nil {
		result.Errores = append(result.Errores, err.Error())
	}
	wailsruntime.EventsEmit(b.ctx, "bancolombia:sync:result", result)
	b.emitBadge()
	return result, err
}

// ObtenerTransferencias returns a paginated list of transfer notifications.
// busqueda filters by remitente/referencia/concepto; estado: "" | "vinculada" | "sin_vincular".
func (b *BancolombiaService) ObtenerTransferencias(page, pageSize int, soloNoLeidas bool, busqueda, estado string) (TransferenciasResponse, error) {
	return b.db.ObtenerTransferenciasPaginado(page, pageSize, soloNoLeidas, busqueda, estado)
}

// MarcarLeida marks a transfer as read and refreshes the badge count.
func (b *BancolombiaService) MarcarLeida(transferUUID string) error {
	if err := b.db.MarcarTransferenciaLeida(transferUUID); err != nil {
		return err
	}
	b.emitBadge()
	return nil
}

// ContarNoLeidas returns the number of unread transfer notifications.
func (b *BancolombiaService) ContarNoLeidas() int {
	return b.db.ContarTransferenciasNoLeidas()
}

// EliminarTransferencia removes a transfer notification by UUID.
func (b *BancolombiaService) EliminarTransferencia(transferUUID string) error {
	if err := b.db.EliminarTransferencia(transferUUID); err != nil {
		return err
	}
	b.emitBadge()
	return nil
}

// EliminarTransferencias bulk-removes multiple transfer notifications.
func (b *BancolombiaService) EliminarTransferencias(uuids []string) error {
	if err := b.db.EliminarTransferencias(uuids); err != nil {
		return err
	}
	b.emitBadge()
	return nil
}

// VincularFactura links a sale invoice to a transfer notification.
func (b *BancolombiaService) VincularFactura(transferUUID, facturaUUID, facturaNumero string) error {
	return b.db.VincularFactura(transferUUID, facturaUUID, facturaNumero)
}

// DesvincularFactura removes the invoice link from a transfer notification.
func (b *BancolombiaService) DesvincularFactura(transferUUID string) error {
	return b.db.DesvincularFactura(transferUUID)
}

// BuscarFacturasVenta searches sale invoices by number or client name.
// An empty busqueda returns the 50 most recent invoices.
func (b *BancolombiaService) BuscarFacturasVenta(busqueda string) ([]FacturaVentaResumen, error) {
	return b.db.BuscarFacturasVenta(busqueda, 50)
}

// BancolombiaAutoPollingState reports whether auto-polling is active.
type BancolombiaAutoPollingState struct {
	Enabled bool `json:"enabled"`
}

// GetAutoPolling returns whether the background ticker is enabled.
func (b *BancolombiaService) GetAutoPolling() BancolombiaAutoPollingState {
	return BancolombiaAutoPollingState{Enabled: b.autoPolling}
}

// SetAutoPolling enables or disables the background polling ticker.
func (b *BancolombiaService) SetAutoPolling(enabled bool) {
	b.autoPolling = enabled
	if !enabled && b.ticker != nil {
		b.ticker.Stop()
		b.ticker = nil
	} else if enabled && b.ticker == nil && b.EstadoAuth().Authenticated {
		b.startTicker()
	}
}

// BancolombiaOpcionesPeriodo holds sync parameters for a period-based sync call.
type BancolombiaOpcionesPeriodo struct {
	Modo  string `json:"modo"`  // hoy | semana | mes | rango | completo
	Desde string `json:"desde"` // YYYY-MM-DD, only for "rango"
	Hasta string `json:"hasta"` // YYYY-MM-DD, only for "rango"
}

// SincronizarConPeriodo performs an immediate sync for the given period.
func (b *BancolombiaService) SincronizarConPeriodo(opts BancolombiaOpcionesPeriodo) (BancolombiaCheckResult, error) {
	result, err := b.sincronizarConPeriodo(opts)
	if err != nil {
		result.Errores = append(result.Errores, err.Error())
	}
	wailsruntime.EventsEmit(b.ctx, "bancolombia:sync:result", result)
	b.emitBadge()
	return result, err
}

// MarcarTodasLeidas marks all transfer notifications as read.
func (b *BancolombiaService) MarcarTodasLeidas() error {
	err := b.db.MarcarTodasLeidas()
	b.emitBadge()
	return err
}

// ── Core sync logic ───────────────────────────────────────────────────────────

func (b *BancolombiaService) sincronizarRecientes() (BancolombiaCheckResult, error) {
	return b.sincronizarConPeriodo(BancolombiaOpcionesPeriodo{Modo: "semana"})
}

func (b *BancolombiaService) sincronizarConPeriodo(opts BancolombiaOpcionesPeriodo) (BancolombiaCheckResult, error) {
	result := BancolombiaCheckResult{
		Ts:      time.Now().Format("15:04:05"),
		Errores: []string{},
	}

	// Only one sync at a time — prevents DB pool exhaustion and UI freeze
	// if the ticker fires while a manual sync is already running.
	if !b.syncMu.TryLock() {
		b.emitLog("warn", "Sincronización ya en progreso, omitiendo.")
		return result, nil
	}
	defer b.syncMu.Unlock()

	svc, err := b.newGmailSvc()
	if err != nil {
		return result, err
	}

	// Both Bancolombia notification domains share the same subdomain suffix.
	// Gmail substring matches the domain, so one query covers both:
	//   alertasynotificaciones@notificacionesbancolombia.com
	//   alertasynotificaciones@an.notificacionesbancolombia.com
	query := `from:notificacionesbancolombia.com`
	maxResults := int64(bancolombiaCheckCount)

	// Use Colombia timezone (UTC-5) for all date calculations.
	// IMPORTANT: we use Unix timestamps (not date strings) in Gmail queries.
	// Date strings like "after:2026/03/02" are interpreted using the Gmail
	// *account* timezone, which may differ from COT. Unix timestamps are
	// absolute and timezone-agnostic, guaranteeing correct midnight boundaries.
	col := time.FixedZone("COT", -5*60*60)
	now := time.Now().In(col)

	// startOfDay returns midnight COT for the given time value.
	startOfDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, col)
	}

	switch opts.Modo {
	case "hoy":
		from := startOfDay(now)
		to := from.AddDate(0, 0, 1)
		query += fmt.Sprintf(" after:%d before:%d", from.Unix(), to.Unix())
		maxResults = 50
	case "semana":
		from := startOfDay(now.AddDate(0, 0, -7))
		query += fmt.Sprintf(" after:%d", from.Unix())
		maxResults = 100
	case "mes":
		from := startOfDay(now.AddDate(0, -1, 0))
		query += fmt.Sprintf(" after:%d", from.Unix())
		maxResults = 200
	case "rango":
		if opts.Desde != "" {
			d, _ := time.ParseInLocation("2006-01-02", opts.Desde, col)
			query += fmt.Sprintf(" after:%d", startOfDay(d).Unix())
		}
		if opts.Hasta != "" {
			h, _ := time.ParseInLocation("2006-01-02", opts.Hasta, col)
			query += fmt.Sprintf(" before:%d", startOfDay(h).AddDate(0, 0, 1).Unix())
		}
		maxResults = 500
	case "completo":
		maxResults = 500
	}

	// Fetch ALL matching messages by following NextPageToken pagination.
	var allMessages []*gmail.Message
	pageToken := ""
	for {
		req := svc.Users.Messages.List("me").Q(query).MaxResults(maxResults)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		resp, err := req.Do()
		if err != nil {
			return result, fmt.Errorf("error listando mensajes Bancolombia: %w", err)
		}
		allMessages = append(allMessages, resp.Messages...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	b.emitLog("info", fmt.Sprintf("Revisando %d mensajes del período…", len(allMessages)))

	for _, m := range allMessages {
		result.Revisados++
		if b.db.ExisteTransferencia(m.Id) {
			continue
		}
		t, err := b.parsearMensaje(svc, m.Id)
		if err != nil || t == nil {
			continue
		}
		isNew, err := b.db.GuardarTransferencia(*t)
		if err != nil {
			result.Errores = append(result.Errores, fmt.Sprintf("msg %s: %v", m.Id, err))
			b.emitLog("error", fmt.Sprintf("Error guardando mensaje %s: %v", m.Id, err))
			continue
		}
		if isNew {
			result.Nuevas++
			b.emitLog("ok", fmt.Sprintf("✓ %s — $%.0f (%s)", t.Remitente, t.Monto, t.RawSubject))
			wailsruntime.EventsEmit(b.ctx, "bancolombia:nueva", t)
		}
	}

	b.emitLog("info", fmt.Sprintf(
		"Finalizado — %d revisados, %d nuevas, %d errores",
		result.Revisados, result.Nuevas, len(result.Errores),
	))

	return result, nil
}

// parsearMensaje downloads a Gmail message and extracts transfer details.
// Returns nil if the message is not a recognizable transfer notification.
func (b *BancolombiaService) parsearMensaje(svc *gmail.Service, messageID string) (*TransferenciaBancolombia, error) {
	msg, err := svc.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return nil, err
	}

	var subject, dateHeader string
	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "Subject":
			subject = h.Value
		case "Date":
			dateHeader = h.Value
		}
	}

	// Try the subject first — many Bancolombia notifications put the full
	// transfer text directly in the subject line (SMS-forwarded style).
	if t := parseBancolombiaTransfer(subject, messageID, dateHeader); t != nil {
		return t, nil
	}

	// Fall back to body text.
	body := bancolombiaExtractBody(msg.Payload)
	return parseBancolombiaTransfer(body, messageID, dateHeader), nil
}

// ── Parser ────────────────────────────────────────────────────────────────────

// parseBancolombiaTransfer extracts transfer details from a text string using
// the two known Bancolombia notification formats.
//
// Returns nil if the text does not match either format.
func parseBancolombiaTransfer(text, emailID, dateHeader string) *TransferenciaBancolombia {
	text = strings.TrimSpace(text)
	if text == "" || !strings.Contains(strings.ToLower(text), "transferencia") {
		return nil
	}

	var monto float64
	var remitente string

	// ── Format A: "transferencia por $MONTO de NOMBRE en tu cuenta/producto"
	if m := reBancoFormatoA.FindStringSubmatch(text); len(m) == 3 {
		monto = parseBancolombiaAmount(m[1])
		remitente = strings.TrimSpace(m[2])
	}

	// ── Format B: "transferencia de NOMBRE por $MONTO en tu cuenta/producto"
	if remitente == "" || monto <= 0 {
		if m := reBancoFormatoB.FindStringSubmatch(text); len(m) == 3 {
			remitente = strings.TrimSpace(m[1])
			monto = parseBancolombiaAmount(m[2])
		}
	}

	if monto <= 0 || remitente == "" {
		return nil
	}

	// Account number
	var cuenta string
	if m := reBancoCuenta.FindStringSubmatch(text); len(m) == 2 {
		cuenta = "*" + m[1]
	}

	// Date/time from body, then from email Date header as fallback
	var fechaStr string
	if m := reBancoFechaHora.FindStringSubmatch(text); len(m) == 3 {
		fechaStr = m[1] + " " + m[2]
	}
	fecha := parseBancolombiaDate(fechaStr, dateHeader)

	// Extract the full notification sentence (the line that contains "transferencia")
	// from the input text — this preserves the original wording including date/time.
	rawSubject := bancolombiaExtractSentence(text)
	if rawSubject == "" {
		rawSubject = fmt.Sprintf("Transferencia de %s por $%.0f", remitente, monto)
	}

	// Detect transfer method from the notification text
	tipoTransfer := bancolombiaDetectTipo(text)

	return &TransferenciaBancolombia{
		UUID:          uuid.NewString(),
		EmailID:       emailID,
		Fecha:         fecha,
		Monto:         monto,
		Remitente:     remitente,
		CuentaDestino: cuenta,
		// Concepto: not present in Bancolombia transfer notifications; reuse for type
		Concepto:   tipoTransfer,
		RawSubject: rawSubject,
	}
}

// parseBancolombiaAmount parses Bancolombia's amount format.
//
// Bancolombia uses comma as thousands separator and period as decimal:
//
//	$65,000      → 65000
//	$3,500       → 3500
//	$70,000.00   → 70000.00
//	$10,000.00   → 10000.00
func parseBancolombiaAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "") // remove thousands separator
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}

// parseBancolombiaDate parses the date extracted from a notification text.
// Accepts DD/MM/YYYY and DD/MM/YY formats. Falls back to email Date header.
func parseBancolombiaDate(dateTimeStr, dateHeader string) time.Time {
	for _, f := range []string{"02/01/2006 15:04", "02/01/06 15:04"} {
		if t, err := time.ParseInLocation(f, dateTimeStr, time.Local); err == nil {
			return t
		}
	}
	// RFC2822 email Date header
	for _, f := range []string{
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
	} {
		if t, err := time.Parse(f, dateHeader); err == nil {
			return t.Local()
		}
	}
	return time.Now()
}

// ── Body extraction ───────────────────────────────────────────────────────────

// bancolombiaExtractBody walks the MIME tree and returns the best plain text.
func bancolombiaExtractBody(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}
	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
		if b, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
			return string(b)
		}
	}

	var plain, htmlBody string
	for _, child := range part.Parts {
		switch child.MimeType {
		case "text/plain":
			if child.Body != nil && child.Body.Data != "" {
				if b, err := base64.URLEncoding.DecodeString(child.Body.Data); err == nil {
					plain += string(b)
				}
			}
		case "text/html":
			if child.Body != nil && child.Body.Data != "" {
				if b, err := base64.URLEncoding.DecodeString(child.Body.Data); err == nil {
					htmlBody += string(b)
				}
			}
		default:
			if t := bancolombiaExtractBody(child); t != "" {
				plain += t
			}
		}
	}

	if plain != "" {
		return plain
	}
	return bancolombiaStripHTML(htmlBody)
}

// bancolombiaStripHTML removes HTML tags and decodes common entities.
func bancolombiaStripHTML(s string) string {
	s = reBancoHTMLTags.ReplaceAllString(s, " ")
	s = reBancoHTMLStrip.ReplaceAllString(s, " ")
	s = strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&aacute;", "á", "&eacute;", "é", "&iacute;", "í",
		"&oacute;", "ó", "&uacute;", "ú", "&ntilde;", "ñ",
		"&Aacute;", "Á", "&Eacute;", "É", "&Iacute;", "Í",
		"&Oacute;", "Ó", "&Uacute;", "Ú", "&Ntilde;", "Ñ",
	).Replace(s)
	return strings.TrimSpace(reBancoWhitespace.ReplaceAllString(s, " "))
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// bancolombiaExtractSentence finds and returns the sentence (or full text if short)
// containing the word "transferencia", without URL artifacts.
func bancolombiaExtractSentence(text string) string {
	// Split on newlines and periods to get candidate sentences
	for _, sep := range []string{"\n", ". "} {
		for _, line := range strings.Split(text, sep) {
			line = strings.TrimSpace(line)
			if strings.Contains(strings.ToLower(line), "transferencia") {
				// Skip lines that are just HTML artifacts (contain URLs or start with "Logo")
				if strings.Contains(line, "http") || strings.HasPrefix(strings.ToLower(line), "logo") {
					continue
				}
				// Trim "Bancolombia: " prefix if present
				line = strings.TrimPrefix(line, "Bancolombia: ")
				return truncate(line, 300)
			}
		}
	}
	return ""
}

// bancolombiaDetectTipo determines the transfer method from the notification text.
func bancolombiaDetectTipo(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "llave"):
		return "Por llave Bancolombia"
	case strings.Contains(lower, "código qr"), strings.Contains(lower, "codigo qr"),
		strings.Contains(lower, " qr "):
		return "Por código QR"
	default:
		return "Transferencia normal"
	}
}

func (b *BancolombiaService) emitBadge() {
	n := b.db.ContarTransferenciasNoLeidas()
	wailsruntime.EventsEmit(b.ctx, "bancolombia:badge", n)
}

func (b *BancolombiaService) emitLog(nivel, mensaje string) {
	entry := BancolombiaLogEntry{
		Nivel:   nivel,
		Mensaje: mensaje,
		Ts:      time.Now().Format("15:04:05"),
	}
	wailsruntime.EventsEmit(b.ctx, "bancolombia:sync:log", entry)
}
