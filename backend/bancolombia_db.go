// bancolombia_db.go — Capa de acceso a datos para las transferencias Bancolombia.
//
// ═══════════════════════════════════════════════════════════════════════════════
// TABLA: transferencias_bancolombia
// ═══════════════════════════════════════════════════════════════════════════════
//
//   uuid           UUID PRIMARY KEY        — identificador único interno
//   email_id       TEXT UNIQUE NOT NULL    — Message-ID de Gmail; evita duplicados
//   fecha          TIMESTAMPTZ             — fecha/hora de la transferencia (COT)
//   monto          NUMERIC(15,2)           — valor recibido en COP
//   remitente      TEXT                    — nombre del remitente extraído del correo
//   referencia     TEXT                    — número de referencia Bancolombia (si aplica)
//   cuenta_destino TEXT                    — últimos dígitos de la cuenta destino
//   concepto       TEXT                    — método de pago: "Por llave Bancolombia" |
//                                            "Por código QR" | "Transferencia normal"
//   raw_subject    TEXT                    — oración completa de la notificación
//   leido          BOOLEAN DEFAULT false   — indica si fue revisada en la UI
//   created_at     TIMESTAMPTZ DEFAULT NOW() — cuándo fue importada a la BD
//
// ═══════════════════════════════════════════════════════════════════════════════
// IDEMPOTENCIA
// ═══════════════════════════════════════════════════════════════════════════════
//
//   GuardarTransferencia usa ON CONFLICT (email_id) DO NOTHING, garantizando que
//   un mismo correo nunca se registre dos veces aunque el sync se ejecute varias
//   veces sobre el mismo período.
//
// ═══════════════════════════════════════════════════════════════════════════════
// PAGINACIÓN
// ═══════════════════════════════════════════════════════════════════════════════
//
//   ObtenerTransferenciasPaginado calcula automáticamente el total de registros
//   y el total acumulado en COP para alimentar los KPIs del módulo de tesorería.

package backend

import (
	"fmt"
	"strings"
	"time"
)

// TransferenciaBancolombia represents a single Bancolombia transfer notification.
type TransferenciaBancolombia struct {
	UUID          string    `json:"uuid"`
	EmailID       string    `json:"emailId"`
	Fecha         time.Time `json:"fecha"    ts_type:"string"`
	Monto         float64   `json:"monto"`
	Remitente     string    `json:"remitente"`
	Referencia    string    `json:"referencia"`
	CuentaDestino string    `json:"cuentaDestino"`
	Concepto      string    `json:"concepto"`
	RawSubject    string    `json:"rawSubject"`
	Leido         bool      `json:"leido"`
	CreadoEn      time.Time `json:"creadoEn" ts_type:"string"`
	FacturaUUID   string    `json:"facturaUuid"`
	FacturaNumero string    `json:"facturaNumero"`
}

// FacturaVentaResumen is a summary of a sale invoice for linking to a transfer.
type FacturaVentaResumen struct {
	UUID          string    `json:"uuid"`
	NumeroFactura string    `json:"numeroFactura"`
	ClienteNombre string    `json:"clienteNombre"`
	Total         float64   `json:"total"`
	Fecha         time.Time `json:"fecha" ts_type:"string"`
}

// TransferenciasResponse is the paginated response sent to the frontend.
type TransferenciasResponse struct {
	Items      []TransferenciaBancolombia `json:"items"`
	Total      int                        `json:"total"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"pageSize"`
	TotalMonto float64                    `json:"totalMonto"`
}

