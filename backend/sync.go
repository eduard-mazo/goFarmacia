package backend

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
	d.Log.Info("[INICIO]: Sincronización Inteligente (refactor)")

	g, ctx := errgroup.WithContext(d.ctx)

	// Sincronizar tablas maestras concurrenemente (no tocar stock/transacciones aquí)
	models := []struct {
		name      string
		uniqueCol string
		cols      []string
	}{
		{"vendedors", "cedula", []string{"id", "created_at", "updated_at", "deleted_at", "uuid", "nombre", "apellido", "cedula", "email", "contrasena", "mfa_enabled", "mfa_secret"}},
		{"clientes", "numero_id", []string{"id", "created_at", "updated_at", "deleted_at", "uuid", "nombre", "apellido", "tipo_id", "numero_id", "telefono", "email", "direccion"}},
		{"proveedors", "nombre", []string{"id", "created_at", "updated_at", "deleted_at", "uuid", "nombre", "telefono", "email"}},
		{"productos", "codigo", []string{"id", "created_at", "updated_at", "deleted_at", "uuid", "nombre", "codigo", "precio_venta", "stock"}},
	}

	for _, m := range models {
		m := m
		g.Go(func() error { return d.syncGenericModel(ctx, m.name, m.uniqueCol, m.cols) })
	}

	if err := g.Wait(); err != nil {
		d.Log.Errorf("Error durante la sincronización de modelos maestros: %v", err)
		return
	}
	d.Log.Info("Sincronización de datos maestros completada exitosamente.")

	if err := d.sincronizarTransaccionesHaciaLocal(); err != nil {
		d.Log.Errorf("Error sincronizando transacciones hacia local: %v", err)
	}
	if err := d.sincronizarOperacionesStockHaciaLocal(); err != nil {
		d.Log.Errorf("Error sincronizando operaciones de stock hacia local: %v", err)
	}

	// Subir operaciones locales pendientes (marcado atómico)
	d.SincronizarOperacionesStockHaciaRemoto()

	d.Log.Info("[FIN]: Sincronización Inteligente")
}

