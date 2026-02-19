"""
migrate.py — Genera archivos SQL listos para importar en la BD actual (UUID-only schema).

Uso:
  1. Edita el bloque CONFIG de abajo con los UUIDs reales de vendedors/clientes.
  2. python3 migrate.py
  3. Revisa los archivos .sql generados.
  4. Aplícalos en orden con psql:
       psql $DATABASE_URL -f 01_productos.sql
       psql $DATABASE_URL -f 02_operacion_stocks.sql
       psql $DATABASE_URL -f 03_facturas.sql
       psql $DATABASE_URL -f 04_detalle_facturas.sql
       psql $DATABASE_URL -f 05_recalcular_stock.sql

IMPORTANTE: Ejecuta las migraciones de golang (migrate up) ANTES de aplicar estos SQL.
"""

import csv
import os
import sys
from datetime import datetime, timezone

# ─────────────────────────────────────────────────────────────────────────────
# CONFIG — Edita este bloque antes de ejecutar el script
# ─────────────────────────────────────────────────────────────────────────────
#
# Mapeo de IDs enteros del sistema antiguo → UUIDs del sistema nuevo.
# Para obtener los UUIDs actuales de tu BD:
#   SELECT uuid, nombre FROM vendedors WHERE deleted_at IS NULL;
#   SELECT uuid, nombre || ' ' || apellido, numero_id FROM clientes WHERE deleted_at IS NULL;
#
VENDEDOR_ID_MAP = {
    "1": "REEMPLAZA-CON-UUID-DE-VENDEDOR-1",   # ej: "a1b2c3d4-..."
    "3": "REEMPLAZA-CON-UUID-DE-VENDEDOR-3",   # ej: "e5f6a7b8-..."
}

CLIENTE_ID_MAP = {
    "1": "REEMPLAZA-CON-UUID-DE-CLIENTE-1",    # cliente general
    "3": "REEMPLAZA-CON-UUID-DE-CLIENTE-3",
    "4": "REEMPLAZA-CON-UUID-DE-CLIENTE-4",
}

# Si un vendedor_id o cliente_id no está en el mapa, usa este UUID de respaldo.
# Debe ser un UUID válido que exista en tu tabla de vendedors / clientes.
DEFAULT_VENDEDOR_UUID = "REEMPLAZA-CON-UUID-VENDEDOR-POR-DEFECTO"
DEFAULT_CLIENTE_UUID  = "REEMPLAZA-CON-UUID-CLIENTE-POR-DEFECTO"
# ─────────────────────────────────────────────────────────────────────────────


PRODUCTOS_FILE       = "productos_rows.csv"
FACTURAS_FILE        = "facturas_rows.csv"
DETALLE_FILE         = "detalle_facturas_rows.csv"
OPERACIONES_FILE     = "operacion_stocks_rows.csv"


def sql_str(v):
    """Escapa un valor para SQL. None → NULL."""
    if v is None or v == "":
        return "NULL"
    v = str(v).replace("'", "''")
    return f"'{v}'"


def sql_num(v, default="0"):
    if v is None or v == "":
        return default
    try:
        f = float(v)
        return str(int(f)) if f == int(f) else str(f)
    except (ValueError, TypeError):
        return default


def sql_bool(v):
    if str(v).lower() in ("true", "t", "1", "yes"):
        return "TRUE"
    return "FALSE"


def validate_config():
    """Avisa si el usuario no editó el CONFIG."""
    placeholders = [v for v in list(VENDEDOR_ID_MAP.values()) + list(CLIENTE_ID_MAP.values())
                    + [DEFAULT_VENDEDOR_UUID, DEFAULT_CLIENTE_UUID]
                    if v.startswith("REEMPLAZA")]
    if placeholders:
        print("⚠️  ATENCIÓN: Edita el bloque CONFIG en migrate.py con los UUIDs reales.")
        print("   Puedes obtenerlos con:")
        print("     SELECT uuid, nombre FROM vendedors WHERE deleted_at IS NULL;")
        print("     SELECT uuid, nombre, apellido, numero_id FROM clientes WHERE deleted_at IS NULL;")
        print()
        print("   El script continuará y generará los archivos SQL, pero los UUIDs")
        print("   de vendedor/cliente estarán como placeholders — reemplázalos antes")
        print("   de ejecutar el SQL en la base de datos.")
        print()


