package backend

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CrearProveedor inserta un nuevo proveedor en la base de datos local.
func (d *Db) CrearProveedor(proveedor *Proveedor) error {
	proveedor.UUID = uuid.New().String()
	proveedor.CreatedAt = time.Now()
	proveedor.UpdatedAt = time.Now()

	query := `
		INSERT INTO proveedors (uuid, nit, nombre, telefono, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := d.DB.Exec(query,
		proveedor.UUID, proveedor.NIT, proveedor.Nombre, proveedor.Telefono,
		proveedor.Email, proveedor.CreatedAt, proveedor.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error al insertar proveedor: %w", err)
	}

	return nil
}

// UpsertProveedorPorNIT inserts or updates a proveedor matched by NIT.
// Used during Gmail invoice sync to auto-populate the suppliers list.
func (d *Db) UpsertProveedorPorNIT(nit, nombre string) error {
	if nit == "" {
		return nil
	}
	_, err := d.DB.Exec(`
		INSERT INTO proveedors (uuid, nit, nombre, telefono, email, created_at, updated_at)
		VALUES ($1, $2, $3, '', '', NOW(), NOW())
		ON CONFLICT (nit) WHERE nit != '' DO UPDATE
		  SET nombre    = EXCLUDED.nombre,
		      updated_at = NOW()`,
		uuid.New().String(), nit, nombre,
	)
	return err
}

// ObtenerProveedoresPaginado recupera una lista paginada de proveedores.
func (d *Db) ObtenerProveedoresPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var proveedores []Proveedor

	baseQuery := "FROM proveedors WHERE deleted_at IS NULL"
	var whereClause string
	var args []interface{}
	argIdx := 1
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		whereClause = fmt.Sprintf(
			" AND (LOWER(nombre) LIKE $%d OR LOWER(email) LIKE $%d OR LOWER(nit) LIKE $%d)",
			argIdx, argIdx+1, argIdx+2,
		)
		args = append(args, searchTerm, searchTerm, searchTerm)
		argIdx += 3
	}

	var total int64
	countQuery := "SELECT COUNT(uuid) " + baseQuery + whereClause
	err := d.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al contar proveedores: %w", err)
	}

	offset := (page - 1) * pageSize
	paginationClause := fmt.Sprintf(" ORDER BY nombre ASC LIMIT %d OFFSET %d", pageSize, offset)

	selectQuery := "SELECT uuid, COALESCE(nit,''), nombre, COALESCE(telefono,''), COALESCE(email,'') " +
		baseQuery + whereClause + paginationClause
	rows, err := d.DB.Query(selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al obtener proveedores paginados: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Proveedor
		if err := rows.Scan(&p.UUID, &p.NIT, &p.Nombre, &p.Telefono, &p.Email); err != nil {
			return PaginatedResult{}, fmt.Errorf("error al escanear proveedor: %w", err)
		}
		proveedores = append(proveedores, p)
	}

	return PaginatedResult{Records: proveedores, TotalRecords: total}, nil
}

// ObtenerProveedorPorUUID busca un proveedor por su UUID.
func (d *Db) ObtenerProveedorPorUUID(uuid string) (Proveedor, error) {
	var p Proveedor
	query := "SELECT uuid, COALESCE(nit,''), nombre, COALESCE(telefono,''), COALESCE(email,'') FROM proveedors WHERE uuid = $1 AND deleted_at IS NULL"

	err := d.DB.QueryRow(query, uuid).Scan(&p.UUID, &p.NIT, &p.Nombre, &p.Telefono, &p.Email)
	if err != nil {
		return Proveedor{}, fmt.Errorf("error al buscar proveedor por UUID %s: %w", uuid, err)
	}

	return p, nil
}

// ActualizarProveedor modifica los datos de un proveedor existente.
func (d *Db) ActualizarProveedor(proveedor *Proveedor) error {
	proveedor.UpdatedAt = time.Now()

	query := `
		UPDATE proveedors
		SET nit = $1, nombre = $2, telefono = $3, email = $4, updated_at = $5
		WHERE uuid = $6`

	_, err := d.DB.Exec(query, proveedor.NIT, proveedor.Nombre, proveedor.Telefono,
		proveedor.Email, proveedor.UpdatedAt, proveedor.UUID)
	if err != nil {
		return fmt.Errorf("error al actualizar proveedor: %w", err)
	}

	return nil
}

// EliminarProveedor realiza un borrado lógico (soft delete) de un proveedor.
func (d *Db) EliminarProveedor(uuid string) error {
	query := "UPDATE proveedors SET deleted_at = $1 WHERE uuid = $2"

	_, err := d.DB.Exec(query, time.Now(), uuid)
	if err != nil {
		return fmt.Errorf("error al eliminar proveedor: %w", err)
	}

	return nil
}
