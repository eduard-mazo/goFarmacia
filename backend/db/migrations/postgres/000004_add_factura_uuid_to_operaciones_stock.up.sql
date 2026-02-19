ALTER TABLE operacion_stocks ADD COLUMN IF NOT EXISTS factura_uuid UUID;

-- Rellenar con el UUID de la factura correspondiente
UPDATE operacion_stocks os
SET factura_uuid = f.uuid
FROM facturas f
WHERE os.factura_id = f.id AND os.factura_uuid IS NULL;

CREATE INDEX IF NOT EXISTS idx_operacion_stocks_factura_uuid ON operacion_stocks (factura_uuid);

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_operacion_factura_uuid') THEN
    ALTER TABLE operacion_stocks ADD CONSTRAINT fk_operacion_factura_uuid
      FOREIGN KEY (factura_uuid) REFERENCES facturas (uuid) ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END $$;