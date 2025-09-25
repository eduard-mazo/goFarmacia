package backend

import (
	"fmt"
	"time"
)

// --- ESTRUCTURAS PARA EL DASHBOARD ---

// VentaHora representa el total de ventas en una hora específica del día.
type VentaHora struct {
	Hora  int     `json:"hora"`
	Total float64 `json:"total"`
}

// ProductoVendido representa un producto y la cantidad total vendida.
type ProductoVendido struct {
	Nombre   string `json:"nombre"`
	Cantidad int    `json:"cantidad"`
}

// VendedorRendimiento representa el total de ventas de un vendedor.
type VendedorRendimiento struct {
	NombreCompleto string  `json:"nombreCompleto"`
	TotalVendido   float64 `json:"totalVendido"`
}

// DashboardData agrupa toda la información necesaria para el dashboard.
type DashboardData struct {
	TotalVentasHoy    float64             `json:"totalVentasHoy"`
	NumeroVentasHoy   int64               `json:"numeroVentasHoy"`
	TicketPromedioHoy float64             `json:"ticketPromedioHoy"`
	VentasPorHora     []VentaHora         `json:"ventasPorHora"`
	TopProductos      []ProductoVendido   `json:"topProductos"`
	ProductosSinStock []Producto          `json:"productosSinStock"`
	TopVendedor       VendedorRendimiento `json:"topVendedor"`
}

// ObtenerDatosDashboard es la función principal que recolecta todas las métricas.
func (d *Db) ObtenerDatosDashboard() (DashboardData, error) {
	var data DashboardData

	// 1. Definir el rango de tiempo (hoy)
	now := time.Now()
	inicioDelDia := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	finDelDia := inicioDelDia.Add(24*time.Hour - 1*time.Nanosecond)

	// 2. Obtener total de ventas y número de transacciones
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

	// 3. Obtener ventas por hora para el gráfico
	var ventasHoraRaw []struct {
		Hora  int
		Total float64
	}
	err = d.LocalDB.Model(&Factura{}).
		Select("CAST(strftime('%H', fecha_emision) AS INTEGER) as hora, SUM(total) as total").
		Where("fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Group("hora").
		Order("hora").
		Scan(&ventasHoraRaw).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener ventas por hora: %w", err)
	}
	// Rellenar las horas sin ventas con 0 para un gráfico completo
	ventasMap := make(map[int]float64)
	for _, vh := range ventasHoraRaw {
		ventasMap[vh.Hora] = vh.Total
	}
	for i := 0; i < 24; i++ {
		total := 0.0
		if val, ok := ventasMap[i]; ok {
			total = val
		}
		data.VentasPorHora = append(data.VentasPorHora, VentaHora{Hora: i, Total: total})
	}

	// 4. Obtener Top 5 productos más vendidos del día
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

	// 5. Obtener Top 5 productos sin stock
	err = d.LocalDB.Where("stock <= 0").Order("nombre ASC").Limit(10).Find(&data.ProductosSinStock).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener productos sin stock: %w", err)
	}

	// 6. Obtener Top Vendedor del día
	err = d.LocalDB.Model(&Factura{}).
		Select("v.nombre || ' ' || v.apellido as nombre_completo, SUM(facturas.total) as total_vendido").
		Joins("JOIN vendedors v ON v.id = facturas.vendedor_id").
		Where("facturas.fecha_emision BETWEEN ? AND ?", inicioDelDia, finDelDia).
		Group("nombre_completo").
		Order("total_vendido DESC").
		Limit(1).
		Scan(&data.TopVendedor).Error
	if err != nil {
		return data, fmt.Errorf("error al obtener top vendedor: %w", err)
	}

	return data, nil
}
