<template>
  <div>
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Gestionar Vendedores</h1>

    <div
      class="flex justify-between items-center mb-6 bg-white p-4 rounded-lg shadow-md"
    >
      <div class="relative">
        <input
          type="text"
          v-model="busqueda"
          @input="debouncedCargarVendedores"
          placeholder="Buscar por nombre, apellido o cédula..."
          class="input pl-10 w-72"
        />
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
          >🔍</span
        >
      </div>
      <button @click="abrirModalParaCrear" class="btn-primary">
        ✨ Registrar Nuevo Vendedor
      </button>
    </div>

    <div class="bg-white shadow-md rounded-lg overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="th">Nombre Completo</th>
            <th class="th">Cédula</th>
            <th class="th">Email</th>
            <th class="th text-right">Acciones</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-if="listaVendedores.length === 0">
            <td colspan="4" class="text-center py-10 text-gray-500">
              No se encontraron vendedores.
            </td>
          </tr>
          <tr v-for="v in listaVendedores" :key="v.id">
            <td class="td font-medium text-gray-900">
              {{ v.Nombre }} {{ v.Apellido }}
            </td>
            <td class="td">{{ v.Cedula }}</td>
            <td class="td">{{ v.Email }}</td>
            <td class="td text-right space-x-2">
              <button @click="verHistorial(v)" class="btn-secondary text-sm">
                Historial
              </button>
              <button @click="editarVendedor(v)" class="btn-edit">
                Editar
              </button>
              <button @click="confirmarEliminacion(v.id!)" class="btn-delete">
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
        {{ totalVendedores }} vendedores)
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
      <div class="bg-white p-8 rounded-lg shadow-2xl w-full max-w-lg">
        <h3 class="text-2xl font-bold mb-6">
          {{ editando ? "Editar" : "Registrar" }} Vendedor
        </h3>
        <form
          @submit.prevent="guardarVendedor"
          class="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-4"
        >
          <div>
            <label class="label">Nombre</label
            ><input v-model="vendedor.Nombre" class="input" required />
          </div>
          <div>
            <label class="label">Apellido</label
            ><input v-model="vendedor.Apellido" class="input" required />
          </div>
          <div>
            <label class="label">Cédula</label
            ><input v-model="vendedor.Cedula" class="input" required />
          </div>
          <div>
            <label class="label">Email</label
            ><input
              v-model="vendedor.Email"
              type="email"
              class="input"
              required
            />
          </div>
          <div class="md:col-span-2">
            <label class="label">Contraseña</label
            ><input
              v-model="vendedor.Contrasena"
              type="password"
              :placeholder="editando ? 'Dejar en blanco para no cambiar' : ''"
              :required="!editando"
              class="input"
            />
          </div>
          <div class="md:col-span-2 flex justify-end space-x-4 pt-4">
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

    <div
      v-if="vendedorSeleccionado"
      class="fixed inset-0 bg-black bg-opacity-60 flex justify-center items-center z-50"
    >
      <div
        class="bg-white p-6 rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col"
      >
        <h3 class="text-2xl font-bold mb-4">
          Historial de Ventas de: {{ vendedorSeleccionado.Nombre }}
          {{ vendedorSeleccionado.Apellido }}
        </h3>
        <div class="flex-grow overflow-y-auto border-t border-b">
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50 sticky top-0">
              <th class="th">N° Factura</th>
              <th class="th">Fecha</th>
              <th class="th">Cliente</th>
              <th class="th">Total</th>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
              <tr v-if="historialFacturas.length === 0">
                <td colspan="4" class="text-center py-10 text-gray-500">
                  Este vendedor no tiene ventas registradas.
                </td>
              </tr>
              <tr v-for="factura in historialFacturas" :key="factura.id">
                <td class="td">{{ factura.NumeroFactura }}</td>
                <td class="td">
                  {{ new Date(factura.fecha_emision).toLocaleString() }}
                </td>
                <td class="td">
                  {{ factura.Cliente.Nombre }} {{ factura.Cliente.Apellido }}
                </td>
                <td class="td font-semibold">
                  ${{ factura.Total.toFixed(2) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="mt-6 text-center">
          <button @click="vendedorSeleccionado = null" class="btn-secondary">
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
import { ref, onMounted, reactive, computed } from "vue";
import { backend } from "../../wailsjs/go/models";
import {
  RegistrarVendedor,
  ObtenerVendedoresPaginado,
  ActualizarVendedor,
  EliminarVendedor,
  ObtenerFacturasPorVendedor,
} from "../../wailsjs/go/backend/Db";
import NotificationModal from "../components/NotificationModal.vue";

type NotificationType = "success" | "error";

// --- STATE ---
const listaVendedores = ref<backend.Vendedor[]>([]);
const vendedor = ref(new backend.Vendedor());
const editando = ref(false);
const mostrarModal = ref(false);
const busqueda = ref("");
let debounceTimer: number;

const vendedorSeleccionado = ref<backend.Vendedor | null>(null);
const historialFacturas = ref<backend.Factura[]>([]);

const notification = reactive({
  show: false,
  message: "",
  type: "success" as NotificationType,
});

const paginaActual = ref(1);
const porPagina = ref(10);
const totalVendedores = ref(0);

const totalPaginas = computed(() =>
  Math.ceil(totalVendedores.value / porPagina.value)
);

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

const cargarVendedores = async () => {
  try {
    const response = await ObtenerVendedoresPaginado(
      paginaActual.value,
      porPagina.value,
      busqueda.value
    );
    listaVendedores.value = response.Records as backend.Vendedor[];
    totalVendedores.value = response.TotalRecords;
  } catch (error) {
    showNotification(`Error al cargar vendedores: ${error}`, "error");
  }
};

const debouncedCargarVendedores = () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    paginaActual.value = 1;
    cargarVendedores();
  }, 300);
};

