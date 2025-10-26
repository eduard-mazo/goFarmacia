create table
    public.clientes (
        id bigserial not null,
        created_at timestamp
        with
            time zone null,
            updated_at timestamp
        with
            time zone null,
            deleted_at timestamp
        with
            time zone null,
            nombre text null,
            apellido text null,
            tipo_id text null,
            numero_id text null,
            telefono text null,
            email text null,
            direccion text null,
            uuid uuid not null,
            constraint clientes_pkey primary key (id),
            constraint clientes_uuid_unique unique (uuid),
            constraint uni_clientes_numero_id unique (numero_id)
    ) TABLESPACE pg_default;

create index IF not exists idx_clientes_deleted_at on public.clientes using btree (deleted_at) TABLESPACE pg_default;

create table
    public.compras (
        id bigserial not null,
        created_at timestamp
        with
            time zone null,
            updated_at timestamp
        with
            time zone null,
            deleted_at timestamp
        with
            time zone null,
            fecha timestamp
        with
            time zone null,
            proveedor_id bigint null,
            factura_numero text null,
            total numeric null,
            uuid uuid not null,
            constraint compras_pkey primary key (id),
            constraint compras_uuid_unique unique (uuid),
            constraint fk_compras_proveedor foreign KEY (proveedor_id) references proveedors (id)
    ) TABLESPACE pg_default;

create index IF not exists idx_compras_deleted_at on public.compras using btree (deleted_at) TABLESPACE pg_default;

create table
    public.detalle_compras (
        id bigserial not null,
        compra_id bigint null,
        producto_id bigint null,
        cantidad bigint null,
        precio_compra_unitario numeric null,
        uuid uuid not null,
        constraint detalle_compras_pkey primary key (id),
        constraint detalle_compras_uuid_unique unique (uuid),
        constraint fk_compras_detalles foreign KEY (compra_id) references compras (id),
        constraint fk_detalle_compras_producto foreign KEY (producto_id) references productos (id)
    ) TABLESPACE pg_default;

create table
    public.detalle_facturas (
        id bigserial not null,
        created_at timestamp
        with
            time zone null,
            updated_at timestamp
        with
            time zone null,
            deleted_at timestamp
        with
            time zone null,
            factura_id bigint null,
            producto_id bigint null,
            cantidad bigint null,
            precio_unitario numeric null,
            precio_total numeric null,
            uuid uuid not null,
            factura_uuid uuid null,
            constraint detalle_facturas_pkey primary key (id),
            constraint detalle_facturas_factura_id_producto_id_key unique (factura_id, producto_id),
            constraint detalle_facturas_uuid_unique unique (uuid),
            constraint fk_detalle_factura_uuid foreign KEY (factura_uuid) references facturas (uuid) on update CASCADE on delete CASCADE,
            constraint fk_detalle_facturas_producto foreign KEY (producto_id) references productos (id),
            constraint fk_facturas_detalles foreign KEY (factura_id) references facturas (id)
    ) TABLESPACE pg_default;

create index IF not exists idx_detalle_facturas_deleted_at on public.detalle_facturas using btree (deleted_at) TABLESPACE pg_default;

create table
    public.facturas (
        id bigserial not null,
        created_at timestamp
        with
            time zone null,
            updated_at timestamp
        with
            time zone null,
            deleted_at timestamp
        with
            time zone null,
            numero_factura text null,
            fecha_emision timestamp
        with
            time zone null,
            vendedor_id bigint null,
            cliente_id bigint null,
            subtotal numeric null,
            iva numeric null,
            total numeric null,
            estado text null,
            metodo_pago text null,
            uuid uuid not null,
            constraint facturas_pkey primary key (id),
            constraint facturas_uuid_unique unique (uuid),
            constraint uni_facturas_numero_factura unique (numero_factura),
            constraint fk_facturas_cliente foreign KEY (cliente_id) references clientes (id),
            constraint fk_facturas_vendedor foreign KEY (vendedor_id) references vendedors (id)
    ) TABLESPACE pg_default;

create index IF not exists idx_facturas_deleted_at on public.facturas using btree (deleted_at) TABLESPACE pg_default;

create table
    public.operacion_stocks (
        id bigserial not null,
        uuid text null,
        producto_id bigint null,
        tipo_operacion text null,
        cantidad_cambio bigint null,
        stock_resultante bigint null,
        vendedor_id bigint null,
        factura_id bigint null,
        timestamp timestamp
        with
            time zone null,
            sincronizado boolean null default false,
            factura_uuid uuid null,
            constraint operacion_stocks_pkey primary key (id),
            constraint fk_factura foreign KEY (factura_id) references facturas (id) on delete set null,
            constraint fk_operacion_factura_uuid foreign KEY (factura_uuid) references facturas (uuid) on update CASCADE on delete set null,
            constraint fk_producto foreign KEY (producto_id) references productos (id) on delete RESTRICT,
            constraint fk_vendedor foreign KEY (vendedor_id) references vendedors (id) on delete RESTRICT
    ) TABLESPACE pg_default;

create unique INDEX IF not exists idx_operacion_stocks_uuid on public.operacion_stocks using btree (uuid) TABLESPACE pg_default;

create index IF not exists idx_operacion_stocks_factura_uuid on public.operacion_stocks using btree (factura_uuid) TABLESPACE pg_default;

create table
    public.productos (
        id bigserial not null,
        created_at timestamp
        with
            time zone null,
            updated_at timestamp
        with
            time zone null,
            deleted_at timestamp
        with
            time zone null,
            nombre text null,
            codigo text null,
            precio_venta numeric null,
            stock bigint null,
            uuid uuid not null,
            constraint productos_pkey primary key (id),
            constraint productos_uuid_unique unique (uuid),
            constraint uni_productos_codigo unique (codigo)
    ) TABLESPACE pg_default;

create index IF not exists idx_productos_deleted_at on public.productos using btree (deleted_at) TABLESPACE pg_default;

create table
    public.proveedors (
        id bigserial not null,
        created_at timestamp
        with
            time zone null,
            updated_at timestamp
        with
            time zone null,
            deleted_at timestamp
        with
            time zone null,
            nombre text null,
            telefono text null,
            email text null,
            uuid uuid not null,
            constraint proveedors_pkey primary key (id),
            constraint proveedors_uuid_unique unique (uuid),
            constraint uni_proveedors_nombre unique (nombre)
    ) TABLESPACE pg_default;

create index IF not exists idx_proveedors_deleted_at on public.proveedors using btree (deleted_at) TABLESPACE pg_default;

create table
    public.vendedors (
        id bigserial not null,
        created_at timestamp
        with
            time zone null,
            updated_at timestamp
        with
            time zone null,
            deleted_at timestamp
        with
            time zone null,
            nombre text null,
            apellido text null,
            cedula text null,
            email text null,
            contrasena text null,
            mfa_secret text null,
            mfa_enabled boolean null default false,
            uuid uuid not null,
            constraint vendedors_pkey primary key (id),
            constraint uni_vendedors_cedula unique (cedula),
            constraint uni_vendedors_email unique (email),
            constraint vendedors_uuid_unique unique (uuid)
    ) TABLESPACE pg_default;

create index IF not exists idx_vendedors_deleted_at on public.vendedors using btree (deleted_at) TABLESPACE pg_default;