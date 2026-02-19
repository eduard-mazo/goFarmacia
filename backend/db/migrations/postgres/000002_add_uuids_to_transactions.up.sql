-- 000001_add_uuids_to_transactions.up.sql
-- 1. Habilita la extensión UUID en PostgreSQL (es idempotente)
-- Esto es necesario para usar el tipo nativo 'uuid' y la función 'uuid_generate_v4()'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Migrar la tabla 'facturas'
ALTER TABLE public.facturas ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.facturas SET uuid = uuid_generate_v4() WHERE uuid IS NULL;
ALTER TABLE public.facturas ALTER COLUMN uuid SET NOT NULL;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'facturas_uuid_unique') THEN
    ALTER TABLE public.facturas ADD CONSTRAINT facturas_uuid_unique UNIQUE (uuid);
  END IF;
END $$;

-- 3. Migrar la tabla 'detalle_facturas'
ALTER TABLE public.detalle_facturas ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.detalle_facturas SET uuid = uuid_generate_v4() WHERE uuid IS NULL;
ALTER TABLE public.detalle_facturas ALTER COLUMN uuid SET NOT NULL;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'detalle_facturas_uuid_unique') THEN
    ALTER TABLE public.detalle_facturas ADD CONSTRAINT detalle_facturas_uuid_unique UNIQUE (uuid);
  END IF;
END $$;

-- 4. Migrar la tabla 'compras'
ALTER TABLE public.compras ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.compras SET uuid = uuid_generate_v4() WHERE uuid IS NULL;
ALTER TABLE public.compras ALTER COLUMN uuid SET NOT NULL;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'compras_uuid_unique') THEN
    ALTER TABLE public.compras ADD CONSTRAINT compras_uuid_unique UNIQUE (uuid);
  END IF;
END $$;

-- 5. Migrar la tabla 'detalle_compras'
ALTER TABLE public.detalle_compras ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.detalle_compras SET uuid = uuid_generate_v4() WHERE uuid IS NULL;
ALTER TABLE public.detalle_compras ALTER COLUMN uuid SET NOT NULL;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'detalle_compras_uuid_unique') THEN
    ALTER TABLE public.detalle_compras ADD CONSTRAINT detalle_compras_uuid_unique UNIQUE (uuid);
  END IF;
END $$;