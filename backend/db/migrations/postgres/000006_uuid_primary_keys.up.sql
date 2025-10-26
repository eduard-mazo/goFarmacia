-- 20251026154000_uuid_primary_keys.up.sql

BEGIN;

-- 1. DROP ALL EXISTING FOREIGN KEYS
ALTER TABLE IF EXISTS public.compras DROP CONSTRAINT IF EXISTS fk_compras_proveedor;
ALTER TABLE IF EXISTS public.detalle_compras DROP CONSTRAINT IF EXISTS fk_compras_detalles;
ALTER TABLE IF EXISTS public.detalle_compras DROP CONSTRAINT IF EXISTS fk_detalle_compras_producto;
ALTER TABLE IF EXISTS public.detalle_facturas DROP CONSTRAINT IF EXISTS fk_detalle_factura_uuid;
ALTER TABLE IF EXISTS public.detalle_facturas DROP CONSTRAINT IF EXISTS fk_detalle_facturas_producto;
ALTER TABLE IF EXISTS public.detalle_facturas DROP CONSTRAINT IF EXISTS fk_facturas_detalles;
ALTER TABLE IF EXISTS public.facturas DROP CONSTRAINT IF EXISTS fk_facturas_cliente;
ALTER TABLE IF EXISTS public.facturas DROP CONSTRAINT IF EXISTS fk_facturas_vendedor;
ALTER TABLE IF EXISTS public.operacion_stocks DROP CONSTRAINT IF EXISTS fk_factura;
ALTER TABLE IF EXISTS public.operacion_stocks DROP CONSTRAINT IF EXISTS fk_operacion_factura_uuid;
ALTER TABLE IF EXISTS public.operacion_stocks DROP CONSTRAINT IF EXISTS fk_producto;
ALTER TABLE IF EXISTS public.operacion_stocks DROP CONSTRAINT IF EXISTS fk_vendedor;

-- 2. ADD NEW UUID-BASED FOREIGN KEY COLUMNS (where missing)
ALTER TABLE public.compras ADD COLUMN IF NOT EXISTS proveedor_uuid uuid;
ALTER TABLE public.detalle_compras ADD COLUMN IF NOT EXISTS compra_uuid uuid;
ALTER TABLE public.detalle_compras ADD COLUMN IF NOT EXISTS producto_uuid uuid;
ALTER TABLE public.detalle_facturas ADD COLUMN IF NOT EXISTS producto_uuid uuid;
ALTER TABLE public.facturas ADD COLUMN IF NOT EXISTS vendedor_uuid uuid;
ALTER TABLE public.facturas ADD COLUMN IF NOT EXISTS cliente_uuid uuid;
ALTER TABLE public.operacion_stocks ADD COLUMN IF NOT EXISTS producto_uuid uuid;
ALTER TABLE public.operacion_stocks ADD COLUMN IF NOT EXISTS vendedor_uuid uuid;
ALTER TABLE public.operacion_stocks ALTER COLUMN uuid SET DATA TYPE uuid USING (uuid::uuid);
ALTER TABLE public.operacion_stocks ALTER COLUMN uuid SET NOT NULL;

-- 3. DATA MIGRATION: POPULATE NEW UUID COLUMNS
-- (Run these before dropping the id columns)
UPDATE public.compras t SET proveedor_uuid = p.uuid FROM public.proveedors p WHERE t.proveedor_id = p.id;
UPDATE public.detalle_compras t SET compra_uuid = c.uuid FROM public.compras c WHERE t.compra_id = c.id;
UPDATE public.detalle_compras t SET producto_uuid = p.uuid FROM public.productos p WHERE t.producto_id = p.id;
UPDATE public.detalle_facturas t SET producto_uuid = p.uuid FROM public.productos p WHERE t.producto_id = p.id;
UPDATE public.facturas t SET vendedor_uuid = v.uuid FROM public.vendedors v WHERE t.vendedor_id = v.id;
UPDATE public.facturas t SET cliente_uuid = c.uuid FROM public.clientes c WHERE t.cliente_id = c.id;
UPDATE public.operacion_stocks t SET producto_uuid = p.uuid FROM public.productos p WHERE t.producto_id = p.id;
UPDATE public.operacion_stocks t SET vendedor_uuid = v.uuid FROM public.vendedors v WHERE t.vendedor_id = v.id;

