-- Agrega factura_uuid y normaliza con facturas.uuid
ALTER TABLE detalle_facturas
ADD COLUMN factura_uuid UUID;

UPDATE detalle_facturas df
SET
    factura_uuid = f.uuid
FROM
    facturas f
WHERE
    df.factura_id = f.id;

ALTER TABLE detalle_facturas ADD CONSTRAINT fk_detalle_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES facturas (uuid) ON UPDATE CASCADE ON DELETE CASCADE;