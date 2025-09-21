<script setup lang="ts">
import { backend } from "@/../wailsjs/go/models";
import {
  Dialog,
  DialogContent,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import ReciboPOS from "@/components/pos/ReciboPOS.vue";
import { Printer } from "lucide-vue-next";

const props = defineProps<{
  factura: backend.Factura | null;
}>();

// El v-model:open se manejará directamente en el template
const emit = defineEmits(["update:open"]);

function handlePrint() {
  // Pequeño truco para asegurar que el DOM está listo antes de imprimir
  setTimeout(() => {
    window.print();
  }, 100);
}
</script>

<template>
  <Dialog
    :open="!!props.factura"
    @update:open="(val) => !val && emit('update:open', false)"
  >
    <DialogContent class="max-w-xs md:max-w-md print-area p-2">
      <ReciboPOS :factura="props.factura" />
      <DialogFooter class="print:hidden">
        <Button variant="outline" @click="emit('update:open', false)"
          >Cerrar</Button
        >
        <Button @click="handlePrint">
          <Printer class="w-4 h-4 mr-2" />
          Imprimir
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<style>
@media print {
  body * {
    visibility: hidden;
  }

  .print-area,
  .print-area * {
    visibility: visible;
  }

  .print-area {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%;
    margin: 0;
    padding: 0;
    border: none;
  }
}
</style>
