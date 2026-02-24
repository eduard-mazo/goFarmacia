<script setup lang="ts">
import { Pencil, Trash2 } from "lucide-vue-next";
import { ref } from "vue";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
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
      <Button variant="ghost" class="h-7 w-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity">
        <Pencil class="w-3.5 h-3.5" />
        <span class="sr-only">Abrir menú</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-36">
      <DropdownMenuItem @click="openEditDialog">
        <Pencil class="w-3.5 h-3.5 mr-2" />
        <span>Editar</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="openDeleteDialog" class="text-destructive focus:text-destructive">
        <Trash2 class="w-3.5 h-3.5 mr-2" />
        <span>Eliminar</span>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <Dialog v-model:open="isEditDialogOpen">
    <DialogContent class="sm:max-w-[440px]">
      <DialogHeader>
        <DialogTitle>Editar Proveedor</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-2">
        <div class="space-y-1.5">
          <Label for="prov-nombre">Nombre</Label>
          <Input id="prov-nombre" v-model="editableProveedor.Nombre" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div class="space-y-1.5">
            <Label for="prov-tel">Teléfono</Label>
            <Input id="prov-tel" v-model="editableProveedor.Telefono" />
          </div>
          <div class="space-y-1.5">
            <Label for="prov-email">Email</Label>
            <Input id="prov-email" type="email" v-model="editableProveedor.Email" />
          </div>
        </div>
      </div>
      <DialogFooter>
        <Button variant="ghost" @click="isEditDialogOpen = false">Cancelar</Button>
        <Button @click="handleSaveChanges">Guardar cambios</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <AlertDialog v-model:open="isDeleteDialogOpen">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>¿Eliminar proveedor?</AlertDialogTitle>
        <AlertDialogDescription>
          Esta acción no se puede deshacer. Se eliminará permanentemente
          <span class="font-semibold">{{ props.proveedor.Nombre }}</span>.
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
