ALTER TABLE operacion_stocks
ADD COLUMN factura_uuid UUID;

-- Rellenar con el UUID de la factura correspondiente
UPDATE operacion_stocks os
SET
    factura_uuid = f.uuid
FROM
    facturas f
WHERE
    os.factura_id = f.id;

CREATE INDEX IF NOT EXISTS idx_operacion_stocks_factura_uuid ON operacion_stocks (factura_uuid);

ALTER TABLE operacion_stocks ADD CONSTRAINT fk_operacion_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES facturas (uuid) ON UPDATE CASCADE ON DELETE SET NULL;