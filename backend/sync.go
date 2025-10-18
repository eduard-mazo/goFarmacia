package backend

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
}

func (d *Db) SincronizacionInteligente() {
	if !d.syncMutex.TryLock() {
		d.Log.Warn("La sincronización inteligente ya está en proceso. Omitiendo esta ejecución.")
		return
	}
	defer d.syncMutex.Unlock()

	if !d.isRemoteDBAvailable() {
		d.Log.Warn("Modo offline: la base de datos remota no está disponible, se omite la sincronización.")
		return
	}
	d.Log.Info("[INICIO]: Sincronización Inteligente")

	g, ctx := errgroup.WithContext(d.ctx)

	// Sincronizar tablas maestras
	g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "vendedors") })
	g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "clientes") })
	g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "proveedors") })
	g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "productos") })

	if err := g.Wait(); err != nil {
		d.Log.Errorf("Error crítico durante la sincronización de modelos maestros: %v. La sincronización se detendrá.", err)
		return
	}
	d.Log.Info("Sincronización de datos maestros completada exitosamente.")

	if err := d.sincronizarOperacionesStockHaciaLocal(); err != nil {
		d.Log.Errorf("Error sincronizando operaciones de stock hacia local: %v", err)
	}

	// Sincronizar transacciones
	if err := d.sincronizarTransaccionesHaciaLocal(); err != nil {
		d.Log.Errorf("Error sincronizando transacciones hacia local: %v", err)
	}

	d.Log.Info("[FIN]: Sincronización Inteligente")
}

// sincronizarTablaMaestra sincroniza una tabla maestra basada en updated_at y sync_log.
func (d *Db) sincronizarTablaMaestra(ctx context.Context, tableName string) error {
	d.Log.Infof("[SINCRONIZANDO MODELO]: %s", tableName)

	var lastSyncTime time.Time
	syncLogQuery := `SELECT last_sync_timestamp FROM sync_log WHERE model_name = ?`
	err := d.LocalDB.QueryRowContext(ctx, syncLogQuery, tableName).Scan(&lastSyncTime)
	if err != nil && err != sql.ErrNoRows {
		d.Log.Errorf("[%s] Error al obtener el último timestamp de sincronización: %v", tableName, err)
		return err
	}

	// Decide la columna unique y el mapping de columnas para upserts
	switch tableName {
	case "vendedors":
		return d.syncGenericModel(ctx, tableName, "cedula", []string{"id", "created_at", "updated_at", "deleted_at", "nombre", "apellido", "cedula", "email", "contrasena", "mfa_enabled", "mfa_secret"})
	case "clientes":
		return d.syncGenericModel(ctx, tableName, "numero_id", []string{"id", "created_at", "updated_at", "deleted_at", "nombre", "apellido", "tipo_id", "numero_id", "telefono", "email", "direccion"})
	case "proveedors":
		return d.syncGenericModel(ctx, tableName, "nombre", []string{"id", "created_at", "updated_at", "deleted_at", "nombre", "telefono", "email"})
	case "productos":
		return d.syncGenericModel(ctx, tableName, "codigo", []string{"id", "created_at", "updated_at", "deleted_at", "nombre", "codigo", "precio_venta", "stock"})
	default:
		return fmt.Errorf("sincronización no implementada para la tabla: %s", tableName)
	}
}

