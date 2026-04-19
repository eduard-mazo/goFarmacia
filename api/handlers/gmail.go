package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"goFarmacia/backend"
)

func GmailEstadoAuth(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, g.EstadoAuth())
	}
}

func GmailIniciarOAuth2(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		url, err := g.IniciarOAuth2()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"url": url})
	}
}

func GmailRevocarAuth(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := g.RevocarAuth(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func GmailConfigDir(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"configDir": g.ConfigDir()})
	}
}

func GmailObtenerCredenciales(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, g.ObtenerCredenciales())
	}
}

func GmailGuardarCredenciales(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "no se pudo leer el cuerpo")
		}
		if err := g.GuardarCredenciales(string(body)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func GmailGetSyncProgress(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, g.GetGmailSyncProgress())
	}
}

func GmailSincronizarFacturas(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go g.SincronizarFacturas()
		return c.JSON(http.StatusAccepted, map[string]string{"status": "iniciado"})
	}
}

func GmailSincronizarConOpciones(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var opts backend.SyncOptions
		if err := c.Bind(&opts); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		go g.SincronizarConOpciones(opts)
		return c.JSON(http.StatusAccepted, map[string]string{"status": "iniciado"})
	}
}

func GmailObtenerFacturasCompra(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
		if pageSize < 1 {
			pageSize = 25
		}
		res, err := g.ObtenerFacturasCompra(page, pageSize, c.QueryParam("q"), c.QueryParam("sortBy"), c.QueryParam("sortOrder"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func GmailObtenerDetalleFacturaCompra(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := g.ObtenerDetalleFacturaCompra(c.Param("uuid"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func GmailActualizarEstadoFacturaCompra(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			Estado string `json:"estado"`
		}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := g.ActualizarEstadoFacturaCompra(c.Param("uuid"), body.Estado); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func GmailObtenerProveedoresConEstadisticas(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
		if pageSize < 1 {
			pageSize = 25
		}
		res, err := g.ObtenerProveedoresConEstadisticas(page, pageSize, c.QueryParam("q"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func GmailObtenerTopProductosDeProveedor(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		if limit < 1 {
			limit = 10
		}
		res, err := g.ObtenerTopProductosDeProveedor(c.Param("nit"), limit)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func GmailObtenerResumenCompras(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := g.ObtenerResumenCompras(c.QueryParam("desde"), c.QueryParam("hasta"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func GmailSincronizarProveedoresDesdeFacturas(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		n, err := g.SincronizarProveedoresDesdeFacturas()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]int{"sincronizados": n})
	}
}

func GmailGetAutoSync(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, g.GetAutoSync())
	}
}

func GmailSetAutoSync(g *backend.GmailService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		g.SetAutoSync(body.Enabled)
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}
