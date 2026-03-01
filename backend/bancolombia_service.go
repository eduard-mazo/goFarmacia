package backend

// bancolombia_service.go — Gmail sync for Bancolombia transfer notifications.
//
// Strategy: hybrid polling
//   • Background ticker fires every 2 minutes and checks the last 30 messages
//     from Bancolombia. This keeps the notification feed current without manual
//     intervention — important for a pharmacy POS where staff need to confirm
//     that a payment arrived before releasing product.
//   • Manual "Verificar ahora" button triggers an immediate check on demand.
//
// OAuth2: reuses the same credentials.json as the DIAN Gmail service but saves
// its token to a separate file (bancolombia_token.json), allowing the user to
// authenticate a different Gmail account.
//
// Gmail query: from:notificaciones.bancolombia.com.co OR from:mensajeria.bancolombia.com.co
//
// Wails events emitted:
//   "bancolombia:nueva"        → TransferenciaBancolombia  (one per new transfer)
//   "bancolombia:badge"        → int  (count of unread notifications)
//   "bancolombia:sync:result"  → BancolombiaCheckResult

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

// BancolombiaService handles Gmail polling for Bancolombia transfer notifications.
// It is bound to Wails and exposed to the frontend.
type BancolombiaService struct {
	ctx       context.Context
	db        *Db
	configDir string
	ticker    *time.Ticker
	done      chan struct{}
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

// NewBancolombiaService creates the service. Shares configDir with GmailService.
func NewBancolombiaService(db *Db) *BancolombiaService {
	home, _ := os.UserHomeDir()
	return &BancolombiaService{
		db:        db,
		configDir: filepath.Join(home, ".config", "goFarmacia"),
		done:      make(chan struct{}),
	}
}

// Startup is called by Wails when the app starts.
// It launches the background polling ticker.
func (b *BancolombiaService) Startup(ctx context.Context) {
	b.ctx = ctx
	_ = os.MkdirAll(b.configDir, 0o700)

	// Emit current badge count on startup so the sidebar shows unread count.
	go func() {
		time.Sleep(2 * time.Second) // Wait for DB to be ready
		b.emitBadge()
	}()

	// Start background ticker only if already authenticated.
	if b.EstadoAuth().Authenticated {
		b.startTicker()
	}
}

// Shutdown stops the background ticker gracefully.
func (b *BancolombiaService) Shutdown() {
	select {
	case <-b.done:
		// Already closed
	default:
		close(b.done)
	}
	if b.ticker != nil {
		b.ticker.Stop()
	}
}

func (b *BancolombiaService) startTicker() {
	if b.ticker != nil {
		return // Already running
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

// ── OAuth2 ──────────────────────────────────────────────────────────────────

func (b *BancolombiaService) credPath() string  { return filepath.Join(b.configDir, "credentials.json") }
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
// Returns the auth URL as fallback if the browser does not open automatically.
func (b *BancolombiaService) IniciarOAuth2() (string, error) {
	cfg, err := b.loadOAuth2Config()
	if err != nil {
		return "", err
	}
	authURL := cfg.AuthCodeURL("bancolombia-state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	wailsruntime.BrowserOpenURL(b.ctx, authURL)
	go b.startCallbackServer(cfg)
	return authURL, nil
}

// RevocarAuth deletes the saved token, forcing re-authentication.
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
			// Start ticker now that we have a token.
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
	client := cfg.Client(context.Background(), tok)
	return gmail.NewService(context.Background(), option.WithHTTPClient(client))
}

// ── Public API ───────────────────────────────────────────────────────────────

// VerificarAhora triggers an immediate sync and returns the result.
// The background ticker continues to run independently.
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
func (b *BancolombiaService) ObtenerTransferencias(page, pageSize int, soloNoLeidas bool) (TransferenciasResponse, error) {
	return b.db.ObtenerTransferenciasPaginado(page, pageSize, soloNoLeidas)
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

// ── Core sync logic ──────────────────────────────────────────────────────────

func (b *BancolombiaService) sincronizarRecientes() (BancolombiaCheckResult, error) {
	result := BancolombiaCheckResult{
		Ts:     time.Now().Format("15:04:05"),
		Errores: []string{},
	}

	svc, err := b.newGmailSvc()
	if err != nil {
		return result, err
	}

	// Query: any email from Bancolombia notification domains, last N messages.
	const query = `from:(notificaciones.bancolombia.com.co OR mensajeria.bancolombia.com.co OR bancolombia.com)`

	resp, err := svc.Users.Messages.List("me").
		Q(query).
		MaxResults(bancolombiaCheckCount).
		Do()
	if err != nil {
		return result, fmt.Errorf("error listando mensajes Bancolombia: %w", err)
	}

	for _, m := range resp.Messages {
		result.Revisados++
		if b.db.ExisteTransferencia(m.Id) {
			continue
		}
		t, err := b.parsearMensaje(svc, m.Id)
		if err != nil || t == nil {
			// Not a transfer notification — skip silently.
			continue
		}
		isNew, err := b.db.GuardarTransferencia(*t)
		if err != nil {
			result.Errores = append(result.Errores, fmt.Sprintf("msg %s: %v", m.Id, err))
			continue
		}
		if isNew {
			result.Nuevas++
			wailsruntime.EventsEmit(b.ctx, "bancolombia:nueva", t)
		}
	}

	return result, nil
}

// parsearMensaje downloads and parses a Gmail message, returning a
// TransferenciaBancolombia if it contains a transfer notification, or nil
// if it is not a transfer message (PSE, Nequi, direct transfer).
func (b *BancolombiaService) parsearMensaje(svc *gmail.Service, messageID string) (*TransferenciaBancolombia, error) {
	msg, err := svc.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return nil, err
	}

	// Extract headers.
	var subject, dateStr string
	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "Subject":
			subject = h.Value
		case "Date":
			dateStr = h.Value
		}
	}

	// Only process messages that look like transfer notifications.
	subjectLower := strings.ToLower(subject)
	isTransfer := strings.Contains(subjectLower, "transferencia") ||
		strings.Contains(subjectLower, "recibiste") ||
		strings.Contains(subjectLower, "recaudos") ||
		strings.Contains(subjectLower, "pago") ||
		strings.Contains(subjectLower, "abono") ||
		strings.Contains(subjectLower, "nequi") ||
		strings.Contains(subjectLower, "pse")
	if !isTransfer {
		return nil, nil
	}

	// Extract body text.
	body := extractMessageBody(msg.Payload)
	if body == "" {
		return nil, nil
	}

	// Parse the relevant fields.
	monto := parseMontoCOP(body)
	if monto <= 0 {
		return nil, nil // Not a money notification
	}

	var fecha time.Time
	if t, err := parseBancolombiaDate(body, dateStr); err == nil {
		fecha = t
	} else {
		fecha = time.Now()
	}

	return &TransferenciaBancolombia{
		UUID:          uuid.NewString(),
		EmailID:       messageID,
		Fecha:         fecha,
		Monto:         monto,
		Remitente:     parseField(body, `(?i)(?:de|remitente|ordenante|pagador)\s*:?\s*([A-ZÁÉÍÓÚÑ][^\n<$]{2,50})`),
		Referencia:    parseField(body, `(?i)(?:confirmaci[oó]n|referencia|n[uú]mero|comprobante)\s*[:#]?\s*(\d{6,20})`),
		CuentaDestino: parseField(body, `(?i)(?:cuenta\s+(?:de\s+)?(?:destino|beneficiario|cr[eé]dito))\s*:?\s*([*\d]+)`),
		Concepto:      parseField(body, `(?i)(?:concepto|descripci[oó]n|detalle)\s*:?\s*([^\n<]{3,80})`),
		RawSubject:    subject,
	}, nil
}

// ── Body extraction ──────────────────────────────────────────────────────────

// extractMessageBody walks the MIME tree and returns plain text.
// Falls back to HTML with tags stripped.
func extractMessageBody(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	// Prefer text/plain
	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
		b, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return string(b)
		}
	}

	// Collect text from multipart children.
	var plain, html string
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
					html += string(b)
				}
			}
		default:
			// Recurse into nested multipart
			if t := extractMessageBody(child); t != "" {
				plain += t
			}
		}
	}

	if plain != "" {
		return plain
	}
	if html != "" {
		return stripHTML(html)
	}
	return ""
}

