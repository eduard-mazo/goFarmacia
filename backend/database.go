package backend

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed db/migrations
var migrationsFS embed.FS

// ==================== STRUCTS ====================

type OperacionStock struct {
	UUID            string    `json:"UUID"`
	ProductoUUID    string    `json:"ProductoUUID"`
	TipoOperacion   string    `json:"TipoOperacion"`
	CantidadCambio  int       `json:"CantidadCambio"`
	StockResultante int       `json:"StockResultante"`
	VendedorUUID    string    `json:"VendedorUUID"`
	FacturaUUID     *string   `json:"FacturaUUID"`
	Timestamp       time.Time `json:"Timestamp" ts_type:"string"`
	Sincronizado    bool      `json:"Sincronizado"`
}

type Claims struct {
	UserUUID  string `json:"UserUUID"`
	Email     string `json:"Email"`
	Nombre    string `json:"Nombre"`
	Cedula    string `json:"Cedula"`
	Role      string `json:"Role"`
	MFAStep   string `json:"MFAStep,omitempty"`
	SetupMode bool   `json:"SetupMode,omitempty"`
	jwt.RegisteredClaims
}

type AjusteStockRequest struct {
	ProductoUUID string `json:"ProductoUUID"`
	NuevoStock   int    `json:"NuevoStock"`
}

type LoginResponse struct {
	MFARequired bool     `json:"MFARequired"`
	Token       string   `json:"Token"`
	Vendedor    Vendedor `json:"Vendedor"`
}

type MFASetupResponse struct {
	Secret   string `json:"Secret"`
	ImageURL string `json:"ImageURL"`
}

type Vendedor struct {
	CreatedAt  time.Time  `json:"CreatedAt" ts_type:"string"`
	UpdatedAt  time.Time  `json:"UpdatedAt" ts_type:"string"`
	DeletedAt  *time.Time `json:"DeletedAt" ts_type:"string"`
	UUID       string     `json:"UUID"`
	Nombre     string     `json:"Nombre"`
	Apellido   string     `json:"Apellido"`
	Cedula     string     `json:"Cedula"`
	Email      string     `json:"Email"`
	Contrasena string     `json:"Contrasena"`
	MFASecret  string     `json:"-"`
	MFAEnabled bool       `json:"MFAEnabled"`
	Role       string     `json:"Role"`
}

type Cliente struct {
	CreatedAt time.Time  `json:"CreatedAt" ts_type:"string"`
	UpdatedAt time.Time  `json:"UpdatedAt" ts_type:"string"`
	DeletedAt *time.Time `json:"DeletedAt" ts_type:"string"`
	UUID      string     `json:"UUID"`
	Nombre    string     `json:"Nombre"`
	Apellido  string     `json:"Apellido"`
	TipoID    string     `json:"TipoID"`
	NumeroID  string     `json:"NumeroID"`
	Telefono  string     `json:"Telefono"`
	Email     string     `json:"Email"`
	Direccion string     `json:"Direccion"`
}

type Producto struct {
	CreatedAt   time.Time  `json:"CreatedAt" ts_type:"string"`
	UpdatedAt   time.Time  `json:"UpdatedAt" ts_type:"string"`
	DeletedAt   *time.Time `json:"DeletedAt" ts_type:"string"`
	UUID        string     `json:"UUID"`
	Nombre      string     `json:"Nombre"`
	Codigo      string     `json:"Codigo"`
	PrecioVenta float64    `json:"PrecioVenta"`
	Stock       int        `json:"Stock"`
}

type ProductoAjusteRequest struct {
	UUID         string  `json:"UUID"`
	Nombre       string  `json:"Nombre"`
	PrecioVenta  float64 `json:"PrecioVenta"`
	StockDeseado int     `json:"Stock"`
	VendedorUUID string  `json:"VendedorUUID,omitempty"`
}

type NuevoProducto struct {
	UUID         string  `json:"UUID"`
	VendedorUUID string  `json:"VendedorUUID"`
	Nombre       string  `json:"Nombre"`
	Codigo       string  `json:"Codigo"`
	PrecioVenta  float64 `json:"PrecioVenta"`
	Stock        int     `json:"Stock"`
}

