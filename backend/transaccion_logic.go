package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---- LÓGICA DE STOCK REFACTORIZADA ----

// RecalcularYActualizarStock ahora usa SQL nativo y debe ser invocado donde se necesite consistencia.
// Acepta tanto un pool de DB como una transacción existente.
func RecalcularYActualizarStock(dbExecutor interface{}, productoUUID string) error {
	var err error
	var stockCalculado int

	// SQL para calcular el stock real desde la fuente de verdad (operacion_stocks)
	query := "SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_uuid = ?"

	// Ejecutar la query de cálculo de stock
	switch txOrDb := dbExecutor.(type) {
	case *sql.Tx:
		err = txOrDb.QueryRow(query, productoUUID).Scan(&stockCalculado)
	case *sql.DB:
		err = txOrDb.QueryRow(query, productoUUID).Scan(&stockCalculado)
	default:
		return fmt.Errorf("tipo de ejecutor de base de datos no válido")
	}
	if err != nil {
		return fmt.Errorf("error al calcular la suma de stock para el producto UUID %s: %w", productoUUID, err)
	}

	// SQL para actualizar la caché en la tabla de productos
	updateQuery := "UPDATE productos SET stock = ? WHERE uuid = ?"

	// Ejecutar la query de actualización
	switch txOrDb := dbExecutor.(type) {
	case *sql.Tx:
		_, err = txOrDb.Exec(updateQuery, stockCalculado, productoUUID)
	case *sql.DB:
		_, err = txOrDb.Exec(updateQuery, stockCalculado, productoUUID)
	}
	if err != nil {
		return fmt.Errorf("error al actualizar el stock en caché para el producto UUID %s: %w", productoUUID, err)
	}

	return nil
}

