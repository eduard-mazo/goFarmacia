package handlers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"goFarmacia/backend"
)

func OutlookEstadoAuth(o *backend.OutlookService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.EstadoAuth())
	}
}

func OutlookIniciarOAuth2(o *backend.OutlookService) echo.HandlerFunc {
	return func(c echo.Context) error {
		url, err := o.IniciarOAuth2()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"url": url})
	}
}

func OutlookRevocarAuth(o *backend.OutlookService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := o.RevocarAuth(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func OutlookObtenerCredenciales(o *backend.OutlookService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.ObtenerCredenciales())
	}
}

func OutlookGuardarCredenciales(o *backend.OutlookService) echo.HandlerFunc {
	return func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "no se pudo leer el cuerpo")
		}
		if err := o.GuardarCredenciales(string(body)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func OutlookGetSyncProgress(o *backend.OutlookService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.GetOutlookSyncProgress())
	}
}

func OutlookSincronizarConOpciones(o *backend.OutlookService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var opts backend.SyncOptions
		if err := c.Bind(&opts); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		go o.SincronizarConOpciones(opts)
		return c.JSON(http.StatusAccepted, map[string]string{"status": "iniciado"})
	}
}