type Factura struct {
	CreatedAt     time.Time        `json:"CreatedAt" ts_type:"string"`
	UpdatedAt     time.Time        `json:"UpdatedAt" ts_type:"string"`
	DeletedAt     *time.Time       `json:"DeletedAt" ts_type:"string"`
	UUID          string           `json:"UUID"`
	NumeroFactura string           `json:"NumeroFactura"`
	FechaEmision  time.Time        `json:"FechaEmision"  ts_type:"string"`
	VendedorUUID  string           `json:"VendedorUUID"`
	Vendedor      Vendedor         `json:"Vendedor"`
	ClienteUUID   string           `json:"ClienteUUID"`
	Cliente       Cliente          `json:"Cliente"`
	Subtotal      float64          `json:"Subtotal"`
	IVA           float64          `json:"IVA"`
	Total         float64          `json:"Total"`
	Estado        string           `json:"Estado"`
	MetodoPago    string           `json:"MetodoPago"`
	Detalles      []DetalleFactura `json:"Detalles"`
}

type DetalleFactura struct {
	CreatedAt      time.Time  `json:"CreatedAt" ts_type:"string"`
	UpdatedAt      time.Time  `json:"UpdatedAt" ts_type:"string"`
	DeletedAt      *time.Time `json:"DeletedAt" ts_type:"string"`
	UUID           string     `json:"UUID"`
	FacturaUUID    string     `json:"FacturaUUID"`
	ProductoUUID   string     `json:"ProductoUUID"`
	Producto       Producto   `json:"Producto"`
	Cantidad       int        `json:"Cantidad"`
	PrecioUnitario float64    `json:"PrecioUnitario"`
	PrecioTotal    float64    `json:"PrecioTotal"`
}

type Proveedor struct {
	CreatedAt time.Time  `json:"CreatedAt" ts_type:"string"`
	UpdatedAt time.Time  `json:"UpdatedAt" ts_type:"string"`
	DeletedAt *time.Time `json:"DeletedAt" ts_type:"string"`
	UUID      string     `json:"uuid"`
	Nombre    string     `json:"Nombre"`
	Telefono  string     `json:"Telefono"`
	Email     string     `json:"Email"`
}

type VentaRequest struct {
	ClienteUUID  string          `json:"ClienteUUID"`
	VendedorUUID string          `json:"VendedorUUID"`
	Productos    []ProductoVenta `json:"Productos"`
	MetodoPago   string          `json:"MetodoPago"`
}

type ProductoVenta struct {
	ProductoUUID   string  `json:"ProductoUUID"`
	Cantidad       int     `json:"Cantidad"`
	PrecioUnitario float64 `json:"PrecioUnitario"`
}

type PaginatedResult struct {
	Records      interface{} `json:"Records"`
	TotalRecords int64       `json:"TotalRecords"`
}

type VendedorUpdateRequest struct {
	UUID             string `json:"UUID"`
	Nombre           string `json:"Nombre"`
	Apellido         string `json:"Apellido"`
	Cedula           string `json:"Cedula"`
	Email            string `json:"Email"`
	ContrasenaActual string `json:"ContrasenaActual,omitempty"`
	ContrasenaNueva  string `json:"ContrasenaNueva,omitempty"`
}

type LoginRequest struct {
	Email      string `json:"Email"`
	Contrasena string `json:"Contrasena"`
}

type Compra struct {
	UUID          string          `json:"UUID"`
	Fecha         time.Time       `json:"Fecha" ts_type:"string"`
	ProveedorUUID string          `json:"ProveedorUUID"`
	FacturaNumero string          `json:"FacturaNumero"`
	Total         float64         `json:"Total"`
	Detalles      []DetalleCompra `json:"Detalles"`
}

type DetalleCompra struct {
	UUID                 string  `json:"UUID"`
	CompraUUID           string  `json:"CompraUUID"`
	ProductoUUID         string  `json:"ProductoUUID"`
	Cantidad             int     `json:"Cantidad"`
	PrecioCompraUnitario float64 `json:"PrecioCompraUnitario"`
}

type CompraRequest struct {
	ProveedorUUID string           `json:"ProveedorUUID"`
	FacturaNumero string           `json:"FacturaNumero"`
	Productos     []CompraProducto `json:"Productos"`
}

type CompraProducto struct {
	ProductoUUID         string  `json:"ProductoUUID"`
	Cantidad             int     `json:"Cantidad"`
	PrecioCompraUnitario float64 `json:"PrecioCompraUnitario"`
}

// DBStatusResponse is returned by GetDBStatus.
type DBStatusResponse struct {
	Connected bool   `json:"Connected"`
	SetupMode bool   `json:"SetupMode"`
	Message   string `json:"Message"`
	DSNHint   string `json:"DSNHint"` // Sanitized DSN (no password)
}

// ==================== DATABASE ====================

