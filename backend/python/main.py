import csv
import uuid
import os
from collections import defaultdict

# --- Nombres de los archivos de entrada ---
PRODUCTOS_FILE = 'productos_rows.csv'
FACTURAS_FILE = 'facturas_rows.csv'
OPERACIONES_FILE = 'operacion_stocks_rows.csv'
# (No necesitamos leer detalle_facturas.csv, ya que la fuente de verdad es 'operacion_stocks')

# --- Nombres de los archivos de salida ---
NUEVAS_FACTURAS_FILE = 'facturas_a_importar.csv'
NUEVOS_DETALLES_FILE = 'detalle_facturas_a_importar.csv'

def cargar_productos():
    """Carga los productos en un diccionario para búsqueda rápida de precios."""
    productos_map = {}
    if not os.path.exists(PRODUCTOS_FILE):
        print(f"Error: No se encontró el archivo '{PRODUCTOS_FILE}'.")
        return None
        
    try:
        with open(PRODUCTOS_FILE, mode='r', encoding='utf-8') as f:
            reader = csv.DictReader(f)
            for row in reader:
                try:
                    # Guardamos el precio_venta y el UUID del producto
                    productos_map[row['id']] = {
                        'precio_venta': float(row['precio_venta']),
                        'uuid': row['uuid'] 
                    }
                except ValueError:
                    print(f"Advertencia: Precio inválido para producto ID {row['id']}. Omitiendo.")
                except KeyError:
                    print(f"Error: 'precio_venta' o 'id' no encontrado en {PRODUCTOS_FILE}. Revisa las cabeceras.")
                    return None
        print(f"✅ Productos cargados: {len(productos_map)} artículos.")
        return productos_map
    except Exception as e:
        print(f"Error fatal al leer {PRODUCTOS_FILE}: {e}")
        return None

def cargar_facturas_existentes():
    """Carga todos los UUID de facturas existentes en un set para alta velocidad."""
    facturas_existentes = set()
    if not os.path.exists(FACTURAS_FILE):
        print(f"Error: No se encontró el archivo '{FACTURAS_FILE}'.")
        return None
        
    try:
        with open(FACTURAS_FILE, mode='r', encoding='utf-8') as f:
            reader = csv.DictReader(f)
            for row in reader:
                if 'uuid' in row and row['uuid']:
                    facturas_existentes.add(row['uuid'])
        print(f"✅ Facturas existentes cargadas: {len(facturas_existentes)} facturas.")
        return facturas_existentes
    except Exception as e:
        print(f"Error fatal al leer {FACTURAS_FILE}: {e}")
        return None

