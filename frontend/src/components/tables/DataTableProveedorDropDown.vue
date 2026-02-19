<script setup lang="ts">
import { MoreHorizontal } from "lucide-vue-next";
import { ref } from "vue";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { backend } from "@/../wailsjs/go/models";

const props = defineProps<{
  proveedor: backend.Proveedor;
}>();

const emit = defineEmits<{
  (e: "edit", value: backend.Proveedor): void;
  (e: "delete", value: backend.Proveedor): void;
}>();

const editableProveedor = ref<backend.Proveedor>(
  backend.Proveedor.createFrom(props.proveedor)
);

const isEditDialogOpen = ref(false);
const isDeleteDialogOpen = ref(false);

function openEditDialog() {
  editableProveedor.value = backend.Proveedor.createFrom(props.proveedor);
  isEditDialogOpen.value = true;
}

function openDeleteDialog() {
  editableProveedor.value = backend.Proveedor.createFrom(props.proveedor);
  isDeleteDialogOpen.value = true;
}

function handleSaveChanges() {
  emit("edit", editableProveedor.value);
  isEditDialogOpen.value = false;
}

function handleDeleteConfirm() {
  emit("delete", props.proveedor);
  isDeleteDialogOpen.value = false;
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Abrir menú</span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuLabel>Acciones</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem @click="openEditDialog">
        <span>Editar proveedor</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="openDeleteDialog" class="text-red-600">
        <span>Eliminar proveedor</span>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <Dialog v-model:open="isEditDialogOpen">
    <DialogContent class="w-11/12 md:max-w-[700px]">
      <DialogHeader>
        <DialogTitle>Editar Proveedor</DialogTitle>
        <DialogDescription>
          Realiza cambios en el proveedor aquí. Haz clic en guardar cuando hayas
          terminado.
        </DialogDescription>
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="nombre" class="text-right">Nombre</Label>
          <Input
            id="nombre"
            v-model="editableProveedor.Nombre"
            class="col-span-3"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="telefono" class="text-right">Teléfono</Label>
          <Input
            id="telefono"
            v-model="editableProveedor.Telefono"
            class="col-span-3"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="email" class="text-right">Email</Label>
          <Input
            id="email"
            type="email"
            v-model="editableProveedor.Email"
            class="col-span-3"
          />
        </div>
      </div>
      <DialogFooter>
        <Button type="submit" @click="handleSaveChanges"
          >Guardar cambios</Button
        >
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <AlertDialog v-model:open="isDeleteDialogOpen">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>¿Estás absolutamente seguro?</AlertDialogTitle>
        <AlertDialogDescription>
          Esta acción no se puede deshacer. Esto eliminará permanentemente el
          proveedor
          <span class="font-semibold">{{ props.proveedor.Nombre }}</span> de la
          base de datos.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancelar</AlertDialogCancel>
        <AlertDialogAction
          :class="buttonVariants({ variant: 'destructive' })"
          @click="handleDeleteConfirm"
        >
          Eliminar
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
