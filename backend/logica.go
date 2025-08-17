package backend

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// --- CRUD DE VENDEDORES ---

func (d *Db) RegistrarVendedor(vendedor Vendedor) (string, error) {
	if err := d.DB.Create(&vendedor).Error; err != nil {
		log.Printf("Error al registrar el vendedor: %v", err)
		return "", fmt.Errorf("error al registrar el vendedor: %w", err)
	}
	return fmt.Sprintf("Vendedor '%s' registrado con ID %d", vendedor.Nombre, vendedor.ID), nil
}

func (d *Db) ObtenerVendedores() ([]Vendedor, error) {
	var vendedores []Vendedor
	if err := d.DB.Find(&vendedores).Error; err != nil {
		return nil, err
	}
	return vendedores, nil
}

func (d *Db) ActualizarVendedor(vendedor Vendedor) (string, error) {
	if err := d.DB.Save(&vendedor).Error; err != nil {
		return "", err
	}
	return "Vendedor actualizado correctamente", nil
}

func (d *Db) EliminarVendedor(id uint) (string, error) {
	if err := d.DB.Delete(&Vendedor{}, id).Error; err != nil {
		return "", err
	}
	return "Vendedor eliminado correctamente", nil
}

// --- CRUD DE CLIENTES ---

func (d *Db) RegistrarCliente(cliente Cliente) (string, error) {
	if err := d.DB.Create(&cliente).Error; err != nil {
		log.Printf("Error al registrar el cliente: %v", err)
		return "", fmt.Errorf("error al registrar el cliente: %w", err)
	}
	return fmt.Sprintf("Cliente '%s' registrado con ID %d", cliente.Nombre, cliente.ID), nil
}

func (d *Db) ObtenerClientes() ([]Cliente, error) {
	var clientes []Cliente
	if err := d.DB.Find(&clientes).Error; err != nil {
		return nil, err
	}
	return clientes, nil
}

func (d *Db) ActualizarCliente(cliente Cliente) (string, error) {
	if err := d.DB.Save(&cliente).Error; err != nil {
		return "", err
	}
	return "Cliente actualizado correctamente", nil
}

func (d *Db) EliminarCliente(id uint) (string, error) {
	if err := d.DB.Delete(&Cliente{}, id).Error; err != nil {
		return "", err
	}
	return "Cliente eliminado correctamente", nil
}

// --- CRUD DE PRODUCTOS ---

func (d *Db) RegistrarProducto(producto Producto) (string, error) {
	if err := d.DB.Create(&producto).Error; err != nil {
		log.Printf("Error al registrar el producto: %v", err)
		return "", fmt.Errorf("error al registrar el producto: %w", err)
	}
	return fmt.Sprintf("Producto '%s' registrado con ID %d", producto.Nombre, producto.ID), nil
}

func (d *Db) ObtenerProductos() ([]Producto, error) {
	var productos []Producto
	if err := d.DB.Find(&productos).Error; err != nil {
		return nil, err
	}
	return productos, nil
}

func (d *Db) ActualizarProducto(producto Producto) (string, error) {
	if err := d.DB.Save(&producto).Error; err != nil {
		return "", err
	}
	return "Producto actualizado correctamente", nil
}

func (d *Db) EliminarProducto(id uint) (string, error) {
	if err := d.DB.Delete(&Producto{}, id).Error; err != nil {
		return "", err
	}
	return "Producto eliminado correctamente", nil
}

func (d *Db) BuscarProductos(query string) ([]Producto, error) {
	var productos []Producto
	result := d.DB.Where("nombre LIKE ? OR codigo LIKE ?", "%"+query+"%", "%"+query+"%").Find(&productos)
	if result.Error != nil {
		return nil, result.Error
	}
	return productos, nil
}

// --- LÓGICA DE VENTAS ---

