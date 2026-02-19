-- final_schema.sql
-- Estado final del schema después de todas las migraciones (000001–000007).
-- Crear desde cero en una BD limpia.

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Sin dependencias ──────────────────────────────────────────────────────

CREATE TABLE public.vendedors (
    uuid        uuid        NOT NULL,
    created_at  timestamptz NULL,
    updated_at  timestamptz NULL,
    deleted_at  timestamptz NULL,
    nombre      text        NULL,
    apellido    text        NULL,
    cedula      text        NULL,
    email       text        NULL,
    contrasena  text        NULL,
    mfa_secret  text        NULL,
    mfa_enabled boolean     NULL DEFAULT false,
    role        varchar(20) NOT NULL DEFAULT 'cajero',
    CONSTRAINT vendedors_pkey PRIMARY KEY (uuid),
    CONSTRAINT uni_vendedors_cedula UNIQUE (cedula),
    CONSTRAINT uni_vendedors_email  UNIQUE (email)
);
CREATE INDEX idx_vendedors_deleted_at ON public.vendedors USING btree (deleted_at);

CREATE TABLE public.clientes (
    uuid       uuid        NOT NULL,
    created_at timestamptz NULL,
    updated_at timestamptz NULL,
    deleted_at timestamptz NULL,
    nombre     text        NULL,
    apellido   text        NULL,
    tipo_id    text        NULL,
    numero_id  text        NULL,
    telefono   text        NULL,
    email      text        NULL,
    direccion  text        NULL,
    CONSTRAINT clientes_pkey       PRIMARY KEY (uuid),
    CONSTRAINT uni_clientes_numero_id UNIQUE (numero_id)
);
CREATE INDEX idx_clientes_deleted_at ON public.clientes USING btree (deleted_at);

CREATE TABLE public.productos (
    uuid         uuid    NOT NULL,
    created_at   timestamptz NULL,
    updated_at   timestamptz NULL,
    deleted_at   timestamptz NULL,
    nombre       text    NULL,
    codigo       text    NULL,
    precio_venta numeric NULL,
    stock        bigint  NULL,
    CONSTRAINT productos_pkey      PRIMARY KEY (uuid),
    CONSTRAINT uni_productos_codigo UNIQUE (codigo)
);
CREATE INDEX idx_productos_deleted_at ON public.productos USING btree (deleted_at);

CREATE TABLE public.proveedors (
    uuid       uuid NOT NULL,
    created_at timestamptz NULL,
    updated_at timestamptz NULL,
    deleted_at timestamptz NULL,
    nombre     text NULL,
    telefono   text NULL,
    email      text NULL,
    CONSTRAINT proveedors_pkey    PRIMARY KEY (uuid),
    CONSTRAINT uni_proveedors_nombre UNIQUE (nombre)
);
CREATE INDEX idx_proveedors_deleted_at ON public.proveedors USING btree (deleted_at);

-- ─── Dependen de vendedors y clientes ─────────────────────────────────────

CREATE TABLE public.facturas (
    uuid           uuid        NOT NULL,
    created_at     timestamptz NULL,
    updated_at     timestamptz NULL,
    deleted_at     timestamptz NULL,
    numero_factura text        NULL,
    fecha_emision  timestamptz NULL,
    vendedor_uuid  uuid        NULL,
    cliente_uuid   uuid        NULL,
    subtotal       numeric     NULL,
    iva            numeric     NULL,
    total          numeric     NULL,
    estado         text        NULL,
    metodo_pago    text        NULL,
    CONSTRAINT facturas_pkey                   PRIMARY KEY (uuid),
    CONSTRAINT uni_facturas_numero_factura     UNIQUE (numero_factura),
    CONSTRAINT fk_facturas_vendedor            FOREIGN KEY (vendedor_uuid) REFERENCES public.vendedors (uuid),
    CONSTRAINT fk_facturas_cliente             FOREIGN KEY (cliente_uuid)  REFERENCES public.clientes  (uuid)
);
CREATE INDEX idx_facturas_deleted_at ON public.facturas USING btree (deleted_at);

