//go:build ignore
// +build ignore

// gmail.go — standalone CLI for DIAN invoice processing via Gmail.
// Run with: go run gmail.go <command>
// This file is excluded from the normal Wails build.
// The ERP integration lives in backend/gmail_service.go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"goFarmacia/internal/analyzer"
	"goFarmacia/internal/excelgenerator"
	"goFarmacia/internal/gmailclient"
	"goFarmacia/internal/logger"
	"goFarmacia/internal/processor"

	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
)

const (
	filteredInvoicesCSV  = "facturas_filtradas.csv"
	pendingMetadataCSV   = "metadata_pendientes.csv"
	pendingDownloadsCSV  = "descargas_pendientes.csv"
	xmlOutputDir         = "facturas_xml"
	finalExcelReportFile = "reporte_productos.xlsx"
)

var srv *gmail.Service

var rootCmd = &cobra.Command{
	Use:   "goPharma",
	Short: "Herramienta CLI para procesar facturas desde Gmail.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error
		srv, err = gmailclient.NewService("credentials.json")
		if err != nil {
			logger.Error("Error al inicializar el servicio de Gmail: %v", err)
			os.Exit(1)
		}
		logger.Success("Conexión con Gmail establecida.")
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Fase 1: Analiza correos y filtra facturas.",
	Run: func(cmd *cobra.Command, args []string) {
		_, err := analyzer.AnalyzeAndFilterSubjects(srv, filteredInvoicesCSV, pendingMetadataCSV)
		if err != nil {
			logger.Error("Fallo crítico en fase de análisis: %v", err)
			os.Exit(1)
		}
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Fase 2: Descarga los adjuntos de las facturas.",
	Run: func(cmd *cobra.Command, args []string) {
		processInvoices(filteredInvoicesCSV)
	},
}

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Fase 3: Genera reporte Excel de los XML descargados.",
	Run: func(cmd *cobra.Command, args []string) {
		processXMLsToExcel(filteredInvoicesCSV)
	},
}

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Reintenta operaciones fallidas.",
}

var retryMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "Reintenta obtener metadatos pendientes.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := analyzer.RetryPendingMetadata(srv, pendingMetadataCSV, filteredInvoicesCSV); err != nil {
			logger.Error("Error en reintento de metadatos: %v", err)
		}
	},
}

var retryDownloadsCmd = &cobra.Command{
	Use:   "downloads",
	Short: "Reintenta descargar facturas pendientes.",
	Run: func(cmd *cobra.Command, args []string) {
		processInvoices(pendingDownloadsCSV)
	},
}

func processXMLsToExcel(csvPath string) {
	invoicesToProcess, err := analyzer.ReadFromCSV(csvPath)
	if err != nil {
		logger.Warn("Archivo '%s' no encontrado. Ejecuta 'analyze' y 'download' primero.", csvPath)
		return
	}

	logger.Info("Procesando XML de %d facturas para generar reporte...", len(invoicesToProcess))

	var allProductLines []excelgenerator.ProductData
	var productMu sync.Mutex
	var wg sync.WaitGroup
	var processed int32

	for _, invoice := range invoicesToProcess {
		wg.Add(1)
		go func(inv analyzer.InvoiceData) {
			defer wg.Done()
			xmlPath := filepath.Join(xmlOutputDir, inv.NIT, fmt.Sprintf("%s.xml", inv.InvoiceNumber))

			if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
				return
			}

			products, err := processor.ParseInvoiceXML(xmlPath)
			if err != nil {
				return
			}

			atomic.AddInt32(&processed, 1)
			if atomic.LoadInt32(&processed)%10 == 0 {
				logger.Progress("Analizando XMLs: %d/%d completados...", atomic.LoadInt32(&processed), len(invoicesToProcess))
			}

			productMu.Lock()
			for _, p := range products {
				allProductLines = append(allProductLines, excelgenerator.ProductData{
					ProviderNIT:        p.SupplierNIT,
					ProviderName:       p.SupplierName,
					CustomerNIT:        p.CustomerNIT,
					CustomerName:       p.CustomerName,
					InvoiceNumber:      p.InvoiceID,
					IssueDate:          p.IssueDate,
					IssueTime:          p.IssueTime,
					Currency:           p.Currency,
					TotalInvoice:       p.TotalInvoice,
					ProductCode:        p.Code,
					ProductDescription: p.Description,
					Quantity:           p.Quantity,
					UnitPrice:          p.UnitPrice,
					TotalLineAmount:    p.Total,
					TaxLineAmount:      p.TaxAmount,
					Properties:         p.Properties,
				})
			}
			productMu.Unlock()
		}(invoice)
	}
	wg.Wait()
	fmt.Println()

	if len(allProductLines) == 0 {
		logger.Warn("No se pudo extraer información de los archivos XML.")
		return
	}

	logger.Info("Creando archivo Excel con %d líneas de productos...", len(allProductLines))
	if err := excelgenerator.CreateProductExcel(allProductLines, finalExcelReportFile); err != nil {
		logger.Error("No se pudo crear el archivo Excel: %v", err)
		return
	}

	logger.Success("Reporte Excel generado: '%s'", finalExcelReportFile)
}

func processInvoices(csvPath string) {
	invoicesToProcess, err := analyzer.ReadFromCSV(csvPath)
	if err != nil {
		logger.Warn("Archivo '%s' no encontrado.", csvPath)
		return
	}

	logger.Info("Descargando %d facturas desde Gmail...", len(invoicesToProcess))

	var wg sync.WaitGroup
	var failedDownloads []analyzer.InvoiceData
	var failedMu sync.Mutex
	var downloaded int32

	os.Remove(pendingDownloadsCSV)

	for _, invoice := range invoicesToProcess {
		wg.Add(1)
		go func(inv analyzer.InvoiceData) {
			defer wg.Done()
			if err := processor.ProcessInvoice(srv, inv); err != nil {
				failedMu.Lock()
				failedDownloads = append(failedDownloads, inv)
				failedMu.Unlock()
			} else {
				atomic.AddInt32(&downloaded, 1)
				if atomic.LoadInt32(&downloaded)%10 == 0 {
					logger.Progress("Descargas: %d/%d completadas...", atomic.LoadInt32(&downloaded), len(invoicesToProcess))
				}
			}
		}(invoice)
	}
	wg.Wait()
	fmt.Println()

	if len(failedDownloads) > 0 {
		logger.Warn("%d descargas fallaron. Guardadas en '%s' para reintento.", len(failedDownloads), pendingDownloadsCSV)
		analyzer.AppendToCSV(failedDownloads, pendingDownloadsCSV)
	} else {
		logger.Success("Todas las facturas se descargaron correctamente.")
	}
}

func init() {
	rootCmd.AddCommand(analyzeCmd, downloadCmd, processCmd, retryCmd)
	retryCmd.AddCommand(retryMetadataCmd, retryDownloadsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
