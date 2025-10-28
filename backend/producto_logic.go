package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CrearProducto inserta un nuevo producto en la base de datos local.
func (d *Db) RegistrarProducto(producto Producto) (Producto, error) {
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return Producto{}, fmt.Errorf("no se pudo iniciar la transacción: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [RegistrarProducto] rollback %v", err)
		}
	}()

	var existente struct {
		UUID      string
		DeletedAt sql.NullTime
	}
	err = tx.QueryRowContext(d.ctx, "SELECT id, uuid, deleted_at FROM productos WHERE codigo = ?", producto.Codigo).Scan(&existente.UUID, &existente.DeletedAt)

	if err != nil && err != sql.ErrNoRows {
		return Producto{}, fmt.Errorf("error al verificar producto existente: %w", err)
	}

	switch {
	case err == nil:
		if existente.DeletedAt.Valid {
			// Restaurar producto eliminado
			d.Log.Infof("Restaurando producto eliminado con ID: %s", existente.UUID)
			update := `
				UPDATE productos
				SET nombre = ?, precio_venta = ?, stock = 0, deleted_at = NULL, updated_at = CURRENT_TIMESTAMP
				WHERE uuid = ?
			`
			if _, err := tx.Exec(update, producto.Nombre, producto.PrecioVenta, existente.UUID); err != nil {
				return Producto{}, fmt.Errorf("error al restaurar producto: %w", err)
			}
			producto.UUID = existente.UUID
			producto.Stock = 0
		} else {
			return Producto{}, fmt.Errorf("el código del producto ya está en uso")
		}

	case errors.Is(err, sql.ErrNoRows):
		// Crear nuevo producto
		insert := `
			INSERT INTO productos (uuid, nombre, codigo, precio_venta, stock, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`
		_, err := tx.Exec(insert, uuid.New().String(), producto.Nombre, producto.Codigo, producto.PrecioVenta, producto.Stock)
		if err != nil {
			return Producto{}, fmt.Errorf("error al registrar nuevo producto: %w", err)
		}

	default:
		return Producto{}, fmt.Errorf("error al buscar producto: %w", err)
	}

	// 2️⃣ Crear operación de stock INICIAL
	op := OperacionStock{
		UUID:           uuid.New().String(),
		TipoOperacion:  "INICIAL",
		CantidadCambio: producto.Stock,
		VendedorUUID:   "SYSTEM_ADMIN",
		Timestamp:      time.Now(),
		Sincronizado:   false,
	}

	opInsert := `
		INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, timestamp, sincronizado)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	if _, err := tx.Exec(opInsert, op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio, producto.Stock, op.VendedorUUID, op.Timestamp, op.Sincronizado); err != nil {
		return Producto{}, fmt.Errorf("error al crear operación de stock inicial: %w", err)
	}

	// 3️⃣ Recalcular y actualizar stock
	if err := RecalcularYActualizarStock(tx, producto.UUID); err != nil {
		return Producto{}, fmt.Errorf("error al calcular stock inicial: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Producto{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncProductoToRemote(producto.UUID)
	go d.SincronizarOperacionesStockHaciaRemoto()

	return producto, nil
}

// ObtenerProductosPaginado recupera una lista paginada de productos con búsqueda.
// EliminarProducto realiza un borrado lógico (soft delete) de un producto.
func (d *Db) EliminarProducto(uuid string) error {
	query := "UPDATE productos SET deleted_at = ? WHERE uuid = ?"

	_, err := d.LocalDB.Exec(query, time.Now(), uuid)
	if err != nil {
		return fmt.Errorf("error al eliminar producto: %w", err)
	}

	go d.syncProductoToRemote(uuid)
	return nil
}

// ActualizarProducto modifica los datos de un producto existente.
func (d *Db) ActualizarProducto(p Producto) (string, error) {
	if p.UUID == "" {
		return "", fmt.Errorf("se requiere un UUID de producto válido para actualizar")
	}

	tx, err := d.LocalDB.Begin()
	if err != nil {
		return "", fmt.Errorf("error al iniciar la transacción: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [ActualizarProducto] rollback %v", err)
		}
	}()

	// 1️⃣ Obtener stock real actual (sumatoria)
	var stockRealActual int
	err = tx.QueryRowContext(d.ctx, `SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_uuid = ?`, p.UUID).
		Scan(&stockRealActual)
	if err != nil {
		return "", fmt.Errorf("error al obtener stock real para producto %s: %w", p.UUID, err)
	}

	// 2️⃣ Actualizar campos base del producto
	update := `
		UPDATE productos
		SET nombre = ?, precio_venta = ?, updated_at = CURRENT_TIMESTAMP
		WHERE uuid = ?
	`
	if _, err := tx.ExecContext(d.ctx, update, p.Nombre, p.PrecioVenta, p.UUID); err != nil {
		return "", fmt.Errorf("error al actualizar producto: %w", err)
	}

	// 3️⃣ Registrar operación de ajuste si hay diferencia
	cambio := p.Stock - stockRealActual
	if cambio != 0 {
		d.Log.Infof("Ajuste de stock para producto UUID %s. Stock real: %d, Stock deseado: %d, Cambio: %d",
			p.UUID, stockRealActual, p.Stock, cambio)

		op := OperacionStock{
			UUID:           uuid.New().String(),
			ProductoUUID:   p.UUID,
			TipoOperacion:  "AJUSTE",
			CantidadCambio: cambio,
			VendedorUUID:   "SYSTEM_ADMIN",
			Timestamp:      time.Now(),
			Sincronizado:   false,
		}

		ajuste := `
			INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, vendedor_uuid, timestamp, sincronizado)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		if _, err := tx.Exec(ajuste, op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio, op.VendedorUUID, op.Timestamp, op.Sincronizado); err != nil {
			return "", fmt.Errorf("error al crear operación de ajuste: %w", err)
		}
	}

	// 4️⃣ Recalcular stock
	if err := RecalcularYActualizarStock(tx, p.UUID); err != nil {
		return "", fmt.Errorf("error final al recalcular stock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("error al confirmar la transacción: %w", err)
	}

	go d.syncProductoToRemote(p.UUID)
	go d.SincronizarOperacionesStockHaciaRemoto()

	return "Producto actualizado correctamente.", nil
}

