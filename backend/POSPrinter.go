package backend

import (
	"fmt"
	"time"

	"github.com/google/gousb"
)

const (
	vendorID  gousb.ID = 0x0416
	productID gousb.ID = 0x5011
)

func (d *Db) VerificarImpresora() bool {
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(vendorID, productID)
	if err != nil || dev == nil {
		d.Log.Warnf("Verificación de impresora fallida: No se encontró el dispositivo. Error: %v", err)
		return false
	}
	defer dev.Close()
	d.Log.Info("Verificación de impresora exitosa: Dispositivo encontrado.")
	return true
}

func (d *Db) ImprimirRecibo(factura Factura) error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(vendorID, productID)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el dispositivo: %w", err)
	}
	if dev == nil {
		return fmt.Errorf("impresora POS58 no encontrada")
	}
	defer dev.Close()

	epOut, close, err := setupEndpoint(dev)
	if err != nil {
		return err
	}
	defer close()

	send := func(data []byte) error {
		if _, err := epOut.Write(data); err != nil {
			return fmt.Errorf("error al escribir en la impresora: %w", err)
		}
		return nil
	}

	if err := send([]byte("\x1B@")); err != nil {
		return err
	}
	if err := send([]byte("\x1B\x74\x10")); err != nil {
		return err
	}
	if err := send(center("DROGUERIA LUNA")); err != nil {
		return err
	}
	if err := send(center("NIT: 70.120.237")); err != nil {
		return err
	}
	if err := send(center("Medellin, Antioquia")); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(left(fmt.Sprintf("Factura: %s", factura.NumeroFactura))); err != nil {
		return err
	}
	fecha, _ := time.Parse(time.RFC3339, factura.FechaEmision.Format(time.RFC3339))
	if err := send(left(fmt.Sprintf("Fecha: %s", fecha.Format("02/01/2006 03:04 PM")))); err != nil {
		return err
	}
	if err := send(left(fmt.Sprintf("Cliente: %s %s", factura.Cliente.NumeroID, factura.Cliente.Apellido))); err != nil {
		return err
	}
	if err := send(left(fmt.Sprintf("Vendedor: %s", factura.Vendedor.Nombre))); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send([]byte("Cant | Producto   |   Total\n")); err != nil {
		return err
	}
	if err := send([]byte("--------------------------------\n")); err != nil {
		return err
	}
	for _, item := range factura.Detalles {
		linea := formatItemLine(item.Cantidad, item.Producto.Nombre, item.PrecioTotal)
		if err := send([]byte(linea)); err != nil {
			return err
		}
	}
	if err := send([]byte("--------------------------------\n")); err != nil {
		return err
	}

	totalStr := fmt.Sprintf("TOTAL: %s", formatCurrency(factura.Total))
	if err := send(boldOn()); err != nil {
		return err
	}
	if err := send(doubleHeightOn()); err != nil {
		return err
	}
	if err := send(right(totalStr)); err != nil {
		return err
	}
	if err := send(doubleHeightOff()); err != nil {
		return err
	}
	if err := send(boldOff()); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(center("Gracias por su compra")); err != nil {
		return err
	}

	if err := send([]byte("\n\n\n\n")); err != nil {
		return err
	}
	if err := send([]byte("\x1D\x56\x42\x00")); err != nil {
		return err
	}

	d.Log.Info("Recibo enviado a la impresora correctamente.")
	return nil
}

func setupEndpoint(dev *gousb.Device) (*gousb.OutEndpoint, func(), error) {
	cfg, err := dev.Config(1)
	if err != nil {
		return nil, nil, fmt.Errorf("error al obtener configuración: %w", err)
	}

	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		return nil, nil, fmt.Errorf("error al abrir interfaz: %w", err)
	}

	var epOut *gousb.OutEndpoint
	for _, ep := range intf.Setting.Endpoints {
		if ep.Address&0x80 == 0 {
			epOut, err = intf.OutEndpoint(int(ep.Address))
			if err != nil {
				intf.Close()
				cfg.Close()
				return nil, nil, fmt.Errorf("error al abrir endpoint: %w", err)
			}
			break
		}

	}

	if epOut == nil {
		intf.Close()
		cfg.Close()
		return nil, nil, fmt.Errorf("no se encontró endpoint OUT")
	}

	closeFunc := func() {
		intf.Close()
		cfg.Close()
	}

	return epOut, closeFunc, nil
}

func formatItemLine(qty int, name string, total float64) string {
	qtyStr := fmt.Sprintf("%d", qty)
	// Trunca el nombre del producto si es muy largo
	maxNameLen := 19
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}

	totalStr := formatCurrency(total)

	// Une las partes con el espaciado correcto. Ancho total: 32 chars.
	// Ej: "1   Producto largo...   $10.000"
	line := fmt.Sprintf("%-4s%-20s%8s\n", qtyStr, name, totalStr)
	return line
}

func formatCurrency(val float64) string {
	// Formateo simple para la impresora, sin decimales.
	return fmt.Sprintf("$%d", int(val))
}

// Comandos de formato
func center(text string) []byte { return []byte("\x1B\x61\x01" + text + "\n") }
func left(text string) []byte   { return []byte("\x1B\x61\x00" + text + "\n") }
func right(text string) []byte  { return []byte("\x1B\x61\x02" + text + "\n") }
func lineBreak() []byte         { return []byte("\n") }
func boldOn() []byte            { return []byte("\x1B\x45\x01") }
func boldOff() []byte           { return []byte("\x1B\x45\x00") }
func doubleHeightOn() []byte    { return []byte("\x1D\x21\x01") }
func doubleHeightOff() []byte   { return []byte("\x1D\x21\x00") }
