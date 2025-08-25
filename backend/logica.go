package backend

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// --- LÓGICA DE AUTENTICACIÓN ---

func (d *Db) LoginVendedor(req LoginRequest) (Vendedor, error) {
	var vendedor Vendedor
	result := d.DB.Where("cedula = ?", req.Cedula).First(&vendedor)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return Vendedor{}, errors.New("vendedor no encontrado")
		}
		return Vendedor{}, result.Error
	}

	if vendedor.Contrasena != req.Contrasena {
		return Vendedor{}, errors.New("contraseña incorrecta")
	}

	return vendedor, nil
}

// --- LÓGICA PAGINADA Y BÚSQUEDA ---

func (d *Db) ObtenerVendedoresPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var vendedores []Vendedor
	var total int64

	query := d.DB.Model(&Vendedor{})

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("Nombre LIKE ? OR Apellido LIKE ? OR Cedula LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	query.Count(&total) // Contar el total de registros que coinciden con la búsqueda

	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&vendedores).Error
	if err != nil {
		return PaginatedResult{}, err
	}

	return PaginatedResult{Records: vendedores, TotalRecords: total}, nil
}

func (d *Db) ObtenerClientesPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var clientes []Cliente
	var total int64

	query := d.DB.Model(&Cliente{})

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("Nombre LIKE ? OR Apellido LIKE ? OR NumeroID LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&clientes).Error
	if err != nil {
		return PaginatedResult{}, err
	}

	return PaginatedResult{Records: clientes, TotalRecords: total}, nil
}

func (d *Db) ObtenerProductosPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var productos []Producto
	var total int64

	query := d.DB.Model(&Producto{})

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("Nombre LIKE ? OR Codigo LIKE ?", searchTerm, searchTerm)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&productos).Error
	if err != nil {
		return PaginatedResult{}, err
	}

	return PaginatedResult{Records: productos, TotalRecords: total}, nil
}

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

// --- LÓGICA DE BÚSQUEDA ---

func (d *Db) BuscarProductos(query string) ([]Producto, error) {
	var productos []Producto
	result := d.DB.Where("nombre LIKE ? OR codigo LIKE ?", "%"+query+"%", "%"+query+"%").Limit(20).Find(&productos)
	if result.Error != nil {
		return nil, result.Error
	}
	return productos, nil
}

func (d *Db) BuscarProductoPorCodigo(codigo string) (Producto, error) {
	var producto Producto
	result := d.DB.Where("codigo = ?", codigo).First(&producto)
	if result.Error != nil {
		return Producto{}, result.Error // Puede ser gorm.ErrRecordNotFound
	}
	return producto, nil
}

// --- CRUD PROVEEDORES ---

func (d *Db) RegistrarProveedor(proveedor Proveedor) (string, error) {
	if err := d.DB.Create(&proveedor).Error; err != nil {
		return "", fmt.Errorf("error al registrar el proveedor: %w", err)
	}
	return "Proveedor registrado correctamente", nil
}

func (d *Db) ObtenerProveedores() ([]Proveedor, error) {
	var proveedores []Proveedor
	if err := d.DB.Find(&proveedores).Error; err != nil {
		return nil, err
	}
	return proveedores, nil
}

// --- LÓGICA DE VENTAS ---