func (d *Db) syncGenericModel(ctx context.Context, tableName, uniqueCol string, cols []string) error {
	d.Log.Infof("[%s] Inicio syncGenericModel (unique: %s)", tableName, uniqueCol)

	syncStartTime := time.Now().UTC() // <-- NUEVO: Marcar el inicio de la sincronización

	// 1) Obtener last_sync_timestamp
	var lastSync time.Time
	err := d.LocalDB.QueryRowContext(ctx, `SELECT last_sync_timestamp FROM sync_log WHERE model_name = ?`, tableName).Scan(&lastSync)
	if err != nil && err != sql.ErrNoRows {
		d.Log.Errorf("[%s] Error leyendo sync_log: %v", tableName, err)
		return err
	}
	d.Log.Infof("[%s] Última sincronización fue en: %v", tableName, lastSync)

	// --- Parte 1: Descargar cambios del Remoto al Local ---

	// Construir la consulta remota dinámicamente
	remoteQuery := fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ","), tableName)
	args := []interface{}{}
	if !lastSync.IsZero() {
		remoteQuery += " WHERE updated_at > $1"
		args = append(args, lastSync)
	}

	rows, err := d.RemoteDB.Query(ctx, remoteQuery, args...)
	if err != nil {
		d.Log.Errorf("[%s] Error consultando remotos: %v", tableName, err)
		return err
	}
	defer rows.Close()

	// Preparar transacción y statement para el UPSERT local
	txLocal, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("[%s] error iniciando tx local: %w", tableName, err)
	}
	defer txLocal.Rollback()

	updateAssignments := []string{}
	whereConditions := []string{}

	for _, col := range cols {
		if col != uniqueCol && col != "id" && col != "created_at" && col != "updated_at" {
			updateAssignments = append(updateAssignments, fmt.Sprintf("%s = excluded.%s", col, col))
			// Compara el valor actual de la tabla (main.<table>.<col>) con el valor "excluido" que se intentó insertar.
			whereConditions = append(whereConditions, fmt.Sprintf("main.%s.%s IS NOT excluded.%s", tableName, col, col))
		}
	}
	// Siempre intentamos actualizar 'updated_at' si hay otros cambios.
	updateAssignments = append(updateAssignments, "updated_at = excluded.updated_at")

	placeholders := "?"
	if len(cols) > 1 {
		placeholders = strings.Repeat("?,", len(cols)-1) + "?"
	}

	// Consulta UPSERT con un WHERE condicional para evitar updates innecesarios.
	upsertSQL := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s WHERE %s`,
		tableName,
		strings.Join(cols, ","),
		placeholders,
		uniqueCol,
		strings.Join(updateAssignments, ", "),
		strings.Join(whereConditions, " OR "), // El UPDATE solo se ejecuta si al menos un campo es diferente.
	)

	insStmt, err := txLocal.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return fmt.Errorf("[%s] error preparando upsert local: %w", tableName, err)
	}
	defer insStmt.Close()

	// Procesar y aplicar cambios remotos a local
	remoteRowCount := 0
	for rows.Next() {

		rawVals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range rawVals {
			valPtrs[i] = &rawVals[i]
		}

		if remoteRowCount == 0 && len(rawVals) > 1 { // Loguear solo el primer registro para no inundar la consola
			d.Log.Debugf("[%s] [LOG-1 Descarga] Procesando registro remoto: ID=%v, UniqueKey=%v, UpdatedAt=%v", tableName, rawVals[0], rawVals[1], rawVals[2])
		}

		if err := rows.Scan(valPtrs...); err != nil {
			d.Log.Errorf("[%s] Error escaneando fila remota: %v", tableName, err)
			continue
		}

		// Sanear la fila directamente sobre el slice
		if err := d.sanitizeRow(tableName, cols, rawVals); err != nil {
			d.Log.Warn(err.Error())
			continue // Ignorar fila con datos críticos faltantes
		}

		res, err := insStmt.ExecContext(ctx, rawVals...)
		if err != nil {
			d.Log.Errorf("[%s] Error en UPSERT de fila remota a local: %v", tableName, err)
			continue
		}
		if remoteRowCount == 0 {
			rowsAffected, _ := res.RowsAffected()
			d.Log.Debugf("[%s] [LOG-2 Resultado UPSERT] Filas afectadas por el primer registro: %d. (1 = Insertado/Actualizado, 0 = Ignorado)", tableName, rowsAffected)
		}
		remoteRowCount++
	}
	d.Log.Infof("[%s] Se recibieron y procesaron %d registros del servidor remoto.", tableName, remoteRowCount)

	// --- Parte 2: Subir cambios del Local al Remoto (CON BATCH) ---

	localQuery := fmt.Sprintf("SELECT %s FROM %s WHERE updated_at > ? AND updated_at < ?", strings.Join(cols, ","), tableName)
	localRows, err := txLocal.QueryContext(ctx, localQuery, lastSync, syncStartTime)
	if err != nil {
		return fmt.Errorf("[%s] error consultando locales para push: %w", tableName, err)
	}
	defer localRows.Close()

	// --- CAMBIO CLAVE: Usar pgx.Batch para subir todos los cambios en una sola operación ---
	batch := &pgx.Batch{}
	remotePlaceholders := make([]string, len(cols))
	for i := range cols {
		remotePlaceholders[i] = fmt.Sprintf("$%d", i+1)
	}
	remoteInsert := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		tableName, strings.Join(cols, ","), strings.Join(remotePlaceholders, ","), uniqueCol, strings.Join(updateAssignments, ","))

	localChangesCount := 0
	for localRows.Next() {

		values := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range values {
			valPtrs[i] = &values[i]
		}
		if remoteRowCount == 0 {
			d.Log.Warnf("[%s] [LOG-3 Subida] Se encontró un registro local modificado para subir, lo cual no debería ocurrir en una sincronización inicial. UniqueKey: %v, UpdatedAt: %v", tableName, valPtrs[1], valPtrs[2])
		}

		if err := localRows.Scan(valPtrs...); err != nil {
			d.Log.Errorf("[%s] Error escaneando fila local para push: %v", tableName, err)
			continue
		}
		batch.Queue(remoteInsert, values...)
		localChangesCount++
	}

	if localChangesCount > 0 {
		d.Log.Infof("[%s] Subiendo %d registros locales al servidor remoto...", tableName, localChangesCount)
		br := d.RemoteDB.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			d.Log.Errorf("[%s] Error ejecutando el lote de subida al remoto: %v", tableName, err)
			// No retornamos error aquí para permitir que la descarga se guarde, pero podrías querer hacerlo.
		}
	} else {
		d.Log.Infof("[%s] No hay cambios locales para subir.", tableName)
	}
	// --- FIN DEL CAMBIO CLAVE ---

	// 5) Actualizar sync_log con el timestamp actual
	_, err = txLocal.ExecContext(ctx, "INSERT INTO sync_log(model_name, last_sync_timestamp) VALUES (?, ?) ON CONFLICT(model_name) DO UPDATE SET last_sync_timestamp = ?", tableName, syncStartTime, syncStartTime)
	if err != nil {
		d.Log.Errorf("[%s] Error actualizando sync_log local: %v", tableName, err)
	}

	if err := txLocal.Commit(); err != nil {
		return fmt.Errorf("[%s] error confirmando tx local: %w", tableName, err)
	}

	d.Log.Infof("[%s] Sincronización completa.", tableName)
	return nil
}

// SincronizarOperacionesStockHaciaRemoto envía operaciones locales no sincronizadas al remoto.
func (d *Db) SincronizarOperacionesStockHaciaRemoto() {
	d.Log.Info("[SINCRONIZANDO]: Subiendo operaciones locales y recalculando stock remoto...")

	query := `SELECT id, uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp
			  FROM operacion_stocks WHERE sincronizado = 0`

	rows, err := d.LocalDB.QueryContext(d.ctx, query)
	if err != nil {
		d.Log.Errorf("error al obtener operaciones de stock locales: %v", err)
		return
	}
	defer rows.Close()

	var ops []OperacionStock
	var localIDsToUpdate []int64
	productosAfectados := make(map[uint]bool)

	for rows.Next() {
		var op OperacionStock
		var localID int64
		var stockResultante sql.NullInt64
		var facturaID sql.NullInt64

		if err := rows.Scan(&localID, &op.UUID, &op.ProductoID, &op.TipoOperacion, &op.CantidadCambio, &stockResultante, &op.VendedorID, &facturaID, &op.Timestamp); err != nil {
			d.Log.Warnf("Omitiendo operación de stock con error de escaneo: %v", err)
			continue
		}
		if stockResultante.Valid {
			op.StockResultante = int(stockResultante.Int64)
		}
		if facturaID.Valid {
			id := uint(facturaID.Int64)
			op.FacturaID = &id
		}
		ops = append(ops, op)
		localIDsToUpdate = append(localIDsToUpdate, localID)
		productosAfectados[op.ProductoID] = true
	}
	if err := rows.Err(); err != nil {
		d.Log.Errorf("error al iterar sobre las operaciones locales: %v", err)
		return
	}

	if len(ops) == 0 {
		d.Log.Info("No se encontraron nuevas operaciones de stock locales para sincronizar.")
		return
	}

	rtx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		d.Log.Errorf("no se pudo iniciar la transacción remota: %v", err)
		return
	}
	defer rtx.Rollback(d.ctx)

	// 1. Subir las nuevas operaciones de stock.
	_, err = rtx.CopyFrom(d.ctx,
		pgx.Identifier{"operacion_stocks"},
		[]string{"uuid", "producto_id", "tipo_operacion", "cantidad_cambio", "stock_resultante", "vendedor_id", "factura_id", "timestamp"},
		pgx.CopyFromSlice(len(ops), func(i int) ([]interface{}, error) {
			o := ops[i]
			return []interface{}{o.UUID, o.ProductoID, o.TipoOperacion, o.CantidadCambio, o.StockResultante, o.VendedorID, o.FacturaID, o.Timestamp}, nil
		}),
	)
	if err != nil && !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		d.Log.Errorf("error en inserción masiva de operaciones de stock: %v", err)
		return
	}

	// 2. CAMBIO CLAVE: Forzar la recalculación COMPLETA del stock en el servidor remoto.
	idsAfectados := make([]uint, 0, len(productosAfectados))
	for id := range productosAfectados {
		idsAfectados = append(idsAfectados, id)
	}

	updateStockCacheSQL := `
		UPDATE productos p
		SET stock = (
			SELECT COALESCE(SUM(os.cantidad_cambio), 0)
			FROM operacion_stocks os
			WHERE os.producto_id = p.id
		)
		WHERE p.id = ANY($1);
	`
	if _, err := rtx.Exec(d.ctx, updateStockCacheSQL, idsAfectados); err != nil {
		d.Log.Errorf("error al recalcular el stock remoto: %v", err)
		return
	}

	if err := rtx.Commit(d.ctx); err != nil {
		d.Log.Errorf("error al confirmar la transacción de operaciones de stock remotas: %v", err)
		return
	}

	// 3. Marcar como sincronizado localmente.
	if len(localIDsToUpdate) > 0 {
		updateQuery := fmt.Sprintf("UPDATE operacion_stocks SET sincronizado = 1 WHERE id IN (%s)", joinInt64s(localIDsToUpdate))
		if _, err := d.LocalDB.ExecContext(d.ctx, updateQuery); err != nil {
			d.Log.Errorf("error al marcar operaciones como sincronizadas: %v", err)
			return
		}
	}

	// 4. Devolver los IDs de los productos cuyo stock fue actualizado.
	d.Log.Infof("Sincronizadas %d operaciones. IDs de productos actualizados en remoto: %v", len(ops), idsAfectados)
}

func (d *Db) ForzarResincronizacionLocalDesdeRemoto() error {
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// BORRÓN Y CUENTA NUEVA LOCAL
	d.Log.Info("Limpiando tablas locales de stock y transacciones...")
	if _, err := tx.Exec("DELETE FROM operacion_stocks"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM detalle_facturas"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM facturas"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM sync_log"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Disparar una sincronización inteligente normal, que ahora descargará todo desde cero.
	d.SincronizacionInteligente()
	return nil
}

func (d *Db) sincronizarTransaccionesHaciaLocal() error {
	d.Log.Info("[SINCRONIZANDO]: Transacciones desde Remoto -> Local")
	ctx := d.ctx
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción local para transacciones: %w", err)
	}
	defer tx.Rollback()

	// --- Sincronizar Facturas ---
	var lastFacturaID int64

	err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM facturas").Scan(&lastFacturaID)
	if err != nil {
		return fmt.Errorf("error al obtener la última factura local: %w", err)
	}

	facturasRemotasQuery := `
		SELECT id, uuid, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at 
		FROM facturas 
		WHERE id > $1 
		ORDER BY id ASC
	`
	rows, err := d.RemoteDB.Query(ctx, facturasRemotasQuery, lastFacturaID)
	if err != nil {
		return fmt.Errorf("error obteniendo facturas remotas: %w", err)
	}
	defer rows.Close()

	facturaStmt, err := tx.PrepareContext(ctx, `
	INSERT INTO
		facturas (
			id,
			uuid,
			numero_factura,
			fecha_emision,
			vendedor_id,
			cliente_id,
			subtotal,
			iva,
			total,
			estado,
			metodo_pago,
			created_at,
			updated_at
		)
	VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (uuid) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("error al preparar statement de facturas: %w", err)
	}
	defer facturaStmt.Close()

	var facturasParaInsertar []Factura
	var facturaIDsRemotos []int64
	for rows.Next() {
		var f Factura
		var uuidStr sql.NullString
		if err := rows.Scan(&f.ID, &uuidStr, &f.NumeroFactura, &f.FechaEmision, &f.VendedorID, &f.ClienteID, &f.Subtotal, &f.IVA, &f.Total, &f.Estado, &f.MetodoPago, &f.CreatedAt, &f.UpdatedAt); err != nil {
			d.Log.Errorf("Error al escanear factura remota: %v", err)
			continue
		}
		if !uuidStr.Valid || uuidStr.String == "" {
			d.Log.Warnf("Factura remota con ID %d tiene un UUID nulo o vacío y será ignorada.", f.ID)
			continue
		}
		f.UUID = uuidStr.String
		facturasParaInsertar = append(facturasParaInsertar, f)
		facturaIDsRemotos = append(facturaIDsRemotos, int64(f.ID))
	}
	rows.Close() // Cerrar explícitamente antes de la siguiente consulta

	if len(facturasParaInsertar) > 0 {
		d.Log.Infof("Se encontraron %d nuevas facturas para sincronizar.", len(facturasParaInsertar))
		facturaStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO facturas (id, uuid, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (uuid) DO NOTHING`)
		if err != nil {
			return fmt.Errorf("error al preparar statement de facturas: %w", err)
		}
		defer facturaStmt.Close()

		for _, f := range facturasParaInsertar {
			// **LA CORRECCIÓN CLAVE ESTÁ AQUÍ**: Añadimos f.UUID
			_, err := facturaStmt.ExecContext(ctx, f.ID, f.UUID, f.NumeroFactura, f.FechaEmision, f.VendedorID, f.ClienteID, f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt)
			if err != nil {
				d.Log.Errorf("Error al insertar factura local (ID remoto: %d): %v", f.ID, err)
			}
		}

		// --- Ahora obtenemos y sincronizamos los detalles para estas facturas ---
		detallesFacturaQuery := `
			SELECT id, uuid, factura_id, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at 
			FROM detalle_facturas 
			WHERE factura_id = ANY($1)`
		detalleRows, err := d.RemoteDB.Query(ctx, detallesFacturaQuery, facturaIDsRemotos)
		if err != nil {
			return fmt.Errorf("error obteniendo detalles de factura remotos: %w", err)
		}
		defer detalleRows.Close()

		detalleStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO detalle_facturas (id, uuid, factura_id, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(uuid) DO NOTHING`)
		if err != nil {
			return fmt.Errorf("error al preparar statement de detalles de factura: %w", err)
		}
		defer detalleStmt.Close()

		detalleCount := 0
		for detalleRows.Next() {
			var df DetalleFactura
			var uuidDetalleStr sql.NullString
			if err := detalleRows.Scan(&df.ID, &uuidDetalleStr, &df.FacturaID, &df.ProductoID, &df.Cantidad, &df.PrecioUnitario, &df.PrecioTotal, &df.CreatedAt, &df.UpdatedAt); err != nil {
				d.Log.Errorf("Error al escanear detalle de factura remoto: %v", err)
				continue
			}
			if !uuidDetalleStr.Valid || uuidDetalleStr.String == "" {
				d.Log.Warnf("Detalle de factura remoto con ID %d tiene un UUID nulo o vacío y será ignorado.", df.ID)
				continue
			}
			df.UUID = uuidDetalleStr.String
			// **LA CORRECCIÓN CLAVE ESTÁ AQUÍ**: Añadimos df.UUID
			_, err := detalleStmt.ExecContext(ctx, df.ID, df.UUID, df.FacturaID, df.ProductoID, df.Cantidad, df.PrecioUnitario, df.PrecioTotal, df.CreatedAt, df.UpdatedAt)
			if err != nil {
				d.Log.Errorf("Error al insertar detalle de factura local (ID remoto: %d): %v", df.ID, err)
			}
			detalleCount++
		}
		d.Log.Infof("Sincronizados %d nuevos detalles de factura a local.", detalleCount)
	}
	var lastCompraID int64
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM compras").Scan(&lastCompraID)
	if err != nil {
		return fmt.Errorf("error al obtener el último ID de compra local: %w", err)
	}

	comprasRemotasQuery := `
		SELECT id, fecha, proveedor_id, factura_numero, total, created_at, updated_at 
		FROM compras 
		WHERE id > $1 
		ORDER BY id ASC`
	compraRows, err := d.RemoteDB.Query(ctx, comprasRemotasQuery, lastCompraID)
	if err != nil {
		return fmt.Errorf("error obteniendo compras remotas: %w", err)
	}
	defer compraRows.Close()

	compraStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO compras (id, fecha, proveedor_id, factura_numero, total, created_at, updated_at) 
		VALUES (?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("error al preparar statement de compras: %w", err)
	}
	defer compraStmt.Close()

	compraCount := 0
	for compraRows.Next() {
		var c Compra
		if err := compraRows.Scan(&c.ID, &c.Fecha, &c.ProveedorID, &c.FacturaNumero, &c.Total, &c.CreatedAt, &c.UpdatedAt); err != nil {
			d.Log.Errorf("Error al escanear compra remota: %v", err)
			continue
		}
		_, err := compraStmt.ExecContext(ctx, c.ID, c.Fecha, c.ProveedorID, c.FacturaNumero, c.Total, c.CreatedAt, c.UpdatedAt)
		if err != nil {
			d.Log.Errorf("Error al insertar compra local: %v", err)
		}
		compraCount++
	}
	if compraCount > 0 {
		d.Log.Infof("Sincronizadas %d nuevas compras a local.", compraCount)
	}

	return tx.Commit()
}

