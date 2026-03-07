package backend

// POSPrinter.go — ESC/POS driver for 58 mm thermal receipt printers.
//
// Printer : ZJ-5890K / POS58 family. USB VID 0x0416 / PID 0x5011.
// Paper   : 58 mm. Font A = 32 cols. Font B = ~42 cols (smaller, 9x17 dots).
// Charset : PC850 (Latin-1 multilingual) — covers full Spanish alphabet.
//
// QR code: generated as a 1-bit raster bitmap and sent via GS v 0 (raster
// bit image command). The ZJ-5890K does NOT support the GS ( k QR vector
// command, so hardware QR generation is not used.
//
// Public API (Wails-bound via *Db):
//   ImprimirRecibo(factura Factura) error
//   VerificarImpresora() bool

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/google/gousb"
	"golang.org/x/text/encoding/charmap"
)

// ── Hardware constants ────────────────────────────────────────────────────────

const (
	posVendorID  gousb.ID = 0x0416
	posProductID gousb.ID = 0x5011

	// ESC t n — PC850 Multilingual (Latin-1) code page.
	cpPC850 byte = 0x02

	// Printable columns per font on 58 mm paper.
	paperColsA = 32 // Font A — normal (12×24 dots)
	paperColsB = 42 // Font B — small  (9×17 dots)

	// QR bitmap size in dots. Must be ≤ printable width in dots (~384).
	qrBitmapSize = 120 // pixels; centered on paper
)

// ── receipt — buffered ESC/POS builder ───────────────────────────────────────

type receipt struct{ buf bytes.Buffer }

func newReceipt() *receipt { return &receipt{} }

func (r *receipt) raw(b ...byte) *receipt { r.buf.Write(b); return r }
func (r *receipt) cmd(b ...byte) *receipt { return r.raw(b...) }

// text encodes UTF-8 → PC850 and appends it.
func (r *receipt) text(s string) *receipt {
	b, err := charmap.CodePage850.NewEncoder().Bytes([]byte(s))
	if err != nil {
		r.buf.WriteString(sanitizeToASCII(s))
		return r
	}
	r.buf.Write(b)
	return r
}

func (r *receipt) ln() *receipt { return r.raw('\n') }
func (r *receipt) bytes() []byte { return r.buf.Bytes() }

// ── ESC/POS command helpers ───────────────────────────────────────────────────

func cmdInit() []byte          { return []byte{0x1B, '@'} }
func cmdCodePage(n byte) []byte { return []byte{0x1B, 0x74, n} }
func cmdCenter() []byte        { return []byte{0x1B, 0x61, 0x01} }
func cmdLeft() []byte          { return []byte{0x1B, 0x61, 0x00} }
func cmdRight() []byte         { return []byte{0x1B, 0x61, 0x02} }
func cmdBoldOn() []byte        { return []byte{0x1B, 0x45, 0x01} }
func cmdBoldOff() []byte       { return []byte{0x1B, 0x45, 0x00} }
func cmdDblHOn() []byte        { return []byte{0x1D, 0x21, 0x11} } // double width + double height
func cmdDblHOff() []byte       { return []byte{0x1D, 0x21, 0x00} }
func cmdFontB() []byte         { return []byte{0x1B, 0x4D, 0x01} } // ESC M 1 — Font B (small)
func cmdFontA() []byte         { return []byte{0x1B, 0x4D, 0x00} } // ESC M 0 — Font A (normal)
func cmdCut() []byte           { return []byte{0x1D, 0x56, 0x42, 0x00} }

// ── QR as raster bitmap ───────────────────────────────────────────────────────

// qrBitmap encodes data as a QR code, scales it to qrBitmapSize×qrBitmapSize,
// and returns an ESC/POS GS v 0 raster image command byte slice.
//
// GS v 0 m xL xH yL yH [data]
//   m  = 0 (normal density)
//   xL+xH*256 = bytes per row = ceil(width/8)
//   yL+yH*256 = rows = height
//   data: 1=black, 0=white, MSB first
func qrBitmap(data string) []byte {
	// Generate QR code
	bc, err := qr.Encode(data, qr.M, qr.Auto)
	if err != nil {
		return nil
	}
	// Scale to target size
	scaled, err := barcode.Scale(bc, qrBitmapSize, qrBitmapSize)
	if err != nil {
		return nil
	}
	return imageToRaster(scaled)
}