func (d *Db) syncGenericModel(ctx context.Context, tableName, uniqueCol string, cols []string) error {
	d.Log.Infof("[%s] Inicio syncGenericModel (unique: %s)", tableName, uniqueCol)
	syncStartTime := time.Now().UTC()

	// 1) lastSync
	var lastSync time.Time
	err := d.LocalDB.QueryRowContext(ctx, `SELECT last_sync_timestamp FROM sync_log WHERE model_name = ?`, tableName).Scan(&lastSync)
	if err != nil && err != sql.ErrNoRows {
		d.Log.Errorf("[%s] Error leyendo sync_log: %v", tableName, err)
		return err
	}
	d.Log.Infof("[%s] Última sincronización: %v", tableName, lastSync)

	// --- Descarga Remoto -> Local ---
	remoteCols := strings.Join(cols, ",")
	remoteQuery := fmt.Sprintf("SELECT %s FROM %s", remoteCols, tableName)
	args := []interface{}{}
	if !lastSync.IsZero() {
		remoteQuery += " WHERE updated_at > $1"
		args = append(args, lastSync)
	}
	rows, err := d.RemoteDB.Query(ctx, remoteQuery, args...)
	if err != nil {
		d.Log.Errorf("[%s] Error consultando remoto: %v", tableName, err)
		return err
	}
	defer rows.Close()

	txLocal, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("[%s] error iniciando tx local: %w", tableName, err)
	}

	placeholders := strings.Repeat("?,", len(cols)-1) + "?"
	upsertAssignments := []string{}
	for _, c := range cols {
		if c == "id" || c == uniqueCol || c == "created_at" {
			continue
		}
		if c == "updated_at" {
			upsertAssignments = append(upsertAssignments, "updated_at = excluded.updated_at")
		} else {
			upsertAssignments = append(upsertAssignments, fmt.Sprintf("%s = excluded.%s", c, c))
		}
	}
	upsertSQL := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s) 
		 ON CONFLICT(%s) DO UPDATE SET %s
		 WHERE excluded.updated_at > updated_at`,
		tableName, strings.Join(cols, ","), placeholders, uniqueCol, strings.Join(upsertAssignments, ", "),
	)
	insStmt, err := txLocal.PrepareContext(ctx, upsertSQL)
	if err != nil {
		if err := txLocal.Rollback(); err != nil {
			d.Log.Errorf("[REMOTO -> LOCAL] - Error durante [syncGenericModel] rollback %v", err)
		}
		return fmt.Errorf("[%s] error preparando upsert local: %w", tableName, err)
	}
	defer insStmt.Close()

	remoteCount := 0
	for rows.Next() {
		rawVals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range rawVals {
			valPtrs[i] = &rawVals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			d.Log.Errorf("[%s] Error escaneando fila remota: %v", tableName, err)
			continue
		}
		for i, colName := range cols {
			if colName == "uuid" {
				if uuidBytes, ok := rawVals[i].([16]uint8); ok {
					rawVals[i] = uuid.UUID(uuidBytes).String()
				}
			}
		}
		if err := d.sanitizeRow(tableName, cols, rawVals); err != nil {
			d.Log.Warn(err.Error())
			continue
		}
		if _, err := insStmt.ExecContext(ctx, rawVals...); err != nil {
			d.Log.Errorf("[%s] Error en UPSERT local (remoto->local): %v", tableName, err)
			continue
		}
		remoteCount++
	}
	d.Log.Infof("[%s] Se recibieron y procesaron %d registros del servidor remoto.", tableName, remoteCount)

	// --- Subida Local -> Remoto ---
	localQuery := fmt.Sprintf("SELECT %s FROM %s WHERE updated_at > ? AND updated_at < ?", strings.Join(cols, ","), tableName)
	localRows, err := txLocal.QueryContext(ctx, localQuery, lastSync, syncStartTime)
	if err != nil {
		txLocal.Rollback()
		return fmt.Errorf("[%s] error consultando locales para push: %w", tableName, err)
	}
	defer localRows.Close()

	remotePlaceholders := make([]string, len(cols))
	for i := range cols {
		remotePlaceholders[i] = fmt.Sprintf("$%d", i+1)
	}
	remoteAssignments := []string{}
	for _, c := range cols {
		if c == "id" || c == uniqueCol || c == "created_at" {
			continue
		}
		if c == "updated_at" {
			remoteAssignments = append(remoteAssignments, "updated_at = EXCLUDED.updated_at")
		} else {
			remoteAssignments = append(remoteAssignments, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
		}
	}
	remoteInsert := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s WHERE EXCLUDED.updated_at > %s.updated_at",
		tableName, strings.Join(cols, ","), strings.Join(remotePlaceholders, ","), uniqueCol, strings.Join(remoteAssignments, ", "), tableName,
	)

	var batch pgx.Batch
	localChanges := 0
	for localRows.Next() {
		values := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range values {
			valPtrs[i] = &values[i]
		}
		if err := localRows.Scan(valPtrs...); err != nil {
			d.Log.Errorf("[%s] Error escaneando fila local para push: %v", tableName, err)
			continue
		}
		batch.Queue(remoteInsert, values...)
		localChanges++
	}

	if localChanges > 0 {
		d.Log.Infof("[%s] Preparando subida de %d registros al remoto...", tableName, localChanges)
		br := d.RemoteDB.SendBatch(ctx, &batch)
		if err := br.Close(); err != nil {
			d.Log.Errorf("[%s] Error ejecutando batch al remoto: %v", tableName, err)
		}
	} else {
		d.Log.Infof("[%s] No hay cambios locales para subir.", tableName)
	}

	// Actualizar sync_log local (INSERT/UPDATE)
	_, err = txLocal.ExecContext(ctx, "INSERT INTO sync_log(model_name, last_sync_timestamp) VALUES (?, ?) ON CONFLICT(model_name) DO UPDATE SET last_sync_timestamp = ?", tableName, syncStartTime, syncStartTime)
	if err != nil {
		txLocal.Rollback()
		d.Log.Errorf("[%s] Error actualizando sync_log local: %v", tableName, err)
		return fmt.Errorf("[%s] error actualizando sync_log: %w", tableName, err)
	}

	if err := txLocal.Commit(); err != nil {
		txLocal.Rollback()
		return fmt.Errorf("[%s] error confirmando tx local: %w", tableName, err)
	}

	d.Log.Infof("[%s] Sincronización completa.", tableName)
	return nil
}

// SincronizarOperacionesStockHaciaRemoto envía operaciones locales no sincronizadas al remoto.
func (d *Db) SincronizarOperacionesStockHaciaRemoto() {
	d.Log.Info("[SINCRONIZANDO]: Subiendo operaciones locales y recalculando stock remoto")

	// 1) Leer ops locales no sincronizadas
	query := `SELECT id, uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp
			  FROM operacion_stocks WHERE sincronizado = 0`
	rows, err := d.LocalDB.QueryContext(d.ctx, query)
	if err != nil {
		d.Log.Errorf("error al obtener operaciones de stock locales: %v", err)
		return
	}
	defer rows.Close()

	type pendingOp struct {
		localID int64
		op      OperacionStock
	}
	var pending []pendingOp
	productosAfectados := map[uint]bool{}

	for rows.Next() {
		var localID sql.NullInt64
		var op OperacionStock
		var stockResult sql.NullInt64
		var facturaID sql.NullInt64
		if err := rows.Scan(&localID, &op.UUID, &op.ProductoID, &op.TipoOperacion, &op.CantidadCambio, &stockResult, &op.VendedorID, &facturaID, &op.Timestamp); err != nil {
			d.Log.Warnf("Omitiendo operación de stock con error de escaneo: %v", err)
			continue
		}
		if stockResult.Valid {
			op.StockResultante = int(stockResult.Int64)
		}
		if facturaID.Valid {
			id := uint(facturaID.Int64)
			op.FacturaID = &id
		}
		if !localID.Valid {
			continue
		}
		pending = append(pending, pendingOp{localID: localID.Int64, op: op})
		productosAfectados[op.ProductoID] = true
	}

	if len(pending) == 0 {
		d.Log.Info("No hay operaciones pendientes para subir.")
		return
	}

	// 2) Begin remote tx and bulk insert using CopyFrom (uuid used to ensure idempotency)
	rtx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		d.Log.Errorf("no se pudo iniciar la transacción remota: %v", err)
		return
	}
	go func() {
		if err := rtx.Rollback(d.ctx); err != nil {
			d.Log.Errorf("[LOCAL -> REMOTO] - Error durante rollback %v", err)
		}
	}()

	_, err = rtx.CopyFrom(d.ctx,
		pgx.Identifier{"operacion_stocks"},
		[]string{"uuid", "producto_id", "tipo_operacion", "cantidad_cambio", "stock_resultante", "vendedor_id", "factura_id", "timestamp"},
		pgx.CopyFromSlice(len(pending), func(i int) ([]interface{}, error) {
			o := pending[i].op
			var fid interface{}
			if o.FacturaID != nil {
				fid = int64(*o.FacturaID)
			} else {
				fid = nil
			}
			return []interface{}{o.UUID, int64(o.ProductoID), o.TipoOperacion, o.CantidadCambio, o.StockResultante, int64(o.VendedorID), fid, o.Timestamp}, nil
		}),
	)
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		d.Log.Errorf("error en inserción masiva de operaciones de stock: %v", err)
		return
	}

	// 3) Recalcular stock remoto de forma atomica en la tx remota
	ids := make([]uint, 0, len(productosAfectados))
	for id := range productosAfectados {
		ids = append(ids, id)
	}

	updateStockSQL := `
		UPDATE productos p
		SET stock = sub.nuevo_stock
		FROM (
			SELECT producto_id, COALESCE(SUM(cantidad_cambio),0) as nuevo_stock
			FROM operacion_stocks
			WHERE producto_id = ANY($1)
			GROUP BY producto_id
		) AS sub
		WHERE p.id = sub.producto_id;
	`
	if _, err := rtx.Exec(d.ctx, updateStockSQL, ids); err != nil {
		d.Log.Errorf("error al recalcular stock remoto: %v", err)
		return
	}

	// 4) Commit remoto
	if err := rtx.Commit(d.ctx); err != nil {
		d.Log.Errorf("error al confirmar tx remota: %v", err)
		return
	}

	// 5) Marcar locales como sincronizadas en un solo UPDATE atómico local
	localIDs := make([]int64, 0, len(pending))
	for _, p := range pending {
		localIDs = append(localIDs, p.localID)
	}
	if len(localIDs) > 0 {
		updateQuery := fmt.Sprintf("UPDATE operacion_stocks SET sincronizado = 1 WHERE id IN (%s)", joinInt64s(localIDs))
		if _, err := d.LocalDB.ExecContext(d.ctx, updateQuery); err != nil {
			d.Log.Errorf("error al marcar operaciones como sincronizadas localmente: %v", err)
			// no return: la subida ya se hizo, pero avisamos del fallo
		}
	}

	d.Log.Infof("Sincronizadas %d operaciones al remoto y actualizado stock para %d productos.", len(pending), len(ids))
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

// Reemplaza la implementación actual por esta versión revisada.
func (d *Db) sincronizarTransaccionesHaciaLocal() error {
	d.Log.Info("[SINCRONIZANDO]: Transacciones desde Remoto -> Local")
	ctx := d.ctx

	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción local para transacciones: %w", err)
	}
	defer tx.Rollback()

	// ---------------------------
	// --- 1) FACTURAS ----------
	// ---------------------------
	var lastFacturaID int64
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM facturas").Scan(&lastFacturaID); err != nil {
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

	// Preparar statement local (SQLite) para insertar facturas (idempotente por uuid)
	insertFactSQL := `
	INSERT INTO facturas (id, uuid, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(uuid) DO NOTHING
	`
	stmtFact, err := tx.PrepareContext(ctx, insertFactSQL)
	if err != nil {
		rows.Close()
		return fmt.Errorf("error preparando statement de facturas: %w", err)
	}
	defer stmtFact.Close()

	var facturas []Factura
	var facturaIDsRemotos []int64

	for rows.Next() {
		var f Factura
		var uuidStr sql.NullString
		if err := rows.Scan(&f.ID, &uuidStr, &f.NumeroFactura, &f.FechaEmision, &f.VendedorID, &f.ClienteID, &f.Subtotal, &f.IVA, &f.Total, &f.Estado, &f.MetodoPago, &f.CreatedAt, &f.UpdatedAt); err != nil {
			d.Log.Errorf("Error al escanear factura remota: %v", err)
			continue
		}
		if !uuidStr.Valid || strings.TrimSpace(uuidStr.String) == "" {
			d.Log.Warnf("Factura remota ID %d sin UUID -> ignorada.", f.ID)
			continue
		}
		f.UUID = uuidStr.String
		facturas = append(facturas, f)
		facturaIDsRemotos = append(facturaIDsRemotos, int64(f.ID))
	}
	// cerrar rows de facturas
	rows.Close()

	if len(facturas) > 0 {
		d.Log.Infof("Se encontraron %d nuevas facturas para sincronizar.", len(facturas))
		for _, f := range facturas {
			if _, err := stmtFact.ExecContext(ctx, f.ID, f.UUID, f.NumeroFactura, f.FechaEmision, f.VendedorID, f.ClienteID, f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt); err != nil {
				d.Log.Errorf("Error insertando factura local (ID remoto %d): %v", f.ID, err)
			}
		}

		// ---------------------------
		// --- 1.a) DETALLES -------
		// ---------------------------
		// Si no hay facturaIDsRemotos (por alguna razón) evitamos la query.
		if len(facturaIDsRemotos) > 0 {
			// Construir placeholders para la consulta remota: $1,$2,...
			placeholders := make([]string, len(facturaIDsRemotos))
			args := make([]interface{}, len(facturaIDsRemotos))
			for i, id := range facturaIDsRemotos {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
				args[i] = id
			}
			detallesFacturaQuery := fmt.Sprintf(`
				SELECT id, uuid, factura_id, factura_uuid, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at 
				FROM detalle_facturas 
				WHERE factura_id IN (%s)
				ORDER BY id ASC
			`, strings.Join(placeholders, ","))

			detalleRows, err := d.RemoteDB.Query(ctx, detallesFacturaQuery, args...)
			if err != nil {
				return fmt.Errorf("error obteniendo detalles de factura remotos: %w", err)
			}

			// Statement local para detalles: ON CONFLICT(uuid) DO NOTHING
			detalleStmt, err := tx.PrepareContext(ctx, `
				INSERT INTO detalle_facturas (id, uuid, factura_id, factura_uuid, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(uuid) DO NOTHING`)
			if err != nil {
				detalleRows.Close()
				return fmt.Errorf("error al preparar statement de detalles de factura: %w", err)
			}
			defer detalleStmt.Close()

			detalleCount := 0
			for detalleRows.Next() {
				var df DetalleFactura
				var uuidDetalleStr sql.NullString
				var uuidFacturaStr sql.NullString
				if err := detalleRows.Scan(&df.ID, &uuidDetalleStr, &df.FacturaID, &uuidFacturaStr, &df.ProductoID, &df.Cantidad, &df.PrecioUnitario, &df.PrecioTotal, &df.CreatedAt, &df.UpdatedAt); err != nil {
					d.Log.Errorf("Error al escanear detalle de factura remoto: %v", err)
					continue
				}
				if !uuidDetalleStr.Valid || strings.TrimSpace(uuidDetalleStr.String) == "" {
					d.Log.Warnf("Detalle de factura remoto con ID %d tiene un UUID nulo o vacío y será ignorado.", df.ID)
					continue
				}
				df.UUID = uuidDetalleStr.String

				if !uuidFacturaStr.Valid || strings.TrimSpace(uuidFacturaStr.String) == "" {
					d.Log.Warnf("Detalle de factura remoto con ID %d tiene un UUID nulo o vacío y será ignorado.", df.ID)
					continue
				}
				df.FacturaUUID = uuidFacturaStr.String

				if _, err := detalleStmt.ExecContext(ctx, df.ID, df.UUID, df.FacturaID, df.FacturaUUID, df.ProductoID, df.Cantidad, df.PrecioUnitario, df.PrecioTotal, df.CreatedAt, df.UpdatedAt); err != nil {
					d.Log.Errorf("Error al insertar detalle de factura local (ID remoto: %d): %v", df.ID, err)
					continue
				}
				detalleCount++
			}
			detalleRows.Close()
			d.Log.Infof("Sincronizados %d nuevos detalles de factura a local.", detalleCount)
		}
	}

	// ---------------------------
	// --- 2) COMPRAS ----------
	// ---------------------------
	// Nota: tu implementación original usa ON CONFLICT(id). Si compras puede crearse localmente,
	// considera añadir uuid a compras y usar ON CONFLICT(uuid) DO NOTHING para idempotencia.
	var lastCompraID int64
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM compras").Scan(&lastCompraID); err != nil {
		return fmt.Errorf("error al obtener el último ID de compra local: %w", err)
	}

	comprasRemotasQuery := `
		SELECT id, fecha, proveedor_id, factura_numero, total, created_at, updated_at 
		FROM compras 
		WHERE id > $1 
		ORDER BY id ASC
	`
	compraRows, err := d.RemoteDB.Query(ctx, comprasRemotasQuery, lastCompraID)
	if err != nil {
		return fmt.Errorf("error obteniendo compras remotas: %w", err)
	}
	defer compraRows.Close()

	// Mantengo ON CONFLICT(id) por compatibilidad, pero si agregas uuid en compras cambia a ON CONFLICT(uuid).
	compraStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO compras (id, fecha, proveedor_id, factura_numero, total, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`)
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
		if _, err := compraStmt.ExecContext(ctx, c.ID, c.Fecha, c.ProveedorID, c.FacturaNumero, c.Total, c.CreatedAt, c.UpdatedAt); err != nil {
			d.Log.Errorf("Error al insertar compra local: %v", err)
			continue
		}
		compraCount++
	}
	if compraCount > 0 {
		d.Log.Infof("Sincronizadas %d nuevas compras a local.", compraCount)
	}

	// ---------------------------
	// --- FIN: commit local ---
	// ---------------------------
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error confirmando transacción local de transacciones: %w", err)
	}

	d.Log.Info("Transacciones sincronizadas hacia local correctamente.")
	return nil
}

