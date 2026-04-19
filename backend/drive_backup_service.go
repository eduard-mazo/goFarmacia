// drive_backup_service.go — Backups automáticos de la BD hacia Google Drive.
//
// ═══════════════════════════════════════════════════════════════════════════════
// DESCRIPCIÓN GENERAL
// ═══════════════════════════════════════════════════════════════════════════════
//
//   Ejecuta pg_dump, comprime con gzip y sube el archivo a una carpeta
//   "goFarmacia Backups" en el Google Drive del usuario autenticado.
//   El backup automático se dispara una vez al día a las 19:00 (hora local).
//   Retención circular: se conservan solo los últimos 5 backups.
//
// ═══════════════════════════════════════════════════════════════════════════════
// AUTENTICACIÓN OAUTH2
// ═══════════════════════════════════════════════════════════════════════════════
//
//   Reutiliza el mismo credentials.json que el servicio DIAN/Bancolombia.
//   Token guardado en drive_token.json (cuenta Drive puede diferir de Gmail).
//   Puerto de callback: 8096  |  Ruta: /drive/oauth2/callback
//   Scope: DriveFileScope (solo archivos creados por la app).
//
// ═══════════════════════════════════════════════════════════════════════════════
// MÉTODOS PÚBLICOS EXPUESTOS A WAILS
// ═══════════════════════════════════════════════════════════════════════════════
//
//   EstadoAuthDrive()                     → DriveAuthStatus
//   IniciarOAuth2Drive()                  → (url string, err error)
//   RevocarAuthDrive()                    → error
//   EjecutarBackupAhora()                 → (DriveBackupResult, error)
//   ListarBackups()                       → ([]DriveBackupFile, error)
//   EliminarBackup(fileID string)         → error
//   RestaurarBackup(fileID string)        → error
//   GetAutoBackup()                       → DriveAutoBackupState
//   SetAutoBackup(enabled bool)
//
// ═══════════════════════════════════════════════════════════════════════════════
// EVENTOS WAILS EMITIDOS
// ═══════════════════════════════════════════════════════════════════════════════
//
//   "drive:backup:done"  → DriveBackupResult  (backup automático completado)
//   "drive:auth:ok"      → nil                 (autenticación completada)

package backend

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	driveCallbackPort = 8096
	driveCallbackPath = "/drive/oauth2/callback"
	driveBackupFolder = "goFarmacia Backups"
	driveBackupHour   = 19 // daily run at 19:00 local time
	driveBackupKeep   = 5  // circular retention: keep only the N most recent
)

// ── Types exposed to Wails ────────────────────────────────────────────────────

// DriveAuthStatus reports Drive authentication state to the frontend.
type DriveAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	CredPresent   bool   `json:"credPresent"`
	ConfigDir     string `json:"configDir"`
}

// DriveBackupResult summarizes one backup run.
type DriveBackupResult struct {
	FileID    string `json:"fileId"`
	FileName  string `json:"fileName"`
	SizeBytes int64  `json:"sizeBytes"`
	Ts        string `json:"ts"`
	Error     string `json:"error,omitempty"`
}

// DriveBackupFile represents a single backup file stored in Drive.
type DriveBackupFile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
	CreatedAt string `json:"createdAt"`
}

// DriveAutoBackupState reports the auto-backup state to the frontend.
type DriveAutoBackupState struct {
	Enabled    bool   `json:"enabled"`
	NextBackup string `json:"nextBackup"`
	LastBackup string `json:"lastBackup"`
}

// ── Service ───────────────────────────────────────────────────────────────────

// DriveBackupService handles automated Google Drive backups.
type DriveBackupService struct {
	db         *Db
	configDir  string
	mu         sync.Mutex
	timer      *time.Timer
	done       chan struct{}
	autoBackup bool
	lastBackup time.Time
	nextBackup time.Time
}

// NewDriveBackupService creates the service.
func NewDriveBackupService(db *Db) *DriveBackupService {
	home, _ := os.UserHomeDir()
	return &DriveBackupService{
		db:         db,
		configDir:  filepath.Join(home, ".config", "goFarmacia"),
		done:       make(chan struct{}),
		autoBackup: true,
	}
}

