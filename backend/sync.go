package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/errgroup"
)

const (
	IDConstraintName            = "facturas_pkey"
	UUIDConstraintName          = "facturas_uuid_unique"
	NumeroFacturaConstraintName = "uni_facturas_numero_factura"
)

func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
	//go d.NormalizarStock()
	//go func() {
	//	if _, err := d.NormalizarStockTodosLosProductos(); err != nil {
	//		d.Log.Errorf("Error sincronizando operaciones de stock hacia local: %v", err)
	//	}
	//}()

}

func (d *Db) SincronizacionInteligente() {
	if !d.syncMutex.TryLock() {
		d.Log.Warn("La sincronización inteligente ya está en proceso. Omitiendo esta ejecución.")
		return
	}
	defer d.syncMutex.Unlock()

	runtime.EventsEmit(d.ctx, "sync:start", "[INICIO]: Sincronización Inteligente (refactor)")

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
		{"vendedors", "cedula", []string{"created_at", "updated_at", "deleted_at", "uuid", "nombre", "apellido", "cedula", "email", "contrasena", "mfa_enabled", "mfa_secret"}},
		{"clientes", "numero_id", []string{"created_at", "updated_at", "deleted_at", "uuid", "nombre", "apellido", "tipo_id", "numero_id", "telefono", "email", "direccion"}},
		{"proveedors", "nombre", []string{"created_at", "updated_at", "deleted_at", "uuid", "nombre", "telefono", "email"}},
		{"productos", "codigo", []string{"created_at", "updated_at", "deleted_at", "uuid", "nombre", "codigo", "precio_venta", "stock"}},
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
	runtime.EventsEmit(d.ctx, "sync:finish", "Sincronización completada exitosamente.")

	d.Log.Info("[FIN]: Sincronización Inteligente")
}

func (d *Db) syncGenericModel(ctx context.Context, tableName, uniqueCol string, cols []string) error {
	d.Log.Infof("[%s] Inicio syncGenericModel (unique: %s)", tableName, uniqueCol)

	// 1) Obtener última sincronización desde la base de datos local
	var lastSync time.Time
	err := d.LocalDB.QueryRowContext(ctx, `
		SELECT last_sync_timestamp 
		FROM sync_log 
		WHERE model_name = ?
	`, tableName).Scan(&lastSync)

	if err != nil && err != sql.ErrNoRows {
		d.Log.Errorf("[%s] Error leyendo sync_log: %v", tableName, err)
		return err
	}

	d.Log.Infof("[%s] Última sincronización: %v", tableName, lastSync)

	// --- DESCARGA Remoto -> Local ---
	remoteCols := strings.Join(cols, ",")
	remoteQuery := fmt.Sprintf(`SELECT %s FROM %s`, remoteCols, tableName)

	var args []any
	if !lastSync.IsZero() {
		remoteQuery += ` WHERE updated_at > $1`
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
	defer func() {
		if rErr := txLocal.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error [ActualizarProducto] rollback: %v", rErr)
		}
	}()

	placeholders := strings.Repeat("?,", len(cols)-1) + "?"
	upsertAssignments := []string{}
	for _, c := range cols {
		if c == "uuid" || c == uniqueCol || c == "created_at" {
			continue
		}
		if c == "updated_at" {
			upsertAssignments = append(upsertAssignments, "updated_at = excluded.updated_at")
		} else {
			upsertAssignments = append(upsertAssignments, fmt.Sprintf("%s = excluded.%s", c, c))
		}
	}

	upsertSQL := fmt.Sprintf(`
		INSERT INTO %s (%s) VALUES (%s)
		ON CONFLICT(%s) DO UPDATE SET %s
		WHERE excluded.updated_at > %s.updated_at
	`, tableName, strings.Join(cols, ","), placeholders, uniqueCol, strings.Join(upsertAssignments, ", "), tableName)

	insStmt, err := txLocal.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return fmt.Errorf("[%s] error preparando upsert local: %w", tableName, err)
	}
	defer insStmt.Close()

	remoteCount := 0
	downloadedUUIDs := []string{}

	for rows.Next() {
		rawVals := make([]any, len(cols))
		valPtrs := make([]any, len(cols))
		for i := range rawVals {
			valPtrs[i] = &rawVals[i]
		}

		if err := rows.Scan(valPtrs...); err != nil {
			d.Log.Errorf("[%s] Error escaneando fila remota: %v", tableName, err)
			continue
		}

		// Convertir UUID si viene en bytes
		for i, colName := range cols {
			if colName == "uuid" {
				switch v := rawVals[i].(type) {
				case [16]uint8:
					rawVals[i] = uuid.UUID(v).String()
				}
				downloadedUUIDs = append(downloadedUUIDs, fmt.Sprintf("%v", rawVals[i]))
			}
		}

		if err := d.sanitizeRow(tableName, cols, rawVals); err != nil {
			d.Log.Warn(err.Error())
			continue
		}

		if _, err := insStmt.ExecContext(ctx, rawVals...); err != nil {
			d.Log.Errorf("[%s] Error en UPSERT local: %v", tableName, err)
			continue
		}

		remoteCount++
	}

	d.Log.Infof("[%s] Recibidos [ %d ] registros del servidor remoto", tableName, remoteCount)
	if len(downloadedUUIDs) > 0 {
		d.Log.Infof("[%s] UUIDs descargados: %v", tableName, downloadedUUIDs)
	}

	// Tomar timestamp después de insert remoto->local
	syncStartTime := time.Now().UTC()
	d.Log.Infof("[%s] Marca de syncStartTime: %s", tableName, syncStartTime)

	// --- SUBIDA Local -> Remoto ---
	if lastSync.IsZero() {
		d.Log.Warnf("[%s] Primera sincronización — NO se suben registros", tableName)
	} else {
		localQuery := fmt.Sprintf(`
		SELECT %s 
		FROM %s 
		WHERE updated_at > ? AND updated_at < ?
	`, strings.Join(cols, ","), tableName)

		localRows, err := txLocal.QueryContext(ctx, localQuery, lastSync, syncStartTime)
		if err != nil {
			return fmt.Errorf("[%s] error consultando locales para push: %w", tableName, err)
		}
		defer localRows.Close()

		remotePlaceholders := make([]string, len(cols))
		for i := range cols {
			remotePlaceholders[i] = fmt.Sprintf("$%d", i+1)
		}

		remoteAssignments := []string{}
		for _, c := range cols {
			if c == "uuid" || c == uniqueCol || c == "created_at" {
				continue
			}
			if c == "updated_at" {
				remoteAssignments = append(remoteAssignments, "updated_at = EXCLUDED.updated_at")
			} else {
				remoteAssignments = append(remoteAssignments, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
			}
		}

		remoteInsert := fmt.Sprintf(`
		INSERT INTO %s (%s) VALUES (%s)
		ON CONFLICT (%s) DO UPDATE SET %s
		WHERE EXCLUDED.updated_at > %s.updated_at
	`, tableName, strings.Join(cols, ","), strings.Join(remotePlaceholders, ","), uniqueCol, strings.Join(remoteAssignments, ", "), tableName)

		var batch pgx.Batch
		localChanges := 0
		uploadedUUIDs := []string{}

		for localRows.Next() {
			values := make([]any, len(cols))
			valPtrs := make([]any, len(cols))
			for i := range values {
				valPtrs[i] = &values[i]
			}

			if err := localRows.Scan(valPtrs...); err != nil {
				d.Log.Errorf("[%s] Error escaneando fila local: %v", tableName, err)
				continue
			}

			for i, c := range cols {
				if c == "uuid" {
					uploadedUUIDs = append(uploadedUUIDs, fmt.Sprintf("%v", values[i]))
				}
			}

			batch.Queue(remoteInsert, values...)
			localChanges++
		}

		if localChanges > 0 {
			d.Log.Infof("[%s] Preparando subida de [ %d ] registros al remoto", tableName, localChanges)
			d.Log.Infof("[%s] UUIDs a subir: %v", tableName, uploadedUUIDs)

			br := d.RemoteDB.SendBatch(ctx, &batch)
			if err := br.Close(); err != nil {
				d.Log.Errorf("[%s] Error ejecutando batch remoto: %v", tableName, err)
			}
		} else {
			d.Log.Infof("[%s] No hay cambios locales para subir", tableName)
		}
	}

	_, err = txLocal.ExecContext(ctx, `
		INSERT INTO sync_log(model_name, last_sync_timestamp) 
		VALUES (?, ?) 
		ON CONFLICT(model_name) DO UPDATE SET last_sync_timestamp = ?
	`, tableName, syncStartTime, syncStartTime)

	if err != nil {
		return fmt.Errorf("[%s] error actualizando sync_log: %w", tableName, err)
	}

	if err := txLocal.Commit(); err != nil {
		return fmt.Errorf("[%s] error confirmando tx local: %w", tableName, err)
	}

	d.Log.Infof("[%s] Sincronización completa", tableName)
	return nil
}

