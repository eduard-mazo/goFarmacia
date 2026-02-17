package tests

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	testDB   *sql.DB
	pool     *dockertest.Pool
	resource *dockertest.Resource
)

// SetupTestDB crea una base de datos PostgreSQL de prueba usando Docker
func SetupTestDB(t *testing.T) *sql.DB {
	if testDB != nil {
		return testDB
	}

	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		t.Fatalf("No se pudo conectar a Docker: %v", err)
	}

	// Iniciar contenedor PostgreSQL
	resource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env: []string{
			"POSTGRES_USER=test_user",
			"POSTGRES_PASSWORD=test_password",
			"POSTGRES_DB=test_db",
			"listen_addresses='*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("No se pudo iniciar contenedor PostgreSQL: %v", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseURL := fmt.Sprintf("postgres://test_user:test_password@%s/test_db?sslmode=disable", hostAndPort)

	// Esperar a que PostgreSQL esté listo
	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}
		return testDB.Ping()
	}); err != nil {
		t.Fatalf("No se pudo conectar a PostgreSQL: %v", err)
	}

	// Ejecutar migraciones
	if err := runTestMigrations(testDB); err != nil {
		t.Fatalf("No se pudieron ejecutar migraciones: %v", err)
	}

	return testDB
}

func TeardownTestDB(t *testing.T) {
	if testDB != nil {
		testDB.Close()
	}
	if pool != nil && resource != nil {
		if err := pool.Purge(resource); err != nil {
			t.Errorf("No se pudo limpiar contenedor: %v", err)
		}
	}
}

func runTestMigrations(db *sql.DB) error {
	// La migración 000001 tiene tablas en orden alfabético con FK cruzadas,
	// lo cual falla en una BD vacía. Creamos las tablas sin FK primero,
	// luego añadimos las constraints, y finalmente aplicamos las migraciones restantes.
	if err := createTablesWithoutFK(db); err != nil {
		return fmt.Errorf("error creando tablas base: %w", err)
	}

	// La migración 000001 es un dump del esquema que ya incluye las columnas
	// añadidas por las migraciones 2-5, por lo que solo aplicamos la 006.
	migrations := []string{
		"../db/migrations/postgres/000006_uuid_primary_keys.up.sql",
	}

	for _, migrationFile := range migrations {
		content, err := os.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("error leyendo migración %s: %w", migrationFile, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("error ejecutando migración %s: %w", migrationFile, err)
		}
	}

	return nil
}

