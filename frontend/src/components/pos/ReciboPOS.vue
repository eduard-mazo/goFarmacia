<script setup lang="ts">
import { backend } from "@/../wailsjs/go/models";

defineProps<{
  factura: backend.Factura | null;
}>();
</script>

<template>
  <div
    class="w-[116mm] bg-white text-black font-mono text-[20px] leading-tight p-2 box-border"
    v-if="factura"
  >
    <div class="text-center">
      <p class="font-bold">goFarmacia</p>
      <p>NIT: 123.456.789-0</p>
      <p>Dirección de la farmacia</p>
    </div>
    <div class="border-t border-dashed border-black my-1"></div>
    <p>Factura: {{ factura.NumeroFactura }}</p>
    <p>Fecha: {{ new Date(factura.fecha_emision).toLocaleString() }}</p>
    <p>Cliente: {{ factura.Cliente.Nombre }} {{ factura.Cliente.Apellido }}</p>
    <p>Vendedor: {{ factura.Vendedor.Nombre }}</p>
    <div class="border-t border-dashed border-black my-1"></div>
    <table class="w-full">
      <thead>
        <tr>
          <th class="text-left font-normal border-b border-black">Cant</th>
          <th class="text-left font-normal border-b border-black">Producto</th>
          <th class="text-right font-normal border-b border-black">Total</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in factura.Detalles" :key="item.id">
          <td>{{ item.Cantidad }}</td>
          <td>{{ item.Producto.Nombre }}</td>
          <td class="text-right">${{ item.PrecioTotal.toFixed(2) }}</td>
        </tr>
      </tbody>
    </table>
    <div class="border-t border-dashed border-black my-1"></div>
    <div class="space-y-1">
      <p class="flex justify-between">
        <span>Subtotal:</span> <span>${{ factura.Subtotal.toFixed(2) }}</span>
      </p>
      <p class="flex justify-between">
        <span>IVA (19%):</span> <span>${{ factura.IVA.toFixed(2) }}</span>
      </p>
      <p class="flex justify-between font-bold">
        <span>Total:</span> <span>${{ factura.Total.toFixed(2) }}</span>
      </p>
    </div>
    <div class="border-t border-dashed border-black my-1"></div>
    <div class="text-center">
      <p>¡Gracias por su compra!</p>
    </div>
  </div>
</template>
