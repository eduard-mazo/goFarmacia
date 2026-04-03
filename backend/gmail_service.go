package backend

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goFarmacia/internal/processor"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	gmailCallbackPort = 8094
	gmailCallbackPath = "/gmail/oauth2/callback"
	gmailSyncWorkers  = 5 // concurrent message-processing goroutines
)

// GmailService handles Gmail OAuth2 auth and DIAN electronic invoice synchronization.
type GmailService struct {
	db             *Db
	configDir      string
	syncProgress   GmailSyncProgress
	syncProgressMu sync.Mutex
}

// GmailAuthStatus reports the current authentication state to the frontend.
type GmailAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	CredPresent   bool   `json:"credPresent"`
	ConfigDir     string `json:"configDir"`
}

// CredencialesInfo holds the safe-to-display fields of credentials.json.
type CredencialesInfo struct {
	Exists    bool   `json:"Exists"`
	Path      string `json:"Path"`
	ProjectID string `json:"ProjectID"`
	ClientID  string `json:"ClientID"`
}

// SyncOptions controls the scope of a Gmail synchronization run.
type SyncOptions struct {
	// Modo: "hoy" | "semana" | "mes" | "rango" | "completo"
	Modo  string `json:"modo"`
	Desde string `json:"desde"` // YYYY-MM-DD (only used when Modo == "rango")
	Hasta string `json:"hasta"` // YYYY-MM-DD (only used when Modo == "rango")
}

// SyncLogEntry is one line in the real-time sync log sent to the frontend.
type SyncLogEntry struct {
	Nivel   string `json:"nivel"`   // "info" | "ok" | "warn" | "error"
	Mensaje string `json:"mensaje"`
	Ts      string `json:"ts"`
}

// SyncResult summarizes one synchronization run.
type SyncResult struct {
	Total      int            `json:"total"`
	Nuevas     int            `json:"nuevas"`
	Duplicadas int            `json:"duplicadas"`
	Errores    []string       `json:"errores"`
	Log        []SyncLogEntry `json:"log"`
}

// GmailSyncProgress holds the current state of an in-progress sync.
// Written from the sync goroutine under syncProgressMu; read via GetGmailSyncProgress
// (JS→Go call — safe on Linux/WebKit2GTK, does NOT use g_idle_add).
type GmailSyncProgress struct {
	Running    bool   `json:"running"`
	Fase       string `json:"fase"`       // "recolectando" | "procesando" | ""
	Total      int    `json:"total"`      // total message IDs found so far
	Procesados int    `json:"procesados"` // messages fully processed
	Nuevas     int    `json:"nuevas"`
	Duplicadas int    `json:"duplicadas"`
	Errores    int    `json:"errores"`
	UltimoNro  string `json:"ultimoNro"` // last invoice number processed
}

// msgProcessResult is the per-message output of procesarMensaje.
// Designed for safe concurrent use in the worker pool.
type msgProcessResult struct {
	nuevas     int
	duplicadas int
	ultimoNro  string
	logs       []SyncLogEntry
	errMsg     string // non-empty on error
}

// NewGmailService creates the service. Credentials and token are stored in
// ~/.config/goFarmacia/ so they survive app updates.
func NewGmailService(db *Db) *GmailService {
	home, _ := os.UserHomeDir()
	return &GmailService{
		db:        db,
		configDir: filepath.Join(home, ".config", "goFarmacia"),
	}
}

// GetGmailSyncProgress returns the current in-progress sync state.
// Safe to call from the frontend at any time (JS→Go message, not g_idle_add).
func (g *GmailService) GetGmailSyncProgress() GmailSyncProgress {
	g.syncProgressMu.Lock()
	defer g.syncProgressMu.Unlock()
	return g.syncProgress
}

// Startup initialises the service.
func (g *GmailService) Startup(_ context.Context) {
	_ = os.MkdirAll(g.configDir, 0o700)
	// Retroactively populate the proveedors table from any existing invoices.
	// This is idempotent and runs in the background so it does not block startup.
	go func() { _, _ = g.db.SincronizarProveedoresDesdeFacturas() }()
}

func (g *GmailService) credPath() string  { return filepath.Join(g.configDir, "credentials.json") }
func (g *GmailService) tokenPath() string { return filepath.Join(g.configDir, "gmail_token.json") }

