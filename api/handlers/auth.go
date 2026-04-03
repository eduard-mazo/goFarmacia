package handlers

import (
	"goFarmacia/backend"
	"net/http"

	"github.com/labstack/echo/v4"
)

func Login(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.LoginRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		resp, err := db.LoginVendedor(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func VerifyMFA(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Token string `json:"token"`
			Code  string `json:"code"`
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		resp, err := db.VerificarLoginMFA(req.Token, req.Code)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func Register(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var v backend.Vendedor
		if err := c.Bind(&v); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		result, err := db.RegistrarVendedor(v)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusCreated, result)
	}
}

func SetupMFA(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct{ Email string }
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		resp, err := db.GenerarMFA(req.Email)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func EnableMFA(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Email string
			Code  string
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		ok, err := db.HabilitarMFA(req.Email, req.Code)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"enabled": ok})
	}
}

func GetDBStatus(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, db.GetDBStatus())
	}
}

func IsSetupMode(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{"setupMode": db.IsSetupMode()})
	}
}

func ConfigurarDB(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct{ DSN string }
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := db.ConfigurarDB(req.DSN); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"ok": true})
	}
}

func TestDBConnection(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct{ DSN string }
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := db.TestDBConnection(req.DSN); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"ok": true})
	}
}
