package backend

import (
	"fmt"
	"strings"
	"time"
)

// CrearProveedor inserta un nuevo proveedor en la base de datos local.
func (d *Db) CrearProveedor(proveedor *Proveedor) error {
	proveedor.CreatedAt = time.Now()
	proveedor.UpdatedAt = time.Now()

	query := `
		INSERT INTO proveedors (nombre, contacto, telefono, direccion, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := d.LocalDB.Exec(query,
		proveedor.Nombre, proveedor.Telefono, proveedor.Email, proveedor.CreatedAt, proveedor.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error al insertar proveedor: %w", err)
	}

	go d.SincronizacionInteligente()
	return nil
}

// ObtenerProveedoresPaginado recupera una lista paginada de proveedores.
func (d *Db) ObtenerProveedoresPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var proveedores []Proveedor

	baseQuery := "FROM proveedors WHERE deleted_at IS NULL"
	var whereClause string
	var args []interface{}
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		whereClause = " AND (LOWER(nombre) LIKE ? OR LOWER(contacto) LIKE ?)"
		args = append(args, searchTerm, searchTerm)
	}

	var total int64
	countQuery := "SELECT COUNT(uuid) " + baseQuery + whereClause
	err := d.LocalDB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al contar proveedores: %w", err)
	}

	offset := (page - 1) * pageSize
	paginationClause := fmt.Sprintf("ORDER BY nombre ASC LIMIT %d OFFSET %d", pageSize, offset)

	selectQuery := "SELECT uuid, nombre, contacto, telefono, direccion " + baseQuery + whereClause + paginationClause
	rows, err := d.LocalDB.Query(selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al obtener proveedores paginados: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Proveedor
		if err := rows.Scan(&p.UUID, &p.Nombre, &p.Telefono, &p.Email); err != nil {
			return PaginatedResult{}, fmt.Errorf("error al escanear proveedor: %w", err)
		}
		proveedores = append(proveedores, p)
	}

	return PaginatedResult{Records: proveedores, TotalRecords: total}, nil
}

// ObtenerProveedorPorUUID busca un proveedor por su ID.
func (d *Db) ObtenerProveedorPorUUID(uuid string) (Proveedor, error) {
	var p Proveedor
	query := "SELECT uuid, nombre, contacto, telefono, direccion FROM proveedors WHERE uuid = ? AND deleted_at IS NULL"

	err := d.LocalDB.QueryRow(query, uuid).Scan(&p.Nombre, &p.Email, &p.Telefono)
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
		SET nombre = ?, telefono = ?, email = ?, updated_at = ?
		WHERE uuid = ?`

	_, err := d.LocalDB.Exec(query, proveedor.Nombre, proveedor.Telefono, proveedor.Email, proveedor.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error al actualizar proveedor: %w", err)
	}

	go d.SincronizacionInteligente()
	return nil
}

// EliminarProveedor realiza un borrado lógico (soft delete) de un proveedor.
func (d *Db) EliminarProveedor(uuid string) error {
	query := "UPDATE proveedors SET deleted_at = ? WHERE uuid = ?"

	_, err := d.LocalDB.Exec(query, time.Now(), uuid)
	if err != nil {
		return fmt.Errorf("error al eliminar proveedor: %w", err)
	}

	go d.SincronizacionInteligente()
	return nil
}