// SincronizarOperacionesStockHaciaRemoto envía las operaciones locales no sincronizadas
// al servidor remoto PostgreSQL, las inserta en bloque y recalcula el stock remoto.
func (d *Db) SincronizarOperacionesStockHaciaRemoto() {
	if !d.isRemoteDBAvailable() {
		d.Log.Warn("[REMOTO] Base de datos remota no disponible, omitiendo sincronización de stock.")
		return
	}

	d.Log.Info("[SYNC STOCK] Iniciando sincronización de operaciones hacia remoto...")

	const selectPendientes = `
	SELECT uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante,
		   vendedor_uuid, factura_uuid, timestamp
	FROM operacion_stocks 
	WHERE sincronizado = 0
	`

	rows, err := d.LocalDB.QueryContext(d.ctx, selectPendientes)
	if err != nil {
		d.Log.Errorf("[SYNC] Error leyendo operaciones locales: %v", err)
		return
	}
	defer rows.Close()

	type pendingOp struct {
		localUUID string
		op        OperacionStock
	}

	var pendientes []pendingOp
	productosAfectados := map[string]bool{}

	for rows.Next() {
		var op OperacionStock
		var stockResult sql.NullInt64
		var vendedorUUID sql.NullString
		var facturaUUID sql.NullString

		if err := rows.Scan(
			&op.UUID, &op.ProductoUUID, &op.TipoOperacion, &op.CantidadCambio,
			&stockResult, &vendedorUUID, &facturaUUID, &op.Timestamp,
		); err != nil {
			d.Log.Warnf("[SYNC] Error leyendo operación de stock, saltando: %v", err)
			continue
		}

		if stockResult.Valid {
			op.StockResultante = int(stockResult.Int64)
		}
		if vendedorUUID.Valid {
			op.VendedorUUID = vendedorUUID.String
		}
		if facturaUUID.Valid {
			f := facturaUUID.String
			op.FacturaUUID = &f
		}

		pendientes = append(pendientes, pendingOp{localUUID: op.UUID, op: op})
		productosAfectados[op.ProductoUUID] = true
	}

	if len(pendientes) == 0 {
		d.Log.Info("[SYNC] No hay operaciones locales pendientes.")
		return
	}

	rtx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		d.Log.Errorf("[SYNC] No se pudo iniciar transacción remota: %v", err)
		return
	}

	commit := false
	defer func() {
		if !commit {
			_ = rtx.Rollback(d.ctx)
		}
	}()

	// --- COPY masivo a la tabla remota ---
	_, err = rtx.CopyFrom(
		d.ctx,
		pgx.Identifier{"operacion_stocks"},
		[]string{"uuid", "producto_uuid", "tipo_operacion", "cantidad_cambio", "stock_resultante",
			"vendedor_uuid", "factura_uuid", "timestamp"},
		pgx.CopyFromSlice(len(pendientes), func(i int) ([]any, error) {
			o := pendientes[i].op
			var vendedor any
			if o.VendedorUUID != "" {
				vendedor = o.VendedorUUID
			}
			var factura any
			if o.FacturaUUID != nil {
				factura = *o.FacturaUUID
			}
			return []any{
				o.UUID, o.ProductoUUID, o.TipoOperacion, o.CantidadCambio,
				o.StockResultante, vendedor, factura, o.Timestamp,
			}, nil
		}),
	)

	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		d.Log.Errorf("[SYNC] Error durante COPY remoto: %v", err)
		return
	}

	// --- Recalcular stock remoto con fuente de verdad ---
	productList := make([]string, 0, len(productosAfectados))
	for p := range productosAfectados {
		productList = append(productList, p)
	}

	_, err = rtx.Exec(d.ctx, `
		UPDATE productos p
		SET stock = sub.nuevo_stock
		FROM (
			SELECT producto_uuid, COALESCE(SUM(cantidad_cambio), 0) AS nuevo_stock
			FROM operacion_stocks
			WHERE producto_uuid = ANY($1)
			GROUP BY producto_uuid
		) sub
		WHERE p.uuid = sub.producto_uuid;
	`, productList)

	if err != nil {
		d.Log.Errorf("[SYNC] Error recalculando stock remoto: %v", err)
		return
	}

	if err := rtx.Commit(d.ctx); err != nil {
		d.Log.Errorf("[SYNC] Error al confirmar la transacción remota: %v", err)
		return
	}
	commit = true

	// --- Marcar operaciones locales como sincronizadas ---
	ids := make([]any, len(pendientes))
	placeholders := make([]string, len(pendientes))
	for i, p := range pendientes {
		ids[i] = p.localUUID
		placeholders[i] = "?"
	}

	localUpdate := fmt.Sprintf(
		"UPDATE operacion_stocks SET sincronizado = 1 WHERE uuid IN (%s)",
		strings.Join(placeholders, ","),
	)
	if _, err := d.LocalDB.ExecContext(d.ctx, localUpdate, ids...); err != nil {
		d.Log.Errorf("[SYNC] Error actualizando flag de sincronización local: %v", err)
	}

	d.Log.Infof("[SYNC COMPLETA] %d operaciones sincronizadas, %d productos recalculados",
		len(pendientes), len(productList))
}

