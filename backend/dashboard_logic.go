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

// --- CAMBIO: La estructura principal ahora usa VentaIndividual ---
type DashboardData struct {
	TotalVentasHoy     float64             `json:"totalVentasHoy"`
	NumeroVentasHoy    int64               `json:"numeroVentasHoy"`
	TicketPromedioHoy  float64             `json:"ticketPromedioHoy"`
	VentasIndividuales []VentaIndividual   `json:"ventasIndividuales"`
	TopProductos       []ProductoVendido   `json:"topProductos"`
	ProductosSinStock  []Producto          `json:"productosSinStock"`
	TopVendedor        VendedorRendimiento `json:"topVendedor"`
}

func (d *Db) ObtenerDatosDashboard() (DashboardData, error) {
	var data DashboardData
	now := time.Now()
	inicioDelDia := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	finDelDia := inicioDelDia.Add(24*time.Hour - 1*time.Nanosecond)

	// Obtener total de ventas y número de transacciones
	var totalVentas struct {
		Total  float64
		Numero int64
	}
	err := d.LocalDB.Model(&Factura{}).
		Select("COALESCE(SUM(total), 0) as total, COUNT(id) as numero").
		Where("fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Scan(&totalVentas).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener total de ventas: %w", err)
	}
	data.TotalVentasHoy = totalVentas.Total
	data.NumeroVentasHoy = totalVentas.Numero
	if data.NumeroVentasHoy > 0 {
		data.TicketPromedioHoy = data.TotalVentasHoy / float64(data.NumeroVentasHoy)
	}

	// --- CAMBIO: Obtener cada venta individual en lugar de agruparlas por hora ---
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

	err = d.LocalDB.Where("stock <= 0").Order("nombre ASC").Limit(5).Find(&data.ProductosSinStock).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener productos sin stock: %w", err)
	}

	// Obtener Top Vendedor del día
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