// Startup initialises the service.
func (s *DriveBackupService) Startup(_ context.Context) {
	_ = os.MkdirAll(s.configDir, 0o700)
	if s.EstadoAuthDrive().Authenticated {
		s.scheduleNext()
	}
}

// Shutdown stops the background timer.
func (s *DriveBackupService) Shutdown() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.mu.Unlock()
}

// ── Public methods ────────────────────────────────────────────────────────────

// EstadoAuthDrive returns the current Drive authentication state.
func (s *DriveBackupService) EstadoAuthDrive() DriveAuthStatus {
	_, credErr := os.Stat(s.credentialsPath())
	_, tokErr := os.Stat(s.tokenPath())
	return DriveAuthStatus{
		Authenticated: credErr == nil && tokErr == nil,
		CredPresent:   credErr == nil,
		ConfigDir:     s.configDir,
	}
}

// IniciarOAuth2Drive starts the OAuth2 authorization flow for Google Drive.
// Opens the system browser and starts a local callback server on port 8096.
func (s *DriveBackupService) IniciarOAuth2Drive() (string, error) {
	cfg, err := s.loadOAuth2Config()
	if err != nil {
		return "", err
	}
	authURL := cfg.AuthCodeURL("state-drive", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	go s.startCallbackServer(cfg)
	return authURL, nil
}

// RevocarAuthDrive removes the saved Drive token and stops the scheduler.
func (s *DriveBackupService) RevocarAuthDrive() error {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.nextBackup = time.Time{}
	s.mu.Unlock()
	return os.Remove(s.tokenPath())
}

// GetAutoBackup returns the current auto-backup state.
func (s *DriveBackupService) GetAutoBackup() DriveAutoBackupState {
	s.mu.Lock()
	defer s.mu.Unlock()
	nextStr := ""
	if s.autoBackup && !s.nextBackup.IsZero() {
		nextStr = s.nextBackup.Format(time.RFC3339)
	}
	lastStr := ""
	if !s.lastBackup.IsZero() {
		lastStr = s.lastBackup.Format(time.RFC3339)
	}
	return DriveAutoBackupState{
		Enabled:    s.autoBackup,
		NextBackup: nextStr,
		LastBackup: lastStr,
	}
}

// SetAutoBackup enables or disables automatic backups.
func (s *DriveBackupService) SetAutoBackup(enabled bool) {
	s.mu.Lock()
	s.autoBackup = enabled
	if !enabled && s.timer != nil {
		s.timer.Stop()
		s.timer = nil
		s.nextBackup = time.Time{}
	}
	shouldStart := enabled && s.timer == nil
	s.mu.Unlock()
	if shouldStart && s.EstadoAuthDrive().Authenticated {
		s.scheduleNext()
	}
}

// EjecutarBackupAhora runs a backup immediately.
func (s *DriveBackupService) EjecutarBackupAhora() (DriveBackupResult, error) {
	return s.runBackup()
}

// ListarBackups returns backup files from Drive, newest first (up to 50).
func (s *DriveBackupService) ListarBackups() ([]DriveBackupFile, error) {
	svc, err := s.newDriveSvc()
	if err != nil {
		return nil, fmt.Errorf("drive service: %w", err)
	}
	folderID, err := s.getOrCreateFolder(svc)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
	list, err := svc.Files.List().
		Q(q).
		Fields("files(id,name,size,createdTime)").
		OrderBy("createdTime desc").
		PageSize(50).
		Do()
	if err != nil {
		return nil, fmt.Errorf("listar backups: %w", err)
	}
	out := make([]DriveBackupFile, 0, len(list.Files))
	for _, f := range list.Files {
		out = append(out, DriveBackupFile{
			ID:        f.Id,
			Name:      f.Name,
			SizeBytes: f.Size,
			CreatedAt: f.CreatedTime,
		})
	}
	return out, nil
}

// EliminarBackup permanently deletes a backup file from Drive.
func (s *DriveBackupService) EliminarBackup(fileID string) error {
	svc, err := s.newDriveSvc()
	if err != nil {
		return fmt.Errorf("drive service: %w", err)
	}
	return svc.Files.Delete(fileID).Do()
}

// ── Internal ──────────────────────────────────────────────────────────────────

func (s *DriveBackupService) tokenPath() string {
	return filepath.Join(s.configDir, "drive_token.json")
}

func (s *DriveBackupService) credentialsPath() string {
	return filepath.Join(s.configDir, "credentials.json")
}

// nextBackupTime returns the next 19:00 local time strictly after `now`.
// If `now` is before today 19:00, returns today 19:00; otherwise tomorrow 19:00.
func nextBackupTime(now time.Time) time.Time {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), driveBackupHour, 0, 0, 0, loc)
	if now.Before(today) {
		return today
	}
	return today.Add(24 * time.Hour)
}