type Db struct {
	ctx       context.Context
	DB        *sql.DB
	Log       *logrus.Logger
	jwtKey    []byte
	setupMode bool   // true when DB is not configured/reachable
	dbError   string // last connection error message
	mu        sync.RWMutex
}

var (
	dbInstance *Db
	once       sync.Once
)

func GetDbInstance() *Db {
	once.Do(func() {
		logger := logrus.New()
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			ForceColors:     false,
			TimestampFormat: "2006-01-02 15:04:05.000",
		})
		logger.SetLevel(logrus.DebugLevel)
		if isConsoleAvailable() {
			logger.SetOutput(os.Stdout)
		} else {
			logger.SetOutput(io.Discard)
		}
		dbInstance = &Db{Log: logger}
	})
	return dbInstance
}

func isConsoleAvailable() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// baseDir returns the directory of the executable in production,
// or CWD in dev mode (wails dev runs from the project root).
func baseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// findEnvFile searches for .env next to the exe, then in CWD.
func findEnvFile() string {
	candidates := []string{
		filepath.Join(baseDir(), ".env"),
		".env",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func (d *Db) Startup(ctx context.Context) {
	d.ctx = ctx
	d.initDB()
}

func (d *Db) initDB() {
	// ── Logger ──────────────────────────────────────────────────────────────
	logDir := filepath.Join(baseDir(), "logs")
	_ = os.MkdirAll(logDir, 0755)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logDir, fmt.Sprintf("app_%s.log", timestamp))
	if file, ferr := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); ferr == nil {
		var writers []io.Writer
		writers = append(writers, file)
		if isConsoleAvailable() {
			writers = append(writers, os.Stdout)
		}
		d.Log.SetOutput(io.MultiWriter(writers...))
	}
	d.Log.Info("Logger inicializado correctamente.")

	// ── Load .env (non-fatal) ───────────────────────────────────────────────
	envFile := findEnvFile()
	d.Log.Infof("Buscando .env en: %s", envFile)
	if err := godotenv.Load(envFile); err != nil {
		d.Log.Warnf("No se pudo cargar .env: %v — continuando sin él.", err)
	}

	// ── JWT key ─────────────────────────────────────────────────────────────
	// Priority: 1) JWT_SECRET_KEY env var, 2) db_config.json, 3) auto-generate & save
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		if cfg, ok := LoadDBConfig(); ok && cfg.JWTSecret != "" {
			secret = cfg.JWTSecret
			d.Log.Info("JWT key cargada desde db_config.json")
		}
	}
	if secret == "" {
		secret = GenerateSecureKey()
		d.Log.Warn("JWT_SECRET_KEY no configurada — generando clave aleatoria y guardando en db_config.json")
		cfg, _ := LoadDBConfig()
		cfg.JWTSecret = secret
		_ = SaveDBConfig(cfg)
	}
	d.jwtKey = []byte(secret)
	d.Log.Info("Clave JWT cargada exitosamente.")

	// ── DB connection ────────────────────────────────────────────────────────
	// Priority: 1) DATABASE_URL env, 2) db_config.json DSN
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		if cfg, ok := LoadDBConfig(); ok {
			dbURL = cfg.DSN
			d.Log.Infof("Usando DSN de db_config.json")
		}
	}

	if dbURL == "" {
		d.Log.Warn("DATABASE_URL no configurada — activando modo configuración.")
		d.mu.Lock()
		d.setupMode = true
		d.dbError = "Base de datos no configurada. Accede a Configuración para establecer la conexión."
		d.mu.Unlock()
		return
	}

	db, err := d.NewPostgresDB(dbURL)
	if err != nil {
		d.Log.Warnf("No se pudo conectar a PostgreSQL: %v — activando modo configuración.", err)
		d.mu.Lock()
		d.setupMode = true
		d.dbError = err.Error()
		d.mu.Unlock()
		return
	}

	d.mu.Lock()
	d.DB = db
	d.setupMode = false
	d.dbError = ""
	d.mu.Unlock()
	d.Log.Info("Conexión a PostgreSQL establecida exitosamente.")

	d.runMigrations("postgres", dbURL)
}

// localTimeZoneName returns the IANA timezone name of the machine (e.g. "America/Bogota").
func localTimeZoneName() string {
	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		return strings.TrimSpace(string(data))
	}
	if link, err := os.Readlink("/etc/localtime"); err == nil {
		if idx := strings.Index(link, "zoneinfo/"); idx >= 0 {
			return link[idx+9:]
		}
	}
	return ""
}