func (d *Db) ForzarResincronizacionLocalDesdeRemoto() error {
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[REMOTO -> LOCAL] - Error durante [ForzarResincronizacionLocalDesdeRemoto] rollback %v", err)
		}
	}()

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
	if !d.isRemoteDBAvailable() {
		return fmt.Errorf("[REMOTO] la base de datos no está disponible, se omite la sincronización")
	}
	d.Log.Info("[SINCRONIZANDO]: Transacciones desde Remoto -> Local")

	ctx := d.ctx
	var args []any

	// -------------------------------------------------
	// 1️⃣ LECTURA DE ÚLTIMA FECHA - FACTURAS (FUERA DE TX)
	// -------------------------------------------------
	var lastFacturaTimeStr sql.NullString
	var lastFacturaTime time.Time
	queryFact := `SELECT MAX(created_at) FROM facturas`

	if err := d.LocalDB.QueryRowContext(ctx, queryFact).Scan(&lastFacturaTimeStr); err != nil {
		return fmt.Errorf("error al obtener fecha de última factura local: %w", err)
	}

	if !lastFacturaTimeStr.Valid || lastFacturaTimeStr.String == "" {
		// Carga inicial: Normalizar a epoch (1970-01-01)
		d.Log.Info("[SINCRONIZANDO] Facturas: No hay registros locales. Realizando carga inicial completa.")
		lastFacturaTime = time.Unix(0, 0)
	} else {
		if parsedTime, parseErr := parseFlexibleTime(lastFacturaTimeStr.String); parseErr == nil {
			lastFacturaTime = parsedTime
			d.Log.Infof("[SINCRONIZANDO] Facturas: Carga incremental desde %v", lastFacturaTime)
		} else {
			// Error de parseo, mejor hacer carga inicial
			d.Log.Warnf("No se pudo parsear la fecha de última factura local '%s': %v. Realizando carga inicial completa.", lastFacturaTimeStr.String, parseErr)
			lastFacturaTime = time.Unix(0, 0)
		}
	}

	// -------------------------------------------------
	// 2️⃣ LECTURA DE ÚLTIMA FECHA - COMPRAS (FUERA DE TX)
	// -------------------------------------------------
	var lastCompraTimeStr sql.NullString
	var lastCompraTime time.Time
	queryComp := `SELECT MAX(created_at) FROM compras`

	if err := d.LocalDB.QueryRowContext(ctx, queryComp).Scan(&lastCompraTimeStr); err != nil {
		return fmt.Errorf("error al obtener fecha de última compra local: %w", err)
	}

	if !lastCompraTimeStr.Valid || lastCompraTimeStr.String == "" {
		// Carga inicial
		d.Log.Info("[SINCRONIZANDO] Compras: No hay registros locales. Realizando carga inicial completa.")
		lastCompraTime = time.Unix(0, 0)
	} else {
		if parsedTime, parseErr := parseFlexibleTime(lastCompraTimeStr.String); parseErr == nil {
			lastCompraTime = parsedTime
			d.Log.Infof("[SINCRONIZANDO] Compras: Carga incremental desde %v", lastCompraTime)
		} else {
			d.Log.Warnf("No se pudo parsear la fecha de última compra local '%s': %v. Realizando carga inicial completa.", lastCompraTimeStr.String, parseErr)
			lastCompraTime = time.Unix(0, 0)
		}
	}

	// -------------------------------------------------
	// 3️⃣ INICIO DE TRANSACCIÓN LOCAL (SÓLO PARA ESCRITURAS)
	// -------------------------------------------------
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción local: %w", err)
	}

	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error rollback en sincronizarTransaccionesHaciaLocal: %v", rErr)
		}
	}()

	// -------------------------------------------------
	// 4️⃣ OBTENER E INSERTAR FACTURAS
	// -------------------------------------------------
	facturasRemotasQuery := `
		SELECT uuid, numero_factura, fecha_emision, vendedor_uuid, cliente_uuid, subtotal, iva, total,
		       estado, metodo_pago, created_at, updated_at
		FROM facturas
		WHERE COALESCE(created_at, '1970-01-01T00:00:00Z') > $1
		ORDER BY created_at ASC`

	args = append(args, lastFacturaTime)

	rows, err := d.RemoteDB.Query(ctx, facturasRemotasQuery, args...)
	if err != nil {
		return fmt.Errorf("error obteniendo facturas remotas: %w", err)
	}
	defer rows.Close()

	insertFactSQL := `
		INSERT INTO facturas (
			uuid, numero_factura, fecha_emision, vendedor_uuid, cliente_uuid, subtotal, iva, total,
			estado, metodo_pago, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO NOTHING`
	stmtFact, err := tx.PrepareContext(ctx, insertFactSQL)
	if err != nil {
		return fmt.Errorf("error preparando statement de facturas: %w", err)
	}
	defer stmtFact.Close()

	var facturas []Factura
	var facturaUUIDsRemotos []string

	for rows.Next() {
		var f Factura
		if err := rows.Scan(
			&f.UUID, &f.NumeroFactura, &f.FechaEmision, &f.VendedorUUID, &f.ClienteUUID,
			&f.Subtotal, &f.IVA, &f.Total, &f.Estado, &f.MetodoPago, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			d.Log.Errorf("Error al escanear factura remota: %v", err)
			continue
		}

		if _, err := stmtFact.ExecContext(ctx,
			f.UUID, f.NumeroFactura, f.FechaEmision, f.VendedorUUID, f.ClienteUUID,
			f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt); err != nil {
			d.Log.Errorf("Error insertando factura local (UUID %s): %v", f.UUID, err)
			continue
		}

		facturas = append(facturas, f)
		facturaUUIDsRemotos = append(facturaUUIDsRemotos, f.UUID)
	}
	// Cerrar rows explícitamente antes de la siguiente consulta
	rows.Close()

	d.Log.Infof("Sincronizadas %d nuevas facturas.", len(facturas))

	// -------------------------------------------------
	// 4.a) DETALLES DE FACTURAS (Dentro de la misma TX)
	// -------------------------------------------------
	if len(facturaUUIDsRemotos) > 0 {
		placeholders := make([]string, len(facturaUUIDsRemotos))
		args = make([]any, len(facturaUUIDsRemotos)) // Reusar args
		for i, uuid := range facturaUUIDsRemotos {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = uuid
		}

		queryDetalles := fmt.Sprintf(`
			SELECT uuid, factura_uuid, producto_uuid, cantidad, precio_unitario,
			       precio_total, created_at, updated_at
			FROM detalle_facturas
			WHERE factura_uuid IN (%s)
			ORDER BY created_at ASC`, strings.Join(placeholders, ","))

		detalleRows, err := d.RemoteDB.Query(ctx, queryDetalles, args...)
		if err != nil {
			return fmt.Errorf("error obteniendo detalles de factura remotos: %w", err)
		}
		defer detalleRows.Close()

		insertDetalleSQL := `
			INSERT INTO detalle_facturas (
				uuid, factura_uuid, producto_uuid, cantidad, precio_unitario, precio_total, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(uuid) DO NOTHING`
		stmtDetalle, err := tx.PrepareContext(ctx, insertDetalleSQL)
		if err != nil {
			return fmt.Errorf("error preparando statement de detalle_facturas: %w", err)
		}
		defer stmtDetalle.Close()

		detalleCount := 0
		for detalleRows.Next() {
			var df DetalleFactura
			if err := detalleRows.Scan(
				&df.UUID, &df.FacturaUUID, &df.ProductoUUID, &df.Cantidad,
				&df.PrecioUnitario, &df.PrecioTotal, &df.CreatedAt, &df.UpdatedAt,
			); err != nil {
				d.Log.Errorf("Error al escanear detalle de factura remoto: %v", err)
				continue
			}

			if _, err := stmtDetalle.ExecContext(ctx,
				df.UUID, df.FacturaUUID, df.ProductoUUID, df.Cantidad,
				df.PrecioUnitario, df.PrecioTotal, df.CreatedAt, df.UpdatedAt); err != nil {
				d.Log.Errorf("Error insertando detalle de factura (UUID %s): %v", df.UUID, err)
				continue
			}
			detalleCount++
		}
		// Cerrar detalleRows explícitamente
		detalleRows.Close()
		d.Log.Infof("Sincronizados %d nuevos detalles de factura.", detalleCount)
	}

	// -------------------------------------------------
	// 5️⃣ OBTENER E INSERTAR COMPRAS (Dentro de la misma TX)
	// -------------------------------------------------
	args = args[:0] // Limpiar slice de argumentos
	comprasQuery := `
		SELECT uuid, fecha, proveedor_uuid, factura_numero, total, created_at, updated_at
		FROM compras
		WHERE COALESCE(created_at, '1970-01-01T00:00:00Z') > $1
		ORDER BY created_at ASC`
	args = append(args, lastCompraTime)

	compraRows, err := d.RemoteDB.Query(ctx, comprasQuery, args...)
	if err != nil {
		return fmt.Errorf("error obteniendo compras remotas: %w", err)
	}
	defer compraRows.Close()

	insertCompraSQL := `
		INSERT INTO compras (
			uuid, fecha, proveedor_uuid, factura_numero, total, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO NOTHING`
	stmtCompra, err := tx.PrepareContext(ctx, insertCompraSQL)
	if err != nil {
		return fmt.Errorf("error preparando statement de compras: %w", err)
	}
	defer stmtCompra.Close()

	compraCount := 0
	for compraRows.Next() {
		var c Compra
		if err := compraRows.Scan(
			&c.UUID, &c.Fecha, &c.ProveedorUUID, &c.FacturaNumero, &c.Total, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			d.Log.Errorf("Error al escanear compra remota: %v", err)
			continue
		}

		if _, err := stmtCompra.ExecContext(ctx,
			c.UUID, c.Fecha, c.ProveedorUUID, c.FacturaNumero, c.Total, c.CreatedAt, c.UpdatedAt); err != nil {
			d.Log.Errorf("Error insertando compra local (UUID %s): %v", c.UUID, err)
			continue
		}
		compraCount++
	}
	compraRows.Close() // Cerrar explícitamente
	d.Log.Infof("Sincronizadas %d nuevas compras.", compraCount)

	// -------------------------------------------------
	// ✅ 6️⃣ COMMIT FINAL
	// -------------------------------------------------
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error confirmando transacción local: %w", err)
	}

	d.Log.Info("Transacciones sincronizadas hacia local correctamente.")
	return nil
}

