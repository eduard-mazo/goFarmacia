package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"goFarmacia/api/handlers"
	apimw "goFarmacia/api/middleware"
	"goFarmacia/backend"
)

func NewRouter(
	db *backend.Db,
	gmail *backend.GmailService,
	bancolombia *backend.BancolombiaService,
	drive *backend.DriveBackupService,
	outlook *backend.OutlookService,
	assets embed.FS,
) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	e.Use(echomw.Recover())
	e.Use(echomw.CORS())
	e.Use(echomw.GzipWithConfig(echomw.GzipConfig{Level: 5}))

	api := e.Group("/api")

	// ── public ──────────────────────────────────────────────────────────────
	pub := api.Group("")

	// auth
	pub.POST("/auth/login", handlers.Login(db))
	pub.POST("/auth/verify-mfa", handlers.VerifyMFA(db))
	pub.POST("/auth/register", handlers.Register(db))

	// db setup (pre-login)
	pub.GET("/config/db-status", handlers.GetDBStatus(db))
	pub.GET("/config/setup-mode", handlers.IsSetupMode(db))
	pub.POST("/config/configurar-db", handlers.ConfigurarDB(db))
	pub.POST("/config/test-connection", handlers.TestDBConnection(db))

	// SSE (authenticated via query param token handled in handler, or open for simplicity)
	pub.GET("/events", handlers.SSEHandler())

	// ── protected ───────────────────────────────────────────────────────────
	priv := api.Group("", apimw.JWTAuth(db))

	// auth extras
	priv.POST("/auth/setup-mfa", handlers.SetupMFA(db))
	priv.POST("/auth/enable-mfa", handlers.EnableMFA(db))

	// productos
	priv.GET("/productos", handlers.ObtenerProductosPaginado(db))
	priv.GET("/productos/:uuid", handlers.ObtenerProductoPorUUID(db))
	priv.POST("/productos", handlers.RegistrarProducto(db))
	priv.PUT("/productos/:uuid", handlers.ActualizarProducto(db))
	priv.DELETE("/productos/:uuid", handlers.EliminarProducto(db))
	priv.GET("/productos/:uuid/historial-stock", handlers.ObtenerHistorialStock(db))
	priv.PUT("/productos/stock/masivo", handlers.ActualizarStockMasivo(db))

	// clientes
	priv.GET("/clientes", handlers.ObtenerClientesPaginado(db))
	priv.POST("/clientes", handlers.RegistrarCliente(db))
	priv.PUT("/clientes/:uuid", handlers.ActualizarCliente(db))
	priv.DELETE("/clientes/:uuid", handlers.EliminarCliente(db))

	// vendedores
	priv.GET("/vendedores", handlers.ObtenerVendedoresPaginado(db))
	priv.PUT("/vendedores/:uuid", handlers.ActualizarVendedor(db))
	priv.PUT("/vendedores/:uuid/perfil", handlers.ActualizarPerfilVendedor(db))
	priv.DELETE("/vendedores/:uuid", handlers.EliminarVendedor(db))

	// proveedores
	priv.GET("/proveedores", handlers.ObtenerProveedoresPaginado(db))
	priv.GET("/proveedores/estadisticas", handlers.ObtenerProveedoresConEstadisticas(db))
	priv.POST("/proveedores", handlers.CrearProveedor(db))
	priv.PUT("/proveedores/:uuid", handlers.ActualizarProveedor(db))
	priv.DELETE("/proveedores/:uuid", handlers.EliminarProveedor(db))
	priv.GET("/proveedores/resumen-compras", handlers.ObtenerResumenCompras(db))
	priv.GET("/proveedores/:nit/top-productos", handlers.ObtenerTopProductosDeProveedor(db))
	priv.POST("/proveedores/sincronizar", handlers.SincronizarProveedoresDesdeFacturas(db))

	// facturas venta
	priv.GET("/facturas", handlers.ObtenerFacturasPaginado(db))
	priv.GET("/facturas/:uuid", handlers.ObtenerDetalleFactura(db))
	priv.POST("/facturas/venta", handlers.RegistrarVenta(db))

	// facturas compra
	priv.GET("/facturas-compra", handlers.ObtenerFacturasCompraPaginado(db))
	priv.GET("/facturas-compra/:uuid", handlers.ObtenerDetalleFacturaCompra(db))
	priv.PUT("/facturas-compra/:uuid/estado", handlers.ActualizarEstadoFacturaCompra(db))

	// transferencias
	priv.GET("/transferencias", handlers.ObtenerTransferenciasPaginado(db))
	priv.GET("/transferencias/buscar-facturas", handlers.BuscarFacturasVenta(db))

	// dashboard
	priv.GET("/dashboard", handlers.ObtenerDatosDashboard(db))
	priv.GET("/dashboard/fechas-ventas", handlers.ObtenerFechasConVentas(db))
	priv.GET("/dashboard/resumen-inventario", handlers.ObtenerResumenInventario(db))
	priv.GET("/dashboard/reporte-ventas", handlers.ObtenerReporteVentasRango(db))

	// admin
	priv.GET("/admin/tablas", handlers.GetTablas(db))
	priv.GET("/admin/tablas/:name", handlers.GetDatosTabla(db))
	priv.GET("/admin/tablas/:name/esquema", handlers.GetEsquemaTabla(db))
	priv.GET("/admin/tablas/:name/export/csv", handlers.ExportarTablaCSV(db))
	priv.GET("/admin/tablas/:name/export/sql", handlers.ExportarTablaSQL(db))
	priv.GET("/admin/export/bd-sql", handlers.ExportarBDSQL(db))
	priv.POST("/admin/tablas/:name/import/csv", handlers.ImportarTablaCSV(db))
	priv.PUT("/admin/tablas/:name/rows/:pk", handlers.ActualizarFilaTabla(db))
	priv.DELETE("/admin/tablas/:name/rows/:pk", handlers.EliminarFilaTabla(db))
	priv.POST("/admin/tablas/:name/action", handlers.AdminTableAction(db))
	priv.POST("/admin/reset", handlers.ResetearTodaLaData(db))
	priv.POST("/admin/normalizar-stock", handlers.NormalizarStock(db))
	priv.GET("/admin/settings/:key", handlers.GetSetting(db))
	priv.PUT("/admin/settings/:key", handlers.SetSetting(db))

	// gmail
	priv.GET("/gmail/auth", handlers.GmailEstadoAuth(gmail))
	priv.POST("/gmail/auth/iniciar", handlers.GmailIniciarOAuth2(gmail))
	priv.DELETE("/gmail/auth", handlers.GmailRevocarAuth(gmail))
	priv.GET("/gmail/config-dir", handlers.GmailConfigDir(gmail))
	priv.GET("/gmail/credenciales", handlers.GmailObtenerCredenciales(gmail))
	priv.PUT("/gmail/credenciales", handlers.GmailGuardarCredenciales(gmail))
	priv.GET("/gmail/sync/progress", handlers.GmailGetSyncProgress(gmail))
	priv.POST("/gmail/sync", handlers.GmailSincronizarFacturas(gmail))
	priv.POST("/gmail/sync/opciones", handlers.GmailSincronizarConOpciones(gmail))
	priv.GET("/gmail/facturas-compra", handlers.GmailObtenerFacturasCompra(gmail))
	priv.GET("/gmail/facturas-compra/:uuid", handlers.GmailObtenerDetalleFacturaCompra(gmail))
	priv.PUT("/gmail/facturas-compra/:uuid/estado", handlers.GmailActualizarEstadoFacturaCompra(gmail))
	priv.GET("/gmail/proveedores/estadisticas", handlers.GmailObtenerProveedoresConEstadisticas(gmail))
	priv.GET("/gmail/proveedores/:nit/top-productos", handlers.GmailObtenerTopProductosDeProveedor(gmail))
	priv.GET("/gmail/proveedores/resumen-compras", handlers.GmailObtenerResumenCompras(gmail))
	priv.POST("/gmail/proveedores/sincronizar", handlers.GmailSincronizarProveedoresDesdeFacturas(gmail))
	priv.GET("/gmail/auto-sync", handlers.GmailGetAutoSync(gmail))
	priv.PUT("/gmail/auto-sync", handlers.GmailSetAutoSync(gmail))

	// outlook (Microsoft Graph)
	priv.GET("/outlook/auth", handlers.OutlookEstadoAuth(outlook))
	priv.POST("/outlook/auth/iniciar", handlers.OutlookIniciarOAuth2(outlook))
	priv.DELETE("/outlook/auth", handlers.OutlookRevocarAuth(outlook))
	priv.GET("/outlook/credenciales", handlers.OutlookObtenerCredenciales(outlook))
	priv.PUT("/outlook/credenciales", handlers.OutlookGuardarCredenciales(outlook))
	priv.GET("/outlook/sync/progress", handlers.OutlookGetSyncProgress(outlook))
	priv.POST("/outlook/sync/opciones", handlers.OutlookSincronizarConOpciones(outlook))
	priv.GET("/outlook/auto-sync", handlers.OutlookGetAutoSync(outlook))
	priv.PUT("/outlook/auto-sync", handlers.OutlookSetAutoSync(outlook))

	// unified sync dashboard
	priv.GET("/sync/status", handlers.SyncStatus(gmail, outlook, drive, bancolombia))
	priv.PUT("/sync/:id/enabled", handlers.SetSyncEnabled(gmail, outlook, drive, bancolombia))

	// bancolombia
	priv.GET("/bancolombia/auth", handlers.BancolombiaEstadoAuth(bancolombia))
	priv.POST("/bancolombia/auth/iniciar", handlers.BancolombiaIniciarOAuth2(bancolombia))
	priv.DELETE("/bancolombia/auth", handlers.BancolombiaRevocarAuth(bancolombia))
	priv.GET("/bancolombia/sync/progress", handlers.BancolombiaGetSyncProgress(bancolombia))
	priv.POST("/bancolombia/sync", handlers.BancolombiaVerificarAhora(bancolombia))
	priv.POST("/bancolombia/sync/periodo", handlers.BancolombiaSincronizarConPeriodo(bancolombia))
	priv.GET("/bancolombia/transferencias", handlers.BancolombiaObtenerTransferencias(bancolombia))
	priv.PUT("/bancolombia/transferencias/:uuid/leida", handlers.BancolombiaMarcarLeida(bancolombia))
	priv.PUT("/bancolombia/transferencias/todas-leidas", handlers.BancolombiaMarcarTodasLeidas(bancolombia))
	priv.GET("/bancolombia/transferencias/no-leidas", handlers.BancolombiaContarNoLeidas(bancolombia))
	priv.DELETE("/bancolombia/transferencias/:uuid", handlers.BancolombiaEliminarTransferencia(bancolombia))
	priv.DELETE("/bancolombia/transferencias", handlers.BancolombiaEliminarTransferencias(bancolombia))
	priv.PUT("/bancolombia/transferencias/:uuid/vincular", handlers.BancolombiaVincularFactura(bancolombia))
	priv.PUT("/bancolombia/transferencias/:uuid/desvincular", handlers.BancolombiaDesvincularFactura(bancolombia))
	priv.GET("/bancolombia/buscar-facturas", handlers.BancolombiaBuscarFacturasVenta(bancolombia))
	priv.GET("/bancolombia/auto-polling", handlers.BancolombiaGetAutoPolling(bancolombia))
	priv.PUT("/bancolombia/auto-polling", handlers.BancolombiaSetAutoPolling(bancolombia))

	// pos (thermal printer — server-side USB access)
	priv.GET("/pos/verificar", handlers.VerificarImpresora(db))
	priv.POST("/pos/imprimir", handlers.ImprimirRecibo(db))

	// drive
	priv.GET("/drive/auth", handlers.DriveEstadoAuth(drive))
	priv.POST("/drive/auth/iniciar", handlers.DriveIniciarOAuth2(drive))
	priv.DELETE("/drive/auth", handlers.DriveRevocarAuth(drive))
	priv.GET("/drive/auto-backup", handlers.DriveGetAutoBackup(drive))
	priv.PUT("/drive/auto-backup", handlers.DriveSetAutoBackup(drive))
	priv.POST("/drive/backup", handlers.DriveEjecutarBackupAhora(drive))
	priv.GET("/drive/backups", handlers.DriveListarBackups(drive))
	priv.DELETE("/drive/backups/:fileID", handlers.DriveEliminarBackup(drive))
	priv.POST("/drive/backups/:fileID/restore", handlers.DriveRestaurarBackup(drive))

	// static frontend — embedded SPA (must be last)
	distFS, _ := fs.Sub(assets, "frontend/dist")
	fileServer := http.FileServer(http.FS(distFS))
	// Real static assets (hashed filenames in /assets/) served directly
	e.GET("/assets/*", echo.WrapHandler(fileServer))
	e.GET("/favicon.ico", echo.WrapHandler(fileServer))
	// All other paths → index.html (Vue Router handles client-side routing)
	e.GET("/*", func(c echo.Context) error {
		idx, err := assets.ReadFile("frontend/dist/index.html")
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "frontend not built")
		}
		return c.HTMLBlob(http.StatusOK, idx)
	})

	return e
}
