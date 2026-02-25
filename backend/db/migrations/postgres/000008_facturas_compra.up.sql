-- Facturas electrónicas recibidas de proveedores (DIAN UBL 2.1)
CREATE TABLE IF NOT EXISTS facturas_compra (
    uuid            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    proveedor_uuid  uuid REFERENCES proveedors(uuid) ON DELETE SET NULL,
    proveedor_nit   text NOT NULL,
    proveedor_nombre text NOT NULL,
    cliente_nit     text NOT NULL DEFAULT '',
    cliente_nombre  text NOT NULL DEFAULT '',
    numero_factura  text NOT NULL,
    cufe            text,
    fecha_emision   timestamp with time zone,
    moneda          text NOT NULL DEFAULT 'COP',
    subtotal        numeric(15,2) NOT NULL DEFAULT 0,
    iva             numeric(15,2) NOT NULL DEFAULT 0,
    total           numeric(15,2) NOT NULL DEFAULT 0,
    estado          text NOT NULL DEFAULT 'PENDIENTE',
    email_message_id text,
    created_at      timestamp with time zone DEFAULT NOW(),
    updated_at      timestamp with time zone DEFAULT NOW(),
    CONSTRAINT facturas_compra_cufe_unique UNIQUE (cufe),
    CONSTRAINT facturas_compra_email_unique UNIQUE (email_message_id)
);

CREATE TABLE IF NOT EXISTS facturas_compra_detalles (
    uuid                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    factura_compra_uuid uuid NOT NULL REFERENCES facturas_compra(uuid) ON DELETE CASCADE,
    codigo_producto     text,
    descripcion         text NOT NULL,
    cantidad            numeric(12,4) NOT NULL DEFAULT 1,
    precio_unitario     numeric(15,2) NOT NULL DEFAULT 0,
    total_linea         numeric(15,2) NOT NULL DEFAULT 0,
    impuesto_linea      numeric(15,2) NOT NULL DEFAULT 0,
    propiedades         jsonb DEFAULT '{}',
    created_at          timestamp with time zone DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fc_proveedor_nit   ON facturas_compra(proveedor_nit);
CREATE INDEX IF NOT EXISTS idx_fc_fecha_emision   ON facturas_compra(fecha_emision DESC);
CREATE INDEX IF NOT EXISTS idx_fc_estado          ON facturas_compra(estado);
CREATE INDEX IF NOT EXISTS idx_fcd_factura_uuid   ON facturas_compra_detalles(factura_compra_uuid);