// --- FUNCIONES DE SINCRONIZACIÓN INDIVIDUAL ---

func (d *Db) syncVendedorToRemote(uuid string) {
	if !d.isRemoteDBAvailable() {
		d.Log.Warn("[REMOTO] la base de datos no está disponible, se omite la sincronización.")
		return
	}
	var v Vendedor
	query := `SELECT uuid, created_at, updated_at, deleted_at, nombre, apellido, cedula, email, contrasena, mfa_enabled FROM vendedors WHERE uuid = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, uuid).Scan(&v.UUID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt, &v.Nombre, &v.Apellido, &v.Cedula, &v.Email, &v.Contrasena, &v.MFAEnabled)
	if err != nil {
		d.Log.Errorf("[LOCAL] syncVendedorToRemote: no se encontró vendedor local UUID %s: %v", uuid, err)
		return
	}

	upsertSQL := `
		INSERT INTO vendedors (uuid, created_at, updated_at, deleted_at, nombre, apellido, cedula, email, contrasena, mfa_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (cedula) DO UPDATE SET
			nombre = EXCLUDED.nombre,
			apellido = EXCLUDED.apellido,
			email = EXCLUDED.email,
			contrasena = EXCLUDED.contrasena,
			mfa_enabled = EXCLUDED.mfa_enabled,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, v.UUID, v.CreatedAt, v.UpdatedAt, v.DeletedAt, v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de vendedor remoto UUID %s: %v", uuid, err)
		return
	}
	d.Log.Infof("Sincronizado vendedor individual UUID %s hacia el remoto.", uuid)
}

func (d *Db) syncClienteToRemote(uuid string) {
	if !d.isRemoteDBAvailable() {
		d.Log.Warn("[REMOTO] la base de datos no está disponible, se omite la sincronización.")
		return
	}
	var c Cliente
	query := `SELECT uuid, created_at, updated_at, deleted_at, nombre, apellido, tipo_id, numero_id, telefono, email, direccion FROM clientes WHERE uuid = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, uuid).Scan(&c.UUID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt, &c.Nombre, &c.Apellido, &c.TipoID, &c.NumeroID, &c.Telefono, &c.Email, &c.Direccion)
	if err != nil {
		d.Log.Errorf("syncClienteToRemote: no se encontró cliente local UUID %s: %v", uuid, err)
		return
	}

	upsertSQL := `
		INSERT INTO clientes (uuid, created_at, updated_at, deleted_at, nombre, apellido, tipo_id, numero_id, telefono, email, direccion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (numero_id) DO UPDATE SET
			nombre = EXCLUDED.nombre, apellido = EXCLUDED.apellido, tipo_id = EXCLUDED.tipo_id, telefono = EXCLUDED.telefono, email = EXCLUDED.email, direccion = EXCLUDED.direccion,
			updated_at = EXCLUDED.updated_at, deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, c.UUID, c.CreatedAt, c.UpdatedAt, c.DeletedAt, c.Nombre, c.Apellido, c.TipoID, c.NumeroID, c.Telefono, c.Email, c.Direccion)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de cliente remoto UUID %s: %v", uuid, err)
		return
	}
	d.Log.Infof("Sincronizado cliente individual UUID %s hacia el remoto.", uuid)
}

