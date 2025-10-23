package backend

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---- LÓGICA DE STOCK REFACTORIZADA ----

// RecalcularYActualizarStock ahora usa SQL nativo y debe ser invocado donde se necesite consistencia.
// Acepta tanto un pool de DB como una transacción existente.
func RecalcularYActualizarStock(dbExecutor interface{}, productoID uint) error {
	var err error
	var stockCalculado int

	// SQL para calcular el stock real desde la fuente de verdad (operacion_stocks)
	query := "SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_id = ?"

	// Ejecutar la query de cálculo de stock
	switch txOrDb := dbExecutor.(type) {
	case *sql.Tx:
		err = txOrDb.QueryRow(query, productoID).Scan(&stockCalculado)
	case *sql.DB:
		err = txOrDb.QueryRow(query, productoID).Scan(&stockCalculado)
	default:
		return fmt.Errorf("tipo de ejecutor de base de datos no válido")
	}
	if err != nil {
		return fmt.Errorf("error al calcular la suma de stock para el producto ID %d: %w", productoID, err)
	}

	// SQL para actualizar la caché en la tabla de productos
	updateQuery := "UPDATE productos SET stock = ? WHERE id = ?"

	// Ejecutar la query de actualización
	switch txOrDb := dbExecutor.(type) {
	case *sql.Tx:
		_, err = txOrDb.Exec(updateQuery, stockCalculado, productoID)
	case *sql.DB:
		_, err = txOrDb.Exec(updateQuery, stockCalculado, productoID)
	}
	if err != nil {
		return fmt.Errorf("error al actualizar el stock en caché para el producto ID %d: %w", productoID, err)
	}

	return nil
}

// calcularStockRealLocal es una función auxiliar para obtener el stock real dentro de una transacción.
func calcularStockRealLocal(tx *sql.Tx, productoID uint) (int, error) {
	var stockCalculado int
	query := "SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_id = ?"
	err := tx.QueryRow(query, productoID).Scan(&stockCalculado)
	if err != nil {
		return 0, err
	}
	return stockCalculado, nil
}

// ---- LÓGICA DE TRANSACCIONES (VENTAS) REFACTORIZADA ----

