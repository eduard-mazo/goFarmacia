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

	queryTotalVentas := "SELECT COALESCE(SUM(total), 0), COUNT(uuid) FROM facturas WHERE fecha_emision BETWEEN $1 AND $2"
	err = d.DB.QueryRow(queryTotalVentas, inicioDelDia, finDelDia).Scan(&data.TotalVentasDia, &data.NumeroVentasDia)
	if err != nil {
		return data, fmt.Errorf("error al obtener total de ventas: %w", err)
	}
	if data.NumeroVentasDia > 0 {
		data.TicketPromedioDia = data.TotalVentasDia / float64(data.NumeroVentasDia)
	}

	queryVentasInd := "SELECT TO_CHAR(fecha_emision, 'YYYY-MM-DD HH24:MI:SS'), total FROM facturas WHERE fecha_emision BETWEEN $1 AND $2 ORDER BY fecha_emision ASC"
	rows, err := d.DB.Query(queryVentasInd, inicioDelDia, finDelDia)
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
		JOIN productos p ON p.uuid = df.producto_uuid
		JOIN facturas f ON f.uuid = df.factura_uuid
		WHERE f.fecha_emision BETWEEN $1 AND $2
		GROUP BY p.nombre
		ORDER BY cantidad DESC
		LIMIT 5`
	rows, err = d.DB.Query(queryTopProd, inicioDelDia, finDelDia)
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
	queryMetodos := "SELECT metodo_pago, COUNT(*) as count FROM facturas WHERE fecha_emision BETWEEN $1 AND $2 GROUP BY metodo_pago"
	rows, err = d.DB.Query(queryMetodos, inicioDelDia, finDelDia)
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
	querySinStock := "SELECT uuid, codigo, nombre, precio_venta, stock FROM productos WHERE stock <= 0 AND deleted_at IS NULL ORDER BY nombre ASC LIMIT 5"
	rows, err = d.DB.Query(querySinStock)
	if err != nil {
		return data, fmt.Errorf("error al obtener productos sin stock: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.UUID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock); err != nil {
			return data, err
		}
		data.ProductosSinStock = append(data.ProductosSinStock, p)
	}

	// 7. Obtener el Top Vendedor del día.
	queryTopVendedor := `
		SELECT v.nombre, SUM(f.total) as total_vendido
		FROM facturas f
		JOIN vendedors v ON v.uuid = f.vendedor_uuid
		WHERE f.fecha_emision BETWEEN $1 AND $2
		GROUP BY v.nombre
		ORDER BY total_vendido DESC
		LIMIT 1`
	err = d.DB.QueryRow(queryTopVendedor, inicioDelDia, finDelDia).Scan(&data.TopVendedor.NombreCompleto, &data.TopVendedor.TotalVendido)
	if err != nil {
		if err == sql.ErrNoRows {
			data.TopVendedor = VendedorRendimiento{NombreCompleto: "N/A", TotalVendido: 0}
		} else {
			return data, fmt.Errorf("error al obtener top vendedor: %w", err)
		}
	}

	return data, nil
}

// ==================== REPORT ENDPOINTS ====================

type ProductoAlerta struct {
	UUID        string  `json:"uuid"`
	Nombre      string  `json:"nombre"`
	Codigo      string  `json:"codigo"`
	Stock       int     `json:"stock"`
	PrecioVenta float64 `json:"precioVenta"`
}

type ResumenInventario struct {
	TotalProductos      int              `json:"totalProductos"`
	ProductosStockBajo  int              `json:"productosStockBajo"`
	ProductosSinStock   int              `json:"productosSinStock"`
	ValorInventario     float64          `json:"valorInventario"`
	ProductosAlerta     []ProductoAlerta `json:"productosAlerta"`
}

type ReporteVentas struct {
	TotalVentas        float64                  `json:"totalVentas"`
	NumeroVentas       int64                    `json:"numeroVentas"`
	TicketPromedio     float64                  `json:"ticketPromedio"`
	VentasIndividuales []VentaIndividual        `json:"ventasIndividuales"`
	TopProductos       []ProductoVendido        `json:"topProductos"`
	TopVendedores      []VendedorRendimiento    `json:"topVendedores"`
	MetodosPago        []map[string]interface{} `json:"metodosPago"`
}

func (d *Db) ObtenerResumenInventario() (ResumenInventario, error) {
	var res ResumenInventario
	res.ProductosAlerta = make([]ProductoAlerta, 0)

	// Total products
	err := d.DB.QueryRow("SELECT COUNT(*) FROM productos WHERE deleted_at IS NULL").Scan(&res.TotalProductos)
	if err != nil {
		return res, fmt.Errorf("error al contar productos: %w", err)
	}

	// Low stock (1-10)
	err = d.DB.QueryRow("SELECT COUNT(*) FROM productos WHERE stock > 0 AND stock <= 10 AND deleted_at IS NULL").Scan(&res.ProductosStockBajo)
	if err != nil {
		return res, fmt.Errorf("error al contar productos stock bajo: %w", err)
	}

	// Out of stock
	err = d.DB.QueryRow("SELECT COUNT(*) FROM productos WHERE stock <= 0 AND deleted_at IS NULL").Scan(&res.ProductosSinStock)
	if err != nil {
		return res, fmt.Errorf("error al contar productos sin stock: %w", err)
	}

	// Inventory value
	err = d.DB.QueryRow("SELECT COALESCE(SUM(precio_venta * stock), 0) FROM productos WHERE deleted_at IS NULL AND stock > 0").Scan(&res.ValorInventario)
	if err != nil {
		return res, fmt.Errorf("error al calcular valor inventario: %w", err)
	}

	// Products needing attention (stock <= 10)
	rows, err := d.DB.Query("SELECT uuid, nombre, codigo, stock, precio_venta FROM productos WHERE stock <= 10 AND deleted_at IS NULL ORDER BY stock ASC LIMIT 20")
	if err != nil {
		return res, fmt.Errorf("error al obtener productos alerta: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p ProductoAlerta
		if err := rows.Scan(&p.UUID, &p.Nombre, &p.Codigo, &p.Stock, &p.PrecioVenta); err != nil {
			return res, err
		}
		res.ProductosAlerta = append(res.ProductosAlerta, p)
	}

	return res, nil
}

func (d *Db) ObtenerReporteVentasRango(fechaInicio, fechaFin string) (ReporteVentas, error) {
	var rep ReporteVentas
	rep.VentasIndividuales = make([]VentaIndividual, 0)
	rep.TopProductos = make([]ProductoVendido, 0)
	rep.TopVendedores = make([]VendedorRendimiento, 0)
	rep.MetodosPago = make([]map[string]interface{}, 0)

	inicio, err := time.Parse("2006-01-02", fechaInicio)
	if err != nil {
		return rep, fmt.Errorf("formato de fecha inicio inválido: %w", err)
	}
	fin, err := time.Parse("2006-01-02", fechaFin)
	if err != nil {
		return rep, fmt.Errorf("formato de fecha fin inválido: %w", err)
	}

	location := inicio.Location()
	desde := time.Date(inicio.Year(), inicio.Month(), inicio.Day(), 0, 0, 0, 0, location)
	hasta := time.Date(fin.Year(), fin.Month(), fin.Day(), 23, 59, 59, 999999999, location)

	// Totals
	err = d.DB.QueryRow("SELECT COALESCE(SUM(total), 0), COUNT(uuid) FROM facturas WHERE fecha_emision BETWEEN $1 AND $2", desde, hasta).Scan(&rep.TotalVentas, &rep.NumeroVentas)
	if err != nil {
		return rep, fmt.Errorf("error al obtener total ventas rango: %w", err)
	}
	if rep.NumeroVentas > 0 {
		rep.TicketPromedio = rep.TotalVentas / float64(rep.NumeroVentas)
	}

	// Individual sales
	rows, err := d.DB.Query("SELECT TO_CHAR(fecha_emision, 'YYYY-MM-DD HH24:MI:SS'), total FROM facturas WHERE fecha_emision BETWEEN $1 AND $2 ORDER BY fecha_emision ASC", desde, hasta)
	if err != nil {
		return rep, fmt.Errorf("error al obtener ventas individuales: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v VentaIndividual
		if err := rows.Scan(&v.Timestamp, &v.Total); err != nil {
			return rep, err
		}
		rep.VentasIndividuales = append(rep.VentasIndividuales, v)
	}

	// Top products
	rows, err = d.DB.Query(`
		SELECT p.nombre, SUM(df.cantidad) as cantidad
		FROM detalle_facturas df
		JOIN productos p ON p.uuid = df.producto_uuid
		JOIN facturas f ON f.uuid = df.factura_uuid
		WHERE f.fecha_emision BETWEEN $1 AND $2
		GROUP BY p.nombre
		ORDER BY cantidad DESC
		LIMIT 10`, desde, hasta)
	if err != nil {
		return rep, fmt.Errorf("error al obtener top productos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p ProductoVendido
		if err := rows.Scan(&p.Nombre, &p.Cantidad); err != nil {
			return rep, err
		}
		rep.TopProductos = append(rep.TopProductos, p)
	}

	// Top vendors
	rows, err = d.DB.Query(`
		SELECT CONCAT(v.nombre, ' ', v.apellido), SUM(f.total) as total_vendido
		FROM facturas f
		JOIN vendedors v ON v.uuid = f.vendedor_uuid
		WHERE f.fecha_emision BETWEEN $1 AND $2
		GROUP BY v.nombre, v.apellido
		ORDER BY total_vendido DESC
		LIMIT 10`, desde, hasta)
	if err != nil {
		return rep, fmt.Errorf("error al obtener top vendedores: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var vr VendedorRendimiento
		if err := rows.Scan(&vr.NombreCompleto, &vr.TotalVendido); err != nil {
			return rep, err
		}
		rep.TopVendedores = append(rep.TopVendedores, vr)
	}

	// Payment methods
	rows, err = d.DB.Query("SELECT metodo_pago, COUNT(*) as count FROM facturas WHERE fecha_emision BETWEEN $1 AND $2 GROUP BY metodo_pago", desde, hasta)
	if err != nil {
		return rep, fmt.Errorf("error al obtener métodos de pago: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var metodo string
		var count int
		if err := rows.Scan(&metodo, &count); err != nil {
			return rep, err
		}
		rep.MetodosPago = append(rep.MetodosPago, map[string]interface{}{"metodo_pago": metodo, "count": count})
	}

	return rep, nil
}

func (d *Db) ObtenerFechasConVentas() ([]string, error) {
	var fechas []string
	query := "SELECT DISTINCT TO_CHAR(fecha_emision, 'YYYY-MM-DD') FROM facturas ORDER BY 1"
	rows, err := d.DB.Query(query)
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
