package backend

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

)

// ==================== STRUCTS ====================

// TableInfo contains overview metadata about a database table.
type TableInfo struct {
	Name      string `json:"Name"`
	Schema    string `json:"Schema"`
	RowCount  int64  `json:"RowCount"`
	SizeBytes int64  `json:"SizeBytes"`
	SizeHuman string `json:"SizeHuman"`
}

// ColumnInfo describes a single table column.
type ColumnInfo struct {
	Position     int    `json:"Position"`
	Name         string `json:"Name"`
	DataType     string `json:"DataType"`
	IsNullable   bool   `json:"IsNullable"`
	Default      string `json:"Default"`
	IsPrimaryKey bool   `json:"IsPrimaryKey"`
	IsForeignKey bool   `json:"IsForeignKey"`
	MaxLength    int    `json:"MaxLength"`
}

// IndexInfo describes a table index.
type IndexInfo struct {
	Name       string `json:"Name"`
	IsUnique   bool   `json:"IsUnique"`
	IsPrimary  bool   `json:"IsPrimary"`
	Definition string `json:"Definition"`
}

// ConstraintInfo describes a table constraint.
type ConstraintInfo struct {
	Name       string `json:"Name"`
	Type       string `json:"Type"`
	Definition string `json:"Definition"`
}

// TableSchema groups the complete schema information for a table.
type TableSchema struct {
	TableInfo   TableInfo        `json:"TableInfo"`
	Columns     []ColumnInfo     `json:"Columns"`
	Indexes     []IndexInfo      `json:"Indexes"`
	Constraints []ConstraintInfo `json:"Constraints"`
}

// TablePreview holds paginated read-only row data.
type TablePreview struct {
	Columns []string            `json:"Columns"`
	Rows    []map[string]string `json:"Rows"` // all values converted to string for safe JSON
	Total   int64               `json:"Total"`
	Limit   int                 `json:"Limit"`
	Offset  int                 `json:"Offset"`
}

// OperationResult is a generic success/message response.
type OperationResult struct {
	Success  bool   `json:"Success"`
	Message  string `json:"Message"`
	FilePath string `json:"FilePath,omitempty"` // populated for export operations
}

// ==================== VALIDATION ====================

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func validateIdentifier(name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("nombre inválido: %q", name)
	}
	return nil
}

// ==================== QUERY METHODS ====================

