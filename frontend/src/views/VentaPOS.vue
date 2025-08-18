<template>
  <div class="grid grid-cols-3 gap-6 h-full">
    <div class="col-span-2 flex flex-col h-full bg-white rounded-lg shadow p-6">
      <div class="mb-4">
        <label
          for="scanner-input"
          class="block text-sm font-medium text-gray-700 mb-1"
          >Escanear Código o Buscar Producto</label
        >
        <input
          ref="scannerInput"
          id="scanner-input"
          v-model="busqueda"
          @keydown.enter.prevent="agregarProductoBuscado"
          placeholder="Presione Enter para agregar..."
          class="block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm p-3"
        />
      </div>

      <div class="flex-grow overflow-y-auto border-t border-b">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Producto
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Cantidad
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Precio
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Subtotal
              </th>
              <th
                class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Acción
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="(item, index) in carrito" :key="item.producto.id">
              <td class="px-6 py-4 whitespace-nowrap">
                {{ item.producto.Nombre }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <input
                  type="number"
                  v-model.number="item.cantidad"
                  min="1"
                  :max="item.producto.Stock"
                  @change="actualizarTotal"
                  class="w-20 text-center border rounded"
                />
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                ${{ item.producto.PrecioVenta?.toFixed(2) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                ${{ (item.producto.PrecioVenta! * item.cantidad).toFixed(2) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right">
                <button
                  @click="quitarDelCarrito(index)"
                  class="text-red-600 hover:text-red-900"
                >
                  Eliminar
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="carrito.length === 0" class="text-center text-gray-500 py-8">
          El carrito está vacío
        </p>
      </div>

      <div class="mt-auto pt-4">
        <div class="text-right text-3xl font-bold">
          Total:
          <span class="text-indigo-600">${{ totalVenta.toFixed(2) }}</span>
        </div>
      </div>
    </div>

    <div class="col-span-1 bg-white rounded-lg shadow p-6">
      <h3 class="text-lg font-medium mb-4">Finalizar Venta</h3>
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700">Cliente</label>
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
          <label class="block text-sm font-medium text-gray-700"
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
        <button
          @click="realizarVenta"
          :disabled="carrito.length === 0 || procesando"
          class="w-full bg-indigo-600 text-white font-bold py-3 rounded-md hover:bg-indigo-700 disabled:bg-gray-400 transition"
        >
          {{ procesando ? "Procesando..." : "Completar Venta" }}
        </button>
      </div>
    </div>

    <div
      v-if="facturaGenerada"
      class="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center"
    >
      <div class="bg-white p-4 rounded-lg shadow-xl">
        <ReciboPOS :factura="facturaGenerada" />
        <div class="mt-4 text-center">
          <button
            @click="facturaGenerada = null"
            class="bg-gray-500 text-white px-4 py-2 rounded"
          >
            Cerrar
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from "vue";
import { useAuthStore } from "../stores/auth";
import { backend } from "../../wailsjs/go/models";
import {
  BuscarProductos,
  ObtenerClientes,
  RegistrarVenta,
} from "../../wailsjs/go/backend/Db";
import ReciboPOS from "../components/ReciboPOS.vue";

interface CarritoItem {
  producto: backend.Producto;
  cantidad: number;
}

const authStore = useAuthStore();
const scannerInput = ref<HTMLInputElement | null>(null);

const clientes = ref<backend.Cliente[]>([]);
const busqueda = ref("");
const carrito = ref<CarritoItem[]>([]);
const venta = ref(new backend.VentaRequest());
const totalVenta = ref(0.0);
const procesando = ref(false);
const facturaGenerada = ref<backend.Factura | null>(null);

const cargarDatosIniciales = async () => {
  try {
    clientes.value = await ObtenerClientes();
    if (clientes.value.length > 0 && clientes.value[0]?.id !== undefined)
      venta.value.ClienteID = clientes.value[0].id!;
    venta.value.MetodoPago = "Efectivo";
  } catch (error) {
    alert(`Error cargando datos: ${error}`);
  }
};

const focusScanner = () => {
  scannerInput.value?.focus();
};

onMounted(() => {
  cargarDatosIniciales();
  focusScanner();
  window.addEventListener("click", focusScanner);
});

onUnmounted(() => {
  window.removeEventListener("click", focusScanner);
});

const agregarProductoBuscado = async () => {
  if (!busqueda.value) return;
  try {
    const productosEncontrados = await BuscarProductos(busqueda.value);

    // CORRECCIÓN 2: Se verifica que el array no esté vacío.
    if (productosEncontrados && productosEncontrados.length > 0) {
      agregarAlCarrito(productosEncontrados[0]!);
      busqueda.value = "";
    } else {
      alert("Producto no encontrado");
    }
  } catch (error) {
    alert(`Error buscando producto: ${error}`);
  }
};

const agregarAlCarrito = (producto: backend.Producto) => {
  if (producto.Stock === 0) {
    alert("Este producto no tiene stock.");
    return;
  }
  const itemExistente = carrito.value.find(
    (item) => item.producto.id === producto.id
  );
  if (itemExistente) {
    if (itemExistente.cantidad < producto.Stock!) itemExistente.cantidad++;
    else alert("No hay más stock disponible.");
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

const realizarVenta = async () => {
  if (procesando.value) return;
  if (!venta.value.ClienteID || carrito.value.length === 0) {
    alert("Selecciona un cliente y agrega productos.");
    return;
  }

  // CORRECCIÓN 1: Se asigna el ID a una constante local.
  const vendedorId = authStore.vendedorId;

  // Se verifica la constante. TypeScript ahora puede garantizar que no es 'undefined' más adelante.
  if (!vendedorId) {
    alert(
      "Error de sesión: Vendedor no identificado. Por favor, inicie sesión de nuevo."
    );
    return;
  }

  procesando.value = true;
  venta.value.VendedorID = vendedorId; // Se usa la constante local segura.
  venta.value.Productos = carrito.value.map((item) => {
    const p = new backend.ProductoVenta();
    p.ID = item.producto.id!;
    p.Cantidad = item.cantidad;
    return p;
  });

  try {
    const factura = await RegistrarVenta(venta.value);
    facturaGenerada.value = factura;
    carrito.value = [];
    totalVenta.value = 0;
    busqueda.value = "";
  } catch (error) {
    alert(`Error al registrar la venta: ${error}`);
  } finally {
    procesando.value = false;
  }
};
</script>