// --- FUNCIONES DE SINCRONIZACIÓN INDIVIDUAL ---

func (d *Db) syncVendedorToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var v Vendedor
	query := `SELECT id, created_at, updated_at, deleted_at, nombre, apellido, cedula, email, contrasena, mfa_enabled FROM vendedors WHERE id = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, id).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt, &v.Nombre, &v.Apellido, &v.Cedula, &v.Email, &v.Contrasena, &v.MFAEnabled)
	if err != nil {
		d.Log.Errorf("syncVendedorToRemote: no se encontró vendedor local ID %d: %v", id, err)
		return
	}

	upsertSQL := `
		INSERT INTO vendedors (id, created_at, updated_at, deleted_at, nombre, apellido, cedula, email, contrasena, mfa_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (cedula) DO UPDATE SET
			nombre = EXCLUDED.nombre,
			apellido = EXCLUDED.apellido,
			email = EXCLUDED.email,
			contrasena = EXCLUDED.contrasena,
			mfa_enabled = EXCLUDED.mfa_enabled,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, v.ID, v.CreatedAt, v.UpdatedAt, v.DeletedAt, v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de vendedor remoto ID %d: %v", id, err)
		return
	}
	d.Log.Infof("Sincronizado vendedor individual ID %d hacia el remoto.", id)
}

