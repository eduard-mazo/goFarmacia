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
	query := "SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_uuid = $1"

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
	updateQuery := "UPDATE productos SET stock = $1 WHERE uuid = $2"

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
	query := "SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_uuid = $1"
	err := tx.QueryRow(query, productoUUID).Scan(&stockCalculado)
	if err != nil {
		return 0, err
	}
	return stockCalculado, nil
}

// ---- LÓGICA DE TRANSACCIONES (VENTAS) REFACTORIZADA ----

func (d *Db) RegistrarVenta(req VentaRequest) (Factura, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return Factura{}, fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[RegistrarVenta] rollback: %v", rErr)
		}
	}()

	// 1. Generar número de factura
	numeroFactura, err := d.generarNumeroFactura(tx)
	if err != nil {
		return Factura{}, fmt.Errorf("error al generar número de factura: %w", err)
	}
	d.Log.Infof("[VENTA] Número de factura generado: %s", numeroFactura)

	now := time.Now()
	factura := Factura{
		UUID:          uuid.New().String(),
		NumeroFactura: numeroFactura,
		FechaEmision:  now,
		VendedorUUID:  req.VendedorUUID,
		ClienteUUID:   req.ClienteUUID,
		Estado:        "PAGADA",
		MetodoPago:    req.MetodoPago,
	}

	var subtotal float64
	var detalles []DetalleFactura

	// 2. Procesar productos
	stmtProd, err := tx.Prepare(`SELECT nombre, precio_venta FROM productos WHERE uuid = $1`)
	if err != nil {
		return Factura{}, fmt.Errorf("error preparando consulta productos: %w", err)
	}
	defer stmtProd.Close()

	for _, item := range req.Productos {
		var nombre string
		var precioVenta float64
		if err := stmtProd.QueryRow(item.ProductoUUID).Scan(&nombre, &precioVenta); err != nil {
			return Factura{}, fmt.Errorf("producto [%s] no encontrado: %w", item.ProductoUUID, err)
		}

		// 2.a Registrar operación de stock centralizada
		if err := d.CrearOperacionStock(
			tx,
			item.ProductoUUID,
			"VENTA",
			item.Cantidad,
			req.VendedorUUID,
			&factura.UUID,
		); err != nil {
			return Factura{}, fmt.Errorf("error registrando operación de stock [%s]: %w", nombre, err)
		}

		precioTotal := float64(item.Cantidad) * item.PrecioUnitario
		subtotal += precioTotal

		detalles = append(detalles, DetalleFactura{
			UUID:           uuid.New().String(),
			ProductoUUID:   item.ProductoUUID,
			Cantidad:       item.Cantidad,
			PrecioUnitario: item.PrecioUnitario,
			PrecioTotal:    precioTotal,
		})
	}

	factura.Subtotal = subtotal
	factura.IVA = subtotal * 0.0 // configurable si aplica
	factura.Total = factura.Subtotal + factura.IVA

	// 3. Insertar factura
	_, err = tx.Exec(`
		INSERT INTO facturas (
			uuid, numero_factura, fecha_emision, vendedor_uuid, cliente_uuid,
			subtotal, iva, total, estado, metodo_pago, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		factura.UUID, factura.NumeroFactura, factura.FechaEmision, factura.VendedorUUID,
		factura.ClienteUUID, factura.Subtotal, factura.IVA, factura.Total,
		factura.Estado, factura.MetodoPago, now, now)
	if err != nil {
		return Factura{}, fmt.Errorf("error insertando factura: %w", err)
	}

	// 4. Insertar detalles
	stmtDet, err := tx.Prepare(`
		INSERT INTO detalle_facturas (
			uuid, factura_uuid, producto_uuid, cantidad, precio_unitario, precio_total,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)
	if err != nil {
		return Factura{}, fmt.Errorf("error preparando statement detalle_facturas: %w", err)
	}
	defer stmtDet.Close()

	for _, det := range detalles {
		if _, err := stmtDet.Exec(det.UUID, factura.UUID, det.ProductoUUID,
			det.Cantidad, det.PrecioUnitario, det.PrecioTotal, now, now); err != nil {
			return Factura{}, fmt.Errorf("error insertando detalle %s: %w", det.ProductoUUID, err)
		}
	}

	// 5. Commit
	if err := tx.Commit(); err != nil {
		return Factura{}, fmt.Errorf("error confirmando transacción de venta: %w", err)
	}

	return d.ObtenerDetalleFactura(factura.UUID)
}

