<script setup lang="ts">
// ... todo el <script setup> del componente de ventas que te proporcioné antes ...
import { ref, computed, watch } from "vue";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardFooter,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Search, Trash2, UserSearch } from "lucide-vue-next";
import { toast } from "vue-sonner";

import { ImportaCSV, SelectFile } from "../../../wailsjs/go/backend/Db";

// --- INTERFACES Y DATOS DE PRUEBA ---
interface Producto {
  id: number;
  codigo: string;
  nombre: string;
  precio: number;
  stock: number;
}

interface ItemCarrito extends Producto {
  cantidad: number;
}

// Mock de productos (simula una base de datos)
const productos = ref<Producto[]>([
  {
    id: 1,
    codigo: "P001",
    nombre: "Paracetamol 500mg x20",
    precio: 2500,
    stock: 50,
  },
  {
    id: 2,
    codigo: "P002",
    nombre: "Ibuprofeno 400mg x10",
    precio: 3500,
    stock: 30,
  },
  {
    id: 3,
    codigo: "P003",
    nombre: "Vitamina C 1000mg Efervescente",
    precio: 15000,
    stock: 25,
  },
  {
    id: 4,
    codigo: "P004",
    nombre: "Amoxicilina 500mg Cápsulas",
    precio: 4500,
    stock: 15,
  },
  {
    id: 5,
    codigo: "L001",
    nombre: "Leche Deslactosada 1L",
    precio: 4200,
    stock: 100,
  },
  { id: 6, codigo: "A001", nombre: "Aspirina 100mg", precio: 8000, stock: 40 },
]);

// --- ESTADO DEL COMPONENTE ---
const busqueda = ref("");
const carrito = ref<ItemCarrito[]>([]);
const metodoPago = ref("efectivo");
const efectivoRecibido = ref<number | undefined>(undefined);
const clienteSeleccionado = ref("Cliente General");

// --- LÓGICA DE BÚSQUEDA Y AUTOCOMPLETADO ---
const resultadosBusqueda = computed(() => {
  if (busqueda.value.length < 2) return [];
  return productos.value.filter(
    (p) =>
      p.nombre.toLowerCase().includes(busqueda.value.toLowerCase()) ||
      p.codigo.toLowerCase().includes(busqueda.value.toLowerCase())
  );
});

// --- LÓGICA DEL CARRITO ---
function agregarAlCarrito(producto: Producto) {
  const itemExistente = carrito.value.find((item) => item.id === producto.id);

  if (itemExistente) {
    if (itemExistente.cantidad < producto.stock) {
      itemExistente.cantidad++;
    } else {
      toast.warning("Stock máximo alcanzado", {
        description: `No hay más stock disponible para ${producto.nombre}.`,
      });
    }
  } else {
    carrito.value.push({ ...producto, cantidad: 1 });
  }
  busqueda.value = "";
}

function manejarBusquedaConEnter() {
  if (resultadosBusqueda.value.length === 1 && resultadosBusqueda.value[0]) {
    agregarAlCarrito(resultadosBusqueda.value[0]);
  }
}

function eliminarDelCarrito(idProducto: number) {
  carrito.value = carrito.value.filter((p) => p.id !== idProducto);
}

function actualizarCantidad(idProducto: number, nuevaCantidad: number) {
  const item = carrito.value.find((p) => p.id === idProducto);
  if (item) {
    if (nuevaCantidad > 0 && nuevaCantidad <= item.stock) {
      item.cantidad = nuevaCantidad;
    } else if (nuevaCantidad > item.stock) {
      item.cantidad = item.stock;
      toast.warning("Stock máximo alcanzado", {
        description: `El stock disponible es de ${item.stock} unidades.`,
      });
    }
  }
}

// --- LÓGICA DE LA VENTA ---
const total = computed(() =>
  carrito.value.reduce((acc, item) => acc + item.precio * item.cantidad, 0)
);

const cambio = computed(() => {
  if (
    metodoPago.value === "efectivo" &&
    efectivoRecibido.value &&
    efectivoRecibido.value > 0
  ) {
    const valor = efectivoRecibido.value - total.value;
    return valor >= 0 ? valor : 0;
  }
  return 0;
});

watch(metodoPago, (nuevoMetodo) => {
  if (nuevoMetodo !== "efectivo") {
    efectivoRecibido.value = undefined;
  }
});

function finalizarVenta() {
  if (carrito.value.length === 0) {
    toast.error("El carrito está vacío", {
      description: "Agrega productos antes de finalizar la venta.",
    });
    return;
  }
  console.log("Venta finalizada:", {
    cliente: clienteSeleccionado.value,
    metodoPago: metodoPago.value,
    total: total.value,
    efectivoRecibido: efectivoRecibido.value,
    cambio: cambio.value,
    items: carrito.value,
  });
  toast.success("Venta realizada con éxito!", {
    description: `Total: $${total.value.toLocaleString()}`,
  });
  carrito.value = [];
  efectivoRecibido.value = undefined;
  busqueda.value = "";
}

async function handleImportProductos() {
  try {
    const filePath = await SelectFile();
    console.log(filePath);

    if (filePath) {
      // Llama a la función del backend.
      // No necesita 'await' porque la función en Go retorna de inmediato.
      ImportaCSV(filePath, "Productos");

      // Aquí puedes mostrar una notificación al usuario tipo "Importación iniciada..."
      alert(
        "La importación de productos ha comenzado. Revisa la consola del backend para ver el progreso."
      );
    }
  } catch (error) {
    console.error("Error al seleccionar archivo:", error);
  }
}
</script>

