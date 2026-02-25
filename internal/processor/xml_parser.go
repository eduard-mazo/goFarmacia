// processor/xml_parser.go
package processor

import (
	"encoding/xml"
	"fmt"
	"os"
)

// --- Estructuras para el parseo del XML (formato UBL 2.1 DIAN) ---

// Invoice representa la estructura principal del XML de la factura.
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

// AttachedDocument se usa cuando la factura está dentro de esta etiqueta.
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

// InvoiceLine representa una línea de producto/servicio en la factura.
type InvoiceLine struct {
	ID                  float64    `xml:"ID"`
	Note                string     `xml:"Note"`
	InvoicedQuantity    float64    `xml:"InvoicedQuantity"`
	LineExtensionAmount float64    `xml:"LineExtensionAmount"`
	TaxTotal            []TaxTotal `xml:"TaxTotal"`
	Item                Item       `xml:"Item"`
	Price               Price      `xml:"Price"`
}

// Item contiene la descripción, código y propiedades adicionales del producto.
type Item struct {
	Description                string                     `xml:"Description"`
	StandardItemIdentification StandardItemIdentification `xml:"StandardItemIdentification"`
	SellersItemIdentification  SellersItemIdentification  `xml:"SellersItemIdentification"`
	AdditionalItemProperties   []AdditionalItemProperty   `xml:"AdditionalItemProperty"`
}

// StandardItemIdentification contiene el código del producto (ej. EAN, PLU).
type StandardItemIdentification struct {
	ID string `xml:"ID"`
}

// SellersItemIdentification
type SellersItemIdentification struct {
	ID string `xml:"ID"`
}

// AdditionalItemProperty representa un par clave-valor de una propiedad.
type AdditionalItemProperty struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

// Price contiene el precio unitario.
type Price struct {
	PriceAmount  float64 `xml:"PriceAmount"`
	BaseQuantity float64 `xml:"BaseQuantity"`
}

// ParsedProduct es la estructura limpia que devolverá la función de parseo.
type ParsedProduct struct {
	// Item level
	Description string
	Code        string
	Quantity    float64
	UnitPrice   float64
	Total       float64
	TaxAmount   float64
	Properties  map[string]string // Propiedades adicionales (Name -> Value)

	// Invoice level
	InvoiceID    string
	IssueDate    string
	IssueTime    string
	SupplierName string
	SupplierNIT  string
	CustomerName string
	CustomerNIT  string
	Currency     string
	TotalInvoice float64
}

// unmarshalInvoice decodes raw XML bytes into an Invoice, handling both direct
// <Invoice> roots and <AttachedDocument> wrappers with CDATA inner invoices.
func unmarshalInvoice(data []byte, invoice *Invoice) error {
	xml.Unmarshal(data, invoice)
	if len(invoice.InvoiceLines) == 0 {
		var doc AttachedDocument
		if err := xml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("formato no reconocido: %w", err)
		}
		if err := xml.Unmarshal([]byte(doc.Attachment.ExternalReference.Description.Text), invoice); err != nil {
			return fmt.Errorf("no se pudo decodificar el CDATA: %w", err)
		}
	}
	if len(invoice.InvoiceLines) == 0 {
		return fmt.Errorf("no se encontraron líneas de factura")
	}
	return nil
}

// buildProducts converts a parsed Invoice into a flat slice of ParsedProduct.
func buildProducts(invoice Invoice) ([]ParsedProduct, error) {
	supplierName := invoice.Supplier.Party.PartyTaxScheme.RegistrationName
	if supplierName == "" && len(invoice.Supplier.Party.PartyName) > 0 {
		supplierName = invoice.Supplier.Party.PartyName[0].Name
	}
	customerName := invoice.Customer.Party.PartyTaxScheme.RegistrationName
	if customerName == "" && len(invoice.Customer.Party.PartyName) > 0 {
		customerName = invoice.Customer.Party.PartyName[0].Name
	}

	var products []ParsedProduct
	for _, line := range invoice.InvoiceLines {
		props := make(map[string]string)
		for _, prop := range line.Item.AdditionalItemProperties {
			props[prop.Name] = prop.Value
		}
		var lineTax float64
		for _, tx := range line.TaxTotal {
			lineTax += tx.TaxAmount
		}
		products = append(products, ParsedProduct{
			Description:  line.Item.Description,
			Code:         line.Item.StandardItemIdentification.ID,
			Quantity:     line.InvoicedQuantity,
			UnitPrice:    line.Price.PriceAmount,
			Total:        line.LineExtensionAmount,
			TaxAmount:    lineTax,
			Properties:   props,
			InvoiceID:    invoice.ID,
			IssueDate:    invoice.IssueDate,
			IssueTime:    invoice.IssueTime,
			SupplierName: supplierName,
			SupplierNIT:  invoice.Supplier.Party.PartyTaxScheme.CompanyID,
			CustomerName: customerName,
			CustomerNIT:  invoice.Customer.Party.PartyTaxScheme.CompanyID,
			Currency:     invoice.DocumentCurrency,
			TotalInvoice: invoice.MonetaryTotal.PayableAmount,
		})
	}
	return products, nil
}

// ParseInvoiceXML lee un archivo XML y extrae los detalles de los productos.
func ParseInvoiceXML(xmlPath string) ([]ParsedProduct, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer '%s': %w", xmlPath, err)
	}
	var invoice Invoice
	if err := unmarshalInvoice(data, &invoice); err != nil {
		return nil, fmt.Errorf("'%s': %w", xmlPath, err)
	}
	return buildProducts(invoice)
}