func (d *Db) RegistrarVenta(req VentaRequest) (Factura, error) {
	tx := d.DB.Begin()
	if tx.Error != nil {
		return Factura{}, fmt.Errorf("error al iniciar la transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	var vendedor Vendedor
	if err := tx.First(&vendedor, req.VendedorID).Error; err != nil {
		return Factura{}, errors.New("vendedor no encontrado")
	}

	var cliente Cliente
	if err := tx.First(&cliente, req.ClienteID).Error; err != nil {
		return Factura{}, errors.New("cliente no encontrado")
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
			return Factura{}, fmt.Errorf("producto con ID %d no encontrado", p.ID)
		}

		if producto.Stock < p.Cantidad {
			return Factura{}, fmt.Errorf("stock insuficiente para %s. Disponible: %d, Solicitado: %d", producto.Nombre, producto.Stock, p.Cantidad)
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
			return Factura{}, fmt.Errorf("error al actualizar el stock de %s: %w", producto.Nombre, err)
		}
	}

	iva = subtotal * 0.19

	factura.Subtotal = subtotal
	factura.IVA = iva
	factura.Total = subtotal + iva
	factura.NumeroFactura = d.generarNumeroFactura()

	if err := tx.Create(&factura).Error; err != nil {
		return Factura{}, fmt.Errorf("error al crear la factura: %w", err)
	}

	for i := range detallesFactura {
		detallesFactura[i].FacturaID = factura.ID
	}
	if err := tx.Create(&detallesFactura).Error; err != nil {
		return Factura{}, fmt.Errorf("error al crear los detalles de la factura: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return Factura{}, fmt.Errorf("error al confirmar la transacción: %w", err)
	}

	// Devolver la factura completa para el visor
	return d.ObtenerDetalleFactura(factura.ID)
}

func (d *Db) generarNumeroFactura() string {
	var ultimaFactura Factura
	result := d.DB.Order("id DESC").Limit(1).Find(&ultimaFactura)

	nuevoNumero := 1000
	if result.Error == nil && ultimaFactura.NumeroFactura != "" {
		if _, err := fmt.Sscanf(ultimaFactura.NumeroFactura, "FAC-%d", &nuevoNumero); err == nil {
			nuevoNumero++
		}
	}
	return fmt.Sprintf("FAC-%d", nuevoNumero)
}

// --- CONSULTAS DE FACTURAS ---

func (d *Db) ObtenerFacturas() ([]Factura, error) {
	var facturas []Factura
	if err := d.DB.Preload("Cliente").Preload("Vendedor").Order("id desc").Find(&facturas).Error; err != nil {
		return nil, err
	}
	return facturas, nil
}

func (d *Db) ObtenerDetalleFactura(facturaID uint) (Factura, error) {
	var factura Factura
	err := d.DB.Preload("Cliente").Preload("Vendedor").Preload("Detalles.Producto").First(&factura, facturaID).Error
	if err != nil {
		return Factura{}, err
	}
	return factura, nil
}

func (d *Db) ObtenerFacturasPorVendedor(vendedorID uint, page, pageSize int) (PaginatedResult, error) {
	var facturas []Factura
	var total int64

	query := d.DB.Model(&Factura{}).Where("vendedor_id = ?", vendedorID)

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Preload("Cliente").Order("id desc").Limit(pageSize).Offset(offset).Find(&facturas).Error
	if err != nil {
		return PaginatedResult{}, err
	}

	return PaginatedResult{Records: facturas, TotalRecords: total}, nil
}

// --- LÓGICA DE INVENTARIO ---

func (d *Db) RegistrarCompra(req CompraRequest) (string, error) {
	tx := d.DB.Begin()
	if tx.Error != nil {
		return "", fmt.Errorf("error al iniciar la transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	// Validar que el proveedor exista
	if err := tx.First(&Proveedor{}, req.ProveedorID).Error; err != nil {
		return "", errors.New("proveedor no encontrado")
	}

	compra := Compra{
		Fecha:         time.Now(),
		ProveedorID:   req.ProveedorID,
		FacturaNumero: req.FacturaNumero,
	}

	var totalCompra float64
	var detallesCompra []DetalleCompra

	for _, p := range req.Productos {
		var producto Producto
		if err := tx.First(&producto, p.ProductoID).Error; err != nil {
			return "", fmt.Errorf("producto con ID %d no encontrado", p.ProductoID)
		}

		precioTotalProducto := p.PrecioCompraUnitario * float64(p.Cantidad)
		totalCompra += precioTotalProducto

		detallesCompra = append(detallesCompra, DetalleCompra{
			ProductoID:           p.ProductoID,
			Cantidad:             p.Cantidad,
			PrecioCompraUnitario: p.PrecioCompraUnitario,
		})

		// Actualizar el stock del producto
		if err := tx.Model(&producto).Update("Stock", gorm.Expr("Stock + ?", p.Cantidad)).Error; err != nil {
			return "", fmt.Errorf("error al actualizar el stock de %s: %w", producto.Nombre, err)
		}
	}

	compra.Total = totalCompra
	if err := tx.Create(&compra).Error; err != nil {
		return "", fmt.Errorf("error al crear la compra: %w", err)
	}

	for i := range detallesCompra {
		detallesCompra[i].CompraID = compra.ID
	}
	if err := tx.Create(&detallesCompra).Error; err != nil {
		return "", fmt.Errorf("error al crear los detalles de la compra: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("error al confirmar la transacción: %w", err)
	}

	return "Compra registrada y stock actualizado correctamente.", nil
}