func (d *Db) ObtenerProductosPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var productos []Producto
	var total int64

	baseQuery := "FROM productos WHERE deleted_at IS NULL"
	var whereClause string
	var args []interface{}

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		whereClause = " AND (LOWER(nombre) LIKE ? OR LOWER(codigo) LIKE ?)" // Espacio al inicio
		args = append(args, searchTerm, searchTerm)
	}

	countQuery := "SELECT COUNT(uuid) " + baseQuery + whereClause
	if err := d.LocalDB.QueryRowContext(d.ctx, countQuery, args...).Scan(&total); err != nil {
		return PaginatedResult{}, fmt.Errorf("error al contar productos: %w", err)
	}

	selectQuery := "SELECT uuid, codigo, nombre, precio_venta, stock " + baseQuery + whereClause

	if sortBy != "" {
		order := "ASC"
		if strings.ToLower(sortOrder) == "desc" {
			order = "DESC"
		}
		// Validar columnas para evitar inyección SQL
		allowedSortBy := map[string]string{"Nombre": "nombre", "Codigo": "codigo", "PrecioVenta": "precio_venta", "Stock": "stock"}
		if col, ok := allowedSortBy[sortBy]; ok {
			selectQuery += fmt.Sprintf(" ORDER BY %s %s", col, order)
		}
	} else {
		selectQuery += " ORDER BY nombre ASC" // Orden por defecto
	}

	offset := (page - 1) * pageSize
	selectQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := d.LocalDB.QueryContext(d.ctx, selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al obtener productos paginados: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.UUID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock); err != nil {
			return PaginatedResult{}, fmt.Errorf("error al escanear producto: %w", err)
		}
		productos = append(productos, p)
	}

	return PaginatedResult{Records: productos, TotalRecords: total}, nil
}

// ObtenerProductoPorUUID busca un producto por su UUID.
func (d *Db) ObtenerProductoPorUUID(uuid string) (Producto, error) {
	var p Producto
	query := "SELECT uuid, codigo, nombre, precio_venta, stock FROM productos WHERE uuid = ? AND deleted_at IS NULL"

	err := d.LocalDB.QueryRow(query, uuid).Scan(&p.UUID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock)
	if err != nil {
		return Producto{}, fmt.Errorf("error al buscar producto por UUID %s: %w", uuid, err)
	}

	return p, nil
}

