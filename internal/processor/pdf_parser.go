// internal/processor/pdf_parser.go
package processor

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// PDFInvoiceData holds text extracted from a PDF invoice and the resulting
// code→name mapping.
type PDFInvoiceData struct {
	RawText      string
	CodeToName   map[string]string // producto_code (UPPER) → product name
}

// ─── LOTE detection & parsing ─────────────────────────────────────────────────

var reLotePrefix = regexp.MustCompile(`(?i)^\s*LOTE\s*:?\s*\S`)

// IsLoteDescription returns true when the description is a batch/lot reference
// (e.g. "LOTE: 25A767 VENCE: 30/03/2027") rather than a real product name.
// Exported so xml_parser.go can use it without import cycles.
func IsLoteDescription(desc string) bool {
	return reLotePrefix.MatchString(strings.TrimSpace(desc))
}

var (
	reLoteVal = regexp.MustCompile(`(?i)LOTE\s*:?\s*(\S+)`)
	reVence   = regexp.MustCompile(`(?i)VENCE\s*:?\s*(\S+)`)
)

// ParseLoteInfo extracts batch number and expiry date from a LOTE description.
// Returns empty strings when not found.
func ParseLoteInfo(desc string) (lote, vence string) {
	if m := reLoteVal.FindStringSubmatch(desc); len(m) == 2 {
		// Skip the token if it looks like a date (meaning no batch number before VENCE)
		if !strings.Contains(m[1], "/") {
			lote = m[1]
		}
	}
	if m := reVence.FindStringSubmatch(desc); len(m) == 2 {
		vence = m[1]
	}
	return
}

// ─── PDF text extraction ──────────────────────────────────────────────────────

// ParsePDFBytes extracts all text from raw PDF bytes and builds a
// code→product-name mapping by scanning for known product codes.
// knownCodes is the list of product codes from the XML (used as anchors).
func ParsePDFBytes(data []byte, knownCodes []string) (PDFInvoiceData, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return PDFInvoiceData{}, fmt.Errorf("leer PDF: %w", err)
	}

	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteRune('\n')
	}

	raw := sb.String()
	codeMap := extractCodeToName(raw, knownCodes)
	return PDFInvoiceData{RawText: raw, CodeToName: codeMap}, nil
}

// extractCodeToName scans PDF text for each known product code and collects
// the following non-trivial text lines as the product name.
//
// DIAN invoice PDFs from software like Siigo place the product code in a column
// and the product name in the adjacent/next column. After pdftotext extraction
// the pattern is typically:
//
//	... [CODE]
//	[Product Name line 1]
//	[Product Name line 2 (optional)]
//	[LOTE/quantity/price line]
//	... [NEXT CODE]
func extractCodeToName(text string, codes []string) map[string]string {
	result := make(map[string]string)
	if len(codes) == 0 {
		return result
	}

	// Sort codes longest-first to avoid a shorter code matching inside a longer one.
	sorted := make([]string, len(codes))
	copy(sorted, codes)
	sort.Slice(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })

	lines := strings.Split(text, "\n")

	for _, code := range sorted {
		if code == "" {
			continue
		}
		codeUp := strings.ToUpper(code)

		for lineIdx, line := range lines {
			// Whole-word match: the code must appear as a standalone token.
			if !containsWholeWord(strings.ToUpper(line), codeUp) {
				continue
			}

			// Collect candidate name lines that follow (skip the code line itself).
			name := collectNameAfter(lines, lineIdx+1)

			// If nothing found after, try the lines immediately BEFORE the code
			// (some PDF layouts place name before code in extraction order).
			if name == "" {
				name = collectNameBefore(lines, lineIdx-1)
			}

			if name != "" {
				result[codeUp] = name
				break
			}
		}
	}
	return result
}

