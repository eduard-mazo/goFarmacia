package backend

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LoginResponse es la estructura que se devuelve al frontend tras un login exitoso.
type LoginResponse struct {
	Token    string   `json:"token"`
	Vendedor Vendedor `json:"vendedor"`
}

// --- VENDEDORES ---

// RegistrarVendedor crea un nuevo vendedor con una contraseña encriptada.
func (d *Db) RegistrarVendedor(vendedor Vendedor) (Vendedor, error) {
	hashedPassword, err := HashPassword(vendedor.Contrasena)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al encriptar la contraseña: %w", err)
	}
	vendedor.Contrasena = hashedPassword

	if err := d.LocalDB.Create(&vendedor).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Vendedor{}, errors.New("la cédula o el email ya están registrados")
		}
		return Vendedor{}, fmt.Errorf("error al registrar localmente: %w", err)
	}
	// La sincronización con ON CONFLICT es correcta para nuevos registros.
	go d.syncVendedorToRemote(vendedor.ID)
	vendedor.Contrasena = ""
	return vendedor, nil
}

// LoginVendedor ahora usa hashes y devuelve un token JWT.
func (d *Db) LoginVendedor(req LoginRequest) (LoginResponse, error) {
	var vendedor Vendedor
	var response LoginResponse
	db := d.LocalDB
	if d.isRemoteDBAvailable() {
		db = d.RemoteDB
	}

	if err := db.Where("email = ?", req.Email).First(&vendedor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response, errors.New("vendedor no encontrado")
		}
		return response, err
	}

	if !CheckPasswordHash(req.Contrasena, vendedor.Contrasena) {
		return response, errors.New("contraseña incorrecta")
	}

	tokenString, err := d.GenerateJWT(vendedor)
	if err != nil {
		return response, fmt.Errorf("no se pudo generar el token: %w", err)
	}

	vendedor.Contrasena = ""
	response = LoginResponse{Token: tokenString, Vendedor: vendedor}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToLocal(vendedor)
	}
	return response, nil
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
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(Nombre) LIKE ? OR LOWER(Apellido) LIKE ? OR Cedula LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&vendedores).Error
	return PaginatedResult{Records: vendedores, TotalRecords: total}, err
}

func (d *Db) ActualizarVendedor(vendedor Vendedor) (string, error) {
	// CORRECCIÓN: Se reemplaza la lógica de sincronización genérica por una específica para actualizaciones.
	if vendedor.ID == 0 {
		return "", errors.New("para actualizar un vendedor se requiere un ID válido")
	}

	if err := d.LocalDB.Save(&vendedor).Error; err != nil {
		return "", err
	}

	go func(v Vendedor) {
		if !d.isRemoteDBAvailable() {
			return
		}
		// Usar .Save() en la BD remota. GORM ejecutará un UPDATE porque el ID está presente.
		// Esto funciona correctamente incluso si se cambia la cédula.
		if err := d.RemoteDB.Save(&v).Error; err != nil {
			d.Log.Warnf("Fallo al sincronizar la actualización para Vendedor ID %d: %v", v.ID, err)
		}
	}(vendedor)

	return "Vendedor actualizado localmente. Sincronizando...", nil
}
func (d *Db) EliminarVendedor(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Vendedor{}, id).Error; err != nil {
		return "", err
	}
	go func() {
		if d.isRemoteDBAvailable() {
			if err := d.RemoteDB.Delete(&Vendedor{}, id).Error; err != nil {
				d.Log.Warnf("Failed to sync delete for Vendedor ID %d: %v", id, err)
			}
		}
	}()
	return "Vendedor eliminado localmente. Sincronizando...", nil
}

// --- CLIENTES ---

func (d *Db) RegistrarCliente(cliente Cliente) (string, error) {
	if err := d.LocalDB.Create(&cliente).Error; err != nil {
		return "", fmt.Errorf("error al registrar localmente: %w", err)
	}
	go d.syncClienteToRemote(cliente.ID)
	return "Cliente registrado localmente. Sincronizando...", nil
}