func (d *Db) syncProductoToRemote(p_uuid string) {
	if !d.isRemoteDBAvailable() {
		d.Log.Warn("[REMOTO] la base de datos no está disponible, se omite la sincronización.")
		return
	}
	var p Producto
	query := `SELECT uuid, created_at, updated_at, deleted_at, nombre, codigo, precio_venta FROM productos WHERE uuid = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, p_uuid).Scan(&p.UUID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Nombre, &p.Codigo, &p.PrecioVenta)
	if err != nil {
		d.Log.Errorf("syncProductoToRemote: no se encontró producto local UUID %s: %v", p_uuid, err)
		return
	}

	upsertSQL := `
		INSERT INTO productos (uuid, created_at, updated_at, deleted_at, nombre, codigo, precio_venta)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (codigo) DO UPDATE SET
			nombre = EXCLUDED.nombre, 
			precio_venta = EXCLUDED.precio_venta,
			updated_at = EXCLUDED.updated_at, 
			deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, p.UUID, p.CreatedAt, p.UpdatedAt, p.DeletedAt, p.Nombre, p.Codigo, p.PrecioVenta)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de datos maestros del producto remoto UUID %s: %v", p_uuid, err)
		return
	}
	d.Log.Infof("Sincronizado datos maestros del producto UUID %s. El stock se calculará por separado.", p_uuid)
}

func (d *Db) syncProveedorToRemote(p_uuid string) {
	if !d.isRemoteDBAvailable() {
		d.Log.Warn("[REMOTO] la base de datos no está disponible, se omite la sincronización.")
		return
	}
	var p Proveedor
	query := `SELECT uuid, created_at, updated_at, deleted_at, nombre, telefono, email FROM proveedors WHERE uuid = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, p_uuid).Scan(&p.UUID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Nombre, &p.Telefono, &p.Email)
	if err != nil {
		d.Log.Errorf("syncProveedorToRemote: no se encontró proveedor local UUID %s: %v", p_uuid, err)
		return
	}

	upsertSQL := `
		INSERT INTO proveedors (uuid, created_at, updated_at, deleted_at, nombre, telefono, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (nombre) DO UPDATE SET
			telefono = EXCLUDED.telefono, email = EXCLUDED.email,
			updated_at = EXCLUDED.updated_at, deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, p.UUID, p.CreatedAt, p.UpdatedAt, p.DeletedAt, p.Nombre, p.Telefono, p.Email)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de proveedor remoto UUID %s: %v", p_uuid, err)
		return
	}
	d.Log.Infof("Sincronizado proveedor individual UUID %s hacia el remoto.", p_uuid)
}

