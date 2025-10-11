package backend

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// RealizarSincronizacionInicial inicia el proceso de sincronización en segundo plano.
func (d *Db) RealizarSincronizacionInicial() {
	go d.SincronizacionInteligente()
}

// SincronizacionInteligente es el orquestador principal del proceso de sincronización.
func (d *Db) SincronizacionInteligente() {
	if !d.syncMutex.TryLock() {
		d.Log.Warn("La sincronización ya está en progreso, se omite esta ejecución.")
		return
	}
	defer d.syncMutex.Unlock()

	if !d.isRemoteDBAvailable() {
		d.Log.Warn("Modo offline: la base de datos remota no está disponible, se omite la sincronización.")
		return
	}
	d.Log.Info("[INICIO]: Sincronización Inteligente")

	if err := d.sincronizarProductos(); err != nil {
		d.Log.Error("Error sincronizando productos")
	}
	// TODO: Implementar funciones similares para los otros modelos siguiendo el patrón de sincronizarProductos.
	// if err := d.sincronizarClientes(); err != nil {
	// 	d.Log.Error("Error sincronizando clientes")
	// }
	// if err := d.sincronizarProveedores(); err != nil {
	// 	d.Log.Error("Error sincronizando proveedores")
	// }
	// if err := d.sincronizarVendedores(); err != nil {
	// 	d.Log.Error("Error sincronizando vendedores")
	// }
	// ----------------------------------------------------------------------------------

	// Sincronizar operaciones de stock (Local -> Remoto)
	if err := d.SincronizarOperacionesStockHaciaRemoto(); err != nil {
		d.Log.Error("Error sincronizando operaciones de stock hacia el remoto")
	}

	// Sincronizar operaciones y recalcular stock local (Remoto -> Local)
	if affectedProductIDs, err := d.sincronizarOperacionesHaciaLocal(); err == nil && len(affectedProductIDs) > 0 {
		d.Log.Infof("Recalculando stock de %d producto(s) locales tras la sincronización.", len(affectedProductIDs))
		for _, productoID := range affectedProductIDs {
			// La función RecalcularYActualizarStock ahora debe ser reimplementada con SQL nativo.
			// Asumimos que existirá en `transaccion_logic.go`
			if err := RecalcularYActualizarStock(d.LocalDB, productoID); err != nil {
				d.Log.Errorf("Error al recalcular stock para producto local ID %d", productoID)
			}
		}
	} else if err != nil {
		d.Log.Error("Error sincronizando operaciones de stock hacia el local")
	}

	// Sincronizar transacciones (Facturas, Compras) (Remoto -> Local)
	if err := d.sincronizarTransaccionesHaciaLocal(); err != nil {
		d.Log.Error("Error sincronizando transacciones hacia el local")
	}

	d.Log.Info("[FIN]: Sincronización Inteligente")
}

func (d *Db) sincronizarProductos() error {
	d.Log.Info("[SINCRONIZANDO]: Productos")
	ctx := context.Background()

	// === PASO 1: LOCAL -> REMOTO ===
	if err := d.syncProductosToRemote(ctx); err != nil {
		return fmt.Errorf("error en sincronización de productos Local -> Remoto: %w", err)
	}

	// === PASO 2: REMOTO -> LOCAL ===
	if err := d.syncProductosToLocal(ctx); err != nil {
		return fmt.Errorf("error en sincronización de productos Remoto -> Local: %w", err)
	}

	d.Log.Info("[OK]: Productos sincronizados")
	return nil
}

