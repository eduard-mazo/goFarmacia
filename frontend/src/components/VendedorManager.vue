<template>
  <div>
    <h1>Gestionar Vendedores</h1>
    <div class="form-section">
      <h3>{{ editando ? "Editar" : "Registrar" }} Vendedor</h3>
      <form @submit.prevent="guardarVendedor">
        <div class="form-field">
          <label>Nombre:</label>
          <input v-model="vendedor.Nombre" required />
        </div>
        <div class="form-field">
          <label>Apellido:</label>
          <input v-model="vendedor.Apellido" required />
        </div>
        <div class="form-field">
          <label>Cédula:</label>
          <input v-model="vendedor.Cedula" required />
        </div>
        <div class="form-field">
          <label>Email:</label>
          <input v-model="vendedor.Email" type="email" required />
        </div>
        <div class="form-field">
          <label>Contraseña:</label>
          <input
            v-model="vendedor.Contrasena"
            type="password"
            :required="!editando"
          />
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
      <h3>Listado de Vendedores</h3>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Nombre Completo</th>
            <th>Cédula</th>
            <th>Email</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in listaVendedores" :key="v.id">
            <td>{{ v.id }}</td>
            <td>{{ v.Nombre }} {{ v.Apellido }}</td>
            <td>{{ v.Cedula }}</td>
            <td>{{ v.Email }}</td>
            <td>
              <button @click="editarVendedor(v)">Editar</button>
              <button @click="eliminarVendedor(v.id!)" class="danger">
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