// imageToRaster converts an image.Image to ESC/POS GS v 0 raster bytes.
func imageToRaster(img image.Image) []byte {
	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y
	bytesPerRow := (w + 7) / 8

	// Build pixel rows
	pixels := make([]byte, bytesPerRow*h)
	for y := range h {
		for x := range w {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// Luminance: dark pixel → bit = 1 (print)
			luma := (r*299 + g*587 + b*114) / 1000
			if luma < 0x8000 { // darker than 50% gray → print dot
				byteIdx := y*bytesPerRow + x/8
				bitIdx := uint(7 - x%8)
				pixels[byteIdx] |= 1 << bitIdx
			}
		}
	}

	xL := byte(bytesPerRow & 0xFF)
	xH := byte(bytesPerRow >> 8)
	yL := byte(h & 0xFF)
	yH := byte(h >> 8)

	var buf bytes.Buffer
	buf.Write([]byte{0x1D, 0x76, 0x30, 0x00, xL, xH, yL, yH})
	buf.Write(pixels)
	return buf.Bytes()
}

// ── sanitize helpers ──────────────────────────────────────────────────────────

func sanitizeToASCII(s string) string {
	return strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
		"ñ", "n", "Ñ", "N", "ü", "u", "Ü", "U",
		"¿", "?", "¡", "!", "€", "E",
	).Replace(s)
}

// ── Currency formatting ───────────────────────────────────────────────────────

func formatCOP(v float64) string {
	n := int64(math.Round(v))
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i, c := range s {
		pos := len(s) - i
		if i > 0 && pos%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, byte(c))
	}
	if neg {
		return "$-" + string(out)
	}
	return "$ " + string(out)
}

// ── Layout helpers ────────────────────────────────────────────────────────────

func separatorA() string { return strings.Repeat("-", paperColsA) }
func separatorB() string { return strings.Repeat("-", paperColsB) }

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func pad(s string, w int) string {
	l := runeLen(s)
	if l >= w {
		return s
	}
	return s + strings.Repeat(" ", w-l)
}

func rpad(s string, w int) string {
	l := runeLen(s)
	if l >= w {
		return s
	}
	return strings.Repeat(" ", w-l) + s
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-3]) + "..."
}

// itemLineB formats one product line for Font B (42 cols).
//
// Layout (42 cols):
//   col  0-3  : qty    4 chars  e.g. " 50x"
//   col  4    : space
//   col  5-26 : name  22 chars
//   col 27-41 : total 15 chars  right-aligned
//
// 4 + 1 + 22 + 15 = 42 ✓
func itemLineB(qty int, name string, total float64) string {
	qtyStr  := rpad(fmt.Sprintf("%dx", qty), 4)
	nameStr := pad(truncate(name, 22), 22)
	totStr  := rpad(formatCOP(total), 15)
	return fmt.Sprintf("%s %s%s", qtyStr, nameStr, totStr)
}

// unitLineB formats the unit-price sub-line for Font B.
func unitLineB(unitPrice float64) string {
	return fmt.Sprintf("     %s c/u", formatCOP(unitPrice))
}

// totalLineB formats a total row: label(27) + value(15) = 42.
func totalLineB(label, val string) string {
	return pad(label, 27) + rpad(val, 15)
}

// ── USB device helpers ───────────────────────────────────────────────────────

func openPrinter() (*gousb.Context, *gousb.Device, error) {
	ctx := gousb.NewContext()
	dev, err := ctx.OpenDeviceWithVIDPID(posVendorID, posProductID)
	if err != nil {
		ctx.Close()
		return nil, nil, fmt.Errorf("error al abrir dispositivo USB: %w", err)
	}
	if dev == nil {
		ctx.Close()
		return nil, nil, fmt.Errorf("impresora POS58 no encontrada (VID=%04x PID=%04x)",
			uint16(posVendorID), uint16(posProductID))
	}
	if err := dev.SetAutoDetach(true); err != nil {
		_ = err
	}
	return ctx, dev, nil
}