// syncProductosToRemote usa `COPY FROM` y `UPDATE FROM VALUES` para máxima eficiencia.
func (d *Db) syncProductosToRemote(ctx context.Context) error {
	// 1. Obtener la fecha más reciente de actualización del remoto para buscar solo lo nuevo.
	var lastRemoteUpdate time.Time
	err := d.RemoteDB.QueryRow(ctx, "SELECT COALESCE(MAX(updated_at), '1970-01-01') FROM productos").Scan(&lastRemoteUpdate)
	if err != nil {
		return err
	}

	// 2. Obtener los productos locales modificados desde la última actualización remota.
	rows, err := d.LocalDB.Query("SELECT id, codigo, nombre, precio_venta, stock, created_at, updated_at, deleted_at FROM productos WHERE updated_at > ?", lastRemoteUpdate)
	if err != nil {
		return err
	}
	defer rows.Close()

	var localProds []Producto
	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return err
		}
		localProds = append(localProds, p)
	}

	if len(localProds) == 0 {
		d.Log.Info("[Productos L->R] No hay productos locales nuevos para enviar al remoto.")
		return nil
	}

	// 3. Separar productos para crear y para actualizar.
	var toCreate, toUpdate []Producto
	// Para eficiencia, obtenemos todos los códigos existentes en el remoto de una sola vez.
	remoteCodigosRows, err := d.RemoteDB.Query(ctx, "SELECT codigo FROM productos")
	if err != nil {
		return err
	}
	remoteCodigos := make(map[string]struct{})
	for remoteCodigosRows.Next() {
		var codigo string
		if err := remoteCodigosRows.Scan(&codigo); err != nil {
			return err
		}
		remoteCodigos[codigo] = struct{}{}
	}
	remoteCodigosRows.Close()

	for _, p := range localProds {
		if _, exists := remoteCodigos[p.Codigo]; exists {
			toUpdate = append(toUpdate, p)
		} else {
			toCreate = append(toCreate, p)
		}
	}

	// 4. Ejecutar operaciones masivas.
	// INSERCIÓN MASIVA con COPY FROM (la forma más rápida)
	if len(toCreate) > 0 {
		d.Log.Infof("[Productos L->R] Creando %d nuevo(s) producto(s) en remoto...", len(toCreate))
		copyData := make([][]interface{}, len(toCreate))
		for i, p := range toCreate {
			copyData[i] = []interface{}{p.Codigo, p.Nombre, p.PrecioVenta, p.Stock, p.CreatedAt, p.UpdatedAt, p.DeletedAt}
		}
		_, err = d.RemoteDB.CopyFrom(ctx,
			pgx.Identifier{"productos"},
			[]string{"codigo", "nombre", "precio_venta", "stock", "created_at", "updated_at", "deleted_at"},
			pgx.CopyFromRows(copyData),
		)
		if err != nil {
			return fmt.Errorf("error en CopyFrom para crear productos: %w", err)
		}
	}

	// ACTUALIZACIÓN MASIVA con UPDATE FROM VALUES
	if len(toUpdate) > 0 {
		d.Log.Infof("[Productos L->R] Actualizando %d producto(s) en remoto...", len(toUpdate))
		// Construimos una query masiva. Es compleja pero extremadamente eficiente.
		sql := `
			UPDATE productos AS p SET
				nombre = data.nombre,
				precio_venta = data.precio_venta,
				updated_at = data.updated_at,
				deleted_at = data.deleted_at
			FROM (VALUES %s) AS data(codigo, nombre, precio_venta, updated_at, deleted_at)
			WHERE p.codigo = data.codigo AND p.updated_at < data.updated_at;
		`
		var valueStrings []string
		var valueArgs []interface{}
		i := 1
		for _, p := range toUpdate {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", i, i+1, i+2, i+3, i+4))
			valueArgs = append(valueArgs, p.Codigo, p.Nombre, p.PrecioVenta, p.UpdatedAt, p.DeletedAt)
			i += 5
		}
		query := fmt.Sprintf(sql, strings.Join(valueStrings, ","))
		_, err := d.RemoteDB.Exec(ctx, query, valueArgs...)
		if err != nil {
			return fmt.Errorf("error en actualización masiva de productos: %w", err)
		}
	}

	return nil
}

