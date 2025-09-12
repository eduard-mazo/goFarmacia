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

// LoginVendedor verifica las credenciales.
// **NOTA DE DISEÑO:** El login es una de las pocas excepciones donde se consulta primero la BD remota
// si está disponible. Esto asegura que las credenciales más actualizadas (ej. un cambio de contraseña
// desde otro terminal) sean utilizadas, mejorando la seguridad.
func (d *Db) LoginVendedor(req LoginRequest) (LoginResponse, error) {
	var vendedor Vendedor
	var response LoginResponse
	db := d.LocalDB // Por defecto, intentar con la local
	if d.isRemoteDBAvailable() {
		db = d.RemoteDB // Si hay conexión, la remota es la fuente de verdad para auth.
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

	// Si el login fue exitoso contra la BD remota, actualizamos la cache local.
	if d.isRemoteDBAvailable() {
		go d.syncVendedorToLocal(vendedor)
	}
	return response, nil
}

// ObtenerVendedoresPaginado ahora consulta siempre la base de datos local para velocidad y consistencia.
func (d *Db) ObtenerVendedoresPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var vendedores []Vendedor
	var total int64
	db := d.LocalDB // CAMBIO: Siempre se lee de la BD local.
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

func (d *Db) ActualizarVendedor(v Vendedor) (string, error) {
	log.Println("Iniciando actualización de vendedor:", v.ID)
	if v.ID == 0 {
		return "", errors.New("para actualizar un vendedor se requiere un ID válido")
	}

	var cur Vendedor
	if err := d.LocalDB.First(&cur, v.ID).Error; err != nil {
		return "", err
	}

	updates := map[string]interface{}{
		"nombre":   v.Nombre,
		"apellido": v.Apellido,
		"cedula":   v.Cedula,
		"email":    strings.ToLower(v.Email),
	}

	// Solo si se envía una nueva contraseña, hasheamos y actualizamos
	if strings.TrimSpace(v.Contrasena) != "" {
		hp, err := HashPassword(v.Contrasena)
		if err != nil {
			return "", fmt.Errorf("error al encriptar la contraseña: %w", err)
		}
		updates["contrasena"] = hp
	}

	if err := d.LocalDB.Model(&Vendedor{}).
		Where("id = ?", v.ID).
		Updates(updates).Error; err != nil {
		return "", err
	}

	// UPSERT remoto por "cedula"
	go func(id uint) {
		if !d.isRemoteDBAvailable() {
			return
		}
		var record Vendedor
		if err := d.LocalDB.First(&record, id).Error; err != nil {
			d.Log.Warnf("Fallo al cargar vendedor actualizado (ID %d) para sync remoto: %v", id, err)
			return
		}

		if err := d.RemoteDB.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "cedula"}},
				DoUpdates: clause.AssignmentColumns([]string{"nombre", "apellido", "email", "contrasena", "updated_at"}),
			}).
			Create(&record).Error; err != nil {
			d.Log.Warnf("Fallo al sincronizar la actualización para Vendedor ID %d: %v", id, err)
		}
	}(v.ID)

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

// ObtenerClientesPaginado ahora consulta siempre la base de datos local para velocidad y consistencia.
func (d *Db) ObtenerClientesPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var clientes []Cliente
	var total int64
	db := d.LocalDB // CAMBIO: Siempre se lee de la BD local.
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

func (d *Db) ActualizarProducto(p Producto) (string, error) {
	if p.ID == 0 {
		return "", errors.New("para actualizar un producto se requiere un ID válido")
	}

	// Carga actual para tener valores previos (p. ej. para detectar cambio de código)
	var cur Producto
	if err := d.LocalDB.First(&cur, p.ID).Error; err != nil {
		return "", err
	}

	// Solo actualizamos campos permitidos (no tocamos created_at ni deleted_at)
	updates := map[string]interface{}{
		"nombre":       p.Nombre,
		"codigo":       p.Codigo,
		"precio_venta": p.PrecioVenta,
		"stock":        p.Stock,
	}

	// Update local sin sobrescribir toda la fila
	if err := d.LocalDB.Model(&Producto{}).
		Where("id = ?", p.ID).
		Updates(updates).Error; err != nil {
		return "", err
	}

	// Sincroniza remoto con UPSERT por "codigo"
	go func(id uint, oldCode, newCode string) {
		if !d.isRemoteDBAvailable() {
			return
		}

		// Cargamos de nuevo el registro local ya actualizado para enviarlo completo
		var record Producto
		if err := d.LocalDB.First(&record, id).Error; err != nil {
			d.Log.Warnf("Fallo al cargar producto actualizado (ID %d) para sync remoto: %v", id, err)
			return
		}

		// UPSERT en remoto tomando "codigo" como clave natural
		if err := d.RemoteDB.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "codigo"}},
				DoUpdates: clause.AssignmentColumns([]string{"nombre", "precio_venta", "stock", "updated_at"}),
			}).
			Create(&record).Error; err != nil {
			d.Log.Warnf("Fallo al sincronizar la actualización para Producto ID %d: %v", id, err)
		}

		if strings.TrimSpace(oldCode) != "" && oldCode != newCode {
			if err := d.RemoteDB.Unscoped().Where("codigo = ?", oldCode).Delete(&Producto{}).Error; err != nil {
				d.Log.Warnf("No se pudo eliminar en remoto el producto con código antiguo '%s': %v", oldCode, err)
			}
		}
	}(p.ID, cur.Codigo, p.Codigo)

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

