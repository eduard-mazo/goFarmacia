package backend

import (
	"fmt"
	"strings"
	"time"
)

// CrearCliente inserta un nuevo cliente en la base de datos local.
func (d *Db) CrearCliente(cliente *Cliente) error {
	cliente.CreatedAt = time.Now()
	cliente.UpdatedAt = time.Now()

	query := `
		INSERT INTO clientes (tipo_id, numero_id, nombre, direccion, telefono, email, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := d.LocalDB.Exec(query,
		cliente.TipoID, cliente.NumeroID, cliente.Nombre, cliente.Direccion, cliente.Telefono, cliente.Email,
		cliente.CreatedAt, cliente.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error al insertar cliente: %w", err)
	}

	// Disparamos una sincronización en segundo plano.
	go d.SincronizacionInteligente()

	return nil
}

// ObtenerClientesPaginado recupera una lista paginada de clientes con opción de búsqueda.
func (d *Db) ObtenerClientesPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var clientes []Cliente

	// Base de la query
	baseQuery := "FROM clientes WHERE deleted_at IS NULL"

	// Construcción de la cláusula WHERE para la búsqueda
	var whereClause string
	var args []interface{}
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		whereClause = " AND (LOWER(nombre) LIKE ? OR LOWER(numero_id) LIKE ? OR LOWER(email) LIKE ?)"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Obtener el total de registros que coinciden
	var total int64
	countQuery := "SELECT COUNT(id) " + baseQuery + whereClause
	err := d.LocalDB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al contar clientes: %w", err)
	}

	// Paginación
	offset := (page - 1) * pageSize
	paginationClause := fmt.Sprintf("ORDER BY nombre ASC LIMIT %d OFFSET %d", pageSize, offset)

	// Query final para obtener los registros de la página actual
	selectQuery := "SELECT id, tipo_id, numero_id, nombre, direccion, telefono, email " + baseQuery + whereClause + paginationClause

	rows, err := d.LocalDB.Query(selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al obtener clientes paginados: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c Cliente
		if err := rows.Scan(&c.ID, &c.TipoID, &c.NumeroID, &c.Nombre, &c.Direccion, &c.Telefono, &c.Email); err != nil {
			return PaginatedResult{}, fmt.Errorf("error al escanear cliente: %w", err)
		}
		clientes = append(clientes, c)
	}

	return PaginatedResult{Records: clientes, TotalRecords: total}, nil
}

// ObtenerClientePorID busca un cliente por su ID.
func (d *Db) ObtenerClientePorID(id uint) (Cliente, error) {
	var c Cliente
	query := "SELECT id, tipo_id, numero_id, nombre, direccion, telefono, email FROM clientes WHERE id = ? AND deleted_at IS NULL"

	err := d.LocalDB.QueryRow(query, id).Scan(&c.ID, &c.TipoID, &c.NumeroID, &c.Nombre, &c.Direccion, &c.Telefono, &c.Email)
	if err != nil {
		return Cliente{}, fmt.Errorf("error al buscar cliente por ID %d: %w", id, err)
	}

	return c, nil
}

// ActualizarCliente modifica los datos de un cliente existente.
func (d *Db) ActualizarCliente(cliente *Cliente) error {
	cliente.UpdatedAt = time.Now()

	query := `
		UPDATE clientes
		SET tipo_id = ?, numero_id = ?, nombre = ?, direccion = ?, telefono = ?, email = ?, updated_at = ?
		WHERE id = ?`

	_, err := d.LocalDB.Exec(query,
		cliente.TipoID, cliente.NumeroID, cliente.Nombre, cliente.Direccion, cliente.Telefono, cliente.Email,
		cliente.UpdatedAt, cliente.ID,
	)
	if err != nil {
		return fmt.Errorf("error al actualizar cliente: %w", err)
	}

	go d.SincronizacionInteligente()

	return nil
}

// EliminarCliente realiza un borrado lógico (soft delete) de un cliente.
func (d *Db) EliminarCliente(id uint) error {
	query := "UPDATE clientes SET deleted_at = ? WHERE id = ?"

	_, err := d.LocalDB.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error al eliminar cliente: %w", err)
	}

	go d.SincronizacionInteligente()

	return nil
}