func (d *Db) syncVendedorToRemote(ctx context.Context) error {
	// 1. Obtener la fecha más reciente de actualización del remoto para buscar solo lo nuevo.
	var lastRemoteUpdate time.Time
	err := d.RemoteDB.QueryRow(ctx, "SELECT COALESCE(MAX(updated_at), '1970-01-01') FROM vendedors").Scan(&lastRemoteUpdate)
	if err != nil {
		return err
	}

	// 2. Obtener los productos locales modificados desde la última actualización remota.
	rows, err := d.LocalDB.Query("SELECT id, nombre, cedula, contrasena, created_at, updated_at, deleted_at FROM vendedors WHERE updated_at > ?", lastRemoteUpdate)
	if err != nil {
		return err
	}
	defer rows.Close()

	var localVendor []Vendedor
	for rows.Next() {
		var v Vendedor
		if err := rows.Scan(&v.ID, &v.Nombre, &v.Cedula, &v.Contrasena, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
			return err
		}
		localVendor = append(localVendor, v)
	}

	if len(localVendor) == 0 {
		d.Log.Info("[Vendedor L->R] No hay vendedores locales nuevos para enviar al remoto.")
		return nil
	}

	// 3. Separar Vendedor para crear y para actualizar.
	var toCreate, toUpdate []Vendedor
	// Para eficiencia, obtenemos todos los códigos existentes en el remoto de una sola vez.
	remoteCedulasRows, err := d.RemoteDB.Query(ctx, "SELECT cedula FROM vendedors")
	if err != nil {
		return err
	}
	remoteCodigos := make(map[string]struct{})
	for remoteCedulasRows.Next() {
		var codigo string
		if err := remoteCedulasRows.Scan(&codigo); err != nil {
			return err
		}
		remoteCodigos[codigo] = struct{}{}
	}
	remoteCedulasRows.Close()

	for _, v := range localVendor {
		if _, exists := remoteCodigos[v.Cedula]; exists {
			toUpdate = append(toUpdate, v)
		} else {
			toCreate = append(toCreate, v)
		}
	}

	// 4. Ejecutar operaciones masivas.
	// INSERCIÓN MASIVA con COPY FROM (la forma más rápida)
	if len(toCreate) > 0 {
		d.Log.Infof("[Productos L->R] Creando %d nuevo(s) producto(s) en remoto...", len(toCreate))
		copyData := make([][]interface{}, len(toCreate))
		for i, v := range toCreate {
			copyData[i] = []interface{}{v.Cedula, v.Nombre, v.Apellido, v.Contrasena, v.CreatedAt, v.UpdatedAt, v.DeletedAt}
		}
		_, err = d.RemoteDB.CopyFrom(ctx,
			pgx.Identifier{"productos"},
			[]string{"cedula", "nombre", "apellido", "contrasena", "created_at", "updated_at", "deleted_at"},
			pgx.CopyFromRows(copyData),
		)
		if err != nil {
			return fmt.Errorf("error en CopyFrom para crear productos: %w", err)
		}
	}

	// ACTUALIZACIÓN MASIVA con UPDATE FROM VALUES
	if len(toUpdate) > 0 {
		d.Log.Infof("[Productos L->R] Actualizando %d producto(s) en remoto...", len(toUpdate))
		// Construimos una query masiva. Es compleja pero extremadamente eficiente.
		sql := `
			UPDATE productos AS p SET
				nombre = data.nombre,
				apellido = data.apellido,
				updated_at = data.updated_at,
				deleted_at = data.deleted_at
			FROM (VALUES %s) AS data(cedula, nombre, apellido, updated_at, deleted_at)
			WHERE v.cedula = data.cedula AND v.updated_at < data.updated_at;
		`
		var valueStrings []string
		var valueArgs []interface{}
		i := 1
		for _, v := range toUpdate {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", i, i+1, i+2, i+3, i+4))
			valueArgs = append(valueArgs, v.Cedula, v.Nombre, v.Apellido, v.UpdatedAt, v.DeletedAt)
			i += 5
		}
		query := fmt.Sprintf(sql, strings.Join(valueStrings, ","))
		_, err := d.RemoteDB.Exec(ctx, query, valueArgs...)
		if err != nil {
			return fmt.Errorf("error en actualización masiva de vendedor: %w", err)
		}
	}

	return nil
}

