package backend

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// RegistrarCliente crea un nuevo cliente o restaura uno eliminado.
func (d *Db) RegistrarCliente(cliente Cliente) (Cliente, error) {
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	var existente Cliente
	err := tx.Unscoped().Where("numero_id = ?", cliente.NumeroID).First(&existente).Error

	if err == nil {
		if existente.DeletedAt.Valid {
			d.Log.Infof("Restaurando cliente eliminado con ID: %d", existente.ID)
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
			return Cliente{}, errors.New("el número de identificación ya está registrado")
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&cliente).Error; err != nil {
			return Cliente{}, fmt.Errorf("error al registrar nuevo cliente: %w", err)
		}
	} else {
		return Cliente{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Cliente{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncClienteToRemote(cliente.ID)
	return cliente, nil
}

// ActualizarCliente actualiza los datos de un cliente.
func (d *Db) ActualizarCliente(cliente Cliente) (string, error) {
	if cliente.ID == 0 {
		return "", errors.New("se requiere un ID de cliente válido")
	}
	if err := d.LocalDB.Save(&cliente).Error; err != nil {
		return "", err
	}
	go d.syncClienteToRemote(cliente.ID)
	return "Cliente actualizado localmente. Sincronizando...", nil
}

// EliminarCliente realiza un borrado lógico de un cliente.
func (d *Db) EliminarCliente(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Cliente{}, id).Error; err != nil {
		return "", err
	}
	go d.syncClienteToRemote(id)
	return "Cliente eliminado localmente. Sincronizando...", nil
}

// ObtenerClientesPaginado retorna una lista paginada de clientes.
func (d *Db) ObtenerClientesPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var clientes []Cliente
	var total int64
	query := d.LocalDB.Model(&Cliente{})

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(nombre) LIKE ? OR LOWER(apellido) LIKE ? OR numero_id LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if sortBy != "" && (sortBy == "Nombre" || sortBy == "Documento" || sortBy == "Email") {
		col := "nombre"
		switch sortBy {
		case "Documento":
			col = "numero_id"
		case "Email":
			col = "email"
		}
		order := "asc"
		if strings.ToLower(sortOrder) == "desc" {
			order = "desc"
		}
		query = query.Order(fmt.Sprintf("%s %s", col, order))
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&clientes).Error
	return PaginatedResult{Records: clientes, TotalRecords: total}, err
}
