-- Archivo de migración inicial para la base de datos remota (PostgreSQL)
-- Habilita la extensión para generar UUIDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE
    IF NOT EXISTS sync_log (
        model_name TEXT PRIMARY KEY,
        last_sync_timestamp TIMESTAMP
        WITH
            TIME ZONE NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS vendedors (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            deleted_at TIMESTAMP
        WITH
            TIME ZONE,
            nombre TEXT,
            apellido TEXT,
            cedula TEXT UNIQUE NOT NULL,
            email TEXT UNIQUE NOT NULL,
            contrasena TEXT,
            mfa_secret TEXT,
            mfa_enabled BOOLEAN DEFAULT false
    );

CREATE TABLE
    IF NOT EXISTS clientes (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            deleted_at TIMESTAMP
        WITH
            TIME ZONE,
            nombre TEXT UNIQUE NOT NULL,
            apellido TEXT,
            tipo_id TEXT,
            numero_id TEXT UNIQUE NOT NULL,
            telefono TEXT,
            email TEXT,
            direccion TEXT
    );

CREATE TABLE
    IF NOT EXISTS proveedors (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            deleted_at TIMESTAMP
        WITH
            TIME ZONE,
            nombre TEXT UNIQUE NOT NULL,
            telefono TEXT,
            email TEXT
    );

CREATE TABLE
    IF NOT EXISTS productos (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            deleted_at TIMESTAMP
        WITH
            TIME ZONE,
            nombre TEXT,
            codigo TEXT UNIQUE NOT NULL,
            precio_venta REAL,
            stock INTEGER
    );

CREATE TABLE
    IF NOT EXISTS operacion_stocks (
        id SERIAL PRIMARY KEY,
        uuid UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4 (),
        producto_id INTEGER NOT NULL REFERENCES productos (id),
        tipo_operacion TEXT,
        cantidad_cambio INTEGER,
        stock_resultante INTEGER,
        vendedor_id INTEGER,
        factura_id INTEGER,
        timestamp TIMESTAMP
        WITH
            TIME ZONE,
            sincronizado BOOLEAN DEFAULT false
    );

CREATE TABLE
    IF NOT EXISTS facturas (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            uuid UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4 (),
            numero_factura TEXT UNIQUE NOT NULL,
            fecha_emision TIMESTAMP
        WITH
            TIME ZONE,
            vendedor_id INTEGER REFERENCES vendedors (id),
            cliente_id INTEGER REFERENCES clientes (id),
            subtotal REAL,
            iva REAL,
            total REAL,
            estado TEXT,
            metodo_pago TEXT
    );

CREATE TABLE
    IF NOT EXISTS detalle_facturas (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            deleted_at TIMESTAMP
        WITH
            TIME ZONE,
            uuid UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4 (),
            factura_id INTEGER REFERENCES facturas (id),
            producto_id INTEGER REFERENCES productos (id),
            cantidad INTEGER,
            precio_unitario REAL,
            precio_total REAL,
            UNIQUE (factura_id, producto_id)
    );

CREATE TABLE
    IF NOT EXISTS compras (
        id SERIAL PRIMARY KEY,
        fecha TIMESTAMP
        WITH
            TIME ZONE,
            proveedor_id INTEGER REFERENCES proveedors (id),
            factura_numero TEXT,
            total REAL,
            created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS detalle_compras (
        id SERIAL PRIMARY KEY,
        compra_id INTEGER REFERENCES compras (id),
        producto_id INTEGER REFERENCES productos (id),
        cantidad INTEGER,
        precio_compra_unitario REAL
    );