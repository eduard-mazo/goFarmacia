<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[700px]">
      <DialogHeader>
        <DialogTitle>Historial de Inventario</DialogTitle>
        <DialogDescription>
          Mostrando todos los movimientos para el producto: **{{
            producto?.Nombre
          }}**.
        </DialogDescription>
      </DialogHeader>

      <div v-if="isLoading" class="flex items-center justify-center h-48">
        <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
      </div>

      <Alert v-else-if="error" variant="destructive">
        <AlertCircle class="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>{{ error }}</AlertDescription>
      </Alert>

      <div v-else class="max-h-[60vh] overflow-y-auto pr-4">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead class="w-[180px]">Fecha</TableHead>
              <TableHead>Operación</TableHead>
              <TableHead class="text-right">Cambio</TableHead>
              <TableHead class="text-right">Stock Resultante</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="historial.length === 0">
              <TableCell :colspan="4" class="h-24 text-center"
                >No hay movimientos registrados para este producto.</TableCell
              >
            </TableRow>
            <TableRow v-for="op in historial" :key="op.id">
              <TableCell>{{ formatDateTime(op.timestamp) }}</TableCell>
              <TableCell>
                <Badge :variant="getOperationVariant(op.tipo_operacion)">{{
                  op.tipo_operacion
                }}</Badge>
              </TableCell>
              <TableCell
                :class="[
                  'text-right font-mono text-sm',
                  op.cantidad_cambio > 0 ? 'text-green-600' : 'text-red-600',
                ]"
              >
                {{ op.cantidad_cambio > 0 ? "+" : "" }}{{ op.cantidad_cambio }}
              </TableCell>
              <TableCell class="text-right font-mono text-sm">{{
                op.stock_resultante
              }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)"
          >Cerrar</Button
        >
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, watch, type PropType } from "vue";
import { Loader2, AlertCircle } from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { ObtenerHistorialStock } from "@/../wailsjs/go/backend/Db";
import type { backend } from "@/../wailsjs/go/models";

const props = defineProps({
  producto: {
    type: Object as PropType<backend.Producto | null>,
    required: true,
  },
  open: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["update:open"]);

const isLoading = ref(false);
const error = ref("");
const historial = ref<backend.OperacionStock[]>([]);

const fetchHistorial = async (productoId: number) => {
  if (!productoId) return;
  isLoading.value = true;
  error.value = "";
  try {
    historial.value = await ObtenerHistorialStock(productoId);
  } catch (err: any) {
    error.value = `No se pudo cargar el historial: ${err}`;
  } finally {
    isLoading.value = false;
  }
};

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen && props.producto) {
      fetchHistorial(props.producto.id);
    } else {
      historial.value = [];
      error.value = "";
    }
  }
);

const formatDateTime = (dateStr: string) =>
  new Date(dateStr).toLocaleString("es-CO", {
    dateStyle: "short",
    timeStyle: "medium",
  });

const getOperationVariant = (
  op: string
): "default" | "destructive" | "secondary" | "outline" => {
  switch (op.toUpperCase()) {
    case "VENTA":
      return "destructive";
    case "COMPRA":
      return "default";
    case "AJUSTE":
      return "outline";
    default:
      return "secondary";
  }
};
</script>
