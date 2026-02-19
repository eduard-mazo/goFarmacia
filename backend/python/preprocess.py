"""
preprocess.py — Remueve columnas inexistentes en el schema local de los SQL de Supabase.
Genera archivos *_clean.sql listos para importar.
"""

import re
import os

# Columnas que Supabase exportó pero NO existen en el schema local
STRIP_MAP = {
    "vendedors_rows.sql": {"reset_password_token", "reset_password_expires"},
}


def parse_values(vals_str):
    """
    Parsea cadena de tuplas SQL: (v1, v2), (v1, v2), ...
    Devuelve lista de listas de strings (valores crudos, sin parsear).
    Maneja: strings con '', null, números, paréntesis anidados.
    """
    tuples = []
    i = 0
    n = len(vals_str)

    while i < n:
        # Avanzar hasta el siguiente '('
        while i < n and vals_str[i] != '(':
            i += 1
        if i >= n:
            break
        i += 1  # saltar '('

        # Leer contenido de la tupla
        vals = []
        cur = ""
        in_str = False
        depth = 0  # paréntesis anidados dentro de la tupla

        while i < n:
            ch = vals_str[i]

            if ch == "'" and not in_str:
                in_str = True
                cur += ch
            elif ch == "'" and in_str:
                # '' = comilla escapada dentro del string
                if i + 1 < n and vals_str[i + 1] == "'":
                    cur += "''"
                    i += 2
                    continue
                in_str = False
                cur += ch
            elif in_str:
                cur += ch
            elif ch == '(':
                depth += 1
                cur += ch
            elif ch == ')' and depth > 0:
                depth -= 1
                cur += ch
            elif ch == ')' and depth == 0:
                # Fin de la tupla
                vals.append(cur.strip())
                tuples.append(vals)
                i += 1
                break
            elif ch == ',' and depth == 0:
                # Separador entre valores de esta tupla
                vals.append(cur.strip())
                cur = ""
            else:
                cur += ch

            i += 1

    return tuples


def rewrite_insert(line, strip_cols):
    """
    Toma una línea INSERT de Supabase y devuelve la misma sin las columnas en strip_cols.
    Retorna None si no hay nada que cambiar.
    """
    col_match = re.search(r'\(([^)]+)\)\s+VALUES', line)
    if not col_match:
        return None

    raw_cols = col_match.group(1)
    cols = [c.strip().strip('"').strip("'") for c in raw_cols.split(",")]
    keep_idx = [i for i, c in enumerate(cols) if c not in strip_cols]

    if len(keep_idx) == len(cols):
        return None  # nada que quitar

    # Reconstruir columnas
    new_cols_str = ", ".join(f'"{cols[i]}"' for i in keep_idx)

    # Tabla
    tbl_match = re.search(r'INSERT INTO (\S+)', line)
    table = tbl_match.group(1)

    # Extraer VALUES
    vals_start = line.index(" VALUES ") + len(" VALUES ")
    vals_raw = line[vals_start:].rstrip(";").strip()

    tuples = parse_values(vals_raw)
    if not tuples:
        return None

    new_tuples = []
    for tup in tuples:
        if len(tup) != len(cols):
            raise ValueError(
                f"Tupla tiene {len(tup)} valores pero se esperaban {len(cols)}.\n"
                f"Cols: {cols}\nTupla: {tup[:5]}..."
            )
        filtered = ", ".join(tup[i] for i in keep_idx)
        new_tuples.append(f"({filtered})")

    return (
        f"INSERT INTO {table} ({new_cols_str}) VALUES "
        + ", ".join(new_tuples)
        + " ON CONFLICT (uuid) DO NOTHING;"
    )


def process_file(src, dst, strip_cols=None):
    """Lee src, aplica reescritura si hay strip_cols, escribe dst."""
    with open(src, encoding="utf-8") as f:
        lines = f.read().splitlines()

    out = []
    rewritten = 0

    for line in lines:
        line = line.strip()
        if not line:
            continue

        if strip_cols and line.upper().startswith("INSERT INTO"):
            new_line = rewrite_insert(line, strip_cols)
            if new_line:
                out.append(new_line)
                rewritten += 1
                continue

        # Si el INSERT no tiene ON CONFLICT, añadirlo
        if line.upper().startswith("INSERT INTO") and "ON CONFLICT" not in line.upper():
            line = line.rstrip(";") + " ON CONFLICT (uuid) DO NOTHING;"

        out.append(line)

    with open(dst, "w", encoding="utf-8") as f:
        f.write("\n".join(out) + "\n")

    note = f"{rewritten} reescritos" if rewritten else "ON CONFLICT añadido"
    print(f"  ✓ {dst}  ({note})")


def main():
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    print("▶ Preprocesando archivos SQL de Supabase...")

    all_files = [
        "vendedors_rows.sql",
        "clientes_rows.sql",
        "productos_rows.sql",
        "facturas_rows.sql",
        "operacion_stocks_rows.sql",
        "detalle_facturas_rows.sql",
    ]

    for src in all_files:
        if not os.path.exists(src):
            print(f"  ⚠  {src} no encontrado, omitiendo")
            continue
        dst = src.replace(".sql", "_clean.sql")
        strip_cols = STRIP_MAP.get(src, None)
        process_file(src, dst, strip_cols)

    print("\n✅ Preprocesamiento listo. Archivos _clean.sql listos para importar.")


if __name__ == "__main__":
    main()
