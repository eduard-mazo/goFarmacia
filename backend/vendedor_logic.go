package backend

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
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

// Login verifica las credenciales de un vendedor.
func (d *Db) Login(cedula, contrasena string) (Vendedor, error) {
	var vendedor Vendedor
	query := "SELECT id, nombre, cedula, contrasena FROM vendedors WHERE cedula = ? AND deleted_at IS NULL"

	err := d.LocalDB.QueryRow(query, cedula).Scan(&vendedor.ID, &vendedor.Nombre, &vendedor.Cedula, &vendedor.Contrasena)
	if err != nil {
		return Vendedor{}, fmt.Errorf("cédula o contraseña incorrecta")
	}

	// Comparar la contraseña hasheada de la BD con la proporcionada.
	err = bcrypt.CompareHashAndPassword([]byte(vendedor.Contrasena), []byte(contrasena))
	if err != nil {
		// El error puede ser por no coincidencia o un error de procesamiento.
		return Vendedor{}, fmt.Errorf("cédula o contraseña incorrecta")
	}

	// No devolver el hash de la contraseña.
	vendedor.Contrasena = ""
	return vendedor, nil
}

// CrearVendedor inserta un nuevo vendedor, hasheando su contraseña.
func (d *Db) CrearVendedor(vendedor *Vendedor) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(vendedor.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error al hashear la contraseña: %w", err)
	}

	vendedor.Contrasena = string(hashedPassword)
	vendedor.CreatedAt = time.Now()
	vendedor.UpdatedAt = time.Now()

	query := "INSERT INTO vendedors (cedula, nombre, contrasena, created_at, updated_at) VALUES (?, ?, ?, ?, ?)"
	_, err = d.LocalDB.Exec(query, vendedor.Cedula, vendedor.Nombre, vendedor.Contrasena, vendedor.CreatedAt, vendedor.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error al insertar vendedor: %w", err)
	}

	go d.SincronizacionInteligente()
	return nil
}

// ObtenerVendedores devuelve una lista de todos los vendedores activos.
func (d *Db) ObtenerVendedores() ([]Vendedor, error) {
	var vendedores []Vendedor
	query := "SELECT id, cedula, nombre FROM vendedors WHERE deleted_at IS NULL ORDER BY nombre ASC"

	rows, err := d.LocalDB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error al obtener vendedores: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v Vendedor
		if err := rows.Scan(&v.ID, &v.Cedula, &v.Nombre); err != nil {
			return nil, fmt.Errorf("error al escanear vendedor: %w", err)
		}
		vendedores = append(vendedores, v)
	}

	return vendedores, nil
}

// ... Puedes añadir aquí las funciones ActualizarVendedor, EliminarVendedor, etc.,
// siguiendo el patrón de los otros archivos _logic.go.