// --- FUNCIONES DE SINCRONIZACIÓN INDIVIDUAL ---

func (d *Db) syncVendedorToRemote(uuid string) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var v Vendedor
	query := `SELECT uuid, created_at, updated_at, deleted_at, nombre, apellido, cedula, email, contrasena, mfa_enabled FROM vendedors WHERE uuid = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, uuid).Scan(&v.UUID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt, &v.Nombre, &v.Apellido, &v.Cedula, &v.Email, &v.Contrasena, &v.MFAEnabled)
	if err != nil {
		d.Log.Errorf("[LOCAL] syncVendedorToRemote: no se encontró vendedor local UUID %d: %v", uuid, err)
		return
	}

	upsertSQL := `
		INSERT INTO vendedors (id, uuid, created_at, updated_at, deleted_at, nombre, apellido, cedula, email, contrasena, mfa_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (cedula) DO UPDATE SET
			nombre = EXCLUDED.nombre,
			apellido = EXCLUDED.apellido,
			email = EXCLUDED.email,
			contrasena = EXCLUDED.contrasena,
			mfa_enabled = EXCLUDED.mfa_enabled,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, v.ID, v.UUID, v.CreatedAt, v.UpdatedAt, v.DeletedAt, v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de vendedor remoto UUID %s: %v", uuid, err)
		return
	}
	d.Log.Infof("Sincronizado vendedor individual UUID %s hacia el remoto.", uuid)
}

