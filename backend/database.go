package backend

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

type OperacionStock struct {
	ID              uint      `json:"id"`
	UUID            string    `json:"uuid"`
	ProductoID      uint      `json:"producto_id"`
	TipoOperacion   string    `json:"tipo_operacion"`
	CantidadCambio  int       `json:"cantidad_cambio"`
	StockResultante int       `json:"stock_resultante"`
	VendedorID      uint      `json:"vendedor_id"`
	FacturaID       *uint     `json:"factura_id"`
	Timestamp       time.Time `json:"timestamp" ts_type:"string"`
	Sincronizado    bool      `json:"sincronizado"`
}

type Claims struct {
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	Nombre  string `json:"nombre"`
	Cedula  string `json:"cedula"`
	MFAStep string `json:"mfa_step,omitempty"`
	jwt.RegisteredClaims
}

type AjusteStockRequest struct {
	ProductoID uint `json:"producto_id"`
	NuevoStock int  `json:"nuevo_stock"`
}

type LoginResponse struct {
	MFARequired bool     `json:"mfa_required"`
	Token       string   `json:"token"`
	Vendedor    Vendedor `json:"vendedor"`
}

type MFASetupResponse struct {
	Secret   string `json:"secret"`
	ImageURL string `json:"image_url"`
}

type Vendedor struct {
	ID         uint       `json:"id"`
	CreatedAt  time.Time  `json:"created_at" ts_type:"string"`
	UpdatedAt  time.Time  `json:"updated_at" ts_type:"string"`
	DeletedAt  *time.Time `json:"deleted_at" ts_type:"string"`
	Nombre     string     `json:"Nombre"`
	Apellido   string     `json:"Apellido"`
	Cedula     string     `json:"Cedula"`
	Email      string     `json:"Email"`
	Contrasena string     `json:"Contrasena"`
	MFASecret  string     `json:"-"`
	MFAEnabled bool       `json:"mfa_enabled"`
}

type Cliente struct {
	ID        uint       `json:"id"`
	CreatedAt time.Time  `json:"created_at" ts_type:"string"`
	UpdatedAt time.Time  `json:"updated_at" ts_type:"string"`
	DeletedAt *time.Time `json:"deleted_at" ts_type:"string"`
	Nombre    string     `json:"Nombre"`
	Apellido  string     `json:"Apellido"`
	TipoID    string     `json:"TipoID"`
	NumeroID  string     `json:"NumeroID"`
	Telefono  string     `json:"Telefono"`
	Email     string     `json:"Email"`
	Direccion string     `json:"Direccion"`
}

type Producto struct {
	ID          uint       `json:"id"`
	CreatedAt   time.Time  `json:"created_at" ts_type:"string"`
	UpdatedAt   time.Time  `json:"updated_at" ts_type:"string"`
	DeletedAt   *time.Time `json:"deleted_at" ts_type:"string"`
	Nombre      string     `json:"Nombre"`
	Codigo      string     `json:"Codigo"`
	PrecioVenta float64    `json:"PrecioVenta"`
	Stock       int        `json:"Stock"`
}

type Factura struct {
	ID            uint             `json:"id"`
	CreatedAt     time.Time        `json:"created_at" ts_type:"string"`
	UpdatedAt     time.Time        `json:"updated_at" ts_type:"string"`
	DeletedAt     *time.Time       `json:"deleted_at" ts_type:"string"`
	UUID          string           `json:"uuid"`
	NumeroFactura string           `json:"NumeroFactura"`
	FechaEmision  time.Time        `json:"fecha_emision"  ts_type:"string"`
	VendedorID    uint             `json:"VendedorID"`
	Vendedor      Vendedor         `json:"Vendedor"`
	ClienteID     uint             `json:"ClienteID"`
	Cliente       Cliente          `json:"Cliente"`
	Subtotal      float64          `json:"Subtotal"`
	IVA           float64          `json:"IVA"`
	Total         float64          `json:"Total"`
	Estado        string           `json:"Estado"`
	MetodoPago    string           `json:"MetodoPago"`
	Detalles      []DetalleFactura `json:"Detalles"`
}

