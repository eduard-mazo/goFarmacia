package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RegistrarCliente crea un nuevo cliente o restaura uno eliminado usando SQL nativo.
func (d *Db) RegistrarCliente(cliente Cliente) (Cliente, error) {
	tx, err := d.LocalDB.BeginTx(d.ctx, nil)
	if err != nil {
		return Cliente{}, fmt.Errorf("error al iniciar la transacción: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(); rErr != nil && !errors.Is(rErr, sql.ErrTxDone) {
			d.Log.Errorf("[LOCAL] - Error durante [RegistrarCliente] rollback %v", rErr)
		}
	}()

	var txTimestamp time.Time = time.Now()
	cliente.Email = strings.ToLower(cliente.Email)
	cliente.UUID = uuid.New().String()
	cliente.CreatedAt = txTimestamp
	cliente.UpdatedAt = txTimestamp

	var existente struct {
		UUID      sql.NullString
		DeletedAt sql.NullTime
	}
	err = tx.QueryRowContext(d.ctx, "SELECT uuid, deleted_at FROM clientes WHERE numero_id = ?", cliente.NumeroID).Scan(&existente.UUID, &existente.DeletedAt)

	if err != nil && err != sql.ErrNoRows {
		return Cliente{}, fmt.Errorf("error al verificar cliente existente: %w", err)
	}

	if err == nil {
		if existente.DeletedAt.Valid {
			d.Log.Infof("Restaurando cliente eliminado con UUID: %s", existente.UUID.String)
			cliente.UUID = existente.UUID.String
			_, err := tx.ExecContext(d.ctx,
				`UPDATE clientes SET nombre=?, apellido=?, tipo_id=?, telefono=?, email=?, direccion=?, deleted_at=NULL, updated_at=? WHERE uuid=?`,
				cliente.Nombre, cliente.Apellido, cliente.TipoID, cliente.Telefono, cliente.Email, cliente.Direccion, cliente.UpdatedAt, cliente.UUID,
			)
			if err != nil {
				return Cliente{}, fmt.Errorf("error al restaurar cliente: %w", err)
			}
		} else {
			return Cliente{}, fmt.Errorf("el número de identificación ya está registrado")
		}
	} else {
		_, err := tx.ExecContext(d.ctx,
			`INSERT INTO clientes (uuid, nombre, apellido, tipo_id, numero_id, telefono, email, direccion, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			cliente.UUID, cliente.Nombre, cliente.Apellido, cliente.TipoID, cliente.NumeroID, cliente.Telefono, cliente.Email, cliente.Direccion, cliente.CreatedAt, cliente.UpdatedAt,
		)
		if err != nil {
			return Cliente{}, fmt.Errorf("error al registrar nuevo cliente: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Cliente{}, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	go d.syncClienteToRemote(cliente.UUID)
	return cliente, nil
}

// ActualizarCliente actualiza los datos de un cliente existente usando SQL nativo.
func (d *Db) ActualizarCliente(cliente Cliente) (string, error) {
	if cliente.UUID == "" {
		return "", errors.New("se requiere un UUID de cliente válido")
	}

	// Sentencia SQL para actualizar todos los campos relevantes.
	query := `
		UPDATE clientes SET 
			nombre = ?, 
			apellido = ?, 
			tipo_id = ?,
			numero_id = ?, 
			telefono = ?, 
			email = ?, 
			direccion = ?, 
			updated_at = ? 
		WHERE uuid = ?`

	_, err := d.LocalDB.ExecContext(d.ctx, query,
		strings.ToLower(cliente.Nombre),
		strings.ToLower(cliente.Apellido),
		cliente.TipoID,
		cliente.NumeroID,
		cliente.Telefono,
		strings.ToLower(cliente.Email),
		cliente.Direccion,
		time.Now(),
		cliente.UUID,
	)

	if err != nil {
		return "", fmt.Errorf("error al actualizar cliente: %w", err)
	}

	go d.syncClienteToRemote(cliente.UUID)
	return "Cliente actualizado correctamente.", nil
}

// EliminarCliente realiza un borrado lógico (soft delete) de un cliente.
func (d *Db) EliminarCliente(uuid string) (string, error) {
	query := "UPDATE clientes SET deleted_at = ? WHERE uuid = ?"

	_, err := d.LocalDB.Exec(query, time.Now(), uuid)
	if err != nil {
		return "", fmt.Errorf("error al eliminar cliente: %w", err)
	}

	go d.syncClienteToRemote(uuid)

	return "Cliente eliminado localmente. Sincronizando...", nil
}

// ObtenerClientesPaginado recupera una lista paginada de clientes con opción de búsqueda.
func (d *Db) ObtenerClientesPaginado(page, pageSize int, search, sortBy, sortOrder string) (PaginatedResult, error) {
	var clientes []Cliente
	var total int64

	var countArgs []interface{}
	countQuery := "SELECT COUNT(*) FROM clientes WHERE deleted_at IS NULL"
	if search != "" {
		countQuery += " AND (LOWER(nombre) LIKE ? OR LOWER(apellido) LIKE ? OR numero_id LIKE ?)"
		searchTerm := "%" + strings.ToLower(search) + "%"
		countArgs = append(countArgs, searchTerm, searchTerm, searchTerm)
	}

	err := d.LocalDB.QueryRowContext(d.ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al contar clientes: %w", err)
	}

	var queryArgs []interface{}
	query := "SELECT uuid, nombre, apellido, tipo_id, numero_id, telefono, email, direccion FROM clientes WHERE deleted_at IS NULL"
	if search != "" {
		query += " AND (LOWER(nombre) LIKE ? OR LOWER(apellido) LIKE ? OR numero_id LIKE ?)"
		searchTerm := "%" + strings.ToLower(search) + "%"
		queryArgs = append(queryArgs, searchTerm, searchTerm, searchTerm)
	}

	if sortBy != "" {
		col := ""
		switch sortBy {
		case "Nombre":
			col = "nombre"
		case "Documento":
			col = "numero_id"
		case "Email":
			col = "email"
		}

		if col != "" {
			order := "ASC"
			if strings.ToLower(sortOrder) == "desc" {
				order = "DESC"
			}
			query += fmt.Sprintf(" ORDER BY %s %s", col, order)
		}
	}

	query += " LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	queryArgs = append(queryArgs, pageSize, offset)

	rows, err := d.LocalDB.QueryContext(d.ctx, query, queryArgs...)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al obtener clientes paginados: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c Cliente
		if err := rows.Scan(&c.UUID, &c.Nombre, &c.Apellido, &c.TipoID, &c.NumeroID, &c.Telefono, &c.Email, &c.Direccion); err != nil {
			return PaginatedResult{}, fmt.Errorf("error al escanear cliente: %w", err)
		}
		clientes = append(clientes, c)
	}

	return PaginatedResult{Records: clientes, TotalRecords: total}, nil
}

// ObtenerClientePorID busca un cliente por su ID.
func (d *Db) ObtenerClientePorID(uuid string) (Cliente, error) {
	var c Cliente
	query := "SELECT uuid, tipo_id, numero_id, nombre, direccion, telefono, email FROM clientes WHERE uuid = ? AND deleted_at IS NULL"

	err := d.LocalDB.QueryRow(query, uuid).Scan(&c.UUID, &c.TipoID, &c.NumeroID, &c.Nombre, &c.Direccion, &c.Telefono, &c.Email)
	if err != nil {
		return Cliente{}, fmt.Errorf("error al buscar cliente por ID %s: %w", uuid, err)
	}

	return c, nil
}