// scheduleNext arms a one-shot timer for the next 19:00 firing. When it fires
// it runs the backup and re-arms itself for the following day.
func (s *DriveBackupService) scheduleNext() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
	}
	next := nextBackupTime(time.Now())
	s.nextBackup = next
	s.timer = time.AfterFunc(time.Until(next), s.onTimerFire)
	s.mu.Unlock()
}

// onTimerFire runs the scheduled backup and re-arms for the next day.
// Called by time.AfterFunc on its own goroutine.
func (s *DriveBackupService) onTimerFire() {
	select {
	case <-s.done:
		return
	default:
	}
	s.mu.Lock()
	enabled := s.autoBackup
	s.mu.Unlock()
	if !enabled {
		return
	}
	result, err := s.runBackup()
	if err != nil {
		s.db.Log.Errorf("[DriveBackup] Error en backup automático: %v", err)
	} else {
		s.db.Log.Infof("[DriveBackup] Backup OK: %s (%d bytes)", result.FileName, result.SizeBytes)
		EventBus.Emit("drive:backup:done", result)
	}
	s.scheduleNext()
}

func (s *DriveBackupService) runBackup() (DriveBackupResult, error) {
	// Resolve DSN
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		if cfg, ok := LoadDBConfig(); ok {
			dsn = cfg.DSN
		}
	}
	if dsn == "" {
		return DriveBackupResult{}, fmt.Errorf("base de datos no configurada: DSN vacío")
	}

	// Fast reachability probe: a dead host would otherwise hang pg_dump on the
	// kernel TCP timeout (~2 min on Linux). Dial the host:port with a 3s budget
	// and fail immediately if it can't be reached.
	if hostPort := extractHostPort(dsn); hostPort != "" {
		conn, err := net.DialTimeout("tcp", hostPort, 3*time.Second)
		if err != nil {
			return DriveBackupResult{}, fmt.Errorf("servidor no disponible en %s: %w", hostPort, err)
		}
		conn.Close()
	}

	// Create temp file for the compressed dump
	tmpFile, err := os.CreateTemp("", "gofarmacia-backup-*.sql.gz")
	if err != nil {
		return DriveBackupResult{}, fmt.Errorf("crear archivo temporal: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// pg_dump | gzip → tmpFile
	gz := gzip.NewWriter(tmpFile)
	cmd := exec.Command("pg_dump", "--no-password", dsn)
	cmd.Stdout = gz
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		tmpFile.Close()
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return DriveBackupResult{}, fmt.Errorf("pg_dump falló: %w", err)
		}
		// Truncate very long stderr so the UI toast stays readable.
		if len(msg) > 500 {
			msg = msg[:500] + "…"
		}
		return DriveBackupResult{}, fmt.Errorf("pg_dump falló: %w — %s", err, msg)
	}
	if err := gz.Close(); err != nil {
		tmpFile.Close()
		return DriveBackupResult{}, fmt.Errorf("cerrar gzip: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return DriveBackupResult{}, fmt.Errorf("cerrar archivo temporal: %w", err)
	}

	info, _ := os.Stat(tmpPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}

	// Open for upload
	f, err := os.Open(tmpPath)
	if err != nil {
		return DriveBackupResult{}, fmt.Errorf("abrir archivo: %w", err)
	}
	defer f.Close()

	// Upload to Drive
	svc, err := s.newDriveSvc()
	if err != nil {
		return DriveBackupResult{}, fmt.Errorf("drive service: %w", err)
	}
	folderID, err := s.getOrCreateFolder(svc)
	if err != nil {
		return DriveBackupResult{}, err
	}

	ts := time.Now()
	fileName := fmt.Sprintf("goFarmacia_%s.sql.gz", ts.Format("2006-01-02_15-04-05"))
	created, err := svc.Files.Create(&drive.File{
		Name:    fileName,
		Parents: []string{folderID},
	}).Media(f).Do()
	if err != nil {
		return DriveBackupResult{}, fmt.Errorf("subir a Drive: %w", err)
	}

	s.mu.Lock()
	s.lastBackup = ts
	s.mu.Unlock()

	// Circular retention: keep only the N most recent backups. Failures here
	// are logged but don't fail the backup — the upload already succeeded.
	if err := s.pruneOldBackups(svc, folderID, driveBackupKeep); err != nil {
		s.db.Log.Warnf("[DriveBackup] Prune falló (no crítico): %v", err)
	}

	return DriveBackupResult{
		FileID:    created.Id,
		FileName:  fileName,
		SizeBytes: size,
		Ts:        ts.Format(time.RFC3339),
	}, nil
}

