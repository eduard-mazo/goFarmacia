package backend

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

const (
	outlookCallbackPort    = 8097
	outlookCallbackPath    = "/outlook/oauth2/callback"
	outlookSyncWorkers     = 5
	graphBaseURL           = "https://graph.microsoft.com/v1.0"
	outlookAutoSyncPeriod  = 30 * time.Minute
	outlookAutoSyncSetting = "outlook.autoSync"
)

// OutlookService handles Microsoft OAuth2 auth and DIAN electronic invoice
// synchronization via Microsoft Graph API (Outlook mail).
type OutlookService struct {
	db             *Db
	configDir      string
	syncProgress   GmailSyncProgress // reuse same progress struct
	syncProgressMu sync.Mutex

	// Auto-sync daemon
	autoMu   sync.Mutex
	autoSync bool
	ticker   *time.Ticker
	done     chan struct{}
	lastSync time.Time
	nextSync time.Time
}

// OutlookCredentials holds Azure AD app registration fields.
// Accepts both camelCase (Azure portal style) and snake_case keys.
type OutlookCredentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	TenantID     string `json:"tenantId"` // "common", "consumers", or a specific tenant
}

// OutlookCredencialesInfo holds safe-to-display fields of the credentials.
type OutlookCredencialesInfo struct {
	Exists   bool   `json:"Exists"`
	Path     string `json:"Path"`
	ClientID string `json:"ClientID"`
	TenantID string `json:"TenantID"`
}

// NewOutlookService creates the service.
func NewOutlookService(db *Db) *OutlookService {
	home, _ := os.UserHomeDir()
	return &OutlookService{
		db:        db,
		configDir: filepath.Join(home, ".config", "goFarmacia"),
		done:      make(chan struct{}),
	}
}

// Startup initialises the service (idempotent).
func (o *OutlookService) Startup(_ context.Context) {
	_ = os.MkdirAll(o.configDir, 0o700)

	// Restore auto-sync daemon state from persisted setting.
	if val, err := o.db.GetSetting(outlookAutoSyncSetting); err == nil && val == "true" {
		o.autoMu.Lock()
		o.autoSync = true
		o.autoMu.Unlock()
		if o.EstadoAuth().Authenticated {
			o.startAutoTicker()
		}
	}
}

// Shutdown stops the background ticker.
func (o *OutlookService) Shutdown() {
	o.autoMu.Lock()
	defer o.autoMu.Unlock()
	if o.ticker != nil {
		o.ticker.Stop()
		o.ticker = nil
	}
	select {
	case <-o.done:
	default:
		close(o.done)
	}
}

// OutlookAutoSyncState reports the auto-sync daemon state to the frontend.
type OutlookAutoSyncState struct {
	Enabled       bool   `json:"enabled"`
	Running       bool   `json:"running"`
	Authenticated bool   `json:"authenticated"`
	NextSync      string `json:"nextSync"`
	LastSync      string `json:"lastSync"`
}

// GetAutoSync returns the current auto-sync daemon state.
func (o *OutlookService) GetAutoSync() OutlookAutoSyncState {
	o.autoMu.Lock()
	defer o.autoMu.Unlock()
	state := OutlookAutoSyncState{
		Enabled:       o.autoSync,
		Running:       o.ticker != nil,
		Authenticated: o.EstadoAuth().Authenticated,
	}
	if !o.nextSync.IsZero() && o.autoSync {
		state.NextSync = o.nextSync.Format(time.RFC3339)
	}
	if !o.lastSync.IsZero() {
		state.LastSync = o.lastSync.Format(time.RFC3339)
	}
	return state
}

// SetAutoSync enables or disables the background sync ticker and persists it.
func (o *OutlookService) SetAutoSync(enabled bool) {
	o.autoMu.Lock()
	o.autoSync = enabled
	if !enabled && o.ticker != nil {
		o.ticker.Stop()
		o.ticker = nil
		// Signal the running goroutine to exit (see matching comment in
		// GmailService.SetAutoSync): ticker.Stop does not close ticker.C, so
		// without this close() the goroutine leaks on every toggle.
		select {
		case <-o.done:
		default:
			close(o.done)
		}
	}
	shouldStart := enabled && o.ticker == nil
	o.autoMu.Unlock()

	val := "false"
	if enabled {
		val = "true"
	}
	_ = o.db.SetSetting(outlookAutoSyncSetting, val)

	if shouldStart && o.EstadoAuth().Authenticated {
		o.startAutoTicker()
	}
}