<template>
  <div class="grid grid-cols-10 gap-6 h-[calc(100vh-8rem)]">
    <div class="col-span-10 lg:col-span-7 flex flex-col gap-6">
      <Button @click="handleImportProductos" class="w-full h-12 text-lg">
        Cargar
      </Button>
      <Card class="flex-1 shadow-lg overflow-hidden">
        <CardContent class="p-4 h-full">
          <div class="relative">
            <Search
              class="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground"
            />
            <Input
              v-model="busqueda"
              placeholder="Buscar producto por código o nombre..."
              @keyup.enter="manejarBusquedaConEnter"
              class="pl-10 text-lg"
            />
            <div
              v-if="resultadosBusqueda.length > 0"
              class="absolute z-10 w-full mt-2 border rounded-lg bg-card shadow-xl max-h-60 overflow-y-auto"
            >
              <ul>
                <li
                  v-for="producto in resultadosBusqueda"
                  :key="producto.id"
                  class="p-3 hover:bg-muted cursor-pointer flex justify-between items-center"
                  @click="agregarAlCarrito(producto)"
                >
                  <div>
                    <p class="font-semibold">{{ producto.nombre }}</p>
                    <p class="text-sm text-muted-foreground">
                      Código: {{ producto.codigo }} | Stock:
                      {{ producto.stock }}
                    </p>
                  </div>
                  <span class="font-mono text-lg"
                    >${{ producto.precio.toLocaleString() }}</span
                  >
                </li>
              </ul>
            </div>
          </div>
          <div class="h-full overflow-y-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead class="w-[100px]">Código</TableHead>
                  <TableHead>Producto</TableHead>
                  <TableHead class="text-center">Cantidad</TableHead>
                  <TableHead class="text-right">Precio Unit.</TableHead>
                  <TableHead class="text-right">Subtotal</TableHead>
                  <TableHead class="text-center">Acción</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <template v-if="carrito.length > 0">
                  <TableRow v-for="item in carrito" :key="item.id">
                    <TableCell class="font-mono">{{ item.codigo }}</TableCell>
                    <TableCell class="font-medium">{{ item.nombre }}</TableCell>
                    <TableCell class="text-center">
                      <Input
                        type="number"
                        class="w-20 text-center mx-auto"
                        :model-value="item.cantidad"
                        @update:model-value="
                          actualizarCantidad(item.id, Number($event))
                        "
                        min="1"
                        :max="item.stock"
                      />
                    </TableCell>
                    <TableCell class="text-right font-mono"
                      >${{ item.precio.toLocaleString() }}</TableCell
                    >
                    <TableCell class="text-right font-mono"
                      >${{
                        (item.precio * item.cantidad).toLocaleString()
                      }}</TableCell
                    >
                    <TableCell class="text-center">
                      <Button
                        size="icon"
                        variant="ghost"
                        @click="eliminarDelCarrito(item.id)"
                      >
                        <Trash2 class="w-5 h-5 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                </template>
                <TableRow v-else>
                  <TableCell
                    colspan="6"
                    class="text-center h-24 text-muted-foreground"
                  >
                    El carrito está vacío
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>

    <div class="col-span-10 lg:col-span-3">
      <Card class="shadow-lg">
        <CardHeader>
          <CardTitle>Gestión de Venta</CardTitle>
        </CardHeader>
        <CardContent class="space-y-6">
          <div class="space-y-2">
            <Label for="cliente">Cliente</Label>
            <div class="flex gap-2">
              <Input id="cliente" :value="clienteSeleccionado" readonly />
              <Button variant="outline" size="icon">
                <UserSearch class="w-5 h-5" />
              </Button>
            </div>
          </div>
          <div class="space-y-2">
            <Label>Método de Pago</Label>
            <Select v-model="metodoPago">
              <SelectTrigger>
                <SelectValue placeholder="Seleccione un método" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="efectivo">Efectivo</SelectItem>
                <SelectItem value="tarjeta">Tarjeta</SelectItem>
                <SelectItem value="transferencia">Transferencia</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div v-if="metodoPago === 'efectivo'" class="space-y-2">
            <Label for="efectivo">Efectivo Recibido</Label>
            <Input
              id="efectivo"
              type="number"
              v-model="efectivoRecibido"
              placeholder="$ 0"
              class="text-right h-12 text-lg font-mono"
            />
          </div>
          <div class="space-y-3 pt-4 border-t">
            <div class="flex justify-between items-center text-lg">
              <span class="text-muted-foreground">Total</span>
              <span class="font-bold font-mono text-2xl"
                >${{ total.toLocaleString() }}</span
              >
            </div>
            <div
              v-if="metodoPago === 'efectivo' && cambio > 0"
              class="flex justify-between items-center text-lg"
            >
              <span class="text-muted-foreground">Cambio</span>
              <span class="font-bold font-mono text-green-400 text-2xl"
                >${{ cambio.toLocaleString() }}</span
              >
            </div>
          </div>
        </CardContent>
        <CardFooter>
          <Button @click="finalizarVenta" class="w-full h-12 text-lg">
            Finalizar Venta
          </Button>
        </CardFooter>
      </Card>
    </div>
  </div>
</template>
