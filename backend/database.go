package backend

import (
	"context"
	"database/sql"
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
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

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
		logger := logrus.New()
		logDir := "logs"
		_ = os.MkdirAll(logDir, 0755)

		timestamp := time.Now().Format("2006-01-02_15-04-05")
		logFile := filepath.Join(logDir, fmt.Sprintf("app_%s.log", timestamp))

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("No se pudo abrir archivo de log: %v\n", err)
		}

		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			ForceColors:     false,
			TimestampFormat: "2006-01-02 15:04:05.000",
		})
		logger.SetLevel(logrus.DebugLevel)

		var writers []io.Writer
		writers = append(writers, file)
		if isConsoleAvailable() {
			writers = append(writers, os.Stdout)
		}
		logger.SetOutput(io.MultiWriter(writers...))

		logger.Info("Logger inicializado correctamente.")
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

func (d *Db) Startup(ctx context.Context) {
	d.ctx = ctx
	d.initDB()
}

func (d *Db) initDB() {
	var err error

	// Cargar variables de entorno
	err = godotenv.Load()
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

	sourceURL := fmt.Sprintf("file://backend/db/migrations/%s", dbType)

	d.Log.Infof("[MIGRATIONS] Iniciando migraciones para '%s' desde '%s'", dbType, sourceURL)

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		d.Log.Errorf("Error al inicializar instancia de migración para '%s': %v", dbType, err)
		return
	}
	defer m.Close()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		d.Log.Errorf("¡¡¡ERROR CRÍTICO al aplicar migración para '%s'!!!: %v", dbType, err)
	} else if err == migrate.ErrNoChange {
		d.Log.Infof("Migración para '%s': No hay cambios que aplicar. Esquema actualizado.", dbType)
	} else {
		d.Log.Infof("Migración para '%s' aplicada exitosamente.", dbType)
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
