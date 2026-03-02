package backend

// bancolombia_service.go — Gmail sync for Bancolombia transfer notifications.
//
// Strategy: hybrid polling
//   • Background ticker fires every 2 minutes and checks the last 30 messages
//     from the Bancolombia notification sender. This keeps the notification
//     feed current without manual intervention — important for a pharmacy POS
//     where staff need to confirm payment before releasing product.
//   • Manual "Verificar ahora" button for immediate on-demand checking.
//
// OAuth2: reuses the same credentials.json as the DIAN Gmail service but saves
// its token to a separate file (bancolombia_token.json), allowing a different
// Gmail account to be authenticated.
//
// Gmail query: from:notificacionesbancolombia.com
//   Matches both sender domains used by Bancolombia:
//     alertasynotificaciones@notificacionesbancolombia.com
//     alertasynotificaciones@an.notificacionesbancolombia.com
//
// Supported email formats
// ──────────────────────────────────────────────────────────────────────────
// Format A (classic):
//   "Bancolombia: Recibiste una transferencia por $65,000 de DAVID SIERRA
//    en tu cuenta **8368, el 07/05/2025 a las 13:59."
//
// Format B (Llave Bancolombia):
//   "Bancolombia: RECEPTOR, recibiste una transferencia de SANDRA LORENA
//    VALENCIA CORREA por $70,000.00 en tu producto *8368 conectado a la
//    llave email@gmail.com el 27/05/25 a las 13:05."
//
// Amount format: comma = thousands separator, period = decimal
//   $65,000 → 65000    |    $70,000.00 → 70000.00
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

// ── Pre-compiled regexes ──────────────────────────────────────────────────────

var (
	// Date/time: "el 07/05/2025 a las 13:59" or "el 27/05/25 a las 13:05"
	reBancoFechaHora = regexp.MustCompile(
		`(?i)el (\d{2}/\d{2}/\d{2,4}) a las (\d{2}:\d{2})`,
	)

	// Format A: "transferencia por $65,000 de NOMBRE en tu"
	// Captures: [1]=monto  [2]=remitente
	reBancoFormatoA = regexp.MustCompile(
		`(?i)transferencia por \$([\d,]+(?:\.[\d]{1,2})?) de ([A-Z][A-Z ]+?) (?:en tu|,)`,
	)

	// Format B: "transferencia de NOMBRE por $70,000.00"
	// Captures: [1]=remitente  [2]=monto
	reBancoFormatoB = regexp.MustCompile(
		`(?i)transferencia de ([A-Z][A-Z ]+?) por \$([\d,]+(?:\.[\d]{1,2})?)`,
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

// ── Core sync logic ───────────────────────────────────────────────────────────

func (b *BancolombiaService) sincronizarRecientes() (BancolombiaCheckResult, error) {
	result := BancolombiaCheckResult{
		Ts:      time.Now().Format("15:04:05"),
		Errores: []string{},
	}

	svc, err := b.newGmailSvc()
	if err != nil {
		return result, err
	}

	// Both Bancolombia notification domains share the same subdomain suffix.
	// Gmail substring matches the domain, so one query covers both:
	//   alertasynotificaciones@notificacionesbancolombia.com
	//   alertasynotificaciones@an.notificacionesbancolombia.com
	const query = `from:notificacionesbancolombia.com`

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

	return &TransferenciaBancolombia{
		UUID:          uuid.NewString(),
		EmailID:       emailID,
		Fecha:         fecha,
		Monto:         monto,
		Remitente:     remitente,
		CuentaDestino: cuenta,
		// Concepto: not present in Bancolombia transfer notifications
		RawSubject: truncate(text, 120),
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

func (b *BancolombiaService) emitBadge() {
	n := b.db.ContarTransferenciasNoLeidas()
	wailsruntime.EventsEmit(b.ctx, "bancolombia:badge", n)
}
