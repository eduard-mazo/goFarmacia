package backend

import (
	"fmt"
	"strings"
	"time"
)

// CrearProducto inserta un nuevo producto en la base de datos local.
func (d *Db) CrearProducto(producto *Producto) error {
	producto.CreatedAt = time.Now()
	producto.UpdatedAt = time.Now()

	query := `
		INSERT INTO productos (codigo, nombre, precio_venta, stock, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := d.LocalDB.Exec(query,
		producto.Codigo, producto.Nombre, producto.PrecioVenta, producto.Stock,
		producto.CreatedAt, producto.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error al insertar producto: %w", err)
	}

	go d.SincronizacionInteligente()
	return nil
}

// ObtenerProductosPaginado recupera una lista paginada de productos con búsqueda.
func (d *Db) ObtenerProductosPaginado(page, pageSize int, search string) (PaginatedResult, error) {
	var productos []Producto

	baseQuery := "FROM productos WHERE deleted_at IS NULL"
	var whereClause string
	var args []interface{}

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		whereClause = " AND (LOWER(nombre) LIKE ? OR LOWER(codigo) LIKE ?)"
		args = append(args, searchTerm, searchTerm)
	}

	var total int64
	countQuery := "SELECT COUNT(id) " + baseQuery + whereClause
	err := d.LocalDB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al contar productos: %w", err)
	}

	offset := (page - 1) * pageSize
	paginationClause := fmt.Sprintf("ORDER BY nombre ASC LIMIT %d OFFSET %d", pageSize, offset)

	selectQuery := "SELECT id, codigo, nombre, precio_venta, stock " + baseQuery + whereClause + paginationClause
	rows, err := d.LocalDB.Query(selectQuery, args...)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("error al obtener productos paginados: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock); err != nil {
			return PaginatedResult{}, fmt.Errorf("error al escanear producto: %w", err)
		}
		productos = append(productos, p)
	}

	return PaginatedResult{Records: productos, TotalRecords: total}, nil
}

// ObtenerProductoPorID busca un producto por su ID.
func (d *Db) ObtenerProductoPorID(id uint) (Producto, error) {
	var p Producto
	query := "SELECT id, codigo, nombre, precio_venta, stock FROM productos WHERE id = ? AND deleted_at IS NULL"

	err := d.LocalDB.QueryRow(query, id).Scan(&p.ID, &p.Codigo, &p.Nombre, &p.PrecioVenta, &p.Stock)
	if err != nil {
		return Producto{}, fmt.Errorf("error al buscar producto por ID %d: %w", id, err)
	}

	return p, nil
}

// ActualizarProducto modifica los datos de un producto existente.
func (d *Db) ActualizarProducto(producto *Producto) error {
	producto.UpdatedAt = time.Now()

	query := "UPDATE productos SET codigo = ?, nombre = ?, precio_venta = ?, updated_at = ? WHERE id = ?"

	_, err := d.LocalDB.Exec(query,
		producto.Codigo, producto.Nombre, producto.PrecioVenta,
		producto.UpdatedAt, producto.ID,
	)
	if err != nil {
		return fmt.Errorf("error al actualizar producto: %w", err)
	}

	go d.SincronizacionInteligente()
	return nil
}

// EliminarProducto realiza un borrado lógico (soft delete) de un producto.
func (d *Db) EliminarProducto(id uint) error {
	query := "UPDATE productos SET deleted_at = ? WHERE id = ?"

	_, err := d.LocalDB.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error al eliminar producto: %w", err)
	}

	go d.SincronizacionInteligente()
	return nil
}
