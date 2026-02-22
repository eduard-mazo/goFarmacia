#!/usr/bin/env bash
# reset_and_import.sh
# Restaura la BD local desde un backup de Supabase generado con pg_dump.
#
# Generar el backup desde Supabase:
#   pg_dump "postgresql://postgres.<ref>:<pass>@aws-1-us-east-1.pooler.supabase.com:5432/postgres" \
#     --schema=public \
#     --clean \
#     --if-exists \
#     --file=backup_supabase.sql
#
# Uso:
#   bash reset_and_import.sh [ruta/al/backup.sql]
#   make db-reset                  (usa backup_supabase.sql por defecto)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLEANED_SQL="$SCRIPT_DIR/.backup_clean.sql"

# ── Limpiar archivo temporal al salir (éxito o error) ─────────────────────────
trap 'rm -f "$CLEANED_SQL"' EXIT

# ── DATABASE_URL desde .env ───────────────────────────────────────────────────
if [ -f "$PROJECT_ROOT/.env" ]; then
  set -a; source "$PROJECT_ROOT/.env"; set +a
fi
DB_URL="${DATABASE_URL:-postgresql://luna:tu_password_seguro@localhost:5432/farmacia_db?sslmode=disable}"

# ── Archivo de backup (primer argumento o valor por defecto) ──────────────────
BACKUP_FILE="${1:-$SCRIPT_DIR/backup_supabase.sql}"

echo ""
echo "═══════════════════════════════════════════════════════"
echo " reset_and_import.sh — Restauración desde Supabase"
echo "═══════════════════════════════════════════════════════"
echo "  Backup : $BACKUP_FILE"
echo "  BD     : $DB_URL"
echo ""

# ── Validar que el archivo existe ────────────────────────────────────────────
if [ ! -f "$BACKUP_FILE" ]; then
  echo "❌  Archivo no encontrado: $BACKUP_FILE"
  echo ""
  echo "    Generarlo con:"
  echo "    pg_dump \"postgresql://<user>:<pass>@<host>:5432/postgres\" \\"
  echo "      --schema=public \\"
  echo "      --clean \\"
  echo "      --if-exists \\"
  echo "      --file=$(basename "$BACKUP_FILE")"
  exit 1
fi

# ── Paso 1: preprocesar — eliminar incompatibilidades de Supabase ─────────────
echo "▶ [1/4] Preprocesando backup..."

# Roles de Supabase que no existen en PostgreSQL local estándar.
# Las líneas que los mencionan en GRANT/REVOKE se eliminan.
SUPA_ROLES="anon|authenticated|service_role|supabase_admin|supabase_auth_admin|supabase_read_only_user|dashboard_user"

grep -vE \
  "OWNER TO |\
^GRANT .* TO (${SUPA_ROLES})|\
^REVOKE .* FROM (${SUPA_ROLES})|\
^ALTER DEFAULT PRIVILEGES" \
  "$BACKUP_FILE" > "$CLEANED_SQL"

ORIG_LINES=$(wc -l < "$BACKUP_FILE")
CLEAN_LINES=$(wc -l < "$CLEANED_SQL")
echo "  ✓ $((ORIG_LINES - CLEAN_LINES)) líneas eliminadas  →  $CLEAN_LINES líneas limpias"

# ── Paso 2: aplicar backup completo (DROP → CREATE → COPY) ───────────────────
echo ""
echo "▶ [2/4] Aplicando backup en BD local..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$CLEANED_SQL"
echo "  ✓ Backup aplicado"

# ── Paso 3: asegurar tabla de migraciones (golang-migrate) ───────────────────
# La tabla schema_migrations es propia de la app y puede no estar en el dump
# de Supabase si las migraciones se gestionaron localmente. Se crea / rellena.
echo ""
echo "▶ [3/4] Actualizando tabla de migraciones..."
psql "$DB_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version bigint  NOT NULL,
    dirty   boolean NOT NULL,
    CONSTRAINT schema_migrations_pkey PRIMARY KEY (version)
);
INSERT INTO public.schema_migrations (version, dirty) VALUES
    (1, false),(2, false),(3, false),(4, false),(5, false),(6, false),(7, false)
ON CONFLICT (version) DO UPDATE SET dirty = false;
SQL
echo "  ✓ Migraciones marcadas hasta versión 7"

# ── Paso 4: verificación ──────────────────────────────────────────────────────
echo ""
echo "▶ [4/4] Verificando datos importados..."
psql "$DB_URL" -c "
SELECT tabla, registros FROM (
  SELECT 'vendedors'         AS tabla, COUNT(*) AS registros FROM vendedors
  UNION ALL SELECT 'clientes',         COUNT(*) FROM clientes
  UNION ALL SELECT 'productos',        COUNT(*) FROM productos
  UNION ALL SELECT 'proveedors',       COUNT(*) FROM proveedors
  UNION ALL SELECT 'facturas',         COUNT(*) FROM facturas
  UNION ALL SELECT 'operacion_stocks', COUNT(*) FROM operacion_stocks
  UNION ALL SELECT 'detalle_facturas', COUNT(*) FROM detalle_facturas
  UNION ALL SELECT 'compras',          COUNT(*) FROM compras
) t ORDER BY tabla;
"

echo ""
echo "═══════════════════════════════════════════════════════"
echo " ✅ IMPORTACIÓN COMPLETA"
echo "═══════════════════════════════════════════════════════"
echo ""
