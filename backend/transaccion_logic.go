package backend

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RecalcularYActualizarStock se convierte en la única función responsable de actualizar la columna 'stock'.
// Calcula el total desde la tabla de operaciones y lo escribe en la tabla de productos.
func RecalcularYActualizarStock(tx *gorm.DB, productoID uint) error {
	if tx.Error != nil {
		return tx.Error
	}
	var stockCalculado int64

	// 1. Calcular el stock real desde la fuente de la verdad (operacion_stocks)
	err := tx.Model(&OperacionStock{}).
		Where("producto_id = ?", productoID).
		Select("COALESCE(SUM(cantidad_cambio), 0)").
		Row().
		Scan(&stockCalculado)

	if err != nil {
		return fmt.Errorf("error al calcular la suma de stock para el producto ID %d: %w", productoID, err)
	}

	// 2. Actualizar el valor en caché en la tabla de productos
	if err := tx.Model(&Producto{}).Where("id = ?", productoID).Update("stock", stockCalculado).Error; err != nil {
		return fmt.Errorf("error al actualizar el stock en caché para el producto ID %d: %w", productoID, err)
	}

	return nil
}

func (d *Db) RegistrarVenta(req VentaRequest) (Factura, error) {
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	factura := Factura{
		FechaEmision: time.Now(),
		VendedorID:   req.VendedorID,
		ClienteID:    req.ClienteID,
		Estado:       "Pagada",
		MetodoPago:   req.MetodoPago,
	}

	var subtotal float64
	var detallesFactura []DetalleFactura
	var opsParaActualizar []*OperacionStock

	for _, p := range req.Productos {
		var producto Producto
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&producto, p.ID).Error; err != nil {
			return Factura{}, fmt.Errorf("producto con ID %d no encontrado", p.ID)
		}

		// VERIFICACIÓN DE STOCK CONTRA LA FUENTE DE VERDAD
		stockActual := d.calcularStockRealLocal(p.ID)
		if stockActual < p.Cantidad {
			return Factura{}, fmt.Errorf("stock insuficiente para %s. Disponible: %d, Solicitado: %d", producto.Nombre, stockActual, p.Cantidad)
		}

		// 1. Registrar la operación de stock
		op := &OperacionStock{
			UUID:           uuid.New().String(),
			ProductoID:     producto.ID,
			TipoOperacion:  "VENTA",
			CantidadCambio: -p.Cantidad,
			VendedorID:     req.VendedorID,
			Timestamp:      time.Now(),
		}
		if err := tx.Create(op).Error; err != nil {
			return Factura{}, fmt.Errorf("error creando operación de stock para %s: %w", producto.Nombre, err)
		}
		opsParaActualizar = append(opsParaActualizar, op)

		// 2. Recalcular el stock total desde la fuente de verdad (operaciones)
		if err := RecalcularYActualizarStock(tx, producto.ID); err != nil {
			return Factura{}, fmt.Errorf("error al recalcular el stock de %s: %w", producto.Nombre, err)
		}

		var stockResultante int
		tx.Model(&OperacionStock{}).Where("producto_id = ?", producto.ID).Select("COALESCE(SUM(cantidad_cambio), 0)").Row().Scan(&stockResultante)
		op.StockResultante = stockResultante

		precioTotalProducto := p.PrecioUnitario * float64(p.Cantidad)
		subtotal += precioTotalProducto
		detallesFactura = append(detallesFactura, DetalleFactura{
			ProductoID:     producto.ID,
			Cantidad:       p.Cantidad,
			PrecioUnitario: p.PrecioUnitario,
			PrecioTotal:    precioTotalProducto,
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

	// Asociar la factura a las operaciones de stock creadas en esta transacción.
	for _, op := range opsParaActualizar {
		if err := tx.Model(op).Update("factura_id", factura.ID).Error; err != nil {
			return Factura{}, fmt.Errorf("error al asociar factura a operación de stock: %w", err)
		}
		// También actualizamos el stock resultante ahora que lo sabemos
		if err := tx.Model(op).Update("stock_resultante", op.StockResultante).Error; err != nil {
			return Factura{}, fmt.Errorf("error al actualizar stock resultante en operación: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return Factura{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncVentaToRemote(factura.ID)
	go d.SincronizarOperacionesStockHaciaRemoto() // Nombre actualizado
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

func (d *Db) RegistrarCompra(req CompraRequest) (Compra, error) {
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	compra := Compra{
		Fecha:         time.Now(),
		ProveedorID:   req.ProveedorID,
		FacturaNumero: req.FacturaNumero,
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
			ProductoID:           p.ProductoID,
			Cantidad:             p.Cantidad,
			PrecioCompraUnitario: p.PrecioCompraUnitario,
		})

		op := OperacionStock{
			UUID:           uuid.New().String(),
			ProductoID:     producto.ID,
			TipoOperacion:  "COMPRA",
			CantidadCambio: p.Cantidad,
			VendedorID:     1, // VendedorID 1 como sistema/admin
			Timestamp:      time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return Compra{}, fmt.Errorf("error creando operación de stock por compra para %s: %w", producto.Nombre, err)
		}

		if err := RecalcularYActualizarStock(tx, producto.ID); err != nil {
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
	go d.SincronizarOperacionesStockHaciaRemoto() // Nombre actualizado

	var compraCompleta Compra
	d.LocalDB.Preload("Proveedor").Preload("Detalles.Producto").First(&compraCompleta, compra.ID)
	return compraCompleta, nil
}
