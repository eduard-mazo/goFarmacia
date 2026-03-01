package backend

// POSPrinter.go — ESC/POS driver for 58 mm thermal receipt printers.
//
// Printer: USB VID 0x0416 / PID 0x5011 (POS58 family).
// Paper  : 58 mm, ~32 columns at normal size.
// Charset: PC850 (Latin-1 multilingual) — covers full Spanish alphabet.
//
// Public API (Wails-bound via *Db):
//   ImprimirRecibo(factura Factura) error
//   VerificarImpresora() bool

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/gousb"
	"golang.org/x/text/encoding/charmap"
)

// ── Hardware constants ────────────────────────────────────────────────────────

const (
	posVendorID  gousb.ID = 0x0416
	posProductID gousb.ID = 0x5011

	// ESC t n — PC850 Multilingual (Latin-1) code page.
	// Covers all Spanish accented chars: á é í ó ú ñ Á É Í Ó Ú Ñ ¿ ¡
	cpPC850 byte = 0x02

	// Printable columns at normal character size on 58 mm paper.
	paperCols = 32
)

// ── receipt — buffered ESC/POS builder ───────────────────────────────────────

// receipt accumulates ESC/POS bytes in memory and flushes them in one USB
// write, which is significantly faster than many small writes.
type receipt struct{ buf bytes.Buffer }

func newReceipt() *receipt { return &receipt{} }

// raw appends raw bytes (ESC/POS commands, ASCII, etc.).
func (r *receipt) raw(b ...byte) *receipt { r.buf.Write(b); return r }

// cmd is an alias for raw — used for named ESC/POS command sequences.
func (r *receipt) cmd(b ...byte) *receipt { return r.raw(b...) }

// text encodes a UTF-8 string to PC850 and appends it.
// A fresh encoder is created for every call to avoid transformer state issues.
// Characters not in PC850 are replaced with '?'.
func (r *receipt) text(s string) *receipt {
	// Create a fresh PC850 encoder each call — avoids state corruption on
	// sequences like repeated calls after an encoding error.
	b, err := charmap.CodePage850.NewEncoder().Bytes([]byte(s))
	if err != nil {
		// Fallback: strip to safe ASCII so the printer does not misparse commands.
		r.buf.WriteString(sanitizeToASCII(s))
		return r
	}
	r.buf.Write(b)
	return r
}

// sanitizeToASCII transliterates common Spanish characters to ASCII equivalents.
// Used only when PC850 encoding fails (should be rare).
func sanitizeToASCII(s string) string {
	return strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
		"ñ", "n", "Ñ", "N", "ü", "u", "Ü", "U",
		"¿", "?", "¡", "!", "€", "E",
	).Replace(s)
}

// ln appends a line-feed.
func (r *receipt) ln() *receipt { return r.raw('\n') }

// bytes returns the assembled payload.
func (r *receipt) bytes() []byte { return r.buf.Bytes() }

// ── ESC/POS command helpers ───────────────────────────────────────────────────

func cmdInit() []byte          { return []byte{0x1B, '@'} }
func cmdCodePage(n byte) []byte { return []byte{0x1B, 0x74, n} }
func cmdCenter() []byte        { return []byte{0x1B, 0x61, 0x01} }
func cmdLeft() []byte          { return []byte{0x1B, 0x61, 0x00} }
func cmdRight() []byte         { return []byte{0x1B, 0x61, 0x02} }
func cmdBoldOn() []byte        { return []byte{0x1B, 0x45, 0x01} }
func cmdBoldOff() []byte       { return []byte{0x1B, 0x45, 0x00} }
func cmdDblHOn() []byte        { return []byte{0x1D, 0x21, 0x01} } // double height only
func cmdDblHOff() []byte       { return []byte{0x1D, 0x21, 0x00} }
func cmdDblWOn() []byte        { return []byte{0x1D, 0x21, 0x10} } // double width only (bits 4-6 = 001)
func cmdDblWOff() []byte       { return []byte{0x1D, 0x21, 0x00} }
func cmdCut() []byte           { return []byte{0x1D, 0x56, 0x42, 0x00} } // partial cut

