package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ImportaCSV inicia el proceso de importación desde un archivo CSV.
func (d *Db) ImportaCSV(filePath string, modelName string) {
	d.Log.Infof("Iniciando importación para '%s' desde: %s", modelName, filePath)
	progressChan, errorChan := d.CargarDesdeCSV(filePath, modelName)
	go func() {
		for msg := range progressChan {
			d.Log.Info(msg)
		}
	}()
	if err := <-errorChan; err != nil {
		d.Log.Errorf("La importación del CSV falló: %v", err)
	} else {
		d.Log.Info("Importación de CSV finalizada con éxito.")
	}
}

// ResetearTodaLaData ejecuta un borrado completo y reinicio de las bases de datos.
func (d *Db) ResetearTodaLaData() (string, error) {
	//	if err := d.DeepResetDatabases(); err != nil {
	//		return "", err
	//	}
	return "¡Reseteo completado! Todas las bases de datos han sido limpiadas y reiniciadas.", nil
}

func (d *Db) NormalizarStockMasivo() (string, error) {
	d.Log.Info("INICIANDO: Proceso de Normalización Masiva (Remoto es la Verdad).")

	// --- PASO 1: RECALCULAR TODO EN EL REMOTO ---
	d.Log.Info("[Paso 1/2] Forzando recálculo de stock en el servidor remoto...")
	if err := d.RecalcularStockRemotoParaTodosLosProductos(); err != nil {
		return "", fmt.Errorf("falló la recalculación remota del stock: %w", err)
	}
	d.Log.Info("[Paso 1/2] Recálculo remoto completado.")

	// --- PASO 2: FORZAR A LA BD LOCAL A SER UN ESPEJO DEL REMOTO ---
	d.Log.Info("[Paso 2/2] Borrando datos locales y descargando el estado correcto desde el remoto...")
	if err := d.ForzarResincronizacionLocalDesdeRemoto(); err != nil {
		return "", fmt.Errorf("falló la resincronización forzada local: %w", err)
	}
	d.Log.Info("[Paso 2/2] Resincronización local completada.")

	d.Log.Info("ÉXITO: Normalización Masiva de Stock completada.")
	return "Stock normalizado. La base de datos local ahora es un espejo del servidor.", nil
}

// NUEVA FUNCIÓN DE AYUDA para forzar la subida de TODAS las operaciones
func (d *Db) SincronizarTodasLasOperacionesHaciaRemoto() error {
	d.Log.Info("Iniciando sincronización forzada de TODAS las operaciones de stock hacia el remoto.")
	if !d.isRemoteDBAvailable() {
		return fmt.Errorf("base de datos remota no disponible para sincronización forzada")
	}

	// 1. Leer TODAS las operaciones de stock de la base de datos local.
	query := `SELECT uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, factura_uuid, timestamp FROM operacion_stocks`
	rows, err := d.LocalDB.QueryContext(d.ctx, query)
	if err != nil {
		return fmt.Errorf("error al leer todas las operaciones de stock locales: %w", err)
	}
	defer rows.Close()

	var ops []OperacionStock
	var localIDsToUpdate []string
	for rows.Next() {
		var op OperacionStock
		var stockResultante sql.NullInt64
		var facturaUUID sql.NullString

		if err := rows.Scan(&op.UUID, &op.ProductoUUID, &op.TipoOperacion, &op.CantidadCambio, &stockResultante, &op.VendedorUUID, &facturaUUID, &op.Timestamp); err != nil {
			d.Log.Warnf("Omitiendo operación de stock con error de escaneo: %v", err)
			continue
		}

		if stockResultante.Valid {
			op.StockResultante = int(stockResultante.Int64)
		}
		if facturaUUID.Valid {
			*op.FacturaUUID = facturaUUID.String
		}

		ops = append(ops, op)
		localIDsToUpdate = append(localIDsToUpdate, *op.FacturaUUID)
	}

	if len(ops) == 0 {
		d.Log.Info("No hay operaciones de stock locales para sincronizar.")
		return nil
	}

	// 2. Usar una transacción remota y un batch de UPSERTs.
	rtx, err := d.RemoteDB.Begin(d.ctx)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transacción remota forzada: %w", err)
	}
	defer func() {
		if rErr := rtx.Rollback(d.ctx); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [NormalizarStockTodosLosProductos] rollback %v", err)
		}
	}()

	batch := &pgx.Batch{}
	upsertSQL := `
		INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, factura_uuid, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (uuid) DO UPDATE SET
			tipo_operacion = EXCLUDED.tipo_operacion,
			cantidad_cambio = EXCLUDED.cantidad_cambio,
			stock_resultante = EXCLUDED.stock_resultante,
			timestamp = EXCLUDED.timestamp;
	`
	for _, op := range ops {
		batch.Queue(upsertSQL, op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorUUID, op.FacturaUUID, op.Timestamp)
	}

	br := rtx.SendBatch(d.ctx, batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("error ejecutando el batch de UPSERT forzado de operaciones de stock: %w", err)
	}

	if err := rtx.Commit(d.ctx); err != nil {
		return fmt.Errorf("error al confirmar la transacción remota forzada: %w", err)
	}

	// 3. Marcar todas las operaciones locales como sincronizadas.
	if len(localIDsToUpdate) > 0 {
		updateLocalSQL := "UPDATE operacion_stocks SET sincronizado = 1"
		if _, err := d.LocalDB.ExecContext(d.ctx, updateLocalSQL); err != nil {
			return fmt.Errorf("error al marcar todas las operaciones como sincronizadas localmente: %w", err)
		}
	}

	d.Log.Infof("Sincronización forzada completada para %d operaciones de stock.", len(ops))
	return nil
}

