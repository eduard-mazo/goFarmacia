package backend

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	UserUUID string `json:"UserUUID"`
	Email    string `json:"Email"`
	Nombre   string `json:"Nombre"`
	Cedula   string `json:"Cedula"`
	Role     string `json:"Role"`
	MFAStep  string `json:"MFAStep,omitempty"`
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

// ==================== DATABASE ====================

type Db struct {
	ctx    context.Context
	DB     *sql.DB // ✅ Una única conexión PostgreSQL
	Log    *logrus.Logger
	jwtKey []byte
}

var (
	dbInstance *Db
	once       sync.Once
)

func GetDbInstance() *Db {
	once.Do(func() {
		// Logger mínimo sin archivo — el archivo se crea en initDB()
		// para evitar log files espurios de las inicializaciones previas de Wails.
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

// baseDir devuelve el directorio del ejecutable en producción,
// o el CWD en modo desarrollo (wails dev corre desde la raíz del proyecto).
func baseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// findEnvFile busca .env en: 1) directorio del exe, 2) CWD.
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
	return candidates[0] // retorna el primero para que el error sea descriptivo
}

func (d *Db) Startup(ctx context.Context) {
	d.ctx = ctx
	d.initDB()
}

func (d *Db) initDB() {
	var err error

	// Crear archivo de log junto al ejecutable (producción) o en CWD (dev)
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

	// Cargar variables de entorno: busca junto al exe, luego en CWD
	envFile := findEnvFile()
	d.Log.Infof("Cargando .env desde: %s", envFile)
	err = godotenv.Load(envFile)
	if err != nil {
		d.Log.Fatalf("Error al cargar archivo .env: %v", err)
	}

	// Cargar JWT Secret
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		d.Log.Fatalf("La variable de entorno JWT_SECRET_KEY no está configurada.")
	}
	d.jwtKey = []byte(secret)
	d.Log.Info("Clave secreta JWT cargada exitosamente.")

	// ✅ Conectar a PostgreSQL local
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		d.Log.Fatalf("DATABASE_URL no está configurada en .env")
	}

	d.DB, err = d.NewPostgresDB(dbURL)
	if err != nil {
		d.Log.Fatalf("Fallo al conectar con PostgreSQL: %v", err)
	}
	d.Log.Info("Conexión a PostgreSQL local establecida exitosamente.")

	// Ejecutar migraciones
	d.runMigrations("postgres", dbURL)
}

func (d *Db) NewPostgresDB(connString string) (*sql.DB, error) {
	if connString == "" {
		return nil, fmt.Errorf("string de conexión no proporcionado")
	}

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	// Configurar pool de conexiones
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	// Verificar conexión
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("no se puede hacer ping a PostgreSQL: %w", err)
	}

	return db, nil
}

func (d *Db) Close() {
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

	// Si la BD quedó en estado "dirty" (migración previa falló a mitad),
	// forzamos la versión actual y reintentamos para recuperarnos automáticamente.
	if err != nil && err != migrate.ErrNoChange {
		var dirtyErr migrate.ErrDirty
		if errors.As(err, &dirtyErr) {
			d.Log.Warnf("[MIGRATIONS] BD sucia en versión %d — forzando versión y reintentando...", dirtyErr.Version)
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