func getOutEndpoint(dev *gousb.Device) (*gousb.OutEndpoint, func(), error) {
	cfg, err := dev.Config(1)
	if err != nil {
		return nil, nil, fmt.Errorf("configuración USB: %w", err)
	}
	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		return nil, nil, fmt.Errorf("interfaz USB: %w", err)
	}
	var epOut *gousb.OutEndpoint
	for _, ep := range intf.Setting.Endpoints {
		if ep.Address&0x80 == 0 {
			if epOut, err = intf.OutEndpoint(int(ep.Address)); err != nil {
				intf.Close()
				cfg.Close()
				return nil, nil, fmt.Errorf("endpoint OUT: %w", err)
			}
			break
		}
	}
	if epOut == nil {
		intf.Close()
		cfg.Close()
		return nil, nil, fmt.Errorf("no se encontró endpoint OUT en la impresora")
	}
	return epOut, func() { intf.Close(); cfg.Close() }, nil
}

// ── Public API ───────────────────────────────────────────────────────────────

func (d *Db) VerificarImpresora() bool {
	ctx, dev, err := openPrinter()
	if err != nil {
		d.Log.Warnf("Impresora no encontrada: %v", err)
		return false
	}
	defer ctx.Close()
	defer dev.Close()
	d.Log.Info("Impresora POS58 detectada correctamente.")
	return true
}

func (d *Db) ImprimirRecibo(factura Factura) error {
	ctx, dev, err := openPrinter()
	if err != nil {
		return err
	}
	defer ctx.Close()
	defer dev.Close()

	epOut, closeEP, err := getOutEndpoint(dev)
	if err != nil {
		return err
	}
	defer closeEP()

	r := buildReceipt(factura)
	if _, err := epOut.Write(r.bytes()); err != nil {
		return fmt.Errorf("error al enviar recibo a la impresora: %w", err)
	}
	d.Log.Infof("Recibo %s enviado correctamente a la impresora.", factura.NumeroFactura)
	return nil
}

// ── Receipt layout ───────────────────────────────────────────────────────────