// calcularStockRealLocal es una función auxiliar para obtener el stock real dentro de una transacción.
func calcularStockRealLocal(tx *sql.Tx, productoUUID string) (int, error) {
	var stockCalculado int
	query := "SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_uuid = ?"
	err := tx.QueryRow(query, productoUUID).Scan(&stockCalculado)
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
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [RegistrarVenta] rollback %v", rErr)
		}
	}()

	// 1. Crear la cabecera de la factura primero para obtener su ID.
	d.Log.Info("1. Crear la cabecera de la factura primero para obtener su UUID.")
	numeroFactura, err := d.generarNumeroFactura(tx)
	d.Log.Infof("Número factura generado [%s]", numeroFactura)
	if err != nil {
		return Factura{}, fmt.Errorf("error al generar número de factura: %w", err)
	}

	factura := Factura{
		UUID:          uuid.New().String(),
		NumeroFactura: numeroFactura,
		FechaEmision:  time.Now(),
		VendedorUUID:  req.VendedorUUID,
		ClienteUUID:   req.ClienteUUID,
		Estado:        "Pagada",
		MetodoPago:    req.MetodoPago,
	}

	var subtotal float64
	var detallesFactura []DetalleFactura
	var opsStock []OperacionStock

	// 2. Procesar cada producto, validar stock y preparar operaciones.
	d.Log.Info("2. Procesar cada producto, validar stock y preparar operaciones.")
	for _, p := range req.Productos {
		var producto Producto
		err := tx.QueryRow("SELECT uuid, nombre, stock FROM productos WHERE uuid = ?", p.ProductoUUID).Scan(&producto.UUID, &producto.Nombre, &producto.Stock)
		if err != nil {
			return Factura{}, fmt.Errorf("producto con UUID %s no encontrado: %w", p.ProductoUUID, err)
		}

		// Verificación de stock contra la fuente de verdad.
		stockActual, err := calcularStockRealLocal(tx, p.ProductoUUID)
		if err != nil {
			return Factura{}, fmt.Errorf("error al calcular stock para %s: %w", producto.Nombre, err)
		}
		if stockActual < p.Cantidad {
			return Factura{}, fmt.Errorf("stock insuficiente para %s. Disponible: %d, Solicitado: %d", producto.Nombre, stockActual, p.Cantidad)
		}

		// Preparar operación de stock
		op := OperacionStock{
			UUID:           uuid.New().String(),
			ProductoUUID:   producto.UUID,
			TipoOperacion:  "VENTA",
			CantidadCambio: -p.Cantidad,
			VendedorUUID:   req.VendedorUUID,
			Timestamp:      time.Now(),
		}
		opsStock = append(opsStock, op)

		// Preparar detalle de factura
		precioTotalProducto := p.PrecioUnitario * float64(p.Cantidad)
		subtotal += precioTotalProducto
		detallesFactura = append(detallesFactura, DetalleFactura{
			UUID:           uuid.New().String(),
			ProductoUUID:   producto.UUID,
			Cantidad:       p.Cantidad,
			PrecioUnitario: p.PrecioUnitario,
			PrecioTotal:    precioTotalProducto,
		})
	}
	factura.Subtotal, factura.IVA, factura.Total = subtotal, 0, subtotal
	// 3. Insertar la factura y obtener el ID.
	d.Log.Info("3. Insertar la factura y obtener el ID.")
	var timestamp = time.Now()
	_, err = tx.ExecContext(d.ctx,
		"INSERT INTO facturas (uuid, numero_factura, fecha_emision, vendedor_uuid, cliente_uuid, subtotal, iva, total, estado, metodo_pago, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		factura.UUID, factura.NumeroFactura, factura.FechaEmision, factura.VendedorUUID, factura.ClienteUUID, factura.Subtotal, factura.IVA, factura.Total, factura.Estado, factura.MetodoPago, timestamp, timestamp,
	)
	if err != nil {
		d.Log.Infof("[LOCAL] - Error al insertar factura: %+v", factura)
		return Factura{}, fmt.Errorf("error al crear la factura: %w", err)
	}

	// 4. Insertar masivamente los detalles y las operaciones de stock.
	d.Log.Info("4. Insertar masivamente los detalles y las operaciones de stock.")
	stmtDetalles, err := tx.PrepareContext(d.ctx, "INSERT INTO detalle_facturas (uuid, factura_uuid, producto_uuid, cantidad, precio_unitario, precio_total, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return Factura{}, err
	}
	defer stmtDetalles.Close()
	for _, detalle := range detallesFactura {
		_, err := stmtDetalles.ExecContext(d.ctx, detalle.UUID, factura.UUID, detalle.ProductoUUID, detalle.Cantidad, detalle.PrecioUnitario, detalle.PrecioTotal, timestamp, timestamp)
		if err != nil {
			return Factura{}, fmt.Errorf("error al crear detalle de factura: %w", err)
		}
	}

	stmtOps, err := tx.Prepare("INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, vendedor_uuid, timestamp, factura_uuid) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return Factura{}, err
	}
	defer stmtOps.Close()
	for _, op := range opsStock {
		_, err := stmtOps.ExecContext(d.ctx, op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio, op.VendedorUUID, op.Timestamp, &factura.UUID)
		if err != nil {
			return Factura{}, fmt.Errorf("error creando operación de stock: %w", err)
		}
	}

	// 5. Recalcular y actualizar la caché de stock para cada producto afectado.
	d.Log.Info("5. Recalcular y actualizar la caché de stock para cada producto afectado.")
	for _, detalle := range detallesFactura {
		if err := RecalcularYActualizarStock(tx, detalle.ProductoUUID); err != nil {
			return Factura{}, fmt.Errorf("error al recalcular stock para producto UUID %s: %w", detalle.ProductoUUID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Factura{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	// 6. Lanzar sincronizaciones en segundo plano.
	d.Log.Info("6. Lanzar sincronizaciones en segundo plano.")
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
	err := tx.QueryRow("SELECT numero_factura FROM facturas ORDER BY uuid DESC LIMIT 1").Scan(&ultimoNumero)
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
		JOIN clientes c ON f.cliente_uuid = c.uuid
		JOIN vendedors v ON f.vendedor_uuid = v.uuid
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
	countQuery := "SELECT COUNT(f.uuid) " + baseQuery + whereClause
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
	orderByClause := "ORDER BY f.uuid DESC"
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
		SELECT f.uuid, f.numero_factura, f.fecha_emision, f.total, c.uuid, c.nombre, v.uuid, v.nombre
	` + baseQuery + whereClause + orderByClause + paginationClause

	rows, err := d.LocalDB.Query(selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var f Factura
		err := rows.Scan(&f.UUID, &f.NumeroFactura, &f.FechaEmision, &f.Total, &f.Cliente.UUID, &f.Cliente.Nombre, &f.Vendedor.UUID, &f.Vendedor.Nombre)
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
						f.uuid,
						f.numero_factura,
						f.fecha_emision,
						f.subtotal,
						f.iva,
						f.total,
						f.estado,
						f.metodo_pago,
						f.cliente_uuid,
						c.uuid,
						c.nombre,
						c.apellido,
						c.numero_id,
						f.vendedor_uuid,
						v.uuid,
						v.nombre,
						v.apellido
					FROM
						facturas f
						JOIN clientes c ON f.cliente_uuid = c.uuid
						JOIN vendedors v ON f.vendedor_uuid = v.uuid
					WHERE
						f.uuid = ?
	`
	// Escaneamos los IDs y también los datos anidados para tener el objeto completo
	err := d.LocalDB.QueryRow(queryFactura, facturaUUID).Scan(
		&factura.UUID, &factura.NumeroFactura, &factura.FechaEmision, &factura.Subtotal, &factura.IVA, &factura.Total, &factura.Estado, &factura.MetodoPago,
		&factura.ClienteUUID, &factura.Cliente.UUID, &factura.Cliente.Nombre, &factura.Cliente.Apellido, &factura.Cliente.NumeroID,
		&factura.VendedorUUID, &factura.Vendedor.UUID, &factura.Vendedor.Nombre, &factura.Vendedor.Apellido,
	)
	if err != nil {
		d.Log.Errorf("Error al obtener la cabecera de la factura UUID %s: %v", facturaUUID, err)
		return Factura{}, err
	}

	factura.Detalles = make([]DetalleFactura, 0)

	// 2. Obtener los detalles de la factura (productos)
	queryDetalles := `
		SELECT d.uuid, d.uuid, d.cantidad, d.precio_unitario, d.precio_total,
			p.uuid, p.codigo, p.nombre
		FROM detalle_facturas d
		JOIN productos p ON d.producto_uuid = p.uuid
		WHERE d.factura_uuid = ?
	`
	rows, err := d.LocalDB.Query(queryDetalles, facturaUUID)
	if err != nil {
		d.Log.Errorf("Error al consultar los detalles para la factura UUID %s: %v", facturaUUID, err)
		return factura, err
	}
	defer rows.Close()

	detailCount := 0
	for rows.Next() {
		var detalle DetalleFactura
		err := rows.Scan(
			&detalle.UUID, &detalle.Cantidad, &detalle.PrecioUnitario, &detalle.PrecioTotal,
			&detalle.Producto.UUID, &detalle.Producto.Codigo, &detalle.Producto.Nombre,
		)
		if err != nil {
			d.Log.Errorf("Error al escanear un detalle de la factura UUID %s: %v", facturaUUID, err)
			continue
		}
		factura.Detalles = append(factura.Detalles, detalle)
		detailCount++
	}

	if err = rows.Err(); err != nil {
		d.Log.Errorf("Error final al iterar los detalles de la factura UUID %s: %v", facturaUUID, err)
		return factura, err
	}

	d.Log.Infof("Se encontraron y adjuntaron %d detalles para la Factura UUID: %s", detailCount, facturaUUID)

	return factura, nil
}

func (d *Db) RegistrarCompra(req CompraRequest) (Compra, error) {
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return Compra{}, fmt.Errorf("error al iniciar transacción de compra: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); err != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [RegistrarCompra] rollback %v", err)
		}
	}()

	var totalCompra float64
	for _, p := range req.Productos {
		totalCompra += p.PrecioCompraUnitario * float64(p.Cantidad)
	}

	compra := Compra{
		Fecha:         time.Now(),
		ProveedorUUID: req.ProveedorUUID,
		FacturaNumero: req.FacturaNumero,
		Total:         totalCompra,
	}

	_, err = tx.Exec("INSERT INTO compras (uuid, fecha, proveedor_uuid, factura_numero, total) VALUES (?, ?, ?, ?, ?)",
		compra.UUID, compra.Fecha, compra.ProveedorUUID, compra.FacturaNumero, compra.Total)
	if err != nil {
		return Compra{}, fmt.Errorf("error al crear la compra: %w", err)
	}

	// Preparar statements para inserciones masivas
	stmtDetalles, err := tx.Prepare("INSERT INTO detalle_compras (compra_uuid, producto_uuid, cantidad, precio_compra_unitario) VALUES (?, ?, ?, ?)")
	if err != nil {
		return Compra{}, err
	}
	defer stmtDetalles.Close()

	stmtOps, err := tx.Prepare("INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, vendedor_uuid, timestamp) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return Compra{}, err
	}
	defer stmtOps.Close()

	for _, p := range req.Productos {
		// Insertar detalle de compra
		_, err := stmtDetalles.Exec(compra.UUID, p.ProductoUUID, p.Cantidad, p.PrecioCompraUnitario)
		if err != nil {
			return Compra{}, fmt.Errorf("error al crear detalle de compra: %w", err)
		}

		// Insertar operación de stock
		_, err = stmtOps.Exec(uuid.New().String(), p.ProductoUUID, "COMPRA", p.Cantidad, "SYSTEM-ADMIN", time.Now())
		if err != nil {
			return Compra{}, fmt.Errorf("error creando operación de stock por compra: %w", err)
		}

		// Recalcular stock para el producto afectado
		if err := RecalcularYActualizarStock(tx, p.ProductoUUID); err != nil {
			return Compra{}, fmt.Errorf("error al actualizar stock por compra: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Compra{}, fmt.Errorf("error al confirmar transacción de compra: %w", err)
	}

	go d.syncCompraToRemote(compra.UUID)

	// Aquí se debería devolver la compra completa, similar a ObtenerDetalleFactura
	return compra, nil
}
