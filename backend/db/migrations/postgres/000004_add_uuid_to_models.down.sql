ALTER TABLE IF EXISTS public.vendedors
DROP CONSTRAINT IF EXISTS vendedors_uuid_unique;

ALTER TABLE IF EXISTS public.vendedors
DROP COLUMN IF EXISTS uuid;

ALTER TABLE IF EXISTS public.clientes
DROP CONSTRAINT IF EXISTS clientes_uuid_unique;

ALTER TABLE IF EXISTS public.clientes
DROP COLUMN IF EXISTS uuid;

ALTER TABLE IF EXISTS public.proveedors
DROP CONSTRAINT IF EXISTS proveedors_uuid_unique;

ALTER TABLE IF EXISTS public.proveedors
DROP COLUMN IF EXISTS uuid;

ALTER TABLE IF EXISTS public.productos
DROP CONSTRAINT IF EXISTS productos_uuid_unique;

ALTER TABLE IF EXISTS public.productos
DROP COLUMN IF EXISTS uuid;