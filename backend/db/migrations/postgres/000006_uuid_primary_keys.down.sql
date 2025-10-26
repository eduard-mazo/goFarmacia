-- 20251026154000_uuid_primary_keys.down.sql
BEGIN;

-- 1. DROP ALL UUID-BASED FOREIGN KEYS
ALTER TABLE IF EXISTS public.compras
DROP CONSTRAINT IF EXISTS fk_compras_proveedor;

ALTER TABLE IF EXISTS public.detalle_compras
DROP CONSTRAINT IF EXISTS fk_detalle_compras_compra;

ALTER TABLE IF EXISTS public.detalle_compras
DROP CONSTRAINT IF EXISTS fk_detalle_compras_producto;

ALTER TABLE IF EXISTS public.detalle_facturas
DROP CONSTRAINT IF EXISTS fk_detalle_factura_uuid;

ALTER TABLE IF EXISTS public.detalle_facturas
DROP CONSTRAINT IF EXISTS fk_detalle_facturas_producto;

ALTER TABLE IF EXISTS public.detalle_facturas
DROP CONSTRAINT IF EXISTS detalle_facturas_factura_producto_key;

ALTER TABLE IF EXISTS public.facturas
DROP CONSTRAINT IF EXISTS fk_facturas_cliente;

ALTER TABLE IF EXISTS public.facturas
DROP CONSTRAINT IF EXISTS fk_facturas_vendedor;

ALTER TABLE IF EXISTS public.operacion_stocks
DROP CONSTRAINT IF EXISTS fk_operacion_factura_uuid;

ALTER TABLE IF EXISTS public.operacion_stocks
DROP CONSTRAINT IF EXISTS fk_producto;

ALTER TABLE IF EXISTS public.operacion_stocks
DROP CONSTRAINT IF EXISTS fk_vendedor;

-- 2. DROP UUID PRIMARY KEYS
ALTER TABLE public.clientes
DROP CONSTRAINT IF EXISTS clientes_pkey;

ALTER TABLE public.compras
DROP CONSTRAINT IF EXISTS compras_pkey;

ALTER TABLE public.detalle_compras
DROP CONSTRAINT IF EXISTS detalle_compras_pkey;

ALTER TABLE public.detalle_facturas
DROP CONSTRAINT IF EXISTS detalle_facturas_pkey;

ALTER TABLE public.facturas
DROP CONSTRAINT IF EXISTS facturas_pkey;

ALTER TABLE public.operacion_stocks
DROP CONSTRAINT IF EXISTS operacion_stocks_pkey;

ALTER TABLE public.productos
DROP CONSTRAINT IF EXISTS productos_pkey;

ALTER TABLE public.proveedors
DROP CONSTRAINT IF EXISTS proveedors_pkey;

ALTER TABLE public.vendedors
DROP CONSTRAINT IF EXISTS vendedors_pkey;

-- 3. ADD BACK ID (BIGSERIAL) COLUMNS AND OLD ID-BASED FK COLUMNS
ALTER TABLE public.clientes
ADD COLUMN id bigserial NOT NULL;

ALTER TABLE public.compras
ADD COLUMN id bigserial NOT NULL,
ADD COLUMN proveedor_id bigint;

ALTER TABLE public.detalle_compras
ADD COLUMN id bigserial NOT NULL,
ADD COLUMN compra_id bigint,
ADD COLUMN producto_id bigint;

ALTER TABLE public.detalle_facturas
ADD COLUMN id bigserial NOT NULL,
ADD COLUMN factura_id bigint,
ADD COLUMN producto_id bigint;

ALTER TABLE public.facturas
ADD COLUMN id bigserial NOT NULL,
ADD COLUMN vendedor_id bigint,
ADD COLUMN cliente_id bigint;

ALTER TABLE public.operacion_stocks
ADD COLUMN id bigserial NOT NULL,
ADD COLUMN producto_id bigint,
ADD COLUMN vendedor_id bigint,
ADD COLUMN factura_id bigint;

ALTER TABLE public.operacion_stocks
ALTER COLUMN uuid
SET
    DATA TYPE text;

ALTER TABLE public.productos
ADD COLUMN id bigserial NOT NULL;

ALTER TABLE public.proveedors
ADD COLUMN id bigserial NOT NULL;

ALTER TABLE public.vendedors
ADD COLUMN id bigserial NOT NULL;

-- 4. DATA MIGRATION: RE-POPULATE OLD ID COLUMNS
-- (This assumes no new rows were created that would mismatch bigserial sequence)
UPDATE public.compras t
SET
    proveedor_id = p.id
FROM
    public.proveedors p
WHERE
    t.proveedor_uuid = p.uuid;

UPDATE public.detalle_compras t
SET
    compra_id = c.id
FROM
    public.compras c
WHERE
    t.compra_uuid = c.uuid;

UPDATE public.detalle_compras t
SET
    producto_id = p.id
FROM
    public.productos p
WHERE
    t.producto_uuid = p.uuid;

UPDATE public.detalle_facturas t
SET
    producto_id = p.id
FROM
    public.productos p
WHERE
    t.producto_uuid = p.uuid;

UPDATE public.detalle_facturas t
SET
    factura_id = f.id
FROM
    public.facturas f
WHERE
    t.factura_uuid = f.uuid;

