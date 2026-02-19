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
-- Only run if the old integer FK columns still exist (fresh migration path).
-- Skipped when running on a schema already migrated to UUID-only.
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'compras' AND column_name = 'proveedor_id') THEN
    UPDATE public.compras t SET proveedor_uuid = p.uuid FROM public.proveedors p WHERE t.proveedor_id = p.id;
  END IF;
END $$;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'detalle_compras' AND column_name = 'compra_id') THEN
    UPDATE public.detalle_compras t SET compra_uuid = c.uuid FROM public.compras c WHERE t.compra_id = c.id;
  END IF;
END $$;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'detalle_compras' AND column_name = 'producto_id') THEN
    UPDATE public.detalle_compras t SET producto_uuid = p.uuid FROM public.productos p WHERE t.producto_id = p.id;
  END IF;
END $$;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'detalle_facturas' AND column_name = 'producto_id') THEN
    UPDATE public.detalle_facturas t SET producto_uuid = p.uuid FROM public.productos p WHERE t.producto_id = p.id;
  END IF;
END $$;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'facturas' AND column_name = 'vendedor_id') THEN
    UPDATE public.facturas t SET vendedor_uuid = v.uuid FROM public.vendedors v WHERE t.vendedor_id = v.id;
  END IF;
END $$;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'facturas' AND column_name = 'cliente_id') THEN
    UPDATE public.facturas t SET cliente_uuid = c.uuid FROM public.clientes c WHERE t.cliente_id = c.id;
  END IF;
END $$;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'operacion_stocks' AND column_name = 'producto_id') THEN
    UPDATE public.operacion_stocks t SET producto_uuid = p.uuid FROM public.productos p WHERE t.producto_id = p.id;
  END IF;
END $$;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'operacion_stocks' AND column_name = 'vendedor_id') THEN
    UPDATE public.operacion_stocks t SET vendedor_uuid = v.uuid FROM public.vendedors v WHERE t.vendedor_id = v.id;
  END IF;
END $$;

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
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'clientes_pkey' AND contype = 'p') THEN
    ALTER TABLE public.clientes ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'compras_pkey' AND contype = 'p') THEN
    ALTER TABLE public.compras ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'detalle_compras_pkey' AND contype = 'p') THEN
    ALTER TABLE public.detalle_compras ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'detalle_facturas_pkey' AND contype = 'p') THEN
    ALTER TABLE public.detalle_facturas ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'facturas_pkey' AND contype = 'p') THEN
    ALTER TABLE public.facturas ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'operacion_stocks_pkey' AND contype = 'p') THEN
    ALTER TABLE public.operacion_stocks ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'productos_pkey' AND contype = 'p') THEN
    ALTER TABLE public.productos ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'proveedors_pkey' AND contype = 'p') THEN
    ALTER TABLE public.proveedors ADD PRIMARY KEY (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'vendedors_pkey' AND contype = 'p') THEN
    ALTER TABLE public.vendedors ADD PRIMARY KEY (uuid);
  END IF;
END $$;

-- 6. DROP OLD ID-BASED COLUMNS (IF EXISTS for idempotency)
ALTER TABLE public.clientes DROP COLUMN IF EXISTS id;
ALTER TABLE public.compras DROP COLUMN IF EXISTS id, DROP COLUMN IF EXISTS proveedor_id;
ALTER TABLE public.detalle_compras DROP COLUMN IF EXISTS id, DROP COLUMN IF EXISTS compra_id, DROP COLUMN IF EXISTS producto_id;
ALTER TABLE public.detalle_facturas DROP COLUMN IF EXISTS id, DROP COLUMN IF EXISTS factura_id, DROP COLUMN IF EXISTS producto_id;
ALTER TABLE public.facturas DROP COLUMN IF EXISTS id, DROP COLUMN IF EXISTS vendedor_id, DROP COLUMN IF EXISTS cliente_id;
ALTER TABLE public.operacion_stocks DROP COLUMN IF EXISTS id, DROP COLUMN IF EXISTS producto_id, DROP COLUMN IF EXISTS vendedor_id, DROP COLUMN IF EXISTS factura_id;
ALTER TABLE public.productos DROP COLUMN IF EXISTS id;
ALTER TABLE public.proveedors DROP COLUMN IF EXISTS id;
ALTER TABLE public.vendedors DROP COLUMN IF EXISTS id;

-- 7. RE-CREATE FOREIGN KEYS USING UUID
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_compras_proveedor') THEN
    ALTER TABLE public.compras ADD CONSTRAINT fk_compras_proveedor FOREIGN KEY (proveedor_uuid) REFERENCES public.proveedors (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_detalle_compras_compra') THEN
    ALTER TABLE public.detalle_compras ADD CONSTRAINT fk_detalle_compras_compra FOREIGN KEY (compra_uuid) REFERENCES public.compras (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_detalle_compras_producto') THEN
    ALTER TABLE public.detalle_compras ADD CONSTRAINT fk_detalle_compras_producto FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_detalle_factura_uuid') THEN
    ALTER TABLE public.detalle_facturas ADD CONSTRAINT fk_detalle_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES public.facturas (uuid) ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_detalle_facturas_producto') THEN
    ALTER TABLE public.detalle_facturas ADD CONSTRAINT fk_detalle_facturas_producto FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'detalle_facturas_factura_producto_key') THEN
    ALTER TABLE public.detalle_facturas ADD CONSTRAINT detalle_facturas_factura_producto_key UNIQUE (factura_uuid, producto_uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_facturas_cliente') THEN
    ALTER TABLE public.facturas ADD CONSTRAINT fk_facturas_cliente FOREIGN KEY (cliente_uuid) REFERENCES public.clientes (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_facturas_vendedor') THEN
    ALTER TABLE public.facturas ADD CONSTRAINT fk_facturas_vendedor FOREIGN KEY (vendedor_uuid) REFERENCES public.vendedors (uuid);
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_operacion_factura_uuid') THEN
    ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_operacion_factura_uuid FOREIGN KEY (factura_uuid) REFERENCES public.facturas (uuid) ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_producto') THEN
    ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_producto FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid) ON DELETE RESTRICT;
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_vendedor') THEN
    ALTER TABLE public.operacion_stocks ADD CONSTRAINT fk_vendedor FOREIGN KEY (vendedor_uuid) REFERENCES public.vendedors (uuid) ON DELETE RESTRICT;
  END IF;
END $$;

COMMIT;
