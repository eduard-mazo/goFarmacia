// processor/xml_parser.go
package processor

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// ─── DIAN UBL 2.1 Document Types ─────────────────────────────────────────────
//
// 01 → Factura de Venta          → XML root <Invoice>
// 02 → Factura de Exportación    → XML root <Invoice>
// 91 → Nota Crédito              → XML root <CreditNote>
// 92 → Nota Débito               → XML root <DebitNote>
//
// Each document can arrive either as a direct XML or wrapped inside an
// <AttachedDocument> envelope where the actual document is CDATA-encoded
// inside Attachment/ExternalReference/Description.

// ─── Shared sub-elements ─────────────────────────────────────────────────────

type Supplier struct {
	Party Party `xml:"Party"`
}

type Customer struct {
	Party Party `xml:"Party"`
}

type Party struct {
	PartyTaxScheme PartyTaxScheme `xml:"PartyTaxScheme"`
	PartyName      []PartyName    `xml:"PartyName"`
}

type PartyTaxScheme struct {
	RegistrationName string `xml:"RegistrationName"`
	CompanyID        string `xml:"CompanyID"`
}

type PartyName struct {
	Name string `xml:"Name"`
}

type TaxTotal struct {
	TaxAmount float64 `xml:"TaxAmount"`
}

type LegalMonetaryTotal struct {
	LineExtensionAmount float64 `xml:"LineExtensionAmount"`
	TaxExclusiveAmount  float64 `xml:"TaxExclusiveAmount"`
	TaxInclusiveAmount  float64 `xml:"TaxInclusiveAmount"`
	PayableAmount       float64 `xml:"PayableAmount"`
}

// Item contains product description, code, and extra properties.
type Item struct {
	Description                string                     `xml:"Description"`
	StandardItemIdentification StandardItemIdentification `xml:"StandardItemIdentification"`
	SellersItemIdentification  SellersItemIdentification  `xml:"SellersItemIdentification"`
	AdditionalItemProperties   []AdditionalItemProperty   `xml:"AdditionalItemProperty"`
}

type StandardItemIdentification struct {
	ID string `xml:"ID"`
}

type SellersItemIdentification struct {
	ID string `xml:"ID"`
}

type AdditionalItemProperty struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

type Price struct {
	PriceAmount  float64 `xml:"PriceAmount"`
	BaseQuantity float64 `xml:"BaseQuantity"`
}

// DiscrepancyResponse links a credit/debit note to its original invoice.
type DiscrepancyResponse struct {
	ReferenceID  string `xml:"ReferenceID"`
	ResponseCode string `xml:"ResponseCode"`
	Description  string `xml:"Description"`
}

// ─── Invoice (tipo 01, 02) ────────────────────────────────────────────────────

type Invoice struct {
	XMLName          xml.Name           `xml:"Invoice"`
	ID               string             `xml:"ID"`
	IssueDate        string             `xml:"IssueDate"`
	IssueTime        string             `xml:"IssueTime"`
	DocumentCurrency string             `xml:"DocumentCurrencyCode"`
	Supplier         Supplier           `xml:"AccountingSupplierParty"`
	Customer         Customer           `xml:"AccountingCustomerParty"`
	TaxTotal         []TaxTotal         `xml:"TaxTotal"`
	MonetaryTotal    LegalMonetaryTotal `xml:"LegalMonetaryTotal"`
	InvoiceLines     []InvoiceLine      `xml:"InvoiceLine"`
}

type InvoiceLine struct {
	ID                  float64    `xml:"ID"`
	Note                string     `xml:"Note"`
	InvoicedQuantity    float64    `xml:"InvoicedQuantity"`
	LineExtensionAmount float64    `xml:"LineExtensionAmount"`
	TaxTotal            []TaxTotal `xml:"TaxTotal"`
	Item                Item       `xml:"Item"`
	Price               Price      `xml:"Price"`
}

// ─── CreditNote (tipo 91) ─────────────────────────────────────────────────────

