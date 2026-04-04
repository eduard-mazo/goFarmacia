package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"goFarmacia/backend"
)

// GET /api/pos/verificar
func VerificarImpresora(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		ok := db.VerificarImpresora()
		return c.JSON(http.StatusOK, map[string]bool{"available": ok})
	}
}

// POST /api/pos/imprimir
func ImprimirRecibo(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var f backend.Factura
		if err := c.Bind(&f); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "datos de factura inválidos")
		}
		if err := db.ImprimirRecibo(f); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Recibo enviado a la impresora"})
	}
}