type DetalleFactura struct {
	ID             uint       `json:"id"`
	CreatedAt      time.Time  `json:"created_at" ts_type:"string"`
	UpdatedAt      time.Time  `json:"updated_at" ts_type:"string"`
	DeletedAt      *time.Time `json:"deleted_at" ts_type:"string"`
	UUID           string     `json:"uuid"`
	FacturaID      uint       `json:"FacturaID"`
	ProductoID     uint       `json:"ProductoID"`
	Producto       Producto   `json:"Producto"`
	Cantidad       int        `json:"Cantidad"`
	PrecioUnitario float64    `json:"PrecioUnitario"`
	PrecioTotal    float64    `json:"PrecioTotal"`
}

type Proveedor struct {
	ID        uint       `json:"id"`
	CreatedAt time.Time  `json:"created_at" ts_type:"string"`
	UpdatedAt time.Time  `json:"updated_at" ts_type:"string"`
	DeletedAt *time.Time `json:"deleted_at" ts_type:"string"`
	Nombre    string     `json:"Nombre"`
	Telefono  string     `json:"Telefono"`
	Email     string     `json:"Email"`
}

type Compra struct {
	ID            uint            `json:"id"`
	CreatedAt     time.Time       `json:"created_at" ts_type:"string"`
	UpdatedAt     time.Time       `json:"updated_at" ts_type:"string"`
	DeletedAt     *time.Time      `json:"deleted_at" ts_type:"string"`
	Fecha         time.Time       `json:"Fecha" ts_type:"string"`
	ProveedorID   uint            `json:"ProveedorID"`
	Proveedor     Proveedor       `json:"Proveedor"`
	FacturaNumero string          `json:"FacturaNumero"`
	Total         float64         `json:"Total"`
	Detalles      []DetalleCompra `json:"Detalles"`
}

type DetalleCompra struct {
	ID                   uint     `json:"id"`
	CompraID             uint     `json:"CompraID"`
	ProductoID           uint     `json:"ProductoID"`
	Producto             Producto `json:"Producto"`
	Cantidad             int      `json:"Cantidad"`
	PrecioCompraUnitario float64  `json:"PrecioCompraUnitario"`
}

type VentaRequest struct {
	ClienteID  uint            `json:"ClienteID"`
	VendedorID uint            `json:"VendedorID"`
	Productos  []ProductoVenta `json:"Productos"`
	MetodoPago string          `json:"MetodoPago"`
}

type ProductoVenta struct {
	ID             uint    `json:"ID"`
	Cantidad       int     `json:"Cantidad"`
	PrecioUnitario float64 `json:"PrecioUnitario"`
}

type LoginRequest struct {
	Email      string `json:"Email"`
	Contrasena string `json:"Contrasena"`
}

type CompraRequest struct {
	ProveedorID   uint                 `json:"ProveedorID"`
	FacturaNumero string               `json:"FacturaNumero"`
	Productos     []ProductoCompraInfo `json:"Productos"`
}

type ProductoCompraInfo struct {
	ProductoID           uint    `json:"ProductoID"`
	Cantidad             int     `json:"Cantidad"`
	PrecioCompraUnitario float64 `json:"PrecioCompraUnitario"`
}

type PaginatedResult struct {
	Records      interface{} `json:"Records"`
	TotalRecords int64       `json:"TotalRecords"`
}

