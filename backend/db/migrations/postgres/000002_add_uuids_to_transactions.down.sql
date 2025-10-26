-- 000001_add_uuids_to_transactions.down.sql

-- Revertir en orden inverso

-- 5. Revertir 'detalle_compras'
ALTER TABLE public.detalle_compras DROP CONSTRAINT IF EXISTS detalle_compras_uuid_unique;
ALTER TABLE public.detalle_compras DROP COLUMN IF EXISTS uuid;

-- 4. Revertir 'compras'
ALTER TABLE public.compras DROP CONSTRAINT IF EXISTS compras_uuid_unique;
ALTER TABLE public.compras DROP COLUMN IF EXISTS uuid;

-- 3. Revertir 'detalle_facturas'
ALTER TABLE public.detalle_facturas DROP CONSTRAINT IF EXISTS detalle_facturas_uuid_unique;
ALTER TABLE public.detalle_facturas DROP COLUMN IF EXISTS uuid;

-- 2. Revertir 'facturas'
ALTER TABLE public.facturas DROP CONSTRAINT IF EXISTS facturas_uuid_unique;
ALTER TABLE public.facturas DROP COLUMN IF EXISTS uuid;