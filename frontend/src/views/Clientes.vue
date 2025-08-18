<template>
  <div>
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Gestionar Clientes</h1>
    <div class="bg-white p-6 rounded-lg shadow-md mb-8">
      <h3 class="text-xl font-semibold mb-4">
        {{ editando ? "Editar" : "Registrar" }} Cliente
      </h3>
      <form
        @submit.prevent="guardarCliente"
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
      >
        <div>
          <label>Nombre</label
          ><input type="text" v-model="cliente.Nombre" required />
        </div>
        <div>
          <label>Apellido</label
          ><input type="text" v-model="cliente.Apellido" required />
        </div>
        <div>
          <label>Tipo ID</label
          ><input type="text" v-model="cliente.TipoID" required />
        </div>
        <div>
          <label>Número ID</label
          ><input type="text" v-model="cliente.NumeroID" required />
        </div>
        <div>
          <label>Teléfono</label><input type="tel" v-model="cliente.Telefono" />
        </div>
        <div>
          <label>Email</label><input type="email" v-model="cliente.Email" />
        </div>
        <div class="form-field md:col-span-2 lg:col-span-3">
          <label>Dirección</label
          ><input type="text" v-model="cliente.Direccion" />
        </div>
        <div class="md:col-span-2 lg:col-span-3 flex items-center space-x-4">
          <button type="submit" class="btn-primary">
            {{ editando ? "Actualizar Cliente" : "Registrar Cliente" }}
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
            <th>Identificación</th>
            <th>Teléfono</th>
            <th class="text-right">Acciones</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="c in listaClientes" :key="c.id">
            <td>{{ c.id }}</td>
            <td class="font-medium text-gray-900">
              {{ c.Nombre }} {{ c.Apellido }}
            </td>
            <td>{{ c.TipoID }} {{ c.NumeroID }}</td>
            <td>{{ c.Telefono }}</td>
            <td class="text-right space-x-2">
              <button @click="editarCliente(c)" class="btn-edit">Editar</button>
              <button @click="eliminarCliente(c.id!)" class="btn-delete">
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
  RegistrarCliente,
  ObtenerClientes,
  ActualizarCliente,
  EliminarCliente,
} from "../../wailsjs/go/backend/Db";

const listaClientes = ref<backend.Cliente[]>([]);
const cliente = ref(new backend.Cliente());
const editando = ref(false);

const cargarClientes = async () => {
  try {
    listaClientes.value = await ObtenerClientes();
  } catch (error) {
    alert(`Error al cargar clientes: ${error}`);
  }
};

const guardarCliente = async () => {
  try {
    let resultado: string;
    if (editando.value) {
      resultado = await ActualizarCliente(cliente.value);
    } else {
      resultado = await RegistrarCliente(cliente.value);
    }
    alert(resultado);
    cancelarEdicion();
    await cargarClientes();
  } catch (error) {
    alert(`Error al guardar cliente: ${error}`);
  }
};

const editarCliente = (c: backend.Cliente) => {
  cliente.value = backend.Cliente.createFrom(c);
  editando.value = true;
};

const cancelarEdicion = () => {
  cliente.value = new backend.Cliente();
  editando.value = false;
};

const eliminarCliente = async (id: number) => {
  if (confirm("¿Estás seguro de que quieres eliminar este cliente?")) {
    try {
      const resultado = await EliminarCliente(id);
      alert(resultado);
      await cargarClientes();
    } catch (error) {
      alert(`Error al eliminar cliente: ${error}`);
    }
  }
};

onMounted(cargarClientes);
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