// ObtenerProductosPaginado ahora consulta siempre la base de datos local para velocidad y consistencia.
func (d *Db) ObtenerProductosPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	d.Log.Infof("Fetching products - Page: %d, PageSize: %d, Search: '%s'", page, pageSize, search)
	var productos []Producto
	var total int64
	db := d.LocalDB // CAMBIO: Siempre se lee de la BD local.
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

// --- PROVEEDORES ---

func (d *Db) RegistrarProveedor(proveedor Proveedor) (string, error) {
	if err := d.LocalDB.Create(&proveedor).Error; err != nil {
		return "", fmt.Errorf("error al registrar proveedor localmente: %w", err)
	}
	go d.syncProveedorToRemote(proveedor.ID)
	return "Proveedor registrado localmente. Sincronizando...", nil
}

func (d *Db) ActualizarProveedor(proveedor Proveedor) (string, error) {
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

	// --- INICIO DE LA CORRECCIÓN ---
	// Añadimos una nueva goroutine para actualizar el stock en la BD remota.
	// Esto no bloquea la respuesta al usuario y resuelve la inconsistencia de datos.
	go func(productosVendidos []ProductoVenta) {
		if !d.isRemoteDBAvailable() {
			return // No hacer nada si estamos offline
		}
		d.Log.Info("Iniciando sincronización de stock en remoto...")
		for _, p := range productosVendidos {
			// Usamos GORM para construir una sentencia UPDATE segura:
			// UPDATE productos SET stock = stock - [cantidad] WHERE id = [id]
			err := d.RemoteDB.Model(&Producto{}).Where("id = ?", p.ID).Update("Stock", gorm.Expr("Stock - ?", p.Cantidad)).Error
			if err != nil {
				// Si falla, solo lo registramos, no detenemos la aplicación.
				d.Log.Warnf("Fallo al sincronizar el stock del producto ID %d en remoto: %v", p.ID, err)
			}
		}
		d.Log.Info("Sincronización de stock en remoto finalizada.")
	}(req.Productos) // Pasamos una copia de los productos de la request
	// --- FIN DE LA CORRECCIÓN ---

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

// ObtenerFacturas ahora consulta siempre la base de datos local para velocidad y consistencia.
func (d *Db) ObtenerFacturas() ([]Factura, error) {
	db := d.LocalDB // CAMBIO: Siempre se lee de la BD local.
	var facturas []Factura
	err := db.Preload("Cliente").Preload("Vendedor").Order("id desc").Find(&facturas).Error
	return facturas, err
}

func (d *Db) ObtenerDetalleFactura(facturaID uint) (Factura, error) {
	var factura Factura
	// Esta función ya consultaba la BD local, lo cual es correcto.
	err := d.LocalDB.Preload("Cliente").Preload("Vendedor").Preload("Detalles.Producto").First(&factura, facturaID).Error
	return factura, err
}

// --- COMPRAS ---

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

	var compraCompleta Compra
	d.LocalDB.Preload("Proveedor").Preload("Detalles.Producto").First(&compraCompleta, compra.ID)
	return compraCompleta, nil
}

// --- LÓGICA DE SINCRONIZACIÓN (sin cambios) ---
// La lógica existente de "sync individual" es correcta para el modelo local-first.

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
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: No se pudo encontrar la Factura ID %d en la BD local. %v", id, err)
		return
	}

	d.syncVendedorToRemote(record.VendedorID)
	d.syncClienteToRemote(record.ClienteID)

	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "numero_factura"}},
		DoNothing: true,
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
	if err := d.LocalDB.Preload("Detalles").First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: No se pudo encontrar la Compra ID %d en la BD local. %v", id, err)
		return
	}

	d.syncProveedorToRemote(record.ProveedorID)

	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "factura_numero"}},
		DoNothing: true,
	}).Create(&record).Error

	if err != nil {
		log.Printf("SYNC FAILED for Compra ID %d: %v", id, err)
	} else {
		log.Printf("SYNC SUCCESS for Compra ID %d", id)
	}
}

// --- IMPORTACIÓN Y RESETEO ---

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

func (d *Db) ResetearTodaLaData() (string, error) {
	if err := d.DeepResetDatabases(); err != nil {
		return "", err
	}
	return "¡Reseteo completado! Todas las bases de datos han sido limpiadas y reiniciadas.", nil
}