// GetTablas returns all user tables in the public schema with row count and size.
func (d *Db) GetTablas() ([]TableInfo, error) {
	if d.DB == nil {
		return nil, fmt.Errorf("no hay conexión a la base de datos")
	}
	ctx, cancel := context.WithTimeout(d.ctx, 15*time.Second)
	defer cancel()

	rows, err := d.DB.QueryContext(ctx, `
		SELECT
			t.table_name,
			t.table_schema,
			COALESCE(pg_total_relation_size(
				quote_ident(t.table_schema)||'.'||quote_ident(t.table_name)
			), 0) AS size_bytes,
			COALESCE(s.n_live_tup, 0) AS row_estimate
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s ON s.relname = t.table_name
		WHERE t.table_schema = 'public'
		  AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Schema, &t.SizeBytes, &t.RowCount); err != nil {
			continue
		}
		t.SizeHuman = humanBytes(t.SizeBytes)
		tables = append(tables, t)
	}
	if tables == nil {
		tables = []TableInfo{}
	}
	return tables, nil
}

// GetEsquemaTabla returns the full schema for the given table.
func (d *Db) GetEsquemaTabla(tableName string) (TableSchema, error) {
	if err := validateIdentifier(tableName); err != nil {
		return TableSchema{}, err
	}
	if d.DB == nil {
		return TableSchema{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
	defer cancel()

	var schema TableSchema
	schema.TableInfo.Name = tableName
	schema.TableInfo.Schema = "public"

	// Table size & row count
	d.DB.QueryRowContext(ctx, `
		SELECT
			COALESCE(pg_total_relation_size(quote_ident('public')||'.'||quote_ident($1)), 0),
			COALESCE(s.n_live_tup, 0)
		FROM pg_stat_user_tables s
		WHERE s.relname = $1
	`, tableName).Scan(&schema.TableInfo.SizeBytes, &schema.TableInfo.RowCount)
	schema.TableInfo.SizeHuman = humanBytes(schema.TableInfo.SizeBytes)

	// Primary key columns
	pkCols := map[string]bool{}
	pkRows, _ := d.DB.QueryContext(ctx, `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema   = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = 'public'
		  AND tc.table_name = $1
	`, tableName)
	if pkRows != nil {
		for pkRows.Next() {
			var c string
			pkRows.Scan(&c)
			pkCols[c] = true
		}
		pkRows.Close()
	}

	// Foreign key columns
	fkCols := map[string]bool{}
	fkRows, _ := d.DB.QueryContext(ctx, `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema   = kcu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = 'public'
		  AND tc.table_name = $1
	`, tableName)
	if fkRows != nil {
		for fkRows.Next() {
			var c string
			fkRows.Scan(&c)
			fkCols[c] = true
		}
		fkRows.Close()
	}

	// Columns
	colRows, err := d.DB.QueryContext(ctx, `
		SELECT
			ordinal_position,
			column_name,
			data_type,
			is_nullable,
			COALESCE(column_default, ''),
			COALESCE(character_maximum_length, 0)
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		return schema, err
	}
	defer colRows.Close()
	for colRows.Next() {
		var col ColumnInfo
		var nullable string
		colRows.Scan(&col.Position, &col.Name, &col.DataType, &nullable, &col.Default, &col.MaxLength)
		col.IsNullable = nullable == "YES"
		col.IsPrimaryKey = pkCols[col.Name]
		col.IsForeignKey = fkCols[col.Name]
		schema.Columns = append(schema.Columns, col)
	}

	// Indexes
	idxRows, err := d.DB.QueryContext(ctx, `
		SELECT
			i.relname,
			ix.indisunique,
			ix.indisprimary,
			pg_get_indexdef(ix.indexrelid)
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i  ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE t.relname = $1 AND n.nspname = 'public'
		ORDER BY i.relname
	`, tableName)
	if err == nil {
		defer idxRows.Close()
		for idxRows.Next() {
			var idx IndexInfo
			idxRows.Scan(&idx.Name, &idx.IsUnique, &idx.IsPrimary, &idx.Definition)
			schema.Indexes = append(schema.Indexes, idx)
		}
	}

	// Constraints
	conRows, err := d.DB.QueryContext(ctx, `
		SELECT
			tc.constraint_name,
			tc.constraint_type,
			COALESCE(pg_get_constraintdef(pgc.oid), '')
		FROM information_schema.table_constraints tc
		JOIN pg_constraint pgc ON pgc.conname = tc.constraint_name
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = $1
		ORDER BY tc.constraint_type, tc.constraint_name
	`, tableName)
	if err == nil {
		defer conRows.Close()
		for conRows.Next() {
			var con ConstraintInfo
			conRows.Scan(&con.Name, &con.Type, &con.Definition)
			schema.Constraints = append(schema.Constraints, con)
		}
	}

	// Ensure non-nil slices for JSON
	if schema.Columns == nil {
		schema.Columns = []ColumnInfo{}
	}
	if schema.Indexes == nil {
		schema.Indexes = []IndexInfo{}
	}
	if schema.Constraints == nil {
		schema.Constraints = []ConstraintInfo{}
	}
	return schema, nil
}

