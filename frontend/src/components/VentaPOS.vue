<template>
  <div>
    <h1>Punto de Venta</h1>
    <div class="pos-container">
      <div class="venta-col">
        <div class="form-section">
          <h3>Detalles de la Venta</h3>
          <div class="venta-controles">
            <div>
              <label>Cliente:</label>
              <select v-model="venta.ClienteID" required>
                <option v-for="c in clientes" :key="c.id" :value="c.id">
                  {{ c.Nombre }} {{ c.Apellido }}
                </option>
              </select>
            </div>
            <div>
              <label>Vendedor:</label>
              <select v-model="venta.VendedorID" required>
                <option v-for="v in vendedores" :key="v.id" :value="v.id">
                  {{ v.Nombre }} {{ v.Apellido }}
                </option>
              </select>
            </div>
            <div>
              <label>Método de Pago:</label>
              <select v-model="venta.MetodoPago" required>
                <option>Efectivo</option>
                <option>Tarjeta</option>
                <option>Transferencia</option>
              </select>
            </div>
          </div>
        </div>

        <div class="table-section carrito">
          <h3>Carrito de Compras</h3>
          <table>
            <thead>
              <tr>
                <th>Producto</th>
                <th>Cantidad</th>
                <th>Precio Unit.</th>
                <th>Subtotal</th>
                <th>Acción</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(item, index) in carrito" :key="item.producto.id">
                <td>{{ item.producto.Nombre }}</td>
                <td>
                  <input
                    type="number"
                    v-model.number="item.cantidad"
                    min="1"
                    :max="item.producto.Stock"
                    @change="actualizarTotal"
                  />
                </td>
                <td>${{ item.producto.PrecioVenta?.toFixed(2) }}</td>
                <td>
                  ${{ (item.producto.PrecioVenta! * item.cantidad).toFixed(2) }}
                </td>
                <td>
                  <button @click="quitarDelCarrito(index)" class="danger">
                    Quitar
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div class="total-final">
            <h3>Total: ${{ totalVenta.toFixed(2) }}</h3>
            <button @click="realizarVenta" :disabled="carrito.length === 0">
              Finalizar Venta
            </button>
          </div>
        </div>
      </div>

      <div class="productos-col">
        <div class="table-section">
          <h3>Buscar Productos</h3>
          <input
            v-model="busqueda"
            @input="buscarProductos"
            placeholder="Buscar por nombre o código..."
          />
          <table>
            <thead>
              <tr>
                <th>Nombre</th>
                <th>Stock</th>
                <th>Precio</th>
                <th>Acción</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in resultadosBusqueda" :key="p.id">
                <td>{{ p.Nombre }}</td>
                <td>{{ p.Stock }}</td>
                <td>${{ p.PrecioVenta?.toFixed(2) }}</td>
                <td>
                  <button
                    @click="agregarAlCarrito(p)"
                    :disabled="p.Stock === 0"
                  >
                    Agregar
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, computed } from "vue";
import { backend } from "../../wailsjs/go/models";
import {
  BuscarProductos,
  ObtenerClientes,
  ObtenerVendedores,
  RegistrarVenta,
} from "../../wailsjs/go/backend/Db";
import { debounce } from "lodash";

// Tipos
interface CarritoItem {
  producto: backend.Producto;
  cantidad: number;
}

// Reactividad
const clientes = ref<backend.Cliente[]>([]);
const vendedores = ref<backend.Vendedor[]>([]);
const busqueda = ref("");
const resultadosBusqueda = ref<backend.Producto[]>([]);
const carrito = ref<CarritoItem[]>([]);
const venta = ref(new backend.VentaRequest());
const totalVenta = ref(0.0);

// Carga inicial de datos
const cargarDatosIniciales = async () => {
  try {
    clientes.value = await ObtenerClientes();
    vendedores.value = await ObtenerVendedores();
    // Asignar valores por defecto si existen
    if (clientes.value.length > 0)
      venta.value.ClienteID = clientes.value[0].id!;
    if (vendedores.value.length > 0)
      venta.value.VendedorID = vendedores.value[0].id!;
  } catch (error) {
    alert(`Error cargando datos: ${error}`);
  }
};

onMounted(cargarDatosIniciales);

// Lógica de búsqueda con debounce para no saturar
const buscarProductos = debounce(async () => {
  if (busqueda.value.length > 2) {
    try {
      resultadosBusqueda.value = await BuscarProductos(busqueda.value);
    } catch (error) {
      console.error("Error en la búsqueda:", error);
    }
  } else {
    resultadosBusqueda.value = [];
  }
}, 300);

// Lógica del carrito
const agregarAlCarrito = (producto: backend.Producto) => {
  const itemExistente = carrito.value.find(
    (item) => item.producto.id === producto.id
  );
  if (itemExistente) {
    if (itemExistente.cantidad < producto.Stock!) {
      itemExistente.cantidad++;
    } else {
      alert("No hay más stock disponible para este producto.");
    }
  } else {
    carrito.value.push({ producto: producto, cantidad: 1 });
  }
  actualizarTotal();
};

const quitarDelCarrito = (index: number) => {
  carrito.value.splice(index, 1);
  actualizarTotal();
};

const actualizarTotal = () => {
  totalVenta.value = carrito.value.reduce((acc, item) => {
    return acc + item.producto.PrecioVenta! * item.cantidad;
  }, 0);
};

// Lógica de Venta
const realizarVenta = async () => {
  if (
    !venta.value.ClienteID ||
    !venta.value.VendedorID ||
    carrito.value.length === 0
  ) {
    alert(
      "Por favor, completa todos los campos y agrega productos al carrito."
    );
    return;
  }

  // Mapear el carrito al formato esperado por el backend
  venta.value.Productos = carrito.value.map((item) => {
    const p = new backend.ProductoVenta();
    p.ID = item.producto.id!;
    p.Cantidad = item.cantidad;
    return p;
  });

  // Asignar método de pago por defecto si no se seleccionó
  if (!venta.value.MetodoPago) venta.value.MetodoPago = "Efectivo";

  try {
    const resultado = await RegistrarVenta(venta.value);
    alert(resultado);
    // Limpiar para la siguiente venta
    carrito.value = [];
    totalVenta.value = 0;
    busqueda.value = "";
    resultadosBusqueda.value = [];
  } catch (error) {
    alert(`Error al registrar la venta: ${error}`);
  }
};
</script>

<style scoped>
.pos-container {
  display: flex;
  gap: 2rem;
}
.venta-col {
  flex: 2;
}
.productos-col {
  flex: 1;
}
.venta-controles {
  display: flex;
  gap: 1.5rem;
  margin-bottom: 1rem;
}
.carrito {
  margin-top: 2rem;
}
.total-final {
  text-align: right;
  margin-top: 1rem;
}
.total-final h3 {
  display: inline-block;
  margin-right: 1rem;
}
</style>
