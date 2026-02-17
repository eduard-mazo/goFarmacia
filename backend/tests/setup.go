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
	// Leer y ejecutar todas las migraciones
	migrations := []string{
		"../db/migrations/postgres/000001_initial_schema.up.sql",
		"../db/migrations/postgres/000002_add_uuids_to_transactions.up.sql",
		"../db/migrations/postgres/000003_add_factura_uuid_to_detalle_facturas.up.sql",
		"../db/migrations/postgres/000004_add_factura_uuid_to_operaciones_stock.up.sql",
		"../db/migrations/postgres/000005_add_uuid_to_models.up.sql",
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