func buildReceipt(f Factura) *receipt {
	r := newReceipt()

	// ── Init ─────────────────────────────────────────────────────────────────
	r.raw(cmdInit()...).raw(cmdCodePage(cpPC850)...)

	// ── Header — Font A, center, bold, double size ────────────────────────────
	r.raw(cmdCenter()...).
		raw(cmdBoldOn()...).
		raw(cmdDblHOn()...).
		text("DROGUERIA LUNA").ln().
		raw(cmdDblHOff()...).
		raw(cmdBoldOff()...)

	r.raw(cmdCenter()...).text("NIT: 70.120.237-8").ln()
	r.raw(cmdCenter()...).text("Calle 94 # 48-33, Medellin").ln()
	r.raw(cmdCenter()...).text("Tel: 305 445 6781").ln()

	// ── Switch to Font B for body (smaller, more cols) ────────────────────────
	r.raw(cmdFontB()...)
	r.raw(cmdLeft()...).text(separatorB()).ln()

	// ── Transaction metadata ──────────────────────────────────────────────────
	fecha := f.FechaEmision.Format("02/01/2006  03:04 PM")
	r.raw(cmdLeft()...).
		text(fmt.Sprintf("Factura : %s", f.NumeroFactura)).ln().
		text(fmt.Sprintf("Fecha   : %s", fecha)).ln().
		text(fmt.Sprintf("Cliente : %s %s", f.Cliente.Nombre, f.Cliente.Apellido)).ln().
		text(fmt.Sprintf("C.C./ID : %s", f.Cliente.NumeroID)).ln().
		text(fmt.Sprintf("Vendedor: %s %s", f.Vendedor.Nombre, f.Vendedor.Apellido)).ln()
	if f.MetodoPago != "" {
		r.text(fmt.Sprintf("Pago    : %s", f.MetodoPago)).ln()
	}

	r.raw(cmdLeft()...).text(separatorB()).ln()

	// ── Column headers ────────────────────────────────────────────────────────
	// Mirrors itemLineB layout: qty(4)+sp(1)+name(22)+price(15) = 42
	r.raw(cmdBoldOn()...).
		text(fmt.Sprintf("%-4s %-22s%15s", "Ud.", "Descripcion", "Total")).ln().
		raw(cmdBoldOff()...)
	r.raw(cmdLeft()...).text(separatorB()).ln()

	// ── Line items ────────────────────────────────────────────────────────────
	for _, item := range f.Detalles {
		r.raw(cmdLeft()...).
			text(itemLineB(item.Cantidad, item.Producto.Nombre, item.PrecioTotal)).ln()
		if item.Cantidad > 1 {
			r.raw(cmdLeft()...).text(unitLineB(item.PrecioUnitario)).ln()
		}
	}

	r.raw(cmdLeft()...).text(separatorB()).ln()

	// ── Totals ────────────────────────────────────────────────────────────────
	r.raw(cmdLeft()...).text(totalLineB("Subtotal:", formatCOP(f.Subtotal))).ln()
	r.raw(cmdLeft()...).text(totalLineB("IVA (19%):", formatCOP(f.IVA))).ln()
	r.raw(cmdLeft()...).text(separatorB()).ln()

	// Grand total — back to Font A, double size
	r.raw(cmdFontA()...).
		raw(cmdRight()...).
		raw(cmdBoldOn()...).
		raw(cmdDblHOn()...).
		text(fmt.Sprintf("%-10s%12s", "TOTAL:", formatCOP(f.Total))).ln().
		raw(cmdDblHOff()...).
		raw(cmdBoldOff()...)

	r.raw(cmdFontB()...)
	r.raw(cmdLeft()...).text(separatorB()).ln()

	// ── QR code (raster bitmap) ───────────────────────────────────────────────
	qrContent := fmt.Sprintf("%s|%s", f.NumeroFactura, f.UUID)
	qrBytes := qrBitmap(qrContent)
	if len(qrBytes) > 0 {
		r.raw(cmdFontA()...)
		r.raw(cmdCenter()...)
		r.raw(qrBytes...)
		r.raw('\n')
		r.raw(cmdFontB()...)
		r.raw(cmdCenter()...).text("Escanea para ver tu factura").ln()
		r.raw(cmdCenter()...).text(f.NumeroFactura).ln()
		r.raw(cmdLeft()...).text(separatorB()).ln()
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	r.raw(cmdFontA()...)
	r.raw(cmdCenter()...).
		raw(cmdBoldOn()...).
		text("Gracias por su compra!").ln().
		raw(cmdBoldOff()...)
	r.raw(cmdCenter()...).
		text("Conserva este recibo").ln().
		text(time.Now().Format("Generado 02/01/2006")).ln()

	// ── Feed + cut ────────────────────────────────────────────────────────────
	r.raw('\n', '\n', '\n').raw(cmdCut()...)

	return r
}

// ── Legacy compatibility ──────────────────────────────────────────────────────

//nolint:unused
func setupEndpoint(dev *gousb.Device) (*gousb.OutEndpoint, func(), error) {
	return getOutEndpoint(dev)
}

//nolint:unused
func formatCurrency(val float64) string { return formatCOP(val) }

//nolint:unused
func encodeText(s string) ([]byte, error) {
	return charmap.CodePage850.NewEncoder().Bytes([]byte(s))
}

//nolint:unused
func selectCodePage(n byte) []byte { return cmdCodePage(n) }

//nolint:unused
func center() []byte { return cmdCenter() }

//nolint:unused
func left() []byte { return cmdLeft() }

//nolint:unused
func right() []byte { return cmdRight() }

//nolint:unused
func lineBreak() []byte { return []byte{'\n'} }

//nolint:unused
func boldOn() []byte { return cmdBoldOn() }

//nolint:unused
func boldOff() []byte { return cmdBoldOff() }

//nolint:unused
func doubleHeightOn() []byte { return cmdDblHOn() }

//nolint:unused
func doubleHeightOff() []byte { return cmdDblHOff() }
