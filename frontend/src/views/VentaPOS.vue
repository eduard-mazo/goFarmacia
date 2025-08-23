<template>
  <div class="grid grid-cols-4 gap-6 h-[calc(100vh-100px)] font-sans">
    <div
      class="col-span-3 flex flex-col h-full bg-white rounded-lg shadow-lg p-6"
    >
      <div class="relative mb-4">
        <label
          for="scanner-input"
          class="block text-lg font-medium text-gray-800 mb-2"
        >
          🔍 Buscar Producto por Nombre o Código
        </label>
        <input
          ref="scannerInput"
          id="scanner-input"
          v-model="busqueda"
          @input="buscarProductosCoincidentes"
          @keydown.down.prevent="moverSeleccion(1)"
          @keydown.up.prevent="moverSeleccion(-1)"
          @keydown.enter.prevent="agregarProductoSeleccionado"
          autocomplete="off"
          placeholder="Escriba aquí para buscar..."
          class="block w-full border-gray-300 rounded-md shadow-sm text-lg p-4 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition"
        />
        <div
          v-if="busqueda && mostrarResultados"
          class="absolute z-10 w-full mt-1 bg-white rounded-md shadow-xl border max-h-60 overflow-y-auto"
        >
          <ul>
            <li
              v-if="resultadosBusqueda.length === 0"
              class="px-4 py-3 text-gray-500"
            >
              No se encontraron productos.
            </li>
            <li
              v-for="(producto, index) in resultadosBusqueda"
              :key="producto.id"
              @click="seleccionarYAgregar(producto)"
              :class="{ 'bg-blue-500 text-white': index === seleccionIndex }"
              class="px-4 py-3 cursor-pointer hover:bg-blue-100"
            >
              <div class="font-bold">{{ producto.Nombre }}</div>
              <div
                class="text-sm text-gray-600"
                :class="{ 'text-white': index === seleccionIndex }"
              >
                Código: {{ producto.Codigo }} | Stock: {{ producto.Stock }} |
                Precio: ${{ producto.PrecioVenta?.toFixed(2) }}
              </div>
            </li>
          </ul>
        </div>
      </div>

      <div class="flex-grow overflow-y-auto border-t border-b border-gray-200">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-100 sticky top-0">
            <tr>
              <th
                class="px-6 py-3 text-left text-xs font-bold text-gray-600 uppercase tracking-wider"
              >
                Producto
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-bold text-gray-600 uppercase tracking-wider"
              >
                Cantidad
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-bold text-gray-600 uppercase tracking-wider"
              >
                Precio
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-bold text-gray-600 uppercase tracking-wider"
              >
                Subtotal
              </th>
              <th
                class="px-6 py-3 text-right text-xs font-bold text-gray-600 uppercase tracking-wider"
              >
                Acción
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-if="carrito.length === 0">
              <td colspan="5" class="text-center text-gray-500 py-12">
                El carrito está vacío 🛒
              </td>
            </tr>
            <tr
              v-for="(item, index) in carrito"
              :key="item.producto.id"
              class="hover:bg-gray-50"
            >
              <td class="px-6 py-4 whitespace-nowrap font-medium text-gray-900">
                {{ item.producto.Nombre }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <input
                  type="number"
                  v-model.number="item.cantidad"
                  min="1"
                  :max="item.producto.Stock"
                  @change="actualizarTotal"
                  class="w-24 text-center border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
                />
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-gray-700">
                ${{ item.producto.PrecioVenta?.toFixed(2) }}
              </td>
              <td
                class="px-6 py-4 whitespace-nowrap font-semibold text-gray-800"
              >
                ${{ (item.producto.PrecioVenta! * item.cantidad).toFixed(2) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right">
                <button
                  @click="quitarDelCarrito(index)"
                  class="text-red-600 hover:text-red-800 font-medium transition"
                >
                  Eliminar
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-auto pt-6 border-t">
        <div class="text-right text-4xl font-extrabold text-gray-800">
          Total:
          <span class="text-blue-600">${{ totalVenta.toFixed(2) }}</span>
        </div>
      </div>
    </div>

    <div class="col-span-1 bg-white rounded-lg shadow-lg p-6 flex flex-col">
      <h3 class="text-2xl font-bold mb-6 text-gray-800 border-b pb-3">
        Finalizar Venta
      </h3>
      <div class="space-y-6">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1"
            >Cliente</label
          >
          <select
            v-model="venta.ClienteID"
            class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md"
          >
            <option v-for="c in clientes" :key="c.id" :value="c.id">
              {{ c.Nombre }} {{ c.Apellido }}
            </option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1"
            >Método de Pago</label
          >
          <select
            v-model="venta.MetodoPago"
            class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md"
          >
            <option>Efectivo</option>
            <option>Tarjeta</option>
            <option>Transferencia</option>
          </select>
        </div>
      </div>
      <div class="mt-auto">
        <button
          @click="realizarVenta"
          :disabled="carrito.length === 0 || procesando"
          class="w-full bg-blue-600 text-white font-bold py-4 rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-all text-lg shadow-md"
        >
          {{ procesando ? "Procesando..." : "Completar Venta 💳" }}
        </button>
      </div>
    </div>

    <div
      v-if="facturaGenerada"
      class="fixed inset-0 bg-black bg-opacity-60 flex justify-center items-center z-50"
    >
      <div class="bg-white p-6 rounded-lg shadow-xl">
        <ReciboPOS :factura="facturaGenerada" />
        <div class="mt-6 text-center">
          <button
            @click="facturaGenerada = null"
            class="bg-gray-600 text-white px-6 py-2 rounded-md hover:bg-gray-700 transition"
          >
            Cerrar
          </button>
        </div>
      </div>
    </div>

    <NotificationModal
      :show="notification.show"
      :message="notification.message"
      :type="notification.type"
      @close="notification.show = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive } from "vue";
import { useAuthStore } from "../stores/auth";
import { backend } from "../../wailsjs/go/models";
import {
  BuscarProductos,
  ObtenerClientes,
  RegistrarVenta,
} from "../../wailsjs/go/backend/Db";
import ReciboPOS from "../components/ReciboPOS.vue";
import NotificationModal from "../components/NotificationModal.vue"; // Importa el nuevo componente

// --- INTERFACES ---
interface CarritoItem {
  producto: backend.Producto;
  cantidad: number;
}
type NotificationType = "success" | "error";

// --- STATE MANAGEMENT ---
const authStore = useAuthStore();
const scannerInput = ref<HTMLInputElement | null>(null);

const clientes = ref<backend.Cliente[]>([]);
const busqueda = ref("");
const resultadosBusqueda = ref<backend.Producto[]>([]);
const mostrarResultados = ref(false);
const seleccionIndex = ref(-1);

const carrito = ref<CarritoItem[]>([]);
const venta = ref(new backend.VentaRequest());
const totalVenta = ref(0.0);
const procesando = ref(false);
const facturaGenerada = ref<backend.Factura | null>(null);

const notification = reactive({
  show: false,
  message: "",
  type: "success" as NotificationType,
});

// --- HOOKS ---
onMounted(() => {
  cargarDatosIniciales();
  scannerInput.value?.focus();
});

// --- METHODS ---
const showNotification = (
  message: string,
  type: NotificationType = "success",
  duration: number = 3000
) => {
  notification.message = message;
  notification.type = type;
  notification.show = true;
  setTimeout(() => {
    notification.show = false;
  }, duration);
};

const cargarDatosIniciales = async () => {
  try {
    clientes.value = await ObtenerClientes();
    if (clientes.value.length > 0 && clientes.value[0]?.id !== undefined)
      venta.value.ClienteID = clientes.value[0].id!;
    venta.value.MetodoPago = "Efectivo";
  } catch (error) {
    showNotification(`Error cargando datos: ${error}`, "error");
  }
};

const buscarProductosCoincidentes = async () => {
  if (!busqueda.value) {
    mostrarResultados.value = false;
    resultadosBusqueda.value = [];
    return;
  }
  try {
    const productos = await BuscarProductos(busqueda.value);
    resultadosBusqueda.value = productos;
    mostrarResultados.value = true;
    seleccionIndex.value = -1; // Reset selection index on new search
  } catch (error) {
    showNotification(`Error buscando producto: ${error}`, "error");
  }
};

const moverSeleccion = (direccion: number) => {
  if (resultadosBusqueda.value.length === 0) return;
  seleccionIndex.value += direccion;
  if (seleccionIndex.value < 0) {
    seleccionIndex.value = resultadosBusqueda.value.length - 1;
  } else if (seleccionIndex.value >= resultadosBusqueda.value.length) {
    seleccionIndex.value = 0;
  }
};

const agregarProductoSeleccionado = () => {
  if (
    seleccionIndex.value >= 0 &&
    seleccionIndex.value < resultadosBusqueda.value.length
  ) {
    const producto = resultadosBusqueda.value[seleccionIndex.value];
    if (producto) {
      seleccionarYAgregar(producto);
    }
  } else if (resultadosBusqueda.value.length > 0) {
    const primerProducto = resultadosBusqueda.value[0];
    if (primerProducto) {
      seleccionarYAgregar(primerProducto);
    }
  }
};

const seleccionarYAgregar = (producto: backend.Producto) => {
  agregarAlCarrito(producto);
  busqueda.value = "";
  mostrarResultados.value = false;
  resultadosBusqueda.value = [];
  scannerInput.value?.focus();
};

const agregarAlCarrito = (producto: backend.Producto) => {
  if (producto.Stock === 0) {
    showNotification("Este producto no tiene stock.", "error");
    return;
  }
  const itemExistente = carrito.value.find(
    (item) => item.producto.id === producto.id
  );
  if (itemExistente) {
    if (itemExistente.cantidad < producto.Stock!) {
      itemExistente.cantidad++;
      showNotification(`${producto.Nombre} cantidad aumentada.`, "success");
    } else {
      showNotification(
        "No hay más stock disponible para este producto.",
        "error"
      );
    }
  } else {
    carrito.value.push({ producto: producto, cantidad: 1 });
    showNotification(`${producto.Nombre} agregado al carrito.`, "success");
  }
  actualizarTotal();
};

const quitarDelCarrito = (index: number) => {
  const item = carrito.value[index];
  if (item) {
    const nombreProducto = item.producto.Nombre;
    carrito.value.splice(index, 1);
    actualizarTotal();
    showNotification(`${nombreProducto} eliminado del carrito.`, "success");
  }
};

const actualizarTotal = () => {
  totalVenta.value = carrito.value.reduce((acc, item) => {
    return acc + (item.producto.PrecioVenta || 0) * item.cantidad;
  }, 0);
};

const realizarVenta = async () => {
  if (procesando.value) return;
  if (!venta.value.ClienteID || carrito.value.length === 0) {
    showNotification(
      "Selecciona un cliente y agrega productos al carrito.",
      "error"
    );
    return;
  }

  const vendedorId = authStore.vendedorId;
  if (!vendedorId) {
    showNotification(
      "Error de sesión: Vendedor no identificado. Inicie sesión de nuevo.",
      "error"
    );
    return;
  }

  procesando.value = true;
  venta.value.VendedorID = vendedorId;
  venta.value.Productos = carrito.value.map((item) => {
    const p = new backend.ProductoVenta();
    p.ID = item.producto.id!;
    p.Cantidad = item.cantidad;
    return p;
  });

  try {
    const factura = await RegistrarVenta(venta.value);
    facturaGenerada.value = factura;
    // Reset state after successful sale
    carrito.value = [];
    totalVenta.value = 0;
    busqueda.value = "";
  } catch (error) {
    showNotification(`Error al registrar la venta: ${error}`, "error");
  } finally {
    procesando.value = false;
  }
};
</script>

<style scoped>
/* Scoped styles for fine-tuning if needed */
</style>