// containsWholeWord returns true if word appears as a whole token in s.
func containsWholeWord(s, word string) bool {
	idx := strings.Index(s, word)
	if idx < 0 {
		return false
	}
	// Check boundaries
	before := idx == 0 || !isAlphanumeric(rune(s[idx-1]))
	after := idx+len(word) >= len(s) || !isAlphanumeric(rune(s[idx+len(word)]))
	return before && after
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// collectNameAfter looks at lines[startIdx..] and returns the product name by
// joining non-trivial lines until a LOTE/numeric/empty break is hit.
func collectNameAfter(lines []string, startIdx int) string {
	var parts []string
	for i := startIdx; i < len(lines) && i < startIdx+5; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if len(parts) > 0 {
				break // stop at blank line once we have something
			}
			continue
		}
		// Stop conditions
		if IsLoteDescription(line) {
			break
		}
		if looksLikePriceOrQtyLine(line) {
			break
		}
		// Skip single character or purely numeric tokens
		if isNumericToken(line) || len(line) <= 1 {
			continue
		}
		parts = append(parts, line)
	}
	name := strings.Join(parts, " ")
	// Sanity: minimum 4 chars, must contain at least one letter
	if len(name) < 4 || !containsLetter(name) {
		return ""
	}
	return name
}

// collectNameBefore looks backwards from endIdx for a product name.
func collectNameBefore(lines []string, endIdx int) string {
	var parts []string
	for i := endIdx; i >= 0 && i > endIdx-5; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if len(parts) > 0 {
				break
			}
			continue
		}
		if IsLoteDescription(line) || looksLikePriceOrQtyLine(line) || isNumericToken(line) {
			break
		}
		if len(line) <= 1 {
			continue
		}
		parts = append([]string{line}, parts...)
	}
	name := strings.Join(parts, " ")
	if len(name) < 4 || !containsLetter(name) {
		return ""
	}
	return name
}

// looksLikePriceOrQtyLine returns true for lines that are amounts/quantities
// (e.g. "3.00", "7,275.00", "0%", "21,825.00").
var reNumericLine = regexp.MustCompile(`^[\d\s.,/%$*\-+]+$`)

func looksLikePriceOrQtyLine(s string) bool {
	return reNumericLine.MatchString(s)
}

func isNumericToken(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) && r != '.' && r != ',' && r != '-' && r != '+' {
			return false
		}
	}
	return true
}

func containsLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// ─── Product enrichment ───────────────────────────────────────────────────────

// EnrichProductsFromPDF replaces LOTE-only descriptions in products with the
// actual product name extracted from the companion PDF.
//
// For each product where:
//   - the current description is a LOTE reference, AND
//   - the product code is found in pdfData.CodeToName
//
// it replaces Description with the PDF name and stores enrichment metadata in
// Properties ("fuente_descripcion" = "pdf", "descripcion_xml" = original).
func EnrichProductsFromPDF(products []ParsedProduct, pdfData PDFInvoiceData) []ParsedProduct {
	if len(pdfData.CodeToName) == 0 {
		return products
	}
	for i, p := range products {
		if p.Code == "" {
			continue
		}
		// Only enrich when description is still a LOTE reference (may have
		// already been set from cbc:Name in xml_parser.go).
		if !IsLoteDescription(p.Description) {
			continue
		}
		pdfName, ok := pdfData.CodeToName[strings.ToUpper(p.Code)]
		if !ok || pdfName == "" {
			continue
		}
		if products[i].Properties == nil {
			products[i].Properties = make(map[string]string)
		}
		// Preserve original LOTE string if not already stored
		if _, exists := products[i].Properties["descripcion_xml"]; !exists {
			products[i].Properties["descripcion_xml"] = p.Description
		}
		// Extract LOTE/VENCE if not already extracted
		if _, hasLote := products[i].Properties["lote"]; !hasLote {
			lote, vence := ParseLoteInfo(p.Description)
			if lote != "" {
				products[i].Properties["lote"] = lote
			}
			if vence != "" {
				products[i].Properties["vencimiento"] = vence
			}
		}
		products[i].Description = pdfName
		products[i].Properties["fuente_descripcion"] = "pdf"
	}
	return products
}