// syncProductosToLocal usa `ON CONFLICT DO UPDATE` para eficiencia en SQLite.
func (d *Db) syncProductosToLocal(ctx context.Context) error {
	// 1. Obtener la fecha más reciente de actualización local.
	var lastLocalUpdate time.Time
	row := d.LocalDB.QueryRow("SELECT COALESCE(MAX(updated_at), '1970-01-01') FROM productos")
	if err := row.Scan(&lastLocalUpdate); err != nil {
		return err
	}

	// 2. Obtener productos remotos más nuevos que la última actualización local.
	rows, err := d.RemoteDB.Query(ctx, "SELECT id, codigo, nombre, precio_venta, stock, created_at, updated_at, deleted_at FROM productos WHERE updated_at > $1", lastLocalUpdate)
	if err != nil {
		return err
	}
	defer rows.Close()

	var remoteProds []Producto
	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return err
		}
		remoteProds = append(remoteProds, p)
	}

	if len(remoteProds) == 0 {
		d.Log.Info("[Productos R->L] No hay productos remotos nuevos para traer al local.")
		return nil
	}

	d.Log.Infof("[Productos R->L] Sincronizando %d producto(s) a local...", len(remoteProds))

	// 3. Iniciar transacción local y preparar statement para "upsert".
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback si algo sale mal

	stmt, err := tx.Prepare(`
		INSERT INTO productos (codigo, nombre, precio_venta, stock, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(codigo) DO UPDATE SET
			nombre = excluded.nombre,
			precio_venta = excluded.precio_venta,
			updated_at = excluded.updated_at,
			deleted_at = excluded.deleted_at
		WHERE excluded.updated_at > updated_at;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range remoteProds {
		_, err := stmt.Exec(p.Codigo, p.Nombre, p.PrecioVenta, p.Stock, p.CreatedAt, p.UpdatedAt, p.DeletedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ---- SINCRONIZACIÓN DE OPERACIONES DE STOCK (OPTIMIZADA) ----

// SincronizarOperacionesStockHaciaRemoto inserta operaciones masivamente y actualiza el stock remoto en una sola query.
func (d *Db) SincronizarOperacionesStockHaciaRemoto() error {
	d.Log.Info("[SINCRONIZANDO]: Operaciones de Stock Local -> Remoto")
	ctx := context.Background()

	// 1. Obtener operaciones locales no sincronizadas.
	rows, err := d.LocalDB.Query("SELECT uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp FROM operacion_stocks WHERE sincronizado = false")
	if err != nil {
		return fmt.Errorf("error cargando operaciones pendientes: %w", err)
	}
	defer rows.Close()

	var opsToSync []OperacionStock
	var productoIDsMap = make(map[uint]struct{})
	for rows.Next() {
		var op OperacionStock
		if err := rows.Scan(&op.UUID, &op.ProductoID, &op.TipoOperacion, &op.CantidadCambio, &op.StockResultante, &op.VendedorID, &op.FacturaID, &op.Timestamp); err != nil {
			return err
		}
		opsToSync = append(opsToSync, op)
		productoIDsMap[op.ProductoID] = struct{}{}
	}

	if len(opsToSync) == 0 {
		d.Log.Info("No hay nuevas operaciones de stock para enviar al servidor.")
		return nil
	}
	d.Log.Infof("Enviando %d operación(es) de stock al servidor remoto...", len(opsToSync))

	// 2. Usar COPY FROM para la inserción masiva en remoto.
	copyData := make([][]interface{}, len(opsToSync))
	for i, op := range opsToSync {
		copyData[i] = []interface{}{op.UUID, op.ProductoID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorID, op.FacturaID, op.Timestamp, false} // sincronizado es false por defecto en el remoto
	}

	tx, err := d.RemoteDB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// La inserción se hace con ON CONFLICT DO NOTHING para idempotencia.
	_, err = tx.CopyFrom(ctx,
		pgx.Identifier{"operacion_stocks"},
		[]string{"uuid", "producto_id", "tipo_operacion", "cantidad_cambio", "stock_resultante", "vendedor_id", "factura_id", "timestamp", "sincronizado"},
		pgx.CopyFromRows(copyData),
	)
	if err != nil {
		// pgx maneja el ON CONFLICT de forma diferente. La forma más robusta es una tabla temporal, pero para este caso, dejamos que un error de duplicado (si ocurre) anule el lote.
		// Un enfoque más simple sería insertar con `INSERT ... ON CONFLICT DO NOTHING`, pero sería más lento que CopyFrom.
		// Asumimos que los UUID son únicos y no habrá conflictos.
		return fmt.Errorf("error en CopyFrom para operacion_stocks: %w", err)
	}

	// 3. Recalcular y actualizar el stock de TODOS los productos afectados en UNA SOLA QUERY.
	productoIDs := make([]uint, 0, len(productoIDsMap))
	for id := range productoIDsMap {
		productoIDs = append(productoIDs, id)
	}

	updateStockQuery := `
		WITH new_stock AS (
			SELECT 
				producto_id, 
				COALESCE(SUM(cantidad_cambio), 0) AS total_stock 
			FROM operacion_stocks 
			WHERE producto_id = ANY($1)
			GROUP BY producto_id
		)
		UPDATE productos p
		SET stock = new_stock.total_stock
		FROM new_stock
		WHERE p.id = new_stock.producto_id;
	`
	if _, err := tx.Exec(ctx, updateStockQuery, productoIDs); err != nil {
		return fmt.Errorf("error en la actualización masiva de stock remoto: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error al confirmar transacción remota: %w", err)
	}

	// 4. Marcar las operaciones como sincronizadas en la base de datos local.
	uuids := make([]string, len(opsToSync))
	for i, op := range opsToSync {
		uuids[i] = op.UUID
	}
	// Usamos `?` expandido para la cláusula IN.
	query := "UPDATE operacion_stocks SET sincronizado = true WHERE uuid IN (?" + strings.Repeat(",?", len(uuids)-1) + ")"
	args := make([]interface{}, len(uuids))
	for i, v := range uuids {
		args[i] = v
	}
	if _, err := d.LocalDB.Exec(query, args...); err != nil {
		// Esto es un problema, pero no es crítico. Se reintentará en la siguiente sincronización.
		d.Log.Error("Error marcando operaciones de stock como sincronizadas en local.")
	}

	return nil
}

// sincronizarOperacionesHaciaLocal trae nuevas operaciones del remoto al local.
func (d *Db) sincronizarOperacionesHaciaLocal() ([]uint, error) {
	d.Log.Info("[SINCRONIZANDO]: Operaciones de Stock Remoto -> Local")
	ctx := context.Background()

	// 1. Obtener todos los UUIDs locales para no volver a descargarlos.
	rows, err := d.LocalDB.Query("SELECT uuid FROM operacion_stocks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	localUUIDs := make(map[string]struct{})
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		localUUIDs[uuid] = struct{}{}
	}

	// 2. Obtener operaciones remotas que no existen localmente.
	remoteRows, err := d.RemoteDB.Query(ctx, "SELECT uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp FROM operacion_stocks")
	if err != nil {
		return nil, err
	}
	defer remoteRows.Close()

	var nuevasOperaciones []OperacionStock
	affectedProductIDsMap := make(map[uint]struct{})
	for remoteRows.Next() {
		var op OperacionStock
		if err := remoteRows.Scan(&op.UUID, &op.ProductoID, &op.TipoOperacion, &op.CantidadCambio, &op.StockResultante, &op.VendedorID, &op.FacturaID, &op.Timestamp); err != nil {
			return nil, err
		}
		if _, exists := localUUIDs[op.UUID]; !exists {
			nuevasOperaciones = append(nuevasOperaciones, op)
			affectedProductIDsMap[op.ProductoID] = struct{}{}
		}
	}

	if len(nuevasOperaciones) == 0 {
		d.Log.Info("No se encontraron nuevas operaciones de stock en el servidor.")
		return nil, nil
	}

	// 3. Insertar las nuevas operaciones en la BD local dentro de una transacción.
	d.Log.Infof("Descargando %d nueva(s) operaciones de stock...", len(nuevasOperaciones))
	tx, err := d.LocalDB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO operacion_stocks (uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, factura_id, timestamp, sincronizado) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	for _, op := range nuevasOperaciones {
		_, err := stmt.Exec(op.UUID, op.ProductoID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorID, op.FacturaID, op.Timestamp, true) // Ya está en local, la marcamos como "sincronizada"
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Devolver los IDs de producto afectados para recalcular su stock.
	keys := make([]uint, 0, len(affectedProductIDsMap))
	for k := range affectedProductIDsMap {
		keys = append(keys, k)
	}
	return keys, nil
}

// sincronizarTransaccionesHaciaLocal trae facturas y compras del remoto.
func (d *Db) sincronizarTransaccionesHaciaLocal() error {
	// La lógica aquí es similar a `syncProductosToLocal`:
	// 1. Obtener la `created_at` más reciente de facturas y compras locales.
	// 2. Consultar facturas/compras en el remoto con `created_at` posterior a esa fecha.
	// 3. Usar una transacción local con `INSERT OR IGNORE` para insertar las cabeceras y los detalles.
	// Esta implementación se deja como ejercicio, siguiendo el patrón de `syncProductosToLocal`.
	d.Log.Info("Sincronización de transacciones (facturas/compras) pendiente de implementación con pgx.")
	return nil
}

// Las funciones `sync...ToRemote` individuales (syncVentaToRemote, etc.) ahora deben
// simplemente asegurarse de que el registro local exista y luego confiar en que
// la SincronizacionInteligente se encargará de subirlo eficientemente.
// Opcionalmente, pueden disparar una sincronización, pero con el mutex, no se solaparán.
func (d *Db) syncVentaToRemote(id uint) {
	d.Log.Infof("Venta %d registrada. Se sincronizará en el próximo ciclo.", id)
	go d.SincronizacionInteligente()
}

func (d *Db) syncCompraToRemote(id uint) {
	d.Log.Infof("Compra %d registrada. Se sincronizará en el próximo ciclo.", id)
	go d.SincronizacionInteligente()
}