type CreditNote struct {
	XMLName             xml.Name             `xml:"CreditNote"`
	ID                  string               `xml:"ID"`
	IssueDate           string               `xml:"IssueDate"`
	IssueTime           string               `xml:"IssueTime"`
	DocumentCurrency    string               `xml:"DocumentCurrencyCode"`
	DiscrepancyResponse DiscrepancyResponse  `xml:"DiscrepancyResponse"`
	Supplier            Supplier             `xml:"AccountingSupplierParty"`
	Customer            Customer             `xml:"AccountingCustomerParty"`
	TaxTotal            []TaxTotal           `xml:"TaxTotal"`
	MonetaryTotal       LegalMonetaryTotal   `xml:"LegalMonetaryTotal"`
	CreditNoteLines     []CreditNoteLine     `xml:"CreditNoteLine"`
}

type CreditNoteLine struct {
	ID                  float64    `xml:"ID"`
	Note                string     `xml:"Note"`
	CreditedQuantity    float64    `xml:"CreditedQuantity"`
	LineExtensionAmount float64    `xml:"LineExtensionAmount"`
	TaxTotal            []TaxTotal `xml:"TaxTotal"`
	Item                Item       `xml:"Item"`
	Price               Price      `xml:"Price"`
}

// ─── DebitNote (tipo 92) ──────────────────────────────────────────────────────

type DebitNote struct {
	XMLName                xml.Name             `xml:"DebitNote"`
	ID                     string               `xml:"ID"`
	IssueDate              string               `xml:"IssueDate"`
	IssueTime              string               `xml:"IssueTime"`
	DocumentCurrency       string               `xml:"DocumentCurrencyCode"`
	DiscrepancyResponse    DiscrepancyResponse  `xml:"DiscrepancyResponse"`
	Supplier               Supplier             `xml:"AccountingSupplierParty"`
	Customer               Customer             `xml:"AccountingCustomerParty"`
	TaxTotal               []TaxTotal           `xml:"TaxTotal"`
	LegalMonetaryTotal     LegalMonetaryTotal   `xml:"LegalMonetaryTotal"`
	RequestedMonetaryTotal LegalMonetaryTotal   `xml:"RequestedMonetaryTotal"`
	DebitNoteLines         []DebitNoteLine      `xml:"DebitNoteLine"`
}

type DebitNoteLine struct {
	ID                  float64    `xml:"ID"`
	Note                string     `xml:"Note"`
	DebitedQuantity     float64    `xml:"DebitedQuantity"`
	LineExtensionAmount float64    `xml:"LineExtensionAmount"`
	TaxTotal            []TaxTotal `xml:"TaxTotal"`
	Item                Item       `xml:"Item"`
	Price               Price      `xml:"Price"`
}

// ─── AttachedDocument envelope ───────────────────────────────────────────────

type AttachedDocument struct {
	XMLName    xml.Name   `xml:"AttachedDocument"`
	Attachment Attachment `xml:"Attachment"`
}

type Attachment struct {
	ExternalReference ExternalReference `xml:"ExternalReference"`
}

type ExternalReference struct {
	Description CData `xml:"Description"`
}

type CData struct {
	Text string `xml:",cdata"`
}

// ─── Output ───────────────────────────────────────────────────────────────────

// ParsedProduct is the unified output for any DIAN document line.
type ParsedProduct struct {
	// Line-level fields
	Description string
	Code        string
	Quantity    float64
	UnitPrice   float64
	Total       float64
	TaxAmount   float64
	Properties  map[string]string

	// Document-level fields
	InvoiceID    string
	IssueDate    string
	IssueTime    string
	SupplierName string
	SupplierNIT  string
	CustomerName string
	CustomerNIT  string
	Currency     string
	TotalInvoice float64

	// Accounting metadata
	// DocumentType: "01"=Factura, "02"=FacturaExportación,
	//               "91"=NotaCrédito, "92"=NotaDébito
	DocumentType  string
	ReferenciaDoc string // For credit/debit notes: original invoice number
}

// ─── Parsing ──────────────────────────────────────────────────────────────────