// ConfigDir returns the directory path where credentials.json must be placed.
func (g *GmailService) ConfigDir() string { return g.configDir }

// EstadoAuth checks whether OAuth2 credentials and token are present.
func (g *GmailService) EstadoAuth() GmailAuthStatus {
	_, credErr := os.Stat(g.credPath())
	_, tokErr := os.Stat(g.tokenPath())
	return GmailAuthStatus{
		Authenticated: credErr == nil && tokErr == nil,
		CredPresent:   credErr == nil,
		ConfigDir:     g.configDir,
	}
}

func (g *GmailService) loadOAuth2Config() (*oauth2.Config, error) {
	b, err := os.ReadFile(g.credPath())
	if err != nil {
		return nil, fmt.Errorf("credentials.json no encontrado en %s", g.credPath())
	}
	cfg, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("credentials.json inválido: %w", err)
	}
	cfg.RedirectURL = fmt.Sprintf("http://localhost:%d%s", gmailCallbackPort, gmailCallbackPath)
	return cfg, nil
}

// IniciarOAuth2 builds the Google auth URL, opens the system browser, and
// starts a local callback server on port 8094 to capture the auth code automatically.
// Returns the auth URL so the frontend can show it as a fallback link.
func (g *GmailService) IniciarOAuth2() (string, error) {
	cfg, err := g.loadOAuth2Config()
	if err != nil {
		return "", err
	}
	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	go g.startCallbackServer(cfg)
	return authURL, nil
}

// RevocarAuth deletes the saved token, forcing re-authentication.
func (g *GmailService) RevocarAuth() error {
	return os.Remove(g.tokenPath())
}

