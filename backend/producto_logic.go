package backend

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RegistrarProducto crea un nuevo producto o restaura uno eliminado.
func (d *Db) RegistrarProducto(producto Producto) (Producto, error) {
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	var existente Producto
	err := tx.Unscoped().Where("codigo = ?", producto.Codigo).First(&existente).Error

	if err == nil {
		if existente.DeletedAt.Valid {
			d.Log.Infof("Restaurando producto eliminado con ID: %d", existente.ID)
			existente.Nombre = producto.Nombre
			existente.PrecioVenta = producto.PrecioVenta
			// El stock se pone en 0 y se crea una operación inicial.
			existente.Stock = 0
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
	} else {
		return Producto{}, err
	}

	// Crear operación de stock INICIAL
	op := OperacionStock{
		UUID: uuid.New().String(), ProductoID: producto.ID, TipoOperacion: "INICIAL",
		CantidadCambio: producto.Stock, VendedorID: 1, Timestamp: time.Now(),
	}
	if err := tx.Create(&op).Error; err != nil {
		return Producto{}, fmt.Errorf("error al crear operación de stock inicial: %w", err)
	}

	// Recalcular el stock para asegurar consistencia desde el principio.
	if err := RecalcularYActualizarStock(tx, producto.ID); err != nil {
		return Producto{}, fmt.Errorf("error al calcular stock inicial: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return Producto{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}
	go d.syncProductoToRemote(producto.ID)
	go d.SincronizarOperacionesStockHaciaRemoto()
	return producto, nil
}

func (d *Db) ActualizarProducto(p Producto) (string, error) {
	if p.ID == 0 {
		return "", errors.New("se requiere un ID de producto válido")
	}
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	var cur Producto
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, p.ID).Error; err != nil {
		return "", err
	}

	if err := tx.Model(&Producto{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"nombre": p.Nombre, "precio_venta": p.PrecioVenta}).Error; err != nil {
		return "", err
	}

	// Si el stock enviado en la UI es diferente al stock real calculado, se crea un ajuste.
	stockReal := d.calcularStockRealLocal(p.ID)
	if cantidadCambio := p.Stock - stockReal; cantidadCambio != 0 {
		op := OperacionStock{
			UUID: uuid.New().String(), ProductoID: p.ID, TipoOperacion: "AJUSTE",
			CantidadCambio: cantidadCambio, VendedorID: 1, Timestamp: time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return "", fmt.Errorf("error al crear operación de ajuste: %w", err)
		}
	}

	// Siempre recalcular desde la fuente de verdad.
	if err := RecalcularYActualizarStock(tx, p.ID); err != nil {
		return "", fmt.Errorf("error al recalcular stock en ajuste: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("error al confirmar transacción: %w", err)
	}
	go d.syncProductoToRemote(p.ID)
	go d.SincronizarOperacionesStockHaciaRemoto()
	return "Producto actualizado localmente. Sincronizando...", nil
}

func (d *Db) EliminarProducto(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Producto{}, id).Error; err != nil {
		return "", err
	}
	go d.syncProductoToRemote(id)
	return "Producto eliminado localmente. Sincronizando...", nil
}

func (d *Db) ObtenerProductosPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var productos []Producto
	var total int64
	query := d.LocalDB.Model(&Producto{})
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(nombre) LIKE ? OR LOWER(codigo) LIKE ?", searchTerm, searchTerm)
	}

	allowedSortBy := map[string]string{"Nombre": "nombre", "Codigo": "codigo", "PrecioVenta": "precio_venta", "Stock": "stock"}
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "asc"
		if strings.ToLower(sortOrder) == "desc" {
			order = "desc"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	} else {
		query = query.Order("nombre asc")
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&productos).Error

	// Es seguro mantener este recálculo aquí. Garantiza que la UI siempre muestre
	// el valor más actualizado posible desde la BD local.
	for i := range productos {
		productos[i].Stock = d.calcularStockRealLocal(productos[i].ID)
	}

	return PaginatedResult{Records: productos, TotalRecords: total}, err
}

// --- NUEVA FUNCIÓN PARA EL FRONTEND ---
func (d *Db) ObtenerHistorialStock(productoID uint) ([]OperacionStock, error) {
	var historial []OperacionStock
	err := d.LocalDB.Where("producto_id = ?", productoID).Order("timestamp desc").Find(&historial).Error
	return historial, err
}
