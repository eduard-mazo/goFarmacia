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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "vue-sonner";
import { RegistrarCliente } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";

const props = defineProps<{
  open: boolean;
}>();

const emit = defineEmits(["update:open", "client-created"]);

const cliente = ref(new backend.Cliente());
const isLoading = ref(false);
const tiposDocumento = ["CC", "CE", "NIT", "PASAPORTE"];

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      cliente.value = new backend.Cliente({
        Nombre: "",
        Apellido: "",
        TipoID: "CC", // --- CORRECCIÓN: Valor por defecto para el dropdown
        NumeroID: "",
        Telefono: "",
        Direccion: "",
        Email: "",
      });
    }
  }
);

async function handleSubmit() {
  if (!cliente.value.Nombre || !cliente.value.NumeroID) {
    toast.error("Campos requeridos", {
      description: "El nombre y el número de documento son obligatorios.",
    });
    return;
  }

  isLoading.value = true;
  try {
    const nuevoCliente = await RegistrarCliente(cliente.value);
    toast.success("Cliente Creado", {
      description: `El cliente "${nuevoCliente.Nombre}" ha sido registrado.`,
    });
    emit("client-created", nuevoCliente);
    emit("update:open", false);
  } catch (error) {
    console.error("Error al registrar cliente:", error);
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
        <DialogTitle>Crear Nuevo Cliente</DialogTitle>
        <DialogDescription>
          Rellena los detalles del nuevo cliente. Haz clic en guardar cuando
          termines.
        </DialogDescription>
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="nombre" class="text-right">Nombre</Label>
          <Input
            id="nombre"
            v-model="cliente.Nombre"
            class="col-span-3 h-10"
            placeholder="Nombre del cliente"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="apellidos" class="text-right">Apellidos</Label>
          <Input
            id="apellidos"
            v-model="cliente.Apellido"
            class="col-span-3 h-10"
            placeholder="Apellido del cliente"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="tipoId" class="text-right">Tipo ID</Label>
          <Select v-model="cliente.TipoID">
            <SelectTrigger class="col-span-3 h-10">
              <SelectValue placeholder="Seleccione un tipo" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="tipo in tiposDocumento"
                :key="tipo"
                :value="tipo"
              >
                {{ tipo }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="numeroId" class="text-right">Número ID</Label>
          <Input
            id="numeroId"
            v-model="cliente.NumeroID"
            class="col-span-3 h-10"
            placeholder="Número de documento"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="email" class="text-right">Email</Label>
          <Input
            id="email"
            v-model="cliente.Email"
            type="email"
            class="col-span-3 h-10"
            placeholder="correo@ejemplo.com"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="dirección" class="text-right">Dirección</Label>
          <Input
            id="dirección"
            v-model="cliente.Direccion"
            class="col-span-3 h-10"
            placeholder="Dirección de residencia"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="telefono" class="text-right">Teléfono</Label>
          <Input
            id="telefono"
            v-model="cliente.Telefono"
            type="text"
            class="col-span-3 h-10"
            placeholder="Número de contacto"
          />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)"
          >Cancelar</Button
        >
        <Button @click="handleSubmit" :disabled="isLoading">
          {{ isLoading ? "Guardando..." : "Guardar Cliente" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