// GetDatosTabla returns paginated row data (all values as strings).
// sortCol and sortDir ("asc"/"desc") are optional; pass empty strings to use default order.
func (d *Db) GetDatosTabla(tableName string, limit int, offset int, sortCol string, sortDir string) (TablePreview, error) {
	if err := validateIdentifier(tableName); err != nil {
		return TablePreview{}, err
	}
	if d.DB == nil {
		return TablePreview{}, fmt.Errorf("no hay conexión a la base de datos")
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	ctx, cancel := context.WithTimeout(d.ctx, 20*time.Second)
	defer cancel()

	var total int64
	d.DB.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM public.%q`, tableName)).Scan(&total)

	// Build ORDER BY clause safely
	orderBy := "1" // default: first column
	if sortCol != "" && validIdentifier.MatchString(sortCol) {
		dir := "ASC"
		if strings.ToUpper(sortDir) == "DESC" {
			dir = "DESC"
		}
		orderBy = fmt.Sprintf("%q %s", sortCol, dir)
	}

	rows, err := d.DB.QueryContext(ctx,
		fmt.Sprintf(`SELECT * FROM public.%q ORDER BY %s LIMIT $1 OFFSET $2`, tableName, orderBy),
		limit, offset)
	if err != nil {
		return TablePreview{}, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return TablePreview{}, err
	}

	result := TablePreview{
		Columns: cols,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Rows:    []map[string]string{},
	}

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)

		row := make(map[string]string, len(cols))
		for i, col := range cols {
			v := vals[i]
			switch x := v.(type) {
			case nil:
				row[col] = "NULL"
			case []byte:
				row[col] = string(x)
			case time.Time:
				row[col] = x.Format("2006-01-02 15:04:05")
			default:
				row[col] = fmt.Sprintf("%v", x)
			}
		}
		result.Rows = append(result.Rows, row)
	}
	return result, nil
}

// ==================== EXPORT ====================

// ExportarTablaCSV exports all rows of a table as a CSV file (file-save dialog).
func (d *Db) ExportarTablaCSV(tableName string) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	tmp, err := os.CreateTemp("", tableName+"-*.csv")
	if err != nil {
		return OperationResult{}, fmt.Errorf("error al crear archivo temporal: %w", err)
	}
	path := tmp.Name()
	tmp.Close()

	ctx, cancel := context.WithTimeout(d.ctx, 60*time.Second)
	defer cancel()

	rows, err := d.DB.QueryContext(ctx, fmt.Sprintf(`SELECT * FROM public.%q`, tableName))
	if err != nil {
		return OperationResult{}, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()

	f, err := os.Create(path)
	if err != nil {
		return OperationResult{}, fmt.Errorf("error al crear archivo: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write(cols)

	count := 0
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		rec := make([]string, len(cols))
		for i, v := range vals {
			switch x := v.(type) {
			case nil:
				rec[i] = ""
			case []byte:
				rec[i] = string(x)
			case time.Time:
				rec[i] = x.Format(time.RFC3339)
			default:
				rec[i] = fmt.Sprintf("%v", x)
			}
		}
		w.Write(rec)
		count++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return OperationResult{}, err
	}
	return OperationResult{
		Success:  true,
		Message:  fmt.Sprintf("%d filas exportadas", count),
		FilePath: path,
	}, nil
}

// ExportarTablaSQL exports a table as SQL INSERT statements.
func (d *Db) ExportarTablaSQL(tableName string) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	tmp, err := os.CreateTemp("", "export-*.sql")
	if err != nil {
		return OperationResult{}, fmt.Errorf("error al crear archivo temporal: %w", err)
	}
	path := tmp.Name()
	tmp.Close()

	ctx, cancel := context.WithTimeout(d.ctx, 60*time.Second)
	defer cancel()

	rows, err := d.DB.QueryContext(ctx, fmt.Sprintf(`SELECT * FROM public.%q`, tableName))
	if err != nil {
		return OperationResult{}, err
	}
	defer rows.Close()
	cols, _ := rows.Columns()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("-- Tabla: %s\n-- Exportado: %s\n\n", tableName, time.Now().Format("2006-01-02 15:04:05")))

	colsQuoted := make([]string, len(cols))
	for i, c := range cols {
		colsQuoted[i] = fmt.Sprintf("%q", c)
	}
	colsList := strings.Join(colsQuoted, ", ")

	count := 0
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)

		valStrs := make([]string, len(cols))
		for i, v := range vals {
			switch x := v.(type) {
			case nil:
				valStrs[i] = "NULL"
			case []byte:
				valStrs[i] = pgQuote(string(x))
			case bool:
				if x {
					valStrs[i] = "TRUE"
				} else {
					valStrs[i] = "FALSE"
				}
			case time.Time:
				valStrs[i] = pgQuote(x.Format(time.RFC3339))
			default:
				valStrs[i] = pgQuote(fmt.Sprintf("%v", x))
			}
		}
		buf.WriteString(fmt.Sprintf("INSERT INTO public.%q (%s) VALUES (%s);\n",
			tableName, colsList, strings.Join(valStrs, ", ")))
		count++
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return OperationResult{}, fmt.Errorf("error al escribir: %w", err)
	}
	return OperationResult{
		Success:  true,
		Message:  fmt.Sprintf("%d filas exportadas", count),
		FilePath: path,
	}, nil
}

// ExportarBDSQL generates a full SQL backup of the database.
// Uses pg_dump if available, otherwise generates INSERT statements for all tables.
func (d *Db) ExportarBDSQL() (OperationResult, error) {
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	tmp, err := os.CreateTemp("", "export-*.sql")
	if err != nil {
		return OperationResult{}, fmt.Errorf("error al crear archivo temporal: %w", err)
	}
	path := tmp.Name()
	tmp.Close()

	// Try pg_dump first
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		if cfg, ok := LoadDBConfig(); ok {
			dbURL = cfg.DSN
		}
	}
	if dbURL != "" {
		if pgDumpPath, lookErr := exec.LookPath("pg_dump"); lookErr == nil {
			cmd := exec.Command(pgDumpPath, "--no-password", "--file", path, dbURL)
			if _, cmdErr := cmd.CombinedOutput(); cmdErr == nil {
				return OperationResult{
					Success:  true,
					Message:  "Backup generado con pg_dump",
					FilePath: path,
				}, nil
			}
			d.Log.Warn("[Backup] pg_dump falló — usando exportación manual")
		}
	}

	// Fallback: manual INSERT backup
	tables, err := d.GetTablas()
	if err != nil {
		return OperationResult{}, err
	}

	ctx, cancel := context.WithTimeout(d.ctx, 120*time.Second)
	defer cancel()

	var buf bytes.Buffer
	buf.WriteString("-- goFarmacia Database Backup\n")
	buf.WriteString(fmt.Sprintf("-- Generado: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString("SET statement_timeout = 0;\nSET client_encoding = 'UTF8';\nBEGIN;\n\n")

	totalRows := 0
	for _, table := range tables {
		buf.WriteString(fmt.Sprintf("\n-- ── %s ──\n", table.Name))
		rows, err := d.DB.QueryContext(ctx, fmt.Sprintf(`SELECT * FROM public.%q`, table.Name))
		if err != nil {
			buf.WriteString(fmt.Sprintf("-- ERROR: %v\n", err))
			continue
		}
		cols, _ := rows.Columns()
		colsQuoted := make([]string, len(cols))
		for i, c := range cols {
			colsQuoted[i] = fmt.Sprintf("%q", c)
		}
		colsList := strings.Join(colsQuoted, ", ")

		for rows.Next() {
			vals := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			rows.Scan(ptrs...)
			valStrs := make([]string, len(cols))
			for i, v := range vals {
				switch x := v.(type) {
				case nil:
					valStrs[i] = "NULL"
				case []byte:
					valStrs[i] = pgQuote(string(x))
				case bool:
					if x {
						valStrs[i] = "TRUE"
					} else {
						valStrs[i] = "FALSE"
					}
				case time.Time:
					valStrs[i] = pgQuote(x.Format(time.RFC3339))
				default:
					valStrs[i] = pgQuote(fmt.Sprintf("%v", x))
				}
			}
			buf.WriteString(fmt.Sprintf("INSERT INTO public.%q (%s) VALUES (%s);\n",
				table.Name, colsList, strings.Join(valStrs, ", ")))
			totalRows++
		}
		rows.Close()
	}
	buf.WriteString("\nCOMMIT;\n")

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return OperationResult{}, fmt.Errorf("error al escribir backup: %w", err)
	}
	return OperationResult{
		Success:  true,
		Message:  fmt.Sprintf("Backup completado: %d filas", totalRows),
		FilePath: path,
	}, nil
}

// ==================== IMPORT ====================

// ImportarTablaCSV imports rows from a CSV file into the given table (append).
func (d *Db) ImportarTablaCSV(tableName string) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	return OperationResult{Success: false, Message: "Usa el endpoint HTTP /api/admin/tablas/" + tableName + "/import para importar vía HTTP"}, nil
}

// ImportarSQL executes SQL from a user-selected file (backup restore, etc.).
func (d *Db) ImportarSQL() (OperationResult, error) {
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	return OperationResult{Success: false, Message: "Usa el endpoint HTTP /api/admin/tablas/.../import para importar vía HTTP"}, nil
}

// ==================== ROW EDIT / DELETE ====================

// ActualizarFilaTabla updates a single cell identified by the primary key.
// Pass isNull=true to set the field to NULL (newValue is ignored).
func (d *Db) ActualizarFilaTabla(tableName, pkColumn, pkValue, updateColumn, newValue string, isNull bool) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if err := validateIdentifier(pkColumn); err != nil {
		return OperationResult{}, err
	}
	if err := validateIdentifier(updateColumn); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
	defer cancel()

	var val interface{}
	if !isNull {
		val = newValue
	}

	// Cast pk to text for comparison — works with uuid, int, bigint, varchar, etc.
	query := fmt.Sprintf(`UPDATE public.%q SET %q = $1 WHERE %q::text = $2`, tableName, updateColumn, pkColumn)
	res, err := d.DB.ExecContext(ctx, query, val, pkValue)
	if err != nil {
		return OperationResult{}, fmt.Errorf("error al actualizar: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return OperationResult{Success: false, Message: "No se encontró la fila para actualizar"}, nil
	}
	return OperationResult{Success: true, Message: fmt.Sprintf("Fila actualizada (%d)", affected)}, nil
}

// EliminarFilaTabla deletes a single row identified by the primary key.
func (d *Db) EliminarFilaTabla(tableName, pkColumn, pkValue string) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if err := validateIdentifier(pkColumn); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}

	ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
	defer cancel()

	query := fmt.Sprintf(`DELETE FROM public.%q WHERE %q::text = $1`, tableName, pkColumn)
	res, err := d.DB.ExecContext(ctx, query, pkValue)
	if err != nil {
		return OperationResult{}, fmt.Errorf("error al eliminar: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return OperationResult{Success: false, Message: "No se encontró la fila para eliminar"}, nil
	}
	return OperationResult{Success: true, Message: "Fila eliminada"}, nil
}

// ==================== TABLE OPERATIONS ====================

// AnalizarTabla runs ANALYZE on the table to update query planner statistics.
func (d *Db) AnalizarTabla(tableName string) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	if _, err := d.DB.ExecContext(ctx, fmt.Sprintf(`ANALYZE public.%q`, tableName)); err != nil {
		return OperationResult{}, err
	}
	return OperationResult{Success: true, Message: "ANALYZE completado para " + tableName}, nil
}

// VacuumTabla runs VACUUM ANALYZE on the table.
func (d *Db) VacuumTabla(tableName string) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}
	// VACUUM cannot run inside a transaction; use a direct connection
	ctx, cancel := context.WithTimeout(d.ctx, 60*time.Second)
	defer cancel()

	if _, err := d.DB.ExecContext(ctx, fmt.Sprintf(`VACUUM ANALYZE public.%q`, tableName)); err != nil {
		return OperationResult{}, err
	}
	return OperationResult{Success: true, Message: "VACUUM ANALYZE completado para " + tableName}, nil
}

// TruncarTabla deletes all rows from the table (TRUNCATE CASCADE).
func (d *Db) TruncarTabla(tableName string) (OperationResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return OperationResult{}, err
	}
	if d.DB == nil {
		return OperationResult{}, fmt.Errorf("no hay conexión a la base de datos")
	}
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	if _, err := d.DB.ExecContext(ctx, fmt.Sprintf(`TRUNCATE public.%q RESTART IDENTITY CASCADE`, tableName)); err != nil {
		return OperationResult{}, err
	}
	return OperationResult{Success: true, Message: "Tabla '" + tableName + "' vaciada correctamente"}, nil
}

// ==================== HELPERS ====================

func pgQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func humanBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/GB)
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/MB)
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/KB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