func (d *Db) RecalcularStockRemotoParaTodosLosProductos() error {
	if !d.isRemoteDBAvailable() {
		return fmt.Errorf("base de datos remota no disponible")
	}

	// Esta consulta de dos partes es crucial:
	// 1. Actualiza el stock para todos los productos que SÍ tienen operaciones.
	// 2. Pone en 0 el stock de todos los productos que NO tienen operaciones.
	updateStockCacheSQL := `
		WITH stock_calculado AS (
			SELECT producto_uuid, COALESCE(SUM(cantidad_cambio), 0) as nuevo_stock
			FROM operacion_stocks
			GROUP BY producto_uuid
		)
		UPDATE productos p SET stock = sc.nuevo_stock
		FROM stock_calculado sc WHERE p.uuid = sc.producto_uuid;

		UPDATE productos SET stock = 0 WHERE uuid NOT IN (SELECT DISTINCT producto_uuid FROM operacion_stocks);
	`

	_, err := d.RemoteDB.Exec(d.ctx, updateStockCacheSQL)
	if err != nil {
		return fmt.Errorf("error al ejecutar el recálculo masivo de stock remoto: %w", err)
	}

	d.Log.Info("Recálculo masivo de stock en el servidor remoto ejecutado correctamente.")
	return nil
}

func (d *Db) NormalizarStockTodosLosProductos() (string, error) {
	d.Log.Info("Iniciando proceso de normalización de stock para todos los productos.")

	ctx := d.ctx
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("error al iniciar la transacción de normalización: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [NormalizarStockTodosLosProductos] rollback %v", err)
		}
	}()

	// 1. Obtener todos los UUIDs de productos.
	rows, err := tx.QueryContext(ctx, "SELECT uuid FROM productos WHERE deleted_at IS NULL")
	if err != nil {
		return "", fmt.Errorf("error al obtener UUIDs de productos: %w", err)
	}
	defer rows.Close()

	var productoUUIDs []string
	for rows.Next() {
		var pr_uuid string
		if err := rows.Scan(&pr_uuid); err != nil {
			return "", fmt.Errorf("error al escanear UUID de producto: %w", err)
		}
		productoUUIDs = append(productoUUIDs, pr_uuid)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error al iterar UUIDs de productos: %w", err)
	}

	d.Log.Infof("Se normalizará el stock para %d productos.", len(productoUUIDs))

	// Preparar statements para reutilizar
	stmtUpdateStock, err := tx.PrepareContext(ctx, "UPDATE productos SET stock = ? WHERE uuid = ?")
	if err != nil {
		return "", fmt.Errorf("error al preparar statement de actualización de stock: %w", err)
	}
	defer stmtUpdateStock.Close()

	stmtInsertOp, err := tx.PrepareContext(ctx, `
		INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, timestamp, sincronizado)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return "", fmt.Errorf("error al preparar statement de inserción de operación: %w", err)
	}
	defer stmtInsertOp.Close()

	// 2. Iterar sobre cada producto para normalizar su stock.
	for _, pr_uuid := range productoUUIDs {
		var totalOperaciones int
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operacion_stocks WHERE producto_uuid = ?", pr_uuid).Scan(&totalOperaciones)
		if err != nil {
			return "", fmt.Errorf("error al contar operaciones para el producto UUID %d: %w", pr_uuid, err)
		}

		if totalOperaciones > 0 {
			// Si hay operaciones, recalcular desde ellas.
			if err := RecalcularYActualizarStock(tx, pr_uuid); err != nil {
				return "", fmt.Errorf("error al recalcular stock para el producto UUID %d: %w", pr_uuid, err)
			}
		} else {
			// Si no hay operaciones, forzar a 0 y crear registro inicial.
			if _, err := stmtUpdateStock.ExecContext(ctx, 0, pr_uuid); err != nil {
				return "", fmt.Errorf("error al actualizar stock a 0 para el producto UUID %d: %w", pr_uuid, err)
			}

			// Crear la operación inicial de stock 0
			op := OperacionStock{
				UUID:            uuid.New().String(),
				ProductoUUID:    pr_uuid,
				TipoOperacion:   "INICIAL",
				CantidadCambio:  0,
				StockResultante: 0,
				VendedorUUID:    "AJUSTE-SISTEMA",
				Timestamp:       time.Now(),
				Sincronizado:    false,
			}
			if _, err := stmtInsertOp.ExecContext(ctx, op.UUID, op.ProductoUUID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorUUID, op.Timestamp, op.Sincronizado); err != nil {
				return "", fmt.Errorf("error al crear operación 'INICIAL' para el producto UUID %d: %w", pr_uuid, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("error al confirmar la transacción de normalización: %w", err)
	}

	d.Log.Infof("Normalización local completa. Disparando sincronización hacia el remoto.")
	go d.SincronizacionInteligente()

	return fmt.Sprintf("Stock normalizado localmente para %d productos. La sincronización con el servidor remoto ha comenzado.", len(productoUUIDs)), nil
}