func (d *Db) RegistrarVenta(req VentaRequest) (string, error) {
	tx := d.DB.Begin()
	if tx.Error != nil {
		return "", fmt.Errorf("error al iniciar la transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	var vendedor Vendedor
	if err := tx.First(&vendedor, req.VendedorID).Error; err != nil {
		return "", errors.New("vendedor no encontrado")
	}

	var cliente Cliente
	if err := tx.First(&cliente, req.ClienteID).Error; err != nil {
		return "", errors.New("cliente no encontrado")
	}

	factura := Factura{
		FechaEmision: time.Now(),
		VendedorID:   vendedor.ID,
		ClienteID:    cliente.ID,
		Estado:       "Pagada",
		MetodoPago:   req.MetodoPago,
	}

	var subtotal, iva float64
	var detallesFactura []DetalleFactura

	for _, p := range req.Productos {
		var producto Producto
		if err := tx.First(&producto, p.ID).Error; err != nil {
			return "", fmt.Errorf("producto con ID %d no encontrado", p.ID)
		}

		if producto.Stock < p.Cantidad {
			return "", fmt.Errorf("stock insuficiente para %s. Disponible: %d, Solicitado: %d", producto.Nombre, producto.Stock, p.Cantidad)
		}

		precioTotalProducto := producto.PrecioVenta * float64(p.Cantidad)
		subtotal += precioTotalProducto

		detallesFactura = append(detallesFactura, DetalleFactura{
			ProductoID:     producto.ID,
			Cantidad:       p.Cantidad,
			PrecioUnitario: producto.PrecioVenta,
			PrecioTotal:    precioTotalProducto,
		})

		if err := tx.Model(&producto).Update("Stock", gorm.Expr("Stock - ?", p.Cantidad)).Error; err != nil {
			return "", fmt.Errorf("error al actualizar el stock de %s: %w", producto.Nombre, err)
		}
	}

	// Calcular IVA sobre el subtotal (ej. 19% en Colombia)
	iva = subtotal * 0.19

	factura.Subtotal = subtotal
	factura.IVA = iva
	factura.Total = subtotal + iva
	factura.NumeroFactura = d.generarNumeroFactura()

	if err := tx.Create(&factura).Error; err != nil {
		return "", fmt.Errorf("error al crear la factura: %w", err)
	}

	for i := range detallesFactura {
		detallesFactura[i].FacturaID = factura.ID
	}
	if err := tx.Create(&detallesFactura).Error; err != nil {
		return "", fmt.Errorf("error al crear los detalles de la factura: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("error al confirmar la transacción: %w", err)
	}

	return fmt.Sprintf("Factura %s registrada con éxito", factura.NumeroFactura), nil
}

func (d *Db) generarNumeroFactura() string {
	var ultimaFactura Factura
	result := d.DB.Order("id DESC").Limit(1).Find(&ultimaFactura)

	nuevoNumero := 1
	if result.Error == nil && ultimaFactura.NumeroFactura != "" {
		fmt.Sscanf(ultimaFactura.NumeroFactura, "FAC-%d", &nuevoNumero)
		nuevoNumero++
	}
	return fmt.Sprintf("FAC-%d", nuevoNumero)
}

// --- CONSULTAS DE FACTURAS ---

func (d *Db) ObtenerFacturas() ([]Factura, error) {
	var facturas []Factura
	// Usamos Preload para cargar también los datos del cliente y vendedor asociados
	if err := d.DB.Preload("Cliente").Preload("Vendedor").Order("id desc").Find(&facturas).Error; err != nil {
		return nil, err
	}
	return facturas, nil
}

func (d *Db) ObtenerDetalleFactura(facturaID uint) (Factura, error) {
	var factura Factura
	// Preload carga las relaciones para obtener todos los datos en una sola consulta
	err := d.DB.Preload("Cliente").Preload("Vendedor").Preload("Detalles.Producto").First(&factura, facturaID).Error
	if err != nil {
		return Factura{}, err
	}
	return factura, nil
}
