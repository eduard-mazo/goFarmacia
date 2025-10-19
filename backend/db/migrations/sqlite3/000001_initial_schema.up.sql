-- Archivo de migración inicial para la base de datos local (SQLite)
CREATE TABLE
    IF NOT EXISTS sync_log (
        model_name TEXT PRIMARY KEY,
        last_sync_timestamp DATETIME NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS vendedors (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
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
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
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
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        nombre TEXT UNIQUE NOT NULL,
        telefono TEXT,
        email TEXT
    );

CREATE TABLE
    IF NOT EXISTS productos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
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
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        uuid TEXT UNIQUE NOT NULL,
        factura_uuid TEXT,
        producto_id INTEGER NOT NULL,
        tipo_operacion TEXT,
        cantidad_cambio INTEGER,
        stock_resultante INTEGER,
        vendedor_id INTEGER,
        factura_id INTEGER,
        timestamp DATETIME,
        sincronizado BOOLEAN DEFAULT false,
        FOREIGN KEY (producto_id) REFERENCES productos (id)
    );

CREATE TABLE
    IF NOT EXISTS facturas (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        uuid TEXT UNIQUE NOT NULL,
        numero_factura TEXT UNIQUE NOT NULL,
        fecha_emision DATETIME,
        vendedor_id INTEGER,
        cliente_id INTEGER,
        subtotal REAL,
        iva REAL,
        total REAL,
        estado TEXT,
        metodo_pago TEXT,
        FOREIGN KEY (vendedor_id) REFERENCES vendedors (id),
        FOREIGN KEY (cliente_id) REFERENCES clientes (id)
    );

CREATE TABLE
    IF NOT EXISTS detalle_facturas (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        deleted_at DATETIME,
        uuid TEXT UNIQUE NOT NULL,
        factura_uuid TEXT NOT NULL,
        factura_id INTEGER,
        producto_id INTEGER,
        cantidad INTEGER,
        precio_unitario REAL,
        precio_total REAL,
        FOREIGN KEY (factura_id) REFERENCES facturas (id),
        FOREIGN KEY (producto_id) REFERENCES productos (id),
        UNIQUE (factura_id, producto_id)
    );

CREATE TABLE
    IF NOT EXISTS compras (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        fecha DATETIME,
        proveedor_id INTEGER,
        factura_numero TEXT,
        total REAL,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        FOREIGN KEY (proveedor_id) REFERENCES proveedors (id)
    );

CREATE TABLE
    IF NOT EXISTS detalle_compras (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        compra_id INTEGER,
        producto_id INTEGER,
        cantidad INTEGER,
        precio_compra_unitario REAL,
        FOREIGN KEY (compra_id) REFERENCES compras (id),
        FOREIGN KEY (producto_id) REFERENCES productos (id)
    );