-- Migración: agregar columna uuid a múltiples tablas

ALTER TABLE IF EXISTS public.vendedors ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.vendedors SET uuid = gen_random_uuid() WHERE uuid IS NULL;
ALTER TABLE public.vendedors ALTER COLUMN uuid SET NOT NULL;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'vendedors_uuid_unique'
    ) THEN
        ALTER TABLE public.vendedors ADD CONSTRAINT vendedors_uuid_unique UNIQUE (uuid);
    END IF;
END $$;

ALTER TABLE IF EXISTS public.clientes ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.clientes SET uuid = gen_random_uuid() WHERE uuid IS NULL;
ALTER TABLE public.clientes ALTER COLUMN uuid SET NOT NULL;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'clientes_uuid_unique'
    ) THEN
        ALTER TABLE public.clientes ADD CONSTRAINT clientes_uuid_unique UNIQUE (uuid);
    END IF;
END $$;

ALTER TABLE IF EXISTS public.proveedors ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.proveedors SET uuid = gen_random_uuid() WHERE uuid IS NULL;
ALTER TABLE public.proveedors ALTER COLUMN uuid SET NOT NULL;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'proveedors_uuid_unique'
    ) THEN
        ALTER TABLE public.proveedors ADD CONSTRAINT proveedors_uuid_unique UNIQUE (uuid);
    END IF;
END $$;

ALTER TABLE IF EXISTS public.productos ADD COLUMN IF NOT EXISTS uuid uuid;
UPDATE public.productos SET uuid = gen_random_uuid() WHERE uuid IS NULL;
ALTER TABLE public.productos ALTER COLUMN uuid SET NOT NULL;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'productos_uuid_unique'
    ) THEN
        ALTER TABLE public.productos ADD CONSTRAINT productos_uuid_unique UNIQUE (uuid);
    END IF;
END $$;
