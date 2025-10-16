package backend

import (
	"fmt"
	"time"

	"github.com/google/gousb"
	"golang.org/x/text/encoding/charmap"
)

const (
	vendorID  gousb.ID = 0x0416
	productID gousb.ID = 0x5011
	// Code Page 16: PC858 (Euro) - Común para Español y Euro.
	// El comando es ESC t n, donde n=16.
	// \x1B\x74\x10 es el comando de inicialización de la función ImprimirRecibo
	// CODE_PAGE_PC858_N byte = 0x10 // n=16
	CODE_PAGE_PC858_N byte = 0x02 // n=2
)

// Encoder para el Code Page 858 (Latin-1 / Euro)
var encoder = charmap.CodePage858.NewEncoder()

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

// Función de ayuda para codificar texto.
func encodeText(text string) ([]byte, error) {
	// Codifica la cadena de UTF-8 al Code Page 858.
	// Esto resuelve problemas con tildes (á, é, í, ó, ú, ñ)
	return encoder.Bytes([]byte(text))
}

// NUEVA FUNCIÓN: Envía el comando para seleccionar el Code Page
func selectCodePage(n byte) []byte {
	// Comando: ESC t n
	return []byte{0x1B, 0x74, n}
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

	if err := send(selectCodePage(CODE_PAGE_PC858_N)); err != nil {
		return err
	}

	sendEncoded := func(text string) error {
		data, err := encodeText(text)
		if err != nil {
			return fmt.Errorf("error al codificar texto '%s': %w", text, err)
		}
		if _, err := epOut.Write(data); err != nil {
			return fmt.Errorf("error al escribir datos codificados: %w", err)
		}
		return nil
	}
	if err := send(center()); err != nil {
		return err
	}
	if err := sendEncoded("DROGUERIA LUNA"); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(center()); err != nil {
		return err
	}
	if err := sendEncoded("NIT: 70.120.237"); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(center()); err != nil {
		return err
	}
	if err := sendEncoded("Medellín, Antioquia"); err != nil {
		return err
	} // Nótese la tilde en "Medellín"
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(center()); err != nil {
		return err
	}
	if err := sendEncoded("Cel: 3054456781"); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(center()); err != nil {
		return err
	}
	if err := sendEncoded("Calle 94 # 48-33"); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(left()); err != nil {
		return err
	}
	if err := sendEncoded(fmt.Sprintf("Factura: %s", factura.NumeroFactura)); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	fecha, _ := time.Parse(time.RFC3339, factura.FechaEmision.Format(time.RFC3339))
	if err := send(left()); err != nil {
		return err
	}
	if err := sendEncoded(fmt.Sprintf("Fecha: %s", fecha.Format("02/01/2006 03:04 PM"))); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(left()); err != nil {
		return err
	}
	if err := sendEncoded(fmt.Sprintf("Cliente: %s %s", factura.Cliente.Nombre, factura.Cliente.Apellido)); err != nil {
		return err
	}
	if err := sendEncoded(fmt.Sprintf("Documento: %s", factura.Cliente.NumeroID)); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(left()); err != nil {
		return err
	}
	if err := sendEncoded(fmt.Sprintf("Vendedor: %s", factura.Vendedor.Nombre)); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	if err := send(lineBreak()); err != nil {
		return err
	}

	// Encabezado de la tabla de productos
	if err := send(left()); err != nil {
		return err
	}
	if err := sendEncoded("Cant | Producto      |    Total"); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	// 4. Corrección de la línea separadora: Ahora usa `sendEncoded`
	if err := send(left()); err != nil {
		return err
	}
	if err := send(boldOn()); err != nil {
		return err
	}
	if err := sendEncoded("- - - - - - - - - - - - - - - -"); err != nil {
		return err
	}
	if err := send(boldOff()); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	// Detalles
	for _, item := range factura.Detalles {
		linea := formatItemLine(item.Cantidad, item.Producto.Nombre, item.PrecioTotal)
		if err := send(left()); err != nil {
			return err
		}
		if err := sendEncoded(linea); err != nil {
			return err
		}
		if err := send(lineBreak()); err != nil {
			return err
		} // El salto de línea se quita de formatItemLine y se pone aquí.
	}

	// Línea separadora final
	if err := send(left()); err != nil {
		return err
	}
	if err := send(boldOn()); err != nil {
		return err
	}
	if err := sendEncoded("- - - - - - - - - - - - - - - -"); err != nil {
		return err
	}
	if err := send(boldOff()); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	// Total:
	totalStr := fmt.Sprintf("TOTAL: %s", formatCurrency(factura.Total))
	if err := send(boldOn()); err != nil {
		return err
	}
	if err := send(doubleHeightOn()); err != nil {
		return err
	}

	if err := send(right()); err != nil {
		return err
	}
	if err := sendEncoded(totalStr); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
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

	// Mensaje final:
	// El caracter especial al inicio probablemente era un error de codificación de la '¡' o de un caracter invisible.
	// Al usar `sendEncoded` y el nuevo `center()`, esto debería corregirse.
	if err := send(center()); err != nil {
		return err
	}
	if err := sendEncoded("¡Gracias por su compra!"); err != nil {
		return err
	}
	if err := send(lineBreak()); err != nil {
		return err
	}

	// Salto de papel y corte
	if err := send([]byte("\n\n\n")); err != nil {
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

	line := fmt.Sprintf("%-4s%-20s%8s", qtyStr, name, totalStr)
	return line
}

func formatCurrency(val float64) string {
	return fmt.Sprintf("$%d", int(val))
}
func center() []byte          { return []byte("\x1B\x61\x01") }
func left() []byte            { return []byte("\x1B\x61\x00") }
func right() []byte           { return []byte("\x1B\x61\x02") }
func lineBreak() []byte       { return []byte("\n") }
func boldOn() []byte          { return []byte("\x1B\x45\x01") }
func boldOff() []byte         { return []byte("\x1B\x45\x00") }
func doubleHeightOn() []byte  { return []byte("\x1D\x21\x01") }
func doubleHeightOff() []byte { return []byte("\x1D\x21\x00") }