func (d *Db) syncClienteToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var c Cliente
	query := `SELECT id, created_at, updated_at, deleted_at, nombre, apellido, tipo_id, numero_id, telefono, email, direccion FROM clientes WHERE id = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, id).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt, &c.Nombre, &c.Apellido, &c.TipoID, &c.NumeroID, &c.Telefono, &c.Email, &c.Direccion)
	if err != nil {
		d.Log.Errorf("syncClienteToRemote: no se encontró cliente local ID %d: %v", id, err)
		return
	}

	upsertSQL := `
		INSERT INTO clientes (id, created_at, updated_at, deleted_at, nombre, apellido, tipo_id, numero_id, telefono, email, direccion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (numero_id) DO UPDATE SET
			nombre = EXCLUDED.nombre, apellido = EXCLUDED.apellido, tipo_id = EXCLUDED.tipo_id, telefono = EXCLUDED.telefono, email = EXCLUDED.email, direccion = EXCLUDED.direccion,
			updated_at = EXCLUDED.updated_at, deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, c.ID, c.CreatedAt, c.UpdatedAt, c.DeletedAt, c.Nombre, c.Apellido, c.TipoID, c.NumeroID, c.Telefono, c.Email, c.Direccion)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de cliente remoto ID %d: %v", id, err)
		return
	}
	d.Log.Infof("Sincronizado cliente individual ID %d hacia el remoto.", id)
}

