package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"goFarmacia/internal/processor"

	"google.golang.org/api/gmail/v1"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// EnriquecerResult holds the outcome of a PDF-based description enrichment job.
type EnriquecerResult struct {
	Procesadas   int `json:"Procesadas"`   // invoices attempted
	Enriquecidas int `json:"Enriquecidas"` // line items whose description was updated
	Errores      int `json:"Errores"`      // invoices where enrichment failed
}

type invoiceEnrichJob struct {
	UUID           string
	EmailMessageID string
	NumeroFactura  string
	ProveedorNIT   string
}

type detalleEnrichData struct {
	UUID       string
	Code       string
	DescActual string
	PropActual map[string]string
}

// ─── DB helpers ───────────────────────────────────────────────────────────────

// obtenerFacturasConLote returns invoices that have at least one line item
// with a LOTE description and a stored Gmail message ID (needed for PDF download).
func (d *Db) obtenerFacturasConLote() ([]invoiceEnrichJob, error) {
	ctx := context.Background()
	rows, err := d.Query(ctx, `
		SELECT DISTINCT
			fc.uuid,
			COALESCE(fc.email_message_id, ''),
			fc.numero_factura,
			COALESCE(fc.proveedor_nit, '')
		FROM facturas_compra fc
		INNER JOIN facturas_compra_detalles fcd
			ON fcd.factura_compra_uuid = fc.uuid
		WHERE fcd.descripcion ILIKE 'LOTE:%'
		  AND fc.email_message_id IS NOT NULL
		  AND fc.email_message_id != ''
		ORDER BY fc.uuid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []invoiceEnrichJob
	for rows.Next() {
		var j invoiceEnrichJob
		if err := rows.Scan(&j.UUID, &j.EmailMessageID, &j.NumeroFactura, &j.ProveedorNIT); err != nil {
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// obtenerDetallesConLote returns line items for one invoice that still have
// a LOTE description and a product code (needed for PDF matching).
func (d *Db) obtenerDetallesConLote(facturaUUID string) ([]detalleEnrichData, error) {
	ctx := context.Background()
	rows, err := d.Query(ctx, `
		SELECT uuid,
		       COALESCE(codigo_producto,''),
		       descripcion,
		       COALESCE(propiedades::text, '{}')
		FROM facturas_compra_detalles
		WHERE factura_compra_uuid = $1
		  AND descripcion ILIKE 'LOTE:%'`, facturaUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []detalleEnrichData
	for rows.Next() {
		var det detalleEnrichData
		var propsJSON string
		if err := rows.Scan(&det.UUID, &det.Code, &det.DescActual, &propsJSON); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(propsJSON), &det.PropActual)
		if det.PropActual == nil {
			det.PropActual = make(map[string]string)
		}
		items = append(items, det)
	}
	return items, nil
}

// actualizarDescripcionDetalle overwrites the description and properties of one
// line item with enriched data from the PDF.
func (d *Db) actualizarDescripcionDetalle(uuid, descripcion string, props map[string]string) error {
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = d.DB.ExecContext(ctx, `
		UPDATE facturas_compra_detalles
		   SET descripcion = $1,
		       propiedades = $2::jsonb
		 WHERE uuid = $3`,
		descripcion, string(propsJSON), uuid,
	)
	return err
}

// ─── Enrichment engine ────────────────────────────────────────────────────────

const enrichWorkers = 8

// EnriquecerDescripciones finds all line items whose description is still a
// LOTE reference, re-downloads the companion PDF from Gmail, and replaces the
// description with the real product name extracted from the PDF.
//
// Uses a pool of enrichWorkers goroutines for parallel Gmail downloads.
// Progress is broadcast via the same "gmail:sync:log" Wails event.
func (g *GmailService) EnriquecerDescripciones() (EnriquecerResult, error) {
	emit := func(nivel, msg string) {
		g.emitSyncLog(nivel, msg)
	}

	jobs, err := g.db.obtenerFacturasConLote()
	if err != nil {
		return EnriquecerResult{}, fmt.Errorf("buscar facturas con LOTE: %w", err)
	}
	if len(jobs) == 0 {
		emit("ok", "No hay facturas con descripciones pendientes de enriquecimiento")
		return EnriquecerResult{}, nil
	}
	emit("info", fmt.Sprintf("Enriqueciendo %d facturas con datos del PDF...", len(jobs)))

	svc, err := g.newGmailSvc()
	if err != nil {
		return EnriquecerResult{}, fmt.Errorf("autenticación Gmail: %w", err)
	}

	jobCh := make(chan invoiceEnrichJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	var (
		procesadas   int64
		enriquecidas int64
		errores      int64
		wg           sync.WaitGroup
	)

	for range enrichWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				n, err := g.enrichInvoice(svc, job, emit)
				atomic.AddInt64(&procesadas, 1)
				if err != nil {
					atomic.AddInt64(&errores, 1)
					emit("warn", fmt.Sprintf("  ✗ %s (%s): %v", job.NumeroFactura, job.ProveedorNIT, err))
				} else if n > 0 {
					atomic.AddInt64(&enriquecidas, int64(n))
					emit("ok", fmt.Sprintf("  ✓ %s — %d descripciones actualizadas", job.NumeroFactura, n))
				}
			}
		}()
	}

	wg.Wait()

	result := EnriquecerResult{
		Procesadas:   int(procesadas),
		Enriquecidas: int(enriquecidas),
		Errores:      int(errores),
	}
	emit("ok", fmt.Sprintf(
		"Enriquecimiento completado: %d facturas, %d descripciones actualizadas, %d errores",
		result.Procesadas, result.Enriquecidas, result.Errores,
	))
	return result, nil
}

// enrichInvoice downloads the PDF for a single invoice, parses it, and updates
// all LOTE-description line items with the real product names. Returns the
// number of line items updated.
func (g *GmailService) enrichInvoice(
	svc *gmail.Service,
	job invoiceEnrichJob,
	emit func(nivel, msg string),
) (int, error) {
	detalles, err := g.db.obtenerDetallesConLote(job.UUID)
	if err != nil || len(detalles) == 0 {
		return 0, nil
	}

	// Collect known product codes to guide the PDF matcher
	codes := make([]string, 0, len(detalles))
	for _, d := range detalles {
		if d.Code != "" {
			codes = append(codes, d.Code)
		}
	}

	pdfBytes, err := g.fetchPDFForInvoice(svc, job)
	if err != nil || len(pdfBytes) == 0 {
		return 0, fmt.Errorf("no se encontró PDF: %w", err)
	}

	pdfData, err := processor.ParsePDFBytes(pdfBytes, codes)
	if err != nil || len(pdfData.CodeToName) == 0 {
		return 0, fmt.Errorf("PDF sin datos de producto reconocibles")
	}

	updated := 0
	for _, det := range detalles {
		codeUp := strings.ToUpper(det.Code)
		name, found := pdfData.CodeToName[codeUp]
		if !found || name == "" {
			continue
		}
		props := det.PropActual
		if props == nil {
			props = make(map[string]string)
		}
		if _, exists := props["descripcion_xml"]; !exists {
			props["descripcion_xml"] = det.DescActual
		}
		lote, vence := processor.ParseLoteInfo(det.DescActual)
		if lote != "" {
			props["lote"] = lote
		}
		if vence != "" {
			props["vencimiento"] = vence
		}
		props["fuente_descripcion"] = "pdf"

		if err := g.db.actualizarDescripcionDetalle(det.UUID, name, props); err != nil {
			emit("warn", fmt.Sprintf("    error en detalle %s: %v", det.UUID, err))
			continue
		}
		updated++
	}
	return updated, nil
}

// fetchPDFForInvoice tries to obtain PDF bytes for an invoice:
//  1. PDF embedded inside the ZIP attachment of the Gmail message
//  2. Direct PDF attachment in the Gmail message
//  3. On-disk cache at facturas_xml/{NIT}/{NumeroFactura}.pdf
func (g *GmailService) fetchPDFForInvoice(
	svc *gmail.Service,
	job invoiceEnrichJob,
) ([]byte, error) {
	msg, err := svc.Users.Messages.Get("me", job.EmailMessageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	// Pass 1 — look for PDF inside the ZIP
	for _, part := range msg.Payload.Parts {
		if !strings.HasSuffix(strings.ToLower(part.Filename), ".zip") || part.Body == nil {
			continue
		}
		zipData, err := g.downloadAttachmentPart(svc, msg.Id, part)
		if err != nil || len(zipData) == 0 {
			continue
		}
		res, err := processor.UnzipInMemoryAll(zipData)
		if err == nil && len(res.PDFFiles) > 0 {
			return res.PDFFiles[0], nil
		}
	}

	// Pass 2 — look for a direct PDF attachment
	for _, part := range msg.Payload.Parts {
		if !strings.HasSuffix(strings.ToLower(part.Filename), ".pdf") || part.Body == nil {
			continue
		}
		pdfData, err := g.downloadAttachmentPart(svc, msg.Id, part)
		if err == nil && len(pdfData) > 0 {
			return pdfData, nil
		}
	}

	// Pass 3 — on-disk cache
	if job.ProveedorNIT != "" && job.NumeroFactura != "" {
		diskPath := fmt.Sprintf("facturas_xml/%s/%s.pdf", job.ProveedorNIT, job.NumeroFactura)
		if data, err := os.ReadFile(diskPath); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	return nil, fmt.Errorf("no se encontró archivo PDF en ZIP ni como adjunto")
}
