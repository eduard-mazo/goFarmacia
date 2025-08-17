<template>
  <div>
    <h1>Gestionar Productos</h1>
    <div class="form-section">
      <h3>{{ editando ? "Editar" : "Registrar" }} Producto</h3>
      <form @submit.prevent="guardarProducto">
        <div class="form-field">
          <label>Código:</label>
          <input v-model="producto.Codigo" required />
        </div>
        <div class="form-field">
          <label>Nombre:</label>
          <input v-model="producto.Nombre" required />
        </div>
        <div class="form-field">
          <label>Precio de Venta:</label>
          <input
            v-model.number="producto.PrecioVenta"
            type="number"
            step="0.01"
            required
          />
        </div>
        <div class="form-field">
          <label>Stock:</label>
          <input v-model.number="producto.Stock" type="number" required />
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
      <h3>Listado de Productos</h3>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Código</th>
            <th>Nombre</th>
            <th>Precio</th>
            <th>Stock</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in listaProductos" :key="p.id">
            <td>{{ p.id }}</td>
            <td>{{ p.Codigo }}</td>
            <td>{{ p.Nombre }}</td>
            <td>${{ p.PrecioVenta?.toFixed(2) }}</td>
            <td>{{ p.Stock }}</td>
            <td>
              <button @click="editarProducto(p)">Editar</button>
              <button @click="eliminarProducto(p.id!)" class="danger">
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