func (d *Db) syncProductoToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var p Producto
	query := `SELECT id, created_at, updated_at, deleted_at, nombre, codigo, precio_venta FROM productos WHERE id = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, id).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Nombre, &p.Codigo, &p.PrecioVenta)
	if err != nil {
		d.Log.Errorf("syncProductoToRemote: no se encontró producto local ID %d: %v", id, err)
		return
	}

	upsertSQL := `
		INSERT INTO productos (id, created_at, updated_at, deleted_at, nombre, codigo, precio_venta)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (codigo) DO UPDATE SET
			nombre = EXCLUDED.nombre, 
			precio_venta = EXCLUDED.precio_venta,
			updated_at = EXCLUDED.updated_at, 
			deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, p.ID, p.CreatedAt, p.UpdatedAt, p.DeletedAt, p.Nombre, p.Codigo, p.PrecioVenta)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de datos maestros del producto remoto ID %d: %v", id, err)
		return
	}
	d.Log.Infof("Sincronizado datos maestros del producto ID %d. El stock se calculará por separado.", id)
}

func (d *Db) syncProveedorToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var p Proveedor
	query := `SELECT id, created_at, updated_at, deleted_at, nombre, telefono, email FROM proveedors WHERE id = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, id).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Nombre, &p.Telefono, &p.Email)
	if err != nil {
		d.Log.Errorf("syncProveedorToRemote: no se encontró proveedor local ID %d: %v", id, err)
		return
	}

	upsertSQL := `
		INSERT INTO proveedors (id, created_at, updated_at, deleted_at, nombre, telefono, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (nombre) DO UPDATE SET
			telefono = EXCLUDED.telefono, email = EXCLUDED.email,
			updated_at = EXCLUDED.updated_at, deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, p.ID, p.CreatedAt, p.UpdatedAt, p.DeletedAt, p.Nombre, p.Telefono, p.Email)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de proveedor remoto ID %d: %v", id, err)
		return
	}
	d.Log.Infof("Sincronizado proveedor individual ID %d hacia el remoto.", id)
}