// syncVentaToRemote: sincroniza una factura + detalles + operaciones de stock relacionadas
// de forma atómica usando la estrategia EAFP (Es más fácil pedir perdón que permiso).
func (d *Db) syncVentaToRemote(facturaUUID string) error {
	ctx := d.ctx
	d.Log.Infof("[LOCAL -> REMOTO] - Iniciando sincronización atómica para Venta UUID %s", facturaUUID)
	runtime.EventsEmit(d.ctx, "sync:start", facturaUUID)

	// --- 1) OBTENER TODOS LOS DATOS LOCALES PRIMERO ---

	// 1a) Obtener factura local
	var f Factura
	err := d.LocalDB.QueryRowContext(ctx, `
		SELECT uuid, numero_factura, fecha_emision, vendedor_uuid, cliente_uuid, subtotal, iva, total, estado, metodo_pago, created_at, updated_at
		FROM facturas WHERE uuid = ?`, facturaUUID).Scan(
		&f.UUID, &f.NumeroFactura, &f.FechaEmision, &f.VendedorUUID, &f.ClienteUUID,
		&f.Subtotal, &f.IVA, &f.Total, &f.Estado, &f.MetodoPago, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			d.Log.Warnf("[LOCAL] - No se encontró la factura UUID [%s] para sincronizar. Omitiendo.", facturaUUID)
			return nil
		}
		return fmt.Errorf("[LOCAL] - factura no encontrada localmente: %w", err)
	}

	// 1b) Obtener detalles locales
	var detallesLocales []DetalleFactura
	rowsDetalles, err := d.LocalDB.QueryContext(ctx, `
		SELECT uuid, factura_uuid, producto_uuid, cantidad, precio_unitario, precio_total, created_at, updated_at 
		FROM detalle_facturas WHERE factura_uuid = ?`, facturaUUID)
	if err != nil {
		return fmt.Errorf("error obteniendo detalles locales: %w", err)
	}
	for rowsDetalles.Next() {
		var df DetalleFactura
		if err := rowsDetalles.Scan(&df.UUID, &df.FacturaUUID, &df.ProductoUUID, &df.Cantidad, &df.PrecioUnitario, &df.PrecioTotal, &df.CreatedAt, &df.UpdatedAt); err != nil {
			d.Log.Errorf("Error al escanear detalle_factura local: %v", err)
			rowsDetalles.Close()
			return err
		}
		detallesLocales = append(detallesLocales, df)
	}
	rowsDetalles.Close()

	// 1c) Obtener operaciones de stock locales
	var operacionesLocales []OperacionStock
	rowsOps, err := d.LocalDB.QueryContext(ctx, `
		SELECT uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, factura_uuid, timestamp 
		FROM operacion_stocks WHERE factura_uuid = ?`, facturaUUID)
	if err != nil {
		return fmt.Errorf("error obteniendo operaciones de stock locales: %w", err)
	}
	for rowsOps.Next() {
		var op OperacionStock
		var stockResultante sql.NullInt64
		if err := rowsOps.Scan(&op.UUID, &op.ProductoUUID, &op.TipoOperacion, &op.CantidadCambio, &stockResultante, &op.VendedorUUID, &op.FacturaUUID, &op.Timestamp); err != nil {
			d.Log.Errorf("Error al escanear operacion_stock local: %v", err)
			rowsOps.Close()
			return err
		}
		operacionesLocales = append(operacionesLocales, op)
	}
	rowsOps.Close()

	// --- 2) INICIAR TRANSACCIÓN REMOTA ATÓMICA ---
	rtx, err := d.RemoteDB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("[REMOTO] - error al iniciar transacción: %w", err)
	}
	// defer rtx.Rollback() se encargará de cualquier 'return err'
	defer func() {
		if rErr := rtx.Rollback(ctx); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
			d.Log.Errorf("[REMOTO] - Error durante [syncVentaToRemote] rollback %v", rErr)
		}
	}()

	// --- 3) EAFP PARA LA FACTURA (INSERT-FIRST) ---
	var facturaFueInsertada bool
	var needsLocalUpdate bool
	finalNumeroFactura := f.NumeroFactura

	insertFacturaSQL := `
		INSERT INTO facturas (uuid, numero_factura, fecha_emision, vendedor_uuid, cliente_uuid, subtotal, iva, total, estado, metodo_pago, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err = rtx.Exec(ctx, insertFacturaSQL,
		f.UUID, f.NumeroFactura, f.FechaEmision, f.VendedorUUID, f.ClienteUUID,
		f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt,
	)

	if err == nil {
		d.Log.Infof("[REMOTO] - Factura %s insertada ", f.UUID)
		facturaFueInsertada = true
	} else {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 = unique_violation

			switch pgErr.ConstraintName {
			case UUIDConstraintName:
				d.Log.Infof("[REMOTO] - Factura UUID [%s] ya existe. No se insertará, pero se marcará como sincronizada.", facturaUUID)
				facturaFueInsertada = false

			case NumeroFacturaConstraintName:
				d.Log.Warnf("[REMOTO] - Colisión de numero_factura [%s]. Buscando nuevo número...", f.NumeroFactura)
				prefix, _, parseErr := parseNumeroFactura(f.NumeroFactura)
				if parseErr != nil {
					return fmt.Errorf("[REMOTO] - Colisión de numero_factura ('%s') formato inválido: %w", f.NumeroFactura, parseErr)
				}
				var maxNum int
				maxQuery := `
					SELECT COALESCE(MAX(CAST(SUBSTRING(numero_factura FROM '(\d+)$') AS INTEGER)), 0)
					FROM facturas 
					WHERE numero_factura LIKE $1`

				if errMax := rtx.QueryRow(ctx, maxQuery, prefix+"%").Scan(&maxNum); errMax != nil {
					return fmt.Errorf("error obteniendo max numero_factura tras colisión: %w", errMax)
				}

				newNum := maxNum + 1
				numeroParaInsertar := fmt.Sprintf("%s%d", prefix, newNum)
				d.Log.Infof("[REMOTO] - Nuevo número asignado: %s", numeroParaInsertar)

				_, errInsert2 := rtx.Exec(ctx, insertFacturaSQL,
					f.UUID, numeroParaInsertar, f.FechaEmision, f.VendedorUUID, f.ClienteUUID,
					f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt,
				)

				if errInsert2 != nil {
					return fmt.Errorf("error en el segundo intento de insert con '%s': %w", numeroParaInsertar, errInsert2)
				}

				d.Log.Infof("[REMOTO] - Factura %s insertada con nuevo número %s.", f.UUID, numeroParaInsertar)
				facturaFueInsertada = true
				needsLocalUpdate = true
				finalNumeroFactura = numeroParaInsertar

			default:
				// Otro error de constraint
				return fmt.Errorf("[REMOTO] - colisión unique desconocida (restricción: %s): %w", pgErr.ConstraintName, err)
			}
		} else {
			// Error que no es 'unique_violation'
			return fmt.Errorf("error al insertar factura (no es colisión unique): %w", err)
		}
	}

	// --- 4) SINCRONIZAR HIJOS (SI LA FACTURA SE INSERTÓ) ---
	if facturaFueInsertada {
		// 4a) Subir los detalles (usando la misma transacción rtx)
		d.Log.Infof("[REMOTO] - Preparando batch para %d detalles_factura...", len(detallesLocales))
		batchDetalles := &pgx.Batch{}
		for _, df := range detallesLocales {
			batchDetalles.Queue(`
				INSERT INTO detalle_facturas (uuid, factura_uuid, producto_uuid, cantidad, precio_unitario, precio_total, created_at, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
				ON CONFLICT (uuid) DO UPDATE 
				SET updated_at = EXCLUDED.updated_at
				WHERE EXCLUDED.updated_at > detalle_facturas.updated_at`,
				df.UUID, df.FacturaUUID, df.ProductoUUID, df.Cantidad, df.PrecioUnitario, df.PrecioTotal, df.CreatedAt, df.UpdatedAt)
		}

		br := rtx.SendBatch(ctx, batchDetalles)
		if err := br.Close(); err != nil {
			return fmt.Errorf("[REMOTO] - Error ejecutando batch de detalles_factura: %w", err)
		}
		d.Log.Infof("[REMOTO] - Batch de detalles_factura enviado.")

		// 4b) Subir las operaciones de stock (usando la misma transacción rtx)
		d.Log.Infof("[REMOTO] - Preparando batch para %d operacion_stocks...", len(operacionesLocales))
		batchOps := &pgx.Batch{}
		for _, op := range operacionesLocales {
			batchOps.Queue(`
				INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, factura_uuid, timestamp)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (uuid) DO NOTHING`,
				op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorUUID, op.FacturaUUID, op.Timestamp)
		}

		brOps := rtx.SendBatch(ctx, batchOps)
		if err := brOps.Close(); err != nil {
			return fmt.Errorf("[REMOTO] - Error ejecutando batch de operacion_stocks: %w", err)
		}
		d.Log.Infof("[REMOTO] - Batch de operacion_stocks enviado.")

	} // Fin de if(facturaFueInsertada)

	// --- 5) COMMIT REMOTO ---
	// Si todo salió bien (o si la factura ya existía y no se hizo nada),
	// confirmamos la transacción remota.
	if err := rtx.Commit(ctx); err != nil {
		return fmt.Errorf("Error confirmando transacción remota: %w", err)
	}

	// --- 6) MARCAR LOCAL COMO SINCRONIZADO ---
	// Solo si el Commit remoto fue exitoso
	d.Log.Infof("[LOCAL] - Marcando venta %s como sincronizada...", f.UUID)

	if needsLocalUpdate {
		// Actualizar el número de factura local si cambió
		_, err = d.LocalDB.ExecContext(ctx, `UPDATE facturas SET numero_factura = ? WHERE uuid = ?`, finalNumeroFactura, f.UUID)
	}
	if err != nil {
		d.Log.Errorf("[LOCAL] - CRÍTICO: Error al renombrar factura local [%s]: %v", f.UUID, err)
	}

	_, err = d.LocalDB.ExecContext(ctx, `UPDATE operacion_stocks SET sincronizado = 1 WHERE factura_uuid = ?`, f.UUID)
	if err != nil {
		d.Log.Errorf("[LOCAL] - CRÍTICO: No se pudo marcar operaciones de stock locales de %s como sincronizadas: %v", f.UUID, err)
	}

	d.Log.Infof("[LOCAL -> REMOTO] - Venta %s y sus hijos sincronizados correctamente.", f.UUID)
	runtime.EventsEmit(d.ctx, "sync:finish", facturaUUID)
	return nil
}

// syncCompraToRemote: sincroniza una compra + detalles + operaciones de stock asociadas.
func (d *Db) syncCompraToRemote(c_uuid string) {
	if !d.isRemoteDBAvailable() {
		d.Log.Warn("[REMOTO] la base de datos no está disponible, se omite la sincronización.")
		return
	}
	d.Log.Infof("Sincronizando compra individual UUID %s hacia el remoto.", c_uuid)

	// Obtener compra y detalles desde local
	var c Compra
	err := d.LocalDB.QueryRowContext(d.ctx, "SELECT uuid, fecha, proveedor_uuid, factura_numero, total, created_at, updated_at FROM compras WHERE uuid = ?", c_uuid).
		Scan(&c.UUID, &c.Fecha, &c.ProveedorUUID, &c.FacturaNumero, &c.Total, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		d.Log.Errorf("syncCompraToRemote: no se encontró compra local UUID %s: %v", c_uuid, err)
		return
	}

	// Asegurar proveedor en remoto
	d.syncProveedorToRemote(c.ProveedorUUID)

	// Recolectar detalles
	rows, err := d.LocalDB.QueryContext(d.ctx, "SELECT producto_uuid, cantidad, precio_compra_unitario FROM detalle_compras WHERE compra_uuid = ?", c_uuid)
	if err == nil {
		for rows.Next() {
			var det DetalleCompra
			if err := rows.Scan(&det.ProductoUUID, &det.Cantidad, &det.PrecioCompraUnitario); err != nil {
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
	defer func() {
		if rErr := rtx.Rollback(d.ctx); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
			d.Log.Errorf("[LOCAL -> REMOTO] - Error durante [syncCompraToRemote] rollback %v", err)
		}
	}()

	_, err = rtx.Exec(d.ctx, `
		INSERT INTO compras (fecha, proveedor_uuid, factura_numero, total, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (uuid) DO UPDATE SET fecha = EXCLUDED.fecha, proveedor_uuid=EXCLUDED.proveedor_uuid, factura_numero=EXCLUDED.factura_numero, total=EXCLUDED.total, updated_at=EXCLUDED.updated_at
	`, c.Fecha, c.ProveedorUUID, c.FacturaNumero, c.Total, c.CreatedAt, c.UpdatedAt)

	if err != nil {
		d.Log.Errorf("syncCompraToRemote: error upserting compra remota: %v", err)
		return
	}

	if _, err := rtx.Exec(d.ctx, "DELETE FROM detalle_compras WHERE compra_uuid = $1", c.UUID); err != nil {
		d.Log.Errorf("syncCompraToRemote: error borrando detalles remotos: %v", err)
	}
	if len(c.Detalles) > 0 {
		_, err := rtx.CopyFrom(d.ctx,
			pgx.Identifier{"detalle_compras"},
			[]string{"compra_uuid", "producto_uuid", "cantidad", "precio_compra_unitario"},
			pgx.CopyFromSlice(len(c.Detalles), func(i int) ([]any, error) {
				det := c.Detalles[i]
				return []any{c.UUID, det.ProductoUUID, det.Cantidad, det.PrecioCompraUnitario}, nil
			}),
		)
		if err != nil {
			d.Log.Errorf("syncCompraToRemote: error reinsertando detalles remotos: %v", err)
		}
	}

	// Sincronizar operaciones de stock de la compra (si existieran)
	opsRows, err := d.LocalDB.QueryContext(d.ctx, "SELECT uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, factura_uuid, timestamp FROM operacion_stocks WHERE factura_uuid IS NULL AND tipo_operacion = 'COMPRA' AND sincronizado = 0")
	if err == nil {
		var localOps []OperacionStock
		for opsRows.Next() {
			var op OperacionStock
			if err := opsRows.Scan(&op.UUID, &op.ProductoUUID, &op.TipoOperacion, &op.CantidadCambio, &op.StockResultante, &op.VendedorUUID, &op.FacturaUUID, &op.Timestamp); err != nil {
				d.Log.Errorf("syncCompraToRemote: error scanning operacion local: %v", err)
				continue
			}
			localOps = append(localOps, op)
		}
		opsRows.Close()

		if len(localOps) > 0 {
			_, err := rtx.CopyFrom(d.ctx,
				pgx.Identifier{"operacion_stocks"},
				[]string{"uuid", "producto_uuid", "tipo_operacion", "cantidad_cambio", "stock_resultante", "vendedor_uuid", "factura_uuid", "timestamp"},
				pgx.CopyFromSlice(len(localOps), func(i int) ([]any, error) {
					op := localOps[i]
					var facturaID interface{}
					if op.FacturaUUID != nil {
						facturaID = *op.FacturaUUID
					} else {
						facturaID = nil
					}
					return []any{op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorUUID, facturaID, op.Timestamp}, nil
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
						SELECT producto_uuid, COALESCE(SUM(cantidad_cambio),0) as nuevo_stock
						FROM operacion_stocks WHERE producto_uuid = ANY($1) GROUP BY producto_uuid
					)
					UPDATE productos p SET stock = sc.nuevo_stock FROM stock_calculado sc WHERE p.uuid = sc.producto_uuid;
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
	d.Log.Infof("Compra UUID %s sincronizada correctamente al remoto.", c_uuid)
}

func (d *Db) syncVendedorToLocal(v Vendedor) {
	// upsert pattern for sqlite: try update, if rows affected==0 then insert
	res, err := d.LocalDB.Exec("UPDATE vendedors SET nombre=?, apellido=?, cedula=?, email=?, contrasena=?, mfa_enabled=?, updated_at=? WHERE uuid=?", v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled, time.Now(), v.UUID)
	if err != nil {
		d.Log.Errorf("syncVendedorToLocal: error updating local vendedor UUID %s: %v", v.UUID, err)
		return
	}
	r, _ := res.RowsAffected()
	if r == 0 {
		_, err = d.LocalDB.Exec("INSERT INTO vendedors (uuid, nombre, apellido, cedula, email, contrasena, mfa_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", v.UUID, v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled, time.Now(), time.Now())
		if err != nil {
			d.Log.Errorf("syncVendedorToLocal: error inserting local vendedor UUID %s: %v", v.UUID, err)
			return
		}
	}
}

func (d *Db) sincronizarOperacionesStockHaciaLocal() error {
	if !d.isRemoteDBAvailable() {
		return fmt.Errorf("[REMOTO] la base de datos no está disponible, se omite la sincronización.")
	}
	d.Log.Info("[SINCRONIZANDO]: Descargando nuevas Operaciones de Stock desde Remoto -> Local")
	ctx := d.ctx

	// 1. Obtener último timestamp local (Corregido: usando parseFlexibleTime y time.Unix(0,0))
	var lastSyncStr sql.NullString
	err := d.LocalDB.QueryRowContext(ctx, "SELECT MAX(timestamp) FROM operacion_stocks").Scan(&lastSyncStr)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error obteniendo el último timestamp local de op. stock: %w", err)
	}

	var lastSyncTime time.Time
	if !lastSyncStr.Valid || lastSyncStr.String == "" {
		d.Log.Info("No hay operaciones locales previas. Se descargará el historial completo de operaciones remotas.")
		lastSyncTime = time.Unix(0, 0)
	} else {
		if parsedTime, parseErr := parseFlexibleTime(lastSyncStr.String); parseErr == nil {
			lastSyncTime = parsedTime
			d.Log.Infof("Sincronizando operaciones de stock desde %v", lastSyncTime)
		} else {
			d.Log.Warnf("No se pudo parsear fecha de op. stock local '%s': %v. Realizando carga inicial completa.", lastSyncStr.String, parseErr)
			lastSyncTime = time.Unix(0, 0)
		}
	}

	// 2. Obtener operaciones remotas (Corregido: Query unificada con COALESCE)
	remoteOpsQuery := `
        SELECT uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, 
               vendedor_uuid, factura_uuid, timestamp
        FROM operacion_stocks
        WHERE COALESCE(timestamp, '1970-01-01T00:00:00Z') > $1
        ORDER BY timestamp ASC`
	args := []any{lastSyncTime}

	rows, err := d.RemoteDB.Query(ctx, remoteOpsQuery, args...)
	if err != nil {
		return fmt.Errorf("error obteniendo operaciones de stock remotas: %w", err)
	}
	defer rows.Close()

	var newOps []OperacionStock
	productosAfectados := make(map[string]bool)

	for rows.Next() {
		var op OperacionStock

		// Corregido: Variables temporales Null-safe para TODOS los campos que pueden ser NULL
		var productoUUID, tipoOperacion, vendedorUUID, facturaUUID sql.NullString
		var cantidadCambio sql.NullFloat64 // Usar NullFloat64 o NullInt64 según tu struct
		var stockResultante sql.NullInt64
		var opTimestamp sql.NullTime

		if err := rows.Scan(
			&op.UUID,         // Asumiendo que UUID no es NULL
			&productoUUID,    // dest[1] - Corregido
			&tipoOperacion,   // dest[2] - Corregido
			&cantidadCambio,  // dest[3] - Corregido
			&stockResultante, // dest[4] - Corregido
			&vendedorUUID,    // dest[5] - Corregido
			&facturaUUID,     // dest[6] - Corregido
			&opTimestamp,     // dest[7] - Corregido
		); err != nil {
			// Este error ahora solo debería saltar por problemas inesperados, no por NULLs
			d.Log.Warnf("Error al escanear operación remota: %v", err)
			continue
		}

		// Transferir valores de forma segura
		op.ProductoUUID = productoUUID.String           // Será "" si es NULL
		op.TipoOperacion = tipoOperacion.String         // Será "" si es NULL
		op.CantidadCambio = int(cantidadCambio.Float64) // Será 0.0 si es NULL
		op.StockResultante = int(stockResultante.Int64) // Será 0 si es NULL
		op.VendedorUUID = vendedorUUID.String           // Será "" si es NULL
		op.FacturaUUID = &facturaUUID.String            // Será "" si es NULL

		if opTimestamp.Valid {
			op.Timestamp = opTimestamp.Time
		} else {
			d.Log.Warnf("Operación de stock con UUID %s tiene timestamp NULL, usando valor cero.", op.UUID)
		}

		newOps = append(newOps, op)

		// Corregido: No recalcular si el producto_uuid es nulo/vacío
		if op.ProductoUUID != "" {
			productosAfectados[op.ProductoUUID] = true
		}
	}
	// Cerrar rows explícitamente antes de la transacción
	rows.Close()

	if len(newOps) == 0 {
		d.Log.Info("La base de datos local de operaciones de stock ya está actualizada.")
		return nil
	}

	// 3. Insertar las operaciones en una transacción local
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error iniciando transacción local: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("rollback en sincronizarOperacionesStockHaciaLocal: %v", rErr)
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO operacion_stocks (
            uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante,
            vendedor_uuid, factura_uuid, timestamp, sincronizado
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
        ON CONFLICT(uuid) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("error al preparar statement local: %w", err)
	}
	defer stmt.Close()
	var failatempt int = 0
	for _, op := range newOps {
		if _, err := stmt.ExecContext(ctx,
			op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio,
			op.StockResultante, op.VendedorUUID, op.FacturaUUID, op.Timestamp,
		); err != nil {
			failatempt++
			d.Log.Errorf("Error al insertar op. stock local (UUID: %s): %v", op.UUID, err)
		}
	}

	// 4. Recalcular stock local
	d.Log.Infof("Recalculando stock local para %d productos afectados...", len(productosAfectados))
	for uuid := range productosAfectados {
		if err := RecalcularYActualizarStock(tx, uuid); err != nil {
			d.Log.Errorf("Error recalculando stock de producto %s: %v", uuid, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar transacción local: %w", err)
	}

	d.Log.Infof("Sincronizadas %d operaciones nuevas desde remoto.", len(newOps)-failatempt)
	return nil
}

func uniqueProductoIDsFromDetallesCompra(detalles []DetalleCompra) []string {
	m := make(map[string]bool)
	for _, d := range detalles {
		m[d.ProductoUUID] = true
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (d *Db) sanitizeRow(tableName string, cols []string, values []any) error {
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

// parseNumeroFactura divide un número de factura en prefijo y número.
// "FAC-1000" -> ("FAC-", 1000, nil)
// "1000"     -> ("", 1000, nil)
// "INVALID"  -> ("", 0, error)
func parseNumeroFactura(numFactura string) (prefix string, num int, err error) {
	lastDash := strings.LastIndex(numFactura, "-")

	// Caso 1: Sin guión (ej: "1000")
	if lastDash == -1 {
		num, err := strconv.Atoi(numFactura)
		if err != nil {
			// No es un número simple, formato no reconocido
			return "", 0, fmt.Errorf("formato no reconocido, no es numérico: %s", numFactura)
		}
		return "", num, nil // Sin prefijo
	}

	// Caso 2: Con guión (ej: "FAC-1000")
	prefix = numFactura[:lastDash+1]  // "FAC-"
	numStr := numFactura[lastDash+1:] // "1000"

	num, err = strconv.Atoi(numStr)
	if err != nil {
		return "", 0, fmt.Errorf("parte numérica inválida tras el guión '%s': %w", numStr, err)
	}

	return prefix, num, nil
}
