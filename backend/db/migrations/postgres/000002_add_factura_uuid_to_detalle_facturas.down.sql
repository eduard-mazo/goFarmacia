ALTER TABLE detalle_facturas
DROP CONSTRAINT IF EXISTS fk_detalle_factura_uuid;

ALTER TABLE detalle_facturas
DROP COLUMN IF EXISTS factura_uuid;