// syncVentaToRemote: sincroniza una factura + detalles + operaciones de stock relacionadas.
func (d *Db) syncVentaToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	d.Log.Infof("Sincronizando venta individual ID %d hacia el remoto.", id)

	f, err := d.ObtenerDetalleFactura(id)
	if err != nil {
		d.Log.Errorf("syncVentaToRemote: no se pudo obtener factura local ID %d: %v", id, err)
		return
	}

	d.syncVendedorToRemote(f.VendedorID)
	d.syncClienteToRemote(f.ClienteID)

	rtx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		d.Log.Errorf("syncVentaToRemote: no se pudo iniciar tx remota: %v", err)
		return
	}
	defer rtx.Rollback(d.ctx)

	upsertFactura := `
		INSERT INTO facturas (uuid, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (uuid) DO UPDATE SET
			numero_factura = EXCLUDED.numero_factura, fecha_emision = EXCLUDED.fecha_emision, vendedor_id = EXCLUDED.vendedor_id,
			cliente_id = EXCLUDED.cliente_id, subtotal = EXCLUDED.subtotal, iva = EXCLUDED.iva, total = EXCLUDED.total,
			estado = EXCLUDED.estado, metodo_pago = EXCLUDED.metodo_pago, updated_at = EXCLUDED.updated_at
		RETURNING id;
	`
	var remoteFacturaID int64
	err = rtx.QueryRow(d.ctx, upsertFactura,
		f.UUID,
		f.NumeroFactura, f.FechaEmision, int64(f.VendedorID), int64(f.ClienteID), f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt,
	).Scan(&remoteFacturaID)

	if err != nil {
		d.Log.Errorf("syncVentaToRemote: error upserting factura remota: %v", err)
		return
	}

	if _, err := rtx.Exec(d.ctx, "DELETE FROM detalle_facturas WHERE factura_id = $1", remoteFacturaID); err != nil {
		d.Log.Errorf("syncVentaToRemote: error borrando detalles remotos: %v", err)
		return
	}
	if len(f.Detalles) > 0 {
		batch := &pgx.Batch{}
		upsertDetalleSQL := `
			INSERT INTO detalle_facturas (uuid, factura_id, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (uuid) DO UPDATE SET
				cantidad = EXCLUDED.cantidad,
				precio_unitario = EXCLUDED.precio_unitario,
				precio_total = EXCLUDED.precio_total,
				updated_at = EXCLUDED.updated_at;
		`

		for _, det := range f.Detalles {
			createdAt := det.CreatedAt
			if createdAt.IsZero() {
				createdAt = f.CreatedAt
			}
			updatedAt := det.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = f.UpdatedAt
			}

			batch.Queue(upsertDetalleSQL, det.UUID, remoteFacturaID, det.ProductoID, det.Cantidad, det.PrecioUnitario, det.PrecioTotal, createdAt, updatedAt)
		}

		br := rtx.SendBatch(d.ctx, batch)
		if err := br.Close(); err != nil {
			d.Log.Errorf("syncVentaToRemote: error ejecutando batch de detalles para factura %d: %v", f.ID, err)
			return
		}
		d.Log.Infof("syncVentaToRemote: Batch de %d detalles procesado para factura %d.", len(f.Detalles), f.ID)
	}

	opsRows, err := d.LocalDB.QueryContext(d.ctx, "SELECT uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp FROM operacion_stocks WHERE factura_id = ?", id)
	if err == nil {
		var localOps []OperacionStock
		for opsRows.Next() {
			var op OperacionStock
			var facturaID sql.NullInt64
			if err := opsRows.Scan(&op.UUID, &op.ProductoID, &op.TipoOperacion, &op.CantidadCambio, &op.StockResultante, &op.VendedorID, &facturaID, &op.Timestamp); err != nil {
				d.Log.Errorf("syncVentaToRemote: error scanning operacion local: %v", err)
				continue
			}
			if facturaID.Valid {
				fid := uint(facturaID.Int64)
				op.FacturaID = &fid
			}
			localOps = append(localOps, op)
		}
		opsRows.Close()

		if len(localOps) > 0 {
			_, err := rtx.CopyFrom(d.ctx,
				pgx.Identifier{"operacion_stocks"},
				[]string{"uuid", "producto_id", "tipo_operacion", "cantidad_cambio", "stock_resultante", "vendedor_id", "factura_id", "timestamp"},
				pgx.CopyFromSlice(len(localOps), func(i int) ([]interface{}, error) {
					op := localOps[i]
					var facturaID interface{}
					if op.FacturaID != nil {
						facturaID = int64(*op.FacturaID)
					} else {
						facturaID = nil
					}
					return []interface{}{op.UUID, int64(op.ProductoID), op.TipoOperacion, op.CantidadCambio, op.StockResultante, int64(op.VendedorID), facturaID, op.Timestamp}, nil
				}),
			)
			if err != nil && !strings.Contains(err.Error(), "duplicate key") {
				d.Log.Errorf("syncVentaToRemote: error copiando operacion_stocks a remoto: %v", err)
			}
			productoIDs := uniqueProductoIDsFromDetalles(f.Detalles)
			if len(productoIDs) > 0 {
				if _, err := rtx.Exec(d.ctx, `
					WITH stock_calculado AS (
						SELECT producto_id, COALESCE(SUM(cantidad_cambio),0) as nuevo_stock
						FROM operacion_stocks WHERE producto_id = ANY($1) GROUP BY producto_id
					)
					UPDATE productos p SET stock = sc.nuevo_stock FROM stock_calculado sc WHERE p.id = sc.producto_id;
				`, productoIDs); err != nil {
					d.Log.Errorf("syncVentaToRemote: error actualizando stock remoto para productos: %v", err)
				}
			}
		}
	}

	if err := rtx.Commit(d.ctx); err != nil {
		d.Log.Errorf("syncVentaToRemote: error confirmando tx remota: %v", err)
		return
	}
	d.Log.Infof("Factura ID %d sincronizada correctamente al remoto.", id)
}

