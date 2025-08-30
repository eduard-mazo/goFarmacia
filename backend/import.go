package backend

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm/clause"
)

const (
	numWorkers = 4   // Número de goroutines trabajadoras. Puedes ajustarlo.
	batchSize  = 500 // Tamaño del lote para inserciones en la BD.
)

func (d *Db) SelectFile() (string, error) {
	return runtime.OpenFileDialog(d.ctx, runtime.OpenDialogOptions{
		Title: "Seleccione un archivo",
	})
}

// CargarDesdeCSV procesa un archivo CSV y carga los datos en la tabla especificada.
// Utiliza goroutines para procesar las filas en paralelo.
// Devuelve dos canales: uno para mensajes de progreso y otro para el error final.
func (d *Db) CargarDesdeCSV(filePath string, modelName string) (<-chan string, <-chan error) {
	progressChan := make(chan string)
	errorChan := make(chan error, 1)

	go func() {
		defer close(progressChan)
		defer close(errorChan)

		file, err := os.Open(filePath)
		if err != nil {
			errorChan <- fmt.Errorf("error al abrir el archivo: %w", err)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		// Si tu CSV usa un separador diferente (ej. ';'), configúralo aquí:
		// reader.Comma = ';'

		// Omitir la fila de encabezado
		if _, err := reader.Read(); err != nil {
			errorChan <- fmt.Errorf("error al leer el encabezado del CSV: %w", err)
			return
		}

		jobs := make(chan []string)
		results := make(chan interface{})
		var wg sync.WaitGroup

		// Iniciar trabajadores (workers)
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go d.csvWorker(i+1, modelName, jobs, results, &wg)
		}

		// Iniciar el colector de resultados que insertará en la BD
		var insertWg sync.WaitGroup
		insertWg.Add(1)
		go d.resultCollector(modelName, results, progressChan, &insertWg)

		// Leer el archivo y enviar trabajos a los workers
		lineCount := 0
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				d.Log.Warnf("Error leyendo la línea %d del CSV: %v", lineCount+1, err)
				continue // Salta la línea con error y continúa
			}
			jobs <- row
			lineCount++
		}
		close(jobs) // No hay más trabajos, cerrar el canal

		wg.Wait()      // Esperar a que todos los workers terminen
		close(results) // Cerrar el canal de resultados

		insertWg.Wait() // Esperar a que el colector termine de insertar el último lote
		progressChan <- fmt.Sprintf("Proceso completado. Total de líneas procesadas: %d", lineCount)
		errorChan <- nil
	}()

	return progressChan, errorChan
}

// csvWorker es la goroutine que procesa una fila del CSV.
func (d *Db) csvWorker(id int, modelName string, jobs <-chan []string, results chan<- interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for row := range jobs {
		record, err := mapRowToStruct(row, modelName)
		if err != nil {
			d.Log.Warnf("Worker %d | Error al mapear la fila %v: %v", id, row, err)
			continue
		}
		results <- record
	}
}

// resultCollector agrupa los resultados y los inserta en la base de datos en lotes.
func (d *Db) resultCollector(modelName string, results <-chan interface{}, progressChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	var batch []interface{}
	totalInserted := 0

	for record := range results {
		batch = append(batch, record)
		if len(batch) >= batchSize {
			d.insertBatch(batch, modelName)
			totalInserted += len(batch)
			progressChan <- fmt.Sprintf("Insertados %d registros...", totalInserted)
			batch = nil // Limpiar el lote
		}
	}

	// Insertar el último lote si queda algo
	if len(batch) > 0 {
		d.insertBatch(batch, modelName)
		totalInserted += len(batch)
		progressChan <- fmt.Sprintf("Insertando último lote... Total insertado: %d", totalInserted)
	}
}

// insertBatch realiza la inserción de un lote de datos en la base de datos.
func (d *Db) insertBatch(batch []interface{}, modelName string) {
	if len(batch) == 0 {
		return
	}

	// Usamos la misma lógica OnConflict que ya tenías
	err := d.LocalDB.Clauses(clause.OnConflict{
		Columns:   getUniqueColumns(modelName),
		DoUpdates: clause.AssignmentColumns(getUpdatableColumns(modelName)),
	}).CreateInBatches(batch, batchSize).Error

	if err != nil {
		d.Log.Errorf("Error al insertar lote para el modelo %s: %v", modelName, err)
	}
}

// mapRowToStruct convierte una fila de CSV (slice de strings) a un struct específico.
// ¡Aquí es donde defines el mapeo de columnas para cada tabla!
func mapRowToStruct(row []string, modelName string) (interface{}, error) {
	switch modelName {
	case "Productos":
		if len(row) < 7 {
			return nil, fmt.Errorf("fila incompleta para Producto, se esperan 7 columnas, se obtuvieron %d", len(row))
		}

		precio, err := strconv.ParseFloat(row[5], 64)
		if err != nil {
			precio = 0 // Valor por defecto si la conversión falla
		}
		stock, err := strconv.Atoi(row[6])
		if err != nil {
			stock = 0 // Valor por defecto
		}

		return Producto{
			// Omitimos created_at, updated_at, deleted_at. GORM los maneja.
			Nombre:      row[3],
			Codigo:      row[4],
			PrecioVenta: precio,
			Stock:       stock,
			CreatedAt:   time.Now(), // Asignamos la fecha actual
			UpdatedAt:   time.Now(),
		}, nil

	// case "Clientes":
	// 	// Implementa la lógica para Clientes aquí
	// 	// return Cliente{...}, nil

	default:
		return nil, fmt.Errorf("modelo desconocido: %s", modelName)
	}
}
