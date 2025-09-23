package backend

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
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
	// Encriptamos la contraseña antes de cualquier operación
	hashedPassword, err := HashPassword(vendedor.Contrasena)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al encriptar la contraseña: %w", err)
	}
	vendedor.Contrasena = hashedPassword

	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return Vendedor{}, fmt.Errorf("error al iniciar transacción: %w", tx.Error)
	}
	defer tx.Rollback() // Se revierte si algo falla

	var existente Vendedor
	// Usamos Unscoped() para buscar registros incluso si están marcados como eliminados.
	err = tx.Unscoped().Where("cedula = ? OR email = ?", vendedor.Cedula, vendedor.Email).First(&existente).Error

	if err == nil {
		// Se encontró un vendedor con la misma cédula o email
		if existente.DeletedAt.Valid { // El vendedor estaba eliminado
			d.Log.Infof("Restaurando vendedor eliminado con ID: %d", existente.ID)
			// Actualizamos los datos y lo "revivimos"
			existente.Nombre = vendedor.Nombre
			existente.Apellido = vendedor.Apellido
			existente.Email = vendedor.Email
			existente.Contrasena = vendedor.Contrasena
			existente.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false} // Esto lo restaura

			if err := tx.Unscoped().Save(&existente).Error; err != nil {
				return Vendedor{}, fmt.Errorf("error al restaurar vendedor: %w", err)
			}
			vendedor = existente
		} else {
			// El vendedor existe y está activo, es un error de duplicado
			return Vendedor{}, errors.New("la cédula o el email ya están registrados en un vendedor activo")
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// No existe, lo creamos
		if err := tx.Create(&vendedor).Error; err != nil {
			return Vendedor{}, fmt.Errorf("error al registrar nuevo vendedor: %w", err)
		}
	} else {
		// Ocurrió otro error en la base de datos
		return Vendedor{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Vendedor{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncVendedorToRemote(vendedor.ID)
	vendedor.Contrasena = "" // Limpiamos la contraseña antes de devolverla
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

	// --- INICIO DE LA CORRECCIÓN: Sincronizar antes de limpiar la contraseña ---
	// Si el login fue exitoso contra la BD remota, actualizamos la cache local.
	if d.isRemoteDBAvailable() {
		go d.syncVendedorToLocal(vendedor)
	}

	// Ahora, con la sincronización en camino, limpiamos la contraseña para la respuesta del frontend.
	vendedor.Contrasena = ""
	response = LoginResponse{Token: tokenString, Vendedor: vendedor}

	return response, nil
}

// ObtenerVendedoresPaginado ahora consulta siempre la base de datos local para velocidad y consistencia.
func (d *Db) ObtenerVendedoresPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	d.Log.Infof("Fetching vendedores - Page: %d, PageSize: %d, Search: '%s'", page, pageSize, search)
	var vendedores []Vendedor
	var total int64
	query := d.LocalDB.Model(&Vendedor{})
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(nombre) LIKE ? OR LOWER(apellido) LIKE ? OR cedula LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	allowedSortBy := map[string]string{"Nombre": "nombre", "Cedula": "cedula", "Email": "email"}
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "ASC"
		if strings.ToLower(sortOrder) == "desc" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&vendedores).Error
	for i := range vendedores {
		vendedores[i].Contrasena = ""
	}
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

func (d *Db) RegistrarCliente(cliente Cliente) (Cliente, error) {
	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return Cliente{}, fmt.Errorf("error al iniciar transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	var existente Cliente
	// Usamos Unscoped() para buscar por el identificador único, incluso si está eliminado.
	err := tx.Unscoped().Where("numero_id = ?", cliente.NumeroID).First(&existente).Error

	if err == nil {
		// Se encontró un cliente con el mismo Número de ID
		if existente.DeletedAt.Valid { // El cliente estaba eliminado
			d.Log.Infof("Restaurando cliente eliminado con ID: %d", existente.ID)
			// Actualizamos sus datos y lo restauramos
			existente.Nombre = cliente.Nombre
			existente.Apellido = cliente.Apellido
			existente.TipoID = cliente.TipoID
			existente.Telefono = cliente.Telefono
			existente.Email = cliente.Email
			existente.Direccion = cliente.Direccion
			existente.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false}

			if err := tx.Unscoped().Save(&existente).Error; err != nil {
				return Cliente{}, fmt.Errorf("error al restaurar cliente: %w", err)
			}
			cliente = existente
		} else {
			// El cliente existe y está activo, es un error
			return Cliente{}, errors.New("el número de identificación ya está registrado en un cliente activo")
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// No existe, se crea uno nuevo
		if err := tx.Create(&cliente).Error; err != nil {
			return Cliente{}, fmt.Errorf("error al registrar nuevo cliente: %w", err)
		}
	} else {
		// Otro error de base de datos
		return Cliente{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Cliente{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncClienteToRemote(cliente.ID)
	return cliente, nil
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
func (d *Db) ObtenerClientesPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	d.Log.Infof("Fetching clientes - Page: %d, PageSize: %d, Search: '%s'", page, pageSize, search)
	var clientes []Cliente
	var total int64
	db := d.LocalDB
	query := db.Model(&Cliente{})
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(nombre) LIKE ? OR LOWER(apellido) LIKE ? OR numero_id LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	allowedSortBy := map[string]string{"Nombre": "nombre", "Documento": "numero_id", "Email": "email"}
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "ASC"
		if strings.ToLower(sortOrder) == "desc" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&clientes).Error
	return PaginatedResult{Records: clientes, TotalRecords: total}, err
}

// --- PRODUCTOS ---

func (d *Db) RegistrarProducto(producto Producto) (Producto, error) {
	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return Producto{}, fmt.Errorf("error al iniciar transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	var existente Producto
	err := tx.Unscoped().Where("codigo = ?", producto.Codigo).First(&existente).Error

	if err == nil {
		if existente.DeletedAt.Valid {
			d.Log.Infof("Restaurando producto eliminado con ID: %d", existente.ID)
			existente.Nombre = producto.Nombre
			existente.PrecioVenta = producto.PrecioVenta
			existente.Stock = producto.Stock
			existente.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false}

			if err := tx.Unscoped().Save(&existente).Error; err != nil {
				return Producto{}, fmt.Errorf("error al restaurar producto: %w", err)
			}
			producto = existente
		} else {
			return Producto{}, errors.New("el código del producto ya está en uso")
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&producto).Error; err != nil {
			return Producto{}, fmt.Errorf("error al registrar nuevo producto: %w", err)
		}

		op := OperacionStock{
			UUID:            uuid.New().String(),
			ProductoID:      producto.ID,
			TipoOperacion:   "INICIAL",
			CantidadCambio:  producto.Stock,
			StockResultante: producto.Stock,
			VendedorID:      1, //  ID de un usuario "SISTEMA" o el admin
			Timestamp:       time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return Producto{}, fmt.Errorf("error al crear la operación de stock inicial: %w", err)
		}
	} else {
		return Producto{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Producto{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}
	go d.syncProductoToRemote(producto.ID)
	go d.SincronizarOperacionesStock()
	return producto, nil
}

func (d *Db) ActualizarProducto(p Producto) (string, error) {
	if p.ID == 0 {
		return "", errors.New("para actualizar un producto se requiere un ID válido")
	}

	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return "", fmt.Errorf("error al iniciar transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	var cur Producto
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, p.ID).Error; err != nil {
		return "", err
	}

	cantidadCambio := p.Stock - cur.Stock

	updates := map[string]interface{}{"nombre": p.Nombre, "precio_venta": p.PrecioVenta}
	if err := tx.Model(&Producto{}).Where("id = ?", p.ID).Updates(updates).Error; err != nil {
		return "", err
	}

	// Si hubo un cambio en el stock, se registra la operación
	if cantidadCambio != 0 {
		op := OperacionStock{
			UUID:            uuid.New().String(),
			ProductoID:      p.ID,
			TipoOperacion:   "AJUSTE",
			CantidadCambio:  cantidadCambio,
			StockResultante: p.Stock,
			VendedorID:      1, // Debería ser el ID del usuario logueado
			Timestamp:       time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return "", fmt.Errorf("error al crear la operación de ajuste: %w", err)
		}

		// Actualizamos el stock en la tabla de productos (nuestra caché)
		if err := tx.Model(&Producto{}).Where("id = ?", p.ID).Update("stock", p.Stock).Error; err != nil {
			return "", err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("error al confirmar transacción: %w", err)
	}
	go d.syncProductoToRemote(p.ID)
	go d.SincronizarOperacionesStock()
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
func (d *Db) ObtenerProductosPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	d.Log.Infof("Fetching products - Page: %d, PageSize: %d, Search: '%s'", page, pageSize, search)
	var productos []Producto
	var total int64
	db := d.LocalDB
	query := db.Model(&Producto{})
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(Nombre) LIKE ? OR LOWER(Codigo) LIKE ?", searchTerm, searchTerm)
	}
	allowedSortBy := map[string]string{"Nombre": "nombre", "Codigo": "codigo", "PrecioVenta": "precio_venta", "Stock": "stock"}
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "ASC"
		if strings.ToLower(sortOrder) == "desc" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&productos).Error
	return PaginatedResult{Records: productos, TotalRecords: total}, err
}

// --- PROVEEDORES ---

func (d *Db) RegistrarProveedor(proveedor Proveedor) (string, error) {
	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return "", fmt.Errorf("error al iniciar transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	var existente Proveedor
	// Usamos Unscoped() para buscar por el nombre único, incluso si está eliminado.
	err := tx.Unscoped().Where("nombre = ?", proveedor.Nombre).First(&existente).Error

	var finalID uint
	if err == nil {
		// Se encontró un proveedor con el mismo nombre
		if existente.DeletedAt.Valid { // El proveedor estaba eliminado
			d.Log.Infof("Restaurando proveedor eliminado con ID: %d", existente.ID)
			existente.Telefono = proveedor.Telefono
			existente.Email = proveedor.Email
			existente.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false}

			if err := tx.Unscoped().Save(&existente).Error; err != nil {
				return "", fmt.Errorf("error al restaurar proveedor: %w", err)
			}
			finalID = existente.ID
		} else {
			// El proveedor existe y está activo
			return "", errors.New("el nombre del proveedor ya está en uso")
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// No existe, se crea uno nuevo
		if err := tx.Create(&proveedor).Error; err != nil {
			return "", fmt.Errorf("error al registrar nuevo proveedor: %w", err)
		}
		finalID = proveedor.ID
	} else {
		// Otro error de base de datos
		return "", err
	}

	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncProveedorToRemote(finalID)
	return "Proveedor registrado o restaurado localmente. Sincronizando...", nil
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

		nuevoStock := producto.Stock - p.Cantidad
		op := OperacionStock{
			UUID:            uuid.New().String(),
			ProductoID:      producto.ID,
			TipoOperacion:   "VENTA",
			CantidadCambio:  -p.Cantidad,
			StockResultante: nuevoStock,
			VendedorID:      req.VendedorID,
			Timestamp:       time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return Factura{}, fmt.Errorf("error creando operación de stock para %s: %w", producto.Nombre, err)
		}

		if err := tx.Model(&producto).Update("Stock", nuevoStock).Error; err != nil {
			return Factura{}, fmt.Errorf("error al actualizar el stock de %s: %w", producto.Nombre, err)
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
	factura.IVA = subtotal * 0
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
	if err := tx.Model(&OperacionStock{}).Where("factura_id IS NULL AND vendedor_id = ? AND timestamp > ?", req.VendedorID, time.Now().Add(-5*time.Second)).Update("factura_id", factura.ID).Error; err != nil {
		return Factura{}, fmt.Errorf("error al asociar factura a operaciones de stock: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return Factura{}, fmt.Errorf("error al confirmar transacción local: %w", err)
	}

	go d.syncVentaToRemote(factura.ID)
	go d.SincronizarOperacionesStock()

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
func (d *Db) ObtenerFacturasPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var facturas []Factura
	var total int64
	db := d.LocalDB
	query := db.Model(&Factura{}).Preload("Cliente").Preload("Vendedor")
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Joins("JOIN clientes ON clientes.id = facturas.cliente_id").
			Joins("JOIN vendedors ON vendedors.id = facturas.vendedor_id").
			Where("LOWER(facturas.numero_factura) LIKE ? OR LOWER(clientes.nombre) LIKE ? OR LOWER(clientes.apellido) LIKE ? OR LOWER(vendedors.nombre) LIKE ?",
				searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// El identificador 'fecha_emision' ahora coincide con el `accessorKey` del frontend
	allowedSortBy := map[string]string{
		"NumeroFactura": "numero_factura",
		"fecha_emision": "fecha_emision",
		"Cliente":       "clientes.nombre",
		"Vendedor":      "vendedors.nombre",
		"Total":         "total",
	}

	if col, ok := allowedSortBy[sortBy]; ok {
		order := "ASC"
		if strings.ToLower(sortOrder) == "desc" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	} else {
		query = query.Order("facturas.id DESC")
	}

	if err := query.Count(&total).Error; err != nil {
		return PaginatedResult{}, err
	}
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&facturas).Error
	if err != nil {
		return PaginatedResult{}, err
	}
	for i := range facturas {
		facturas[i].Vendedor.Contrasena = ""
	}
	return PaginatedResult{Records: facturas, TotalRecords: total}, nil
}

func (d *Db) ObtenerDetalleFactura(facturaID uint) (Factura, error) {
	var factura Factura
	err := d.LocalDB.Preload("Cliente").Preload("Vendedor").Preload("Detalles.Producto").First(&factura, facturaID).Error
	if err == nil {
		factura.Vendedor.Contrasena = ""
	}
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

// --- LÓGICA DE SINCRONIZACIÓN ---

func (d *Db) syncVendedorToRemote(id uint) {
	if !d.isRemoteDBAvailable() {
		return
	}
	var record Vendedor
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil { // Usar Unscoped por si se acaba de restaurar
		log.Printf("SYNC ERROR: Could not find Vendedor ID %d in local DB to sync. %v", id, err)
		return
	}
	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cedula"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "apellido", "email", "contrasena", "updated_at", "deleted_at"}),
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
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Could not find Cliente ID %d in local DB to sync. %v", id, err)
		return
	}
	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "numero_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "apellido", "tipo_id", "telefono", "email", "direccion", "updated_at", "deleted_at"}),
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
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Could not find Producto ID %d in local DB to sync. %v", id, err)
		return
	}
	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "codigo"}},
		DoUpdates: clause.AssignmentColumns([]string{"nombre", "precio_venta", "stock", "updated_at", "deleted_at"}),
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
	if err := d.LocalDB.Unscoped().First(&record, id).Error; err != nil {
		log.Printf("SYNC ERROR: Could not find Proveedor ID %d in local DB. %v", id, err)
		return
	}
	err := d.RemoteDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "nombre"}},
		DoUpdates: clause.AssignmentColumns([]string{"telefono", "email", "updated_at", "deleted_at"}),
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
