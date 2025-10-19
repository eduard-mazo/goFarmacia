ALTER TABLE operacion_stocks
DROP CONSTRAINT IF EXISTS fk_operacion_factura_uuid;

ALTER TABLE operacion_stocks
DROP COLUMN IF EXISTS factura_uuid;