def procesar_operaciones(productos_map, facturas_existentes):
    """
    Procesa las operaciones de stock y genera los datos faltantes.
    """
    if not os.path.exists(OPERACIONES_FILE):
        print(f"Error: No se encontró el archivo '{OPERACIONES_FILE}'.")
        return None, None

    facturas_a_crear = {}
    detalles_a_crear = []
    
    # ID de cliente por defecto. Ajusta si tienes un ID de "Cliente Genérico".
    CLIENTE_GENERICO_ID = 1 
    # Vendedor por defecto (admin/sistema).
    VENDEDOR_GENERICO_ID = 1 

    print("ℹ️  Procesando operaciones de stock para encontrar inconsistencias...")
    
    try:
        with open(OPERACIONES_FILE, mode='r', encoding='utf-8') as f:
            reader = csv.DictReader(f)
            
            for i, op_row in enumerate(reader):
                tipo_op = op_row.get('tipo_operacion', '')
                factura_uuid = op_row.get('factura_uuid', '')

                # --- 1. Filtrar solo operaciones relevantes ---
                if tipo_op != 'VENTA' or not factura_uuid:
                    continue

                # --- 2. La lógica clave: ¿Falta esta factura? ---
                if factura_uuid in facturas_existentes:
                    continue
                    
                # --- 3. ¡Inconsistencia encontrada! ---
                # Esta factura_uuid está en stocks pero no en facturas.
                
                try:
                    producto_id = op_row['producto_id']
                    # cantidad_cambio es negativo para VENTAS, lo queremos positivo
                    cantidad = abs(int(float(op_row['cantidad_cambio']))) 
                    timestamp = op_row.get('timestamp', 'now()') # Usamos el timestamp de la operación
                    vendedor_id = op_row.get('vendedor_id', VENDEDOR_GENERICO_ID) or VENDEDOR_GENERICO_ID
                except (ValueError, TypeError, KeyError) as e:
                    print(f"Advertencia: Fila de operación (línea {i+2}) inválida. Error: {e}. Omitiendo: {op_row}")
                    continue

                # --- 4. Obtener datos del producto (precio) ---
                producto_info = productos_map.get(producto_id)
                if not producto_info:
                    print(f"Advertencia: Producto ID {producto_id} (de operación {op_row['uuid']}) no encontrado en productos.csv. Omitiendo detalle.")
                    continue
                
                precio_unitario = producto_info['precio_venta']
                precio_total = precio_unitario * cantidad
                
                # --- 5. Crear el nuevo 'detalle_factura' ---
                nuevo_detalle = {
                    'uuid': str(uuid.uuid4()),
                    'factura_uuid': factura_uuid,
                    'producto_id': producto_id,
                    'cantidad': cantidad,
                    'precio_unitario': precio_unitario,
                    'precio_total': precio_total,
                    'created_at': timestamp,
                    'updated_at': timestamp,
                    'factura_id': None, # Lo dejamos nulo, la FK es con factura_uuid
                    'deleted_at': None
                }
                detalles_a_crear.append(nuevo_detalle)

                # --- 6. Crear o actualizar la 'factura' a crear ---
                if factura_uuid not in facturas_a_crear:
                    # Es el primer producto de esta factura
                    facturas_a_crear[factura_uuid] = {
                        'uuid': factura_uuid,
                        'numero_factura': f"RECUPERADA-{factura_uuid[:8]}",
                        'fecha_emision': timestamp,
                        'vendedor_id': vendedor_id,
                        'cliente_id': CLIENTE_GENERICO_ID,
                        'subtotal': precio_total,
                        'iva': 0.0,
                        'total': precio_total,
                        'estado': 'Pagada',
                        'metodo_pago': 'Efectivo',
                        'created_at': timestamp,
                        'updated_at': timestamp,
                        'deleted_at': None
                    }
                else:
                    # Esta factura ya tiene otros productos, solo actualizamos totales
                    facturas_a_crear[factura_uuid]['subtotal'] += precio_total
                    facturas_a_crear[factura_uuid]['total'] += precio_total
                    # Usamos el timestamp más reciente para updated_at
                    if timestamp > facturas_a_crear[factura_uuid]['updated_at']:
                        facturas_a_crear[factura_uuid]['updated_at'] = timestamp

        print(f"ℹ️  Procesamiento terminado.")
        print(f"   Se crearán {len(facturas_a_crear)} nuevas facturas.")
        print(f"   Se crearán {len(detalles_a_crear)} nuevos detalles.")

        return list(facturas_a_crear.values()), detalles_a_crear

    except Exception as e:
        print(f"Error fatal procesando {OPERACIONES_FILE}: {e}")
        return None, None

def escribir_csv(filename, data, headers):
    """Escribe una lista de diccionarios a un archivo CSV."""
    if not data:
        print(f"ℹ️  No hay datos para escribir en {filename}.")
        return

    try:
        with open(filename, mode='w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=headers, extrasaction='ignore')
            writer.writeheader()
            writer.writerows(data)
        print(f"✅ Archivo generado: {filename}")
    except Exception as e:
        print(f"Error fatal escribiendo {filename}: {e}")

def main():
    print("--- Iniciando Script de Recuperación de Facturas ---")
    
    productos = cargar_productos()
    if productos is None:
        return

    facturas_exist = cargar_facturas_existentes()
    if facturas_exist is None:
        return

    nuevas_facturas, nuevos_detalles = procesar_operaciones(productos, facturas_exist)
    
    if nuevas_facturas:
        # Definimos las cabeceras para la importación en Supabase
        # IMPORTANTE: Omitimos 'id' (bigserial)
        factura_headers = [
            'uuid', 'numero_factura', 'fecha_emision', 'vendedor_id', 'cliente_id',
            'subtotal', 'iva', 'total', 'estado', 'metodo_pago', 'created_at', 
            'updated_at', 'deleted_at'
        ]
        escribir_csv(NUEVAS_FACTURAS_FILE, nuevas_facturas, factura_headers)

    if nuevos_detalles:
        # Omitimos 'id' (bigserial)
        detalle_headers = [
            'uuid', 'factura_uuid', 'producto_id', 'cantidad', 'precio_unitario',
            'precio_total', 'created_at', 'updated_at', 'factura_id', 'deleted_at'
        ]
        escribir_csv(NUEVOS_DETALLES_FILE, nuevos_detalles, detalle_headers)

    print("--- Script finalizado ---")

if __name__ == "__main__":
    main()