// syncCompraToRemote: sincroniza una compra + detalles + operaciones de stock asociadas.
func (d *Db) syncCompraToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	d.Log.Infof("Sincronizando compra individual ID %d hacia el remoto.", id)

	// Obtener compra y detalles desde local
	var c Compra
	err := d.LocalDB.QueryRowContext(d.ctx, "SELECT id, fecha, proveedor_id, factura_numero, total, created_at, updated_at FROM compras WHERE id = ?", id).
		Scan(&c.ID, &c.Fecha, &c.ProveedorID, &c.FacturaNumero, &c.Total, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		d.Log.Errorf("syncCompraToRemote: no se encontró compra local ID %d: %v", id, err)
		return
	}

	// Asegurar proveedor en remoto
	d.syncProveedorToRemote(c.ProveedorID)

	// Recolectar detalles
	rows, err := d.LocalDB.QueryContext(d.ctx, "SELECT producto_id, cantidad, precio_compra_unitario FROM detalle_compras WHERE compra_id = ?", id)
	if err == nil {
		for rows.Next() {
			var det DetalleCompra
			if err := rows.Scan(&det.ProductoID, &det.Cantidad, &det.PrecioCompraUnitario); err != nil {
				d.Log.Errorf("syncCompraToRemote: error scanning detalle: %v", err)
				continue
			}
			c.Detalles = append(c.Detalles, det)
		}
		rows.Close()
	}

	// Upsert compra y detalles en remoto
	rtx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		d.Log.Errorf("syncCompraToRemote: no se pudo iniciar tx remota: %v", err)
		return
	}
	defer rtx.Rollback(d.ctx)

	_, err = rtx.Exec(d.ctx, `
		INSERT INTO compras (id, fecha, proveedor_id, factura_numero, total, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (id) DO UPDATE SET fecha = EXCLUDED.fecha, proveedor_id=EXCLUDED.proveedor_id, factura_numero=EXCLUDED.factura_numero, total=EXCLUDED.total, updated_at=EXCLUDED.updated_at
	`, int64(c.ID), c.Fecha, int64(c.ProveedorID), c.FacturaNumero, c.Total, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		d.Log.Errorf("syncCompraToRemote: error upserting compra remota: %v", err)
		return
	}

	// Reinsert detalles (borrar y copy)
	if _, err := rtx.Exec(d.ctx, "DELETE FROM detalle_compras WHERE compra_id = $1", int64(c.ID)); err != nil {
		d.Log.Errorf("syncCompraToRemote: error borrando detalles remotos: %v", err)
		// continue
	}
	if len(c.Detalles) > 0 {
		_, err := rtx.CopyFrom(d.ctx,
			pgx.Identifier{"detalle_compras"},
			[]string{"compra_id", "producto_id", "cantidad", "precio_compra_unitario"},
			pgx.CopyFromSlice(len(c.Detalles), func(i int) ([]interface{}, error) {
				det := c.Detalles[i]
				return []interface{}{int64(c.ID), int64(det.ProductoID), det.Cantidad, det.PrecioCompraUnitario}, nil
			}),
		)
		if err != nil {
			d.Log.Errorf("syncCompraToRemote: error reinsertando detalles remotos: %v", err)
		}
	}

	// Sincronizar operaciones de stock de la compra (si existieran)
	opsRows, err := d.LocalDB.QueryContext(d.ctx, "SELECT uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp FROM operacion_stocks WHERE factura_id IS NULL AND tipo_operacion = 'COMPRA' AND sincronizado = 0")
	if err == nil {
		var localOps []OperacionStock
		for opsRows.Next() {
			var op OperacionStock
			var facturaID sql.NullInt64
			if err := opsRows.Scan(&op.UUID, &op.ProductoID, &op.TipoOperacion, &op.CantidadCambio, &op.StockResultante, &op.VendedorID, &facturaID, &op.Timestamp); err != nil {
				d.Log.Errorf("syncCompraToRemote: error scanning operacion local: %v", err)
				continue
			}
			if facturaID.Valid {
				fid := uint(facturaID.Int64)
				op.FacturaID = &fid
			}
			localOps = append(localOps, op)
		}
		opsRows.Close()

		if len(localOps) > 0 {
			_, err := rtx.CopyFrom(d.ctx,
				pgx.Identifier{"operacion_stocks"},
				[]string{"uuid", "producto_id", "tipo_operacion", "cantidad_cambio", "stock_resultante", "vendedor_id", "factura_id", "timestamp"},
				pgx.CopyFromSlice(len(localOps), func(i int) ([]interface{}, error) {
					op := localOps[i]
					var facturaID interface{}
					if op.FacturaID != nil {
						facturaID = int64(*op.FacturaID)
					} else {
						facturaID = nil
					}
					return []interface{}{op.UUID, int64(op.ProductoID), op.TipoOperacion, op.CantidadCambio, op.StockResultante, int64(op.VendedorID), facturaID, op.Timestamp}, nil
				}),
			)
			if err != nil && !strings.Contains(err.Error(), "duplicate key") {
				d.Log.Errorf("syncCompraToRemote: error copiando operacion_stocks a remoto: %v", err)
			}
			// actualizar stock remoto para productos comprados
			prodIDs := uniqueProductoIDsFromDetallesCompra(c.Detalles)
			if len(prodIDs) > 0 {
				if _, err := rtx.Exec(d.ctx, `
					WITH stock_calculado AS (
						SELECT producto_id, COALESCE(SUM(cantidad_cambio),0) as nuevo_stock
						FROM operacion_stocks WHERE producto_id = ANY($1) GROUP BY producto_id
					)
					UPDATE productos p SET stock = sc.nuevo_stock FROM stock_calculado sc WHERE p.id = sc.producto_id;
				`, prodIDs); err != nil {
					d.Log.Errorf("syncCompraToRemote: error actualizando stock remoto: %v", err)
				}
			}
		}
	}

	if err := rtx.Commit(d.ctx); err != nil {
		d.Log.Errorf("syncCompraToRemote: error confirmando tx remota: %v", err)
		return
	}
	d.Log.Infof("Compra ID %d sincronizada correctamente al remoto.", id)
}

func (d *Db) syncVendedorToLocal(v Vendedor) {
	// upsert pattern for sqlite: try update, if rows affected==0 then insert
	res, err := d.LocalDB.Exec("UPDATE vendedors SET nombre=?, apellido=?, cedula=?, email=?, contrasena=?, mfa_enabled=?, updated_at=? WHERE id=?", v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled, time.Now(), v.ID)
	if err != nil {
		d.Log.Errorf("syncVendedorToLocal: error updating local vendedor ID %d: %v", v.ID, err)
		return
	}
	r, _ := res.RowsAffected()
	if r == 0 {
		_, err = d.LocalDB.Exec("INSERT INTO vendedors (id, nombre, apellido, cedula, email, contrasena, mfa_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", v.ID, v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled, time.Now(), time.Now())
		if err != nil {
			d.Log.Errorf("syncVendedorToLocal: error inserting local vendedor ID %d: %v", v.ID, err)
			return
		}
	}
}