def load_productos():
    """Devuelve (id→uuid_map, lista de filas limpias)."""
    id_to_uuid = {}
    rows = []
    with open(PRODUCTOS_FILE, encoding="utf-8") as f:
        for row in csv.DictReader(f):
            if not row.get("uuid"):
                continue
            id_to_uuid[row["id"]] = row["uuid"]
            rows.append(row)
    print(f"  Productos leídos: {len(rows)}")
    return id_to_uuid, rows


def gen_01_productos(rows):
    lines = [
        "-- ════════════════════════════════════════════════════════",
        "-- 01_productos.sql",
        "-- Importa el catálogo de productos (1075 productos reales).",
        "-- ON CONFLICT DO NOTHING → seguro re-ejecutar.",
        "-- ════════════════════════════════════════════════════════",
        "",
        "BEGIN;",
        "",
    ]
    for r in rows:
        deleted = sql_str(r.get("deleted_at") or None)
        lines.append(
            f"INSERT INTO productos (uuid, nombre, codigo, precio_venta, stock, created_at, updated_at, deleted_at) "
            f"VALUES ("
            f"{sql_str(r['uuid'])}, "
            f"{sql_str(r.get('nombre'))}, "
            f"{sql_str(r.get('codigo'))}, "
            f"{sql_num(r.get('precio_venta'))}, "
            f"{sql_num(r.get('stock'))}, "
            f"{sql_str(r.get('created_at') or 'now()')}, "
            f"{sql_str(r.get('updated_at') or 'now()')}, "
            f"{deleted}"
            f") ON CONFLICT (uuid) DO NOTHING;"
        )
    lines += ["", "COMMIT;", ""]
    return "\n".join(lines)


def gen_02_operacion_stocks(rows, prod_id_map):
    """
    Mapea producto_id (int) → producto_uuid.
    cantidad_cambio ya viene negativo para VENTAs en los datos históricos.
    """
    lines = [
        "-- ════════════════════════════════════════════════════════",
        "-- 02_operacion_stocks.sql",
        "-- Historial de movimientos de stock (2762 registros).",
        "-- ON CONFLICT DO NOTHING → seguro re-ejecutar.",
        "-- ════════════════════════════════════════════════════════",
        "",
        "BEGIN;",
        "",
    ]
    skipped = 0
    for r in rows:
        prod_uuid = prod_id_map.get(r.get("producto_id", ""))
        if not prod_uuid:
            skipped += 1
            continue

        # vendedor_uuid puede ser NULL (no teníamos el CSV de vendedors)
        vend_uuid = VENDEDOR_ID_MAP.get(r.get("vendedor_id", ""), DEFAULT_VENDEDOR_UUID)
        if vend_uuid.startswith("REEMPLAZA"):
            vend_uuid_sql = "NULL"
        else:
            vend_uuid_sql = sql_str(vend_uuid)

        factura_uuid = r.get("factura_uuid") or None

        lines.append(
            f"INSERT INTO operacion_stocks "
            f"(uuid, producto_uuid, tipo_operacion, cantidad_cambio, stock_resultante, "
            f"vendedor_uuid, factura_uuid, timestamp, sincronizado) VALUES ("
            f"{sql_str(r['uuid'])}, "
            f"{sql_str(prod_uuid)}, "
            f"{sql_str(r.get('tipo_operacion'))}, "
            f"{sql_num(r.get('cantidad_cambio'))}, "
            f"{sql_num(r.get('stock_resultante'))}, "
            f"{vend_uuid_sql}, "
            f"{sql_str(factura_uuid)}, "
            f"{sql_str(r.get('timestamp') or 'now()')}, "
            f"{sql_bool(r.get('sincronizado', 'false'))}"
            f") ON CONFLICT (uuid) DO NOTHING;"
        )
    lines += ["", "COMMIT;", ""]
    if skipped:
        print(f"  ⚠️  operacion_stocks: {skipped} filas omitidas (producto_id sin UUID)")
    return "\n".join(lines)


