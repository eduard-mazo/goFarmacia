<template>
  <div>
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Gestionar Productos</h1>

    <div class="bg-white p-6 rounded-lg shadow-md mb-8">
      <h3 class="text-xl font-semibold mb-4">
        {{ editando ? "Editar" : "Registrar" }} Producto
      </h3>
      <form
        @submit.prevent="guardarProducto"
        class="grid grid-cols-1 md:grid-cols-2 gap-6"
      >
        <div class="form-field">
          <label class="block text-sm font-medium text-gray-700 mb-1">Código</label>
          <input type="text" v-model="producto.Codigo" class="input" required />
        </div>
        <div class="form-field">
          <label class="block text-sm font-medium text-gray-700 mb-1">Nombre</label>
          <input type="text" v-model="producto.Nombre" class="input" required />
        </div>
        <div class="form-field">
          <label class="block text-sm font-medium text-gray-700 mb-1">Precio de Venta</label>
          <input
            type="number"
            step="0.01"
            v-model.number="producto.PrecioVenta"
            class="input"
            required
          />
        </div>
        <div class="form-field">
          <label class="block text-sm font-medium text-gray-700 mb-1">Stock</label>
          <input
            type="number"
            v-model.number="producto.Stock"
            class="input"
            required
          />
        </div>
        <div class="md:col-span-2 flex items-center space-x-4">
          <button type="submit" class="btn-primary">
            {{ editando ? "Actualizar Producto" : "Registrar Producto" }}
          </button>
          <button
            v-if="editando"
            @click="cancelarEdicion"
            type="button"
            class="btn-secondary"
          >
            Cancelar
          </button>
        </div>
      </form>
    </div>

    <div class="bg-white shadow-md rounded-lg overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="th">ID</th>
            <th class="th">Código</th>
            <th class="th">Nombre</th>
            <th class="th">Precio</th>
            <th class="th">Stock</th>
            <th class="th text-right">Acciones</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="p in listaProductos" :key="p.id">
            <td class="td">{{ p.id }}</td>
            <td class="td font-mono">{{ p.Codigo }}</td>
            <td class="td font-medium text-gray-900">{{ p.Nombre }}</td>
            <td class="td">${{ p.PrecioVenta?.toFixed(2) }}</td>
            <td class="td">{{ p.Stock }}</td>
            <td class="td text-right space-x-2">
              <button @click="editarProducto(p)" class="btn-edit">
                Editar
              </button>
              <button @click="eliminarProducto(p.id!)" class="btn-delete">
                Eliminar
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from "vue";
import { backend } from "../../wailsjs/go/models";
import {
  RegistrarProducto,
  ObtenerProductos,
  ActualizarProducto,
  EliminarProducto,
} from "../../wailsjs/go/backend/Db";

const listaProductos = ref<backend.Producto[]>([]);
const producto = ref(new backend.Producto());
const editando = ref(false);

const cargarProductos = async () => {
  try {
    listaProductos.value = await ObtenerProductos();
  } catch (error) {
    alert(`Error al cargar productos: ${error}`);
  }
};

const guardarProducto = async () => {
  try {
    let resultado: string;
    if (editando.value) {
      resultado = await ActualizarProducto(producto.value);
    } else {
      resultado = await RegistrarProducto(producto.value);
    }
    alert(resultado);
    cancelarEdicion();
    await cargarProductos();
  } catch (error) {
    alert(`Error al guardar producto: ${error}`);
  }
};

const editarProducto = (p: backend.Producto) => {
  producto.value = backend.Producto.createFrom(p);
  editando.value = true;
};

const cancelarEdicion = () => {
  producto.value = new backend.Producto();
  editando.value = false;
};

const eliminarProducto = async (id: number) => {
  if (confirm("¿Estás seguro de que quieres eliminar este producto?")) {
    try {
      const resultado = await EliminarProducto(id);
      alert(resultado);
      await cargarProductos();
    } catch (error) {
      alert(`Error al eliminar producto: ${error}`);
    }
  }
};

onMounted(cargarProductos);
</script>

<style scoped>
@reference "../style.css";
.label {
  @apply block text-sm font-medium text-gray-700 mb-1;
}
.input {
  @apply block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm p-2;
}
.btn-primary {
  @apply bg-indigo-600 text-white font-bold py-2 px-4 rounded-md hover:bg-indigo-700 transition disabled:bg-gray-400;
}
.btn-secondary {
  @apply bg-gray-200 text-gray-700 font-bold py-2 px-4 rounded-md hover:bg-gray-300 transition;
}
.btn-edit {
  @apply text-indigo-600 hover:text-indigo-900 font-medium;
}
.btn-delete {
  @apply text-red-600 hover:text-red-900 font-medium;
}
.th {
  @apply px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider;
}
.td {
  @apply px-6 py-4 whitespace-nowrap text-sm text-gray-600;
}
</style>
