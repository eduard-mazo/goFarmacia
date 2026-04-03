package handlers

import (
	"goFarmacia/backend"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SSEHandler streams server-sent events to connected browser clients.
// Replace wailsruntime.EventsEmit — services now call backend.EventBus.Emit(...).
func SSEHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
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