// pruneOldBackups deletes backups beyond the N most recent in the folder.
func (s *DriveBackupService) pruneOldBackups(svc *drive.Service, folderID string, keep int) error {
	q := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
	list, err := svc.Files.List().
		Q(q).
		Fields("files(id,name,createdTime)").
		OrderBy("createdTime desc").
		PageSize(100).
		Do()
	if err != nil {
		return fmt.Errorf("listar para prune: %w", err)
	}
	if len(list.Files) <= keep {
		return nil
	}
	for _, f := range list.Files[keep:] {
		if err := svc.Files.Delete(f.Id).Do(); err != nil {
			s.db.Log.Warnf("[DriveBackup] No se pudo eliminar %s: %v", f.Name, err)
			continue
		}
		s.db.Log.Infof("[DriveBackup] Prune: eliminado %s", f.Name)
	}
	return nil
}

// RestaurarBackup downloads the given backup file from Drive, decompresses it,
// and applies it to the configured Postgres database.
//
// Strategy (b): before piping the dump in, reset the target schema with
// `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` so the restore can
// succeed against a DB that already contains objects.
func (s *DriveBackupService) RestaurarBackup(fileID string) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		if cfg, ok := LoadDBConfig(); ok {
			dsn = cfg.DSN
		}
	}
	if dsn == "" {
		return fmt.Errorf("base de datos no configurada: DSN vacío")
	}

	// Fast reachability probe (same as runBackup).
	if hostPort := extractHostPort(dsn); hostPort != "" {
		conn, err := net.DialTimeout("tcp", hostPort, 3*time.Second)
		if err != nil {
			return fmt.Errorf("servidor no disponible en %s: %w", hostPort, err)
		}
		conn.Close()
	}

	// Download the backup from Drive to a temp gzip file.
	svc, err := s.newDriveSvc()
	if err != nil {
		return fmt.Errorf("drive service: %w", err)
	}
	resp, err := svc.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("descargar backup: %w", err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "gofarmacia-restore-*.sql.gz")
	if err != nil {
		return fmt.Errorf("crear archivo temporal: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("escribir backup: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("cerrar archivo temporal: %w", err)
	}

	// Reset schema, then stream decompressed dump into psql.
	f, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("abrir backup: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	resetSQL := "DROP SCHEMA public CASCADE; CREATE SCHEMA public;\n"

	cmd := exec.Command("psql", "--no-password", "--set", "ON_ERROR_STOP=1", "-d", dsn)
	cmd.Stdin = io.MultiReader(strings.NewReader(resetSQL), gz)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 500 {
			msg = msg[:500] + "…"
		}
		if msg == "" {
			return fmt.Errorf("psql falló: %w", err)
		}
		return fmt.Errorf("psql falló: %w — %s", err, msg)
	}
	s.db.Log.Infof("[DriveBackup] Restore OK desde fileID=%s", fileID)
	EventBus.Emit("drive:restore:done", map[string]string{"fileId": fileID})
	return nil
}

