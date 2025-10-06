<script setup lang="ts">
import { ref, watch } from "vue";
import { backend } from "@/../wailsjs/go/models";
import { Dialog, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import ReciboPOS from "@/components/pos/ReciboPOS.vue";
import { Printer, Loader2 } from "lucide-vue-next";
import { toast } from "vue-sonner";

import { VerificarImpresora, ImprimirRecibo } from "@/../wailsjs/go/backend/Db";

const props = defineProps<{
  factura: backend.Factura | null;
}>();

const emit = defineEmits(["update:open"]);

const isOpen = ref(false);
const isPrinterAvailable = ref(false);
const isPrinting = ref(false);

watch(
  () => props.factura,
  (newFactura) => {
    if (newFactura) {
      isOpen.value = true;
    }
  }
);

watch(isOpen, async (newIsOpenValue) => {
  if (!newIsOpenValue) {
    setTimeout(() => {
      emit("update:open", false);
    }, 300);
  } else {
    isPrinterAvailable.value = await VerificarImpresora();
    if (!isPrinterAvailable.value) {
      toast.info("Impresora no detectada", {
        description: "La impresión de recibos físicos está deshabilitada.",
      });
    }
  }
});

async function handleThermalPrint() {
  if (!props.factura) return;
  isPrinting.value = true;
  try {
    await ImprimirRecibo(props.factura);
    toast.success("Recibo enviado a la impresora");
  } catch (error) {
    toast.error("Error de impresión", {
      description: `No se pudo conectar con la impresora. ${error}`,
    });
  } finally {
    isPrinting.value = false;
  }
}
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="max-w-xs md:max-w-md p-4">
      <ReciboPOS v-if="props.factura" :factura="props.factura" />

      <DialogFooter>
        <Button variant="outline" @click="isOpen = false">Cerrar</Button>
        <Button
          @click="handleThermalPrint"
          :disabled="!isPrinterAvailable || isPrinting"
        >
          <Loader2 v-if="isPrinting" class="w-4 h-4 mr-2 animate-spin" />
          <Printer v-else class="w-4 h-4 mr-2" />
          {{ isPrinting ? "Imprimiendo..." : "Imprimir" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
