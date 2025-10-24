package backend

import (
	"database/sql"
	"fmt"
	"time"
)

type VentaIndividual struct {
	Timestamp string  `json:"timestamp"`
	Total     float64 `json:"total"`
}

type ProductoVendido struct {
	Nombre   string `json:"nombre"`
	Cantidad int    `json:"cantidad"`
}

type VendedorRendimiento struct {
	NombreCompleto string  `json:"nombreCompleto"`
	TotalVendido   float64 `json:"totalVendido"`
}

type DashboardData struct {
	TotalVentasDia     float64                  `json:"totalVentasDia"`
	NumeroVentasDia    int64                    `json:"numeroVentasDia"`
	TicketPromedioDia  float64                  `json:"ticketPromedioDia"`
	VentasIndividuales []VentaIndividual        `json:"ventasIndividuales"`
	TopProductos       []ProductoVendido        `json:"topProductos"`
	ProductosSinStock  []Producto               `json:"productosSinStock"`
	TopVendedor        VendedorRendimiento      `json:"topVendedor"`
	MetodosPago        []map[string]interface{} `json:"metodosPago"`
}

func (d *Db) ObtenerDatosDashboard(fechaStr string) (DashboardData, error) {
	var data DashboardData
	var err error

	var fechaSeleccionada time.Time
	if fechaStr == "" {
		fechaSeleccionada = time.Now()
	} else {
		fechaSeleccionada, err = time.Parse("2006-01-02", fechaStr)
		if err != nil {
			return data, fmt.Errorf("formato de fecha inválido: %w", err)
		}
	}

	location := fechaSeleccionada.Location()
	inicioDelDia := time.Date(fechaSeleccionada.Year(), fechaSeleccionada.Month(), fechaSeleccionada.Day(), 0, 0, 0, 0, location)
	finDelDia := inicioDelDia.Add(24*time.Hour - 1*time.Nanosecond)

	// Inicializar slices para evitar `null` en la respuesta JSON.
	data.VentasIndividuales = make([]VentaIndividual, 0)
	data.TopProductos = make([]ProductoVendido, 0)
	data.ProductosSinStock = make([]Producto, 0)
	data.MetodosPago = make([]map[string]interface{}, 0)

	queryTotalVentas := "SELECT COALESCE(SUM(total), 0), COUNT(id) FROM facturas WHERE fecha_emision BETWEEN ? AND ?"
	err = d.LocalDB.QueryRow(queryTotalVentas, inicioDelDia, finDelDia).Scan(&data.TotalVentasDia, &data.NumeroVentasDia)
	if err != nil {
		return data, fmt.Errorf("error al obtener total de ventas: %w", err)
	}
	if data.NumeroVentasDia > 0 {
		data.TicketPromedioDia = data.TotalVentasDia / float64(data.NumeroVentasDia)
	}

	queryVentasInd := "SELECT strftime('%Y-%m-%d %H:%M:%S', fecha_emision), total FROM facturas WHERE fecha_emision BETWEEN ? AND ? ORDER BY fecha_emision ASC"
	rows, err := d.LocalDB.Query(queryVentasInd, inicioDelDia, finDelDia)
	if err != nil {
		return data, fmt.Errorf("error al obtener ventas individuales: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v VentaIndividual
		if err := rows.Scan(&v.Timestamp, &v.Total); err != nil {
			return data, err
		}
		data.VentasIndividuales = append(data.VentasIndividuales, v)
	}

	queryTopProd := `
		SELECT p.nombre, SUM(df.cantidad) as cantidad
		FROM detalle_facturas df
		JOIN productos p ON p.id = df.producto_id
		JOIN facturas f ON f.id = df.factura_id
		WHERE f.fecha_emision BETWEEN ? AND ?
		GROUP BY p.nombre
		ORDER BY cantidad DESC
		LIMIT 5`
	rows, err = d.LocalDB.Query(queryTopProd, inicioDelDia, finDelDia)
	if err != nil {
		return data, fmt.Errorf("error al obtener top productos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p ProductoVendido
		if err := rows.Scan(&p.Nombre, &p.Cantidad); err != nil {
			return data, err
		}
		data.TopProductos = append(data.TopProductos, p)
	}

	// 5. Obtener distribución de Métodos de Pago.
	queryMetodos := "SELECT metodo_pago, COUNT(*) as count FROM facturas WHERE fecha_emision BETWEEN ? AND ? GROUP BY metodo_pago"
	rows, err = d.LocalDB.Query(queryMetodos, inicioDelDia, finDelDia)
	if err != nil {
		return data, fmt.Errorf("error al obtener métodos de pago: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var metodo string
		var count int
		if err := rows.Scan(&metodo, &count); err != nil {
			return data, err
		}
		data.MetodosPago = append(data.MetodosPago, map[string]interface{}{"metodo_pago": metodo, "count": count})
	}

	// 6. Obtener Top 5 Productos sin stock.
	querySinStock := "SELECT id, uuid, codigo, nombre, precio_venta, stock FROM productos WHERE stock <= 0 AND deleted_at IS NULL ORDER BY nombre ASC LIMIT 5"
	rows, err = d.LocalDB.Query(querySinStock)
	if err != nil {
		return data, fmt.Errorf("error al obtener productos sin stock: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.ID, &p.UUID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock); err != nil {
			return data, err
		}
		data.ProductosSinStock = append(data.ProductosSinStock, p)
	}

	// 7. Obtener el Top Vendedor del día.
	queryTopVendedor := `
		SELECT v.nombre, SUM(f.total) as total_vendido
		FROM facturas f
		JOIN vendedors v ON v.id = f.vendedor_id
		WHERE f.fecha_emision BETWEEN ? AND ?
		GROUP BY v.nombre
		ORDER BY total_vendido DESC
		LIMIT 1`
	err = d.LocalDB.QueryRow(queryTopVendedor, inicioDelDia, finDelDia).Scan(&data.TopVendedor.NombreCompleto, &data.TopVendedor.TotalVendido)
	if err != nil {
		if err == sql.ErrNoRows {
			data.TopVendedor = VendedorRendimiento{NombreCompleto: "N/A", TotalVendido: 0}
		} else {
			return data, fmt.Errorf("error al obtener top vendedor: %w", err)
		}
	}

	return data, nil
}

func (d *Db) ObtenerFechasConVentas() ([]string, error) {
	var fechas []string
	query := "SELECT DISTINCT strftime('%Y-%m-%d', fecha_emision) FROM facturas ORDER BY 1"
	rows, err := d.LocalDB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error al obtener fechas con ventas: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fecha string
		if err := rows.Scan(&fecha); err != nil {
			return nil, err
		}
		fechas = append(fechas, fecha)
	}
	return fechas, nil
}