func (d *Db) ActualizarCliente(cliente Cliente) (string, error) {
	// CORRECCIÓN: Lógica de sincronización específica para actualizaciones.
	if cliente.ID == 0 {
		return "", errors.New("para actualizar un cliente se requiere un ID válido")
	}
	if err := d.LocalDB.Save(&cliente).Error; err != nil {
		return "", err
	}
	go func(c Cliente) {
		if !d.isRemoteDBAvailable() {
			return
		}
		if err := d.RemoteDB.Save(&c).Error; err != nil {
			d.Log.Warnf("Fallo al sincronizar la actualización para Cliente ID %d: %v", c.ID, err)
		}
	}(cliente)
	return "Cliente actualizado localmente. Sincronizando...", nil
}

func (d *Db) EliminarCliente(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Cliente{}, id).Error; err != nil {
		return "", err
	}
	go func() {
		if d.isRemoteDBAvailable() {
			if err := d.RemoteDB.Delete(&Cliente{}, id).Error; err != nil {
				d.Log.Warnf("Failed to sync delete for Cliente ID %d: %v", id, err)
			}
		}
	}()
	return "Cliente eliminado localmente. Sincronizando...", nil
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
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(Nombre) LIKE ? OR LOWER(Apellido) LIKE ? OR NumeroID LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&clientes).Error
	return PaginatedResult{Records: clientes, TotalRecords: total}, err
}

// --- PRODUCTOS ---

func (d *Db) RegistrarProducto(producto Producto) (string, error) {
	if err := d.LocalDB.Create(&producto).Error; err != nil {
		return "", fmt.Errorf("error al registrar localmente: %w", err)
	}
	go d.syncProductoToRemote(producto.ID)
	return "Producto registrado localmente. Sincronizando...", nil
}

func (d *Db) ActualizarProducto(producto Producto) (string, error) {
	// CORRECCIÓN: Lógica de sincronización específica para actualizaciones.
	if producto.ID == 0 {
		return "", errors.New("para actualizar un producto se requiere un ID válido")
	}
	if err := d.LocalDB.Save(&producto).Error; err != nil {
		return "", err
	}
	go func(p Producto) {
		if !d.isRemoteDBAvailable() {
			return
		}
		if err := d.RemoteDB.Save(&p).Error; err != nil {
			d.Log.Warnf("Fallo al sincronizar la actualización para Producto ID %d: %v", p.ID, err)
		}
	}(producto)
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

func (d *Db) ObtenerProductosPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	d.Log.Infof("Fetching products - Page: %d, PageSize: %d, Search: '%s'", page, pageSize, search)
	var productos []Producto
	var total int64
	db := d.LocalDB
	if d.isRemoteDBAvailable() {
		db = d.RemoteDB
	}
	query := db.Model(&Producto{})

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(Nombre) LIKE ? OR LOWER(Codigo) LIKE ?", searchTerm, searchTerm)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&productos).Error
	return PaginatedResult{Records: productos, TotalRecords: total}, err
}

// --- PROVEEDORES (NUEVO) ---

func (d *Db) RegistrarProveedor(proveedor Proveedor) (string, error) {
	if err := d.LocalDB.Create(&proveedor).Error; err != nil {
		return "", fmt.Errorf("error al registrar proveedor localmente: %w", err)
	}
	go d.syncProveedorToRemote(proveedor.ID)
	return "Proveedor registrado localmente. Sincronizando...", nil
}

func (d *Db) ActualizarProveedor(proveedor Proveedor) (string, error) {
	// CORRECCIÓN: Lógica de sincronización específica para actualizaciones.
	if proveedor.ID == 0 {
		return "", errors.New("para actualizar un proveedor se requiere un ID válido")
	}
	if err := d.LocalDB.Save(&proveedor).Error; err != nil {
		return "", err
	}
	go func(p Proveedor) {
		if !d.isRemoteDBAvailable() {
			return
		}
		if err := d.RemoteDB.Save(&p).Error; err != nil {
			d.Log.Warnf("Fallo al sincronizar la actualización para Proveedor ID %d: %v", p.ID, err)
		}
	}(proveedor)
	return "Proveedor actualizado localmente. Sincronizando...", nil
}