func (d *Db) sincronizarOperacionesStockHaciaLocal() error {
	d.Log.Info("[SINCRONIZANDO]: Descargando nuevas Operaciones de Stock desde Remoto -> Local")
	ctx := d.ctx

	// 1. Encontrar la operación más reciente que tenemos localmente para saber desde dónde pedir.
	var lastSync time.Time
	var lastSyncStr sql.NullString
	err := d.LocalDB.QueryRowContext(ctx, "SELECT MAX(timestamp) FROM operacion_stocks").Scan(&lastSyncStr)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error obteniendo el último timestamp local de op. stock: %w", err)
	}

	if lastSyncStr.Valid && lastSyncStr.String != "" {
		// Usamos la función flexible que ya creamos para parsear la fecha de SQLite.
		if parsedTime, parseErr := parseFlexibleTime(lastSyncStr.String); parseErr == nil {
			lastSync = parsedTime
		}
	}
	// Si lastSync sigue en su valor cero, se descargarán todas las operaciones.

	// 2. Pedir al remoto (Postgres) todas las operaciones más nuevas que la última que tenemos.
	remoteOpsQuery := `
		SELECT uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp 
		FROM operacion_stocks 
		WHERE timestamp > $1 
		ORDER BY timestamp ASC`
	rows, err := d.RemoteDB.Query(ctx, remoteOpsQuery, lastSync)
	if err != nil {
		return fmt.Errorf("error obteniendo operaciones de stock remotas: %w", err)
	}
	defer rows.Close()

	var newOps []OperacionStock
	productosAfectados := make(map[uint]bool)

	for rows.Next() {
		var op OperacionStock
		var stockResultante sql.NullInt64
		var facturaID sql.NullInt64

		err := rows.Scan(
			&op.UUID,
			&op.ProductoID,
			&op.TipoOperacion,
			&op.CantidadCambio,
			&stockResultante,
			&op.VendedorID,
			&facturaID,
			&op.Timestamp,
		)
		if err != nil {
			d.Log.Warnf("Error al escanear una operación de stock remota, omitiendo: %v", err)
			continue
		}

		if stockResultante.Valid {
			op.StockResultante = int(stockResultante.Int64)
		}
		if facturaID.Valid {
			id := uint(facturaID.Int64)
			op.FacturaID = &id
		}

		newOps = append(newOps, op)
		productosAfectados[op.ProductoID] = true
	}

	if len(newOps) == 0 {
		d.Log.Info("La base de datos local de operaciones de stock ya está actualizada.")
		return nil
	}

	// 3. Insertar las nuevas operaciones en la base de datos local (SQLite) en una única transacción.
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción local para op. stock: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO operacion_stocks (uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp, sincronizado)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1) ON CONFLICT(uuid) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("error al preparar statement local de op. stock: %w", err)
	}
	defer stmt.Close()

	for _, op := range newOps {
		_, err := stmt.ExecContext(ctx, op.UUID, op.ProductoID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorID, op.FacturaID, op.Timestamp)
		if err != nil {
			// Logueamos el error pero intentamos continuar con las demás operaciones.
			d.Log.Errorf("Error al insertar op. stock local (UUID: %s): %v", op.UUID, err)
		}
	}

	// 4. Recalcular el stock local para todos los productos que recibieron nuevos movimientos.
	d.Log.Infof("Recalculando stock local para %d productos afectados...", len(productosAfectados))
	for id := range productosAfectados {
		if err := RecalcularYActualizarStock(tx, id); err != nil {
			// Este error es importante, si falla el recálculo, el stock local quedará inconsistente.
			d.Log.Errorf("CRÍTICO: Error al recalcular stock local para producto %d tras sincronización: %v. Se intentará continuar.", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar la transacción de sincronización de op. stock: %w", err)
	}

	d.Log.Infof("Sincronizadas %d nuevas operaciones de stock desde el remoto. Stock local actualizado.", len(newOps))
	return nil
}

func joinInt64s(nums []int64) string {
	if len(nums) == 0 {
		return ""
	}
	var b strings.Builder
	for i, n := range nums {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("%d", n))
	}
	return b.String()
}

func uniqueProductoIDsFromDetalles(detalles []DetalleFactura) []uint {
	m := make(map[uint]bool)
	for _, d := range detalles {
		m[d.ProductoID] = true
	}
	out := make([]uint, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func uniqueProductoIDsFromDetallesCompra(detalles []DetalleCompra) []uint {
	m := make(map[uint]bool)
	for _, d := range detalles {
		m[d.ProductoID] = true
	}
	out := make([]uint, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (d *Db) sanitizeRow(tableName string, cols []string, values []interface{}) error {
	now := time.Now().UTC()
	colMap := make(map[string]int, len(cols))
	for i, name := range cols {
		colMap[name] = i
	}

	// Función auxiliar para verificar campos clave
	checkRequired := func(key string) bool {
		idx, ok := colMap[key]
		if !ok {
			return false
		} // La columna no está en la selección
		if values[idx] == nil || strings.TrimSpace(fmt.Sprint(values[idx])) == "" {
			return true // Es nulo o vacío
		}
		return false
	}

	// Asignar valor por defecto si es nil
	setDefault := func(key string, defaultValue interface{}) {
		if idx, ok := colMap[key]; ok && values[idx] == nil {
			values[idx] = defaultValue
		}
	}

	// Validar campos únicos/requeridos
	uniqueKey := ""
	switch tableName {
	case "productos":
		uniqueKey = "codigo"
	case "clientes":
		uniqueKey = "numero_id"
	case "vendedors":
		uniqueKey = "cedula"
	case "proveedors":
		uniqueKey = "nombre"
	}
	if uniqueKey != "" && checkRequired(uniqueKey) {
		return fmt.Errorf("[%s] fila sin '%s', registro ignorado", tableName, uniqueKey)
	}

	// Asignar valores por defecto para campos comunes
	setDefault("created_at", now)
	setDefault("nombre", "SIN NOMBRE")

	// Asignar valores por defecto específicos de la tabla
	if tableName == "productos" {
		setDefault("precio_venta", 0.0)
		setDefault("stock", 0)
	}

	return nil
}

func parseFlexibleTime(dateStr string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05.999999-07:00", // Formato de SQLite con zona horaria
		"2006-01-02 15:04:05.999999Z07:00", // Formato común de PostgreSQL (UTC)
		"2006-01-02 15:04:05",              // Formato sin microsegundos ni zona horaria
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		parsedTime, err := time.Parse(layout, dateStr)
		if err == nil {
			return parsedTime, nil
		}
	}

	return time.Time{}, fmt.Errorf("no se pudo parsear la fecha '%s' con ninguno de los formatos conocidos", dateStr)
}