func (d *Db) syncClienteToRemote(uuid string) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var c Cliente
	query := `SELECT uuid, created_at, updated_at, deleted_at, nombre, apellido, tipo_id, numero_id, telefono, email, direccion FROM clientes WHERE uuid = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, uuid).Scan(&c.UUID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt, &c.Nombre, &c.Apellido, &c.TipoID, &c.NumeroID, &c.Telefono, &c.Email, &c.Direccion)
	if err != nil {
		d.Log.Errorf("syncClienteToRemote: no se encontró cliente local ID %s: %v", uuid, err)
		return
	}

	upsertSQL := `
		INSERT INTO clientes (id, uuid, created_at, updated_at, deleted_at, nombre, apellido, tipo_id, numero_id, telefono, email, direccion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (numero_id) DO UPDATE SET
			nombre = EXCLUDED.nombre, apellido = EXCLUDED.apellido, tipo_id = EXCLUDED.tipo_id, telefono = EXCLUDED.telefono, email = EXCLUDED.email, direccion = EXCLUDED.direccion,
			updated_at = EXCLUDED.updated_at, deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, c.ID, c.UUID, c.CreatedAt, c.UpdatedAt, c.DeletedAt, c.Nombre, c.Apellido, c.TipoID, c.NumeroID, c.Telefono, c.Email, c.Direccion)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de cliente remoto UUID %d: %v", uuid, err)
		return
	}
	d.Log.Infof("Sincronizado cliente individual UUID %s hacia el remoto.", uuid)
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
// Sincroniza una venta (factura) local hacia el servidor remoto sin duplicar
func (d *Db) syncVentaToRemote(facturaUUID string) error {
	ctx := d.ctx
	d.Log.Infof("[LOCAL -> REMOTO] - Sincronizando venta factura UUID %s", facturaUUID)

	// 1) Obtener factura local
	var f Factura
	err := d.LocalDB.QueryRowContext(ctx, `
		SELECT uuid, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at
		FROM facturas WHERE uuid = ?`, facturaUUID).Scan(
		&f.UUID, &f.NumeroFactura, &f.FechaEmision, &f.VendedorID, &f.ClienteID,
		&f.Subtotal, &f.IVA, &f.Total, &f.Estado, &f.MetodoPago, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return fmt.Errorf("factura no encontrada localmente: %w", err)
	}

	// 2) Verificar si ya existe en remoto
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM facturas WHERE uuid = $1)`
	err = d.RemoteDB.QueryRow(ctx, checkQuery, f.UUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error verificando existencia de factura remota: %w", err)
	}
	if exists {
		d.Log.Infof("[REMOTO] - Factura UUID %s ya existe en remoto, se omite inserción.", f.UUID)
		_, _ = d.LocalDB.ExecContext(ctx, `UPDATE facturas SET sincronizado = 1 WHERE uuid = ?`, f.UUID)
		return nil
	}
	d.Log.Infof("[REMOTO] - Factura UUID %s no existe en remoto", f.UUID)
	// 3) Insertar factura en remoto (usa ON CONFLICT(uuid))
	insertFacturaSQL := `
		INSERT INTO facturas (uuid, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (uuid) DO UPDATE 
		SET updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.updated_at > facturas.updated_at
	`
	_, err = d.RemoteDB.Exec(ctx, insertFacturaSQL, f.UUID, f.NumeroFactura, f.FechaEmision, f.VendedorID, f.ClienteID, f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error insertando factura remota: %w", err)
	}
	d.Log.Infof("[REMOTO] - Se ejecutó INSERT remoto para Factura UUID %s", f.UUID)
	// 4) Subir los detalles (solo si la factura fue insertada)
	d.Log.Infof("[LOCAL] - consultando detalles_factura para Factura UUID %s", f.UUID)
	rows, err := d.LocalDB.QueryContext(ctx, `
		SELECT uuid, factura_id, factura_uuid, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at 
		FROM detalle_facturas WHERE factura_uuid = ?`, facturaUUID)
	if err != nil {
		return fmt.Errorf("error obteniendo detalles locales: %w", err)
	}
	defer rows.Close()

	tx, err := d.RemoteDB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error iniciando tx remota: %w", err)
	}
	go func() {
		if err := tx.Rollback(ctx); err != nil {
			d.Log.Errorf("[LOCAL -> REMOTO] - Error durante [syncVentaToRemote] rollback %v", err)
		}
	}()
	d.Log.Infof("[REMOTO] - Se prepara INSERT por batch para detalles_facturas de Factura UUID %s", f.UUID)
	batch := &pgx.Batch{}
	for rows.Next() {
		var df DetalleFactura
		if err := rows.Scan(&df.UUID, &df.FacturaID, &df.FacturaUUID, &df.ProductoID, &df.Cantidad, &df.PrecioUnitario, &df.PrecioTotal, &df.CreatedAt, &df.UpdatedAt); err != nil {
			d.Log.Errorf("Error al escanear detalle_factura remota: %v", err)
			continue
		}
		batch.Queue(`
			INSERT INTO detalle_facturas (uuid, factura_id, factura_uuid, producto_id, cantidad, precio_unitario, precio_total, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (uuid) DO UPDATE 
			SET updated_at = EXCLUDED.updated_at
			WHERE EXCLUDED.updated_at > detalle_facturas.updated_at`,
			df.UUID, df.FacturaID, df.FacturaUUID, df.ProductoID, df.Cantidad, df.PrecioUnitario, df.PrecioTotal, df.CreatedAt, df.UpdatedAt)
	}
	d.Log.Infof("[REMOTO] - Se envía INSERT por batch para detalles_facturas de Factura UUID %s", f.UUID)
	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("[REMOTO] - Error ejecutando batch de detalles_factura: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Error confirmando detalles remotos: %w", err)
	}

	// 5) Marcar factura y detalles locales como sincronizados
	_, _ = d.LocalDB.ExecContext(ctx, `UPDATE facturas SET sincronizado = 1 WHERE uuid = ?`, f.UUID)
	_, _ = d.LocalDB.ExecContext(ctx, `UPDATE detalle_facturas SET sincronizado = 1 WHERE factura_uuid = ?`, f.UUID)

	d.Log.Infof("[REMOTO] - Factura %s y sus detalles sincronizados correctamente.", f.UUID)
	return nil
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
	go func() {
		if err := rtx.Rollback(d.ctx); err != nil {
			d.Log.Errorf("[LOCAL -> REMOTO] - Error durante [syncCompraToRemote] rollback %v", err)
		}
	}()

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
		SELECT uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, factura_uuid, timestamp 
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
			&op.FacturaUUID,
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
		INSERT INTO operacion_stocks (uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, factura_uuid, timestamp, sincronizado)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1) ON CONFLICT(uuid) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("error al preparar statement local de op. stock: %w", err)
	}
	defer stmt.Close()

	for _, op := range newOps {
		_, err := stmt.ExecContext(ctx, op.UUID, op.ProductoID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorID, op.FacturaID, op.FacturaUUID, op.Timestamp)
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
