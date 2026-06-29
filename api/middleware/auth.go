package middleware

import (
	"goFarmacia/backend"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// JWTAuth returns an Echo middleware that validates Bearer tokens.
// The Claims are stored in the Echo context under key "claims".
func JWTAuth(db *backend.Db) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header requerido")
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims := &backend.Claims{}
			tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				return db.JWTKey(), nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !tkn.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "Token inválido o expirado")
			}
			if claims.MFAStep == "pending" {
				return echo.NewHTTPError(http.StatusUnauthorized, "MFA pendiente de verificación")
			}
			c.Set("claims", claims)
			return next(c)
		}
	}
}

// Claims returns the JWT claims from the Echo context (set by JWTAuth middleware).
func GetClaims(c echo.Context) *backend.Claims {
	if v := c.Get("claims"); v != nil {
		if cl, ok := v.(*backend.Claims); ok {
			return cl
		}
	}
	return nil
}

// RequireRole returns middleware that allows the request only if the
// authenticated user's role matches one of the allowed roles. It MUST be
// chained after JWTAuth, since it reads the claims that JWTAuth stores on the
// Echo context. Returns 403 when the role is insufficient.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "No autenticado")
			}
			for _, r := range roles {
				if claims.Role == r {
					return next(c)
				}
			}
			return echo.NewHTTPError(http.StatusForbidden, "Permisos insuficientes para esta operación")
		}
	}
}