func (s *DriveBackupService) getOrCreateFolder(svc *drive.Service) (string, error) {
	q := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and trashed=false", driveBackupFolder)
	list, err := svc.Files.List().Q(q).Fields("files(id)").PageSize(1).Do()
	if err != nil {
		return "", fmt.Errorf("buscar carpeta Drive: %w", err)
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}
	created, err := svc.Files.Create(&drive.File{
		Name:     driveBackupFolder,
		MimeType: "application/vnd.google-apps.folder",
	}).Do()
	if err != nil {
		return "", fmt.Errorf("crear carpeta Drive: %w", err)
	}
	return created.Id, nil
}

func (s *DriveBackupService) loadOAuth2Config() (*oauth2.Config, error) {
	b, err := os.ReadFile(s.credentialsPath())
	if err != nil {
		return nil, fmt.Errorf("credentials.json no encontrado en %s: %w", s.configDir, err)
	}
	cfg, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("credentials.json inválido: %w", err)
	}
	cfg.RedirectURL = fmt.Sprintf("http://localhost:%d%s", driveCallbackPort, driveCallbackPath)
	return cfg, nil
}

func (s *DriveBackupService) saveToken(tok *oauth2.Token) error {
	f, err := os.OpenFile(s.tokenPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

func (s *DriveBackupService) newDriveSvc() (*drive.Service, error) {
	cfg, err := s.loadOAuth2Config()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(s.tokenPath())
	if err != nil {
		return nil, fmt.Errorf("drive_token.json no encontrado: %w", err)
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("token inválido: %w", err)
	}
	// savingTokenSource persists refreshed tokens to disk (prevents 401 after restart)
	src := &savingTokenSource{
		inner: cfg.TokenSource(context.Background(), tok),
		last:  tok.AccessToken,
		save:  s.saveToken,
	}
	client := oauth2.NewClient(context.Background(), src)
	return drive.NewService(context.Background(), option.WithHTTPClient(client))
}

func (s *DriveBackupService) startCallbackServer(cfg *oauth2.Config) {
	mux := http.NewServeMux()
	srv := &http.Server{Addr: fmt.Sprintf(":%d", driveCallbackPort), Handler: mux}
	mux.HandleFunc(driveCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Código de autorización no recibido", http.StatusBadRequest)
			return
		}
		tok, err := cfg.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "Error al obtener token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.saveToken(tok); err != nil {
			http.Error(w, "Error al guardar token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, `<html><body style="font-family:sans-serif;padding:2rem;max-width:480px;margin:2rem auto">
<h2 style="color:#16a34a">✅ Google Drive conectado</h2>
<p>Los backups automáticos de goFarmacia están activos.</p>
<p style="color:#6b7280;font-size:0.875rem">Puedes cerrar esta ventana y regresar a la aplicación.</p>
</body></html>`)
		go func() {
			time.Sleep(2 * time.Second)
			_ = srv.Shutdown(context.Background())
			if s.EstadoAuthDrive().Authenticated {
				s.scheduleNext()
				EventBus.Emit("drive:auth:ok", nil)
			}
		}()
	})
	_ = srv.ListenAndServe()
}

// extractHostPort pulls the host:port from a Postgres DSN for a pre-flight
// TCP probe. Supports both URL form (postgres://…) and libpq key=value form.
// Returns "" for local/unix-socket DSNs we shouldn't probe or on parse failure
// (caller proceeds directly to pg_dump in that case).
func extractHostPort(dsn string) string {
	host, port := "", "5432"
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil || u.Host == "" {
			return ""
		}
		host = u.Hostname()
		if p := u.Port(); p != "" {
			port = p
		}
	} else {
		for _, kv := range strings.Fields(dsn) {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				continue
			}
			switch parts[0] {
			case "host":
				host = parts[1]
			case "port":
				port = parts[1]
			}
		}
	}
	if host == "" || host == "localhost" || strings.HasPrefix(host, "/") {
		return ""
	}
	return net.JoinHostPort(host, port)
}
