<template>
  <div>
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Gestionar Vendedores</h1>
    <div class="bg-white p-6 rounded-lg shadow-md mb-8">
      <h3 class="text-xl font-semibold mb-4">
        {{ editando ? "Editar" : "Registrar" }} Vendedor
      </h3>
      <form
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
        @submit.prevent="guardarVendedor"
      >
        <div>
          <label>Nombre:</label>
          <input v-model="vendedor.Nombre" required />
        </div>
        <div>
          <label>Apellido:</label>
          <input v-model="vendedor.Apellido" required />
        </div>
        <div>
          <label>Cédula:</label>
          <input v-model="vendedor.Cedula" required />
        </div>
        <div>
          <label>Email:</label>
          <input v-model="vendedor.Email" type="email" required />
        </div>
        <div>
          <label>Contraseña:</label>
          <input
            v-model="vendedor.Contrasena"
            type="password"
            :required="!editando"
          />
        </div>
        <div class="md:col-span-2 lg:col-span-3 flex items-center space-x-4">
          <button type="submit" class="btn-primary">
            {{ editando ? "Actualizar" : "Registrar" }}
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
            <th>ID</th>
            <th>Nombre Completo</th>
            <th>Cédula</th>
            <th>Email</th>
            <th class="text-right">Acciones</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="v in listaVendedores" :key="v.id">
            <td>{{ v.id }}</td>
            <td class="font-medium text-gray-900">
              {{ v.Nombre }} {{ v.Apellido }}
            </td>
            <td>{{ v.Cedula }}</td>
            <td>{{ v.Email }}</td>
            <td class="text-right space-x-2">
              <button @click="editarVendedor(v)" class="btn-edit">
                Editar
              </button>
              <button @click="eliminarVendedor(v.id!)" class="btn-delete">
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
  RegistrarVendedor,
  ObtenerVendedores,
  ActualizarVendedor,
  EliminarVendedor,
} from "../../wailsjs/go/backend/Db";

const listaVendedores = ref<backend.Vendedor[]>([]);
const vendedor = ref(new backend.Vendedor());
const editando = ref(false);

const cargarVendedores = async () => {
  try {
    listaVendedores.value = await ObtenerVendedores();
  } catch (error) {
    alert(`Error al cargar vendedores: ${error}`);
  }
};

const guardarVendedor = async () => {
  try {
    let resultado: string;
    if (editando.value) {
      resultado = await ActualizarVendedor(vendedor.value);
    } else {
      resultado = await RegistrarVendedor(vendedor.value);
    }
    alert(resultado);
    cancelarEdicion();
    await cargarVendedores();
  } catch (error) {
    alert(`Error al guardar vendedor: ${error}`);
  }
};

const editarVendedor = (v: backend.Vendedor) => {
  // Clonamos el objeto para no modificar la lista directamente
  vendedor.value = backend.Vendedor.createFrom(v);
  editando.value = true;
};

const cancelarEdicion = () => {
  vendedor.value = new backend.Vendedor();
  editando.value = false;
};

const eliminarVendedor = async (id: number) => {
  if (confirm("¿Estás seguro de que quieres eliminar este vendedor?")) {
    try {
      const resultado = await EliminarVendedor(id);
      alert(resultado);
      await cargarVendedores();
    } catch (error) {
      alert(`Error al eliminar vendedor: ${error}`);
    }
  }
};

onMounted(cargarVendedores);
</script>

<style scoped>
@reference "../style.css";
label {
  @apply block text-sm font-medium text-gray-700 mb-1;
}
input {
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
th {
  @apply px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider;
}
td {
  @apply px-6 py-4 whitespace-nowrap text-sm text-gray-600;
}
</style>
