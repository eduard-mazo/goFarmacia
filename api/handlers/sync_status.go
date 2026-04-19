package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"goFarmacia/backend"
)

// SyncStatusRow is one entry in the unified sync-status payload. Shapes fields
// common to Drive/Gmail/Outlook/Bancolombia so the Settings page can render a
// single list without per-provider branching.
type SyncStatusRow struct {
	ID            string `json:"id"`    // "drive" | "gmail" | "outlook" | "bancolombia"
	Label         string `json:"label"` // human-friendly name
	Authenticated bool   `json:"authenticated"`
	CredPresent   bool   `json:"credPresent"`
	Enabled       bool   `json:"enabled"`
	Running       bool   `json:"running"`
	NextRun       string `json:"nextRun"` // RFC3339 or ""
	LastRun       string `json:"lastRun"` // RFC3339 or ""
}

// SyncStatus returns the combined state of every background sync daemon.
// Used by the Configuración page to render the "Sincronizaciones automáticas" card.
func SyncStatus(
	gmail *backend.GmailService,
	outlook *backend.OutlookService,
	drive *backend.DriveBackupService,
	bancolombia *backend.BancolombiaService,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows := make([]SyncStatusRow, 0, 4)

		// Drive backup
		dAuth := drive.EstadoAuthDrive()
		dAuto := drive.GetAutoBackup()
		rows = append(rows, SyncStatusRow{
			ID:            "drive",
			Label:         "Backup a Google Drive",
			Authenticated: dAuth.Authenticated,
			CredPresent:   dAuth.CredPresent,
			Enabled:       dAuto.Enabled,
			Running:       dAuto.Enabled && dAuth.Authenticated,
			NextRun:       dAuto.NextBackup,
			LastRun:       dAuto.LastBackup,
		})

		// Gmail facturas
		gAuto := gmail.GetAutoSync()
		rows = append(rows, SyncStatusRow{
			ID:            "gmail",
			Label:         "Facturas — Gmail",
			Authenticated: gAuto.Authenticated,
			CredPresent:   gmail.EstadoAuth().CredPresent,
			Enabled:       gAuto.Enabled,
			Running:       gAuto.Running,
			NextRun:       gAuto.NextSync,
			LastRun:       gAuto.LastSync,
		})

		// Outlook facturas
		oAuto := outlook.GetAutoSync()
		rows = append(rows, SyncStatusRow{
			ID:            "outlook",
			Label:         "Facturas — Microsoft Outlook",
			Authenticated: oAuto.Authenticated,
			CredPresent:   outlook.EstadoAuth().CredPresent,
			Enabled:       oAuto.Enabled,
			Running:       oAuto.Running,
			NextRun:       oAuto.NextSync,
			LastRun:       oAuto.LastSync,
		})

		// Bancolombia transferencias
		bAuth := bancolombia.EstadoAuth()
		bAuto := bancolombia.GetAutoPolling()
		rows = append(rows, SyncStatusRow{
			ID:            "bancolombia",
			Label:         "Transferencias — Bancolombia",
			Authenticated: bAuth.Authenticated,
			CredPresent:   bAuth.CredPresent,
			Enabled:       bAuto.Enabled,
			Running:       bAuto.Enabled && bAuth.Authenticated,
		})

		return c.JSON(http.StatusOK, rows)
	}
}

// SetSyncEnabled toggles the auto-flag on the provider named by :id.
func SetSyncEnabled(
	gmail *backend.GmailService,
	outlook *backend.OutlookService,
	drive *backend.DriveBackupService,
	bancolombia *backend.BancolombiaService,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		switch id {
		case "drive":
			drive.SetAutoBackup(body.Enabled)
		case "gmail":
			gmail.SetAutoSync(body.Enabled)
		case "outlook":
			outlook.SetAutoSync(body.Enabled)
		case "bancolombia":
			bancolombia.SetAutoPolling(body.Enabled)
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "id desconocido: "+id)
		}
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
}