// ── QR code (ESC/POS model 2) ────────────────────────────────────────────────

// qrData builds the ESC/POS GS ( k command sequence to store and print a QR code.
//
// Parameters:
//
//	data       — string to encode (e.g. "F-0001|550e8400-e29b-…")
//	moduleSize — dot size 1–8 (3 = ~3 mm per module, suits 58 mm paper)
//	eccLevel   — error correction: 'L', 'M', 'Q', 'H'
//
// Requirements: printer must support ESC/POS QR model 2 (GS ( k).
// Most POS58/POS80 printers manufactured after ~2015 support this.
func qrData(data string, moduleSize byte, eccLevel byte) []byte {
	if moduleSize < 1 || moduleSize > 8 {
		moduleSize = 3
	}
	eccByte := map[byte]byte{'L': 48, 'M': 49, 'Q': 50, 'H': 51}[eccLevel]
	if eccByte == 0 {
		eccByte = 49 // default M
	}

	var buf bytes.Buffer

	// 1. Select model 2 (fn=65).
	buf.Write([]byte{0x1D, 0x28, 0x6B, 0x04, 0x00, 0x31, 0x41, 0x32, 0x00})

	// 2. Set module size (fn=67).
	buf.Write([]byte{0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x43, moduleSize})

	// 3. Set error-correction level (fn=69).
	buf.Write([]byte{0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x45, eccByte})

	// 4. Store data in printer buffer (fn=80).
	//    Payload length pL+pH counts: cn(1) + fn(1) + m(1) + data.
	pl := len(data) + 3
	pL := byte(pl & 0xFF)
	pH := byte(pl >> 8)
	buf.Write([]byte{0x1D, 0x28, 0x6B, pL, pH, 0x31, 0x50, 0x30})
	buf.WriteString(data)

	// 5. Print the stored symbol (fn=81).
	buf.Write([]byte{0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x51, 0x30})

	return buf.Bytes()
}

// ── Currency formatting ───────────────────────────────────────────────────────

// formatCOP formats a float64 as Colombian pesos: "$ 1.234.567"
// Thousands separator is '.' (dot), no decimals (pesos are whole currency).
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

// separator returns a full-width dashed line.
func separator() string { return strings.Repeat("-", paperCols) }

// runeLen returns the visible rune count (not byte count) of a string.
func runeLen(s string) int { return utf8.RuneCountInString(s) }

// pad right-pads s with spaces to width w (rune-aware).
func pad(s string, w int) string {
	l := runeLen(s)
	if l >= w {
		return s
	}
	return s + strings.Repeat(" ", w-l)
}

// rpad left-pads s with spaces to width w (right-aligns).
func rpad(s string, w int) string {
	l := runeLen(s)
	if l >= w {
		return s
	}
	return strings.Repeat(" ", w-l) + s
}

// truncate cuts s at max n runes. Uses ASCII "..." so CP850 encoding is safe.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-3]) + "..."
}

