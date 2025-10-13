package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

func (d *Db) RegistrarVendedor(vendedor Vendedor) (Vendedor, error) {
	d.Log.Info("[REGISTRANDO]: Vendedor: ", vendedor.Email)

	hashedPassword, err := HashPassword(vendedor.Contrasena)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al encriptar la contraseña: %w", err)
	}

	vendedor.Email = strings.ToLower(vendedor.Email)
	vendedor.CreatedAt = time.Now()
	vendedor.UpdatedAt = time.Now()
	vendedor.Contrasena = hashedPassword

	ctx := d.ctx
	tx, err := d.LocalDB.BeginTx(ctx, nil)
	if err != nil {
		return Vendedor{}, fmt.Errorf("error al iniciar transacción local para transacciones: %w", err)
	}
	defer tx.Rollback()

	var existenteID sql.NullInt64
	var deletedAt sql.NullTime
	err = tx.QueryRow("SELECT id, deleted_at FROM vendedors WHERE cedula = ? OR email = ?",
		vendedor.Cedula, vendedor.Email).Scan(&existenteID, &deletedAt)
	if err != nil && err != sql.ErrNoRows {
		if err != sql.ErrNoRows {
			return Vendedor{}, err
		}
	}

	if existenteID.Valid {
		if deletedAt.Valid {
			_, err = tx.Exec("UPDATE vendedors SET nombre = ?, apellido = ?, email = ?, contrasena = ?, deleted_at = NULL, updated_at = ? WHERE id = ?",
				vendedor.Nombre, vendedor.Apellido, vendedor.Email, vendedor.Contrasena, time.Now(), existenteID.Int64)
			if err != nil {
				return Vendedor{}, err
			}
			vendedor.ID = uint(existenteID.Int64)
		} else {
			return Vendedor{}, fmt.Errorf("la cédula o el email ya están registrados en un vendedor activo")
		}
	} else {
		res, err := tx.Exec("INSERT INTO vendedors (nombre, apellido, cedula, email, contrasena, mfa_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", vendedor.Nombre, vendedor.Apellido, vendedor.Cedula, vendedor.Email, vendedor.Contrasena, vendedor.MFAEnabled, time.Now(), time.Now())
		if err != nil {
			return Vendedor{}, err
		}
		last, err := res.LastInsertId()
		if err != nil {
			return Vendedor{}, err
		}
		vendedor.ID = uint(last)
	}

	if err := tx.Commit(); err != nil {
		return Vendedor{}, err
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
	var err error

	if d.isRemoteDBAvailable() {
		ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
		defer cancel()

		row := d.RemoteDB.QueryRow(ctx, `
			SELECT id, nombre, apellido, cedula, email, contrasena, mfa_enabled
			FROM vendedors
			WHERE email = $1 AND deleted_at IS NULL
		`, req.Email)

		err = row.Scan(&vendedor.ID, &vendedor.Nombre, &vendedor.Apellido,
			&vendedor.Cedula, &vendedor.Email, &vendedor.Contrasena, &vendedor.MFAEnabled)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
				d.Log.Warn("Login remoto falló, intentando con base local...")
			} else {
				d.Log.Errorf("Error consultando remoto: %v", err)
			}
		} else {
			d.Log.Info("Login exitoso en base remota")
		}
	}

	if err != nil {
		row := d.LocalDB.QueryRow(`
			SELECT id, nombre, apellido, cedula, email, contrasena, mfa_enabled
			FROM vendedors
			WHERE email = ? AND deleted_at IS NULL
		`, req.Email)
		err = row.Scan(&vendedor.ID, &vendedor.Nombre, &vendedor.Apellido,
			&vendedor.Cedula, &vendedor.Email, &vendedor.Contrasena, &vendedor.MFAEnabled)
		if err != nil {
			if err == sql.ErrNoRows {
				return response, errors.New("vendedor no encontrado o credenciales incorrectas")
			}
			return response, err
		}
		d.Log.Info("Login exitoso en base local")
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

	vendedor.Contrasena = ""
	response.Vendedor = vendedor
	response.MFARequired = true
	response.Token = tokenString
	return response, nil
}

func (d *Db) ActualizarPerfilVendedor(req VendedorUpdateRequest) (string, error) {
	if req.ID == 0 {
		return "", errors.New("se requiere un ID de vendedor válido")
	}

	// obtener actual
	var vendedorActual Vendedor
	row := d.LocalDB.QueryRow("SELECT id, contrasena FROM vendedors WHERE id = ?", req.ID)
	err := row.Scan(&vendedorActual.ID, &vendedorActual.Contrasena)
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
		_, err = d.LocalDB.Exec("UPDATE vendedors SET contrasena = ?, updated_at = ? WHERE id = ?", hashedPassword, time.Now(), req.ID)
		if err != nil {
			return "", err
		}
	}

	_, err = d.LocalDB.Exec("UPDATE vendedors SET nombre = ?, apellido = ?, cedula = ?, email = ?, updated_at = ? WHERE id = ?", req.Nombre, req.Apellido, req.Cedula, strings.ToLower(req.Email), time.Now(), req.ID)
	if err != nil {
		return "", err
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(req.ID)
	}

	return "Perfil actualizado correctamente.", nil
}

func (d *Db) ActualizarVendedor(vendedor Vendedor) (Vendedor, error) {
	if vendedor.ID == 0 {
		return Vendedor{}, errors.New("se requiere un ID de vendedor válido para actualizar")
	}

	ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE vendedors
		SET nombre = ?, apellido = ?, cedula = ?, email = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	res, err := d.LocalDB.ExecContext(ctx, query,
		vendedor.Nombre,
		vendedor.Apellido,
		vendedor.Cedula,
		strings.ToLower(vendedor.Email),
		time.Now(),
		vendedor.ID,
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
		go d.syncVendedorToRemote(vendedor.ID)
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
		SELECT id, nombre, apellido, cedula, email, mfa_enabled, created_at, updated_at
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
		baseQuery += " ORDER BY id DESC"
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
		if err := rows.Scan(&v.ID, &v.Nombre, &v.Apellido, &v.Cedula, &v.Email, &v.MFAEnabled, &v.CreatedAt, &v.UpdatedAt); err != nil {
			d.Log.Errorf("error al escanear vendedor: %v", err)
			continue
		}
		v.Contrasena = ""
		vendedores = append(vendedores, v)
	}

	result.Records = vendedores
	return result, nil
}

func (d *Db) EliminarVendedor(id uint) (string, error) {
	if id == 0 {
		return "", errors.New("ID de vendedor no válido")
	}

	// Soft delete: marcar deleted_at
	_, err := d.LocalDB.Exec(`
		UPDATE vendedors
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, time.Now(), id)
	if err != nil {
		return "", fmt.Errorf("error eliminando vendedor: %w", err)
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(id)
	}

	return "Vendedor marcado como eliminado localmente. Sincronizando...", nil
}
