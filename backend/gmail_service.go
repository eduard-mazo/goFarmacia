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
	"time"

	"goFarmacia/internal/processor"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	gmailCallbackPort = 8094
	gmailCallbackPath = "/gmail/oauth2/callback"
)

// GmailService handles Gmail OAuth2 auth and DIAN electronic invoice synchronization.
// It is bound to Wails and exposed to the frontend.
type GmailService struct {
	ctx       context.Context
	db        *Db
	configDir string
}

// GmailAuthStatus reports the current authentication state to the frontend.
type GmailAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	CredPresent   bool   `json:"credPresent"`
	ConfigDir     string `json:"configDir"`
}

// SyncResult summarizes one synchronization run.
type SyncResult struct {
	Total      int      `json:"total"`
	Nuevas     int      `json:"nuevas"`
	Duplicadas int      `json:"duplicadas"`
	Errores    []string `json:"errores"`
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

// Startup is called by Wails when the app starts.
func (g *GmailService) Startup(ctx context.Context) {
	g.ctx = ctx
	_ = os.MkdirAll(g.configDir, 0o700)
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
	wailsruntime.BrowserOpenURL(g.ctx, authURL)
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
	client := cfg.Client(context.Background(), tok)
	return gmail.NewService(context.Background(), option.WithHTTPClient(client))
}

// SincronizarFacturas downloads emails with ZIP attachments from Gmail,
// extracts DIAN XML invoices, parses them, and saves new ones to the database.
func (g *GmailService) SincronizarFacturas() (SyncResult, error) {
	svc, err := g.newGmailSvc()
	if err != nil {
		return SyncResult{}, err
	}

	result := SyncResult{Errores: []string{}}
	query := "has:attachment filename:zip"
	var pageToken string

	for {
		call := svc.Users.Messages.List("me").Q(query).MaxResults(100)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return result, fmt.Errorf("error listando mensajes Gmail: %w", err)
		}

		for _, m := range resp.Messages {
			result.Total++
			if err := g.procesarMensaje(svc, m.Id, &result); err != nil {
				result.Errores = append(result.Errores, fmt.Sprintf("msg %s: %v", m.Id, err))
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return result, nil
}

func (g *GmailService) procesarMensaje(svc *gmail.Service, messageID string, result *SyncResult) error {
	msg, err := svc.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}

	zipData, err := g.extractZipAttachment(svc, msg)
	if err != nil {
		return nil // Not a DIAN invoice email — skip silently
	}

	xmlFiles, err := processor.UnzipInMemory(zipData)
	if err != nil || len(xmlFiles) == 0 {
		return nil
	}

	for _, xmlData := range xmlFiles {
		products, err := processor.ParseInvoiceXMLBytes(xmlData)
		if err != nil || len(products) == 0 {
			continue
		}

		cufe := extractCUFE(xmlData)
		exists, _ := g.db.ExisteFacturaCompra(messageID, cufe)
		if exists {
			result.Duplicadas++
			return nil
		}

		sample := products[0]
		fechaEmision, _ := time.ParseInLocation("2006-01-02", sample.IssueDate, time.Local)

		factura := FacturaCompra{
			UUID:            uuid.NewString(),
			ProveedorNIT:    sample.SupplierNIT,
			ProveedorNombre: sample.SupplierName,
			ClienteNIT:      sample.CustomerNIT,
			ClienteNombre:   sample.CustomerName,
			NumeroFactura:   sample.InvoiceID,
			CUFE:            cufe,
			FechaEmision:    fechaEmision,
			Moneda:          sample.Currency,
			Total:           sample.TotalInvoice,
			Estado:          "PENDIENTE",
			EmailMessageID:  messageID,
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
			return fmt.Errorf("guardar factura %s: %w", sample.InvoiceID, err)
		}
		result.Nuevas++
	}
	return nil
}

func (g *GmailService) extractZipAttachment(svc *gmail.Service, msg *gmail.Message) ([]byte, error) {
	for _, part := range msg.Payload.Parts {
		if !strings.HasSuffix(strings.ToLower(part.Filename), ".zip") || part.Body == nil {
			continue
		}
		if len(part.Body.Data) > 0 {
			return base64.URLEncoding.DecodeString(part.Body.Data)
		}
		if part.Body.AttachmentId != "" {
			att, err := svc.Users.Messages.Attachments.Get("me", msg.Id, part.Body.AttachmentId).Do()
			if err != nil {
				return nil, err
			}
			return base64.URLEncoding.DecodeString(att.Data)
		}
	}
	return nil, fmt.Errorf("no ZIP attachment")
}

// extractCUFE scans raw XML bytes for the CUFE/UUID value used for deduplication.
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
func (g *GmailService) ObtenerFacturasCompra(page, pageSize int, busqueda string) (FacturasCompraResponse, error) {
	return g.db.ObtenerFacturasCompraPaginado(page, pageSize, busqueda)
}

// ObtenerDetalleFacturaCompra returns one purchase invoice with all line items.
func (g *GmailService) ObtenerDetalleFacturaCompra(facturaUUID string) (FacturaCompra, error) {
	return g.db.ObtenerDetalleFacturaCompra(facturaUUID)
}

// ActualizarEstadoFacturaCompra updates the processing estado of a purchase invoice.
func (g *GmailService) ActualizarEstadoFacturaCompra(facturaUUID, estado string) error {
	return g.db.ActualizarEstadoFacturaCompra(facturaUUID, estado)
}
