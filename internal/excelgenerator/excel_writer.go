// excelgenerator/excel_writer.go
package excelgenerator

import (
	"fmt"
	"sort"

	"github.com/xuri/excelize/v2"
)

// ProductData es la estructura de datos consolidada para escribir en el Excel.
type ProductData struct {
	// Información del Proveedor
	ProviderNIT  string
	ProviderName string

	// Información del Cliente
	CustomerNIT  string
	CustomerName string

	// Información de la Factura
	InvoiceNumber string
	IssueDate     string
	IssueTime     string
	Currency      string
	TotalInvoice  float64

	// Información del Producto
	ProductCode        string
	ProductDescription string
	Quantity           float64
	UnitPrice          float64
	TotalLineAmount    float64
	TaxLineAmount      float64
	Properties         map[string]string // Propiedades dinámicas
}

// CreateProductExcel crea un archivo Excel a partir de una lista de productos con columnas dinámicas.
func CreateProductExcel(data []ProductData, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Productos"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("no se pudo crear la hoja de cálculo: %w", err)
	}

	// --- Lógica para Cabeceras Dinámicas ---

	// 1. Definir cabeceras estáticas
	staticHeaders := []string{
		"NIT Proveedor", "Razón Social Proveedor",
		"NIT Cliente", "Razón Social Cliente",
		"N° Factura", "Fecha Emisión", "Hora Emisión", "Moneda", "Total Factura",
		"Código Producto", "Descripción Producto", "Cantidad", "Precio Unitario", "Subtotal Línea", "IVA/Impuesto Línea",
	}

	// 2. Recopilar todas las claves de propiedades únicas de todos los productos
	propertyKeysSet := make(map[string]struct{})
	for _, product := range data {
		for key := range product.Properties {
			propertyKeysSet[key] = struct{}{}
		}
	}

	// 3. Crear una lista ordenada de las claves dinámicas para un orden consistente
	dynamicHeaders := make([]string, 0, len(propertyKeysSet))
	for key := range propertyKeysSet {
		dynamicHeaders = append(dynamicHeaders, key)
	}
	sort.Strings(dynamicHeaders) // Ordenar alfabéticamente

	// 4. Combinar cabeceras estáticas y dinámicas
	allHeaders := append(staticHeaders, dynamicHeaders...)

	// Escribir la fila de cabecera en el Excel
	for i, header := range allHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// --- Escribir los datos de los productos ---
	for i, product := range data {
		row := i + 2 // Empezar desde la fila 2

		// Escribir datos estáticos
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), product.ProviderNIT)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), product.ProviderName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), product.CustomerNIT)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), product.CustomerName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), product.InvoiceNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), product.IssueDate)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), product.IssueTime)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), product.Currency)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), product.TotalInvoice)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), product.ProductCode)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), product.ProductDescription)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), product.Quantity)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), product.UnitPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), product.TotalLineAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), product.TaxLineAmount)

		// Escribir datos dinámicos
		for j, headerKey := range dynamicHeaders {
			colIndex := len(staticHeaders) + j + 1
			cell, _ := excelize.CoordinatesToCellName(colIndex, row)

			if value, ok := product.Properties[headerKey]; ok {
				f.SetCellValue(sheetName, cell, value)
			}
		}
	}

	// Aplicar formato a las columnas de números
	style, err := f.NewStyle(&excelize.Style{
		NumFmt: 2, // Formato "0.00"
	})
	if err == nil {
		// I, L, M, N, O son Total Factura, Cantidad, Precio Unitario, Subtotal Línea, IVA Línea
		f.SetColStyle(sheetName, "I:I", style)
		f.SetColStyle(sheetName, "L:O", style)
	}

	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("no se pudo guardar el archivo Excel: %w", err)
	}

	return nil
}
