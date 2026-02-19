#!/usr/bin/env bash
# reset_and_import.sh
# Borra y repuebla la BD local desde los backups de Supabase.
#
# Requisitos:
#   - psql, migrate (golang-migrate) instalados en el PATH
#   - python3 disponible
#   - Ejecutar desde el directorio backend/python/
#
# Uso:
#   cd backend/python
#   bash reset_and_import.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_URL="postgresql://luna:tu_password_seguro@localhost:5432/farmacia_db?sslmode=disable"

echo ""
echo "═══════════════════════════════════════════════════════"
echo " reset_and_import.sh — Restauración desde backup"
echo "═══════════════════════════════════════════════════════"
echo ""

# ── Paso 1: vaciar schema ──────────────────────────────────
echo "▶ [1/5] Borrando schema público..."
psql "$DB_URL" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO PUBLIC;"
echo "  ✓ Schema vaciado"

# ── Paso 2: re-crear tablas con schema final ──────────────
echo ""
echo "▶ [2/5] Creando schema final..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$SCRIPT_DIR/final_schema.sql"
echo "  ✓ Schema creado (migraciones marcadas como aplicadas)"

# ── Paso 3: preprocesar SQL (quitar columnas extras) ──────
echo ""
echo "▶ [3/5] Preprocesando archivos SQL..."
python3 "$SCRIPT_DIR/preprocess.py"

# ── Paso 4: importar en orden de FK ──────────────────────
echo ""
echo "▶ [4/5] Importando datos..."

echo "  → vendedors..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$SCRIPT_DIR/vendedors_rows_clean.sql"

echo "  → clientes..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$SCRIPT_DIR/clientes_rows_clean.sql"

echo "  → productos..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$SCRIPT_DIR/productos_rows_clean.sql"

echo "  → facturas..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$SCRIPT_DIR/facturas_rows_clean.sql"

echo "  → operacion_stocks..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$SCRIPT_DIR/operacion_stocks_rows_clean.sql"

echo "  → detalle_facturas..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$SCRIPT_DIR/detalle_facturas_rows_clean.sql"

echo "  ✓ Datos importados"

# ── Paso 5: recalcular stock ──────────────────────────────
echo ""
echo "▶ [5/5] Recalculando stock desde operacion_stocks..."
psql "$DB_URL" -v ON_ERROR_STOP=1 <<'SQL'
BEGIN;
UPDATE productos p
SET stock = COALESCE(
    (SELECT SUM(os.cantidad_cambio)
     FROM operacion_stocks os
     WHERE os.producto_uuid = p.uuid),
    0
);
COMMIT;
SQL
echo "  ✓ Stock recalculado"

# ── Verificación final ────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════"
echo " ✅ IMPORTACIÓN COMPLETA"
echo "═══════════════════════════════════════════════════════"
psql "$DB_URL" -c "
SELECT 'vendedors'       AS tabla, COUNT(*) AS registros FROM vendedors
UNION ALL SELECT 'clientes',       COUNT(*) FROM clientes
UNION ALL SELECT 'productos',      COUNT(*) FROM productos
UNION ALL SELECT 'facturas',       COUNT(*) FROM facturas
UNION ALL SELECT 'operacion_stocks', COUNT(*) FROM operacion_stocks
UNION ALL SELECT 'detalle_facturas', COUNT(*) FROM detalle_facturas
ORDER BY tabla;
"
