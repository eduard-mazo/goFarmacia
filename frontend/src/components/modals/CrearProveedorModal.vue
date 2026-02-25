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
import { CrearProveedor } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";

const props = defineProps<{
  open: boolean;
}>();

const emit = defineEmits(["update:open", "proveedor-created"]);

const proveedor = ref(new backend.Proveedor());
const isLoading = ref(false);

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      proveedor.value = new backend.Proveedor({
        NIT: "",
        Nombre: "",
        Telefono: "",
        Email: "",
      });
    }
  }
);

async function handleSubmit() {
  if (!proveedor.value.Nombre) {
    toast.error("Campo requerido", {
      description: "El nombre del proveedor es obligatorio.",
    });
    return;
  }

  isLoading.value = true;
  try {
    await CrearProveedor(proveedor.value);
    toast.success("Proveedor Creado", {
      description: `El proveedor "${proveedor.value.Nombre}" ha sido registrado.`,
    });
    emit("proveedor-created");
    emit("update:open", false);
  } catch (error) {
    console.error("Error al registrar proveedor:", error);
    toast.error("Error al registrar", { description: `${error}` });
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => emit('update:open', value)">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Crear Nuevo Proveedor</DialogTitle>
        <DialogDescription>
          Rellena los detalles del nuevo proveedor. Haz clic en guardar cuando
          termines.
        </DialogDescription>
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="nit" class="text-right">NIT</Label>
          <Input
            id="nit"
            v-model="proveedor.NIT"
            class="col-span-3 h-10"
            placeholder="NIT del proveedor (ej. 900123456)"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="nombre" class="text-right">Nombre</Label>
          <Input
            id="nombre"
            v-model="proveedor.Nombre"
            class="col-span-3 h-10"
            placeholder="Nombre del proveedor"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="telefono" class="text-right">Teléfono</Label>
          <Input
            id="telefono"
            v-model="proveedor.Telefono"
            type="text"
            class="col-span-3 h-10"
            placeholder="Número de contacto"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="email" class="text-right">Email</Label>
          <Input
            id="email"
            v-model="proveedor.Email"
            type="email"
            class="col-span-3 h-10"
            placeholder="correo@ejemplo.com"
          />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)"
          >Cancelar</Button
        >
        <Button @click="handleSubmit" :disabled="isLoading">
          {{ isLoading ? "Guardando..." : "Guardar Proveedor" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
