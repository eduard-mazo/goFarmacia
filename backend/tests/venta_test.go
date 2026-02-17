package tests

import (
	"context"
	"goFarmacia/backend"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVenta_RegistrarVenta(t *testing.T) {
	// Setup
	db := SetupTestDB(t)
	defer TeardownTestDB(t)
	defer CleanDB(db)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	dbInstance := backend.NewTestDb(db, context.Background(), logger)

	// Crear datos de prueba
	vendedorUUID := crearVendedorTest(t, dbInstance)
	clienteUUID := crearClienteTest(t, dbInstance)
	productoUUID := crearProductoTest(t, dbInstance, 100) // 100 unidades en stock

	t.Run("Registrar venta exitosa", func(t *testing.T) {
		ventaReq := backend.VentaRequest{
			ClienteUUID:  clienteUUID,
			VendedorUUID: vendedorUUID,
			MetodoPago:   "efectivo",
			Productos: []backend.ProductoVenta{
				{
					ProductoUUID:   productoUUID,
					Cantidad:       5,
					PrecioUnitario: 5000.00,
				},
			},
		}

		factura, err := dbInstance.RegistrarVenta(ventaReq)
		require.NoError(t, err)
		assert.NotEmpty(t, factura.UUID)
		assert.NotEmpty(t, factura.NumeroFactura)
		assert.Equal(t, 25000.00, factura.Total) // 5 * 5000

		// Verificar stock actualizado
		var stock int
		err = db.QueryRow(`SELECT stock FROM productos WHERE uuid = $1`, productoUUID).Scan(&stock)
		require.NoError(t, err)
		assert.Equal(t, 95, stock) // 100 - 5
	})

	t.Run("Stock insuficiente debe fallar", func(t *testing.T) {
		ventaReq := backend.VentaRequest{
			ClienteUUID:  clienteUUID,
			VendedorUUID: vendedorUUID,
			MetodoPago:   "efectivo",
			Productos: []backend.ProductoVenta{
				{
					ProductoUUID:   productoUUID,
					Cantidad:       200, // Más del stock disponible
					PrecioUnitario: 5000.00,
				},
			},
		}

		_, err := dbInstance.RegistrarVenta(ventaReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stock insuficiente")
	})
}

func crearVendedorTest(t *testing.T, db *backend.Db) string {
	vendedorUUID := uuid.New().String()
	_, err := db.DB.Exec(`
		INSERT INTO vendedors (uuid, nombre, apellido, cedula, email, contrasena, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, vendedorUUID, "Juan", "Perez", "1234567890", "juan@test.com", "hash")
	require.NoError(t, err)
	return vendedorUUID
}

func crearClienteTest(t *testing.T, db *backend.Db) string {
	clienteUUID := uuid.New().String()
	_, err := db.DB.Exec(`
		INSERT INTO clientes (uuid, nombre, apellido, tipo_id, numero_id, telefono, email, direccion, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`, clienteUUID, "Cliente", "Test", "CC", "9876543210", "555-1234", "test@test.com", "Calle 123")
	require.NoError(t, err)
	return clienteUUID
}

func crearProductoTest(t *testing.T, db *backend.Db, stock int) string {
	productoUUID := uuid.New().String()
	_, err := db.DB.Exec(`
		INSERT INTO productos (uuid, nombre, codigo, precio_venta, stock, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, productoUUID, "Test Producto", "TEST"+uuid.New().String()[:8], 5000.00, stock)
	require.NoError(t, err)

	// Crear operación de stock inicial (fuente de verdad para el cálculo de stock)
	_, err = db.DB.Exec(`
		INSERT INTO operacion_stocks (uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, vendedor_uuid, timestamp, sincronizado)
		VALUES ($1, $2, 'INICIAL', $3, $3, 'TEST-SYSTEM', NOW(), false)
	`, uuid.New().String(), productoUUID, stock)
	require.NoError(t, err)

	return productoUUID
}
