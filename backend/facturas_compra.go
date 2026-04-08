package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FacturaCompra represents an electronic purchase invoice received via Gmail (DIAN UBL 2.1).
type FacturaCompra struct {
	UUID            string                 `json:"UUID"`
	ProveedorUUID   *string                `json:"ProveedorUUID"`
	ProveedorNIT    string                 `json:"ProveedorNIT"`
	ProveedorNombre string                 `json:"ProveedorNombre"`
	ClienteNIT      string                 `json:"ClienteNIT"`
	ClienteNombre   string                 `json:"ClienteNombre"`
	NumeroFactura   string                 `json:"NumeroFactura"`
	CUFE            string                 `json:"CUFE"`
	FechaEmision    time.Time              `json:"FechaEmision" ts_type:"string"`
	Moneda          string                 `json:"Moneda"`
	Subtotal        float64                `json:"Subtotal"`
	IVA             float64                `json:"IVA"`
	Total           float64                `json:"Total"`
	Estado              string                 `json:"Estado"` // PENDIENTE | PROCESADA | IGNORADA
	TipoDocumento       string                 `json:"TipoDocumento"`   // 01=Factura 91=NotaCrédito 92=NotaDébito
	ReferenciaDocumento string                 `json:"ReferenciaDocumento"` // Para notas: número factura origen
	EmailMessageID      string                 `json:"EmailMessageID"`
	Detalles        []FacturaCompraDetalle `json:"Detalles"`
}

// FacturaCompraDetalle represents one product line from a DIAN electronic invoice.
type FacturaCompraDetalle struct {
	UUID              string            `json:"UUID"`
	FacturaCompraUUID string            `json:"FacturaCompraUUID"`
	CodigoProducto    string            `json:"CodigoProducto"`
	Descripcion       string            `json:"Descripcion"`
	Cantidad          float64           `json:"Cantidad"`
	PrecioUnitario    float64           `json:"PrecioUnitario"`
	TotalLinea        float64           `json:"TotalLinea"`
	ImpuestoLinea     float64           `json:"ImpuestoLinea"`
	Propiedades       map[string]string `json:"Propiedades"`
}

// FacturasCompraResponse is the paginated response returned to the frontend.
type FacturasCompraResponse struct {
	Records      []FacturaCompra `json:"Records"`
	TotalRecords int             `json:"TotalRecords"`
}

// ExisteFacturaCompra returns true if an invoice already exists.
// Deduplication priority:
//  1. CUFE (DIAN cryptographic hash — unique per document) when present.
//  2. email_message_id when CUFE is absent (prevents double-import of undocumented mails).
//
// Using email_message_id alone would cause false-positives when a single ZIP
// contains multiple XML documents (e.g. a credit note + the original invoice):
// the first document saves with that messageID, making the second appear as a duplicate.
func (d *Db) ExisteFacturaCompra(emailMessageID, cufe string) (bool, error) {
	ctx := context.Background()
	var count int
	err := d.QueryRow(ctx,
		`SELECT COUNT(*) FROM facturas_compra
		 WHERE (cufe IS NOT NULL AND cufe != '' AND cufe = $2)
		    OR (email_message_id = $1 AND (cufe IS NULL OR cufe = ''))`,
		emailMessageID, cufe,
	).Scan(&count)
	return count > 0, err
}

// GuardarFacturaCompra inserts a purchase invoice and all its line items in one transaction.
func (d *Db) GuardarFacturaCompra(f FacturaCompra) error {
	ctx := context.Background()
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	cufe := f.CUFE
	if cufe == "" {
		cufe = f.UUID // use UUID as fallback to satisfy unique constraint
	}

	tipoDoc := f.TipoDocumento
	if tipoDoc == "" {
		tipoDoc = "01"
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO facturas_compra
			(uuid, proveedor_nit, proveedor_nombre, cliente_nit, cliente_nombre,
			 numero_factura, cufe, fecha_emision, moneda, subtotal, iva, total,
			 estado, tipo_documento, referencia_documento, email_message_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		f.UUID, f.ProveedorNIT, f.ProveedorNombre, f.ClienteNIT, f.ClienteNombre,
		f.NumeroFactura, cufe, f.FechaEmision, f.Moneda,
		f.Subtotal, f.IVA, f.Total, f.Estado,
		tipoDoc, f.ReferenciaDocumento, f.EmailMessageID,
	)
	if err != nil {
		return fmt.Errorf("error insertando factura_compra: %w", err)
	}

	for _, det := range f.Detalles {
		props, _ := json.Marshal(det.Propiedades)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO facturas_compra_detalles
				(uuid, factura_compra_uuid, codigo_producto, descripcion,
				 cantidad, precio_unitario, total_linea, impuesto_linea, propiedades)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			det.UUID, f.UUID, det.CodigoProducto, det.Descripcion,
			det.Cantidad, det.PrecioUnitario, det.TotalLinea, det.ImpuestoLinea,
			string(props),
		)
		if err != nil {
			return fmt.Errorf("error insertando detalle '%s': %w", det.Descripcion, err)
		}
	}

	return tx.Commit()
}

// validFacturaCompraSort maps frontend column IDs to safe SQL column names.
var validFacturaCompraSort = map[string]string{
	"NumeroFactura":   "numero_factura",
	"FechaEmision":    "fecha_emision",
	"ProveedorNombre": "proveedor_nombre",
	"Total":           "total",
	"Estado":          "estado",
}

