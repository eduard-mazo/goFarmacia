package handlers

import (
	"io"
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

// POST /api/pos/imprimir-imagen
// Accepts a multipart file upload (field name: "imagen") or raw image body.
func ImprimirImagen(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var imgData []byte

		// Try multipart form file first.
		file, err := c.FormFile("imagen")
		if err == nil {
			src, err := file.Open()
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "no se pudo abrir el archivo")
			}
			defer src.Close()
			imgData, err = io.ReadAll(src)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "error al leer el archivo")
			}
		} else {
			// Fallback: read raw body.
			imgData, err = io.ReadAll(c.Request().Body)
			if err != nil || len(imgData) == 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "no se recibió imagen")
			}
		}

		if err := db.ImprimirImagen(imgData); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Imagen enviada a la impresora"})
	}
}
