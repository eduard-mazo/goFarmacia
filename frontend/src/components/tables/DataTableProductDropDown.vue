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
import HistorialStockModal from "@/components/modals/HistorialStockModal.vue";

// --- Props and Emits ---
const props = defineProps<{
  producto: backend.Producto;
}>();

const emit = defineEmits<{
  (e: "edit", value: backend.Producto): void;
  (e: "delete", value: backend.Producto): void;
}>();

// --- State for Dialogs ---
const editableProduct = ref<backend.Producto>(
  backend.Producto.createFrom(props.producto)
);

const isEditDialogOpen = ref(false);
const isDeleteDialogOpen = ref(false);
const isHistoryDialogOpen = ref(false);

// --- Handlers ---
function openEditDialog() {
  editableProduct.value = backend.Producto.createFrom(props.producto);
  isEditDialogOpen.value = true;
}

function openDeleteDialog() {
  editableProduct.value = backend.Producto.createFrom(props.producto);
  isDeleteDialogOpen.value = true;
}

function openHistoryDialog() {
  editableProduct.value = backend.Producto.createFrom(props.producto);
  isHistoryDialogOpen.value = true;
}

function handleSaveChanges() {
  emit("edit", editableProduct.value);
  isEditDialogOpen.value = false; // Close dialog after saving
}

function handleDeleteConfirm() {
  emit("delete", props.producto);
  isDeleteDialogOpen.value = false; // Ensure dialog closes
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
        <span>Editar producto</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="openHistoryDialog">
        <span>Historial</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="openDeleteDialog" class="text-red-600">
        <span>Eliminar producto</span>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
  <Dialog v-model:open="isEditDialogOpen">
    <DialogContent class="w-11/12 md:max-w-[700px]">
      <DialogHeader>
        <DialogTitle>Editar Producto</DialogTitle>
        <DialogDescription>
          Realiza cambios en el producto aquí. Haz clic en guardar cuando hayas
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
          <Label for="code" class="text-right">Código</Label>
          <Input
            id="code"
            v-model="editableProduct.Codigo"
            class="col-span-3"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="price" class="text-right">Precio Venta</Label>
          <Input
            id="price"
            type="number"
            v-model.number="editableProduct.PrecioVenta"
            class="col-span-3"
          />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="stock" class="text-right">Stock</Label>
          <Input
            id="stock"
            type="number"
            v-model.number="editableProduct.Stock"
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
          producto
          <span class="font-semibold">{{ props.producto.Nombre }}</span> de la
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
  <HistorialStockModal
    v-model:open="isHistoryDialogOpen"
    :producto="editableProduct"
  />
</template>
