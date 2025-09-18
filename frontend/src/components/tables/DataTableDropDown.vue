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

// Define props to accept a Vendedor object
const props = defineProps<{
  vendedor: backend.Vendedor;
}>();

const emit = defineEmits<{
  (e: "edit", value: backend.Vendedor): void;
  (e: "delete", value: backend.Vendedor): void;
}>();

// --- State for Dialogs ---
const editableProduct = ref<backend.Vendedor>(
  backend.Vendedor.createFrom(props.vendedor)
);

const isEditDialogOpen = ref(false);
const isDeleteDialogOpen = ref(false);

// --- Handlers ---
function openEditDialog() {
  editableProduct.value = backend.Vendedor.createFrom(props.vendedor);
  isEditDialogOpen.value = true;
}

function openDeleteDialog() {
  editableProduct.value = backend.Vendedor.createFrom(props.vendedor);
  isEditDialogOpen.value = true;
}

function handleSaveChanges() {
  emit("edit", editableProduct.value);
  isEditDialogOpen.value = false; // Close dialog after saving
}

function handleDeleteConfirm() {
  emit("delete", props.vendedor);
  isDeleteDialogOpen.value = false; // Ensure dialog closes
}
</script>

<template>
  <!-- The DropdownMenu now only controls showing the menu items -->
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

      <!-- FIX: Use @click to programmatically open the correct dialog -->
      <DropdownMenuItem @click="openEditDialog">
        <span>Editar vendedor</span>
      </DropdownMenuItem>

      <DropdownMenuItem @click="openDeleteDialog" class="text-red-600">
        <span>Eliminar vendedor</span>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <!-- FIX: Dialog components are now siblings, not nested -->

  <!-- Edit Product Dialog -->
  <Dialog v-model:open="isEditDialogOpen">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Editar Vendedor</DialogTitle>
        <DialogDescription>
          Realiza cambios en el vendedor aquí. Haz clic en guardar cuando hayas
          terminado.
        </DialogDescription>
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="name" class="text-right">Nombre</Label>
          <Input
            id="name"
            v-model="editableProduct.Nombre"
            class="col-span-3"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="cedula" class="text-right">Cédula</Label>
          <Input
            id="cedula"
            v-model="editableProduct.Cedula"
            class="col-span-3"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="email" class="text-right">Email</Label>
          <Input
            id="email"
            type="email"
            v-model.number="editableProduct.Email"
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

  <!-- Delete Confirmation Dialog -->
  <AlertDialog v-model:open="isDeleteDialogOpen">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>¿Estás absolutamente seguro?</AlertDialogTitle>
        <AlertDialogDescription>
          Esta acción no se puede deshacer. Esto eliminará permanentemente el
          producto
          <span class="font-semibold">{{ props.vendedor.Nombre }}</span> de la
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
