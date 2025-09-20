<script setup lang="ts">
import { ref, watch } from "vue";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "vue-sonner";
import { ActualizarProducto } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";

const props = defineProps<{
  open: boolean;
  producto: backend.Producto;
}>();

const emit = defineEmits(["update:open", "stock-updated"]);

const nuevoStock = ref<number>(0);
const isLoading = ref(false);

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      nuevoStock.value = props.producto.Stock;
    }
  }
);

async function handleSubmit() {
  if (nuevoStock.value < 0 || nuevoStock.value === null) {
    toast.error("Cantidad inválida", {
      description: "El stock no puede ser un número negativo.",
    });
    return;
  }

  isLoading.value = true;
  try {
    const productoActualizado = new backend.Producto({
      ...props.producto,
      Stock: Number(nuevoStock.value)
    });

    await ActualizarProducto(productoActualizado);

    toast.success("Stock Actualizado", {
      description: `El stock de "${props.producto.Nombre}" ahora es ${nuevoStock.value}.`,
    });

    emit("stock-updated");
    emit("update:open", false);
  } catch (error) {
    console.error("Error al actualizar stock:", error);
    toast.error("Error al actualizar", { description: `${error}` });
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => emit('update:open', value)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Ajustar Stock</DialogTitle>
        <DialogDescription>
          Modifica la cantidad de inventario para el producto.
        </DialogDescription>
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <div class="font-medium">
          <p class="text-sm text-muted-foreground">{{ producto.Codigo }}</p>
          <p>{{ producto.Nombre }}</p>
        </div>
        <div class="grid grid-cols-3 items-center gap-4">
          <Label for="stock-actual" class="text-right">Stock Actual</Label>
          <Input
            id="stock-actual"
            :model-value="producto.Stock"
            class="col-span-2 h-10"
            readonly
            disabled
          />
        </div>
        <div class="grid grid-cols-3 items-center gap-4">
          <Label for="stock-nuevo" class="text-right">Nuevo Stock</Label>
          <Input
            id="stock-nuevo"
            v-model="nuevoStock"
            type="number"
            class="col-span-2 h-10"
          />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)"
          >Cancelar</Button
        >
        <Button @click="handleSubmit" :disabled="isLoading">
          {{ isLoading ? "Guardando..." : "Guardar Cambios" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
