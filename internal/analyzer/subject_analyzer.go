// internal/analyzer/subject_analyzer.go
package analyzer

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sync"
	"sync/atomic"

	"goFarmacia/internal/gmailclient"
	"goFarmacia/internal/logger"

	"google.golang.org/api/gmail/v1"
)

// writeMessageIDsToCSV escribe una lista de IDs de mensaje a un archivo CSV.
func writeMessageIDsToCSV(messageIDs []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo de pendientes '%s': %w", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"MessageID"}); err != nil {
		return err
	}

	for _, id := range messageIDs {
		if err := writer.Write([]string{id}); err != nil {
			logger.Warn("No se pudo escribir el MessageID %s al archivo de pendientes", id)
		}
	}
	return nil
}

// AnalyzeAndFilterSubjects escanea todos los correos, los filtra por asunto y guarda los resultados.
func AnalyzeAndFilterSubjects(srv *gmail.Service, csvFilename string, pendingCsvFilename string) ([]InvoiceData, error) {
	logger.Info("Obteniendo la lista completa de mensajes de Gmail...")
	var allMessages []*gmail.Message
	pageToken := ""
	for {
		resp, err := gmailclient.ListMessagesWithPagination(srv, "", pageToken)
		if err != nil {
			return nil, fmt.Errorf("no se pudo obtener la lista de mensajes: %w", err)
		}
		allMessages = append(allMessages, resp.Messages...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
		logger.Progress("Mensajes encontrados: %d...", len(allMessages))
	}
	fmt.Println() // Nueva línea tras el progreso

	if len(allMessages) == 0 {
		logger.Warn("No se encontraron mensajes en la cuenta.")
		return nil, nil
	}

	logger.Info("Analizando %d mensajes para identificar facturas...", len(allMessages))

	// Procesa los IDs de mensajes obtenidos.
	filteredInvoices, failedIDs, err := processMessageIDs(srv, allMessages)
	if err != nil {
		return nil, err
	}

	// Guarda los IDs que fallaron durante la obtención de metadatos.
	if len(failedIDs) > 0 {
		logger.Warn("%d mensajes fallaron por cuota de API. Guardando para reintento en '%s'.", len(failedIDs), pendingCsvFilename)
		if err := writeMessageIDsToCSV(failedIDs, pendingCsvFilename); err != nil {
			logger.Error("No se pudo guardar el archivo de metadatos pendientes: %v", err)
		}
	}

	// Guarda las facturas procesadas con éxito.
	if len(filteredInvoices) > 0 {
		if err := WriteToCSV(filteredInvoices, csvFilename); err != nil {
			return nil, fmt.Errorf("no se pudo escribir el archivo CSV principal '%s': %w", csvFilename, err)
		}
		logger.Success("¡Proceso completado! %d facturas guardadas en '%s'", len(filteredInvoices), csvFilename)
	} else {
		logger.Warn("No se encontraron facturas que coincidan con los patrones buscados.")
	}

	return filteredInvoices, nil
}

// RetryPendingMetadata procesa los message IDs desde el archivo de pendientes.
func RetryPendingMetadata(srv *gmail.Service, pendingCsvPath string, outputCsvPath string) error {
	pendingIDs, err := ReadMessageIDsFromCSV(pendingCsvPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("No hay metadatos pendientes en '%s'.", pendingCsvPath)
			return nil
		}
		return fmt.Errorf("error al leer el archivo de pendientes: %w", err)
	}

	if len(pendingIDs) == 0 {
		logger.Info("No hay metadatos pendientes para procesar.")
		return nil
	}

	logger.Info("Reintentando obtención de metadatos para %d mensajes...", len(pendingIDs))

	var messagesToProcess []*gmail.Message
	for _, id := range pendingIDs {
		messagesToProcess = append(messagesToProcess, &gmail.Message{Id: id})
	}

	successful, failed, err := processMessageIDs(srv, messagesToProcess)
	if err != nil {
		return err
	}

	if len(failed) > 0 {
		logger.Warn("%d mensajes volvieron a fallar. Actualizando '%s'.", len(failed), pendingCsvPath)
		writeMessageIDsToCSV(failed, pendingCsvPath)
	} else {
		logger.Success("Todos los metadatos pendientes se procesaron correctamente.")
		os.Remove(pendingCsvPath)
	}

	if len(successful) > 0 {
		logger.Info("Añadiendo %d facturas recuperadas a '%s'.", len(successful), outputCsvPath)
		AppendToCSV(successful, outputCsvPath)
	}

	return nil
}

// processMessageIDs es la lógica central para obtener y parsear metadatos de forma concurrente.
func processMessageIDs(srv *gmail.Service, messages []*gmail.Message) (successful []InvoiceData, failed []string, err error) {
	re := regexp.MustCompile(`^([^;]+);([^;]+);([^;]+);([^;]+);([^;]+)$`)

	var (
		successfulInvoicesMu sync.Mutex
		failedMessagesMu     sync.Mutex
		wg                   sync.WaitGroup
		messagesChan         = make(chan *gmail.Message, 100)
		processedCount       int32
		totalCount           = int32(len(messages))
	)

	const maxConcurrency = 10

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range messagesChan {
				atomic.AddInt32(&processedCount, 1)
				if atomic.LoadInt32(&processedCount)%50 == 0 {
					logger.Progress("Progreso: %d/%d mensajes procesados...", atomic.LoadInt32(&processedCount), totalCount)
				}

				meta, err := gmailclient.GetMessageMetadata(srv, msg.Id)
				if err != nil {
					// Silent failure for logs to avoid noise, but track for retry
					failedMessagesMu.Lock()
					failed = append(failed, msg.Id)
					failedMessagesMu.Unlock()
					continue
				}

				for _, h := range meta.Payload.Headers {
					if h.Name == "Subject" {
						matches := re.FindStringSubmatch(h.Value)
						if len(matches) == 6 {
							invoice := InvoiceData{
								MessageID:      meta.Id,
								NIT:            matches[1],
								CompanyName:    matches[2],
								InvoiceNumber:  matches[3],
								Consecutive:    matches[4],
								CommercialName: matches[5],
							}
							successfulInvoicesMu.Lock()
							successful = append(successful, invoice)
							successfulInvoicesMu.Unlock()
						}
						break
					}
				}
			}
		}()
	}

	for _, msg := range messages {
		messagesChan <- msg
	}
	close(messagesChan)

	wg.Wait()
	fmt.Println() // Limpiar línea de progreso
	return successful, failed, nil
}
