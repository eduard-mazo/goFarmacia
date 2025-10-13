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

	// Sincronizar modelos maestros en paralelo
	g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "vendedors") })
	g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "clientes") })
	// g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "proveedors") })
	// g.Go(func() error { return d.sincronizarTablaMaestra(ctx, "productos") })

	if err := g.Wait(); err != nil {
		d.Log.Errorf("Error durante la sincronización de modelos maestros: %v", err)
		return
	}

	// Sincronizar operaciones y transacciones
	if err := d.SincronizarOperacionesStockHaciaRemoto(); err != nil {
		d.Log.Errorf("Error sincronizando operaciones de stock hacia el remoto: %v", err)
	}
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

	placeholders := strings.Repeat("?,", len(cols)-1) + "?"
	updateAssignments := []string{}
	for _, c := range cols {
		if c != "id" {
			updateAssignments = append(updateAssignments, fmt.Sprintf("%s=excluded.%s", c, c))
		}
	}
	upsertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s",
		tableName, strings.Join(cols, ","), placeholders, uniqueCol, strings.Join(updateAssignments, ","))

	insStmt, err := txLocal.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return fmt.Errorf("[%s] error preparando upsert local: %w", tableName, err)
	}
	defer insStmt.Close()

	// Procesar y aplicar cambios remotos a local
	remoteRowCount := 0
	for rows.Next() {
		remoteRowCount++
		rawVals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range rawVals {
			valPtrs[i] = &rawVals[i]
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

		if _, err := insStmt.ExecContext(ctx, rawVals...); err != nil {
			d.Log.Errorf("[%s] Error en upsert de fila remota a local: %v", tableName, err)
			continue
		}
	}
	d.Log.Infof("[%s] Se recibieron y procesaron %d registros del servidor remoto.", tableName, remoteRowCount)

	// --- Parte 2: Subir cambios del Local al Remoto (CON BATCH) ---

	localQuery := fmt.Sprintf("SELECT %s FROM %s WHERE updated_at > ?", strings.Join(cols, ","), tableName)
	localRows, err := txLocal.QueryContext(ctx, localQuery, lastSync)
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
		localChangesCount++
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
	now := time.Now().UTC()
	_, err = txLocal.ExecContext(ctx, "INSERT INTO sync_log(model_name, last_sync_timestamp) VALUES (?, ?) ON CONFLICT(model_name) DO UPDATE SET last_sync_timestamp = ?", tableName, now, now)
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
func (d *Db) SincronizarOperacionesStockHaciaRemoto() error {
	d.Log.Info("[SINCRONIZANDO]: Operaciones de Stock desde Local -> Remoto")
	query := `SELECT id, uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp
              FROM operacion_stocks WHERE sincronizado = 0`

	rows, err := d.LocalDB.QueryContext(d.ctx, query)
	if err != nil {
		return fmt.Errorf("error al obtener operaciones de stock locales: %w", err)
	}
	defer rows.Close()

	var ops []OperacionStock
	var localIDsToUpdate []int64
	productosAfectados := make(map[uint]bool)

	for rows.Next() {
		var op OperacionStock
		var localID int64
		if err := rows.Scan(&localID, &op.UUID, &op.ProductoID, &op.TipoOperacion, &op.CantidadCambio, &op.StockResultante, &op.VendedorID, &op.FacturaID, &op.Timestamp); err != nil {
			d.Log.Errorf("Error al escanear operación de stock local: %v", err)
			continue
		}
		ops = append(ops, op)
		localIDsToUpdate = append(localIDsToUpdate, localID)
		productosAfectados[op.ProductoID] = true
	}

	if len(ops) == 0 {
		d.Log.Info("No se encontraron nuevas operaciones de stock locales para sincronizar.")
		return nil
	}

	tx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transacción remota: %w", err)
	}
	defer tx.Rollback(d.ctx)

	_, err = tx.CopyFrom(d.ctx,
		pgx.Identifier{"operacion_stocks"},
		[]string{"uuid", "producto_id", "tipo_operacion", "cantidad_cambio", "stock_resultante", "vendedor_id", "factura_id", "timestamp"},
		pgx.CopyFromSlice(len(ops), func(i int) ([]interface{}, error) {
			o := ops[i]
			return []interface{}{o.UUID, o.ProductoID, o.TipoOperacion, o.CantidadCambio, o.StockResultante, o.VendedorID, o.FacturaID, o.Timestamp}, nil
		}),
	)
	if err != nil && !strings.Contains(err.Error(), "duplicate key value") {
		return fmt.Errorf("error en inserción masiva de operaciones de stock: %w", err)
	}

	idsAfectados := make([]uint, 0, len(productosAfectados))
	for id := range productosAfectados {
		idsAfectados = append(idsAfectados, id)
	}

	updateStockCacheSQL := `
		WITH stock_calculado AS (
			SELECT producto_id, SUM(cantidad_cambio) as nuevo_stock
			FROM operacion_stocks
			WHERE producto_id = ANY($1) GROUP BY producto_id
		)
		UPDATE productos p SET stock = sc.nuevo_stock
		FROM stock_calculado sc WHERE p.id = sc.producto_id;`

	if _, err := tx.Exec(d.ctx, updateStockCacheSQL, idsAfectados); err != nil {
		return fmt.Errorf("error al actualizar la caché de stock remota: %w", err)
	}

	if err := tx.Commit(d.ctx); err != nil {
		return fmt.Errorf("error al confirmar la transacción de operaciones de stock remotas: %w", err)
	}

	updateLocalSQL := fmt.Sprintf("UPDATE operacion_stocks SET sincronizado = 1 WHERE id IN (%s)", joinInt64s(localIDsToUpdate))
	if _, err := d.LocalDB.ExecContext(d.ctx, updateLocalSQL); err != nil {
		return fmt.Errorf("error al marcar operaciones como sincronizadas localmente: %w", err)
	}

	d.Log.Infof("Sincronizadas %d operaciones de stock hacia el remoto. Stock remoto actualizado para %d productos.", len(ops), len(idsAfectados))
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
	var ultimaFacturaLocal time.Time
	var ultimaFacturaNull sql.NullTime

	err = tx.QueryRowContext(ctx, "SELECT MAX(created_at) FROM facturas").Scan(&ultimaFacturaNull)
	if err != nil {
		return fmt.Errorf("error al obtener la última factura local: %w", err)
	}

	if ultimaFacturaNull.Valid {
		ultimaFacturaLocal = ultimaFacturaNull.Time
	} else {
		ultimaFacturaLocal = time.Unix(0, 0)
	}

	facturasRemotasQuery := `SELECT id, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at FROM facturas WHERE created_at > $1`
	rows, err := d.RemoteDB.Query(ctx, facturasRemotasQuery, ultimaFacturaLocal)
	if err != nil {
		return fmt.Errorf("error obteniendo facturas remotas: %w", err)
	}
	defer rows.Close()

	facturaStmt, err := tx.PrepareContext(ctx, `INSERT INTO facturas (id, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("error al preparar statement de facturas: %w", err)
	}
	defer facturaStmt.Close()

	facturaCount := 0
	for rows.Next() {
		var f Factura
		if err := rows.Scan(&f.ID, &f.NumeroFactura, &f.FechaEmision, &f.VendedorID, &f.ClienteID, &f.Subtotal, &f.IVA, &f.Total, &f.Estado, &f.MetodoPago, &f.CreatedAt, &f.UpdatedAt); err != nil {
			d.Log.Errorf("Error al escanear factura remota: %v", err)
			continue
		}
		_, err := facturaStmt.ExecContext(ctx, f.ID, f.NumeroFactura, f.FechaEmision, f.VendedorID, f.ClienteID, f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt)
		if err != nil {
			d.Log.Errorf("Error al insertar factura local: %v", err)
			continue
		}
		facturaCount++
	}
	if facturaCount > 0 {
		d.Log.Infof("Sincronizadas %d nuevas facturas a local.", facturaCount)
	}

	// --- Sincronizar Compras ---
	var ultimaCompraLocal time.Time
	var ultimaCompraNull sql.NullTime

	err = tx.QueryRowContext(ctx, "SELECT MAX(created_at) FROM compras").Scan(&ultimaCompraNull)
	if err != nil {
		return fmt.Errorf("error al obtener la última compra local: %w", err)
	}

	if ultimaCompraNull.Valid {
		ultimaCompraLocal = ultimaCompraNull.Time
	} else {
		ultimaCompraLocal = time.Unix(0, 0)
	}

	comprasRemotasQuery := `SELECT id, fecha, proveedor_id, factura_numero, total, created_at, updated_at FROM compras WHERE created_at > $1`
	compraRows, err := d.RemoteDB.Query(ctx, comprasRemotasQuery, ultimaCompraLocal)
	if err != nil {
		return fmt.Errorf("error obteniendo compras remotas: %w", err)
	}
	defer compraRows.Close()

	compraStmt, err := tx.PrepareContext(ctx, `INSERT INTO compras (id, fecha, proveedor_id, factura_numero, total, created_at, updated_at) VALUES (?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`)
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
			continue
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
	query := `SELECT id, created_at, updated_at, deleted_at, nombre, codigo, precio_venta, stock FROM productos WHERE id = ?`
	err := d.LocalDB.QueryRowContext(d.ctx, query, id).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Nombre, &p.Codigo, &p.PrecioVenta, &p.Stock)
	if err != nil {
		d.Log.Errorf("syncProductoToRemote: no se encontró producto local ID %d: %v", id, err)
		return
	}

	upsertSQL := `
		INSERT INTO productos (id, created_at, updated_at, deleted_at, nombre, codigo, precio_venta, stock)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (codigo) DO UPDATE SET
			nombre = EXCLUDED.nombre, precio_venta = EXCLUDED.precio_venta, stock = EXCLUDED.stock,
			updated_at = EXCLUDED.updated_at, deleted_at = EXCLUDED.deleted_at;`

	_, err = d.RemoteDB.Exec(d.ctx, upsertSQL, p.ID, p.CreatedAt, p.UpdatedAt, p.DeletedAt, p.Nombre, p.Codigo, p.PrecioVenta, p.Stock)
	if err != nil {
		d.Log.Errorf("Error en UPSERT de producto remoto ID %d: %v", id, err)
		return
	}
	d.Log.Infof("Sincronizado producto individual ID %d hacia el remoto.", id)
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

	// Obtener factura y detalles desde local
	f, err := d.ObtenerDetalleFactura(id)
	if err != nil {
		d.Log.Errorf("syncVentaToRemote: no se pudo obtener factura local ID %d: %v", id, err)
		return
	}

	// Asegurar Vendedor y Cliente en remoto
	d.syncVendedorToRemote(f.VendedorID)
	d.syncClienteToRemote(f.ClienteID)

	// Abrir transacción remota
	rtx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		d.Log.Errorf("syncVentaToRemote: no se pudo iniciar tx remota: %v", err)
		return
	}
	defer rtx.Rollback(d.ctx)

	// Upsert factura (por numero_factura / id)
	upsertFactura := `
		INSERT INTO facturas (id, numero_factura, fecha_emision, vendedor_id, cliente_id, subtotal, iva, total, estado, metodo_pago, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			numero_factura = EXCLUDED.numero_factura, fecha_emision = EXCLUDED.fecha_emision, vendedor_id = EXCLUDED.vendedor_id,
			cliente_id = EXCLUDED.cliente_id, subtotal = EXCLUDED.subtotal, iva = EXCLUDED.iva, total = EXCLUDED.total,
			estado = EXCLUDED.estado, metodo_pago = EXCLUDED.metodo_pago, updated_at = EXCLUDED.updated_at;
	`
	_, err = rtx.Exec(d.ctx, upsertFactura, int64(f.ID), f.NumeroFactura, f.FechaEmision, int64(f.VendedorID), int64(f.ClienteID), f.Subtotal, f.IVA, f.Total, f.Estado, f.MetodoPago, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		d.Log.Errorf("syncVentaToRemote: error upserting factura remota: %v", err)
		return
	}

	// Borrar detalles existentes de esta factura en remoto y reinsertar (asegura consistencia)
	if _, err := rtx.Exec(d.ctx, "DELETE FROM detalle_facturas WHERE factura_id = $1", int64(f.ID)); err != nil {
		d.Log.Errorf("syncVentaToRemote: error borrando detalles remotos: %v", err)
		return
	}
	// Reinsertar detalles
	if len(f.Detalles) > 0 {
		_, err := rtx.CopyFrom(d.ctx,
			pgx.Identifier{"detalle_facturas"},
			[]string{"factura_id", "producto_id", "cantidad", "precio_unitario", "precio_total"},
			pgx.CopyFromSlice(len(f.Detalles), func(i int) ([]interface{}, error) {
				det := f.Detalles[i]
				return []interface{}{int64(f.ID), int64(det.ProductoID), det.Cantidad, det.PrecioUnitario, det.PrecioTotal}, nil
			}),
		)
		if err != nil {
			d.Log.Errorf("syncVentaToRemote: error reinsertando detalles en remoto: %v", err)
			return
		}
	}

	// Sincronizar las operaciones de stock asociadas (si existen) desde local -> remoto
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
				// no return; intentamos continuar con commit
			}
			// actualizar cache stock remoto para producto(s) de la factura
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

func (d *Db) syncVendedorToLocal(v Vendedor) error {
	// upsert pattern for sqlite: try update, if rows affected==0 then insert
	res, err := d.LocalDB.Exec("UPDATE vendedors SET nombre=?, apellido=?, cedula=?, email=?, contrasena=?, mfa_enabled=?, updated_at=? WHERE id=?", v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled, time.Now(), v.ID)
	if err != nil {
		return err
	}
	r, _ := res.RowsAffected()
	if r == 0 {
		_, err = d.LocalDB.Exec("INSERT INTO vendedors (id, nombre, apellido, cedula, email, contrasena, mfa_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", v.ID, v.Nombre, v.Apellido, v.Cedula, v.Email, v.Contrasena, v.MFAEnabled, time.Now(), time.Now())
		if err != nil {
			return err
		}
	}
	return nil
}

// Helpers:

func (d *Db) calcularStockRealLocal(productoID uint) int {
	var stockCalculado int
	query := "SELECT COALESCE(SUM(cantidad_cambio), 0) FROM operacion_stocks WHERE producto_id = ?"
	err := d.LocalDB.QueryRowContext(d.ctx, query, productoID).Scan(&stockCalculado)
	if err != nil {
		d.Log.Errorf("Error al calcular stock real local para producto ID %d: %v", productoID, err)
		return 0
	}
	return stockCalculado
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
	setDefault("updated_at", now)
	setDefault("nombre", "SIN NOMBRE")

	// Asignar valores por defecto específicos de la tabla
	if tableName == "productos" {
		setDefault("precio_venta", 0.0)
		setDefault("stock", 0)
	}

	return nil
}
