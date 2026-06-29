<script setup lang="ts">
import { Pencil, History, Trash2 } from "lucide-vue-next";
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
import { backend } from "@/../bridge/go/models";
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
  isEditDialogOpen.value = false;
}

function handleDeleteConfirm() {
  emit("delete", props.producto);
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
    <DropdownMenuContent align="end" class="w-40">
      <DropdownMenuItem @click="openEditDialog">
        <Pencil class="w-3.5 h-3.5 mr-2" />
        <span>Editar</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="openHistoryDialog">
        <History class="w-3.5 h-3.5 mr-2" />
        <span>Historial</span>
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
        <DialogTitle>Editar Producto</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-2">
        <div class="space-y-1.5">
          <Label for="prod-name">Nombre</Label>
          <Input id="prod-name" v-model="editableProduct.Nombre" />
        </div>
        <div class="space-y-1.5">
          <Label for="prod-code">Código</Label>
          <Input id="prod-code" v-model="editableProduct.Codigo" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div class="space-y-1.5">
            <Label for="prod-price">Precio Venta</Label>
            <Input id="prod-price" type="number" v-model.number="editableProduct.PrecioVenta" />
          </div>
          <div class="space-y-1.5">
            <Label for="prod-stock">Stock</Label>
            <Input id="prod-stock" type="number" v-model.number="editableProduct.Stock" />
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
        <AlertDialogTitle>¿Eliminar producto?</AlertDialogTitle>
        <AlertDialogDescription>
          Esta acción no se puede deshacer. Se eliminará permanentemente
          <span class="font-semibold">{{ props.producto.Nombre }}</span>.
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