// parseRawBytes attempts to decode raw XML as any known DIAN document type.
// It does NOT handle the AttachedDocument envelope.
func parseRawBytes(data []byte) ([]ParsedProduct, error) {
	// Try Invoice (01 / 02)
	var inv Invoice
	xml.Unmarshal(data, &inv) //nolint:errcheck — fall through on mismatch
	if len(inv.InvoiceLines) > 0 {
		return buildInvoiceProducts(inv), nil
	}

	// Try CreditNote (91)
	var cn CreditNote
	xml.Unmarshal(data, &cn) //nolint:errcheck
	if len(cn.CreditNoteLines) > 0 {
		return buildCreditNoteProducts(cn), nil
	}

	// Try DebitNote (92)
	var dn DebitNote
	xml.Unmarshal(data, &dn) //nolint:errcheck
	if len(dn.DebitNoteLines) > 0 {
		return buildDebitNoteProducts(dn), nil
	}

	return nil, fmt.Errorf("tipo de documento no reconocido o sin líneas")
}

// ParseDocumentXMLBytes parses any DIAN UBL 2.1 document from raw XML bytes,
// handling both direct documents and AttachedDocument CDATA envelopes.
func ParseDocumentXMLBytes(data []byte) ([]ParsedProduct, error) {
	// 1. Try direct document first (most common case)
	products, err := parseRawBytes(data)
	if err == nil {
		return products, nil
	}

	// 2. Try AttachedDocument envelope
	var doc AttachedDocument
	if xmlErr := xml.Unmarshal(data, &doc); xmlErr != nil {
		return nil, fmt.Errorf("formato no reconocido: %w", err)
	}
	inner := strings.TrimSpace(doc.Attachment.ExternalReference.Description.Text)
	if inner == "" {
		return nil, fmt.Errorf("AttachedDocument sin CDATA interno")
	}

	// 3. Parse the inner document (could be any type)
	products, err = parseRawBytes([]byte(inner))
	if err != nil {
		return nil, fmt.Errorf("no se pudo decodificar el CDATA: %w", err)
	}
	return products, nil
}

// ParseInvoiceXMLBytes is kept for backward compatibility; delegates to
// ParseDocumentXMLBytes which handles all document types.
func ParseInvoiceXMLBytes(xmlData []byte) ([]ParsedProduct, error) {
	return ParseDocumentXMLBytes(xmlData)
}

// ParseInvoiceXML reads an XML file from disk and delegates to ParseDocumentXMLBytes.
func ParseInvoiceXML(xmlPath string) ([]ParsedProduct, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer '%s': %w", xmlPath, err)
	}
	return ParseDocumentXMLBytes(data)
}

// ─── Builders ────────────────────────────────────────────────────────────────

func resolvePartyName(p Party) string {
	name := p.PartyTaxScheme.RegistrationName
	if name == "" && len(p.PartyName) > 0 {
		name = p.PartyName[0].Name
	}
	return name
}

func buildInvoiceProducts(inv Invoice) []ParsedProduct {
	supplierName := resolvePartyName(inv.Supplier.Party)
	customerName := resolvePartyName(inv.Customer.Party)
	var products []ParsedProduct
	for _, line := range inv.InvoiceLines {
		props := make(map[string]string)
		for _, prop := range line.Item.AdditionalItemProperties {
			props[prop.Name] = prop.Value
		}
		var lineTax float64
		for _, tx := range line.TaxTotal {
			lineTax += tx.TaxAmount
		}
		code := line.Item.StandardItemIdentification.ID
		if code == "" {
			code = line.Item.SellersItemIdentification.ID
		}
		products = append(products, ParsedProduct{
			Description:   line.Item.Description,
			Code:          code,
			Quantity:      line.InvoicedQuantity,
			UnitPrice:     line.Price.PriceAmount,
			Total:         line.LineExtensionAmount,
			TaxAmount:     lineTax,
			Properties:    props,
			InvoiceID:     inv.ID,
			IssueDate:     inv.IssueDate,
			IssueTime:     inv.IssueTime,
			SupplierName:  supplierName,
			SupplierNIT:   inv.Supplier.Party.PartyTaxScheme.CompanyID,
			CustomerName:  customerName,
			CustomerNIT:   inv.Customer.Party.PartyTaxScheme.CompanyID,
			Currency:      inv.DocumentCurrency,
			TotalInvoice:  inv.MonetaryTotal.PayableAmount,
			DocumentType:  "01",
			ReferenciaDoc: "",
		})
	}
	return products
}

