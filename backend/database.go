package backend

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

type OperacionStock struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UUID            string    `gorm:"uniqueIndex" json:"uuid"`
	ProductoID      uint      `json:"producto_id"`
	TipoOperacion   string    `json:"tipo_operacion"`
	CantidadCambio  int       `json:"cantidad_cambio"`
	StockResultante int       `json:"stock_resultante"`
	VendedorID      uint      `json:"vendedor_id"`
	FacturaID       *uint     `json:"factura_id"`
	Timestamp       time.Time `json:"timestamp"`
	Sincronizado    bool      `gorm:"default:false" json:"sincronizado"`
}

type Claims struct {
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	Nombre  string `json:"nombre"`
	Cedula  string `json:"cedula"`
	MFAStep string `json:"mfa_step,omitempty"` // Para el flujo de MFA
	jwt.RegisteredClaims
}

type LoginResponse struct {
	MFARequired bool     `json:"mfa_required"`
	Token       string   `json:"token"` // Será un token temporal si MFA es requerido
	Vendedor    Vendedor `json:"vendedor"`
}

type MFASetupResponse struct {
	Secret   string `json:"secret"`    // Para mostrar al usuario como backup
	ImageURL string `json:"image_url"` // QR Code en formato DataURL
}

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
	MFASecret  string         `json:"-"`
	MFAEnabled bool           `gorm:"default:false" json:"mfa_enabled"`
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
	ctx      context.Context
	LocalDB  *gorm.DB // For SQLite
	RemoteDB *gorm.DB // For Supabase (PostgreSQL)
	Log      *logrus.Logger
	jwtKey   []byte
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
	d.LocalDB, err = gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:farmacia.db"}, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		d.Log.Fatalf("FATAL: Failed to connect to local SQLite database: %v", err)
	}
	d.Log.Info("✅ Local SQLite database connected successfully.")
	models := []interface{}{&Vendedor{}, &Cliente{}, &Producto{}, &Factura{}, &DetalleFactura{}, &Proveedor{}, &Compra{}, &DetalleCompra{}, &OperacionStock{}}

	err = d.LocalDB.AutoMigrate(models...)
	if err != nil {
		d.Log.Fatalf("FATAL: Failed to migrate local database schema: %v", err)
	}
	d.Log.Info("✅ Local database schema migrated successfully.")

	err = godotenv.Load()
	if err != nil {
		d.Log.Fatalf("Error loading .env file: %v", err)
	}
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		d.Log.Fatalf("FATAL: La variable de entorno JWT_SECRET_KEY no está configurada.")
	}
	d.jwtKey = []byte(secret)
	d.Log.Info("✅ Clave secreta JWT cargada exitosamente.")
	d.RemoteDB, err = gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		d.Log.Warnf("WARNING: Failed to connect to remote Supabase database. App will run in offline mode. Error: %v", err)
		d.RemoteDB = nil
	} else {
		d.Log.Info("✅ Remote Supabase database connected successfully.")
		d.Log.Info("Migrating remote database schema...")
		err = d.RemoteDB.AutoMigrate(models...)
		if err != nil {
			d.Log.Errorf("ERROR: Failed to migrate remote schema. Falling back to offline mode. Error: %v", err)
			d.RemoteDB = nil
		} else {
			d.Log.Info("✅ Remote database schema migrated successfully.")
		}
	}
}

func (d *Db) isRemoteDBAvailable() bool {
	return d.RemoteDB != nil
}

// DeepResetDatabases realiza un reseteo completo y destructivo de ambas bases de datos.
// ¡ADVERTENCIA! ESTA ACCIÓN BORRARÁ TODOS LOS DATOS PERMANENTEMENTE.
func (d *Db) DeepResetDatabases() error {
	d.Log.Warn("¡INICIANDO DEEP RESET! Todos los datos serán eliminados.")
	modelsToReset := []interface{}{
		&DetalleFactura{}, &DetalleCompra{}, &Factura{}, &Compra{},
		&Vendedor{}, &Cliente{}, &Producto{}, &Proveedor{}, &OperacionStock{},
	}

	d.Log.Info("Reseteando la base de datos local...")
	err := d.LocalDB.Migrator().DropTable(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al eliminar las tablas locales: %v", err)
		return fmt.Errorf("error al eliminar las tablas locales: %w", err)
	}
	d.Log.Info("Tablas locales eliminadas con éxito.")
	err = d.LocalDB.AutoMigrate(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al recrear el esquema local: %v", err)
		return fmt.Errorf("error al recrear el esquema local: %w", err)
	}
	d.Log.Info("✅ Esquema local recreado con éxito.")

	if !d.isRemoteDBAvailable() {
		d.Log.Warn("La base de datos remota no está disponible. Omitiendo reseteo remoto.")
		return nil
	}

	d.Log.Info("Reseteando la base de datos remota...")
	err = d.RemoteDB.Migrator().DropTable(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al eliminar las tablas remotas: %v", err)
		return fmt.Errorf("error al eliminar las tablas remotas: %w", err)
	}
	d.Log.Info("Tablas remotas eliminadas con éxito.")

	err = d.RemoteDB.AutoMigrate(modelsToReset...)
	if err != nil {
		d.Log.Errorf("Error al recrear el esquema remoto: %v", err)
		return fmt.Errorf("error al recrear el esquema remoto: %w", err)
	}
	d.Log.Info("✅ Esquema remoto recreado con éxito.")

	d.Log.Warn("DEEP RESET completado en ambas bases de datos.")
	return nil
}