type VendedorUpdateRequest struct {
	ID               uint   `json:"ID"`
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

func NewDb() *Db {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	logger.SetLevel(logrus.DebugLevel)
	return &Db{Log: logger}
}

func (d *Db) Startup(ctx context.Context) {
	d.ctx = ctx
	fmt.Println("DB backend starting up...")
	d.initDB()
	d.RealizarSincronizacionInicial()
}

func (d *Db) initDB() {
	var err error
	d.LocalDB, err = d.NewLocalDB()
	if err != nil {
		d.Log.Fatalf("Fallo al conectar la Base de datos local SQLite: %v", err)
	}
	d.Log.Info("Conección a la Base de datos local SQLite establecida.")

	if err := d.AutoMigrate(); err != nil {
		d.Log.Fatalf("Fallo al realizar AutoMigrate en Base de datos local SQLite: %v", err)
	}
	d.Log.Info("Base de datos local SQLite migrada correctamente.")
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
	d.RemoteDB, err = d.NewRemoteDB(os.Getenv("DATABASE_URL"))

	if err != nil {
		d.Log.Warnf("No se pudo conectar a la Base de datos remota PostgreSQL se trabajará OFFLINE: %v", err)
		d.RemoteDB = nil
	} else {
		d.Log.Info("Base de datos remota PostgreSQL conectada exitosamente.")
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

func (d *Db) NewLocalDB() (*sql.DB, error) {
	dsn := "file:farmacia.db?_cache=shared&_journal_mode=WAL&_foreign_keys=1"
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

// AutoMigrate crea/actualiza el esquema de la BD local.
func (d *Db) AutoMigrate() error {
	// Se añade la tabla sync_log para la sincronización inteligente
	schema := `
		CREATE TABLE IF NOT EXISTS sync_log (
			model_name TEXT PRIMARY KEY,
			last_sync_timestamp DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS vendedors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			nombre TEXT,
			apellido TEXT,
			cedula TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			contrasena TEXT,
			mfa_secret TEXT,
			mfa_enabled BOOLEAN DEFAULT false
		);
		CREATE TABLE IF NOT EXISTS clientes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			nombre TEXT UNIQUE NOT NULL,
			apellido TEXT,
			tipo_id TEXT,
			numero_id TEXT UNIQUE NOT NULL,
			telefono TEXT,
			email TEXT,
			direccion TEXT
		);
		CREATE TABLE IF NOT EXISTS proveedors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			nombre TEXT UNIQUE NOT NULL,
			telefono TEXT,
			email TEXT
		);
		CREATE TABLE IF NOT EXISTS productos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			nombre TEXT,
			codigo TEXT UNIQUE NOT NULL,
			precio_venta REAL,
			stock INTEGER
		);
		CREATE TABLE IF NOT EXISTS operacion_stocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid TEXT UNIQUE NOT NULL,
			producto_id INTEGER NOT NULL,
			tipo_operacion TEXT,
			cantidad_cambio INTEGER,
			stock_resultante INTEGER,
			vendedor_id INTEGER,
			factura_id INTEGER,
			timestamp DATETIME,
			sincronizado BOOLEAN DEFAULT false,
			FOREIGN KEY (producto_id) REFERENCES productos (id)
		);
		CREATE TABLE IF NOT EXISTS facturas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			uuid TEXT UNIQUE NOT NULL,
			numero_factura TEXT UNIQUE NOT NULL,
			fecha_emision DATETIME,
			vendedor_id INTEGER,
			cliente_id INTEGER,
			subtotal REAL,
			iva REAL,
			total REAL,
			estado TEXT,
			metodo_pago TEXT,
			FOREIGN KEY (vendedor_id) REFERENCES vendedors (id),
			FOREIGN KEY (cliente_id) REFERENCES clientes (id)
		);
		CREATE TABLE IF NOT EXISTS detalle_facturas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			uuid TEXT UNIQUE NOT NULL,
			factura_id INTEGER,
			producto_id INTEGER,
			cantidad INTEGER,
			precio_unitario REAL,
			precio_total REAL,
			FOREIGN KEY (factura_id) REFERENCES facturas (id),
			FOREIGN KEY (producto_id) REFERENCES productos (id),
			UNIQUE (factura_id, producto_id)
		);
		CREATE TABLE IF NOT EXISTS compras (
			id INTEGER PRIMARY KEY AUTOINCREMENT, fecha DATETIME, proveedor_id INTEGER, factura_numero TEXT,
			total REAL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
			FOREIGN KEY (proveedor_id) REFERENCES proveedors (id)
		);
		CREATE TABLE IF NOT EXISTS detalle_compras (
			id INTEGER PRIMARY KEY AUTOINCREMENT, compra_id INTEGER, producto_id INTEGER, cantidad INTEGER,
			precio_compra_unitario REAL,
			FOREIGN KEY (compra_id) REFERENCES compras (id), FOREIGN KEY (producto_id) REFERENCES productos (id)
		);
	`
	_, err := d.LocalDB.Exec(schema)
	return err
}