func buildCreditNoteProducts(cn CreditNote) []ParsedProduct {
	supplierName := resolvePartyName(cn.Supplier.Party)
	customerName := resolvePartyName(cn.Customer.Party)
	refDoc := cn.DiscrepancyResponse.ReferenceID
	var products []ParsedProduct
	for _, line := range cn.CreditNoteLines {
		props := make(map[string]string)
		for _, prop := range line.Item.AdditionalItemProperties {
			props[prop.Name] = prop.Value
		}
		var lineTax float64
		for _, tx := range line.TaxTotal {
			lineTax += tx.TaxAmount
		}
		code := line.Item.StandardItemIdentification.ID
		if code == "" {
			code = line.Item.SellersItemIdentification.ID
		}
		products = append(products, ParsedProduct{
			Description:   line.Item.Description,
			Code:          code,
			Quantity:      line.CreditedQuantity,
			UnitPrice:     line.Price.PriceAmount,
			Total:         line.LineExtensionAmount,
			TaxAmount:     lineTax,
			Properties:    props,
			InvoiceID:     cn.ID,
			IssueDate:     cn.IssueDate,
			IssueTime:     cn.IssueTime,
			SupplierName:  supplierName,
			SupplierNIT:   cn.Supplier.Party.PartyTaxScheme.CompanyID,
			CustomerName:  customerName,
			CustomerNIT:   cn.Customer.Party.PartyTaxScheme.CompanyID,
			Currency:      cn.DocumentCurrency,
			TotalInvoice:  cn.MonetaryTotal.PayableAmount,
			DocumentType:  "91",
			ReferenciaDoc: refDoc,
		})
	}
	return products
}

func buildDebitNoteProducts(dn DebitNote) []ParsedProduct {
	supplierName := resolvePartyName(dn.Supplier.Party)
	customerName := resolvePartyName(dn.Customer.Party)
	refDoc := dn.DiscrepancyResponse.ReferenceID
	// DebitNote may use RequestedMonetaryTotal or LegalMonetaryTotal
	total := dn.LegalMonetaryTotal.PayableAmount
	if total == 0 {
		total = dn.RequestedMonetaryTotal.PayableAmount
	}
	var products []ParsedProduct
	for _, line := range dn.DebitNoteLines {
		props := make(map[string]string)
		for _, prop := range line.Item.AdditionalItemProperties {
			props[prop.Name] = prop.Value
		}
		var lineTax float64
		for _, tx := range line.TaxTotal {
			lineTax += tx.TaxAmount
		}
		code := line.Item.StandardItemIdentification.ID
		if code == "" {
			code = line.Item.SellersItemIdentification.ID
		}
		products = append(products, ParsedProduct{
			Description:   line.Item.Description,
			Code:          code,
			Quantity:      line.DebitedQuantity,
			UnitPrice:     line.Price.PriceAmount,
			Total:         line.LineExtensionAmount,
			TaxAmount:     lineTax,
			Properties:    props,
			InvoiceID:     dn.ID,
			IssueDate:     dn.IssueDate,
			IssueTime:     dn.IssueTime,
			SupplierName:  supplierName,
			SupplierNIT:   dn.Supplier.Party.PartyTaxScheme.CompanyID,
			CustomerName:  customerName,
			CustomerNIT:   dn.Customer.Party.PartyTaxScheme.CompanyID,
			Currency:      dn.DocumentCurrency,
			TotalInvoice:  total,
			DocumentType:  "92",
			ReferenciaDoc: refDoc,
		})
	}
	return products
}
