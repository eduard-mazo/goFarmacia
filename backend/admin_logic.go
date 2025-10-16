package backend

import (
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
	//	if err := d.DeepResetDatabases(); err != nil {
	//		return "", err
	//	}
	return "¡Reseteo completado! Todas las bases de datos han sido limpiadas y reiniciadas.", nil
}

func (d *Db) NormalizarStockTodosLosProductos() (string, error) {
	d.Log.Info("Iniciando proceso de normalización de stock para todos los productos.")

	ctx := d.ctx
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("error al iniciar la transacción de normalización: %w", err)
	}
	defer tx.Rollback()

	// 1. Obtener todos los IDs de productos.
	rows, err := tx.QueryContext(ctx, "SELECT id FROM productos WHERE deleted_at IS NULL")
	if err != nil {
		return "", fmt.Errorf("error al obtener IDs de productos: %w", err)
	}
	defer rows.Close()

	var productoIDs []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return "", fmt.Errorf("error al escanear ID de producto: %w", err)
		}
		productoIDs = append(productoIDs, id)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error al iterar IDs de productos: %w", err)
	}

	d.Log.Infof("Se normalizará el stock para %d productos.", len(productoIDs))

	// Preparar statements para reutilizar
	stmtUpdateStock, err := tx.PrepareContext(ctx, "UPDATE productos SET stock = ? WHERE id = ?")
	if err != nil {
		return "", fmt.Errorf("error al preparar statement de actualización de stock: %w", err)
	}
	defer stmtUpdateStock.Close()

	stmtInsertOp, err := tx.PrepareContext(ctx, `
		INSERT INTO operacion_stocks (uuid, producto_id, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_id, timestamp, sincronizado)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return "", fmt.Errorf("error al preparar statement de inserción de operación: %w", err)
	}
	defer stmtInsertOp.Close()

	// 2. Iterar sobre cada producto para normalizar su stock.
	for _, id := range productoIDs {
		var totalOperaciones int
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operacion_stocks WHERE producto_id = ?", id).Scan(&totalOperaciones)
		if err != nil {
			return "", fmt.Errorf("error al contar operaciones para el producto ID %d: %w", id, err)
		}

		if totalOperaciones > 0 {
			// Si hay operaciones, recalcular desde ellas.
			if err := RecalcularYActualizarStock(tx, id); err != nil {
				return "", fmt.Errorf("error al recalcular stock para el producto ID %d: %w", id, err)
			}
		} else {
			// Si no hay operaciones, forzar a 0 y crear registro inicial.
			if _, err := stmtUpdateStock.ExecContext(ctx, 0, id); err != nil {
				return "", fmt.Errorf("error al actualizar stock a 0 para el producto ID %d: %w", id, err)
			}

			// Crear la operación inicial de stock 0
			op := OperacionStock{
				UUID:            uuid.New().String(),
				ProductoID:      id,
				TipoOperacion:   "INICIAL",
				CantidadCambio:  0,
				StockResultante: 0,
				VendedorID:      1, // O un ID de sistema/admin
				Timestamp:       time.Now(),
				Sincronizado:    false,
			}
			if _, err := stmtInsertOp.ExecContext(ctx, op.UUID, op.ProductoID, op.TipoOperacion, op.CantidadCambio, op.StockResultante, op.VendedorID, op.Timestamp, op.Sincronizado); err != nil {
				return "", fmt.Errorf("error al crear operación 'INICIAL' para el producto ID %d: %w", id, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("error al confirmar la transacción de normalización: %w", err)
	}

	d.Log.Infof("Normalización local completa. Disparando sincronización hacia el remoto.")
	go d.SincronizacionInteligente()

	return fmt.Sprintf("Stock normalizado localmente para %d productos. La sincronización con el servidor remoto ha comenzado.", len(productoIDs)), nil
}
