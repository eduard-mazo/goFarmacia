<script setup lang="ts">
import { ref, watch } from "vue";
import { backend } from "@/../wailsjs/go/models";
import { Dialog, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import ReciboPOS from "@/components/pos/ReciboPOS.vue";
import { Printer } from "lucide-vue-next";

const props = defineProps<{
  factura: backend.Factura | null;
}>();

const emit = defineEmits(["update:open"]);

const isOpen = ref(false);

watch(
  () => props.factura,
  (newFactura) => {
    if (newFactura) {
      isOpen.value = true;
    }
  }
);

watch(isOpen, (newIsOpenValue) => {
  if (!newIsOpenValue) {
    setTimeout(() => {
      emit("update:open", false);
    }, 300);
  }
});

function handlePrint() {
  setTimeout(() => {
    window.print();
  }, 100);
}
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="max-w-xs md:max-w-md print-area p-2">
      <ReciboPOS v-if="props.factura" :factura="props.factura" />

      <DialogFooter class="print:hidden">
        <Button variant="outline" @click="isOpen = false">Cerrar</Button>
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