func (d *Db) EliminarProveedor(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Proveedor{}, id).Error; err != nil {
		return "", err
	}
	go func() {
		if d.isRemoteDBAvailable() {
			if err := d.RemoteDB.Delete(&Proveedor{}, id).Error; err != nil {
				d.Log.Warnf("Failed to sync delete for Proveedor ID %d: %v", id, err)
			}
		}
	}()
	return "Proveedor eliminado localmente. Sincronizando...", nil
}

// --- VENTAS (FACTURAS) ---

func (d *Db) RegistrarVenta(req VentaRequest) (Factura, error) {
	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return Factura{}, fmt.Errorf("error al iniciar transacción local: %w", tx.Error)
	}
	defer tx.Rollback() // Se revierte si algo falla

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

	var subtotal float64
	var detallesFactura []DetalleFactura

	for _, p := range req.Productos {
		var producto Producto
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&producto, p.ID).Error; err != nil {
			return Factura{}, fmt.Errorf("producto con ID %d no encontrado", p.ID)
		}

		if producto.Stock < p.Cantidad {
			return Factura{}, fmt.Errorf("stock insuficiente para %s. Disponible: %d, Solicitado: %d", producto.Nombre, producto.Stock, p.Cantidad)
		}

		precioTotalProducto := p.PrecioUnitario * float64(p.Cantidad)
		subtotal += precioTotalProducto

		detallesFactura = append(detallesFactura, DetalleFactura{
			ProductoID:     producto.ID,
			Cantidad:       p.Cantidad,
			PrecioUnitario: p.PrecioUnitario,
			PrecioTotal:    precioTotalProducto,
		})

		if err := tx.Model(&producto).Update("Stock", gorm.Expr("Stock - ?", p.Cantidad)).Error; err != nil {
			return Factura{}, fmt.Errorf("error al actualizar el stock de %s: %w", producto.Nombre, err)
		}
	}

	factura.Subtotal = subtotal
	factura.IVA = subtotal * 0.19
	factura.Total = factura.Subtotal + factura.IVA
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

	// Sincronizar la venta con la base de datos remota en segundo plano
	go d.syncVentaToRemote(factura.ID)

	return d.ObtenerDetalleFactura(factura.ID)
}

func (d *Db) generarNumeroFactura() string {
	var ultimaFactura Factura
	nuevoNumero := 1000
	if err := d.LocalDB.Order("id desc").Limit(1).Find(&ultimaFactura).Error; err == nil && ultimaFactura.NumeroFactura != "" {
		if _, sscanfErr := fmt.Sscanf(ultimaFactura.NumeroFactura, "FAC-%d", &nuevoNumero); sscanfErr == nil {
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

// --- COMPRAS (NUEVO) ---

func (d *Db) RegistrarCompra(req CompraRequest) (Compra, error) {
	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return Compra{}, fmt.Errorf("error al iniciar transacción de compra: %w", tx.Error)
	}
	defer tx.Rollback()

	compra := Compra{
		Fecha:         time.Now(),
		ProveedorID:   req.ProveedorID,
		FacturaNumero: req.FacturaNumero,
	}

	var totalCompra float64
	var detallesCompra []DetalleCompra

	for _, p := range req.Productos {
		var producto Producto
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&producto, p.ProductoID).Error; err != nil {
			return Compra{}, fmt.Errorf("producto con ID %d no encontrado", p.ProductoID)
		}

		precioTotalProducto := p.PrecioCompraUnitario * float64(p.Cantidad)
		totalCompra += precioTotalProducto

		detallesCompra = append(detallesCompra, DetalleCompra{
			ProductoID:           p.ProductoID,
			Cantidad:             p.Cantidad,
			PrecioCompraUnitario: p.PrecioCompraUnitario,
		})

		if err := tx.Model(&producto).Update("Stock", gorm.Expr("Stock + ?", p.Cantidad)).Error; err != nil {
			return Compra{}, fmt.Errorf("error al actualizar stock por compra para %s: %w", producto.Nombre, err)
		}
	}

	compra.Total = totalCompra
	if err := tx.Create(&compra).Error; err != nil {
		return Compra{}, fmt.Errorf("error al crear la compra: %w", err)
	}

	for i := range detallesCompra {
		detallesCompra[i].CompraID = compra.ID
	}
	if err := tx.Create(&detallesCompra).Error; err != nil {
		return Compra{}, fmt.Errorf("error al crear detalles de compra: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return Compra{}, fmt.Errorf("error al confirmar transacción de compra: %w", err)
	}

	go d.syncCompraToRemote(compra.ID)

	// Devolvemos la compra con sus detalles para confirmación
	var compraCompleta Compra
	d.LocalDB.Preload("Proveedor").Preload("Detalles.Producto").First(&compraCompleta, compra.ID)
	return compraCompleta, nil
}

// --- LÓGICA DE SINCRONIZACIÓN ---

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
	}
}

