-- Archivo de migración inicial para la base de datos local (SQLite)
CREATE TABLE
    IF NOT EXISTS sync_log (
        model_name TEXT PRIMARY KEY,
        last_sync_timestamp DATETIME NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS vendedors (
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
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
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
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
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
        nombre TEXT UNIQUE NOT NULL,
        telefono TEXT,
        email TEXT
    );

CREATE TABLE
    IF NOT EXISTS productos (
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        nombre TEXT,
        codigo TEXT UNIQUE NOT NULL,
        precio_venta REAL,
        stock INTEGER
    );

CREATE TABLE
    IF NOT EXISTS operacion_stocks (
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
        factura_uuid TEXT,
        producto_uuid TEXT NOT NULL,
        tipo_operacion TEXT,
        cantidad_cambio INTEGER,
        stock_resultante INTEGER,
        vendedor_uuid TEXT,
        timestamp DATETIME,
        sincronizado BOOLEAN DEFAULT false,
        FOREIGN KEY (producto_uuid) REFERENCES productos (uuid)
    );

CREATE TABLE
    IF NOT EXISTS facturas (
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
        numero_factura TEXT UNIQUE NOT NULL,
        fecha_emision DATETIME,
        vendedor_uuid TEXT,
        cliente_uuid TEXT,
        subtotal REAL,
        iva REAL,
        total REAL,
        estado TEXT,
        metodo_pago TEXT,
        FOREIGN KEY (vendedor_uuid) REFERENCES vendedors (uuid),
        FOREIGN KEY (cliente_uuid) REFERENCES clientes (uuid)
    );

CREATE TABLE
    IF NOT EXISTS detalle_facturas (
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
        factura_uuid TEXT NOT NULL,
        producto_uuid TEXT NOT NULL,
        cantidad INTEGER,
        precio_unitario REAL,
        precio_total REAL,
        FOREIGN KEY (factura_uuid) REFERENCES facturas (uuid),
        FOREIGN KEY (producto_uuid) REFERENCES productos (uuid),
        UNIQUE (factura_uuid, producto_uuid)
    );

CREATE TABLE
    IF NOT EXISTS compras (
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
        fecha DATETIME,
        proveedor_uuid TEXT NOT NULL,
        factura_numero TEXT,
        total REAL,
        FOREIGN KEY (proveedor_uuid) REFERENCES proveedors (uuid)
    );

CREATE TABLE
    IF NOT EXISTS detalle_compra (
        uuid TEXT UNIQUE PRIMARY KEY NOT NULL,
        compra_uuid TEXT NOT NULL,
        producto_uuid TEXT NOT NULL,
        cantidad INTEGER,
        precio_compra_unitario REAL,
        FOREIGN KEY (compra_uuid) REFERENCES compras (uuid),
        FOREIGN KEY (producto_uuid) REFERENCES productos (uuid)
    );