// injectTimezone appends the machine timezone to a PostgreSQL DSN so that
// TO_CHAR and date comparisons use local time rather than UTC.
func injectTimezone(connString string) string {
	tz := localTimeZoneName()
	if tz == "" {
		return connString
	}
	if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") {
		sep := "?"
		if strings.Contains(connString, "?") {
			sep = "&"
		}
		return connString + sep + "TimeZone=" + url.QueryEscape(tz)
	}
	// key=value format
	return connString + " timezone=" + tz
}

func (d *Db) NewPostgresDB(connString string) (*sql.DB, error) {
	if connString == "" {
		return nil, fmt.Errorf("string de conexión no proporcionado")
	}

	db, err := sql.Open("postgres", injectTimezone(connString))
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("no se puede hacer ping a PostgreSQL: %w", err)
	}

	return db, nil
}

func (d *Db) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.DB != nil {
		d.DB.Close()
		d.Log.Info("Conexión a PostgreSQL cerrada.")
	}
}

func (d *Db) runMigrations(dbType string, dsn string) {
	if dsn == "" {
		d.Log.Warnf("No hay DSN para la migración de '%s', omitiendo.", dbType)
		return
	}

	d.Log.Infof("[MIGRATIONS] Iniciando migraciones para '%s' (embebidas)", dbType)

	src, err := iofs.New(migrationsFS, fmt.Sprintf("db/migrations/%s", dbType))
	if err != nil {
		d.Log.Errorf("[MIGRATIONS] Error al crear fuente iofs para '%s': %v", dbType, err)
		return
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		d.Log.Errorf("Error al inicializar instancia de migración para '%s': %v", dbType, err)
		return
	}
	defer m.Close()

	err = m.Up()

	if err != nil && err != migrate.ErrNoChange {
		var dirtyErr migrate.ErrDirty
		if errors.As(err, &dirtyErr) {
			d.Log.Warnf("[MIGRATIONS] BD sucia en versión %d — forzando y reintentando...", dirtyErr.Version)
			if fErr := m.Force(dirtyErr.Version); fErr != nil {
				d.Log.Errorf("[MIGRATIONS] No se pudo forzar versión %d: %v", dirtyErr.Version, fErr)
				return
			}
			err = m.Up()
		}
	}

	if err != nil && err != migrate.ErrNoChange {
		d.Log.Errorf("[MIGRATIONS] Error al aplicar migración para '%s': %v", dbType, err)
	} else if err == migrate.ErrNoChange {
		d.Log.Infof("[MIGRATIONS] Esquema '%s' actualizado, sin cambios pendientes.", dbType)
	} else {
		d.Log.Infof("[MIGRATIONS] Migración '%s' aplicada exitosamente.", dbType)
	}
}

// ==================== DB STATUS & CONFIGURATION ====================

// IsSetupMode returns true when the app started without a valid DB connection.
func (d *Db) IsSetupMode() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.setupMode
}

// GetDBStatus returns the current database connection status.
func (d *Db) GetDBStatus() DBStatusResponse {
	d.mu.RLock()
	defer d.mu.RUnlock()

	msg := "Conectado y operativo"
	if d.setupMode {
		msg = d.dbError
		if msg == "" {
			msg = "Base de datos no configurada"
		}
	}

	dsnHint := ""
	if cfg, ok := LoadDBConfig(); ok {
		dsnHint = sanitizeDSN(cfg.DSN)
	} else if envDSN := os.Getenv("DATABASE_URL"); envDSN != "" {
		dsnHint = sanitizeDSN(envDSN)
	}

	return DBStatusResponse{
		Connected: !d.setupMode && d.DB != nil,
		SetupMode: d.setupMode,
		Message:   msg,
		DSNHint:   dsnHint,
	}
}

// TestDBConnection attempts to ping the given DSN without saving it.
func (d *Db) TestDBConnection(dsn string) error {
	if dsn == "" {
		return fmt.Errorf("el DSN no puede estar vacío")
	}
	db, err := d.NewPostgresDB(dsn)
	if err != nil {
		// If DB doesn't exist, still consider it a "reachable server"
		if isDatabaseNotExistsErr(err) {
			return fmt.Errorf("servidor alcanzable, pero la base de datos no existe (se creará automáticamente al guardar)")
		}
		return err
	}
	db.Close()
	return nil
}

