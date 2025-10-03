package backend

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

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
		if existente.DeletedAt.Valid {
			d.Log.Infof("Restaurando vendedor eliminado con ID: %d", existente.ID)
			existente.Nombre = vendedor.Nombre
			existente.Apellido = vendedor.Apellido
			existente.Email = vendedor.Email
			existente.Contrasena = vendedor.Contrasena
			existente.DeletedAt = gorm.DeletedAt{Time: time.Time{}, Valid: false}

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

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(vendedor.ID)
	}

	vendedor.Contrasena = ""
	return vendedor, nil
}

func (d *Db) LoginVendedor(req LoginRequest) (LoginResponse, error) {
	var vendedor Vendedor
	var response LoginResponse

	db := d.LocalDB
	if d.isRemoteDBAvailable() {
		db = d.RemoteDB
	}

	if err := db.Where("email = ?", req.Email).First(&vendedor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response, errors.New("vendedor no encontrado o credenciales incorrectas")
		}
		return response, err
	}

	if !CheckPasswordHash(req.Contrasena, vendedor.Contrasena) {
		return response, errors.New("vendedor no encontrado o credenciales incorrectas")
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToLocal(vendedor)
	}

	vendedor.Contrasena = ""
	response.Vendedor = vendedor

	if !vendedor.MFAEnabled {
		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &Claims{
			UserID: vendedor.ID,
			Email:  vendedor.Email,
			Nombre: vendedor.Nombre,
			Cedula: vendedor.Cedula,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(d.jwtKey)
		if err != nil {
			return response, fmt.Errorf("no se pudo generar el token: %w", err)
		}
		response.MFARequired = false
		response.Token = tokenString
		return response, nil
	}

	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		UserID:  vendedor.ID,
		Email:   vendedor.Email,
		MFAStep: "pending",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(d.jwtKey)
	if err != nil {
		return response, err
	}

	response.MFARequired = true
	response.Token = tokenString
	return response, nil
}

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

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(vendedorActual.ID)
	}

	return "Perfil actualizado correctamente.", nil
}

func (d *Db) ActualizarVendedor(vendedor Vendedor) (Vendedor, error) {
	if vendedor.ID == 0 {
		return Vendedor{}, errors.New("se requiere un ID de vendedor válido para actualizar")
	}

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

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(vendedor.ID)
	}

	vendedor.Contrasena = ""
	return vendedor, nil
}

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

func (d *Db) EliminarVendedor(id uint) (string, error) {
	if err := d.LocalDB.Session(&gorm.Session{SkipHooks: true}).Delete(&Vendedor{}, id).Error; err != nil {
		return "", err
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(id)
	}

	return "Vendedor eliminado localmente. Sincronizando...", nil
}
