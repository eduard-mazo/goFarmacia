package backend

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *Db) LoginVendedor(req LoginRequest) (Vendedor, error) {
	var vendedor Vendedor

	if d.isRemoteDBAvailable() {
		result := d.RemoteDB.Where("cedula = ?", req.Cedula).First(&vendedor)
		if result.Error == nil {
			if vendedor.Contrasena != req.Contrasena {
				return Vendedor{}, errors.New("contraseña incorrecta")
			}
			go d.syncVendedorToLocal(vendedor)
			return vendedor, nil
		}
	}

	log.Println("Login: Falling back to local database check.")
	result := d.LocalDB.Where("cedula = ?", req.Cedula).First(&vendedor)
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

func (d *Db) ObtenerVendedoresPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var vendedores []Vendedor
	var total int64
	db := d.LocalDB

	if d.isRemoteDBAvailable() {
		db = d.RemoteDB
	}

	query := db.Model(&Vendedor{})
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("Nombre LIKE ? OR Apellido LIKE ? OR Cedula LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&vendedores).Error

	return PaginatedResult{Records: vendedores, TotalRecords: total}, err
}

// ImportaCSV inicia el proceso de carga masiva desde un archivo CSV.
func (d *Db) ImportaCSV(filePath string, modelName string) {
	d.Log.Infof("Iniciando importación para el modelo '%s' desde el archivo: %s", modelName, filePath)

	progressChan, errorChan := d.CargarDesdeCSV(filePath, modelName)

	// Goroutine para escuchar el progreso y no bloquear
	go func() {
		for msg := range progressChan {
			d.Log.Info(msg) // Imprime el progreso en el log del backend
			// Opcional: Podrías emitir un evento al frontend aquí
			// runtime.EventsEmit(d.ctx, "csvProgress", msg)
		}
	}()

	// Esperar el resultado final
	err := <-errorChan
	if err != nil {
		d.Log.Errorf("La importación del CSV falló: %v", err)
		// Opcional: Emitir un evento de error al frontend
		// runtime.EventsEmit(d.ctx, "csvError", err.Error())
	} else {
		d.Log.Info("Importación de CSV finalizada con éxito.")
		// Opcional: Emitir un evento de éxito
		// runtime.EventsEmit(d.ctx, "csvSuccess", "Importación completada")
	}
}

func (d *Db) ObtenerClientesPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var clientes []Cliente
	var total int64
	db := d.LocalDB
	if d.isRemoteDBAvailable() {
		db = d.RemoteDB
	}
	query := db.Model(&Cliente{})
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("Nombre LIKE ? OR Apellido LIKE ? OR NumeroID LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&clientes).Error
	return PaginatedResult{Records: clientes, TotalRecords: total}, err
}

func (d *Db) ObtenerProductosPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var productos []Producto
	var total int64
	db := d.LocalDB
	if d.isRemoteDBAvailable() {
		db = d.RemoteDB
	}
	query := db.Model(&Producto{})
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("Nombre LIKE ? OR Codigo LIKE ?", searchTerm, searchTerm)
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&productos).Error
	return PaginatedResult{Records: productos, TotalRecords: total}, err
}

func (d *Db) RegistrarVendedor(vendedor Vendedor) (string, error) {
	if err := d.LocalDB.Create(&vendedor).Error; err != nil {
		return "", fmt.Errorf("error al registrar localmente: %w", err)
	}
	go d.syncVendedorToRemote(vendedor.ID)
	return "Vendedor registrado localmente. Sincronizando...", nil
}

func (d *Db) ActualizarVendedor(vendedor Vendedor) (string, error) {
	if err := d.LocalDB.Save(&vendedor).Error; err != nil {
		return "", err
	}
	go d.syncVendedorToRemote(vendedor.ID)
	return "Vendedor actualizado localmente. Sincronizando...", nil
}

func (d *Db) EliminarVendedor(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Vendedor{}, id).Error; err != nil {
		return "", err
	}
	go func() {
		if d.isRemoteDBAvailable() {
			if err := d.RemoteDB.Delete(&Vendedor{}, id).Error; err != nil {
				d.Log.Fatalf("Failed to sync delete for Vendedor ID %d: %v", id, err)
			}
		}
	}()
	return "Vendedor eliminado localmente. Sincronizando...", nil
}

func (d *Db) RegistrarCliente(cliente Cliente) (string, error) {
	if err := d.LocalDB.Create(&cliente).Error; err != nil {
		return "", fmt.Errorf("error al registrar localmente: %w", err)
	}
	go d.syncClienteToRemote(cliente.ID)
	return "Cliente registrado localmente. Sincronizando...", nil
}

func (d *Db) ActualizarCliente(cliente Cliente) (string, error) {
	if err := d.LocalDB.Save(&cliente).Error; err != nil {
		return "", err
	}
	go d.syncClienteToRemote(cliente.ID)
	return "Cliente actualizado localmente. Sincronizando...", nil
}

