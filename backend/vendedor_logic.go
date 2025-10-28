package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (d *Db) RegistrarVendedor(vendedor Vendedor) (Vendedor, error) {
	d.Log.Info("[REGISTRANDO]: Vendedor: ", vendedor.Email)

	hashedPassword, err := HashPassword(vendedor.Contrasena)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al encriptar la contraseña: %w", err)
	}

	vendedor.Email = strings.ToLower(vendedor.Email)
	vendedor.UUID = uuid.New().String()
	vendedor.CreatedAt = time.Now()
	vendedor.UpdatedAt = time.Now()
	vendedor.Contrasena = hashedPassword

	ctx := d.ctx
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al iniciar transacción local para transacciones: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); err != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [RegistrarVendedor] rollback %v", err)
		}
	}()

	var deletedAt sql.NullTime
	var existenteUUID sql.NullString
	err = tx.QueryRow("SELECT uuid, deleted_at FROM vendedors WHERE cedula = ? OR email = ?",
		vendedor.Cedula, vendedor.Email).Scan(&existenteUUID, &deletedAt)
	if err != nil && err != sql.ErrNoRows {
		if err != sql.ErrNoRows {
			return Vendedor{}, err
		}
	}

	if existenteUUID.Valid {
		if deletedAt.Valid {
			_, err = tx.Exec("UPDATE vendedors SET nombre = ?, apellido = ?, email = ?, contrasena = ?, deleted_at = NULL, updated_at = ? WHERE uuid = ?",
				vendedor.Nombre, vendedor.Apellido, vendedor.Email, vendedor.Contrasena, time.Now(), existenteUUID.String)
			if err != nil {
				return Vendedor{}, err
			}
			vendedor.UUID = existenteUUID.String
		} else {
			return Vendedor{}, fmt.Errorf("la cédula o el email ya están registrados en un vendedor activo")
		}
	} else {
		_, err := tx.Exec("INSERT INTO vendedors (uuid, nombre, apellido, cedula, email, contrasena, mfa_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", vendedor.UUID, vendedor.Nombre, vendedor.Apellido, vendedor.Cedula, vendedor.Email, vendedor.Contrasena, vendedor.MFAEnabled, time.Now(), time.Now())
		if err != nil {
			return Vendedor{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Vendedor{}, err
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(vendedor.UUID)
	}

	vendedor.Contrasena = ""
	return vendedor, nil
}

func (d *Db) LoginVendedor(req LoginRequest) (LoginResponse, error) {
	d.Log.Infof("Intento log con %s", req)
	var vendedor Vendedor
	var response LoginResponse
	var err error

	if d.isRemoteDBAvailable() {
		ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
		defer cancel()

		row := d.RemoteDB.QueryRow(ctx, `
			SELECT uuid, nombre, apellido, cedula, email, contrasena, mfa_enabled
			FROM vendedors
			WHERE email = $1 AND deleted_at IS NULL
		`, req.Email)

		err = row.Scan(&vendedor.UUID, &vendedor.Nombre, &vendedor.Apellido,
			&vendedor.Cedula, &vendedor.Email, &vendedor.Contrasena, &vendedor.MFAEnabled)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
				d.Log.Warn("Login remoto falló, intentando con base local...")
			} else {
				d.Log.Errorf("Error consultando remoto: %v", err)
			}
		}
	}

	if err != nil {
		row := d.LocalDB.QueryRow(`
			SELECT uuid, nombre, apellido, cedula, email, contrasena, mfa_enabled
			FROM vendedors
			WHERE email = ? AND deleted_at IS NULL
		`, req.Email)
		err = row.Scan(&vendedor.UUID, &vendedor.Nombre, &vendedor.Apellido,
			&vendedor.Cedula, &vendedor.Email, &vendedor.Contrasena, &vendedor.MFAEnabled)
		if err != nil {
			if err == sql.ErrNoRows {
				return response, errors.New("vendedor no encontrado o credenciales incorrectas")
			}
			return response, err
		}
	}

	if !CheckPasswordHash(req.Contrasena, vendedor.Contrasena) {
		return response, errors.New("vendedor no encontrado o credenciales incorrectas")
	}
	if d.isRemoteDBAvailable() {
		go d.syncVendedorToLocal(vendedor)
	}

	if !vendedor.MFAEnabled {
		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &Claims{
			UserUUID: vendedor.UUID,
			Email:    vendedor.Email,
			Nombre:   vendedor.Nombre,
			Cedula:   vendedor.Cedula,
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
	} else {
		expirationTime := time.Now().Add(5 * time.Minute)
		claims := &Claims{
			UserUUID: vendedor.UUID,
			Email:    vendedor.Email,
			MFAStep:  "pending",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(d.jwtKey)
		if err != nil {
			return response, err
		}
		response.Token = tokenString
		response.MFARequired = true
	}
	vendedor.Contrasena = ""
	response.Vendedor = vendedor

	d.Log.Infof("Fin proceso Login con response: %+v", response)
	return response, nil
}

func (d *Db) ActualizarPerfilVendedor(req VendedorUpdateRequest) (string, error) {
	if req.UUID == "" {
		return "", errors.New("se requiere un UUID de vendedor válido")
	}

	// obtener actual
	var vendedorActual Vendedor
	row := d.LocalDB.QueryRow("SELECT uuid, contrasena FROM vendedors WHERE uuid = ?", req.UUID)
	err := row.Scan(&vendedorActual.UUID, &vendedorActual.Contrasena)
	if err != nil {
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
		_, err = d.LocalDB.Exec("UPDATE vendedors SET contrasena = ?, updated_at = ? WHERE uuid = ?", hashedPassword, time.Now(), req.UUID)
		if err != nil {
			return "", err
		}
	}

	_, err = d.LocalDB.Exec("UPDATE vendedors SET nombre = ?, apellido = ?, cedula = ?, email = ?, updated_at = ? WHERE uuid = ?", req.Nombre, req.Apellido, req.Cedula, strings.ToLower(req.Email), time.Now(), req.UUID)
	if err != nil {
		return "", err
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(req.UUID)
	}

	return "Perfil actualizado correctamente.", nil
}

func (d *Db) ActualizarVendedor(vendedor Vendedor) (Vendedor, error) {
	if vendedor.UUID == "" {
		return Vendedor{}, errors.New("se requiere un ID de vendedor válido para actualizar")
	}

	ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE vendedors
		SET nombre = ?, apellido = ?, cedula = ?, email = ?, updated_at = ?
		WHERE uuid = ? AND deleted_at IS NULL
	`

	res, err := d.LocalDB.ExecContext(ctx, query,
		vendedor.Nombre,
		vendedor.Apellido,
		vendedor.Cedula,
		strings.ToLower(vendedor.Email),
		time.Now(),
		vendedor.UUID,
	)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al actualizar vendedor: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return Vendedor{}, errors.New("no se encontró el vendedor para actualizar o los datos no cambiaron")
	}

	// Sincronización remota asincrónica
	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(vendedor.UUID)
	}

	vendedor.Contrasena = ""
	return vendedor, nil
}

func (d *Db) ObtenerVendedoresPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var result PaginatedResult
	var vendedores []Vendedor

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Construcción dinámica del query SQL
	baseQuery := `
		SELECT uuid, nombre, apellido, cedula, email, mfa_enabled, created_at, updated_at
		FROM vendedors
		WHERE deleted_at IS NULL
	`
	var args []interface{}
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		baseQuery += " AND (LOWER(nombre) LIKE ? OR LOWER(apellido) LIKE ? OR LOWER(cedula) LIKE ?)"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Orden dinámico (validado)
	order := "asc"
	if strings.ToLower(sortOrder) == "desc" {
		order = "desc"
	}
	switch strings.ToLower(sortBy) {
	case "nombre", "cedula", "email":
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", strings.ToLower(sortBy), order)
	default:
		baseQuery += " ORDER BY uuid DESC"
	}

	// Contar total
	countQuery := "SELECT COUNT(*) FROM (" + baseQuery + ")"
	err := d.LocalDB.QueryRow(countQuery, args...).Scan(&result.TotalRecords)
	if err != nil {
		return result, fmt.Errorf("error al contar vendedores: %w", err)
	}

	// Agregar paginación
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := d.LocalDB.Query(baseQuery, args...)
	if err != nil {
		return result, fmt.Errorf("error al consultar vendedores: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v Vendedor
		if err := rows.Scan(&v.UUID, &v.Nombre, &v.Apellido, &v.Cedula, &v.Email, &v.MFAEnabled, &v.CreatedAt, &v.UpdatedAt); err != nil {
			d.Log.Errorf("error al escanear vendedor: %v", err)
			continue
		}
		v.Contrasena = ""
		vendedores = append(vendedores, v)
	}

	result.Records = vendedores
	return result, nil
}

func (d *Db) EliminarVendedor(uuid string) (string, error) {
	if uuid == "" {
		return "", errors.New("UUID de vendedor no válido")
	}

	// Soft delete: marcar deleted_at
	_, err := d.LocalDB.Exec(`
		UPDATE vendedors
		SET deleted_at = ?
		WHERE uuid = ? AND deleted_at IS NULL
	`, time.Now(), uuid)
	if err != nil {
		return "", fmt.Errorf("error eliminando vendedor: %w", err)
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(uuid)
	}

	return "Vendedor marcado como eliminado localmente. Sincronizando...", nil
}
