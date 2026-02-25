-- tipo_documento classifies the DIAN UBL 2.1 document:
--   '01' → Factura de Venta (default)
--   '02' → Factura de Exportación
--   '91' → Nota Crédito
--   '92' → Nota Débito
ALTER TABLE facturas_compra
    ADD COLUMN IF NOT EXISTS tipo_documento text NOT NULL DEFAULT '01';

-- referencia_documento links credit/debit notes back to their original invoice number.
ALTER TABLE facturas_compra
    ADD COLUMN IF NOT EXISTS referencia_documento text NOT NULL DEFAULT '';
