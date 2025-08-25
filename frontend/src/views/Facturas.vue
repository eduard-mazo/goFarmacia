<template>
  <div>
    <h1 class="text-2xl font-bold mb-6">Historial de Facturas</h1>

    <div class="bg-white shadow overflow-hidden sm:rounded-lg">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="th">N° Factura</th>
            <th class="th">Fecha</th>
            <th class="th">Cliente</th>
            <th class="th">Vendedor</th>
            <th class="th">Meetodo de Pago</th>
            <th class="th">Total</th>
            <th class="th text-right">Acciones</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="factura in facturas" :key="factura.id">
            <td class="td">{{ factura.NumeroFactura }}</td>
            <td class="td">
              {{ new Date(factura.fecha_emision).toLocaleString() }}
            </td>
            <td class="td">
              {{ factura.Cliente.Nombre }} {{ factura.Cliente.Apellido }}
            </td>
            <td class="td">{{ factura.Vendedor.Nombre }}</td>
            <td class="td font-semibold">{{ factura.MetodoPago }}</td>
            <td class="td font-semibold">${{ factura.Total.toFixed(2) }}</td>
            <td class="td text-right">
              <button
                @click="verDetalle(factura.id!)"
                class="text-indigo-600 hover:text-indigo-900"
              >
                Ver Recibo
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="facturaSeleccionada"
      class="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center"
    >
      <div class="bg-white p-4 rounded-lg shadow-xl">
        <ReciboPOS :factura="facturaSeleccionada" />
        <div class="mt-4 text-center">
          <button
            @click="facturaSeleccionada = null"
            class="bg-gray-500 text-white px-4 py-2 rounded"
          >
            Cerrar
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { backend } from "../../wailsjs/go/models";
import {
  ObtenerFacturas,
  ObtenerDetalleFactura,
} from "../../wailsjs/go/backend/Db";
import ReciboPOS from "../components/ReciboPOS.vue";

const facturas = ref<backend.Factura[]>([]);
const facturaSeleccionada = ref<backend.Factura | null>(null);

const cargarFacturas = async () => {
  try {
    facturas.value = await ObtenerFacturas();
  } catch (error) {
    alert(`Error al cargar facturas: ${error}`);
  }
};

const verDetalle = async (id: number) => {
  try {
    facturaSeleccionada.value = await ObtenerDetalleFactura(id);
  } catch (error) {
    alert(`Error al obtener detalle: ${error}`);
  }
};

onMounted(cargarFacturas);
</script>

<style scoped>
@reference "../style.css";
.th {
  @apply px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider;
}
.td {
  @apply px-6 py-4 whitespace-nowrap text-sm text-gray-700;
}
</style>
