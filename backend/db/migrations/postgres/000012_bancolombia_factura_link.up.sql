ALTER TABLE transferencias_bancolombia
    ADD COLUMN factura_uuid UUID REFERENCES facturas(uuid) ON DELETE SET NULL,
    ADD COLUMN factura_numero TEXT NOT NULL DEFAULT '';