// ObtenerFacturasCompraPaginado returns a paginated list of purchase invoices.
func (d *Db) ObtenerFacturasCompraPaginado(page, pageSize int, busqueda, sortField, sortDir string) (FacturasCompraResponse, error) {
	ctx := context.Background()
	offset := (page - 1) * pageSize

	args := []interface{}{}
	whereClause := ""
	argIdx := 1

	if q := strings.TrimSpace(busqueda); q != "" {
		like := "%" + q + "%"
		whereClause = fmt.Sprintf(
			`WHERE (proveedor_nombre ILIKE $%d OR proveedor_nit ILIKE $%d OR numero_factura ILIKE $%d)`,
			argIdx, argIdx+1, argIdx+2,
		)
		args = append(args, like, like, like)
		argIdx += 3
	}

	// Build ORDER BY — validate to prevent SQL injection
	col, ok := validFacturaCompraSort[sortField]
	if !ok {
		col = "fecha_emision"
	}
	dir := "DESC"
	if strings.ToLower(sortDir) == "asc" {
		dir = "ASC"
	}
	orderClause := fmt.Sprintf("ORDER BY %s %s", col, dir)

	var total int
	if err := d.QueryRow(ctx, "SELECT COUNT(*) FROM facturas_compra "+whereClause, args...).Scan(&total); err != nil {
		return FacturasCompraResponse{}, err
	}

	args = append(args, pageSize, offset)
	query := fmt.Sprintf(`
		SELECT uuid, proveedor_nit, proveedor_nombre, cliente_nit, cliente_nombre,
		       numero_factura, COALESCE(cufe,''), fecha_emision, moneda,
		       subtotal, iva, total, estado,
		       COALESCE(tipo_documento,'01'), COALESCE(referencia_documento,''),
		       COALESCE(email_message_id,'')
		FROM facturas_compra %s
		%s
		LIMIT $%d OFFSET $%d`, whereClause, orderClause, argIdx, argIdx+1)

	rows, err := d.Query(ctx, query, args...)
	if err != nil {
		return FacturasCompraResponse{}, err
	}
	defer rows.Close()

	var records []FacturaCompra
	for rows.Next() {
		var f FacturaCompra
		if err := rows.Scan(
			&f.UUID, &f.ProveedorNIT, &f.ProveedorNombre,
			&f.ClienteNIT, &f.ClienteNombre, &f.NumeroFactura, &f.CUFE,
			&f.FechaEmision, &f.Moneda, &f.Subtotal, &f.IVA, &f.Total,
			&f.Estado, &f.TipoDocumento, &f.ReferenciaDocumento,
			&f.EmailMessageID,
		); err != nil {
			return FacturasCompraResponse{}, err
		}
		records = append(records, f)
	}
	if records == nil {
		records = []FacturaCompra{}
	}
	return FacturasCompraResponse{Records: records, TotalRecords: total}, nil
}

// ObtenerDetalleFacturaCompra returns one purchase invoice with all its line items.
func (d *Db) ObtenerDetalleFacturaCompra(uuid string) (FacturaCompra, error) {
	ctx := context.Background()
	var f FacturaCompra
	err := d.QueryRow(ctx, `
		SELECT uuid, proveedor_nit, proveedor_nombre, cliente_nit, cliente_nombre,
		       numero_factura, COALESCE(cufe,''), fecha_emision, moneda,
		       subtotal, iva, total, estado,
		       COALESCE(tipo_documento,'01'), COALESCE(referencia_documento,''),
		       COALESCE(email_message_id,'')
		FROM facturas_compra WHERE uuid = $1`, uuid,
	).Scan(
		&f.UUID, &f.ProveedorNIT, &f.ProveedorNombre,
		&f.ClienteNIT, &f.ClienteNombre, &f.NumeroFactura, &f.CUFE,
		&f.FechaEmision, &f.Moneda, &f.Subtotal, &f.IVA, &f.Total,
		&f.Estado, &f.TipoDocumento, &f.ReferenciaDocumento,
		&f.EmailMessageID,
	)
	if err != nil {
		return FacturaCompra{}, err
	}

	rows, err := d.Query(ctx, `
		SELECT uuid, COALESCE(codigo_producto,''), descripcion,
		       cantidad, precio_unitario, total_linea, impuesto_linea,
		       COALESCE(propiedades::text,'{}')
		FROM facturas_compra_detalles
		WHERE factura_compra_uuid = $1
		ORDER BY created_at`, uuid)
	if err != nil {
		return f, err
	}
	defer rows.Close()

	for rows.Next() {
		var det FacturaCompraDetalle
		var propsJSON string
		det.FacturaCompraUUID = uuid
		if err := rows.Scan(
			&det.UUID, &det.CodigoProducto, &det.Descripcion,
			&det.Cantidad, &det.PrecioUnitario, &det.TotalLinea,
			&det.ImpuestoLinea, &propsJSON,
		); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(propsJSON), &det.Propiedades)
		f.Detalles = append(f.Detalles, det)
	}
	if f.Detalles == nil {
		f.Detalles = []FacturaCompraDetalle{}
	}
	return f, nil
}

// ActualizarEstadoFacturaCompra updates the estado field of a purchase invoice.
func (d *Db) ActualizarEstadoFacturaCompra(uuid, estado string) error {
	_, err := d.Exec(context.Background(),
		`UPDATE facturas_compra SET estado = $1, updated_at = NOW() WHERE uuid = $2`,
		estado, uuid,
	)
	return err
}