// itemLine formats one receipt line for a product.
//
// Layout (exactly 32 cols):
//
//	col 0-2   : qty   right-aligned  3 chars  e.g. "2x "
//	col 3     : space
//	col 4-20  : name  left-aligned  17 chars
//	col 21-31 : total right-aligned 11 chars  e.g. "  $ 12.000"
//
// 3 + 1 + 17 + 11 = 32 ✓
func itemLine(qty int, name string, total float64) string {
	qtyStr  := rpad(fmt.Sprintf("%dx", qty), 3)
	nameStr := pad(truncate(name, 17), 17)
	totStr  := rpad(formatCOP(total), 11)
	return fmt.Sprintf("%s %s%s", qtyStr, nameStr, totStr)
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
	// On Linux the kernel usblp driver may hold the device.
	// SetAutoDetach releases it automatically so we can claim the interface.
	if err := dev.SetAutoDetach(true); err != nil {
		// Non-fatal — some systems do not need it.
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
		if ep.Address&0x80 == 0 { // OUT endpoint (direction bit = 0)
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

// VerificarImpresora returns true if the POS printer is connected and accessible.
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

// ImprimirRecibo sends a fully-formatted sales receipt to the POS printer.
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

// buildReceipt assembles the full ESC/POS byte stream for a sales receipt.
func buildReceipt(f Factura) *receipt {
	r := newReceipt()

	// ── Init ──────────────────────────────────────────────────────────────────
	r.raw(cmdInit()...).
		raw(cmdCodePage(cpPC850)...)

	// ── Header: store name (double-width, centered) ───────────────────────────
	r.raw(cmdCenter()...).
		raw(cmdBoldOn()...).
		raw(cmdDblWOn()...).
		text("DROGUERIA LUNA").ln().
		raw(cmdDblWOff()...).
		raw(cmdBoldOff()...)

	// Store details (proper UTF-8 — encoder handles á, é, í, ó, ú, ñ)
	r.raw(cmdCenter()...).text("NIT: 70.120.237-8").ln()
	r.raw(cmdCenter()...).text("Calle 94 # 48-33, Medellín").ln()
	r.raw(cmdCenter()...).text("Tel: 305 445 6781").ln()

	r.raw(cmdLeft()...).text(separator()).ln()

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

	r.raw(cmdLeft()...).text(separator()).ln()

	// ── Column headers ────────────────────────────────────────────────────────
	// Header row mirrors itemLine layout: qty(3)+sp(1)+name(17)+price(11) = 32
	r.raw(cmdBoldOn()...).
		text(fmt.Sprintf("%-3s %-17s%11s", "Ud.", "Descripcion", "Total")).ln().
		raw(cmdBoldOff()...)

	r.raw(cmdLeft()...).text(separator()).ln()

	// ── Line items ────────────────────────────────────────────────────────────
	for _, item := range f.Detalles {
		r.raw(cmdLeft()...).
			text(itemLine(item.Cantidad, item.Producto.Nombre, item.PrecioTotal)).ln()

		// Show unit price as a sub-line when qty > 1 for clarity.
		if item.Cantidad > 1 {
			unitLine := fmt.Sprintf("    %s c/u", rpad(formatCOP(item.PrecioUnitario), 9))
			r.raw(cmdLeft()...).text(unitLine).ln()
		}
	}

	r.raw(cmdLeft()...).text(separator()).ln()

	// ── Totals ────────────────────────────────────────────────────────────────
	// Label(21) + value(11) = 32 cols
	printTotal := func(label, val string) {
		r.raw(cmdLeft()...).text(pad(label, 21) + rpad(val, 11)).ln()
	}

	printTotal("Subtotal:", formatCOP(f.Subtotal))
	printTotal("IVA (19%):", formatCOP(f.IVA))

	r.raw(cmdLeft()...).text(separator()).ln()

	// Grand total — bold + double height
	r.raw(cmdRight()...).
		raw(cmdBoldOn()...).
		raw(cmdDblHOn()...).
		text(pad("TOTAL:", 21) + rpad(formatCOP(f.Total), 11)).ln().
		raw(cmdDblHOff()...).
		raw(cmdBoldOff()...)

	r.raw(cmdLeft()...).text(separator()).ln()

	// ── QR code ───────────────────────────────────────────────────────────────
	// Encodes "F-0001|550e8400-e29b-41d4-a716-446655440000"
	// Scan to look up invoice by NumeroFactura or UUID.
	qrContent := fmt.Sprintf("%s|%s", f.NumeroFactura, f.UUID)

	r.raw(cmdCenter()...)
	r.raw(qrData(qrContent, 3, 'M')...)

	// Extra feeds so the QR symbol clears the print head before cut.
	r.raw('\n', '\n')

	r.raw(cmdCenter()...).text("Escanea para ver tu factura").ln()
	r.raw(cmdCenter()...).text(truncate(f.NumeroFactura, paperCols)).ln()

	r.raw(cmdLeft()...).text(separator()).ln()

	// ── Footer ────────────────────────────────────────────────────────────────
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
// Keep old unexported helpers so any existing call sites in other files
// continue to compile without modification.

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