// Calcula el stock previo, el resultante y actualiza el producto dentro de la misma transacción.
func (d *Db) CrearOperacionStock(
	tx *sql.Tx,
	productoUUID string,
	tipoOperacion string,
	cambio int,
	vendedorUUID string,
	facturaUUID *string,
) error {

	if productoUUID == "" {
		return fmt.Errorf("[CrearOperacionStock] productoUUID vacío")
	}

	// 1. Obtener stock previo dentro de la misma transacción
	stockPrevio, err := calcularStockRealLocal(tx, productoUUID)
	if err != nil {
		return fmt.Errorf("[CrearOperacionStock] error obteniendo stock previo: %w", err)
	}

	// 2. Calcular nuevo stock resultante según tipo de operación
	var stockResultante int
	switch tipoOperacion {
	case "VENTA", "AJUSTE_NEGATIVO", "DEVOLUCION_CLIENTE":
		stockResultante = stockPrevio - cambio
		if stockResultante < 0 {
			return fmt.Errorf("stock insuficiente [%s] disponible %d solicitado %d",
				productoUUID, stockPrevio, cambio)
		}
	default: // COMPRA, AJUSTE_POSITIVO, DEVOLUCION_PROVEEDOR, INICIAL, etc.
		stockResultante = stockPrevio + cambio
	}

	// 3. Insertar operación de stock
	insertSQL := `
		INSERT INTO operacion_stocks (
			uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante,
			vendedor_uuid, factura_uuid, timestamp, sincronizado
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = tx.Exec(insertSQL,
		uuid.New().String(),
		productoUUID,
		tipoOperacion,
		cambio,
		stockResultante,
		vendedorUUID,
		facturaUUID,
		time.Now(),
		false,
	)
	if err != nil {
		return fmt.Errorf("[CrearOperacionStock] error insertando operación: %w", err)
	}

	// 4. Actualizar stock del producto
	_, err = tx.Exec(`
		UPDATE productos
		SET stock = $1, updated_at = CURRENT_TIMESTAMP
		WHERE uuid = $2`,
		stockResultante,
		productoUUID,
	)
	if err != nil {
		return fmt.Errorf("[CrearOperacionStock] error actualizando stock producto: %w", err)
	}

	d.Log.Debugf("[CrearOperacionStock] %s -> Stock previo %d, cambio %+d, nuevo %d",
		productoUUID, stockPrevio, cambio, stockResultante)

	return nil
}

func (d *Db) generarNumeroFactura(tx *sql.Tx) (string, error) {
	var maxNum sql.NullInt64

	query := `
		SELECT COALESCE(MAX(CAST(SUBSTR(numero_factura, 5) AS INTEGER)), 0)
		FROM facturas
		WHERE numero_factura LIKE 'FAC-%'`

	err := tx.QueryRow(query).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("error al consultar max numero_factura: %w", err)
	}

	nuevoNumero := 1000

	if maxNum.Valid && maxNum.Int64 > 0 {
		nuevoNumero = int(maxNum.Int64) + 1
	}

	if nuevoNumero < 1000 {
		nuevoNumero = 1000
	}

	return fmt.Sprintf("FAC-%d", nuevoNumero), nil
}

func (d *Db) ObtenerFacturasPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var (
		facturas []Factura
		args     []any
		where    string
	)

	// Base query join
	baseQuery := `
		FROM facturas f
		JOIN clientes c ON f.cliente_uuid = c.uuid
		JOIN vendedors v ON f.vendedor_uuid = v.uuid
	`

	argIdx := 1
	// Search filter
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		where = fmt.Sprintf(`
			WHERE LOWER(f.numero_factura) LIKE $%d
			   OR LOWER(c.nombre) LIKE $%d
			   OR LOWER(v.nombre) LIKE $%d
		`, argIdx, argIdx+1, argIdx+2)
		args = append(args, searchTerm, searchTerm, searchTerm)
		argIdx += 3
	}

	// Count total records
	var total int64
	countQuery := "SELECT COUNT(f.uuid) " + baseQuery + where

	if err := d.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return PaginatedResult{}, fmt.Errorf("Error contando facturas: %w", err)
	}

	// Safe ordering options
	allowedSortBy := map[string]string{
		"NumeroFactura": "f.numero_factura",
		"FechaEmision":  "f.fecha_emision",
		"Cliente":       "c.nombre",
		"Vendedor":      "v.nombre",
		"Total":         "f.total",
	}

	orderBy := "ORDER BY f.fecha_emision DESC, f.numero_factura DESC"
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "ASC"
		if strings.ToLower(sortOrder) == "desc" {
			order = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", col, order)
	}

	// Paginación
	offset := (page - 1) * pageSize
	pagination := fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	// Query final para obtener los registros
	selectQuery := `
		SELECT
			f.uuid,
			f.numero_factura,
			f.fecha_emision,
			f.total,
			c.uuid,
			c.nombre,
			v.uuid,
			v.nombre
	` + baseQuery + where + " " + orderBy + pagination

	rows, err := d.DB.Query(selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("Error realizando consulta facturas: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var f Factura
		if err := rows.Scan(
			&f.UUID, &f.NumeroFactura, &f.FechaEmision, &f.Total,
			&f.Cliente.UUID, &f.Cliente.Nombre,
			&f.Vendedor.UUID, &f.Vendedor.Nombre,
		); err != nil {
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
						f.uuid = $1
	`
	err := d.DB.QueryRow(queryFactura, facturaUUID).Scan(
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
		SELECT d.uuid, d.cantidad, d.precio_unitario, d.precio_total,
			p.uuid, p.codigo, p.nombre
		FROM detalle_facturas d
		JOIN productos p ON d.producto_uuid = p.uuid
		WHERE d.factura_uuid = $1
	`
	rows, err := d.DB.Query(queryDetalles, facturaUUID)
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
	tx, err := d.DB.Begin()
	if err != nil {
		return Compra{}, fmt.Errorf("error al iniciar transacción de compra: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [RegistrarCompra] rollback %v", rErr)
		}
	}()

	var totalCompra float64
	for _, p := range req.Productos {
		totalCompra += p.PrecioCompraUnitario * float64(p.Cantidad)
	}

	compra := Compra{
		UUID:          uuid.New().String(),
		Fecha:         time.Now(),
		ProveedorUUID: req.ProveedorUUID,
		FacturaNumero: req.FacturaNumero,
		Total:         totalCompra,
	}

	_, err = tx.Exec("INSERT INTO compras (uuid, fecha, proveedor_uuid, factura_numero, total) VALUES ($1, $2, $3, $4, $5)",
		compra.UUID, compra.Fecha, compra.ProveedorUUID, compra.FacturaNumero, compra.Total)
	if err != nil {
		return Compra{}, fmt.Errorf("error al crear la compra: %w", err)
	}

	// Preparar statements para inserciones masivas
	stmtDetalles, err := tx.Prepare("INSERT INTO detalle_compras (compra_uuid, producto_uuid, cantidad, precio_compra_unitario) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return Compra{}, err
	}
	defer stmtDetalles.Close()

	stmtOps, err := tx.Prepare("INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, vendedor_uuid, timestamp) VALUES ($1, $2, $3, $4, $5, $6)")
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

	return compra, nil
}
