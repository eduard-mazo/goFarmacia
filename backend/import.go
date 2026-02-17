package backend

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	numWorkers = 4
	batchSize  = 500
)

// ImportLog estructura para llevar un registro del proceso de importación.
type ImportLog struct {
	TotalRows       int
	SuccessfulRows  int
	FailedRows      int
	FailedRowErrors []string
	mu              sync.Mutex
}

// AddProcessingError registra un error durante el mapeo de una fila.
func (l *ImportLog) AddProcessingError(rowNum int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.FailedRows++
	if len(l.FailedRowErrors) < 100 {
		l.FailedRowErrors = append(l.FailedRowErrors, fmt.Sprintf("Línea %d (procesamiento): %v", rowNum, err))
	}
}

// CAMBIO: Nuevas funciones para manejar el resultado de la inserción de lotes.
// AddBatchSuccess registra el éxito de la inserción de un lote completo.
func (l *ImportLog) AddBatchSuccess(count int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.SuccessfulRows += count
}

// AddBatchError registra el fallo de la inserción de un lote completo.
func (l *ImportLog) AddBatchError(count int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.FailedRows += count
	if len(l.FailedRowErrors) < 100 {
		l.FailedRowErrors = append(l.FailedRowErrors, fmt.Sprintf("Lote de %d registros falló: %v", count, err))
	}
}

// String genera un resumen del log de importación.
func (l *ImportLog) String() string {
	var errorSummary strings.Builder
	if len(l.FailedRowErrors) > 0 {
		errorSummary.WriteString("\nDetalle de errores:")
		for _, e := range l.FailedRowErrors {
			errorSummary.WriteString("\n - " + e)
		}
		if len(l.FailedRowErrors) >= 100 {
			errorSummary.WriteString("\n - ... (y más errores)")
		}
	} else {
		errorSummary.WriteString("\nNo se encontraron errores.")
	}

	return fmt.Sprintf(
		"--- Resumen de Importación ---\n"+
			"Líneas totales leídas: %d\n"+
			"Registros insertados/actualizados: %d\n"+
			"Filas con errores (omitidas): %d%s\n"+
			"---------------------------------",
		l.TotalRows, l.SuccessfulRows, l.FailedRows, errorSummary.String(),
	)
}

func (d *Db) SelectFile() (string, error) {
	return runtime.OpenFileDialog(d.ctx, runtime.OpenDialogOptions{
		Title: "Seleccione un archivo CSV",
	})
}

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
		reader.Comma = ','

		headers, err := reader.Read()
		if err != nil {
			errorChan <- fmt.Errorf("error al leer el encabezado del CSV: %w", err)
			return
		}

		headerMap := make(map[string]int)
		for i, h := range headers {
			headerMap[strings.TrimSpace(strings.ToLower(h))] = i
		}

		if err := validateHeaders(modelName, headerMap); err != nil {
			errorChan <- err
			return
		}

		importLog := &ImportLog{}
		jobs := make(chan map[string]string)
		results := make(chan interface{})
		var wg sync.WaitGroup

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go d.csvWorker(i+1, modelName, jobs, results, &wg, importLog)
		}

		var insertWg sync.WaitGroup
		insertWg.Add(1)
		go d.resultCollector(modelName, results, progressChan, &insertWg, importLog)

		lineCount := 1
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			lineCount++
			if err != nil {
				d.Log.Warnf("Error leyendo la línea %d del CSV: %v. Omitiendo.", lineCount, err)
				importLog.AddProcessingError(lineCount, err)
				continue
			}

			rowMap := make(map[string]string)
			for header, index := range headerMap {
				if index < len(row) {
					rowMap[header] = row[index]
				}
			}
			jobs <- rowMap
		}
		close(jobs)
		importLog.TotalRows = lineCount - 1

		wg.Wait()
		close(results)

		insertWg.Wait()

		finalSummary := importLog.String()
		d.Log.Info(finalSummary)
		progressChan <- finalSummary
		errorChan <- nil
	}()

	return progressChan, errorChan
}

func (d *Db) csvWorker(id int, modelName string, jobs <-chan map[string]string, results chan<- interface{}, wg *sync.WaitGroup, log *ImportLog) {
	defer wg.Done()
	for rowMap := range jobs {
		record, err := mapRowToStruct(rowMap, modelName)
		if err != nil {
			log.AddProcessingError(0, err) // El número de línea real no está disponible aquí, pero el error es informativo.
			d.Log.Warnf("Worker %d | Error al mapear fila: %v", id, err)
			continue
		}
		results <- record
	}
}

// resultCollector agrupa los resultados y los inserta en la base de datos en lotes.
func (d *Db) resultCollector(modelName string, results <-chan interface{}, progressChan chan<- string, wg *sync.WaitGroup, log *ImportLog) {
	defer wg.Done()
	var batch []interface{}

	for record := range results {
		batch = append(batch, record)
		// CAMBIO: Ya no se incrementa el éxito aquí. Se hace en insertBatch.
		if len(batch) >= batchSize {
			d.insertBatch(batch, modelName, progressChan, log)
			batch = nil // Limpiar el lote.
		}
	}

	if len(batch) > 0 {
		d.insertBatch(batch, modelName, progressChan, log)
	}
}

