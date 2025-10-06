package backend

import (
	"fmt"
	"log"
	"strings"
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

func (d *Db) NormalizarStockProductos() (string, error) {
	var productos []Producto
	if err := d.LocalDB.Find(&productos).Error; err != nil {
		return "", fmt.Errorf("error al obtener la lista de productos: %w", err)
	}

	var errores []string
	log.Printf("Iniciando normalización de stock para %d productos...", len(productos))

	for _, p := range productos {
		tx := d.LocalDB.Begin()
		if tx.Error != nil {
			log.Printf("Error al iniciar transacción para producto ID %d: %v", p.ID, tx.Error)
			errores = append(errores, fmt.Sprintf("Producto ID %d: %v", p.ID, tx.Error))
			continue
		}

		if err := RecalcularYActualizarStock(tx, p.ID); err != nil {
			log.Printf("Error al recalcular stock para producto ID %d: %v", p.ID, err)
			errores = append(errores, fmt.Sprintf("Producto ID %d: %v", p.ID, err))
			tx.Rollback()
			continue
		}

		if err := tx.Commit().Error; err != nil {
			log.Printf("Error al confirmar transacción para producto ID %d: %v", p.ID, err)
			errores = append(errores, fmt.Sprintf("Producto ID %d: %v", p.ID, err))
		}
	}

	if len(errores) > 0 {
		return "", fmt.Errorf("la normalización finalizó con %d errores: %s", len(errores), strings.Join(errores, "; "))
	}

	log.Printf("Normalización de stock completada exitosamente.")
	return fmt.Sprintf("Stock de %d productos normalizado correctamente.", len(productos)), nil
}
