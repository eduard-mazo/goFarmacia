-- 1. Eliminar tablas "hijas" (las que tienen llaves foráneas apuntando a otras)
-- Estas tablas dependen de facturas, compras, productos, y vendedors.
DROP TABLE IF EXISTS public.detalle_facturas;

DROP TABLE IF EXISTS public.detalle_compras;

DROP TABLE IF EXISTS public.operacion_stocks;

-- 2. Eliminar tablas "intermedias"
-- Estas tablas dependen de clientes, vendedors, y proveedors.
DROP TABLE IF EXISTS public.facturas;

DROP TABLE IF EXISTS public.compras;

-- 3. Eliminar tablas "madre" o "raíz" (no tienen dependencias de llave foránea)
DROP TABLE IF EXISTS public.productos;

DROP TABLE IF EXISTS public.vendedors;

DROP TABLE IF EXISTS public.clientes;

DROP TABLE IF EXISTS public.proveedors;