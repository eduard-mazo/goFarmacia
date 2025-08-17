<template>
  <div>
    <h1>Gestionar Clientes</h1>
    <div class="form-section">
      <h3>{{ editando ? "Editar" : "Registrar" }} Cliente</h3>
      <form @submit.prevent="guardarCliente">
        <div class="form-field">
          <label>Nombre:</label>
          <input v-model="cliente.Nombre" required />
        </div>
        <div class="form-field">
          <label>Apellido:</label>
          <input v-model="cliente.Apellido" required />
        </div>
        <div class="form-field">
          <label>Tipo ID:</label>
          <input v-model="cliente.TipoID" required />
        </div>
        <div class="form-field">
          <label>Número ID:</label>
          <input v-model="cliente.NumeroID" required />
        </div>
        <div class="form-field">
          <label>Teléfono:</label>
          <input v-model="cliente.Telefono" />
        </div>
        <div class="form-field">
          <label>Email:</label>
          <input v-model="cliente.Email" type="email" />
        </div>
        <div class="form-field">
          <label>Dirección:</label>
          <input v-model="cliente.Direccion" />
        </div>
        <div style="grid-column: 1 / -1">
          <button type="submit">
            {{ editando ? "Actualizar" : "Registrar" }}
          </button>
          <button v-if="editando" @click="cancelarEdicion" type="button">
            Cancelar
          </button>
        </div>
      </form>
    </div>

    <div class="table-section">
      <h3>Listado de Clientes</h3>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Nombre Completo</th>
            <th>Identificación</th>
            <th>Teléfono</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in listaClientes" :key="c.id">
            <td>{{ c.id }}</td>
            <td>{{ c.Nombre }} {{ c.Apellido }}</td>
            <td>{{ c.TipoID }} {{ c.NumeroID }}</td>
            <td>{{ c.Telefono }}</td>
            <td>
              <button @click="editarCliente(c)">Editar</button>
              <button @click="eliminarCliente(c.id!)" class="danger">
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
