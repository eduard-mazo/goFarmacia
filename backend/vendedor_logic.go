package backend

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// LoginResponse es la estructura que se devuelve al frontend tras un login exitoso.
type LoginResponse struct {
	Token    string   `json:"token"`
	Vendedor Vendedor `json:"vendedor"`
}

// RegistrarVendedor crea un nuevo vendedor o restaura uno eliminado con la misma cédula/email.
func (d *Db) RegistrarVendedor(vendedor Vendedor) (Vendedor, error) {
	hashedPassword, err := HashPassword(vendedor.Contrasena)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al encriptar la contraseña: %w", err)
	}
	vendedor.Contrasena = hashedPassword

	tx := d.LocalDB.Begin()
	if tx.Error != nil {
		return Vendedor{}, fmt.Errorf("error al iniciar transacción: %w", tx.Error)
	}
	defer tx.Rollback()

	var existente Vendedor
	err = tx.Unscoped().Where("cedula = ? OR email = ?", vendedor.Cedula, vendedor.Email).First(&existente).Error

	if err == nil {
		if existente.DeletedAt.Valid { // El vendedor estaba eliminado y se va a reutilizar.
			d.Log.Infof("Restaurando vendedor eliminado con ID: %d", existente.ID)
			existente.Nombre = vendedor.Nombre
			existente.Apellido = vendedor.Apellido
			existente.Email = vendedor.Email
			existente.Contrasena = vendedor.Contrasena
			existente.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false} // ¡Esta es la lógica de reutilización!

			if err := tx.Unscoped().Save(&existente).Error; err != nil {
				return Vendedor{}, fmt.Errorf("error al restaurar vendedor: %w", err)
			}
			vendedor = existente
		} else {
			return Vendedor{}, errors.New("la cédula o el email ya están registrados en un vendedor activo")
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&vendedor).Error; err != nil {
			return Vendedor{}, fmt.Errorf("error al registrar nuevo vendedor: %w", err)
		}
	} else {
		return Vendedor{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Vendedor{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncVendedorToRemote(vendedor.ID)
	vendedor.Contrasena = ""
	return vendedor, nil
}

// LoginVendedor verifica las credenciales, priorizando la BD remota si está disponible.
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

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToLocal(vendedor)
	}

	vendedor.Contrasena = ""
	response = LoginResponse{Token: tokenString, Vendedor: vendedor}
	return response, nil
}

// ActualizarPerfilVendedor actualiza los datos del PROPIO usuario, incluyendo la contraseña de forma segura.
func (d *Db) ActualizarPerfilVendedor(req VendedorUpdateRequest) (string, error) {
	if req.ID == 0 {
		return "", errors.New("se requiere un ID de vendedor válido")
	}

	var vendedorActual Vendedor
	if err := d.LocalDB.First(&vendedorActual, req.ID).Error; err != nil {
		return "", errors.New("vendedor no encontrado")
	}

	if req.ContrasenaNueva != "" {
		if !CheckPasswordHash(req.ContrasenaActual, vendedorActual.Contrasena) {
			return "", errors.New("la contraseña actual es incorrecta")
		}
		hashedPassword, err := HashPassword(req.ContrasenaNueva)
		if err != nil {
			return "", fmt.Errorf("error al encriptar la nueva contraseña: %w", err)
		}
		vendedorActual.Contrasena = hashedPassword
	}

	vendedorActual.Nombre = req.Nombre
	vendedorActual.Apellido = req.Apellido
	vendedorActual.Cedula = req.Cedula
	vendedorActual.Email = strings.ToLower(req.Email)

	if err := d.LocalDB.Save(&vendedorActual).Error; err != nil {
		return "", err
	}

	go d.syncVendedorToRemote(vendedorActual.ID)
	return "Perfil actualizado correctamente.", nil
}

func (d *Db) ActualizarVendedor(vendedor Vendedor) (Vendedor, error) {
	if vendedor.ID == 0 {
		return Vendedor{}, errors.New("se requiere un ID de vendedor válido para actualizar")
	}

	// Creamos un mapa solo con los campos que queremos actualizar para evitar sobreescribir la contraseña.
	updates := map[string]interface{}{
		"Nombre":   vendedor.Nombre,
		"Apellido": vendedor.Apellido,
		"Cedula":   vendedor.Cedula,
		"Email":    strings.ToLower(vendedor.Email),
	}

	result := d.LocalDB.Model(&Vendedor{}).Where("id = ?", vendedor.ID).Updates(updates)
	if result.Error != nil {
		return Vendedor{}, result.Error
	}

	if result.RowsAffected == 0 {
		return Vendedor{}, errors.New("no se encontró el vendedor para actualizar o los datos no cambiaron")
	}

	// Sincronizamos en segundo plano
	go d.syncVendedorToRemote(vendedor.ID)

	// Devolvemos el objeto actualizado sin la contraseña
	vendedor.Contrasena = ""
	return vendedor, nil
}

// ObtenerVendedoresPaginado retorna una lista paginada de vendedores desde la BD local.
func (d *Db) ObtenerVendedoresPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var vendedores []Vendedor
	var total int64
	query := d.LocalDB.Model(&Vendedor{})

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(nombre) LIKE ? OR LOWER(apellido) LIKE ? OR cedula LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if sortBy != "" && (sortBy == "Nombre" || sortBy == "Cedula" || sortBy == "Email") {
		order := "asc"
		if strings.ToLower(sortOrder) == "desc" {
			order = "desc"
		}
		query = query.Order(fmt.Sprintf("%s %s", strings.ToLower(sortBy), order))
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Limit(pageSize).Offset(offset).Find(&vendedores).Error
	for i := range vendedores {
		vendedores[i].Contrasena = ""
	}
	return PaginatedResult{Records: vendedores, TotalRecords: total}, err
}

// EliminarVendedor realiza un borrado lógico (soft delete) de un vendedor.
func (d *Db) EliminarVendedor(id uint) (string, error) {
	if err := d.LocalDB.Delete(&Vendedor{}, id).Error; err != nil {
		return "", err
	}
	go d.syncVendedorToRemote(id) // Sincroniza el borrado
	return "Vendedor eliminado localmente. Sincronizando...", nil
}
