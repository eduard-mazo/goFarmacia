package handlers

import (
	"goFarmacia/backend"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func intQ(c echo.Context, key string, def int) int {
	v := c.QueryParam(key)
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}

func boolQ(c echo.Context, key string) bool {
	return c.QueryParam(key) == "true"
}

// ─── Productos ───────────────────────────────────────────────────────────────

func ObtenerProductosPaginado(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerProductosPaginado(
			intQ(c, "page", 1), intQ(c, "pageSize", 25),
			c.QueryParam("q"), c.QueryParam("sortBy"), c.QueryParam("sortOrder"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerProductoPorUUID(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerProductoPorUUID(c.Param("uuid"))
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func RegistrarProducto(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.NuevoProducto
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		res, err := db.RegistrarProducto(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusCreated, res)
	}
}

func ActualizarProducto(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.ProductoAjusteRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		msg, err := db.ActualizarProducto(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

func EliminarProducto(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := db.EliminarProducto(c.Param("uuid")); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.NoContent(http.StatusNoContent)
	}
}

func ObtenerHistorialStock(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerHistorialStock(c.Param("uuid"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ActualizarStockMasivo(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req []backend.AjusteStockRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		msg, err := db.ActualizarStockMasivo(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

// ─── Clientes ────────────────────────────────────────────────────────────────

func ObtenerClientesPaginado(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerClientesPaginado(
			intQ(c, "page", 1), intQ(c, "pageSize", 25),
			c.QueryParam("q"), c.QueryParam("sortBy"), c.QueryParam("sortOrder"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func RegistrarCliente(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.Cliente
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		res, err := db.RegistrarCliente(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusCreated, res)
	}
}

func ActualizarCliente(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.Cliente
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		msg, err := db.ActualizarCliente(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

func EliminarCliente(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		msg, err := db.EliminarCliente(c.Param("uuid"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

// ─── Vendedores ──────────────────────────────────────────────────────────────

func ObtenerVendedoresPaginado(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerVendedoresPaginado(
			intQ(c, "page", 1), intQ(c, "pageSize", 25),
			c.QueryParam("q"), c.QueryParam("sortBy"), c.QueryParam("sortOrder"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ActualizarVendedor(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.Vendedor
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		res, err := db.ActualizarVendedor(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ActualizarPerfilVendedor(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.VendedorUpdateRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		msg, err := db.ActualizarPerfilVendedor(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

func EliminarVendedor(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		msg, err := db.EliminarVendedor(c.Param("uuid"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

// ─── Proveedores ─────────────────────────────────────────────────────────────

func ObtenerProveedoresPaginado(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerProveedoresPaginado(
			intQ(c, "page", 1), intQ(c, "pageSize", 25), c.QueryParam("q"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerProveedoresConEstadisticas(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerProveedoresConEstadisticas(
			intQ(c, "page", 1), intQ(c, "pageSize", 25), c.QueryParam("q"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func CrearProveedor(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.Proveedor
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := db.CrearProveedor(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusCreated, req)
	}
}

func ActualizarProveedor(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.Proveedor
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := db.ActualizarProveedor(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, req)
	}
}

func EliminarProveedor(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := db.EliminarProveedor(c.Param("uuid")); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.NoContent(http.StatusNoContent)
	}
}

func ObtenerResumenCompras(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerResumenCompras(c.QueryParam("desde"), c.QueryParam("hasta"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerTopProductosDeProveedor(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerTopProductosDeProveedor(c.Param("nit"), intQ(c, "limit", 10))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func SincronizarProveedoresDesdeFacturas(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		n, err := db.SincronizarProveedoresDesdeFacturas()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"sincronizados": n})
	}
}

// ─── Facturas (ventas) ───────────────────────────────────────────────────────

func ObtenerFacturasPaginado(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerFacturasPaginado(
			intQ(c, "page", 1), intQ(c, "pageSize", 25),
			c.QueryParam("q"), c.QueryParam("sortBy"), c.QueryParam("sortOrder"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerDetalleFactura(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerDetalleFactura(c.Param("uuid"))
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func RegistrarVenta(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req backend.VentaRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		res, err := db.RegistrarVenta(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusCreated, res)
	}
}

// ─── Facturas Compra ─────────────────────────────────────────────────────────

func ObtenerFacturasCompraPaginado(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerFacturasCompraPaginado(
			intQ(c, "page", 1), intQ(c, "pageSize", 25),
			c.QueryParam("q"), c.QueryParam("sortBy"), c.QueryParam("sortDir"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerDetalleFacturaCompra(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerDetalleFacturaCompra(c.Param("uuid"))
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ActualizarEstadoFacturaCompra(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct{ Estado string }
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := db.ActualizarEstadoFacturaCompra(c.Param("uuid"), req.Estado); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.NoContent(http.StatusNoContent)
	}
}

// ─── Transferencias Bancolombia (Db methods) ──────────────────────────────────

func ObtenerTransferenciasPaginado(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerTransferenciasPaginado(
			intQ(c, "page", 1), intQ(c, "pageSize", 25),
			boolQ(c, "soloNoLeidas"),
			c.QueryParam("busqueda"), c.QueryParam("estado"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func BuscarFacturasVenta(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.BuscarFacturasVenta(c.QueryParam("q"), intQ(c, "limit", 20))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

func ObtenerDatosDashboard(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerDatosDashboard(c.QueryParam("fecha"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerFechasConVentas(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerFechasConVentas()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerResumenInventario(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerResumenInventario()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ObtenerReporteVentasRango(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ObtenerReporteVentasRango(c.QueryParam("desde"), c.QueryParam("hasta"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

// ─── Admin ───────────────────────────────────────────────────────────────────

func GetTablas(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.GetTablas()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func GetDatosTabla(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.GetDatosTabla(
			c.Param("name"),
			intQ(c, "limit", 100), intQ(c, "offset", 0),
			c.QueryParam("sortCol"), c.QueryParam("sortDir"),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func GetEsquemaTabla(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.GetEsquemaTabla(c.Param("name"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ExportarTablaCSV(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ExportarTablaCSV(c.Param("name"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if res.FilePath == "" {
			return c.JSON(http.StatusOK, res)
		}
		defer os.Remove(res.FilePath)
		return c.Attachment(res.FilePath, c.Param("name")+".csv")
	}
}

func ExportarTablaSQL(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ExportarTablaSQL(c.Param("name"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if res.FilePath == "" {
			return c.JSON(http.StatusOK, res)
		}
		defer os.Remove(res.FilePath)
		return c.Attachment(res.FilePath, c.Param("name")+".sql")
	}
}

func ExportarBDSQL(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := db.ExportarBDSQL()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if res.FilePath == "" {
			return c.JSON(http.StatusOK, res)
		}
		defer os.Remove(res.FilePath)
		return c.Attachment(res.FilePath, "backup.sql")
	}
}

func ImportarTablaCSV(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "archivo requerido")
		}
		src, err := file.Open()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		defer src.Close()

		tmp, err := os.CreateTemp("", "import-*.csv")
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		defer os.Remove(tmp.Name())
		if _, err := io.Copy(tmp, src); err != nil {
			tmp.Close()
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		tmp.Close()

		progressCh, errCh := db.CargarDesdeCSV(tmp.Name(), c.Param("name"))
		var msgs []string
		for msg := range progressCh {
			msgs = append(msgs, msg)
		}
		if err := <-errCh; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"messages": msgs})
	}
}

func ActualizarFilaTabla(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			TableName    string
			PKColumn     string
			PKValue      string
			UpdateColumn string
			NewValue     string
			IsNull       bool
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		res, err := db.ActualizarFilaTabla(req.TableName, req.PKColumn, req.PKValue, req.UpdateColumn, req.NewValue, req.IsNull)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func EliminarFilaTabla(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			TableName string
			PKColumn  string
			PKValue   string
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		res, err := db.EliminarFilaTabla(req.TableName, req.PKColumn, req.PKValue)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func AdminTableAction(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")
		action := c.Param("action")
		var res backend.OperationResult
		var err error
		switch action {
		case "analizar":
			res, err = db.AnalizarTabla(name)
		case "vacuum":
			res, err = db.VacuumTabla(name)
		case "truncar":
			res, err = db.TruncarTabla(name)
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "acción desconocida: "+action)
		}
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	}
}

func ResetearTodaLaData(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		msg, err := db.ResetearTodaLaData()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

func NormalizarStock(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		msg, err := db.NormalizarStockTodosLosProductos()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"message": msg})
	}
}

func GetSetting(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		val, err := db.GetSetting(c.Param("key"))
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return c.JSON(http.StatusOK, echo.Map{"value": val})
	}
}

func SetSetting(db *backend.Db) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Key   string
			Value string
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err := db.SetSetting(req.Key, req.Value); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.NoContent(http.StatusNoContent)
	}
}
