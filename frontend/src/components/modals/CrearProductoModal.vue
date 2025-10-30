<script setup lang="ts">
import { ref, watch } from "vue";
import { useAuthStore } from "@/stores/auth";
import { storeToRefs } from "pinia";
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
import { RegistrarProducto } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";

// Props para controlar el modal y pre-rellenar el código
const props = defineProps<{
  open: boolean;
  initialCodigo?: string;
}>();

const authStore = useAuthStore();
const { user: authenticatedUser } = storeToRefs(authStore);
// Eventos para comunicar el resultado al componente padre
const emit = defineEmits(["update:open", "product-created"]);

// Estado del formulario
const producto = ref<backend.NuevoProducto>(new backend.NuevoProducto());
const isLoading = ref(false);

// Sincronizar el código inicial cuando el modal se abre
watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      producto.value = new backend.NuevoProducto({
        VendedorUUID: authenticatedUser.value?.UUID,
        Codigo: props.initialCodigo || "",
        Nombre: "",
        PrecioVenta: 0,
        Stock: 0,
      });
    }
  }
);

async function handleSubmit() {
  if (!producto.value.Nombre || !producto.value.Codigo) {
    toast.error("Campos requeridos", {
      description: "El nombre y el código son obligatorios.",
    });
    return;
  }

  isLoading.value = true;
  try {
    const nuevoProducto = await RegistrarProducto(producto.value);
    toast.success("Producto Creado", {
      description: `El producto "${nuevoProducto.Nombre}" ha sido registrado.`,
    });
    emit("product-created", nuevoProducto);
    emit("update:open", false);
  } catch (error) {
    console.error("Error al registrar producto:", error);
    toast.error("Error al registrar", { description: `${error}` });
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => emit('update:open', value)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Crear Nuevo Producto</DialogTitle>
        <DialogDescription>
          Rellena los detalles del nuevo artículo. Haz clic en guardar cuando
          termines.
        </DialogDescription>
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="codigo" class="text-right">Código</Label>
          <Input
            id="codigo"
            v-model="producto.Codigo"
            class="col-span-3 h-10"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="nombre" class="text-right">Nombre</Label>
          <Input
            id="nombre"
            v-model="producto.Nombre"
            class="col-span-3 h-10"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="precio" class="text-right">Precio Venta</Label>
          <Input
            id="precio"
            type="number"
            v-model="producto.PrecioVenta"
            class="col-span-3 h-10"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="stock" class="text-right">Stock Inicial</Label>
          <Input
            id="stock"
            type="number"
            v-model="producto.Stock"
            class="col-span-3 h-10"
          />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)"
          >Cancelar</Button
        >
        <Button @click="handleSubmit" :disabled="isLoading">
          {{ isLoading ? "Guardando..." : "Guardar Producto" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