func (d *Db) syncVentaToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Factura
	// 1. Cargar la factura y sus detalles desde la BD local
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: No se pudo encontrar la Factura ID %d en la BD local. %v", id, err)
		return
	}

	// 2. Asegurar que el VENDEDOR exista en la BD remota ANTES de sincronizar la factura
	d.syncVendedorToRemote(record.VendedorID)

	// 3. Asegurar que el CLIENTE exista en la BD remota ANTES de sincronizar la factura
	d.syncClienteToRemote(record.ClienteID)

	// 4. Ahora, con las dependencias en su lugar, sincronizar la factura
	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "numero_factura"}},
		DoNothing: true, // Las facturas no se actualizan, se anulan y se crean nuevas.
	}).Create(&record).Error

	if err != nil {
		log.Printf("SYNC FAILED for Factura ID %d: %v", id, err)
	} else {
		log.Printf("SYNC SUCCESS for Factura ID %d", id)
	}
}

func (d *Db) syncProveedorToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Proveedor
	if err := d.LocalDB.First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Could not find Proveedor ID %d in local DB. %v", id, err)
		return
	}
	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "nombre"}},
		DoUpdates: clause.AssignmentColumns([]string{"telefono", "email", "updated_at"}),
	}).Create(&record).Error

	if err != nil {
		log.Printf("SYNC FAILED for Proveedor ID %d: %v", id, err)
	}
}

func (d *Db) syncCompraToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Compra
	// 1. Cargar la compra y sus detalles desde la BD local
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: No se pudo encontrar la Compra ID %d en la BD local. %v", id, err)
		return
	}

	// 2. Asegurar que el PROVEEDOR exista en la BD remota ANTES de sincronizar la compra
	d.syncProveedorToRemote(record.ProveedorID)

	// 3. Ahora, con las dependencias en su lugar, sincronizar la compra
	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "factura_numero"}}, // Suponiendo que el número de factura del proveedor es único.
		DoNothing: true,
	}).Create(&record).Error

	if err != nil {
		log.Printf("SYNC FAILED for Compra ID %d: %v", id, err)
	} else {
		log.Printf("SYNC SUCCESS for Compra ID %d", id)
	}
}

// --- IMPORTACIÓN Y RESETEO ---

// ImportaCSV inicia el proceso de carga masiva desde un archivo CSV.
func (d *Db) ImportaCSV(filePath string, modelName string) {
	d.Log.Infof("Iniciando importación para el modelo '%s' desde el archivo: %s", modelName, filePath)
	progressChan, errorChan := d.CargarDesdeCSV(filePath, modelName)
	go func() {
		for msg := range progressChan {
			d.Log.Info(msg)
		}
	}()
	if err := <-errorChan; err != nil {
		d.Log.Errorf("La importación del CSV falló: %v", err)
	} else {
		d.Log.Info("Importación de CSV finalizada con éxito.")
	}
}

// ResetearTodaLaData expone la funcionalidad de reseteo profundo al frontend.
func (d *Db) ResetearTodaLaData() (string, error) {
	if err := d.DeepResetDatabases(); err != nil {
		return "", err
	}
	return "¡Reseteo completado! Todas las bases de datos han sido limpiadas y reiniciadas.", nil
}
