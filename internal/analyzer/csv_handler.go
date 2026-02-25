// analyzer/csv_handler.go
package analyzer

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
)

// InvoiceData representa una única factura filtrada.
type InvoiceData struct {
	MessageID      string
	NIT            string
	CompanyName    string
	InvoiceNumber  string
	Consecutive    string
	CommercialName string
}

// WriteToCSV escribe un slice de InvoiceData a un archivo CSV, sobreescribiendo el archivo.
func WriteToCSV(invoices []InvoiceData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo '%s': %w", filename, err)
	}
	defer file.Close()

	return writeData(file, invoices, true)
}

// AppendToCSV añade un slice de InvoiceData a un archivo CSV existente.
func AppendToCSV(invoices []InvoiceData, filename string) error {
	// Comprueba si el archivo existe para saber si hay que escribir la cabecera.
	_, err := os.Stat(filename)
	writeHeader := os.IsNotExist(err)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo abrir o crear el archivo '%s' para añadir datos: %w", filename, err)
	}
	defer file.Close()

	return writeData(file, invoices, writeHeader)
}

// writeData es una función auxiliar para escribir los datos en el archivo.
func writeData(file *os.File, invoices []InvoiceData, writeHeader bool) error {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	if writeHeader {
		headers := []string{"MessageID", "NIT", "CompanyName", "InvoiceNumber", "Consecutive", "CommercialName"}
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	for _, invoice := range invoices {
		row := []string{
			invoice.MessageID,
			invoice.NIT,
			invoice.CompanyName,
			invoice.InvoiceNumber,
			invoice.Consecutive,
			invoice.CommercialName,
		}
		if err := writer.Write(row); err != nil {
			// Loguear y continuar podría ser una opción, pero fallar es más seguro.
			return fmt.Errorf("error al escribir la fila para la factura %s: %w", invoice.InvoiceNumber, err)
		}
	}
	return writer.Error() // Retorna cualquier error acumulado por el writer.
}

// ReadFromCSV lee un archivo CSV de facturas y devuelve un slice de InvoiceData.
func ReadFromCSV(filename string) ([]InvoiceData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err // El error es manejado por el llamador (os.IsNotExist).
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 { // Si el archivo está vacío o solo tiene cabecera.
		return []InvoiceData{}, nil
	}

	var invoices []InvoiceData
	// Empieza desde el índice 1 para saltar la fila de la cabecera.
	for i, record := range records[1:] {
		if len(record) < 6 {
			log.Printf("Advertencia: Se encontró una fila (línea %d) con menos de 6 columnas en '%s', será ignorada.", i+2, filename)
			continue
		}
		invoice := InvoiceData{
			MessageID:      record[0],
			NIT:            record[1],
			CompanyName:    record[2],
			InvoiceNumber:  record[3],
			Consecutive:    record[4],
			CommercialName: record[5],
		}
		invoices = append(invoices, invoice)
	}

	return invoices, nil
}

// ReadMessageIDsFromCSV lee un CSV que solo contiene IDs de mensajes.
func ReadMessageIDsFromCSV(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error al leer los registros del CSV '%s': %w", filename, err)
	}

	if len(records) < 2 {
		return []string{}, nil
	}

	var ids []string
	for _, record := range records[1:] { // Saltar cabecera
		if len(record) > 0 && record[0] != "" {
			ids = append(ids, record[0])
		}
	}
	return ids, nil
}
