<script setup lang="ts">
import { backend } from "@/../wailsjs/go/models";

defineProps<{
  factura: backend.Factura | null;
}>();

const formatCurrency = (value: number) => {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    maximumFractionDigits: 0,
  }).format(value);
};

const formatDate = (dateString: string) => {
  if (!dateString) return "---";
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return "Fecha inválida";
  return date.toLocaleString("es-CO", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};
</script>

<template>
  <div
    class="max-w-full bg-white text-black font-mono text-[14px] leading-normal p-4 box-border"
    v-if="factura"
  >
    <div class="text-center mb-2">
      <p class="font-bold text-lg">DROGUERÍA LUNA</p>
      <p>NIT: 70.120.237</p>
      <p>Medellín, Antioquia</p>
    </div>
    <div class="border-t border-dashed border-black my-2"></div>
    <div class="space-y-1">
      <p><span class="font-bold">Factura:</span> {{ factura.NumeroFactura }}</p>
      <p>
        <span class="font-bold">Fecha:</span>
        {{ formatDate(factura.FechaEmision) }}
      </p>
      <p>
        <span class="font-bold">Cliente:</span> {{ factura.Cliente.Nombre }}
        {{ factura.Cliente.Apellido }}
      </p>
      <p>
        <span class="font-bold">Vendedor:</span> {{ factura.Vendedor.Nombre }}
      </p>
    </div>
    <div class="border-t border-dashed border-black my-2"></div>
    <table class="w-full text-sm">
      <thead>
        <tr>
          <th
            class="text-left font-bold border-b-2 border-dashed border-black pb-1"
          >
            Cant
          </th>
          <th
            class="text-left font-bold border-b-2 border-dashed border-black pb-1"
          >
            Producto
          </th>
          <th
            class="text-right font-bold border-b-2 border-dashed border-black pb-1"
          >
            Total
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in factura.Detalles" :key="item.UUID">
          <td class="py-1 align-top">{{ item.Cantidad }}</td>
          <td class="py-1">{{ item.Producto.Nombre }}</td>
          <td class="text-right py-1 align-top">
            {{ formatCurrency(item.PrecioTotal) }}
          </td>
        </tr>
      </tbody>
    </table>
    <div class="border-t border-dashed border-black my-2"></div>
    <div class="space-y-1 text-base">
      <p class="flex justify-between font-bold text-lg">
        <span>Total:</span> <span>{{ formatCurrency(factura.Total) }}</span>
      </p>
    </div>
    <div class="border-t border-dashed border-black my-2"></div>
    <div class="text-center mt-2">
      <p>¡Gracias por su compra!</p>
    </div>
  </div>
</template>
