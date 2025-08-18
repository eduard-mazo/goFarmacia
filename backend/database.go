package backend

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- MODELOS DE LA BASE DE DATOS ---
// CORRECCIÓN DEFINITIVA: Se agrega la etiqueta `wails:"ts.type=string"` a CADA campo `time.Time`
// para que Wails sepa cómo convertirlo a TypeScript. Esto resuelve el error "Not found: time.Time".

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

// --- ESTRUCTURAS PARA REQUESTS DEL FRONTEND ---

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

// --- LÓGICA DE LA BASE DE DATOS ---

type Db struct {
	ctx context.Context
	DB  *gorm.DB
}

func NewDb() *Db {
	return &Db{}
}

func (d *Db) Startup(ctx context.Context) {
	d.ctx = ctx
	fmt.Println("DB backend inicializado")
	d.initDB()
}

func (d *Db) initDB() {
	var err error
	d.DB, err = gorm.Open(sqlite.Open("farmacia.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Fallo al conectar a la base de datos: %v", err)
	}

	err = d.DB.AutoMigrate(&Vendedor{}, &Cliente{}, &Producto{}, &Factura{}, &DetalleFactura{})
	if err != nil {
		log.Fatalf("Fallo en la migración de la base de datos: %v", err)
	}

	log.Println("Base de datos conectada y migrada exitosamente")
}
