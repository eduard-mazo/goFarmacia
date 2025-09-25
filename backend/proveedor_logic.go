package backend

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// RegistrarProveedor crea un nuevo proveedor o restaura uno eliminado.
func (d *Db) RegistrarProveedor(proveedor Proveedor) (string, error) {
	tx := d.LocalDB.Begin()
	defer tx.Rollback()

	var existente Proveedor
	err := tx.Unscoped().Where("nombre = ?", proveedor.Nombre).First(&existente).Error

	var finalID uint
	if err == nil {
		if existente.DeletedAt.Valid {
			d.Log.Infof("Restaurando proveedor eliminado con ID: %d", existente.ID)
			existente.Telefono = proveedor.Telefono
			existente.Email = proveedor.Email
			existente.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false}
			if err := tx.Unscoped().Save(&existente).Error; err != nil {
				return "", fmt.Errorf("error al restaurar proveedor: %w", err)
			}
			finalID = existente.ID
		} else {
			return "", errors.New("el nombre del proveedor ya está en uso")
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&proveedor).Error; err != nil {
			return "", fmt.Errorf("error al registrar nuevo proveedor: %w", err)
		}
		finalID = proveedor.ID
	} else {
		return "", err
	}

	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("error al confirmar transacción: %w", err)
	}
	go d.syncProveedorToRemote(finalID)
	return "Proveedor registrado localmente. Sincronizando...", nil
}

// ActualizarProveedor actualiza los datos de un proveedor.
func (d *Db) ActualizarProveedor(proveedor Proveedor) (string, error) {
	if proveedor.ID == 0 {
		return "", errors.New("se requiere un ID de proveedor válido")
	}
	if err := d.LocalDB.Save(&proveedor).Error; err != nil {
		return "", err
	}
	go d.syncProveedorToRemote(proveedor.ID)
	return "Proveedor actualizado localmente. Sincronizando...", nil
}

// EliminarProveedor realiza un borrado lógico de un proveedor.
func (d *Db) EliminarProveedor(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Proveedor{}, id).Error; err != nil {
		return "", err
	}
	go d.syncProveedorToRemote(id)
	return "Proveedor eliminado localmente. Sincronizando...", nil
}