func (d *Db) RegistrarVenta(req VentaRequest) (Factura, error) {
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return Factura{}, fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	// 1. Crear la cabecera de la factura primero para obtener su ID.
	numeroFactura, err := d.generarNumeroFactura(tx)
	if err != nil {
		return Factura{}, fmt.Errorf("error al generar número de factura: %w", err)
	}

	factura := Factura{
		UUID:          uuid.New().String(),
		NumeroFactura: numeroFactura,
		FechaEmision:  time.Now(),
		VendedorID:    req.VendedorID,
		ClienteID:     req.ClienteID,
		Estado:        "Pagada",
		MetodoPago:    req.MetodoPago,
	}

	var subtotal float64
	var detallesFactura []DetalleFactura
	var opsStock []OperacionStock

	// 2. Procesar cada producto, validar stock y preparar operaciones.
	for _, p := range req.Productos {
		var producto Producto
		err := tx.QueryRow("SELECT id, nombre, stock FROM productos WHERE id = ?", p.ID).Scan(&producto.ID, &producto.Nombre, &producto.Stock)
		if err != nil {
			return Factura{}, fmt.Errorf("producto con ID %d no encontrado: %w", p.ID, err)
		}

		// Verificación de stock contra la fuente de verdad.
		stockActual, err := calcularStockRealLocal(tx, p.ID)
		if err != nil {
			return Factura{}, fmt.Errorf("error al calcular stock para %s: %w", producto.Nombre, err)
		}
		if stockActual < p.Cantidad {
			return Factura{}, fmt.Errorf("stock insuficiente para %s. Disponible: %d, Solicitado: %d", producto.Nombre, stockActual, p.Cantidad)
		}

		// Preparar operación de stock
		op := OperacionStock{
			UUID:           uuid.New().String(),
			ProductoID:     producto.ID,
			TipoOperacion:  "VENTA",
			CantidadCambio: -p.Cantidad,
			VendedorID:     req.VendedorID,
			Timestamp:      time.Now(),
		}
		opsStock = append(opsStock, op)

		// Preparar detalle de factura
		precioTotalProducto := p.PrecioUnitario * float64(p.Cantidad)
		subtotal += precioTotalProducto
		detallesFactura = append(detallesFactura, DetalleFactura{
			UUID:           uuid.New().String(),
			ProductoID:     producto.ID,
			Cantidad:       p.Cantidad,
			PrecioUnitario: p.PrecioUnitario,
			PrecioTotal:    precioTotalProducto,
		})
	}
	factura.Subtotal, factura.IVA, factura.Total = subtotal, 0, subtotal
	// 3. Insertar la factura y obtener el ID.
	var timestamp = time.Now()
	res, err := tx.ExecContext(d.ctx,
		"INSERT INTO facturas (uuid, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		factura.UUID, factura.NumeroFactura, factura.FechaEmision, factura.VendedorID, factura.ClienteID, factura.Subtotal, factura.IVA, factura.Total, factura.Estado, factura.MetodoPago, timestamp, timestamp,
	)
	if err != nil {
		return Factura{}, fmt.Errorf("error al crear la factura: %w", err)
	}
	facturaID, err := res.LastInsertId()
	if err != nil {
		return Factura{}, fmt.Errorf("error al obtener ID de la factura: %w", err)
	}
	factura.ID = uint(facturaID)

	// 4. Insertar masivamente los detalles y las operaciones de stock.
	stmtDetalles, err := tx.PrepareContext(d.ctx, "INSERT INTO detalle_facturas (uuid, factura_id, factura_uuid, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return Factura{}, err
	}
	defer stmtDetalles.Close()
	for _, detalle := range detallesFactura {
		_, err := stmtDetalles.ExecContext(d.ctx, detalle.UUID, factura.ID, factura.UUID, detalle.ProductoID, detalle.Cantidad, detalle.PrecioUnitario, detalle.PrecioTotal, timestamp, timestamp)
		if err != nil {
			return Factura{}, fmt.Errorf("error al crear detalle de factura: %w", err)
		}
	}

	stmtOps, err := tx.Prepare("INSERT INTO operacion_stocks (uuid, producto_id, tipo_operacion, cantidad_cambio, vendedor_id, timestamp, factura_id, factura_uuid) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return Factura{}, err
	}
	defer stmtOps.Close()
	facturaIDUint := uint(facturaID)
	for _, op := range opsStock {
		_, err := stmtOps.ExecContext(d.ctx, op.UUID, op.ProductoID, op.TipoOperacion, op.CantidadCambio, op.VendedorID, op.Timestamp, &facturaIDUint, &factura.UUID)
		if err != nil {
			return Factura{}, fmt.Errorf("error creando operación de stock: %w", err)
		}
	}

	// 5. Recalcular y actualizar la caché de stock para cada producto afectado.
	for _, detalle := range detallesFactura {
		if err := RecalcularYActualizarStock(tx, detalle.ProductoID); err != nil {
			return Factura{}, fmt.Errorf("error al recalcular stock para producto ID %d: %w", detalle.ProductoID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Factura{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	// 6. Lanzar sincronizaciones en segundo plano.
	go func(uuid string) {
		if err := d.syncVentaToRemote(uuid); err != nil {
			d.Log.Errorf("error sincronizando venta remota para factura UUID %s: %v", uuid, err)
		}
	}(factura.UUID)

	// Devolver la factura completa
	return d.ObtenerDetalleFactura(factura.UUID)
}

func (d *Db) generarNumeroFactura(tx *sql.Tx) (string, error) {
	var ultimoNumero sql.NullString
	err := tx.QueryRow("SELECT numero_factura FROM facturas ORDER BY id DESC LIMIT 1").Scan(&ultimoNumero)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	nuevoNumero := 1000
	if ultimoNumero.Valid {
		var numeroActual int
		if _, err := fmt.Sscanf(ultimoNumero.String, "FAC-%d", &numeroActual); err == nil {
			nuevoNumero = numeroActual + 1
		}
	}
	return fmt.Sprintf("FAC-%d", nuevoNumero), nil
}

func (d *Db) ObtenerFacturasPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var facturas []Factura

	// Base de la query
	baseQuery := `
		FROM facturas f
		JOIN clientes c ON f.cliente_id = c.id
		JOIN vendedors v ON f.vendedor_id = v.id
	`
	// Construcción de la cláusula WHERE para la búsqueda
	var whereClause string
	var args []interface{}
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		whereClause = "WHERE LOWER(f.numero_factura) LIKE ? OR LOWER(c.nombre) LIKE ? OR LOWER(v.nombre) LIKE ?"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Obtener el total de registros
	var total int64
	countQuery := "SELECT COUNT(f.id) " + baseQuery + whereClause
	err := d.LocalDB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return PaginatedResult{}, err
	}

	// Construcción de la cláusula ORDER BY de forma segura
	allowedSortBy := map[string]string{
		"NumeroFactura": "f.numero_factura",
		"fecha_emision": "f.fecha_emision",
		"Cliente":       "c.nombre",
		"Vendedor":      "v.nombre",
		"Total":         "f.total",
	}
	orderByClause := "ORDER BY f.id DESC"
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "ASC"
		if strings.ToLower(sortOrder) == "desc" {
			order = "DESC"
		}
		orderByClause = fmt.Sprintf("ORDER BY %s %s", col, order)
	}

	// Paginación
	offset := (page - 1) * pageSize
	paginationClause := fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	// Query final para obtener los registros
	selectQuery := `
		SELECT f.id, f.uuid, f.numero_factura, f.fecha_emision, f.total, c.id, c.nombre, v.id, v.nombre
	` + baseQuery + whereClause + orderByClause + paginationClause

	rows, err := d.LocalDB.Query(selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var f Factura
		err := rows.Scan(&f.ID, &f.UUID, &f.NumeroFactura, &f.FechaEmision, &f.Total, &f.Cliente.ID, &f.Cliente.Nombre, &f.Vendedor.ID, &f.Vendedor.Nombre)
		if err != nil {
			return PaginatedResult{}, err
		}
		facturas = append(facturas, f)
	}

	return PaginatedResult{Records: facturas, TotalRecords: total}, nil
}

func (d *Db) ObtenerDetalleFactura(facturaUUID string) (Factura, error) {
	var factura Factura
	d.Log.Infof("[LOCAL] - Obteniendo detalles para Factura UUID: %s", facturaUUID)

	// 1. Obtener la factura principal y los datos del cliente/vendedor
	queryFactura := `
					SELECT
						f.id,
						f.uuid,
						f.numero_factura,
						f.fecha_emision,
						f.subtotal,
						f.iva,
						f.total,
						f.estado,
						f.metodo_pago,
						f.cliente_id,
						c.id,
						c.nombre,
						c.apellido,
						c.numero_id,
						f.vendedor_id,
						v.id,
						v.nombre,
						v.apellido
					FROM
						facturas f
						JOIN clientes c ON f.cliente_id = c.id
						JOIN vendedors v ON f.vendedor_id = v.id
					WHERE
						f.uuid = ?
	`
	// Escaneamos los IDs y también los datos anidados para tener el objeto completo
	err := d.LocalDB.QueryRow(queryFactura, facturaUUID).Scan(
		&factura.ID, &factura.UUID, &factura.NumeroFactura, &factura.FechaEmision, &factura.Subtotal, &factura.IVA, &factura.Total, &factura.Estado, &factura.MetodoPago,
		&factura.ClienteID, &factura.Cliente.ID, &factura.Cliente.Nombre, &factura.Cliente.Apellido, &factura.Cliente.NumeroID,
		&factura.VendedorID, &factura.Vendedor.ID, &factura.Vendedor.Nombre, &factura.Vendedor.Apellido,
	)
	if err != nil {
		d.Log.Errorf("Error al obtener la cabecera de la factura ID %s: %v", facturaUUID, err)
		return Factura{}, err
	}

	factura.Detalles = make([]DetalleFactura, 0)

	// 2. Obtener los detalles de la factura (productos)
	queryDetalles := `
		SELECT d.id, d.uuid, d.cantidad, d.precio_unitario, d.precio_total,
			p.id, p.codigo, p.nombre
		FROM detalle_facturas d
		JOIN productos p ON d.producto_id = p.id
		WHERE d.factura_uuid = ?
	`
	rows, err := d.LocalDB.Query(queryDetalles, facturaUUID)
	if err != nil {
		d.Log.Errorf("Error al consultar los detalles para la factura ID %s: %v", facturaUUID, err)
		return factura, err
	}
	defer rows.Close()

	detailCount := 0
	for rows.Next() {
		var detalle DetalleFactura
		err := rows.Scan(
			&detalle.ID, &detalle.UUID, &detalle.Cantidad, &detalle.PrecioUnitario, &detalle.PrecioTotal,
			&detalle.Producto.ID, &detalle.Producto.Codigo, &detalle.Producto.Nombre,
		)
		if err != nil {
			d.Log.Errorf("Error al escanear un detalle de la factura ID %s: %v", facturaUUID, err)
			continue
		}
		factura.Detalles = append(factura.Detalles, detalle)
		detailCount++
	}

	if err = rows.Err(); err != nil {
		d.Log.Errorf("Error final al iterar los detalles de la factura ID %s: %v", facturaUUID, err)
		return factura, err
	}

	d.Log.Infof("Se encontraron y adjuntaron %d detalles para la Factura ID: %s", detailCount, facturaUUID)

	return factura, nil
}

func (d *Db) RegistrarCompra(req CompraRequest) (Compra, error) {
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return Compra{}, fmt.Errorf("error al iniciar transacción de compra: %w", err)
	}
	defer tx.Rollback()

	var totalCompra float64
	for _, p := range req.Productos {
		totalCompra += p.PrecioCompraUnitario * float64(p.Cantidad)
	}

	compra := Compra{
		Fecha:         time.Now(),
		ProveedorID:   req.ProveedorID,
		FacturaNumero: req.FacturaNumero,
		Total:         totalCompra,
	}

	res, err := tx.Exec("INSERT INTO compras (fecha, proveedor_id, factura_numero, total) VALUES (?, ?, ?, ?)",
		compra.Fecha, compra.ProveedorID, compra.FacturaNumero, compra.Total)
	if err != nil {
		return Compra{}, fmt.Errorf("error al crear la compra: %w", err)
	}
	compraID, err := res.LastInsertId()
	if err != nil {
		return Compra{}, fmt.Errorf("error al obtener ID de compra: %w", err)
	}
	compra.ID = uint(compraID)

	// Preparar statements para inserciones masivas
	stmtDetalles, err := tx.Prepare("INSERT INTO detalle_compras (compra_id, producto_id, cantidad, precio_compra_unitario) VALUES (?, ?, ?, ?)")
	if err != nil {
		return Compra{}, err
	}
	defer stmtDetalles.Close()

	stmtOps, err := tx.Prepare("INSERT INTO operacion_stocks (uuid, producto_id, tipo_operacion, cantidad_cambio, vendedor_id, timestamp) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return Compra{}, err
	}
	defer stmtOps.Close()

	for _, p := range req.Productos {
		// Insertar detalle de compra
		_, err := stmtDetalles.Exec(compra.ID, p.ProductoID, p.Cantidad, p.PrecioCompraUnitario)
		if err != nil {
			return Compra{}, fmt.Errorf("error al crear detalle de compra: %w", err)
		}

		// Insertar operación de stock
		_, err = stmtOps.Exec(uuid.New().String(), p.ProductoID, "COMPRA", p.Cantidad, 1, time.Now()) // VendedorID 1 como sistema/admin
		if err != nil {
			return Compra{}, fmt.Errorf("error creando operación de stock por compra: %w", err)
		}

		// Recalcular stock para el producto afectado
		if err := RecalcularYActualizarStock(tx, p.ProductoID); err != nil {
			return Compra{}, fmt.Errorf("error al actualizar stock por compra: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Compra{}, fmt.Errorf("error al confirmar transacción de compra: %w", err)
	}

	go d.syncCompraToRemote(compra.ID)

	// Aquí se debería devolver la compra completa, similar a ObtenerDetalleFactura
	return compra, nil
}