UPDATE public.facturas t
SET
    vendedor_id = v.id
FROM
    public.vendedors v
WHERE
    t.vendedor_uuid = v.uuid;

UPDATE public.facturas t
SET
    cliente_id = c.id
FROM
    public.clientes c
WHERE
    t.cliente_uuid = c.uuid;

UPDATE public.operacion_stocks t
SET
    producto_id = p.id
FROM
    public.productos p
WHERE
    t.producto_uuid = p.uuid;

UPDATE public.operacion_stocks t
SET
    vendedor_id = v.id
FROM
    public.vendedors v
WHERE
    t.vendedor_uuid = v.uuid;

UPDATE public.operacion_stocks t
SET
    factura_id = f.id
FROM
    public.facturas f
WHERE
    t.factura_uuid = f.uuid;

-- 5. DROP NEW UUID-BASED FK COLUMNS
ALTER TABLE public.compras
DROP COLUMN proveedor_uuid;

ALTER TABLE public.detalle_compras
DROP COLUMN compra_uuid,
DROP COLUMN producto_uuid;

ALTER TABLE public.detalle_facturas
DROP COLUMN producto_uuid;

ALTER TABLE public.facturas
DROP COLUMN vendedor_uuid,
DROP COLUMN cliente_uuid;

ALTER TABLE public.operacion_stocks
DROP COLUMN producto_uuid,
DROP COLUMN vendedor_uuid;

-- 6. RE-CREATE ID PKEYS AND UUID UNIQUE CONSTRAINTS
ALTER TABLE public.clientes ADD CONSTRAINT clientes_pkey PRIMARY KEY (id),
ADD CONSTRAINT clientes_uuid_unique UNIQUE (uuid);

ALTER TABLE public.compras ADD CONSTRAINT compras_pkey PRIMARY KEY (id),
ADD CONSTRAINT compras_uuid_unique UNIQUE (uuid);

ALTER TABLE public.detalle_compras ADD CONSTRAINT detalle_compras_pkey PRIMARY KEY (id),
ADD CONSTRAINT detalle_compras_uuid_unique UNIQUE (uuid);

ALTER TABLE public.detalle_facturas ADD CONSTRAINT detalle_facturas_pkey PRIMARY KEY (id),
ADD CONSTRAINT detalle_facturas_uuid_unique UNIQUE (uuid),
ADD CONSTRAINT detalle_facturas_factura_id_producto_id_key UNIQUE (factura_id, producto_id);

ALTER TABLE public.facturas ADD CONSTRAINT facturas_pkey PRIMARY KEY (id),
ADD CONSTRAINT facturas_uuid_unique UNIQUE (uuid);

ALTER TABLE public.operacion_stocks ADD CONSTRAINT operacion_stocks_pkey PRIMARY KEY (id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_operacion_stocks_uuid ON public.operacion_stocks USING btree (uuid);

ALTER TABLE public.productos ADD CONSTRAINT productos_pkey PRIMARY KEY (id),
ADD CONSTRAINT productos_uuid_unique UNIQUE (uuid);

ALTER TABLE public.proveedors ADD CONSTRAINT proveedors_pkey PRIMARY KEY (id),
ADD CONSTRAINT proveedors_uuid_unique UNIQUE (uuid);

ALTER TABLE public.vendedors ADD CONSTRAINT vendedors_pkey PRIMARY KEY (id),
ADD CONSTRAINT vendedors_uuid_unique UNIQUE (uuid);

-- 7. RE-CREATE OLD ID-BASED FOREIGN KEYS
ALTER TABLE public.compras ADD CONSTRAINT fk_compras_proveedor FOREIGN KEY (proveedor_id) REFERENCES public.proveedors (id);

ALTER TABLE public.detalle_compras ADD CONSTRAINT fk_compras_detalles FOREIGN KEY (compra_id) REFERENCES public.compras (id);

ALTER TABLE public.detalle_compras ADD CONSTRAINT fk_detalle_compras_producto FOREIGN KEY (producto_id) REFERENCES public.productos (id);

ALTER TABLE public.detalle_facturas ADD CONSTRAINT fk_detalle_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES public.facturas (uuid) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE public.detalle_facturas ADD CONSTRAINT fk_detalle_facturas_producto FOREIGN KEY (producto_id) REFERENCES public.productos (id);

ALTER TABLE public.detalle_facturas ADD CONSTRAINT fk_facturas_detalles FOREIGN KEY (factura_id) REFERENCES public.facturas (id);

ALTER TABLE public.facturas ADD CONSTRAINT fk_facturas_cliente FOREIGN KEY (cliente_id) REFERENCES public.clientes (id);

ALTER TABLE public.facturas ADD CONSTRAINT fk_facturas_vendedor FOREIGN KEY (vendedor_id) REFERENCES public.vendedors (id);

ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_factura FOREIGN KEY (factura_id) REFERENCES public.facturas (id) ON DELETE SET NULL;

ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_operacion_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES public.facturas (uuid) ON UPDATE CASCADE ON DELETE SET NULL;

ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_producto FOREIGN KEY (producto_id) REFERENCES public.productos (id) ON DELETE RESTRICT;

ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_vendedor FOREIGN KEY (vendedor_id) REFERENCES public.vendedors (id) ON DELETE RESTRICT;

COMMIT;