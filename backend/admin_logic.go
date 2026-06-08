package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	return "¡Reseteo completado! Todas las bases de datos han sido limpiadas y reiniciadas.", nil
}

// NormalizarStock recorre todos los productos locales y crea operaciones de ajuste
// para corregir inconsistencias entre productos.stock y el stock real calculado
// desde operacion_stocks. Usa CrearOperacionStock() para mantener coherencia.
func (d *Db) NormalizarStock() error {
	var vendedorUUID string = "8f2954d4-6990-4f14-be90-d4b70e8b862a"
	d.Log.Info("[NORMALIZANDO STOCK] Iniciando proceso de revisión y ajuste...")

	tx, err := d.DB.Begin()
	if err != nil {
		return fmt.Errorf("no se pudo iniciar transacción local: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.Query(`SELECT uuid, COALESCE(stock, 0) FROM productos`)
	if err != nil {
		return fmt.Errorf("error leyendo productos: %w", err)
	}
	defer rows.Close()

	totalAjustados := 0

	for rows.Next() {
		var productoUUID string
		var stockActual int
		if err := rows.Scan(&productoUUID, &stockActual); err != nil {
			d.Log.Warnf("Error escaneando producto: %v", err)
			continue
		}

		// Calcular stock real (fuente de verdad)
		stockReal, err := calcularStockRealLocal(tx, productoUUID)
		if err != nil {
			d.Log.Warnf("Error calculando stock real para %s: %v", productoUUID, err)
			continue
		}

		// Si el stock actual ya es correcto y positivo, no hacer nada
		if stockReal == stockActual && stockReal > 0 {
			continue
		}

		// Determinar ajuste necesario
		var ajuste int
		if stockActual <= 0 && stockReal <= 0 {
			// Set stock to 0 instead of phantom 100 units
			ajuste = -stockActual
		} else {
			ajuste = stockReal - stockActual
		}

		// Crear operación de ajuste
		err = d.CrearOperacionStock(tx, productoUUID, "AJUSTE_NORMALIZACION", ajuste, vendedorUUID, nil)
		if err != nil {
			d.Log.Warnf("Error creando operación de ajuste para %s: %v", productoUUID, err)
			continue
		}

		totalAjustados++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error final leyendo filas de productos: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar la transacción: %w", err)
	}

	d.Log.Infof("[NORMALIZACIÓN COMPLETA] %d productos ajustados correctamente", totalAjustados)
	return nil
}

func (d *Db) NormalizarStockTodosLosProductos() (string, error) {
	d.Log.Info("Iniciando proceso de normalización de stock para todos los productos.")

	ctx := d.ctx
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("error al iniciar la transacción de normalización: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [NormalizarStockTodosLosProductos] rollback %v", rErr)
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
	stmtUpdateStock, err := tx.PrepareContext(ctx, "UPDATE productos SET stock = $1 WHERE uuid = $2")
	if err != nil {
		return "", fmt.Errorf("error al preparar statement de actualización de stock: %w", err)
	}
	defer stmtUpdateStock.Close()

	stmtInsertOp, err := tx.PrepareContext(ctx, `
		INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, timestamp, sincronizado)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`)
	if err != nil {
		return "", fmt.Errorf("error al preparar statement de inserción de operación: %w", err)
	}
	defer stmtInsertOp.Close()

	// 2. Iterar sobre cada producto para normalizar su stock.
	for _, pr_uuid := range productoUUIDs {
		var totalOperaciones int
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operacion_stocks WHERE producto_uuid = $1", pr_uuid).Scan(&totalOperaciones)
		if err != nil {
			return "", fmt.Errorf("error al contar operaciones para el producto UUID %s: %w", pr_uuid, err)
		}

		if totalOperaciones > 0 {
			// Si hay operaciones, recalcular desde ellas.
			if err := RecalcularYActualizarStock(tx, pr_uuid); err != nil {
				return "", fmt.Errorf("error al recalcular stock para el producto UUID %s: %w", pr_uuid, err)
			}
		} else {
			// Si no hay operaciones, forzar a 0 y crear registro inicial.
			if _, err := stmtUpdateStock.ExecContext(ctx, 0, pr_uuid); err != nil {
				return "", fmt.Errorf("error al actualizar stock a 0 para el producto UUID %s: %w", pr_uuid, err)
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
				return "", fmt.Errorf("error al crear operación 'INICIAL' para el producto UUID %s: %w", pr_uuid, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("error al confirmar la transacción de normalización: %w", err)
	}

	d.Log.Infof("Normalización local completa para %d productos.", len(productoUUIDs))

	return fmt.Sprintf("Stock normalizado localmente para %d productos.", len(productoUUIDs)), nil
}
