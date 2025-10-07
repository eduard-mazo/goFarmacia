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
	if err := d.DeepResetDatabases(); err != nil {
		return "", err
	}
	return "¡Reseteo completado! Todas las bases de datos han sido limpiadas y reiniciadas.", nil
}

func (d *Db) NormalizarStockTodosLosProductos() (string, error) {
	d.Log.Info("Iniciando proceso de normalización de stock para todos los productos.")

	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return "", fmt.Errorf("error al iniciar la transacción de normalización: %w", tx.Error)
	}
	defer tx.Rollback()

	// 1. Obtener los IDs de todos los productos
	var idsProductos []uint
	if err := tx.Model(&Producto{}).Pluck("id", &idsProductos).Error; err != nil {
		return "", fmt.Errorf("error al obtener los IDs de los productos: %w", err)
	}

	if len(idsProductos) == 0 {
		d.Log.Info("No hay productos para normalizar.")
		return "No se encontraron productos para normalizar.", nil
	}

	// 2. Iterar sobre cada ID y recalcular su stock
	for _, id := range idsProductos {
		if err := RecalcularYActualizarStock(tx, id); err != nil {
			// Si falla para un producto, se revierte toda la operación
			return "", fmt.Errorf("falló la normalización para el producto ID %d: %w", id, err)
		}
	}

	// 3. Si todo es exitoso, confirmar la transacción
	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("error al confirmar la transacción de normalización: %w", err)
	}

	d.Log.Infof("Proceso de normalización de stock completado exitosamente para %d productos.", len(idsProductos))
	return fmt.Sprintf("Stock normalizado para %d productos.", len(idsProductos)), nil
}
