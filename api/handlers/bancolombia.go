package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"goFarmacia/backend"
)

func BancolombiaEstadoAuth(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, b.EstadoAuth())
	}
}

func BancolombiaIniciarOAuth2(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		url, err := b.IniciarOAuth2()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"url": url})
	}
}

func BancolombiaRevocarAuth(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := b.RevocarAuth(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaGetSyncProgress(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, b.GetSyncProgress())
	}
}

func BancolombiaVerificarAhora(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		go b.VerificarAhora()
		return c.JSON(http.StatusAccepted, map[string]string{"status": "iniciado"})
	}
}

func BancolombiaObtenerTransferencias(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
		if pageSize < 1 {
			pageSize = 25
		}
		soloNoLeidas := c.QueryParam("soloNoLeidas") == "true"
		res, err := b.ObtenerTransferencias(page, pageSize, soloNoLeidas, c.QueryParam("q"), c.QueryParam("estado"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func BancolombiaMarcarLeida(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := b.MarcarLeida(c.Param("uuid")); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaMarcarTodasLeidas(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := b.MarcarTodasLeidas(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaContarNoLeidas(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]int{"count": b.ContarNoLeidas()})
	}
}

func BancolombiaEliminarTransferencia(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := b.EliminarTransferencia(c.Param("uuid")); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaEliminarTransferencias(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			UUIDs []string `json:"uuids"`
		}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := b.EliminarTransferencias(body.UUIDs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaVincularFactura(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			FacturaUUID   string `json:"facturaUUID"`
			FacturaNumero string `json:"facturaNumero"`
		}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := b.VincularFactura(c.Param("uuid"), body.FacturaUUID, body.FacturaNumero); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaDesvincularFactura(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := b.DesvincularFactura(c.Param("uuid")); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaBuscarFacturasVenta(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := b.BuscarFacturasVenta(c.QueryParam("q"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func BancolombiaGetAutoPolling(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, b.GetAutoPolling())
	}
}

func BancolombiaSetAutoPolling(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		b.SetAutoPolling(body.Enabled)
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func BancolombiaSincronizarConPeriodo(b *backend.BancolombiaService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var opts backend.BancolombiaOpcionesPeriodo
		if err := c.Bind(&opts); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		go b.SincronizarConPeriodo(opts)
		return c.JSON(http.StatusAccepted, map[string]string{"status": "iniciado"})
	}
}