func (o *OutlookService) startAutoTicker() {
	o.autoMu.Lock()
	if o.ticker != nil {
		o.autoMu.Unlock()
		return
	}
	select {
	case <-o.done:
		o.done = make(chan struct{})
	default:
	}
	o.ticker = time.NewTicker(outlookAutoSyncPeriod)
	o.nextSync = time.Now().Add(outlookAutoSyncPeriod)
	done := o.done
	ticker := o.ticker
	o.autoMu.Unlock()

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				o.autoMu.Lock()
				enabled := o.autoSync
				o.autoMu.Unlock()
				if !enabled {
					continue
				}
				if !o.EstadoAuth().Authenticated {
					continue
				}
				result := o.ejecutarSync(SyncOptions{Modo: "hoy"})
				o.autoMu.Lock()
				o.lastSync = time.Now()
				o.nextSync = time.Now().Add(outlookAutoSyncPeriod)
				o.autoMu.Unlock()
				EventBus.Emit("outlook:sync:result", result)
			}
		}
	}()
}

func (o *OutlookService) credPath() string {
	return filepath.Join(o.configDir, "outlook_credentials.json")
}
func (o *OutlookService) tokenPath() string {
	return filepath.Join(o.configDir, "outlook_token.json")
}

// ─── Credentials management ─────────────────────────────────────────────────

func (o *OutlookService) loadCredentials() (OutlookCredentials, error) {
	var creds OutlookCredentials
	data, err := os.ReadFile(o.credPath())
	if err != nil {
		return creds, fmt.Errorf("outlook_credentials.json no encontrado en %s", o.credPath())
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return creds, fmt.Errorf("outlook_credentials.json inválido: %w", err)
	}
	if creds.ClientID == "" {
		return creds, fmt.Errorf("clientId vacío en outlook_credentials.json")
	}
	if creds.TenantID == "" {
		creds.TenantID = "common"
	}
	return creds, nil
}

// ObtenerCredenciales returns safe-to-display credential fields.
func (o *OutlookService) ObtenerCredenciales() OutlookCredencialesInfo {
	info := OutlookCredencialesInfo{Path: o.credPath()}
	creds, err := o.loadCredentials()
	if err != nil {
		return info
	}
	info.Exists = true
	info.ClientID = creds.ClientID
	info.TenantID = creds.TenantID
	return info
}

// GuardarCredenciales validates and saves credentials JSON.
func (o *OutlookService) GuardarCredenciales(jsonContent string) error {
	var creds OutlookCredentials
	if err := json.Unmarshal([]byte(jsonContent), &creds); err != nil {
		return fmt.Errorf("JSON inválido: %w", err)
	}
	if creds.ClientID == "" {
		return fmt.Errorf("clientId es requerido")
	}
	if creds.TenantID == "" {
		creds.TenantID = "common"
	}
	data, _ := json.MarshalIndent(creds, "", "  ")
	return os.WriteFile(o.credPath(), data, 0o600)
}

// ─── OAuth2 ──────────────────────────────────────────────────────────────────

func (o *OutlookService) oauth2Config() (*oauth2.Config, error) {
	creds, err := o.loadCredentials()
	if err != nil {
		return nil, err
	}
	tenant := creds.TenantID
	if tenant == "" {
		tenant = "common"
	}
	return &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenant),
			TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant),
		},
		Scopes:      []string{"https://graph.microsoft.com/Mail.Read", "offline_access"},
		RedirectURL: fmt.Sprintf("http://localhost:%d%s", outlookCallbackPort, outlookCallbackPath),
	}, nil
}

