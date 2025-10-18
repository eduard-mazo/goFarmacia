-- 000001_add_uuids_to_transactions.up.sql
-- 1. Habilita la extensión UUID en PostgreSQL (es idempotente)
-- Esto es necesario para usar el tipo nativo 'uuid' y la función 'uuid_generate_v4()'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Migrar la tabla 'facturas'
-- Primero añadimos la columna permitiendo nulos
ALTER TABLE public.facturas
ADD COLUMN uuid uuid;

-- Rellenamos todas las filas existentes con un UUID nuevo
UPDATE public.facturas
SET
    uuid = uuid_generate_v4 ()
WHERE
    uuid IS NULL;

-- Ahora que no hay nulos, la hacemos NOT NULL
ALTER TABLE public.facturas
ALTER COLUMN uuid
SET
    NOT NULL;

-- Finalmente, añadimos la constraint UNIQUE
ALTER TABLE public.facturas ADD CONSTRAINT facturas_uuid_unique UNIQUE (uuid);

-- 3. Migrar la tabla 'detalle_facturas'
ALTER TABLE public.detalle_facturas
ADD COLUMN uuid uuid;

UPDATE public.detalle_facturas
SET
    uuid = uuid_generate_v4 ()
WHERE
    uuid IS NULL;

ALTER TABLE public.detalle_facturas
ALTER COLUMN uuid
SET
    NOT NULL;

ALTER TABLE public.detalle_facturas ADD CONSTRAINT detalle_facturas_uuid_unique UNIQUE (uuid);

-- 4. Repetir para 'compras' (si también las modificaste)
ALTER TABLE public.compras
ADD COLUMN uuid uuid;

UPDATE public.compras
SET
    uuid = uuid_generate_v4 ()
WHERE
    uuid IS NULL;

ALTER TABLE public.compras
ALTER COLUMN uuid
SET
    NOT NULL;

ALTER TABLE public.compras ADD CONSTRAINT compras_uuid_unique UNIQUE (uuid);

-- 5. Repetir para 'detalle_compras' (si también las modificaste)
ALTER TABLE public.detalle_compras
ADD COLUMN uuid uuid;

UPDATE public.detalle_compras
SET
    uuid = uuid_generate_v4 ()
WHERE
    uuid IS NULL;

ALTER TABLE public.detalle_compras
ALTER COLUMN uuid
SET
    NOT NULL;

ALTER TABLE public.detalle_compras ADD CONSTRAINT detalle_compras_uuid_unique UNIQUE (uuid);