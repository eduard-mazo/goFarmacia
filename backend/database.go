package backend

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

// --- DATABASE STRUCTS (remain the same) ---

type Vendedor struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time      `json:"created_at" wails:"ts.type=string"`
	UpdatedAt  time.Time      `json:"updated_at" wails:"ts.type=string"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Nombre     string         `json:"Nombre"`
	Apellido   string         `json:"Apellido"`
	Cedula     string         `gorm:"unique" json:"Cedula"`
	Email      string         `gorm:"unique" json:"Email"`
	Contrasena string         `json:"Contrasena"`
}

type Cliente struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at" wails:"ts.type=string"`
	UpdatedAt time.Time      `json:"updated_at" wails:"ts.type=string"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Nombre    string         `json:"Nombre"`
	Apellido  string         `json:"Apellido"`
	TipoID    string         `json:"TipoID"`
	NumeroID  string         `gorm:"unique" json:"NumeroID"`
	Telefono  string         `json:"Telefono"`
	Email     string         `json:"Email"`
	Direccion string         `json:"Direccion"`
}

type Producto struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at" wails:"ts.type=string"`
	UpdatedAt   time.Time      `json:"updated_at" wails:"ts.type=string"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Nombre      string         `json:"Nombre"`
	Codigo      string         `gorm:"unique" json:"Codigo"`
	PrecioVenta float64        `json:"PrecioVenta"`
	Stock       int            `json:"Stock"`
}

type Factura struct {
	ID            uint             `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time        `json:"created_at" wails:"ts.type=string"`
	UpdatedAt     time.Time        `json:"updated_at" wails:"ts.type=string"`
	DeletedAt     gorm.DeletedAt   `gorm:"index" json:"-"`
	NumeroFactura string           `gorm:"unique" json:"NumeroFactura"`
	FechaEmision  time.Time        `json:"fecha_emision" wails:"ts.type=string"`
	VendedorID    uint             `json:"VendedorID"`
	Vendedor      Vendedor         `gorm:"foreignKey:VendedorID" json:"Vendedor"`
	ClienteID     uint             `json:"ClienteID"`
	Cliente       Cliente          `gorm:"foreignKey:ClienteID" json:"Cliente"`
	Subtotal      float64          `json:"Subtotal"`
	IVA           float64          `json:"IVA"`
	Total         float64          `json:"Total"`
	Estado        string           `json:"Estado"`
	MetodoPago    string           `json:"MetodoPago"`
	Detalles      []DetalleFactura `gorm:"foreignKey:FacturaID" json:"Detalles"`
}

type DetalleFactura struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time      `json:"created_at" wails:"ts.type=string"`
	UpdatedAt      time.Time      `json:"updated_at" wails:"ts.type=string"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	FacturaID      uint           `json:"FacturaID"`
	ProductoID     uint           `json:"ProductoID"`
	Producto       Producto       `json:"Producto"`
	Cantidad       int            `json:"Cantidad"`
	PrecioUnitario float64        `json:"PrecioUnitario"`
	PrecioTotal    float64        `json:"PrecioTotal"`
}

type Proveedor struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at" wails:"ts.type=string"`
	UpdatedAt time.Time      `json:"updated_at" wails:"ts.type=string"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Nombre    string         `gorm:"unique" json:"Nombre"`
	Telefono  string         `json:"Telefono"`
	Email     string         `json:"Email"`
}

type Compra struct {
	ID            uint            `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time       `json:"created_at" wails:"ts.type=string"`
	UpdatedAt     time.Time       `json:"updated_at" wails:"ts.type=string"`
	DeletedAt     gorm.DeletedAt  `gorm:"index" json:"-"`
	Fecha         time.Time       `json:"Fecha" wails:"ts.type=string"`
	ProveedorID   uint            `json:"ProveedorID"`
	Proveedor     Proveedor       `json:"Proveedor"`
	FacturaNumero string          `json:"FacturaNumero"`
	Total         float64         `json:"Total"`
	Detalles      []DetalleCompra `gorm:"foreignKey:CompraID" json:"Detalles"`
}

type DetalleCompra struct {
	ID                   uint     `gorm:"primaryKey" json:"id"`
	CompraID             uint     `json:"CompraID"`
	ProductoID           uint     `json:"ProductoID"`
	Producto             Producto `json:"Producto"`
	Cantidad             int      `json:"Cantidad"`
	PrecioCompraUnitario float64  `json:"PrecioCompraUnitario"`
}

// --- REQUEST/RESPONSE STRUCTS (remain the same) ---

type VentaRequest struct {
	ClienteID  uint            `json:"ClienteID"`
	VendedorID uint            `json:"VendedorID"`
	Productos  []ProductoVenta `json:"Productos"`
	MetodoPago string          `json:"MetodoPago"`
}

