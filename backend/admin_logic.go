package backend

import (
	"fmt"
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

	// Usamos el contexto principal de la aplicación para que la operación se pueda cancelar.
	ctx := d.ctx

	// 1. Iniciar una transacción para asegurar la atomicidad de la operación.
	// Todo se confirma o nada lo hace.
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("error al iniciar la transacción de normalización: %w", err)
	}
	// Defer Rollback se encargará de revertir la transacción si algo sale mal.
	defer tx.Rollback()

	// 2. Calcular el stock real para cada producto desde la fuente de la verdad (operacion_stocks).
	// Esta consulta suma todos los movimientos de stock por producto en una sola pasada.
	rows, err := tx.QueryContext(ctx, `
		SELECT producto_id, SUM(cantidad_cambio)
		FROM operacion_stocks
		GROUP BY producto_id
	`)
	if err != nil {
		return "", fmt.Errorf("error al calcular el stock desde operacion_stocks: %w", err)
	}
	defer rows.Close()

	// Creamos un mapa para guardar los stocks calculados [productID -> stock]
	stocksCalculados := make(map[uint]int)
	for rows.Next() {
		var productoID uint
		var stockCalculado int
		if err := rows.Scan(&productoID, &stockCalculado); err != nil {
			return "", fmt.Errorf("error al escanear el resultado del cálculo de stock: %w", err)
		}
		stocksCalculados[productoID] = stockCalculado
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error durante la iteración de los resultados de stock: %w", err)
	}

	if len(stocksCalculados) == 0 {
		d.Log.Info("No hay operaciones de stock para procesar. La normalización no es necesaria.")
		// Opcionalmente, podrías querer poner a 0 el stock de todos los productos aquí.
		// _, err := tx.ExecContext(ctx, "UPDATE productos SET stock = 0")
		// if err != nil { ... }
		return "No se encontraron operaciones de stock para normalizar.", nil
	}

	// 3. Actualizar la tabla de productos en lote.
	// Primero, reseteamos el stock de TODOS los productos a 0. Esto asegura que los
	// productos sin movimientos en operacion_stocks queden con stock 0, manteniendo la consistencia.
	if _, err := tx.ExecContext(ctx, "UPDATE productos SET stock = 0"); err != nil {
		return "", fmt.Errorf("error al resetear el stock de los productos: %w", err)
	}

	// Preparamos una única sentencia de actualización para reutilizarla.
	// Esto es mucho más eficiente que construir una nueva consulta en cada iteración.
	stmt, err := tx.PrepareContext(ctx, "UPDATE productos SET stock = ? WHERE id = ?")
	if err != nil {
		return "", fmt.Errorf("error al preparar la sentencia de actualización de stock: %w", err)
	}
	defer stmt.Close()

	// Iteramos sobre el mapa de stocks calculados y ejecutamos la actualización para cada producto.
	for productoID, stock := range stocksCalculados {
		if _, err := stmt.ExecContext(ctx, stock, productoID); err != nil {
			// Si falla para un producto, se revierte toda la operación gracias al defer tx.Rollback().
			return "", fmt.Errorf("falló la actualización de stock para el producto ID %d: %w", productoID, err)
		}
	}

	// 4. Si todo es exitoso, confirmar la transacción para hacer los cambios permanentes.
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("error al confirmar la transacción de normalización: %w", err)
	}

	d.Log.Infof("Proceso de normalización de stock completado exitosamente para %d productos.", len(stocksCalculados))
	return fmt.Sprintf("Stock normalizado para %d productos.", len(stocksCalculados)), nil
}
