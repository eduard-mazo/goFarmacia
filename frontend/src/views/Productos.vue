<template>
  <div>
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Gestionar Productos</h1>

    <div
      class="flex justify-between items-center mb-6 bg-white p-4 rounded-lg shadow-md"
    >
      <div class="relative">
        <input
          type="text"
          v-model="busqueda"
          @input="debouncedCargarProductos"
          placeholder="Buscar por nombre o código..."
          class="input pl-10"
        />
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
          >🔍</span
        >
      </div>
      <button @click="abrirModalParaCrear" class="btn-primary">
        ✨ Registrar Nuevo Producto
      </button>
    </div>

    <div class="bg-white shadow-md rounded-lg overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="th">Código</th>
            <th class="th">Nombre</th>
            <th class="th">Precio</th>
            <th class="th">Stock</th>
            <th class="th text-right">Acciones</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-if="listaProductos.length === 0">
            <td colspan="5" class="text-center py-10 text-gray-500">
              No se encontraron productos.
            </td>
          </tr>
          <tr v-for="p in listaProductos" :key="p.id">
            <td class="td font-mono">{{ p.Codigo }}</td>
            <td class="td font-medium text-gray-900">{{ p.Nombre }}</td>
            <td class="td">${{ p.PrecioVenta?.toFixed(2) }}</td>
            <td class="td">{{ p.Stock }}</td>
            <td class="td text-right space-x-2">
              <button @click="editarProducto(p)" class="btn-edit">
                Editar
              </button>
              <button @click="confirmarEliminacion(p.id!)" class="btn-delete">
                Eliminar
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="flex justify-between items-center mt-4" v-if="totalPaginas > 1">
      <span class="text-sm text-gray-700">
        Página {{ paginaActual }} de {{ totalPaginas }} (Total:
        {{ totalProductos }} productos)
      </span>
      <div>
        <button
          @click="cambiarPagina(paginaActual - 1)"
          :disabled="paginaActual === 1"
          class="btn-secondary"
        >
          Anterior
        </button>
        <button
          @click="cambiarPagina(paginaActual + 1)"
          :disabled="paginaActual === totalPaginas"
          class="btn-secondary ml-2"
        >
          Siguiente
        </button>
      </div>
    </div>

    <div
      v-if="mostrarModal"
      class="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-50"
    >
      <div class="bg-white p-8 rounded-lg shadow-2xl w-full max-w-md">
        <h3 class="text-2xl font-bold mb-6">
          {{ editando ? "Editar" : "Registrar" }} Producto
        </h3>
        <form @submit.prevent="guardarProducto" class="space-y-4">
          <div>
            <label class="label">Código</label>
            <input
              type="text"
              v-model="producto.Codigo"
              class="input"
              required
            />
          </div>
          <div>
            <label class="label">Nombre</label>
            <input
              type="text"
              v-model="producto.Nombre"
              class="input"
              required
            />
          </div>
          <div>
            <label class="label">Precio de Venta</label>
            <input
              type="number"
              step="0.01"
              v-model.number="producto.PrecioVenta"
              class="input"
              required
            />
          </div>
          <div>
            <label class="label">Stock</label>
            <input
              type="number"
              v-model.number="producto.Stock"
              class="input"
              required
            />
          </div>
          <div class="flex justify-end space-x-4 pt-4">
            <button type="button" @click="cerrarModal" class="btn-secondary">
              Cancelar
            </button>
            <button type="submit" class="btn-primary">
              {{ editando ? "Actualizar" : "Registrar" }}
            </button>
          </div>
        </form>
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
import { ref, onMounted, reactive, computed } from "vue";
import { backend } from "../../wailsjs/go/models";
import {
  RegistrarProducto,
  ObtenerProductosPaginado,
  ActualizarProducto,
  EliminarProducto,
} from "../../wailsjs/go/backend/Db";
import NotificationModal from "../components/NotificationModal.vue";

type NotificationType = "success" | "error";

// --- STATE ---
const listaProductos = ref<backend.Producto[]>([]);
const producto = ref(new backend.Producto());
const editando = ref(false);
const mostrarModal = ref(false);
const busqueda = ref("");
let debounceTimer: number;

const notification = reactive({
  show: false,
  message: "",
  type: "success" as NotificationType,
});

// Paginación State
const paginaActual = ref(1);
const productosPorPagina = ref(10);
const totalProductos = ref(0);

// --- COMPUTED ---
const totalPaginas = computed(() =>
  Math.ceil(totalProductos.value / productosPorPagina.value)
);

// --- METHODS ---
const showNotification = (
  message: string,
  type: NotificationType = "success"
) => {
  notification.message = message;
  notification.type = type;
  notification.show = true;
  setTimeout(() => {
    notification.show = false;
  }, 3000);
};

const cargarProductos = async () => {
  try {
    const response = await ObtenerProductosPaginado(
      paginaActual.value,
      productosPorPagina.value,
      busqueda.value
    );
    listaProductos.value = response.Records as backend.Producto[];
    totalProductos.value = response.TotalRecords;
  } catch (error) {
    showNotification(`Error al cargar productos: ${error}`, "error");
  }
};

const debouncedCargarProductos = () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    paginaActual.value = 1; // Reset to first page on new search
    cargarProductos();
  }, 300); // 300ms delay
};

const cambiarPagina = (nuevaPagina: number) => {
  if (nuevaPagina > 0 && nuevaPagina <= totalPaginas.value) {
    paginaActual.value = nuevaPagina;
    cargarProductos();
  }
};

const abrirModalParaCrear = () => {
  editando.value = false;
  producto.value = new backend.Producto();
  mostrarModal.value = true;
};

const editarProducto = (p: backend.Producto) => {
  editando.value = true;
  producto.value = backend.Producto.createFrom(p);
  mostrarModal.value = true;
};

const cerrarModal = () => {
  mostrarModal.value = false;
};

const guardarProducto = async () => {
  try {
    let resultado: string;
    if (editando.value) {
      resultado = await ActualizarProducto(producto.value);
    } else {
      resultado = await RegistrarProducto(producto.value);
    }
    showNotification(resultado, "success");
    cerrarModal();
    await cargarProductos();
  } catch (error) {
    showNotification(`Error al guardar producto: ${error}`, "error");
  }
};

const confirmarEliminacion = (id: number) => {
  if (confirm("¿Estás seguro de que quieres eliminar este producto?")) {
    eliminarProducto(id);
  }
};

const eliminarProducto = async (id: number) => {
  try {
    const resultado = await EliminarProducto(id);
    showNotification(resultado, "success");
    await cargarProductos();
  } catch (error) {
    showNotification(`Error al eliminar producto: ${error}`, "error");
  }
};

onMounted(cargarProductos);
</script>

<style scoped>
@reference "../style.css";
</style>