-- 4. DROP OLD CONSTRAINTS (PKeys, old FKs, old UNIQUEs)
ALTER TABLE public.clientes DROP CONSTRAINT IF EXISTS clientes_pkey, DROP CONSTRAINT IF EXISTS clientes_uuid_unique;
ALTER TABLE public.compras DROP CONSTRAINT IF EXISTS compras_pkey, DROP CONSTRAINT IF EXISTS compras_uuid_unique;
ALTER TABLE public.detalle_compras DROP CONSTRAINT IF EXISTS detalle_compras_pkey, DROP CONSTRAINT IF EXISTS detalle_compras_uuid_unique;
ALTER TABLE public.detalle_facturas DROP CONSTRAINT IF EXISTS detalle_facturas_pkey, DROP CONSTRAINT IF EXISTS detalle_facturas_uuid_unique, DROP CONSTRAINT IF EXISTS detalle_facturas_factura_id_producto_id_key;
ALTER TABLE public.facturas DROP CONSTRAINT IF EXISTS facturas_pkey, DROP CONSTRAINT IF EXISTS facturas_uuid_unique;
ALTER TABLE public.operacion_stocks DROP CONSTRAINT IF EXISTS operacion_stocks_pkey;
DROP INDEX IF EXISTS public.idx_operacion_stocks_uuid;
ALTER TABLE public.productos DROP CONSTRAINT IF EXISTS productos_pkey, DROP CONSTRAINT IF EXISTS productos_uuid_unique;
ALTER TABLE public.proveedors DROP CONSTRAINT IF EXISTS proveedors_pkey, DROP CONSTRAINT IF EXISTS proveedors_uuid_unique;
ALTER TABLE public.vendedors DROP CONSTRAINT IF EXISTS vendedors_pkey, DROP CONSTRAINT IF EXISTS vendedors_uuid_unique;

-- 5. SET NEW UUID PRIMARY KEYS
ALTER TABLE public.clientes ADD PRIMARY KEY (uuid);
ALTER TABLE public.compras ADD PRIMARY KEY (uuid);
ALTER TABLE public.detalle_compras ADD PRIMARY KEY (uuid);
ALTER TABLE public.detalle_facturas ADD PRIMARY KEY (uuid);
ALTER TABLE public.facturas ADD PRIMARY KEY (uuid);
ALTER TABLE public.operacion_stocks ADD PRIMARY KEY (uuid);
ALTER TABLE public.productos ADD PRIMARY KEY (uuid);
ALTER TABLE public.proveedors ADD PRIMARY KEY (uuid);
ALTER TABLE public.vendedors ADD PRIMARY KEY (uuid);

-- 6. DROP OLD ID-BASED COLUMNS
ALTER TABLE public.clientes DROP COLUMN id;
ALTER TABLE public.compras DROP COLUMN id, DROP COLUMN proveedor_id;
ALTER TABLE public.detalle_compras DROP COLUMN id, DROP COLUMN compra_id, DROP COLUMN producto_id;
ALTER TABLE public.detalle_facturas DROP COLUMN id, DROP COLUMN factura_id, DROP COLUMN producto_id;
ALTER TABLE public.facturas DROP COLUMN id, DROP COLUMN vendedor_id, DROP COLUMN cliente_id;
ALTER TABLE public.operacion_stocks DROP COLUMN id, DROP COLUMN producto_id, DROP COLUMN vendedor_id, DROP COLUMN factura_id;
ALTER TABLE public.productos DROP COLUMN id;
ALTER TABLE public.proveedors DROP COLUMN id;
ALTER TABLE public.vendedors DROP COLUMN id;

-- 7. RE-CREATE FOREIGN KEYS USING UUID
ALTER TABLE public.compras ADD CONSTRAINT fk_compras_proveedor FOREIGN KEY (proveedor_uuid) REFERENCES public.proveedors (uuid);
ALTER TABLE public.detalle_compras ADD CONSTRAINT fk_detalle_compras_compra FOREIGN KEY (compra_uuid) REFERENCES public.compras (uuid);
ALTER TABLE public.detalle_compras ADD CONSTRAINT fk_detalle_compras_producto FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid);
ALTER TABLE public.detalle_facturas ADD CONSTRAINT fk_detalle_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES public.facturas (uuid) ON UPDATE CASCADE ON DELETE CASCADE;
ALTER TABLE public.detalle_facturas ADD CONSTRAINT fk_detalle_facturas_producto FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid);
ALTER TABLE public.detalle_facturas ADD CONSTRAINT detalle_facturas_factura_producto_key UNIQUE (factura_uuid, producto_uuid);
ALTER TABLE public.facturas ADD CONSTRAINT fk_facturas_cliente FOREIGN KEY (cliente_uuid) REFERENCES public.clientes (uuid);
ALTER TABLE public.facturas ADD CONSTRAINT fk_facturas_vendedor FOREIGN KEY (vendedor_uuid) REFERENCES public.vendedors (uuid);
ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_operacion_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES public.facturas (uuid) ON UPDATE CASCADE ON DELETE SET NULL;
ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_producto FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid) ON DELETE RESTRICT;
ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_vendedor FOREIGN KEY (vendedor_uuid) REFERENCES public.vendedors (uuid) ON DELETE RESTRICT;

COMMIT;