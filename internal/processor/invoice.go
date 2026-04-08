// internal/processor/invoice.go
// NOTE: DIAN UBL 2.1 document UUIDs use different scheme names per document type:
//   Invoices (01/02) → schemeName="CUFE-SHA384"
//   Credit notes (91) → schemeName="CUDE-SHA384"
//   Debit notes (92)  → schemeName="CUDE-SHA384"
// Credit/debit notes also embed the ORIGINAL invoice's CUFE inside BillingReference.
// String-search extraction would pick that up first — use the struct UUID field instead.
package processor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"goFarmacia/internal/analyzer"
	"goFarmacia/internal/gmailclient"

	"google.golang.org/api/gmail/v1"
)

const outputDir = "facturas_xml"

// ProcessInvoice handles the download and extraction of a single invoice attachment.
func ProcessInvoice(srv *gmail.Service, invoiceData analyzer.InvoiceData) error {
	// 1. Get the full message to find attachment details.
	msg, err := gmailclient.GetMessageFull(srv, invoiceData.MessageID)
	if err != nil {
		return fmt.Errorf("al obtener el correo completo para factura %s: %w", invoiceData.InvoiceNumber, err)
	}

	// 2. Find the .zip attachment within the message.
	attachmentId, _, err := FindZipAttachment(msg)
	if err != nil {
		return fmt.Errorf("al buscar adjunto para factura %s: %w", invoiceData.InvoiceNumber, err)
	}

	// 3. Download the attachment data.
	zipData, err := gmailclient.GetAttachment(srv, invoiceData.MessageID, attachmentId)
	if err != nil {
		return fmt.Errorf("al descargar adjunto para factura %s: %w", invoiceData.InvoiceNumber, err)
	}

	// 4. Unzip the data and save the XML file.
	if err := UnzipAndSaveXML(zipData, &invoiceData); err != nil {
		return fmt.Errorf("al guardar XML de factura %s: %w", invoiceData.InvoiceNumber, err)
	}

	return nil
}

// FindZipAttachment searches for a .zip file attachment in a Gmail message.
func FindZipAttachment(msg *gmail.Message) (id, filename string, err error) {
	if msg.Payload != nil {
		for _, part := range msg.Payload.Parts {
			if strings.HasSuffix(strings.ToLower(part.Filename), ".zip") && part.Body != nil && part.Body.AttachmentId != "" {
				return part.Body.AttachmentId, part.Filename, nil
			}
		}
	}
	return "", "", fmt.Errorf("no se encontró un archivo .zip adjunto")
}

// UnzipAndSaveXML extracts an XML file from zip data and saves it to a structured directory.
func UnzipAndSaveXML(zipData []byte, data *analyzer.InvoiceData) error {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("no se pudo leer el archivo zip: %w", err)
	}

	nitDir := filepath.Join(outputDir, data.NIT)
	if err := os.MkdirAll(nitDir, 0755); err != nil {
		return fmt.Errorf("no se pudo crear el directorio para el NIT %s: %w", data.NIT, err)
	}

	found := false
	for _, f := range zipReader.File {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext == ".xml" {
			zipFile, err := f.Open()
			if err != nil {
				return fmt.Errorf("no se pudo abrir el archivo '%s' desde el zip: %w", f.Name, err)
			}
			defer zipFile.Close()

			outputPath := filepath.Join(nitDir, fmt.Sprintf("%s%s", data.InvoiceNumber, ext))
			outputFile, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("no se pudo crear el archivo de salida '%s': %w", outputPath, err)
			}
			defer outputFile.Close()

			if _, err := io.Copy(outputFile, zipFile); err != nil {
				return fmt.Errorf("no se pudo escribir en el archivo de salida '%s': %w", outputPath, err)
			}
			found = true
		}
	}

	if !found {
		return fmt.Errorf("no se encontró un archivo .xml en el zip para la factura %s", data.InvoiceNumber)
	}

	return nil
}
