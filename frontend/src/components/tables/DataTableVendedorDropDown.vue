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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { backend } from "@/../bridge/go/models";

const props = defineProps<{
  vendedor: backend.Vendedor;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: "edit", value: backend.Vendedor): void;
  (e: "delete", value: backend.Vendedor): void;
}>();

const editableVendedor = ref<backend.Vendedor>(
  backend.Vendedor.createFrom(props.vendedor)
);

const isEditDialogOpen = ref(false);
const isDeleteDialogOpen = ref(false);

function openEditDialog() {
  if (props.disabled) return;
  editableVendedor.value = backend.Vendedor.createFrom(props.vendedor);
  isEditDialogOpen.value = true;
}

function openDeleteDialog() {
  if (props.disabled) return;
  editableVendedor.value = backend.Vendedor.createFrom(props.vendedor);
  isDeleteDialogOpen.value = true;
}

function handleSaveChanges() {
  emit("edit", editableVendedor.value);
  isEditDialogOpen.value = false;
}

function handleDeleteConfirm() {
  emit("delete", props.vendedor);
  isDeleteDialogOpen.value = false;
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="h-7 w-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity" :disabled="props.disabled">
        <Pencil class="w-3.5 h-3.5" />
        <span class="sr-only">Abrir menú</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-36">
      <DropdownMenuItem @click="openEditDialog" :disabled="props.disabled">
        <Pencil class="w-3.5 h-3.5 mr-2" />
        <span>Editar</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="openDeleteDialog" class="text-destructive focus:text-destructive" :disabled="props.disabled">
        <Trash2 class="w-3.5 h-3.5 mr-2" />
        <span>Eliminar</span>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <Dialog v-model:open="isEditDialogOpen">
    <DialogContent class="sm:max-w-[440px]">
      <DialogHeader>
        <DialogTitle>Editar Vendedor</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-2">
        <div class="space-y-1.5">
          <Label for="vend-nombre">Nombre</Label>
          <Input id="vend-nombre" v-model="editableVendedor.Nombre" />
        </div>
        <div class="space-y-1.5">
          <Label for="vend-cedula">Cédula</Label>
          <Input id="vend-cedula" v-model="editableVendedor.Cedula" readonly disabled />
        </div>
        <div class="space-y-1.5">
          <Label for="vend-email">Email</Label>
          <Input id="vend-email" type="email" v-model="editableVendedor.Email" />
        </div>
        <div class="space-y-1.5">
          <Label>Rol</Label>
          <Select v-model="editableVendedor.Role">
            <SelectTrigger>
              <SelectValue placeholder="Selecciona un rol" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="cajero">Cajero</SelectItem>
              <SelectItem value="admin">Administrador</SelectItem>
            </SelectContent>
          </Select>
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
        <AlertDialogTitle>¿Eliminar vendedor?</AlertDialogTitle>
        <AlertDialogDescription>
          Esta acción no se puede deshacer. Se eliminará permanentemente
          <span class="font-semibold">{{ props.vendedor.Nombre }}</span>.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancelar</AlertDialogCancel>
        <AlertDialogAction :class="buttonVariants({ variant: 'destructive' })" @click="handleDeleteConfirm">
          Eliminar
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
