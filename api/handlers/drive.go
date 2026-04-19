package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"goFarmacia/backend"
)

func DriveEstadoAuth(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, s.EstadoAuthDrive())
	}
}

func DriveIniciarOAuth2(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		url, err := s.IniciarOAuth2Drive()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"url": url})
	}
}

func DriveRevocarAuth(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := s.RevocarAuthDrive(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func DriveGetAutoBackup(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, s.GetAutoBackup())
	}
}

func DriveSetAutoBackup(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		s.SetAutoBackup(body.Enabled)
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func DriveEjecutarBackupAhora(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := s.EjecutarBackupAhora()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func DriveListarBackups(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := s.ListarBackups()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func DriveEliminarBackup(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := s.EliminarBackup(c.Param("fileID")); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}

func DriveRestaurarBackup(s *backend.DriveBackupService) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := s.RestaurarBackup(c.Param("fileID")); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}
