package backend

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ─── Structs ─────────────────────────────────────────────────────────────────

// ProveedorStats enriquece un Proveedor con datos agregados de sus facturas de compra.
type ProveedorStats struct {
	UUID           string  `json:"uuid"`
	NIT            string  `json:"NIT"`
	Nombre         string  `json:"Nombre"`
	Telefono       string  `json:"Telefono"`
	Email          string  `json:"Email"`
	TotalFacturas  int     `json:"TotalFacturas"`
	TotalComprado  float64 `json:"TotalComprado"`
	UltimaCompra   string  `json:"UltimaCompra"`   // ISO date string, empty if none
	TopProductos   []ProductoComprado `json:"TopProductos"`
}

// ProductoComprado represents a purchased product with aggregated purchase stats.
type ProductoComprado struct {
	Descripcion    string  `json:"Descripcion"`
	CodigoProducto string  `json:"CodigoProducto"`
	TotalCantidad  float64 `json:"TotalCantidad"`
	TotalComprado  float64 `json:"TotalComprado"`
	NumFacturas    int     `json:"NumFacturas"`
}

// ProveedoresStatsResponse is the paginated response for proveedores con estadísticas.
type ProveedoresStatsResponse struct {
	Records      []ProveedorStats `json:"Records"`
	TotalRecords int              `json:"TotalRecords"`
}

// ResumenCompras is a top-level summary of purchase activity.
type ResumenCompras struct {
	TotalGastado    float64            `json:"TotalGastado"`
	NumFacturas     int                `json:"NumFacturas"`
	NumProveedores  int                `json:"NumProveedores"`
	TopProveedores  []ProveedorStats   `json:"TopProveedores"`
	TopProductos    []ProductoComprado `json:"TopProductos"`
}

// ─── Queries ─────────────────────────────────────────────────────────────────