// stripHTML removes HTML tags and decodes basic entities.
func stripHTML(s string) string {
	// Remove script/style blocks.
	reScript := regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	s = reScript.ReplaceAllString(s, " ")
	// Remove all tags.
	reTags := regexp.MustCompile(`<[^>]+>`)
	s = reTags.ReplaceAllString(s, " ")
	// Decode common entities.
	s = strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&aacute;", "á", "&eacute;", "é", "&iacute;", "í",
		"&oacute;", "ó", "&uacute;", "ú", "&ntilde;", "ñ",
		"&Aacute;", "Á", "&Eacute;", "É", "&Iacute;", "Í",
		"&Oacute;", "Ó", "&Uacute;", "Ú", "&Ntilde;", "Ñ",
	).Replace(s)
	// Collapse whitespace.
	reWS := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(reWS.ReplaceAllString(s, " "))
}

// ── Parsing helpers ──────────────────────────────────────────────────────────

// parseMontoCOP extracts the transfer amount from the email body.
// Bancolombia format: "$1.500.000,00" or "$1.500.000" or "1500000".
func parseMontoCOP(body string) float64 {
	// Look for patterns like $1.500.000,00 or $ 1.500.000 or COP 1.500.000
	reAmount := regexp.MustCompile(
		`(?i)(?:\$|COP)\s*([\d]{1,3}(?:\.[\d]{3})*(?:,[\d]{1,2})?)`,
	)
	matches := reAmount.FindAllStringSubmatch(body, -1)

	var best float64
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		// Normalize: remove thousand-dots, replace decimal-comma with dot
		raw := strings.ReplaceAll(m[1], ".", "")
		raw = strings.ReplaceAll(raw, ",", ".")
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v > best {
			best = v
		}
	}
	return best
}

// parseBancolombiaDate tries to find a date in the email body or falls back
// to the RFC2822 Date header.
func parseBancolombiaDate(body, dateHeader string) (time.Time, error) {
	// Try body patterns like "01/03/2026 10:35 a.m." or "2026-03-01"
	reDate := regexp.MustCompile(`(\d{2}/\d{2}/\d{4})\s+(\d{1,2}:\d{2})\s*(?:a\.?m\.?|p\.?m\.?)?`)
	if m := reDate.FindStringSubmatch(body); len(m) >= 3 {
		s := m[1] + " " + m[2]
		if t, err := time.ParseInLocation("02/01/2006 15:04", s, time.Local); err == nil {
			return t, nil
		}
	}

	// Parse RFC2822 Date header.
	formats := []string{
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, dateHeader); err == nil {
			return t.Local(), nil
		}
	}
	return time.Time{}, fmt.Errorf("no date found")
}

// parseField extracts the first capture group of a regex from the body text.
func parseField(body, pattern string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(body)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (b *BancolombiaService) emitBadge() {
	n := b.db.ContarTransferenciasNoLeidas()
	wailsruntime.EventsEmit(b.ctx, "bancolombia:badge", n)
}
