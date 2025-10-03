package backend

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// --- VENTAS (FACTURAS) ---

func (d *Db) RegistrarVenta(req VentaRequest) (Factura, error) {
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	factura := Factura{
		FechaEmision: time.Now(), VendedorID: req.VendedorID, ClienteID: req.ClienteID,
		Estado: "Pagada", MetodoPago: req.MetodoPago,
	}

	var subtotal float64
	var detallesFactura []DetalleFactura

	for _, p := range req.Productos {
		var producto Producto
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&producto, p.ID).Error; err != nil {
			return Factura{}, fmt.Errorf("producto con ID %d no encontrado", p.ID)
		}
		if producto.Stock < p.Cantidad {
			return Factura{}, fmt.Errorf("stock insuficiente para %s. Disponible: %d, Solicitado: %d", producto.Nombre, producto.Stock, p.Cantidad)
		}

		nuevoStock := producto.Stock - p.Cantidad
		op := OperacionStock{
			UUID: uuid.New().String(), ProductoID: producto.ID, TipoOperacion: "VENTA",
			CantidadCambio: -p.Cantidad, StockResultante: nuevoStock, VendedorID: req.VendedorID, Timestamp: time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return Factura{}, fmt.Errorf("error creando operación de stock para %s: %w", producto.Nombre, err)
		}

		if err := tx.Model(&producto).Update("Stock", nuevoStock).Error; err != nil {
			return Factura{}, fmt.Errorf("error al actualizar el stock de %s: %w", producto.Nombre, err)
		}

		precioTotalProducto := p.PrecioUnitario * float64(p.Cantidad)
		subtotal += precioTotalProducto
		detallesFactura = append(detallesFactura, DetalleFactura{
			ProductoID: producto.ID, Cantidad: p.Cantidad, PrecioUnitario: p.PrecioUnitario, PrecioTotal: precioTotalProducto,
		})
	}

	factura.Subtotal, factura.IVA, factura.Total = subtotal, 0, subtotal
	factura.NumeroFactura = d.generarNumeroFactura()

	if err := tx.Create(&factura).Error; err != nil {
		return Factura{}, fmt.Errorf("error al crear la factura: %w", err)
	}

	for i := range detallesFactura {
		detallesFactura[i].FacturaID = factura.ID
	}
	if err := tx.Create(&detallesFactura).Error; err != nil {
		return Factura{}, fmt.Errorf("error al crear los detalles de la factura: %w", err)
	}
	if err := tx.Model(&OperacionStock{}).Where("factura_id IS NULL AND tipo_operacion = 'VENTA' AND vendedor_id = ?", req.VendedorID).Update("factura_id", factura.ID).Error; err != nil {
		return Factura{}, fmt.Errorf("error al asociar factura a operaciones de stock: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return Factura{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncVentaToRemote(factura.ID)
	go d.SincronizarOperacionesStock()
	return d.ObtenerDetalleFactura(factura.ID)
}

func (d *Db) generarNumeroFactura() string {
	var ultimaFactura Factura
	nuevoNumero := 1000
	if d.LocalDB.Order("id desc").First(&ultimaFactura).Error == nil {
		if _, err := fmt.Sscanf(ultimaFactura.NumeroFactura, "FAC-%d", &nuevoNumero); err == nil {
			nuevoNumero++
		}
	}
	return fmt.Sprintf("FAC-%d", nuevoNumero)
}

func (d *Db) ObtenerFacturasPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var facturas []Factura
	var total int64
	query := d.LocalDB.Model(&Factura{}).Preload("Cliente").Preload("Vendedor")

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Joins("JOIN clientes ON clientes.id = facturas.cliente_id").
			Joins("JOIN vendedors ON vendedors.id = facturas.vendedor_id").
			Where("LOWER(facturas.numero_factura) LIKE ? OR LOWER(clientes.nombre) LIKE ? OR LOWER(vendedors.nombre) LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	allowedSortBy := map[string]string{"NumeroFactura": "numero_factura", "fecha_emision": "fecha_emision", "Cliente": "clientes.nombre", "Vendedor": "vendedors.nombre", "Total": "total"}
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "asc"
		if strings.ToLower(sortOrder) == "desc" {
			order = "desc"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	} else {
		query = query.Order("facturas.id DESC")
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&facturas).Error
	for i := range facturas {
		facturas[i].Vendedor.Contrasena = ""
	}
	return PaginatedResult{Records: facturas, TotalRecords: total}, err
}

func (d *Db) ObtenerDetalleFactura(facturaID uint) (Factura, error) {
	var factura Factura
	err := d.LocalDB.Preload("Cliente").Preload("Vendedor").Preload("Detalles.Producto").First(&factura, facturaID).Error
	if err == nil {
		factura.Vendedor.Contrasena = ""
	}
	return factura, err
}

// --- COMPRAS ---

func (d *Db) RegistrarCompra(req CompraRequest) (Compra, error) {
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	compra := Compra{
		Fecha: time.Now(), ProveedorID: req.ProveedorID, FacturaNumero: req.FacturaNumero,
	}

	var totalCompra float64
	var detallesCompra []DetalleCompra

	for _, p := range req.Productos {
		var producto Producto
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&producto, p.ProductoID).Error; err != nil {
			return Compra{}, fmt.Errorf("producto con ID %d no encontrado", p.ProductoID)
		}
		precioTotalProducto := p.PrecioCompraUnitario * float64(p.Cantidad)
		totalCompra += precioTotalProducto
		detallesCompra = append(detallesCompra, DetalleCompra{
			ProductoID: p.ProductoID, Cantidad: p.Cantidad, PrecioCompraUnitario: p.PrecioCompraUnitario,
		})

		nuevoStock := producto.Stock + p.Cantidad
		op := OperacionStock{
			UUID: uuid.New().String(), ProductoID: producto.ID, TipoOperacion: "COMPRA",
			CantidadCambio: p.Cantidad, StockResultante: nuevoStock, VendedorID: 1, Timestamp: time.Now(), // VendedorID 1 como sistema/admin
		}
		if err := tx.Create(&op).Error; err != nil {
			return Compra{}, fmt.Errorf("error creando operación de stock por compra para %s: %w", producto.Nombre, err)
		}
		if err := tx.Model(&producto).Update("Stock", nuevoStock).Error; err != nil {
			return Compra{}, fmt.Errorf("error al actualizar stock por compra para %s: %w", producto.Nombre, err)
		}
	}

	compra.Total = totalCompra
	if err := tx.Create(&compra).Error; err != nil {
		return Compra{}, fmt.Errorf("error al crear la compra: %w", err)
	}

	for i := range detallesCompra {
		detallesCompra[i].CompraID = compra.ID
	}
	if err := tx.Create(&detallesCompra).Error; err != nil {
		return Compra{}, fmt.Errorf("error al crear detalles de compra: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return Compra{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncCompraToRemote(compra.ID)
	go d.SincronizarOperacionesStock()

	var compraCompleta Compra
	d.LocalDB.Preload("Proveedor").Preload("Detalles.Producto").First(&compraCompleta, compra.ID)
	return compraCompleta, nil
}
