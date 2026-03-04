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
func (d *Db) ObtenerTransferenciasPaginado(page, pageSize int, soloNoLeidas bool) (TransferenciasResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	whereClause := ""
	args := []any{pageSize, offset}
	if soloNoLeidas {
		whereClause = "WHERE leido = false"
		args = []any{pageSize, offset}
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(1) FROM transferencias_bancolombia %s`, whereClause)
	if err := d.DB.QueryRow(countQ).Scan(&total); err != nil {
		return TransferenciasResponse{}, fmt.Errorf("count transferencias: %w", err)
	}

	var totalMonto float64
	montoQ := fmt.Sprintf(`SELECT COALESCE(SUM(monto),0) FROM transferencias_bancolombia %s`, whereClause)
	_ = d.DB.QueryRow(montoQ).Scan(&totalMonto)

	dataQ := fmt.Sprintf(`
		SELECT uuid, email_id, COALESCE(fecha, NOW()), monto, remitente,
		       referencia, cuenta_destino, concepto, raw_subject, leido, created_at
		FROM   transferencias_bancolombia %s
		ORDER  BY fecha DESC NULLS LAST
		LIMIT  $1 OFFSET $2`, whereClause)

	rows, err := d.DB.Query(dataQ, args...)
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
			&t.Leido, &t.CreadoEn,
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
