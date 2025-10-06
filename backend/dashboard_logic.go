package backend

import (
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

	data.VentasIndividuales = make([]VentaIndividual, 0)
	data.TopProductos = make([]ProductoVendido, 0)
	data.ProductosSinStock = make([]Producto, 0)
	data.MetodosPago = make([]map[string]interface{}, 0)

	var totalVentas struct {
		Total  float64
		Numero int64
	}
	err = d.LocalDB.Model(&Factura{}).
		Select("COALESCE(SUM(total), 0) as total, COUNT(id) as numero").
		Where("fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Scan(&totalVentas).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener total de ventas: %w", err)
	}
	data.TotalVentasDia = totalVentas.Total
	data.NumeroVentasDia = totalVentas.Numero
	if data.NumeroVentasDia > 0 {
		data.TicketPromedioDia = data.TotalVentasDia / float64(data.NumeroVentasDia)
	}

	err = d.LocalDB.Model(&Factura{}).
		Select("fecha_emision as timestamp, total").
		Where("fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Order("fecha_emision ASC").
		Scan(&data.VentasIndividuales).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener ventas individuales: %w", err)
	}

	err = d.LocalDB.Model(&DetalleFactura{}).
		Select("p.nombre, SUM(detalle_facturas.cantidad) as cantidad").
		Joins("JOIN productos p ON p.id = detalle_facturas.producto_id").
		Joins("JOIN facturas f ON f.id = detalle_facturas.factura_id").
		Where("f.fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Group("p.nombre").
		Order("cantidad DESC").
		Limit(5).
		Scan(&data.TopProductos).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener top productos: %w", err)
	}

	err = d.LocalDB.Model(&Factura{}).
		Select("metodo_pago, COUNT(*) as count").
		Where("fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Group("metodo_pago").
		Scan(&data.MetodosPago).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener métodos de pago: %w", err)
	}

	err = d.LocalDB.Where("stock <= 0").Order("nombre ASC").Limit(5).Find(&data.ProductosSinStock).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener productos sin stock: %w", err)
	}

	err = d.LocalDB.Model(&Factura{}).
		Select("v.nombre || ' ' || v.apellido as nombre_completo, SUM(facturas.total) as total_vendido").
		Joins("JOIN vendedors v ON v.id = facturas.vendedor_id").
		Where("facturas.fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Group("nombre_completo").
		Order("total_vendido DESC").
		Limit(1).
		Scan(&data.TopVendedor).Error
	if err != nil || data.TopVendedor.NombreCompleto == "" {
		data.TopVendedor = VendedorRendimiento{
			NombreCompleto: "N/A",
			TotalVendido:   0,
		}
	}

	return data, nil
}

func (d *Db) ObtenerFechasConVentas() ([]string, error) {
	var fechas []string
	err := d.LocalDB.Model(&Factura{}).
		Distinct().
		Pluck("strftime('%Y-%m-%d', fecha_emision)", &fechas).
		Error
	if err != nil {
		return nil, fmt.Errorf("error al obtener fechas con ventas: %w", err)
	}
	return fechas, nil
}