// ObtenerProveedoresConEstadisticas returns paginated suppliers enriched with purchase stats.
func (d *Db) ObtenerProveedoresConEstadisticas(page, pageSize int, busqueda string) (ProveedoresStatsResponse, error) {
	ctx := context.Background()

	args := []any{}
	whereClause := "WHERE p.deleted_at IS NULL"
	argIdx := 1

	if q := strings.TrimSpace(busqueda); q != "" {
		like := "%" + q + "%"
		whereClause += fmt.Sprintf(
			" AND (p.nombre ILIKE $%d OR p.nit ILIKE $%d OR p.email ILIKE $%d)",
			argIdx, argIdx+1, argIdx+2,
		)
		args = append(args, like, like, like)
		argIdx += 3
	}

	var total int
	if err := d.QueryRow(ctx,
		"SELECT COUNT(*) FROM proveedors p "+whereClause, args...,
	).Scan(&total); err != nil {
		return ProveedoresStatsResponse{}, err
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	rows, err := d.Query(ctx, fmt.Sprintf(`
		SELECT
			p.uuid, COALESCE(p.nit,''), p.nombre,
			COALESCE(p.telefono,''), COALESCE(p.email,''),
			COUNT(DISTINCT fc.uuid)            AS total_facturas,
			COALESCE(SUM(fc.total), 0)         AS total_comprado,
			COALESCE(MAX(fc.fecha_emision::text), '') AS ultima_compra
		FROM proveedors p
		LEFT JOIN facturas_compra fc ON fc.proveedor_nit = p.nit AND p.nit != ''
		%s
		GROUP BY p.uuid, p.nit, p.nombre, p.telefono, p.email
		ORDER BY total_comprado DESC, p.nombre ASC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1), args...)
	if err != nil {
		return ProveedoresStatsResponse{}, err
	}
	defer rows.Close()

	var records []ProveedorStats
	for rows.Next() {
		var s ProveedorStats
		if err := rows.Scan(
			&s.UUID, &s.NIT, &s.Nombre, &s.Telefono, &s.Email,
			&s.TotalFacturas, &s.TotalComprado, &s.UltimaCompra,
		); err != nil {
			return ProveedoresStatsResponse{}, err
		}
		s.TopProductos = []ProductoComprado{}
		records = append(records, s)
	}
	if records == nil {
		records = []ProveedorStats{}
	}
	return ProveedoresStatsResponse{Records: records, TotalRecords: total}, nil
}

// ObtenerTopProductosDeProveedor returns the top purchased products for one supplier.
func (d *Db) ObtenerTopProductosDeProveedor(nit string, limit int) ([]ProductoComprado, error) {
	ctx := context.Background()
	rows, err := d.Query(ctx, `
		SELECT
			d.descripcion,
			COALESCE(d.codigo_producto, '') AS codigo,
			SUM(d.cantidad)                 AS total_cantidad,
			SUM(d.total_linea)              AS total_comprado,
			COUNT(DISTINCT fc.uuid)         AS num_facturas
		FROM facturas_compra_detalles d
		JOIN facturas_compra fc ON fc.uuid = d.factura_compra_uuid
		WHERE fc.proveedor_nit = $1
		GROUP BY d.descripcion, codigo
		ORDER BY total_comprado DESC
		LIMIT $2`, nit, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProductoComprado
	for rows.Next() {
		var p ProductoComprado
		if err := rows.Scan(&p.Descripcion, &p.CodigoProducto,
			&p.TotalCantidad, &p.TotalComprado, &p.NumFacturas); err != nil {
			continue
		}
		result = append(result, p)
	}
	if result == nil {
		result = []ProductoComprado{}
	}
	return result, nil
}

// ObtenerResumenCompras returns aggregate purchase stats for the dashboard.
// desde/hasta are YYYY-MM-DD strings (empty = no filter).
func (d *Db) ObtenerResumenCompras(desde, hasta string) (ResumenCompras, error) {
	ctx := context.Background()
	var res ResumenCompras

	whereClause, args := buildComprasWhere(desde, hasta)

	// Totals
	if err := d.QueryRow(ctx,
		"SELECT COALESCE(SUM(total),0), COUNT(*), COUNT(DISTINCT proveedor_nit) FROM facturas_compra "+whereClause,
		args...,
	).Scan(&res.TotalGastado, &res.NumFacturas, &res.NumProveedores); err != nil {
		return res, err
	}

	// Top 5 providers
	rows, err := d.Query(ctx, `
		SELECT proveedor_nit, proveedor_nombre,
		       COUNT(*) AS num_facturas, SUM(total) AS total_comprado
		FROM facturas_compra `+whereClause+`
		GROUP BY proveedor_nit, proveedor_nombre
		ORDER BY total_comprado DESC LIMIT 5`, args...)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var s ProveedorStats
		s.TopProductos = []ProductoComprado{}
		if err := rows.Scan(&s.NIT, &s.Nombre, &s.TotalFacturas, &s.TotalComprado); err != nil {
			continue
		}
		res.TopProveedores = append(res.TopProveedores, s)
	}
	rows.Close()
	if res.TopProveedores == nil {
		res.TopProveedores = []ProveedorStats{}
	}

	// Top 10 products by spend
	prodArgs := append([]any{}, args...)
	prodArgIdx := len(prodArgs) + 1
	prodRows, err := d.Query(ctx, fmt.Sprintf(`
		SELECT
			d.descripcion,
			COALESCE(d.codigo_producto,''),
			SUM(d.cantidad)       AS total_cantidad,
			SUM(d.total_linea)    AS total_comprado,
			COUNT(DISTINCT fc.uuid) AS num_facturas
		FROM facturas_compra_detalles d
		JOIN facturas_compra fc ON fc.uuid = d.factura_compra_uuid
		%s
		GROUP BY d.descripcion, d.codigo_producto
		ORDER BY total_comprado DESC LIMIT $%d`,
		buildComprasWhereJoined(desde, hasta), prodArgIdx,
	), append(prodArgs, 10)...)
	if err != nil {
		return res, err
	}
	defer prodRows.Close()
	for prodRows.Next() {
		var p ProductoComprado
		if err := prodRows.Scan(&p.Descripcion, &p.CodigoProducto,
			&p.TotalCantidad, &p.TotalComprado, &p.NumFacturas); err != nil {
			continue
		}
		res.TopProductos = append(res.TopProductos, p)
	}
	if res.TopProductos == nil {
		res.TopProductos = []ProductoComprado{}
	}

	return res, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildComprasWhere(desde, hasta string) (string, []any) {
	var clauses []string
	var args []any
	idx := 1
	if desde != "" {
		if t, err := time.ParseInLocation("2006-01-02", desde, time.Local); err == nil {
			clauses = append(clauses, fmt.Sprintf("fecha_emision >= $%d", idx))
			args = append(args, t)
			idx++
		}
	}
	if hasta != "" {
		if t, err := time.ParseInLocation("2006-01-02", hasta, time.Local); err == nil {
			clauses = append(clauses, fmt.Sprintf("fecha_emision < $%d", idx))
			args = append(args, t.AddDate(0, 0, 1))
			idx++
		}
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

// buildComprasWhereJoined is the same but prefixed for a JOIN context (fc alias).
func buildComprasWhereJoined(desde, hasta string) string {
	var clauses []string
	idx := 1
	if desde != "" {
		clauses = append(clauses, fmt.Sprintf("fc.fecha_emision >= $%d", idx))
		idx++
	}
	if hasta != "" {
		clauses = append(clauses, fmt.Sprintf("fc.fecha_emision < $%d", idx))
	}
	if len(clauses) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(clauses, " AND ")
}
