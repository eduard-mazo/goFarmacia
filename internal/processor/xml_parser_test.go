package processor

import (
	"encoding/xml"
	"testing"
)

func TestParseInvoiceXMLContent(t *testing.T) {
	// Mock de un XML UBL 2.1 simplificado
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
    <cbc:ID>SETT123</cbc:ID>
    <cbc:IssueDate>2026-02-23</cbc:IssueDate>
    <cbc:IssueTime>10:00:00</cbc:IssueTime>
    <cbc:DocumentCurrencyCode>COP</cbc:DocumentCurrencyCode>
    <cac:AccountingSupplierParty>
        <cac:Party>
            <cac:PartyTaxScheme>
                <cbc:RegistrationName>Proveedor Test</cbc:RegistrationName>
                <cbc:CompanyID>900123456</cbc:CompanyID>
            </cac:PartyTaxScheme>
        </cac:Party>
    </cac:AccountingSupplierParty>
    <cac:AccountingCustomerParty>
        <cac:Party>
            <cac:PartyTaxScheme>
                <cbc:RegistrationName>Cliente Test</cbc:RegistrationName>
                <cbc:CompanyID>800987654</cbc:CompanyID>
            </cac:PartyTaxScheme>
        </cac:Party>
    </cac:AccountingCustomerParty>
    <cac:LegalMonetaryTotal>
        <cbc:PayableAmount currencyID="COP">119000.00</cbc:PayableAmount>
    </cac:LegalMonetaryTotal>
    <cac:InvoiceLine>
        <cbc:ID>1</cbc:ID>
        <cbc:InvoicedQuantity>2</cbc:InvoicedQuantity>
        <cbc:LineExtensionAmount currencyID="COP">100000.00</cbc:LineExtensionAmount>
        <cac:TaxTotal>
            <cbc:TaxAmount currencyID="COP">19000.00</cbc:TaxAmount>
        </cac:TaxTotal>
        <cac:Item>
            <cbc:Description>Producto de Prueba</cbc:Description>
            <cac:StandardItemIdentification>
                <cbc:ID>770123456789</cbc:ID>
            </cac:StandardItemIdentification>
        </cac:Item>
        <cac:Price>
            <cbc:PriceAmount currencyID="COP">50000.00</cbc:PriceAmount>
        </cac:Price>
    </cac:InvoiceLine>
</Invoice>`

	var inv Invoice
	err := xml.Unmarshal([]byte(xmlContent), &inv)
	if err != nil {
		t.Fatalf("Error al deserializar XML de prueba: %v", err)
	}

	if inv.ID != "SETT123" {
		t.Errorf("Esperaba ID SETT123, obtuve %s", inv.ID)
	}

	if inv.MonetaryTotal.PayableAmount != 119000.00 {
		t.Errorf("Esperaba Total 119000, obtuve %f", inv.MonetaryTotal.PayableAmount)
	}

	if len(inv.InvoiceLines) != 1 {
		t.Fatalf("Esperaba 1 línea de factura, obtuve %d", len(inv.InvoiceLines))
	}

	line := inv.InvoiceLines[0]
	if line.Item.Description != "Producto de Prueba" {
		t.Errorf("Descripción incorrecta: %s", line.Item.Description)
	}
}