type ProductoVenta struct {
	ID       uint `json:"ID"`
	Cantidad int  `json:"Cantidad"`
}

type LoginRequest struct {
	Cedula     string `json:"Cedula"`
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

// --- DATABASE LOGIC ---

// Db struct now holds connections for both local and remote databases
type Db struct {
	ctx      context.Context
	LocalDB  *gorm.DB // For SQLite
	RemoteDB *gorm.DB // For Supabase (PostgreSQL)
	Log      *logrus.Logger
}

func NewDb() *Db {
	// Configura el logger aquí
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	logger.SetLevel(logrus.DebugLevel) // Muestra todo durante el desarrollo

	// Retorna la instancia de Db con el logger ya inicializado
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

	// --- 1. Initialize Local SQLite Database ---
	// Silenciar los logs de GORM para la base de datos local
	d.LocalDB, err = gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:farmacia.db"}, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to local SQLite database: %v", err)
	}
	log.Println("✅ Local SQLite database connected successfully.")

	// List of all models to migrate
	models := []interface{}{&Vendedor{}, &Cliente{}, &Producto{}, &Factura{}, &DetalleFactura{}, &Proveedor{}, &Compra{}, &DetalleCompra{}}

	// Auto-migrate schema for the local database
	err = d.LocalDB.AutoMigrate(models...)
	if err != nil {
		log.Fatalf("FATAL: Failed to migrate local database schema: %v", err)
	}
	log.Println("✅ Local database schema migrated successfully.")

	// --- 2. Initialize Remote Supabase Database ---
	// Carga las variables de entorno del archivo .env
	// Esto debe ser lo primero que hagas en la función main
	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	d.RemoteDB, err = gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		log.Printf("WARNING: Failed to connect to remote Supabase database. App will run in offline mode. Error: %v", err)
		// We don't use log.Fatalf here, so the app can start in offline mode.
		d.RemoteDB = nil // Ensure RemoteDB is nil if connection fails
	} else {
		log.Println("✅ Remote Supabase database connected successfully.")
	}
}

// Check if the remote database is available
func (d *Db) isRemoteDBAvailable() bool {
	return d.RemoteDB != nil
}

// DeepResetDatabases realiza un reseteo completo y destructivo de ambas bases de datos.
// ¡ADVERTENCIA! ESTA ACCIÓN BORRARÁ TODOS LOS DATOS PERMANENTEMENTE.
func (d *Db) DeepResetDatabases() error {
	d.Log.Warn("¡INICIANDO DEEP RESET! Todos los datos serán eliminados.")

	// Lista de modelos en orden inverso de dependencia para un borrado seguro.
	// Los detalles se borran antes que las tablas maestras (Factura, Compra).
	modelsToReset := []interface{}{
		&DetalleFactura{}, &DetalleCompra{}, &Factura{}, &Compra{},
		&Vendedor{}, &Cliente{}, &Producto{}, &Proveedor{},
	}

	// --- 1. Resetear Base de Datos Local (SQLite) ---
	d.Log.Info("Reseteando la base de datos local...")
	// Eliminar todas las tablas.
	err := d.LocalDB.Migrator().DropTable(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al eliminar las tablas locales: %v", err)
		return fmt.Errorf("error al eliminar las tablas locales: %w", err)
	}
	d.Log.Info("Tablas locales eliminadas con éxito.")

	// Volver a crear el esquema.
	err = d.LocalDB.AutoMigrate(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al recrear el esquema local: %v", err)
		return fmt.Errorf("error al recrear el esquema local: %w", err)
	}
	d.Log.Info("✅ Esquema local recreado con éxito.")

	// --- 2. Resetear Base de Datos Remota (Supabase/PostgreSQL) ---
	if !d.isRemoteDBAvailable() {
		d.Log.Warn("La base de datos remota no está disponible. Omitiendo reseteo remoto.")
		return nil
	}

	d.Log.Info("Reseteando la base de datos remota...")
	// Eliminar todas las tablas.
	err = d.RemoteDB.Migrator().DropTable(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al eliminar las tablas remotas: %v", err)
		return fmt.Errorf("error al eliminar las tablas remotas: %w", err)
	}
	d.Log.Info("Tablas remotas eliminadas con éxito.")

	// Volver a crear el esquema.
	err = d.RemoteDB.AutoMigrate(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al recrear el esquema remoto: %v", err)
		return fmt.Errorf("error al recrear el esquema remoto: %w", err)
	}
	d.Log.Info("✅ Esquema remoto recreado con éxito.")

	d.Log.Warn("DEEP RESET completado en ambas bases de datos.")
	return nil
}