// createTablesWithoutFK reproduce la migración 000001 pero en orden correcto
// de dependencias para una BD vacía.
func createTablesWithoutFK(db *sql.DB) error {
	stmts := []string{
		// Tablas independientes primero
		`CREATE TABLE public.vendedors (
			id bigserial NOT NULL,
			created_at timestamptz NULL,
			updated_at timestamptz NULL,
			deleted_at timestamptz NULL,
			nombre text NULL,
			apellido text NULL,
			cedula text NULL,
			email text NULL,
			contrasena text NULL,
			mfa_secret text NULL,
			mfa_enabled boolean NULL DEFAULT false,
			uuid uuid NOT NULL,
			CONSTRAINT vendedors_pkey PRIMARY KEY (id),
			CONSTRAINT uni_vendedors_cedula UNIQUE (cedula),
			CONSTRAINT uni_vendedors_email UNIQUE (email),
			CONSTRAINT vendedors_uuid_unique UNIQUE (uuid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vendedors_deleted_at ON public.vendedors USING btree (deleted_at)`,

		`CREATE TABLE public.proveedors (
			id bigserial NOT NULL,
			created_at timestamptz NULL,
			updated_at timestamptz NULL,
			deleted_at timestamptz NULL,
			nombre text NULL,
			telefono text NULL,
			email text NULL,
			uuid uuid NOT NULL,
			CONSTRAINT proveedors_pkey PRIMARY KEY (id),
			CONSTRAINT proveedors_uuid_unique UNIQUE (uuid),
			CONSTRAINT uni_proveedors_nombre UNIQUE (nombre)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_proveedors_deleted_at ON public.proveedors USING btree (deleted_at)`,

		`CREATE TABLE public.clientes (
			id bigserial NOT NULL,
			created_at timestamptz NULL,
			updated_at timestamptz NULL,
			deleted_at timestamptz NULL,
			nombre text NULL,
			apellido text NULL,
			tipo_id text NULL,
			numero_id text NULL,
			telefono text NULL,
			email text NULL,
			direccion text NULL,
			uuid uuid NOT NULL,
			CONSTRAINT clientes_pkey PRIMARY KEY (id),
			CONSTRAINT clientes_uuid_unique UNIQUE (uuid),
			CONSTRAINT uni_clientes_numero_id UNIQUE (numero_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_clientes_deleted_at ON public.clientes USING btree (deleted_at)`,

		`CREATE TABLE public.productos (
			id bigserial NOT NULL,
			created_at timestamptz NULL,
			updated_at timestamptz NULL,
			deleted_at timestamptz NULL,
			nombre text NULL,
			codigo text NULL,
			precio_venta numeric NULL,
			stock bigint NULL,
			uuid uuid NOT NULL,
			CONSTRAINT productos_pkey PRIMARY KEY (id),
			CONSTRAINT productos_uuid_unique UNIQUE (uuid),
			CONSTRAINT uni_productos_codigo UNIQUE (codigo)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_productos_deleted_at ON public.productos USING btree (deleted_at)`,

		// Tablas con FK (dependencias ya creadas arriba)
		`CREATE TABLE public.compras (
			id bigserial NOT NULL,
			created_at timestamptz NULL,
			updated_at timestamptz NULL,
			deleted_at timestamptz NULL,
			fecha timestamptz NULL,
			proveedor_id bigint NULL,
			factura_numero text NULL,
			total numeric NULL,
			uuid uuid NOT NULL,
			CONSTRAINT compras_pkey PRIMARY KEY (id),
			CONSTRAINT compras_uuid_unique UNIQUE (uuid),
			CONSTRAINT fk_compras_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedors (id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_compras_deleted_at ON public.compras USING btree (deleted_at)`,

		`CREATE TABLE public.facturas (
			id bigserial NOT NULL,
			created_at timestamptz NULL,
			updated_at timestamptz NULL,
			deleted_at timestamptz NULL,
			numero_factura text NULL,
			fecha_emision timestamptz NULL,
			vendedor_id bigint NULL,
			cliente_id bigint NULL,
			subtotal numeric NULL,
			iva numeric NULL,
			total numeric NULL,
			estado text NULL,
			metodo_pago text NULL,
			uuid uuid NOT NULL,
			CONSTRAINT facturas_pkey PRIMARY KEY (id),
			CONSTRAINT facturas_uuid_unique UNIQUE (uuid),
			CONSTRAINT uni_facturas_numero_factura UNIQUE (numero_factura),
			CONSTRAINT fk_facturas_cliente FOREIGN KEY (cliente_id) REFERENCES clientes (id),
			CONSTRAINT fk_facturas_vendedor FOREIGN KEY (vendedor_id) REFERENCES vendedors (id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_facturas_deleted_at ON public.facturas USING btree (deleted_at)`,

		`CREATE TABLE public.detalle_compras (
			id bigserial NOT NULL,
			compra_id bigint NULL,
			producto_id bigint NULL,
			cantidad bigint NULL,
			precio_compra_unitario numeric NULL,
			uuid uuid NOT NULL,
			CONSTRAINT detalle_compras_pkey PRIMARY KEY (id),
			CONSTRAINT detalle_compras_uuid_unique UNIQUE (uuid),
			CONSTRAINT fk_compras_detalles FOREIGN KEY (compra_id) REFERENCES compras (id),
			CONSTRAINT fk_detalle_compras_producto FOREIGN KEY (producto_id) REFERENCES productos (id)
		)`,

		`CREATE TABLE public.detalle_facturas (
			id bigserial NOT NULL,
			created_at timestamptz NULL,
			updated_at timestamptz NULL,
			deleted_at timestamptz NULL,
			factura_id bigint NULL,
			producto_id bigint NULL,
			cantidad bigint NULL,
			precio_unitario numeric NULL,
			precio_total numeric NULL,
			uuid uuid NOT NULL,
			factura_uuid uuid NULL,
			CONSTRAINT detalle_facturas_pkey PRIMARY KEY (id),
			CONSTRAINT detalle_facturas_factura_id_producto_id_key UNIQUE (factura_id, producto_id),
			CONSTRAINT detalle_facturas_uuid_unique UNIQUE (uuid),
			CONSTRAINT fk_detalle_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES facturas (uuid) ON UPDATE CASCADE ON DELETE CASCADE,
			CONSTRAINT fk_detalle_facturas_producto FOREIGN KEY (producto_id) REFERENCES productos (id),
			CONSTRAINT fk_facturas_detalles FOREIGN KEY (factura_id) REFERENCES facturas (id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_detalle_facturas_deleted_at ON public.detalle_facturas USING btree (deleted_at)`,

		`CREATE TABLE public.operacion_stocks (
			id bigserial NOT NULL,
			uuid text NULL,
			producto_id bigint NULL,
			tipo_operacion text NULL,
			cantidad_cambio bigint NULL,
			stock_resultante bigint NULL,
			vendedor_id bigint NULL,
			factura_id bigint NULL,
			timestamp timestamptz NULL,
			sincronizado boolean NULL DEFAULT false,
			factura_uuid uuid NULL,
			CONSTRAINT operacion_stocks_pkey PRIMARY KEY (id),
			CONSTRAINT fk_factura FOREIGN KEY (factura_id) REFERENCES facturas (id) ON DELETE SET NULL,
			CONSTRAINT fk_operacion_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES facturas (uuid) ON UPDATE CASCADE ON DELETE SET NULL,
			CONSTRAINT fk_producto FOREIGN KEY (producto_id) REFERENCES productos (id) ON DELETE RESTRICT,
			CONSTRAINT fk_vendedor FOREIGN KEY (vendedor_id) REFERENCES vendedors (id) ON DELETE RESTRICT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_operacion_stocks_uuid ON public.operacion_stocks USING btree (uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_operacion_stocks_factura_uuid ON public.operacion_stocks USING btree (factura_uuid)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("error ejecutando DDL: %w\nSQL: %.100s...", err, stmt)
		}
	}

	return nil
}

// CleanDB limpia todas las tablas para tests aislados
func CleanDB(db *sql.DB) error {
	tables := []string{
		"detalle_facturas",
		"facturas",
		"operacion_stocks",
		"productos",
		"clientes",
		"vendedors",
		"proveedors",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return err
		}
	}

	return nil
}