func (d *Db) EliminarCliente(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Cliente{}, id).Error; err != nil {
		return "", err
	}
	go func() {
		if d.isRemoteDBAvailable() {
			if err := d.RemoteDB.Delete(&Cliente{}, id).Error; err != nil {
				d.Log.Fatalf("Failed to sync delete for Cliente ID %d: %v", id, err)
			}
		}
	}()
	return "Cliente eliminado localmente. Sincronizando...", nil
}

func (d *Db) RegistrarProducto(producto Producto) (string, error) {
	if err := d.LocalDB.Create(&producto).Error; err != nil {
		return "", fmt.Errorf("error al registrar localmente: %w", err)
	}
	go d.syncProductoToRemote(producto.ID)
	return "Producto registrado localmente. Sincronizando...", nil
}

func (d *Db) ActualizarProducto(producto Producto) (string, error) {
	if err := d.LocalDB.Save(&producto).Error; err != nil {
		return "", err
	}
	go d.syncProductoToRemote(producto.ID)
	return "Producto actualizado localmente. Sincronizando...", nil
}

func (d *Db) EliminarProducto(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Producto{}, id).Error; err != nil {
		return "", err
	}
	go func() {
		if d.isRemoteDBAvailable() {
			if err := d.RemoteDB.Delete(&Producto{}, id).Error; err != nil {
				log.Printf("Failed to sync delete for Producto ID %d: %v", id, err)
			}
		}
	}()
	return "Producto eliminado localmente. Sincronizando...", nil
}

func (d *Db) RegistrarVenta(req VentaRequest) (Factura, error) {
	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return Factura{}, fmt.Errorf("error al iniciar transacción local: %w", tx.Error)
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
		return Factura{}, fmt.Errorf("error al confirmar transacción local: %w", err)
	}

	return d.ObtenerDetalleFactura(factura.ID)
}

func (d *Db) generarNumeroFactura() string {
	var ultimaFactura Factura
	result := d.LocalDB.Order("id DESC").Limit(1).Find(&ultimaFactura)

	nuevoNumero := 1000
	if result.Error == nil && ultimaFactura.NumeroFactura != "" {
		if _, err := fmt.Sscanf(ultimaFactura.NumeroFactura, "FAC-%d", &nuevoNumero); err == nil {
			nuevoNumero++
		}
	}
	return fmt.Sprintf("FAC-%d", nuevoNumero)
}

func (d *Db) ObtenerFacturas() ([]Factura, error) {
	db := d.LocalDB
	if d.isRemoteDBAvailable() {
		db = d.RemoteDB
	}
	var facturas []Factura
	err := db.Preload("Cliente").Preload("Vendedor").Order("id desc").Find(&facturas).Error
	return facturas, err
}

func (d *Db) ObtenerDetalleFactura(facturaID uint) (Factura, error) {
	var factura Factura
	err := d.LocalDB.Preload("Cliente").Preload("Vendedor").Preload("Detalles.Producto").First(&factura, facturaID).Error
	return factura, err
}

func (d *Db) syncVendedorToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Vendedor
	if err := d.LocalDB.First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Could not find Vendedor ID %d in local DB to sync. %v", id, err)
		return
	}

	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cedula"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "apellido", "email", "contrasena", "updated_at"}),
	}).Create(&record).Error

	if err != nil {
		log.Printf("SYNC FAILED for Vendedor ID %d: %v", id, err)
	} else {
		log.Printf("SYNC SUCCESS for Vendedor ID %d.", id)
	}
}

func (d *Db) syncVendedorToLocal(vendedor Vendedor) {
	err := d.LocalDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cedula"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "apellido", "email", "contrasena", "updated_at"}),
	}).Create(&vendedor).Error

	if err != nil {
		log.Printf("LOCAL CACHE FAILED for Vendedor ID %d: %v", vendedor.ID, err)
	}
}

func (d *Db) syncClienteToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Cliente
	if err := d.LocalDB.First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Could not find Cliente ID %d in local DB to sync. %v", id, err)
		return
	}

	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "numero_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "apellido", "tipo_id", "telefono", "email", "direccion", "updated_at"}),
	}).Create(&record).Error

	if err != nil {
		log.Printf("SYNC FAILED for Cliente ID %d: %v", id, err)
	} else {
		log.Printf("SYNC SUCCESS for Cliente ID %d.", id)
	}
}

func (d *Db) syncProductoToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Producto
	if err := d.LocalDB.First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Could not find Producto ID %d in local DB to sync. %v", id, err)
		return
	}

	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "codigo"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "precio_venta", "stock", "updated_at"}),
	}).Create(&record).Error

	if err != nil {
		log.Printf("SYNC FAILED for Producto ID %d: %v", id, err)
	} else {
		log.Printf("SYNC SUCCESS for Producto ID %d.", id)
	}
}

// ResetearTodaLaData expone la funcionalidad de reseteo profundo al frontend.
// Devuelve un mensaje de éxito o un error.
func (d *Db) ResetearTodaLaData() (string, error) {
	err := d.DeepResetDatabases()
	if err != nil {
		// Devuelve el error al frontend para que pueda ser mostrado.
		return "", err
	}
	return "¡Reseteo completado! Todas las bases de datos han sido limpiadas y reiniciadas.", nil
}
