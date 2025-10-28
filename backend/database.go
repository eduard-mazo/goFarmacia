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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

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

type Compra struct {
	CreatedAt     time.Time       `json:"CreatedAt" ts_type:"string"`
	UpdatedAt     time.Time       `json:"UpdatedAt" ts_type:"string"`
	DeletedAt     *time.Time      `json:"DeletedAt" ts_type:"string"`
	UUID          string          `json:"uuid"`
	Fecha         time.Time       `json:"Fecha" ts_type:"string"`
	ProveedorUUID string          `json:"proveedor_uuid"`
	Proveedor     Proveedor       `json:"proveedor"`
	FacturaNumero string          `json:"FacturaNumero"`
	Total         float64         `json:"Total"`
	Detalles      []DetalleCompra `json:"Detalles"`
}

type DetalleCompra struct {
	CreatedAt            time.Time  `json:"CreatedAt" ts_type:"string"`
	UpdatedAt            time.Time  `json:"UpdatedAt" ts_type:"string"`
	DeletedAt            *time.Time `json:"DeletedAt" ts_type:"string"`
	UUID                 string     `json:"UUID"`
	CompraUUID           uint       `json:"coCompraUUIDmpra_uuid"`
	ProductoUUID         string     `json:"ProductoUUID"`
	Producto             Producto   `json:"Producto"`
	Cantidad             int        `json:"Cantidad"`
	PrecioCompraUnitario float64    `json:"PrecioCompraUnitario"`
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

type LoginRequest struct {
	Email      string `json:"Email"`
	Contrasena string `json:"Contrasena"`
}

type CompraRequest struct {
	ProveedorUUID string               `json:"ProveedorUUID"`
	FacturaNumero string               `json:"FacturaNumero"`
	Productos     []ProductoCompraInfo `json:"Productos"`
}

type ProductoCompraInfo struct {
	ProductoUUID         string  `json:"ProductoUUID"`
	Cantidad             int     `json:"Cantidad"`
	PrecioCompraUnitario float64 `json:"PrecioCompraUnitario"`
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

type Db struct {
	ctx       context.Context
	LocalDB   *sql.DB
	RemoteDB  *pgxpool.Pool
	Log       *logrus.Logger
	syncMutex sync.Mutex
	jwtKey    []byte
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
	d.RealizarSincronizacionInicial()
}

func (d *Db) initDB() {
	var err error

	localDBPath := "farmacia.db"
	localDSN := fmt.Sprintf("file:%s?_cache=shared&_journal_mode=WAL&_foreign_keys=1", localDBPath)

	d.LocalDB, err = d.NewLocalDB(localDSN)
	if err != nil {
		d.Log.Fatalf("Fallo al conectar la Base de datos local SQLite: %v", err)
	}
	d.Log.Info("Conección a la Base de datos local SQLite establecida.")

	d.runMigrations("sqlite3", localDBPath)

	err = godotenv.Load()
	if err != nil {
		d.Log.Fatalf("Error al cargar archivo .env: %v", err)
	}
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		d.Log.Fatalf("La variable de entorno JWT_SECRET_KEY no está configurada.")
	}
	d.jwtKey = []byte(secret)
	d.Log.Info("Clave secreta JWT cargada exitosamente.")

	remoteDSN := os.Getenv("DATABASE_URL")
	if remoteDSN != "" {
		d.RemoteDB, err = d.NewRemoteDB(remoteDSN)
		if err != nil {
			d.Log.Warnf("No se pudo conectar a la Base de datos remota PostgreSQL, se trabajará OFFLINE: %v", err)
			d.RemoteDB = nil
		} else {
			d.Log.Info("Base de datos remota PostgreSQL conectada exitosamente.")
			d.runMigrations("postgres", remoteDSN)
		}
	} else {
		d.Log.Warn("DATABASE_URL no está configurada. Se trabajará OFFLINE.")
	}
}

func (d *Db) isRemoteDBAvailable() bool {
	if d.RemoteDB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(d.ctx, 3*time.Second)
	defer cancel()
	return d.RemoteDB.Ping(ctx) == nil
}

func (d *Db) NewLocalDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.PingContext(d.ctx); err != nil {
		return nil, fmt.Errorf("No se puede hacer ping con Base de datos local SQLite: %w", err)
	}
	return db, nil
}

func (d *Db) NewRemoteDB(connString string) (*pgxpool.Pool, error) {
	if connString == "" {
		return nil, fmt.Errorf("String de conexión a Base de datos remota no proporcionada")
	}

	pool, err := pgxpool.New(d.ctx, connString)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(d.ctx); err != nil {
		return nil, fmt.Errorf("No se puede hacer ping con Base de datos remota PostgreSQL: %w", err)
	}

	return pool, nil
}

func (d *Db) Close() {
	if d.LocalDB != nil {
		d.LocalDB.Close()
	}
	if d.RemoteDB != nil {
		d.RemoteDB.Close()
	}
}

func (d *Db) runMigrations(dbType string, dsn string) {
	if dsn == "" {
		d.Log.Warnf("No hay DSN para la migración de '%s', omitiendo.", dbType)
		return
	}

	sourceURL := fmt.Sprintf("file://backend/db/migrations/%s", dbType)
	var databaseURL string
	if dbType == "sqlite3" {
		databaseURL = "sqlite3://" + dsn
	} else {
		databaseURL = dsn
	}

	d.Log.Infof("[MIGRATIONS] Iniciando migraciones para '%s' desde '%s'", dbType, sourceURL)

	// ✅ Protegemos con timeout si es remoto (evita bloqueos)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		m, err := migrate.New(sourceURL, databaseURL)
		if err != nil {
			done <- fmt.Errorf("Error al inicializar instancia de migración para '%s': %v", dbType, err)
			return
		}

		err = m.Up()
		if err != nil && err != migrate.ErrNoChange {
			done <- fmt.Errorf("¡¡¡ERROR CRÍTICO al aplicar migración para '%s'!!!: %v", dbType, err)
		} else if err == migrate.ErrNoChange {
			d.Log.Infof("Migración para '%s': No hay cambios que aplicar. Esquema actualizado.", dbType)
			done <- nil
		} else {
			d.Log.Infof("Migración para '%s' aplicada exitosamente.", dbType)
			done <- nil
		}

		_, _ = m.Close()
	}()

	select {
	case <-ctx.Done():
		d.Log.Errorf("Timeout al ejecutar migraciones para '%s' (más de 10s). Se omite.", dbType)
	case err := <-done:
		if err != nil {
			d.Log.Error(err)
		}
	}

	d.Log.Infof("[MIGRATIONS] Finalizadas migraciones para '%s'", dbType)
}