def gen_03_facturas(rows):
    """
    Mapea vendedor_id y cliente_id → UUID usando VENDEDOR_ID_MAP / CLIENTE_ID_MAP.
    """
    lines = [
        "-- ════════════════════════════════════════════════════════",
        "-- 03_facturas.sql",
        "-- Historial de facturas (738 registros).",
        "-- ⚠ Requiere que 02_operacion_stocks.sql ya esté aplicado si",
        "--   operacion_stocks tiene FK a facturas (ON DELETE SET NULL).",
        "-- ON CONFLICT DO NOTHING → seguro re-ejecutar.",
        "-- ════════════════════════════════════════════════════════",
        "",
        "BEGIN;",
        "",
    ]
    for r in rows:
        vend_id = r.get("vendedor_id", "")
        cli_id  = r.get("cliente_id", "")
        vend_uuid = VENDEDOR_ID_MAP.get(vend_id, DEFAULT_VENDEDOR_UUID)
        cli_uuid  = CLIENTE_ID_MAP.get(cli_id,  DEFAULT_CLIENTE_UUID)

        deleted = sql_str(r.get("deleted_at") or None)
        lines.append(
            f"INSERT INTO facturas "
            f"(uuid, numero_factura, fecha_emision, vendedor_uuid, cliente_uuid, "
            f"subtotal, iva, total, estado, metodo_pago, created_at, updated_at, deleted_at) VALUES ("
            f"{sql_str(r['uuid'])}, "
            f"{sql_str(r.get('numero_factura'))}, "
            f"{sql_str(r.get('fecha_emision') or 'now()')}, "
            f"{sql_str(vend_uuid)}, "
            f"{sql_str(cli_uuid)}, "
            f"{sql_num(r.get('subtotal'))}, "
            f"{sql_num(r.get('iva'))}, "
            f"{sql_num(r.get('total'))}, "
            f"{sql_str(r.get('estado') or 'PAGADA')}, "
            f"{sql_str(r.get('metodo_pago') or 'efectivo')}, "
            f"{sql_str(r.get('created_at') or 'now()')}, "
            f"{sql_str(r.get('updated_at') or 'now()')}, "
            f"{deleted}"
            f") ON CONFLICT (uuid) DO NOTHING;"
        )
    lines += ["", "COMMIT;", ""]
    return "\n".join(lines)


def gen_04_detalle_facturas(rows, prod_id_map):
    """
    Mapea producto_id (int) → producto_uuid usando el mapa de productos.
    factura_uuid ya existe como columna en el CSV.
    """
    lines = [
        "-- ════════════════════════════════════════════════════════",
        "-- 04_detalle_facturas.sql",
        "-- Detalle de líneas de factura (1225 registros).",
        "-- Requiere 03_facturas.sql aplicado primero.",
        "-- ON CONFLICT DO NOTHING → seguro re-ejecutar.",
        "-- ════════════════════════════════════════════════════════",
        "",
        "BEGIN;",
        "",
    ]
    skipped = 0
    for r in rows:
        prod_uuid    = prod_id_map.get(r.get("producto_id", ""))
        factura_uuid = r.get("factura_uuid") or r.get("factura_id")  # prefer factura_uuid column
        if not prod_uuid or not factura_uuid:
            skipped += 1
            continue

        deleted = sql_str(r.get("deleted_at") or None)
        lines.append(
            f"INSERT INTO detalle_facturas "
            f"(uuid, factura_uuid, producto_uuid, cantidad, precio_unitario, precio_total, "
            f"created_at, updated_at, deleted_at) VALUES ("
            f"{sql_str(r['uuid'])}, "
            f"{sql_str(factura_uuid)}, "
            f"{sql_str(prod_uuid)}, "
            f"{sql_num(r.get('cantidad'), '1')}, "
            f"{sql_num(r.get('precio_unitario'))}, "
            f"{sql_num(r.get('precio_total'))}, "
            f"{sql_str(r.get('created_at') or 'now()')}, "
            f"{sql_str(r.get('updated_at') or 'now()')}, "
            f"{deleted}"
            f") ON CONFLICT (uuid) DO NOTHING;"
        )
    lines += ["", "COMMIT;", ""]
    if skipped:
        print(f"  ⚠️  detalle_facturas: {skipped} filas omitidas (producto o factura sin UUID)")
    return "\n".join(lines)