// EstadoAuth reports whether credentials and token are present.
func (o *OutlookService) EstadoAuth() GmailAuthStatus {
	_, credErr := os.Stat(o.credPath())
	_, tokErr := os.Stat(o.tokenPath())
	return GmailAuthStatus{
		Authenticated: credErr == nil && tokErr == nil,
		CredPresent:   credErr == nil,
		ConfigDir:     o.configDir,
	}
}

// IniciarOAuth2 starts the Microsoft OAuth2 flow.
func (o *OutlookService) IniciarOAuth2() (string, error) {
	cfg, err := o.oauth2Config()
	if err != nil {
		return "", err
	}
	authURL := cfg.AuthCodeURL("state-token",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	go o.startCallbackServer(cfg)
	return authURL, nil
}

// RevocarAuth deletes the saved token.
func (o *OutlookService) RevocarAuth() error {
	o.autoMu.Lock()
	if o.ticker != nil {
		o.ticker.Stop()
		o.ticker = nil
		select {
		case <-o.done:
		default:
			close(o.done)
		}
	}
	o.autoMu.Unlock()
	return os.Remove(o.tokenPath())
}

func (o *OutlookService) startCallbackServer(cfg *oauth2.Config) {
	mux := http.NewServeMux()
	srv := &http.Server{Addr: fmt.Sprintf(":%d", outlookCallbackPort), Handler: mux}

	mux.HandleFunc(outlookCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}
		tok, err := cfg.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = o.saveToken(tok)
		// If auto-sync was enabled before token existed, start ticker now.
		o.autoMu.Lock()
		shouldStart := o.autoSync && o.ticker == nil
		o.autoMu.Unlock()
		if shouldStart {
			o.startAutoTicker()
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding:3rem">
			<h2>Microsoft autenticado correctamente</h2>
			<p>Puedes cerrar esta pestaña y volver a la aplicación.</p>
			<script>setTimeout(()=>window.close(),2000)</script></body></html>`)
		go func() {
			time.Sleep(2 * time.Second)
			_ = srv.Shutdown(context.Background())
		}()
	})

	_ = srv.ListenAndServe()
}

func (o *OutlookService) saveToken(tok *oauth2.Token) error {
	data, _ := json.MarshalIndent(tok, "", "  ")
	return os.WriteFile(o.tokenPath(), data, 0o600)
}

func (o *OutlookService) httpClient() (*http.Client, error) {
	cfg, err := o.oauth2Config()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(o.tokenPath())
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
		save:  o.saveToken,
	}
	return oauth2.NewClient(context.Background(), src), nil
}

// ─── Sync ────────────────────────────────────────────────────────────────────

// GetOutlookSyncProgress returns the current sync state (polled by frontend).
func (o *OutlookService) GetOutlookSyncProgress() GmailSyncProgress {
	o.syncProgressMu.Lock()
	defer o.syncProgressMu.Unlock()
	return o.syncProgress
}

// SincronizarConOpciones starts Outlook sync in a background goroutine.
func (o *OutlookService) SincronizarConOpciones(opts SyncOptions) {
	go func() {
		result := o.ejecutarSync(opts)
		EventBus.Emit("outlook:sync:result", result)
	}()
}

func (o *OutlookService) ejecutarSync(opts SyncOptions) SyncResult {
	result := SyncResult{Errores: []string{}, Log: []SyncLogEntry{}}
	var logMu sync.Mutex

	// Reset in-memory progress state.
	o.syncProgressMu.Lock()
	o.syncProgress = GmailSyncProgress{
		Running: true,
		Fase:    "recolectando",
		Log:     []SyncLogEntry{}, // Clear logs for the new run
	}
	o.syncProgressMu.Unlock()

	log := func(nivel, msg string) {
		entry := SyncLogEntry{
			Nivel:   nivel,
			Mensaje: msg,
			Ts:      time.Now().Format("15:04:05"),
		}
		logMu.Lock()
		result.Log = append(result.Log, entry)
		logMu.Unlock()

		o.syncProgressMu.Lock()
		o.syncProgress.Log = append(o.syncProgress.Log, entry)
		if len(o.syncProgress.Log) > 200 {
			o.syncProgress.Log = o.syncProgress.Log[len(o.syncProgress.Log)-200:]
		}
		o.syncProgressMu.Unlock()

		// Also emit via EventBus for real-time UI updates
		EventBus.Emit("outlook:sync:log", entry)
	}

	defer func() {
		o.syncProgressMu.Lock()
		o.syncProgress.Running = false
		o.syncProgress.Fase = ""
		o.syncProgressMu.Unlock()
	}()

	client, err := o.httpClient()
	if err != nil {
		log("error", "Error autenticando Microsoft: "+err.Error())
		return result
	}

	log("info", fmt.Sprintf("Modo: %s — recolectando IDs de mensajes (Outlook)…", opts.Modo))

	// ── Phase 1: collect message IDs via Microsoft Graph ──────────────────────
	allIDs, err := o.collectMessageIDs(client, opts)
	if err != nil {
		log("error", "Error listando mensajes: "+err.Error())
		result.Errores = append(result.Errores, err.Error())
		return result
	}

	o.syncProgressMu.Lock()
	o.syncProgress.Total = len(allIDs)
	o.syncProgressMu.Unlock()

	if len(allIDs) == 0 {
		log("warn", "Sin facturas con adjunto ZIP en el rango seleccionado")
		return result
	}

	result.Total = len(allIDs)
	log("info", fmt.Sprintf("%d mensajes encontrados — procesando con %d workers…", len(allIDs), outlookSyncWorkers))

	o.syncProgressMu.Lock()
	o.syncProgress.Fase = "procesando"
	o.syncProgressMu.Unlock()

	// ── Phase 2: parallel worker pool ─────────────────────────────────────────
	jobs := make(chan string, len(allIDs))
	resultCh := make(chan msgProcessResult, outlookSyncWorkers*4)

	var wg sync.WaitGroup
	for i := 0; i < outlookSyncWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msgID := range jobs {
				resultCh <- o.procesarMensaje(client, msgID)
			}
		}()
	}

	for _, id := range allIDs {
		jobs <- id
	}
	close(jobs)

	go func() { wg.Wait(); close(resultCh) }()

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

		o.syncProgressMu.Lock()
		o.syncProgress.Procesados++
		o.syncProgress.Nuevas = result.Nuevas
		o.syncProgress.Duplicadas = result.Duplicadas
		o.syncProgress.Errores = len(result.Errores)
		if mr.ultimoNro != "" {
			o.syncProgress.UltimoNro = mr.ultimoNro
		}
		o.syncProgressMu.Unlock()
	}

	log("ok", fmt.Sprintf(
		"Listo — %d revisados · %d nuevas · %d duplicadas · %d errores",
		result.Total, result.Nuevas, result.Duplicadas, len(result.Errores),
	))

	if result.Nuevas > 0 {
		if n, err := o.db.SincronizarProveedoresDesdeFacturas(); err == nil && n > 0 {
			log("info", fmt.Sprintf("Proveedores: %d registros actualizados", n))
		}
	}

	return result
}

// ─── Microsoft Graph helpers ─────────────────────────────────────────────────

// graphMessage is the subset of fields we need from a Graph message response.
type graphMessage struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
}

type graphMessageList struct {
	Value    []graphMessage `json:"value"`
	NextLink string         `json:"@odata.nextLink"`
}

// graphAttachment represents a single Graph attachment.
type graphAttachment struct {
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	ContentBytes string `json:"contentBytes"` // base64-encoded
}

type graphAttachmentList struct {
	Value []graphAttachment `json:"value"`
}

// collectMessageIDs pages through Graph messages matching the date filter.
func (o *OutlookService) collectMessageIDs(
	client *http.Client, opts SyncOptions,
) ([]string, error) {
	filter := "hasAttachments eq true"
	dateFilter := o.buildDateFilter(opts)
	if dateFilter != "" {
		filter += " and " + dateFilter
	}

	endpoint := fmt.Sprintf(
		"%s/me/messages?$filter=%s&$top=100&$select=id,subject&$orderby=receivedDateTime desc",
		graphBaseURL, url.QueryEscape(filter),
	)

	var allIDs []string
	for endpoint != "" {
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("graph request: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("graph %d: %s", resp.StatusCode, string(body))
		}

		var list graphMessageList
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("graph decode: %w", err)
		}

		for _, m := range list.Value {
			allIDs = append(allIDs, m.ID)
		}

		o.syncProgressMu.Lock()
		o.syncProgress.Total = len(allIDs)
		o.syncProgressMu.Unlock()

		endpoint = list.NextLink
	}
	return allIDs, nil
}

// buildDateFilter converts SyncOptions into a Microsoft Graph $filter clause.
func (o *OutlookService) buildDateFilter(opts SyncOptions) string {
	now := time.Now().In(time.Local)
	iso := func(t time.Time) string { return t.Format("2006-01-02T15:04:05Z") }

	switch opts.Modo {
	case "hoy":
		d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		return fmt.Sprintf("receivedDateTime ge %s", iso(d))
	case "semana":
		d := now.AddDate(0, 0, -7)
		return fmt.Sprintf("receivedDateTime ge %s", iso(d))
	case "mes":
		d := now.AddDate(0, -1, 0)
		return fmt.Sprintf("receivedDateTime ge %s", iso(d))
	case "rango":
		parts := []string{}
		if opts.Desde != "" {
			parts = append(parts, fmt.Sprintf("receivedDateTime ge %sT00:00:00Z", opts.Desde))
		}
		if opts.Hasta != "" {
			if t, err := time.ParseInLocation("2006-01-02", opts.Hasta, time.Local); err == nil {
				parts = append(parts, fmt.Sprintf("receivedDateTime lt %s", iso(t.AddDate(0, 0, 1))))
			}
		}
		return strings.Join(parts, " and ")
	default: // "completo"
		return ""
	}
}

// procesarMensaje fetches one Outlook message, extracts ZIP, and delegates
// to the shared procesarZipContenido function.
func (o *OutlookService) procesarMensaje(client *http.Client, messageID string) (mr msgProcessResult) {
	emit := func(nivel, msg string) {
		mr.logs = append(mr.logs, SyncLogEntry{
			Nivel:   nivel,
			Mensaje: msg,
			Ts:      time.Now().Format("15:04:05"),
		})
	}

	// Get message subject
	subject, err := o.getMessageSubject(client, messageID)
	if err != nil {
		mr.errMsg = fmt.Sprintf("get message: %v", err)
		return
	}

	// Download ZIP attachment
	zipData, err := o.extractZipAttachment(client, messageID)
	if err != nil {
		// Not a DIAN invoice email — silently skip.
		return
	}

	// Preserve logs appended via the emit closure — see matching comment in
	// gmail_service.go procesarMensaje.
	result := procesarZipContenido(o.db, zipData, "outlook:"+messageID, subject, emit)
	result.logs = mr.logs
	return result
}

func (o *OutlookService) getMessageSubject(client *http.Client, messageID string) (string, error) {
	endpoint := fmt.Sprintf("%s/me/messages/%s?$select=subject", graphBaseURL, messageID)
	resp, err := client.Do(mustReq("GET", endpoint))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("graph %d: %s", resp.StatusCode, string(body))
	}
	var msg graphMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return "", err
	}
	return msg.Subject, nil
}

func (o *OutlookService) extractZipAttachment(client *http.Client, messageID string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/me/messages/%s/attachments", graphBaseURL, messageID)
	resp, err := client.Do(mustReq("GET", endpoint))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("graph %d: %s", resp.StatusCode, string(body))
	}

	var list graphAttachmentList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	var lastErr error
	for _, att := range list.Value {
		if !strings.HasSuffix(strings.ToLower(att.Name), ".zip") || att.ContentBytes == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(att.ContentBytes)
		if err == nil && len(data) > 0 {
			return data, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("error decodificando ZIP: %w", lastErr)
	}
	return nil, fmt.Errorf("no ZIP attachment")
}

// mustReq creates a simple GET request — panics only on malformed URLs (never in practice).
func mustReq(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	return req
}