// ConfigurarDB saves the DSN, creates the database if needed, connects, and runs migrations.
// After this call the app exits setup mode.
func (d *Db) ConfigurarDB(dsn string) error {
	if dsn == "" {
		return fmt.Errorf("el DSN no puede estar vacío")
	}

	// Try direct connection first
	db, err := d.NewPostgresDB(dsn)
	if err != nil {
		if isDatabaseNotExistsErr(err) {
			d.Log.Infof("[ConfigurarDB] Base de datos no existe — intentando crearla...")
			if cErr := d.createDatabaseIfNeeded(dsn); cErr != nil {
				return fmt.Errorf("no se pudo crear la base de datos: %w", cErr)
			}
			db, err = d.NewPostgresDB(dsn)
			if err != nil {
				return fmt.Errorf("error al reconectar después de crear la base de datos: %w", err)
			}
		} else {
			return err
		}
	}

	// Load current config to preserve JWTSecret
	cfg, _ := LoadDBConfig()
	cfg.DSN = dsn
	if cfg.JWTSecret == "" {
		// Persist the in-memory JWT key so it survives restart
		cfg.JWTSecret = string(d.jwtKey)
	}
	if err := SaveDBConfig(cfg); err != nil {
		db.Close()
		return fmt.Errorf("error al guardar configuración: %w", err)
	}

	// Swap connection
	d.mu.Lock()
	if d.DB != nil {
		d.DB.Close()
	}
	d.DB = db
	d.setupMode = false
	d.dbError = ""
	d.mu.Unlock()

	d.Log.Infof("[ConfigurarDB] Base de datos configurada y conectada exitosamente.")
	d.runMigrations("postgres", dsn)
	return nil
}

// ==================== DSN HELPERS ====================

func isDatabaseNotExistsErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") ||
		(strings.Contains(msg, "database") && strings.Contains(msg, "not exist"))
}

// switchToSystemDB replaces the dbname in the DSN with "postgres" so we can
// connect to the server to create the target database.
// Returns (originalDBName, newDSN, error).
func switchToSystemDB(dsn string) (string, string, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", "", err
		}
		dbName := strings.TrimPrefix(u.Path, "/")
		if dbName == "" {
			return "", "", fmt.Errorf("no se encontró el nombre de la base de datos en el DSN")
		}
		u.Path = "/postgres"
		return dbName, u.String(), nil
	}
	// Key=value format
	parts := strings.Fields(dsn)
	var dbName string
	newParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.HasPrefix(p, "dbname=") {
			dbName = strings.TrimPrefix(p, "dbname=")
			newParts = append(newParts, "dbname=postgres")
		} else {
			newParts = append(newParts, p)
		}
	}
	if dbName == "" {
		return "", "", fmt.Errorf("no se encontró 'dbname' en el DSN")
	}
	return dbName, strings.Join(newParts, " "), nil
}

func (d *Db) createDatabaseIfNeeded(dsn string) error {
	dbName, systemDSN, err := switchToSystemDB(dsn)
	if err != nil {
		return err
	}

	sysDB, err := sql.Open("postgres", systemDSN)
	if err != nil {
		return err
	}
	defer sysDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sysDB.PingContext(ctx); err != nil {
		return fmt.Errorf("no se puede conectar al servidor PostgreSQL: %w", err)
	}

	var exists bool
	err = sysDB.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil // DB already exists
	}

	// CREATE DATABASE does not support parameters, use Sprintf with quoted identifier
	_, err = sysDB.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	if err != nil {
		return fmt.Errorf("error al crear la base de datos '%s': %w", dbName, err)
	}
	d.Log.Infof("[ConfigurarDB] Base de datos '%s' creada exitosamente.", dbName)
	return nil
}

// sanitizeDSN removes the password from a DSN for safe display.
func sanitizeDSN(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return dsn
		}
		if u.User != nil {
			u.User = url.User(u.User.Username())
		}
		return u.String()
	}
	// Key=value format
	parts := strings.Fields(dsn)
	newParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.HasPrefix(p, "password=") {
			newParts = append(newParts, "password=***")
		} else {
			newParts = append(newParts, p)
		}
	}
	return strings.Join(newParts, " ")
}

// ==================== HELPER METHODS ====================

func (d *Db) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return d.DB.BeginTx(ctx, nil)
}

func (d *Db) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.DB.QueryRowContext(ctx, query, args...)
}

func (d *Db) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.DB.QueryContext(ctx, query, args...)
}

func (d *Db) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.DB.ExecContext(ctx, query, args...)
}

// NewTestDb creates a Db instance for testing with an existing *sql.DB connection.
func NewTestDb(db *sql.DB, ctx context.Context, log *logrus.Logger) *Db {
	return &Db{
		DB:     db,
		ctx:    ctx,
		Log:    log,
		jwtKey: []byte("test-secret-key"),
	}
}