const cambiarPagina = (nuevaPagina: number) => {
  if (nuevaPagina > 0 && nuevaPagina <= totalPaginas.value) {
    paginaActual.value = nuevaPagina;
    cargarVendedores();
  }
};

const abrirModalParaCrear = () => {
  editando.value = false;
  vendedor.value = new backend.Vendedor();
  mostrarModal.value = true;
};

const editarVendedor = (v: backend.Vendedor) => {
  editando.value = true;
  // Creamos una copia para no modificar la lista directamente
  vendedor.value = backend.Vendedor.createFrom(v);
  vendedor.value.Contrasena = ""; // La contraseña no se debe precargar por seguridad
  mostrarModal.value = true;
};

const cerrarModal = () => {
  mostrarModal.value = false;
};

const guardarVendedor = async () => {
  try {
    let resultado: string;
    if (editando.value) {
      resultado = await ActualizarVendedor(vendedor.value);
    } else {
      resultado = await RegistrarVendedor(vendedor.value);
    }
    showNotification(resultado, "success");
    cerrarModal();
    await cargarVendedores();
  } catch (error) {
    showNotification(`Error al guardar vendedor: ${error}`, "error");
  }
};

const confirmarEliminacion = (id: number) => {
  if (
    confirm(
      "¿Estás seguro de que quieres eliminar este vendedor? Esto podría afectar facturas históricas."
    )
  ) {
    eliminarVendedor(id);
  }
};

const eliminarVendedor = async (id: number) => {
  try {
    const resultado = await EliminarVendedor(id);
    showNotification(resultado, "success");
    await cargarVendedores();
  } catch (error) {
    showNotification(`Error al eliminar vendedor: ${error}`, "error");
  }
};

const verHistorial = async (v: backend.Vendedor) => {
  vendedorSeleccionado.value = v;
  historialFacturas.value = []; // Limpiar historial anterior
  try {
    const response = await ObtenerFacturasPorVendedor(v.id!, 1, 100); // Carga hasta 100 facturas
    historialFacturas.value = response.Records as backend.Factura[];
  } catch (error) {
    showNotification(`Error al cargar historial: ${error}`, "error");
  }
};

onMounted(cargarVendedores);
</script>

<style scoped>
@reference "../style.css";
</style>
