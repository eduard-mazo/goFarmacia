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
			UUID: uuid.New().String(), ProductoID: producto.ID, TipoOperacion: "INICIAL",
			CantidadCambio: producto.Stock, StockResultante: producto.Stock, VendedorID: 1, Timestamp: time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return Producto{}, fmt.Errorf("error al crear operación de stock inicial: %w", err)
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

// ActualizarProducto actualiza los datos de un producto y registra ajustes de stock.
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

	if cantidadCambio := p.Stock - cur.Stock; cantidadCambio != 0 {
		op := OperacionStock{
			UUID: uuid.New().String(), ProductoID: p.ID, TipoOperacion: "AJUSTE",
			CantidadCambio: cantidadCambio, StockResultante: p.Stock, VendedorID: 1, Timestamp: time.Now(),
		}
		if err := tx.Create(&op).Error; err != nil {
			return "", fmt.Errorf("error al crear operación de ajuste: %w", err)
		}
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

// EliminarProducto realiza un borrado lógico de un producto.
func (d *Db) EliminarProducto(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Producto{}, id).Error; err != nil {
		return "", err
	}
	go d.syncProductoToRemote(id)
	return "Producto eliminado localmente. Sincronizando...", nil
}

// ObtenerProductosPaginado retorna una lista paginada de productos.
func (d *Db) ObtenerProductosPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var productos []Producto
	var total int64
	query := d.LocalDB.Model(&Producto{})
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(Nombre) LIKE ? OR LOWER(Codigo) LIKE ?", searchTerm, searchTerm)
	}

	allowedSortBy := map[string]string{"Nombre": "nombre", "Codigo": "codigo", "PrecioVenta": "precio_venta", "Stock": "stock"}
	if col, ok := allowedSortBy[sortBy]; ok {
		order := "asc"
		if strings.ToLower(sortOrder) == "desc" {
			order = "desc"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&productos).Error

	for i := range productos {
		productos[i].Stock = d.calcularStockRealLocal(productos[i].ID)
	}
	return PaginatedResult{Records: productos, TotalRecords: total}, err
}