func (d *Db) ObtenerHistorialStock(productoUUID string) ([]OperacionStock, error) {

	query := `
		SELECT 
			uuid, producto_uuid, tipo_operacion, cantidad_cambio, 
			stock_resultante, vendedor_uuid, factura_uuid, timestamp, sincronizado
		FROM 
			operacion_stocks
		WHERE 
			producto_uuid = ?
		ORDER BY 
			timestamp DESC
	`

	rows, err := d.LocalDB.QueryContext(d.ctx, query, productoUUID)
	if err != nil {
		return []OperacionStock{}, fmt.Errorf("error al ejecutar la consulta de historial de stock: %w", err)
	}
	defer rows.Close()

	var historial []OperacionStock
	for rows.Next() {
		var op OperacionStock

		err := rows.Scan(
			&op.UUID,
			&op.ProductoUUID,
			&op.TipoOperacion,
			&op.CantidadCambio,
			&op.StockResultante,
			&op.VendedorUUID,
			&op.FacturaUUID,
			&op.Timestamp,
			&op.Sincronizado,
		)

		if err != nil {
			d.Log.Errorf("Error al escanear una fila del historial de stock: %v", err)
			continue // Opcional: podrías devolver el error si prefieres que la operación falle por completo.
		}

		historial = append(historial, op)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error durante la iteración del historial de stock: %w", err)
	}

	return historial, nil
}

func (d *Db) ActualizarStockMasivo(ajustes []AjusteStockRequest) (string, error) {
	if len(ajustes) == 0 {
		return "No hay ajustes para procesar.", nil
	}
	tx, err := d.LocalDB.BeginTx(d.ctx, nil)
	if err != nil {
		return "", fmt.Errorf("error al iniciar la transacción masiva: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [ActualizarStockMasivo] rollback %v", err)
		}
	}()
	// 1. Preparar IDs para la consulta en lote.
	productoUUIDs := make([]string, 0, len(ajustes))
	mapaAjustes := make(map[string]int, len(ajustes))
	for _, a := range ajustes {
		productoUUIDs = append(productoUUIDs, a.ProductoUUID)
		mapaAjustes[a.ProductoUUID] = a.NuevoStock
	}

	// Implementación de consulta IN (...) para SQLite
	args := make([]interface{}, len(productoUUIDs))
	for i, id := range productoUUIDs {
		args[i] = id
	}
	query := `SELECT producto_uuid, COALESCE(SUM(cantidad_cambio), 0)
			  FROM operacion_stocks
			  WHERE producto_uuid IN (?` + strings.Repeat(",?", len(productoUUIDs)-1) + `)
			  GROUP BY producto_uuid`

	// 2. Obtener stocks reales actuales en una sola consulta.
	rows, err := tx.QueryContext(d.ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("error al obtener stocks reales en lote: %w", err)
	}
	defer rows.Close()

	stocksReales := make(map[string]int)
	for rows.Next() {
		var uuid string
		var stock int
		if err := rows.Scan(&uuid, &stock); err != nil {
			return "", err
		}
		stocksReales[uuid] = stock
	}

	// 3. Preparar la sentencia para la inserción en lote de ajustes.
	stmt, err := tx.PrepareContext(d.ctx, `
		INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, vendedor_uuid, timestamp) 
		VALUES (?, ?, 'AJUSTE', ?, 1, ?)`)
	if err != nil {
		return "", fmt.Errorf("error al preparar la inserción de ajustes: %w", err)
	}
	defer stmt.Close()

	// 4. Calcular cambios y ejecutar la inserción en lote.
	for _, uuid := range productoUUIDs {
		cantidadCambio := mapaAjustes[uuid] - stocksReales[uuid]
		if cantidadCambio != 0 {
			if _, err := stmt.ExecContext(d.ctx, uuid, uuid, cantidadCambio, time.Now()); err != nil {
				return "", fmt.Errorf("error al insertar ajuste para producto UUID %s: %w", uuid, err)
			}
		}
	}

	// 5. ¡Uso de la función auxiliar en bucle para cada producto afectado!
	for _, id := range productoUUIDs {
		if err := RecalcularYActualizarStock(tx, id); err != nil {
			return "", fmt.Errorf("error al recalcular stock en lote para producto ID %s: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("error al confirmar la transacción masiva: %w", err)
	}

	go d.SincronizarOperacionesStockHaciaRemoto()

	return "Stock actualizado masivamente.", nil
}