def gen_05_recalcular_stock():
    """
    Recalcula el stock de TODOS los productos desde operacion_stocks como fuente de verdad.
    Ejecutar siempre al final.
    """
    return """\
-- ════════════════════════════════════════════════════════
-- 05_recalcular_stock.sql
-- Recalcula stock de todos los productos desde operacion_stocks.
-- Ejecutar SIEMPRE como paso final.
-- ════════════════════════════════════════════════════════

BEGIN;

UPDATE productos p
SET stock = COALESCE(
    (SELECT SUM(os.cantidad_cambio)
     FROM operacion_stocks os
     WHERE os.producto_uuid = p.uuid),
    0
);

COMMIT;

-- Verificación rápida: productos con stock negativo (indica datos inconsistentes)
SELECT uuid, codigo, nombre, stock
FROM productos
WHERE stock < 0
ORDER BY stock ASC
LIMIT 20;
"""


def write_file(name, content):
    with open(name, "w", encoding="utf-8") as f:
        f.write(content)
    kb = len(content) / 1024
    print(f"  ✅ {name} ({kb:.1f} KB)")


def main():
    print("═══════════════════════════════════════════════")
    print(" migrate.py — Generador de SQL para migración")
    print("═══════════════════════════════════════════════")
    print()
    validate_config()

    print("📂 Leyendo archivos CSV...")
    prod_id_map, prod_rows = load_productos()

    with open(FACTURAS_FILE, encoding="utf-8") as f:
        factura_rows = list(csv.DictReader(f))
    print(f"  Facturas leídas: {len(factura_rows)}")

    with open(DETALLE_FILE, encoding="utf-8") as f:
        detalle_rows = list(csv.DictReader(f))
    print(f"  Detalles leídos: {len(detalle_rows)}")

    with open(OPERACIONES_FILE, encoding="utf-8") as f:
        op_rows = list(csv.DictReader(f))
    print(f"  Operaciones leídas: {len(op_rows)}")

    print()
    print("⚙️  Generando archivos SQL...")
    write_file("01_productos.sql",         gen_01_productos(prod_rows))
    write_file("02_operacion_stocks.sql",  gen_02_operacion_stocks(op_rows, prod_id_map))
    write_file("03_facturas.sql",          gen_03_facturas(factura_rows))
    write_file("04_detalle_facturas.sql",  gen_04_detalle_facturas(detalle_rows, prod_id_map))
    write_file("05_recalcular_stock.sql",  gen_05_recalcular_stock())

    print()
    print("═══════════════════════════════════════════════")
    print(" ✅ ARCHIVOS GENERADOS")
    print("═══════════════════════════════════════════════")
    print()
    print("PASOS A SEGUIR:")
    print()
    print(" 1. Obtén los UUIDs reales de tu BD:")
    print("      psql $DATABASE_URL -c \"SELECT uuid, nombre FROM vendedors WHERE deleted_at IS NULL;\"")
    print("      psql $DATABASE_URL -c \"SELECT uuid, nombre, apellido, numero_id FROM clientes WHERE deleted_at IS NULL;\"")
    print()
    print(" 2. Edita el bloque CONFIG en migrate.py con esos UUIDs y vuelve a ejecutar.")
    print()
    print(" 3. Aplica los SQL EN ORDEN:")
    print("      psql $DATABASE_URL -f 01_productos.sql")
    print("      psql $DATABASE_URL -f 02_operacion_stocks.sql")
    print("      psql $DATABASE_URL -f 03_facturas.sql")
    print("      psql $DATABASE_URL -f 04_detalle_facturas.sql")
    print("      psql $DATABASE_URL -f 05_recalcular_stock.sql")
    print()
    print(" 4. Verifica la migración:")
    print("      psql $DATABASE_URL -c \"SELECT COUNT(*) FROM productos;\"   -- debe ser ~1075")
    print("      psql $DATABASE_URL -c \"SELECT COUNT(*) FROM facturas;\"    -- debe ser ~738")
    print("      psql $DATABASE_URL -c \"SELECT COUNT(*) FROM operacion_stocks;\" -- ~2762")
    print()
    print(" NOTAS:")
    print("   • Los INSERT usan ON CONFLICT DO NOTHING → re-ejecutar es seguro.")
    print("   • 05_recalcular_stock.sql siempre debe ser el último paso.")
    print("   • Los datos nuevos (creados después de la exportación) no se sobreescriben.")


if __name__ == "__main__":
    main()
