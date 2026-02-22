#!/usr/bin/env bash
# reset_and_import.sh — Restaura DATABASE_URL desde otra BD o desde un archivo.
#
# ╔══════════════════════════════════════════════════════════════╗
# ║  FLUJO                                                       ║
# ║                                                              ║
# ║  DESTINO: siempre DATABASE_URL del .env                      ║
# ║                                                              ║
# ║  ORIGEN (argumento $1 / variable SOURCE):                    ║
# ║    • vacío / sin arg  → backend/python/backup_supabase.sql   ║
# ║    • ruta a archivo   → usa ese .sql directamente            ║
# ║    • DSN postgres://… → pg_dump en vivo, luego restaura      ║
# ║                                                              ║
# ╠══════════════════════════════════════════════════════════════╣
# ║  Uso desde Makefile (recomendado):                           ║
# ║    make db-reset                                             ║
# ║    make db-reset SOURCE=backup_supabase.sql                  ║
# ║    make db-reset SOURCE="postgresql://user:pass@host/db"     ║
# ║                                                              ║
# ║  Uso directo:                                                ║
# ║    bash reset_and_import.sh                                  ║
# ║    bash reset_and_import.sh /ruta/al/backup.sql              ║
# ║    bash reset_and_import.sh "postgresql://user:pass@host/db" ║
# ╚══════════════════════════════════════════════════════════════╝

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLEANED_SQL="$SCRIPT_DIR/.backup_clean.sql"
TMP_DUMP="$SCRIPT_DIR/.backup_live_dump.sql"

# ── Limpiar archivos temporales al salir (éxito o error) ──────────────────────
trap 'rm -f "$CLEANED_SQL" "$TMP_DUMP"' EXIT

# ── DATABASE_URL desde .env (strip \r para archivos con line-endings Windows) ─
if [ -f "$PROJECT_ROOT/.env" ]; then
  set -a
  source <(tr -d '\r' < "$PROJECT_ROOT/.env")
  set +a
fi
DEST_DSN="${DATABASE_URL:-postgresql://luna:tu_password_seguro@localhost:5432/farmacia_db?sslmode=disable}"

# ── Determinar origen ─────────────────────────────────────────────────────────
SOURCE_ARG="${1:-}"
MODE=""        # "file" | "live"
BACKUP_FILE=""

if [ -z "$SOURCE_ARG" ]; then
  # Sin argumento → archivo por defecto en backend/db/
  MODE="file"
  BACKUP_FILE="$PROJECT_ROOT/backend/db/backup_supabase.sql"
elif [[ "$SOURCE_ARG" == postgres://* || "$SOURCE_ARG" == postgresql://* ]]; then
  # Es un DSN → pg_dump en vivo
  MODE="live"
  BACKUP_FILE="$TMP_DUMP"
else
  # Es una ruta a archivo
  MODE="file"
  BACKUP_FILE="$SOURCE_ARG"
fi

# ── Header ────────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════"
echo " reset_and_import.sh"
echo "═══════════════════════════════════════════════════════"
if [ "$MODE" = "live" ]; then
  # Ocultar contraseña en el log
  DISPLAY_SRC=$(echo "$SOURCE_ARG" | sed 's|://[^:]*:[^@]*@|://***:***@|')
  echo "  Origen  : $DISPLAY_SRC  (pg_dump en vivo)"
else
  echo "  Origen  : $BACKUP_FILE"
fi
DISPLAY_DEST=$(echo "$DEST_DSN" | sed 's|://[^:]*:[^@]*@|://***:***@|')
echo "  Destino : $DISPLAY_DEST  (DATABASE_URL)"
echo ""

# ── Paso 1 (modo live): pg_dump desde el origen ───────────────────────────────
if [ "$MODE" = "live" ]; then
  echo "▶ [1/5] pg_dump desde origen..."
  command -v pg_dump >/dev/null 2>&1 \
    || { echo "❌  pg_dump no encontrado. Instalar: sudo apt-get install postgresql-client"; exit 1; }
  pg_dump "$SOURCE_ARG" \
    --schema=public \
    --clean \
    --if-exists \
    --file="$TMP_DUMP"
  echo "  ✓ Dump generado ($(wc -l < "$TMP_DUMP") líneas)"
else
  echo "▶ [1/5] Verificando archivo..."
  [ -f "$BACKUP_FILE" ] || {
    echo "❌  Archivo no encontrado: $BACKUP_FILE"
    echo ""
    echo "    Opciones:"
    echo "      make db-reset SOURCE=\"postgresql://user:pass@host/db\"   # dump en vivo"
    echo "      make db-reset SOURCE=/ruta/al/backup.sql                 # desde archivo"
    exit 1
  }
  echo "  ✓ $(wc -l < "$BACKUP_FILE") líneas en $(basename "$BACKUP_FILE")"
fi

# ── Paso 2: preprocesar — eliminar incompatibilidades de Supabase ─────────────
echo ""
echo "▶ [2/5] Preprocesando SQL..."

# Roles propios de Supabase que no existen en un PostgreSQL estándar.
SUPA_ROLES="anon|authenticated|service_role|supabase_admin|supabase_auth_admin|supabase_read_only_user|dashboard_user"

grep -vE \
  "OWNER TO |\
^SET transaction_timeout|\
^DROP SCHEMA |\
^CREATE SCHEMA |\
^GRANT .* TO (${SUPA_ROLES})|\
^REVOKE .* FROM (${SUPA_ROLES})|\
^ALTER DEFAULT PRIVILEGES" \
  "$BACKUP_FILE" > "$CLEANED_SQL"

ORIG=$(wc -l < "$BACKUP_FILE")
CLEAN=$(wc -l < "$CLEANED_SQL")
echo "  ✓ $((ORIG - CLEAN)) líneas eliminadas  →  $CLEAN líneas limpias"

# ── Paso 3: aplicar backup (DROP → CREATE → COPY) ────────────────────────────
echo ""
echo "▶ [3/5] Aplicando en destino..."
psql "$DEST_DSN" -v ON_ERROR_STOP=1 -f "$CLEANED_SQL"
echo "  ✓ Backup aplicado"

# ── Paso 4: asegurar tabla de migraciones (golang-migrate) ───────────────────
echo ""
echo "▶ [4/5] Actualizando schema_migrations..."
psql "$DEST_DSN" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version bigint  NOT NULL,
    dirty   boolean NOT NULL,
    CONSTRAINT schema_migrations_pkey PRIMARY KEY (version)
);
INSERT INTO public.schema_migrations (version, dirty) VALUES
    (1,false),(2,false),(3,false),(4,false),(5,false),(6,false),(7,false)
ON CONFLICT (version) DO UPDATE SET dirty = false;
SQL
echo "  ✓ Migraciones marcadas hasta versión 7"

# ── Paso 5: verificación ──────────────────────────────────────────────────────
echo ""
echo "▶ [5/5] Verificando registros en destino..."
psql "$DEST_DSN" -c "
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