// insertBatch realiza la inserción de un lote de datos en la base de datos usando raw SQL.
func (d *Db) insertBatch(batch []interface{}, modelName string, progressChan chan<- string, log *ImportLog) {
	if len(batch) == 0 {
		return
	}

	progressChan <- fmt.Sprintf("Intentando insertar lote de %d registros para '%s'...", len(batch), modelName)

	tx, err := d.DB.Begin()
	if err != nil {
		d.Log.Errorf("Error al iniciar transacción para lote: %v", err)
		log.AddBatchError(len(batch), err)
		return
	}

	switch modelName {
	case "Productos":
		err = d.insertProductoBatch(tx, batch)
	case "Clientes":
		err = d.insertClienteBatch(tx, batch)
	default:
		err = fmt.Errorf("modelo '%s' desconocido para inserción por lotes", modelName)
	}

	if err != nil {
		_ = tx.Rollback()
		d.Log.Errorf("Error al insertar lote para el modelo %s: %v", modelName, err)
		log.AddBatchError(len(batch), err)
		return
	}

	if err := tx.Commit(); err != nil {
		d.Log.Errorf("Error al confirmar lote para el modelo %s: %v", modelName, err)
		log.AddBatchError(len(batch), err)
		return
	}

	log.AddBatchSuccess(len(batch))
}

func (d *Db) insertProductoBatch(tx *sql.Tx, batch []interface{}) error {
	stmt, err := tx.Prepare(`
		INSERT INTO productos (uuid, nombre, codigo, precio_venta, stock, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (codigo) DO UPDATE SET
			nombre = EXCLUDED.nombre,
			precio_venta = EXCLUDED.precio_venta,
			stock = EXCLUDED.stock,
			updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return fmt.Errorf("error preparando statement de productos: %w", err)
	}
	defer stmt.Close()

	for _, item := range batch {
		p := item.(Producto)
		if _, err := stmt.Exec(uuid.New().String(), p.Nombre, p.Codigo, p.PrecioVenta, p.Stock); err != nil {
			return fmt.Errorf("error insertando producto '%s': %w", p.Codigo, err)
		}
	}
	return nil
}

func (d *Db) insertClienteBatch(tx *sql.Tx, batch []interface{}) error {
	stmt, err := tx.Prepare(`
		INSERT INTO clientes (uuid, nombre, apellido, tipo_id, numero_id, telefono, email, direccion, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (numero_id) DO UPDATE SET
			nombre = EXCLUDED.nombre,
			apellido = EXCLUDED.apellido,
			tipo_id = EXCLUDED.tipo_id,
			telefono = EXCLUDED.telefono,
			email = EXCLUDED.email,
			direccion = EXCLUDED.direccion,
			updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return fmt.Errorf("error preparando statement de clientes: %w", err)
	}
	defer stmt.Close()

	for _, item := range batch {
		c := item.(Cliente)
		if _, err := stmt.Exec(uuid.New().String(), c.Nombre, c.Apellido, c.TipoID, c.NumeroID, c.Telefono, c.Email, c.Direccion); err != nil {
			return fmt.Errorf("error insertando cliente '%s': %w", c.NumeroID, err)
		}
	}
	return nil
}

// mapRowToStruct convierte un mapa de (header -> valor) a un struct específico.
func mapRowToStruct(rowMap map[string]string, modelName string) (interface{}, error) {
	switch modelName {
	case "Productos":
		nombre := rowMap["nombre"]
		codigo := rowMap["codigo"]

		precioStr := strings.Replace(rowMap["precio_venta"], ",", ".", -1)
		precio, err := strconv.ParseFloat(precioStr, 64)
		if err != nil {
			return nil, fmt.Errorf("campo 'precio_venta' inválido ('%s') para el código '%s'", rowMap["precio_venta"], codigo)
		}

		stock, err := strconv.Atoi(rowMap["stock"])
		if err != nil {
			return nil, fmt.Errorf("campo 'stock' inválido ('%s') para el código '%s'", rowMap["stock"], codigo)
		}

		if nombre == "" || codigo == "" {
			return nil, fmt.Errorf("los campos 'nombre' y 'codigo' no pueden estar vacíos")
		}

		// CAMBIO: Devolver el struct por valor, no como puntero.
		return Producto{
			Nombre:      nombre,
			Codigo:      codigo,
			PrecioVenta: precio,
			Stock:       stock,
		}, nil

	case "Clientes":
		if rowMap["nombre"] == "" || rowMap["numero_id"] == "" {
			return nil, fmt.Errorf("los campos 'nombre' y 'numeroid' no pueden estar vacíos")
		}

		// CAMBIO: Devolver el struct por valor, no como puntero.
		return Cliente{
			Nombre:    rowMap["nombre"],
			Apellido:  rowMap["apellido"],
			TipoID:    rowMap["tipo_id"],
			NumeroID:  rowMap["numero_id"],
			Telefono:  rowMap["telefono"],
			Email:     rowMap["email"],
			Direccion: rowMap["direccion"],
		}, nil

	default:
		return nil, fmt.Errorf("modelo desconocido: %s", modelName)
	}
}

// (La función validateHeaders no necesita cambios)
func validateHeaders(modelName string, headerMap map[string]int) error {
	var requiredHeaders []string
	switch modelName {
	case "Productos":
		requiredHeaders = []string{"nombre", "codigo", "precio_venta", "stock"}
	case "Clientes":
		requiredHeaders = []string{"nombre", "apellido", "tipo_id", "numero_id", "telefono", "email", "direccion"}
	default:
		return fmt.Errorf("modelo desconocido '%s' para validación de headers", modelName)
	}

	var missingHeaders []string
	for _, h := range requiredHeaders {
		if _, ok := headerMap[h]; !ok {
			missingHeaders = append(missingHeaders, h)
		}
	}

	if len(missingHeaders) > 0 {
		return fmt.Errorf("el archivo CSV no contiene los encabezados requeridos: [%s]", strings.Join(missingHeaders, ", "))
	}
	return nil
}