-- ─── Dependen de facturas y productos ────────────────────────────────────

CREATE TABLE public.detalle_facturas (
    uuid            uuid    NOT NULL,
    created_at      timestamptz NULL,
    updated_at      timestamptz NULL,
    deleted_at      timestamptz NULL,
    factura_uuid    uuid    NULL,
    producto_uuid   uuid    NULL,
    cantidad        bigint  NULL,
    precio_unitario numeric NULL,
    precio_total    numeric NULL,
    CONSTRAINT detalle_facturas_pkey                    PRIMARY KEY (uuid),
    CONSTRAINT detalle_facturas_factura_producto_key    UNIQUE (factura_uuid, producto_uuid),
    CONSTRAINT fk_detalle_factura_uuid                  FOREIGN KEY (factura_uuid)  REFERENCES public.facturas  (uuid) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_detalle_facturas_producto             FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid)
);
CREATE INDEX idx_detalle_facturas_deleted_at ON public.detalle_facturas USING btree (deleted_at);

-- ─── Dependen de proveedors ────────────────────────────────────────────────

CREATE TABLE public.compras (
    uuid           uuid        NOT NULL,
    created_at     timestamptz NULL,
    updated_at     timestamptz NULL,
    deleted_at     timestamptz NULL,
    fecha          timestamptz NULL,
    proveedor_uuid uuid        NULL,
    factura_numero text        NULL,
    total          numeric     NULL,
    CONSTRAINT compras_pkey          PRIMARY KEY (uuid),
    CONSTRAINT fk_compras_proveedor  FOREIGN KEY (proveedor_uuid) REFERENCES public.proveedors (uuid)
);
CREATE INDEX idx_compras_deleted_at ON public.compras USING btree (deleted_at);

CREATE TABLE public.detalle_compras (
    uuid                  uuid    NOT NULL,
    compra_uuid           uuid    NULL,
    producto_uuid         uuid    NULL,
    cantidad              bigint  NULL,
    precio_compra_unitario numeric NULL,
    CONSTRAINT detalle_compras_pkey             PRIMARY KEY (uuid),
    CONSTRAINT fk_detalle_compras_compra        FOREIGN KEY (compra_uuid)   REFERENCES public.compras   (uuid),
    CONSTRAINT fk_detalle_compras_producto      FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid)
);

-- ─── Dependen de facturas, productos y vendedors ──────────────────────────

CREATE TABLE public.operacion_stocks (
    uuid             uuid        NOT NULL,
    producto_uuid    uuid        NULL,
    tipo_operacion   text        NULL,
    cantidad_cambio  bigint      NULL,
    stock_resultante bigint      NULL,
    vendedor_uuid    uuid        NULL,
    factura_uuid     uuid        NULL,
    timestamp        timestamptz NULL,
    sincronizado     boolean     NULL DEFAULT false,
    CONSTRAINT operacion_stocks_pkey        PRIMARY KEY (uuid),
    CONSTRAINT fk_operacion_factura_uuid    FOREIGN KEY (factura_uuid)  REFERENCES public.facturas  (uuid) ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT fk_producto                  FOREIGN KEY (producto_uuid) REFERENCES public.productos (uuid) ON DELETE RESTRICT,
    CONSTRAINT fk_vendedor                  FOREIGN KEY (vendedor_uuid) REFERENCES public.vendedors (uuid) ON DELETE RESTRICT
);
CREATE INDEX idx_operacion_stocks_factura_uuid ON public.operacion_stocks USING btree (factura_uuid);

-- ─── Tabla de control de migraciones (golang-migrate) ────────────────────

CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version bigint  NOT NULL,
    dirty   boolean NOT NULL,
    CONSTRAINT schema_migrations_pkey PRIMARY KEY (version)
);

-- Marcar las 7 migraciones como aplicadas (sin ejecutarlas individualmente)
INSERT INTO public.schema_migrations (version, dirty) VALUES
    (1, false),
    (2, false),
    (3, false),
    (4, false),
    (5, false),
    (6, false),
    (7, false)
ON CONFLICT (version) DO UPDATE SET dirty = false;

COMMIT;