func (g *GmailService) startCallbackServer(cfg *oauth2.Config) {
	mux := http.NewServeMux()
	srv := &http.Server{Addr: fmt.Sprintf(":%d", gmailCallbackPort), Handler: mux}

	mux.HandleFunc(gmailCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Código de autorización no recibido", http.StatusBadRequest)
			return
		}
		tok, err := cfg.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "Error al intercambiar el código: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := g.saveToken(tok); err != nil {
			http.Error(w, "Error al guardar el token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html><head><meta charset="utf-8">
<style>body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f8fafc}
.card{background:#fff;border-radius:12px;padding:48px;text-align:center;box-shadow:0 4px 24px rgba(0,0,0,.08);max-width:400px}
h2{color:#16a34a;margin-bottom:8px}p{color:#64748b}</style></head>
<body><div class="card"><h2>✅ Autenticación exitosa</h2>
<p>Gmail conectado correctamente. Puedes cerrar esta ventana.</p></div></body></html>`)
		go func() {
			time.Sleep(500 * time.Millisecond)
			srv.Shutdown(context.Background())
		}()
	})
	_ = srv.ListenAndServe()
}

func (g *GmailService) saveToken(tok *oauth2.Token) error {
	f, err := os.OpenFile(g.tokenPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

func (g *GmailService) newGmailSvc() (*gmail.Service, error) {
	cfg, err := g.loadOAuth2Config()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(g.tokenPath())
	if err != nil {
		return nil, fmt.Errorf("no autenticado — ejecuta IniciarOAuth2 primero")
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("token inválido: %w", err)
	}
	// savingTokenSource (defined in bancolombia_service.go, same package) persists
	// refreshed access tokens to disk so subsequent app restarts don't hit 401/invalid_grant.
	src := &savingTokenSource{inner: cfg.TokenSource(context.Background(), tok), save: g.saveToken}
	client := oauth2.NewClient(context.Background(), src)
	return gmail.NewService(context.Background(), option.WithHTTPClient(client))
}

// SincronizarFacturas runs a full historical sync. Kept for backward compat.
func (g *GmailService) SincronizarFacturas() {
	g.SincronizarConOpciones(SyncOptions{Modo: "completo"})
}

// SincronizarConOpciones starts Gmail sync in a background goroutine and returns
// immediately so the Wails call never blocks the UI thread.
// Events emitted:
//
//	"gmail:sync:log"      → SyncLogEntry  (only for new invoices and errors; NOT for duplicates)
//	"gmail:sync:progreso" → progress counters (emitted every 10 messages)
//	"gmail:sync:result"   → SyncResult  (emitted once when complete)
func (g *GmailService) SincronizarConOpciones(opts SyncOptions) {
	go func() {
		result := g.ejecutarSync(opts)
		EventBus.Emit("gmail:sync:result", result)
	}()
}

// ejecutarSync performs the actual Gmail sync in two phases:
//  1. Collect all message IDs via paginated list calls (fast).
//  2. Process messages concurrently with gmailSyncWorkers goroutines.
//
// NO EventsEmit calls happen here — all results are collected in memory and
// delivered in ONE "gmail:sync:result" event at the end. In-flight progress
// is exposed via GetGmailSyncProgress() for frontend polling.
func (g *GmailService) ejecutarSync(opts SyncOptions) SyncResult {
	result := SyncResult{Errores: []string{}, Log: []SyncLogEntry{}}
	var logMu sync.Mutex
	log := func(nivel, msg string) {
		logMu.Lock()
		result.Log = append(result.Log, SyncLogEntry{
			Nivel:   nivel,
			Mensaje: msg,
			Ts:      time.Now().Format("15:04:05"),
		})
		logMu.Unlock()
	}

	// Reset in-memory progress state.
	g.syncProgressMu.Lock()
	g.syncProgress = GmailSyncProgress{Running: true, Fase: "recolectando"}
	g.syncProgressMu.Unlock()
	defer func() {
		g.syncProgressMu.Lock()
		g.syncProgress.Running = false
		g.syncProgress.Fase = ""
		g.syncProgressMu.Unlock()
	}()

	svc, err := g.newGmailSvc()
	if err != nil {
		log("error", "Error autenticando Gmail: "+err.Error())
		return result
	}

	query := g.buildGmailQuery(opts)
	log("info", fmt.Sprintf("Modo: %s — recolectando IDs de mensajes…", opts.Modo))

	// ── Phase 1: collect all message IDs across all pages ─────────────────────
	var allIDs []string
	var pageToken string
	for {
		call := svc.Users.Messages.List("me").Q(query).MaxResults(500)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			log("error", "Error listando mensajes: "+err.Error())
			result.Errores = append(result.Errores, err.Error())
			return result
		}
		for _, m := range resp.Messages {
			allIDs = append(allIDs, m.Id)
		}
		// Update live count so frontend can show "encontrados: N" while collecting.
		g.syncProgressMu.Lock()
		g.syncProgress.Total = len(allIDs)
		g.syncProgressMu.Unlock()

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	if len(allIDs) == 0 {
		log("warn", "Sin facturas con adjunto ZIP en el rango seleccionado")
		return result
	}

	result.Total = len(allIDs)
	log("info", fmt.Sprintf("%d mensajes encontrados — procesando con %d workers…", len(allIDs), gmailSyncWorkers))

	g.syncProgressMu.Lock()
	g.syncProgress.Fase = "procesando"
	g.syncProgressMu.Unlock()

	// ── Phase 2: parallel worker pool ─────────────────────────────────────────
	jobs := make(chan string, len(allIDs))
	resultCh := make(chan msgProcessResult, gmailSyncWorkers*4)

	var wg sync.WaitGroup
	for i := 0; i < gmailSyncWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msgID := range jobs {
				resultCh <- g.procesarMensaje(svc, msgID)
			}
		}()
	}

	// Feed all IDs (non-blocking — channel is pre-sized).
	for _, id := range allIDs {
		jobs <- id
	}
	close(jobs)

	// Close result channel once all workers finish.
	go func() { wg.Wait(); close(resultCh) }()

	// Collect results and update in-memory progress as each message finishes.
	for mr := range resultCh {
		result.Nuevas += mr.nuevas
		result.Duplicadas += mr.duplicadas
		if mr.errMsg != "" {
			result.Errores = append(result.Errores, mr.errMsg)
			log("error", fmt.Sprintf("✗ %s", mr.errMsg))
		}
		logMu.Lock()
		result.Log = append(result.Log, mr.logs...)
		logMu.Unlock()

		g.syncProgressMu.Lock()
		g.syncProgress.Procesados++
		g.syncProgress.Nuevas = result.Nuevas
		g.syncProgress.Duplicadas = result.Duplicadas
		g.syncProgress.Errores = len(result.Errores)
		if mr.ultimoNro != "" {
			g.syncProgress.UltimoNro = mr.ultimoNro
		}
		g.syncProgressMu.Unlock()
	}

	log("ok", fmt.Sprintf(
		"Listo — %d revisados · %d nuevas · %d duplicadas · %d errores",
		result.Total, result.Nuevas, result.Duplicadas, len(result.Errores),
	))

	if result.Nuevas > 0 {
		if n, err := g.db.SincronizarProveedoresDesdeFacturas(); err == nil && n > 0 {
			log("info", fmt.Sprintf("Proveedores: %d registros actualizados", n))
		}
	}

	return result
}

// buildGmailQuery converts SyncOptions into a Gmail search query string.
func (g *GmailService) buildGmailQuery(opts SyncOptions) string {
	base := "has:attachment filename:zip"
	now := time.Now().In(time.Local)

	switch opts.Modo {
	case "hoy":
		d := now.Format("2006/01/02")
		return fmt.Sprintf("%s after:%s", base, d)
	case "semana":
		d := now.AddDate(0, 0, -7).Format("2006/01/02")
		return fmt.Sprintf("%s after:%s", base, d)
	case "mes":
		d := now.AddDate(0, -1, 0).Format("2006/01/02")
		return fmt.Sprintf("%s after:%s", base, d)
	case "rango":
		q := base
		if opts.Desde != "" {
			q += " after:" + strings.ReplaceAll(opts.Desde, "-", "/")
		}
		if opts.Hasta != "" {
			// Gmail "before:" is exclusive, add one day
			if t, err := time.ParseInLocation("2006-01-02", opts.Hasta, time.Local); err == nil {
				q += " before:" + t.AddDate(0, 0, 1).Format("2006/01/02")
			}
		}
		return q
	default: // "completo"
		return base
	}
}

// procesarMensaje fetches, unzips, parses XML+PDF and saves one Gmail message.
// Designed to be called concurrently from the worker pool — writes no shared state.
func (g *GmailService) procesarMensaje(svc *gmail.Service, messageID string) (mr msgProcessResult) {
	emit := func(nivel, msg string) {
		mr.logs = append(mr.logs, SyncLogEntry{
			Nivel:   nivel,
			Mensaje: msg,
			Ts:      time.Now().Format("15:04:05"),
		})
	}

	msg, err := svc.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		mr.errMsg = fmt.Sprintf("get message: %v", err)
		return
	}

	subject := ""
	for _, h := range msg.Payload.Headers {
		if h.Name == "Subject" {
			subject = h.Value
			break
		}
	}

	zipData, err := g.extractZipAttachment(svc, msg)
	if err != nil {
		// Not a DIAN invoice email — silently skip.
		return
	}

	emit("info", fmt.Sprintf("→ ZIP: %s", subject))

	unzipped, err := processor.UnzipInMemoryAll(zipData)
	if err != nil || len(unzipped.XMLFiles) == 0 {
		emit("warn", fmt.Sprintf("  ZIP sin XML válido: %s", subject))
		return
	}

	// Parse PDF (if present) in parallel with XML processing.
	var (
		pdfData processor.PDFInvoiceData
		pdfWg   sync.WaitGroup
	)
	if len(unzipped.PDFFiles) > 0 {
		pdfWg.Add(1)
		go func() {
			defer pdfWg.Done()
			if d, e := processor.ParsePDFBytes(unzipped.PDFFiles[0], nil); e == nil {
				pdfData = d
			}
		}()
	}

	for _, xmlData := range unzipped.XMLFiles {
		products, err := processor.ParseDocumentXMLBytes(xmlData)
		if err != nil || len(products) == 0 {
			emit("warn", fmt.Sprintf("  XML no parseable: %v", err))
			continue
		}

		pdfWg.Wait()
		if pdfData.RawText != "" {
			codes := make([]string, 0, len(products))
			for _, p := range products {
				if p.Code != "" {
					codes = append(codes, p.Code)
				}
			}
			if refined, e := processor.ParsePDFBytes(unzipped.PDFFiles[0], codes); e == nil {
				pdfData = refined
			}
			products = processor.EnrichProductsFromPDF(products, pdfData)
		}

		cufe := extractCUFE(xmlData)
		sample := products[0]
		tipoLabel := tipoDocumentoLabel(sample.DocumentType)

		exists, _ := g.db.ExisteFacturaCompra(messageID, cufe)
		if exists {
			mr.duplicadas++
			return
		}

		fechaEmision, _ := time.ParseInLocation("2006-01-02", sample.IssueDate, time.Local)
		factura := FacturaCompra{
			UUID:                uuid.NewString(),
			ProveedorNIT:        sample.SupplierNIT,
			ProveedorNombre:     sample.SupplierName,
			ClienteNIT:          sample.CustomerNIT,
			ClienteNombre:       sample.CustomerName,
			NumeroFactura:       sample.InvoiceID,
			CUFE:                cufe,
			FechaEmision:        fechaEmision,
			Moneda:              sample.Currency,
			Total:               sample.TotalInvoice,
			Estado:              "PENDIENTE",
			TipoDocumento:       sample.DocumentType,
			ReferenciaDocumento: sample.ReferenciaDoc,
			EmailMessageID:      messageID,
		}

		var totalIVA, subtotal float64
		for _, p := range products {
			totalIVA += p.TaxAmount
			subtotal += p.Total
			factura.Detalles = append(factura.Detalles, FacturaCompraDetalle{
				UUID:           uuid.NewString(),
				CodigoProducto: p.Code,
				Descripcion:    p.Description,
				Cantidad:       p.Quantity,
				PrecioUnitario: p.UnitPrice,
				TotalLinea:     p.Total,
				ImpuestoLinea:  p.TaxAmount,
				Propiedades:    p.Properties,
			})
		}
		factura.Subtotal = subtotal
		factura.IVA = totalIVA

		if err := g.db.GuardarFacturaCompra(factura); err != nil {
			mr.errMsg = fmt.Sprintf("guardar factura %s: %v", sample.InvoiceID, err)
			return
		}
		_ = g.db.UpsertProveedorPorNIT(sample.SupplierNIT, sample.SupplierName)
		mr.nuevas++
		mr.ultimoNro = sample.InvoiceID
		emit("ok", fmt.Sprintf(
			"  ✓ %s [%s] — %s — %d líneas",
			sample.InvoiceID, tipoLabel, sample.SupplierName, len(products),
		))
	}
	return
}

func (g *GmailService) extractZipAttachment(svc *gmail.Service, msg *gmail.Message) ([]byte, error) {
	for _, part := range msg.Payload.Parts {
		if !strings.HasSuffix(strings.ToLower(part.Filename), ".zip") || part.Body == nil {
			continue
		}
		data, err := g.downloadAttachmentPart(svc, msg.Id, part)
		if err == nil && len(data) > 0 {
			return data, nil
		}
	}
	return nil, fmt.Errorf("no ZIP attachment")
}

// downloadAttachmentPart downloads a single Gmail message part (attachment).
// It handles both inline base64 data and remote attachment IDs.
func (g *GmailService) downloadAttachmentPart(
	svc *gmail.Service, messageID string, part *gmail.MessagePart,
) ([]byte, error) {
	if part.Body == nil {
		return nil, fmt.Errorf("no body")
	}
	if len(part.Body.Data) > 0 {
		return base64.URLEncoding.DecodeString(part.Body.Data)
	}
	if part.Body.AttachmentId != "" {
		att, err := svc.Users.Messages.Attachments.Get("me", messageID, part.Body.AttachmentId).Do()
		if err != nil {
			return nil, err
		}
		return base64.URLEncoding.DecodeString(att.Data)
	}
	return nil, fmt.Errorf("no data and no attachment ID")
}

// emitSyncLog emits a single log entry on the "gmail:sync:log" Wails event.
// Used by background jobs (enrichment, backfill) to report progress to the UI.
func (g *GmailService) emitSyncLog(nivel, msg string) {
	entry := SyncLogEntry{
		Nivel:   nivel,
		Mensaje: msg,
		Ts:      time.Now().Format("15:04:05"),
	}
	EventBus.Emit("gmail:sync:log", entry)
}

// extractCUFE scans raw XML bytes for the CUFE/UUID value used for deduplication.
// tipoDocumentoLabel returns a human-readable short label for DIAN document types.
func tipoDocumentoLabel(tipo string) string {
	switch tipo {
	case "01", "02":
		return "Factura"
	case "91":
		return "Nota Crédito"
	case "92":
		return "Nota Débito"
	default:
		if tipo == "" {
			return "Factura"
		}
		return tipo
	}
}

func extractCUFE(xmlData []byte) string {
	s := string(xmlData)
	// Look for the CUFE attribute pattern in UBL 2.1 DIAN invoices
	markers := []string{`schemeName="CUFE-SHA384">`, `schemeName="CUFE-SHA256">`}
	for _, marker := range markers {
		start := strings.Index(s, marker)
		if start == -1 {
			continue
		}
		rest := s[start+len(marker):]
		end := strings.Index(rest, "<")
		if end > 0 {
			return strings.TrimSpace(rest[:end])
		}
	}
	return ""
}

// ObtenerFacturasCompra returns a paginated list of purchase invoices for the frontend.
func (g *GmailService) ObtenerFacturasCompra(page, pageSize int, busqueda, sortField, sortDir string) (FacturasCompraResponse, error) {
	return g.db.ObtenerFacturasCompraPaginado(page, pageSize, busqueda, sortField, sortDir)
}

// ObtenerDetalleFacturaCompra returns one purchase invoice with all line items.
func (g *GmailService) ObtenerDetalleFacturaCompra(facturaUUID string) (FacturaCompra, error) {
	return g.db.ObtenerDetalleFacturaCompra(facturaUUID)
}

// ActualizarEstadoFacturaCompra updates the processing estado of a purchase invoice.
func (g *GmailService) ActualizarEstadoFacturaCompra(facturaUUID, estado string) error {
	return g.db.ActualizarEstadoFacturaCompra(facturaUUID, estado)
}

// ObtenerProveedoresConEstadisticas returns paginated suppliers enriched with purchase stats.
func (g *GmailService) ObtenerProveedoresConEstadisticas(page, pageSize int, busqueda string) (ProveedoresStatsResponse, error) {
	return g.db.ObtenerProveedoresConEstadisticas(page, pageSize, busqueda)
}

// ObtenerTopProductosDeProveedor returns the top purchased products for a given supplier NIT.
func (g *GmailService) ObtenerTopProductosDeProveedor(nit string, limit int) ([]ProductoComprado, error) {
	return g.db.ObtenerTopProductosDeProveedor(nit, limit)
}

// ObtenerResumenCompras returns aggregate purchase KPIs for the dashboard.
func (g *GmailService) ObtenerResumenCompras(desde, hasta string) (ResumenCompras, error) {
	return g.db.ObtenerResumenCompras(desde, hasta)
}

// SincronizarProveedoresDesdeFacturas bridges existing purchase invoices into
// the proveedors table. Returns the number of rows upserted.
func (g *GmailService) SincronizarProveedoresDesdeFacturas() (int, error) {
	return g.db.SincronizarProveedoresDesdeFacturas()
}

// ObtenerCredenciales reads credentials.json and returns its safe-to-display fields.
func (g *GmailService) ObtenerCredenciales() CredencialesInfo {
	path := g.credPath()
	info := CredencialesInfo{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		return info
	}
	info.Exists = true
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return info
	}
	for _, key := range []string{"installed", "web"} {
		obj, ok := raw[key].(map[string]any)
		if !ok {
			continue
		}
		if v, ok := obj["project_id"].(string); ok {
			info.ProjectID = v
		}
		if v, ok := obj["client_id"].(string); ok {
			info.ClientID = v
		}
		break
	}
	return info
}

// GuardarCredenciales validates and writes new credentials.json content.
func (g *GmailService) GuardarCredenciales(jsonContent string) error {
	var raw map[string]any
	if err := json.Unmarshal([]byte(jsonContent), &raw); err != nil {
		return fmt.Errorf("JSON inválido: %w", err)
	}
	hasKey := false
	for _, key := range []string{"installed", "web"} {
		if _, ok := raw[key]; ok {
			hasKey = true
			break
		}
	}
	if !hasKey {
		return fmt.Errorf("el JSON debe tener la clave 'installed' o 'web'")
	}
	if err := os.MkdirAll(g.configDir, 0o700); err != nil {
		return fmt.Errorf("crear directorio de configuración: %w", err)
	}
	return os.WriteFile(g.credPath(), []byte(jsonContent), 0o600)
}
