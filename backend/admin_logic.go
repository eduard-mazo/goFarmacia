package backend

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