// GuardarTransferencia inserts a transfer notification.
// Returns true if it was new, false if it already existed (duplicate email_id).
func (d *Db) GuardarTransferencia(t TransferenciaBancolombia) (bool, error) {
	const q = `
		INSERT INTO transferencias_bancolombia
		       (uuid, email_id, fecha, monto, remitente, referencia,
		        cuenta_destino, concepto, raw_subject)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (email_id) DO NOTHING`

	res, err := d.DB.Exec(q,
		t.UUID, t.EmailID, t.Fecha, t.Monto, t.Remitente, t.Referencia,
		t.CuentaDestino, t.Concepto, t.RawSubject,
	)
	if err != nil {
		return false, fmt.Errorf("guardar transferencia: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

// ExisteTransferencia checks whether an email_id is already stored.
func (d *Db) ExisteTransferencia(emailID string) bool {
	var n int
	_ = d.DB.QueryRow(
		`SELECT COUNT(1) FROM transferencias_bancolombia WHERE email_id = $1`, emailID,
	).Scan(&n)
	return n > 0
}

// MarcarLeida marks a transfer as read by UUID.
func (d *Db) MarcarTransferenciaLeida(uuid string) error {
	_, err := d.DB.Exec(
		`UPDATE transferencias_bancolombia SET leido = true WHERE uuid = $1`, uuid,
	)
	return err
}

// ObtenerTransferenciasPaginado returns a paginated list of transfer notifications.
// busqueda filters by remitente/referencia/concepto (case-insensitive substring).
// estado: "" = todos, "vinculada" = con factura, "sin_vincular" = sin factura.
func (d *Db) ObtenerTransferenciasPaginado(page, pageSize int, soloNoLeidas bool, busqueda, estado string) (TransferenciasResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	// Build dynamic WHERE clause
	var conds []string
	var filterArgs []any
	argIdx := 1

	if soloNoLeidas {
		conds = append(conds, "leido = false")
	}
	if busqueda != "" {
		like := "%" + strings.ToLower(busqueda) + "%"
		conds = append(conds, fmt.Sprintf(
			"(LOWER(remitente) LIKE $%d OR LOWER(referencia) LIKE $%d OR LOWER(concepto) LIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		filterArgs = append(filterArgs, like)
		argIdx++
	}
	switch estado {
	case "vinculada":
		conds = append(conds, "factura_uuid IS NOT NULL")
	case "sin_vincular":
		conds = append(conds, "factura_uuid IS NULL")
	}

	whereClause := ""
	if len(conds) > 0 {
		whereClause = "WHERE " + strings.Join(conds, " AND ")
	}

	// COUNT + SUM using filter args only
	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(1) FROM transferencias_bancolombia %s`, whereClause)
	if err := d.DB.QueryRow(countQ, filterArgs...).Scan(&total); err != nil {
		return TransferenciasResponse{}, fmt.Errorf("count transferencias: %w", err)
	}

	var totalMonto float64
	montoQ := fmt.Sprintf(`SELECT COALESCE(SUM(monto),0) FROM transferencias_bancolombia %s`, whereClause)
	_ = d.DB.QueryRow(montoQ, filterArgs...).Scan(&totalMonto)

	// Add LIMIT/OFFSET args after filter args
	dataArgs := append(filterArgs, pageSize, offset)
	limitIdx := argIdx
	offsetIdx := argIdx + 1

	dataQ := fmt.Sprintf(`
		SELECT uuid, email_id, COALESCE(fecha, NOW()), monto, remitente,
		       referencia, cuenta_destino, concepto, raw_subject, leido, created_at,
		       COALESCE(factura_uuid::text, ''), COALESCE(factura_numero, '')
		FROM   transferencias_bancolombia %s
		ORDER  BY fecha DESC NULLS LAST
		LIMIT  $%d OFFSET $%d`, whereClause, limitIdx, offsetIdx)

	rows, err := d.DB.Query(dataQ, dataArgs...)
	if err != nil {
		return TransferenciasResponse{}, fmt.Errorf("query transferencias: %w", err)
	}
	defer rows.Close()

	var items []TransferenciaBancolombia
	for rows.Next() {
		var t TransferenciaBancolombia
		if err := rows.Scan(
			&t.UUID, &t.EmailID, &t.Fecha, &t.Monto, &t.Remitente,
			&t.Referencia, &t.CuentaDestino, &t.Concepto, &t.RawSubject,
			&t.Leido, &t.CreadoEn, &t.FacturaUUID, &t.FacturaNumero,
		); err != nil {
			return TransferenciasResponse{}, err
		}
		items = append(items, t)
	}
	if items == nil {
		items = []TransferenciaBancolombia{}
	}

	return TransferenciasResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalMonto: totalMonto,
	}, nil
}

// EliminarTransferencia deletes a transfer notification by UUID.
func (d *Db) EliminarTransferencia(uuid string) error {
	_, err := d.DB.Exec(
		`DELETE FROM transferencias_bancolombia WHERE uuid = $1`, uuid,
	)
	return err
}

// MarcarTodasLeidas marks every unread transfer notification as read.
func (d *Db) MarcarTodasLeidas() error {
	_, err := d.DB.Exec(
		`UPDATE transferencias_bancolombia SET leido = true WHERE leido = false`,
	)
	return err
}

// ContarTransferenciasNoLeidas returns the count of unread transfer notifications.
func (d *Db) ContarTransferenciasNoLeidas() int {
	var n int
	_ = d.DB.QueryRow(
		`SELECT COUNT(1) FROM transferencias_bancolombia WHERE leido = false`,
	).Scan(&n)
	return n
}

// EliminarTransferencias bulk-deletes transfer notifications by UUID slice.
func (d *Db) EliminarTransferencias(uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}
	// Build $1,$2,… placeholders
	placeholders := make([]string, len(uuids))
	args := make([]any, len(uuids))
	for i, u := range uuids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = u
	}
	q := fmt.Sprintf(
		`DELETE FROM transferencias_bancolombia WHERE uuid IN (%s)`,
		strings.Join(placeholders, ","),
	)
	_, err := d.DB.Exec(q, args...)
	return err
}

// VincularFactura links a sale invoice to a transfer notification.
func (d *Db) VincularFactura(transferUUID, facturaUUID, facturaNumero string) error {
	_, err := d.DB.Exec(
		`UPDATE transferencias_bancolombia SET factura_uuid = $1, factura_numero = $2 WHERE uuid = $3`,
		facturaUUID, facturaNumero, transferUUID,
	)
	return err
}

// DesvincularFactura removes the invoice link from a transfer notification.
func (d *Db) DesvincularFactura(transferUUID string) error {
	_, err := d.DB.Exec(
		`UPDATE transferencias_bancolombia SET factura_uuid = NULL, factura_numero = '' WHERE uuid = $1`,
		transferUUID,
	)
	return err
}

// BuscarFacturasVenta searches sale invoices by number or client name.
func (d *Db) BuscarFacturasVenta(busqueda string, limit int) ([]FacturaVentaResumen, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	like := "%" + strings.ToLower(busqueda) + "%"
	const q = `
		SELECT f.uuid, f.numero_factura,
		       COALESCE(c.nombre || ' ' || c.apellido, '') AS cliente_nombre,
		       COALESCE(f.total, 0),
		       COALESCE(f.fecha_emision, NOW())
		FROM   facturas f
		LEFT   JOIN clientes c ON f.cliente_id = c.id
		WHERE  f.deleted_at IS NULL
		  AND  (LOWER(f.numero_factura) LIKE $1 OR LOWER(COALESCE(c.nombre,'') || ' ' || COALESCE(c.apellido,'')) LIKE $1)
		ORDER  BY f.fecha_emision DESC
		LIMIT  $2`

	rows, err := d.DB.Query(q, like, limit)
	if err != nil {
		return nil, fmt.Errorf("buscar facturas venta: %w", err)
	}
	defer rows.Close()

	var result []FacturaVentaResumen
	for rows.Next() {
		var r FacturaVentaResumen
		if err := rows.Scan(&r.UUID, &r.NumeroFactura, &r.ClienteNombre, &r.Total, &r.Fecha); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	if result == nil {
		result = []FacturaVentaResumen{}
	}
	return result, nil
}
