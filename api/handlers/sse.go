package handlers

import (
	"goFarmacia/backend"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// SSEHandler streams server-sent events to authenticated browser clients.
// Replace wailsruntime.EventsEmit — services now call backend.EventBus.Emit(...).
//
// EventSource cannot send an Authorization header, so the JWT is accepted via
// the "token" query parameter (falling back to the header for non-browser
// clients). The stream can carry operational logs and sync detail, so it must
// not be readable without a valid, non-MFA-pending token.
func SSEHandler(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenStr := c.QueryParam("token")
		if tokenStr == "" {
			tokenStr = strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
		}
		claims := &backend.Claims{}
		tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			return db.JWTKey(), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !tkn.Valid || claims.MFAStep == "pending" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Token inválido o expirado")
		}

		w := c.Response()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
		w.WriteHeader(http.StatusOK)

		ch := backend.EventBus.Subscribe()
		defer backend.EventBus.Unsubscribe(ch)

		flusher, ok := w.Writer.(http.Flusher)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "streaming not supported")
		}

		// Send an initial ping so the client knows the connection is alive.
		_, _ = w.Write([]byte("data: {\"event\":\"connected\"}\n\n"))
		flusher.Flush()

		ctx := c.Request().Context()
		for {
			select {
			case msg, open := <-ch:
				if !open {
					return nil
				}
				_, _ = w.Write(msg)
				flusher.Flush()
			case <-ctx.Done():
				return nil
			}
		}
	}
}
