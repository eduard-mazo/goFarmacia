<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/stores/auth";

// Componentes de UI
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
import { Search, Trash2, UserSearch, PlusCircle } from "lucide-vue-next";
import { toast } from "vue-sonner";
import CrearProductoModal from "@/components/CrearProductoModal.vue";
import BuscarClienteModal from "@/components/BuscarClienteModal.vue";

// Funciones e interfaces del Backend (generadas por Wails)
import {
  ObtenerClientesPaginado,
  ObtenerProductosPaginado,
  RegistrarVenta,
} from "../../../wailsjs/go/backend/Db";
import { backend } from "../../../wailsjs/go/models";

//[INTERFACES]
interface ItemCarrito extends backend.Producto {
  cantidad: number;
}

//[STORE] - AUTENTICACIÓN
const authStore = useAuthStore();
const { user: authenticatedUser } = storeToRefs(authStore);

//[COMPONENTE] - ESTADOS
const busqueda = ref("");
const productosEncontrados = ref<backend.Producto[]>([]);
const carrito = ref<ItemCarrito[]>([]);
const metodoPago = ref("efectivo");
const efectivoRecibido = ref<number | undefined>(undefined);
const clienteSeleccionado = ref("Cliente General");
const clienteID = ref(1);
const debounceTimer = ref<number | undefined>(undefined);
const isLoading = ref(false);

//[MODALS] - ESTADOS
const isCreateModalOpen = ref(false);
const isClienteModalOpen = ref(false);

//[REF] - DOM
const searchInputRef = ref<{ $el: HTMLInputElement } | null>(null);
const searchResultsContainerRef = ref<HTMLElement | null>(null);
const searchResultItemsRef = ref<HTMLLIElement[]>([]);

//[SEARCH] - LOGIC
const highlightedIndex = ref(-1);

watch(busqueda, (nuevoValor) => {
  highlightedIndex.value = -1;
  searchResultItemsRef.value = []; // Limpiar las refs de los items
  clearTimeout(debounceTimer.value);
  if (nuevoValor.length < 2) {
    productosEncontrados.value = [];
    return;
  }
  isLoading.value = true;
  debounceTimer.value = setTimeout(async () => {
    try {
      const resultado = await ObtenerProductosPaginado(1, 10, nuevoValor);
      productosEncontrados.value =
        (resultado.Records as backend.Producto[]) || [];
      if (productosEncontrados.value.length > 0) {
        highlightedIndex.value = 0;
      }
    } catch (error) {
      console.error("Error al buscar productos:", error);
      toast.error("Error de búsqueda", {
        description: "No se pudieron obtener los productos.",
      });
    } finally {
      isLoading.value = false;
    }
  }, 300);
});

//[CAR] - LOGIC
function agregarAlCarrito(producto: backend.Producto) {
  if (!producto) return;

  const itemExistente = carrito.value.find((item) => item.id === producto.id);

  if (itemExistente) {
    if (itemExistente.cantidad < itemExistente.Stock) {
      itemExistente.cantidad++;
    } else {
      toast.warning("Stock máximo alcanzado", {
        description: `No hay más stock disponible para ${producto.Nombre}.`,
      });
    }
  } else {
    const nuevoProducto = {
      ...new backend.Producto(producto),
      cantidad: 1,
    } as ItemCarrito;
    carrito.value.push(nuevoProducto);
  }

  busqueda.value = "";
  productosEncontrados.value = [];
  nextTick(() => {
    searchInputRef.value?.$el?.focus();
  });
}

function handleProductCreated(nuevoProducto: backend.Producto) {
  toast.success("Producto agregado al carrito", {
    description: `"${nuevoProducto.Nombre}" listo para la venta.`,
  });
  agregarAlCarrito(nuevoProducto);
}

function manejarBusquedaConEnter() {
  if (
    highlightedIndex.value >= 0 &&
    productosEncontrados.value[highlightedIndex.value]
  ) {
    agregarAlCarrito(productosEncontrados.value[highlightedIndex.value]!);
  } else if (productosEncontrados.value.length === 1) {
    agregarAlCarrito(productosEncontrados.value[0]!);
  }
}

//[KEYBOARD NAVIGATION]
function moverSeleccion(direccion: "arriba" | "abajo") {
  if (productosEncontrados.value.length === 0) return;
  if (direccion === "abajo") {
    highlightedIndex.value =
      (highlightedIndex.value + 1) % productosEncontrados.value.length;
  } else if (direccion === "arriba") {
    highlightedIndex.value =
      (highlightedIndex.value - 1 + productosEncontrados.value.length) %
      productosEncontrados.value.length;
  }
}

// --- LÓGICA DE SCROLL AUTOMÁTICO ---
watch(highlightedIndex, (newIndex) => {
  if (
    newIndex < 0 ||
    !searchResultsContainerRef.value ||
    !searchResultItemsRef.value[newIndex]
  ) {
    return;
  }
  const highlightedItem = searchResultItemsRef.value[newIndex];
  const container = searchResultsContainerRef.value;
  const itemTop = highlightedItem.offsetTop;
  const itemBottom = itemTop + highlightedItem.offsetHeight;
  const containerTop = container.scrollTop;
  const containerBottom = containerTop + container.clientHeight;

  if (itemTop < containerTop) {
    container.scrollTop = itemTop;
  } else if (itemBottom > containerBottom) {
    container.scrollTop = itemBottom - container.clientHeight;
  }
});

function eliminarDelCarrito(idProducto: number) {
  carrito.value = carrito.value.filter((p) => p.id !== idProducto);
}

function actualizarCantidad(idProducto: number, nuevaCantidad: number) {
  const item = carrito.value.find((p) => p.id === idProducto);
  if (item) {
    if (nuevaCantidad > 0 && nuevaCantidad <= item.Stock) {
      item.cantidad = nuevaCantidad;
    } else if (nuevaCantidad > item.Stock) {
      item.cantidad = item.Stock;
      toast.warning("Stock máximo alcanzado", {
        description: `El stock disponible es de ${item.Stock} unidades.`,
      });
    }
  }
}

//[SAILES] - LOGIC
const total = computed(() =>
  carrito.value.reduce((acc, item) => acc + item.PrecioVenta * item.cantidad, 0)
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

async function finalizarVenta() {
  if (carrito.value.length === 0) {
    toast.error("El carrito está vacío", {
      description: "Agrega productos antes de finalizar la venta.",
    });
    return;
  }

  // --- NUEVA VALIDACIÓN DE PRECIO DE VENTA ---
  const productoInvalido = carrito.value.find(
    (item) => !item.PrecioVenta || item.PrecioVenta <= 0
  );

  if (productoInvalido) {
    toast.error("Precio de Venta Inválido", {
      description: `El producto "${productoInvalido.Nombre}" tiene un precio de venta no válido. Por favor, corríjalo.`,
    });
    return;
  }
  // --- FIN DE LA VALIDACIÓN ---

  if (!authenticatedUser.value?.id) {
    toast.error("Vendedor no identificado", {
      description:
        "No se ha podido identificar al vendedor. Por favor, inicie sesión de nuevo.",
    });
    return;
  }

  const ventaRequest = new backend.VentaRequest({
    ClienteID: clienteID.value,
    VendedorID: authenticatedUser.value.id,
    MetodoPago: metodoPago.value,
    Productos: carrito.value.map((item) => ({
      ID: item.id,
      Cantidad: item.cantidad,
      PrecioUnitario: item.PrecioVenta,
    })),
  });

  try {
    const facturaCreada = await RegistrarVenta(ventaRequest);
    toast.success("¡Venta registrada con éxito!", {
      description: `Factura N° ${
        facturaCreada.NumeroFactura
      } por un total de $${facturaCreada.Total.toLocaleString()}`,
    });

    carrito.value = [];
    efectivoRecibido.value = undefined;
    busqueda.value = "";
    await cargarClienteGeneralPorDefecto();
  } catch (error) {
    console.error("Error al registrar la venta:", error);
    toast.error("Error al registrar la venta", {
      description: `Hubo un problema: ${error}`,
    });
  }
}

//[INIT] - CARGA INICIAL
async function cargarClienteGeneralPorDefecto() {
  try {
    const res = await ObtenerClientesPaginado(1, 1, "222222222");
    if (res && res.Records && res.Records.length > 0) {
      const clienteGeneral = res.Records[0] as backend.Cliente;
      clienteID.value = clienteGeneral.id;
      clienteSeleccionado.value = `${clienteGeneral.Nombre} ${clienteGeneral.Apellido}`;
    } else {
      clienteID.value = 1;
      clienteSeleccionado.value = "Cliente General";
    }
  } catch (error) {
    console.warn("No se pudo cargar el cliente general por defecto:", error);
    clienteID.value = 1;
    clienteSeleccionado.value = "Cliente General";
  }
}

function handleClienteSeleccionado(cliente: backend.Cliente) {
  clienteID.value = cliente.id;
  clienteSeleccionado.value = `${cliente.Nombre} ${cliente.Apellido}`;
  isClienteModalOpen.value = false;
}

// MANEJO DE ATAJOS DE TECLADO
function handleKeyDown(event: KeyboardEvent) {
  if (event.key === "F12") {
    event.preventDefault();
    finalizarVenta();
  } else if (event.key === "F10") {
    event.preventDefault();
    searchInputRef.value?.$el?.focus();
  }
}

//[LIFECYCLE] - MOUNT & UNMOUNT
onMounted(() => {
  window.addEventListener("keydown", handleKeyDown);
  nextTick(() => {
    cargarClienteGeneralPorDefecto();
    searchInputRef.value?.$el?.focus();
  });
});

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeyDown);
});
</script>

<template>
  <CrearProductoModal
    v-model:open="isCreateModalOpen"
    :initial-codigo="busqueda"
    @product-created="handleProductCreated"
  />
  <BuscarClienteModal
    v-model:open="isClienteModalOpen"
    @cliente-seleccionado="handleClienteSeleccionado"
  />

  <div class="grid grid-cols-10 gap-6 h-[calc(100vh-8rem)]">
    <div class="col-span-10 lg:col-span-7 flex flex-col gap-6">
      <Card class="flex-1 overflow-hidden">
        <CardContent class="p-4 h-full flex flex-col">
          <div class="relative">
            <Search
              class="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground"
            />
            <Input
              ref="searchInputRef"
              v-model="busqueda"
              placeholder="Buscar por nombre o código... (F10)"
              @keyup.enter="manejarBusquedaConEnter"
              @keydown.down.prevent="moverSeleccion('abajo')"
              @keydown.up.prevent="moverSeleccion('arriba')"
              class="pl-10 text-lg h-12"
            />
            <div
              v-if="productosEncontrados.length > 0"
              ref="searchResultsContainerRef"
              class="absolute z-10 w-full mt-2 border rounded-lg bg-card shadow-xl max-h-60 overflow-y-auto"
            >
              <ul>
                <li
                  v-for="(producto, index) in productosEncontrados"
                  :key="producto.id"
                  :ref="
                    (el) => {
                      if (el)
                        searchResultItemsRef[index] = el as HTMLLIElement;
                    }
                  "
                  class="p-3 hover:bg-muted cursor-pointer flex justify-between items-center"
                  :class="{
                    'bg-primary text-primary-foreground hover:bg-primary':
                      index === highlightedIndex,
                  }"
                  @click="agregarAlCarrito(producto)"
                >
                  <div>
                    <p class="font-semibold">{{ producto.Nombre }}</p>
                    <p
                      class="text-sm"
                      :class="
                        index === highlightedIndex
                          ? 'text-primary-foreground/80'
                          : 'text-muted-foreground'
                      "
                    >
                      Código: {{ producto.Codigo }} | Stock:
                      {{ producto.Stock }}
                    </p>
                  </div>
                  <span class="font-mono text-lg"
                    >${{ producto.PrecioVenta.toLocaleString() }}</span
                  >
                </li>
              </ul>
            </div>
            <div
              v-else-if="
                busqueda.length >= 2 &&
                !isLoading &&
                productosEncontrados.length === 0
              "
              class="absolute z-10 w-full mt-2 border rounded-lg bg-card shadow-xl p-4 text-center text-muted-foreground flex flex-col items-center gap-3"
            >
              <p>No se encontraron productos para "{{ busqueda }}"</p>
              <Button @click="isCreateModalOpen = true" variant="outline">
                <PlusCircle class="w-4 h-4 mr-2" />
                Crear Producto
              </Button>
            </div>
          </div>

          <div class="flex-1 overflow-y-auto mt-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead class="w-[100px]">Código</TableHead>
                  <TableHead>Producto</TableHead>
                  <TableHead class="text-center w-32">Cantidad</TableHead>
                  <TableHead class="text-right w-40">Precio Unit.</TableHead>
                  <TableHead class="text-right w-40">Subtotal</TableHead>
                  <TableHead class="text-center w-20">Acción</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <template v-if="carrito.length > 0">
                  <TableRow v-for="item in carrito" :key="item.id">
                    <TableCell class="font-mono">{{ item.Codigo }}</TableCell>
                    <TableCell class="font-medium">{{ item.Nombre }}</TableCell>
                    <TableCell class="text-center">
                      <Input
                        type="number"
                        class="w-20 text-center mx-auto h-10"
                        :model-value="item.cantidad"
                        @update:model-value="
                          actualizarCantidad(item.id, Number($event))
                        "
                        min="1"
                        :max="item.Stock"
                      />
                    </TableCell>
                    <TableCell class="text-right font-mono">
                      <Input
                        type="number"
                        class="w-32 text-right mx-auto h-10 font-mono"
                        v-model="item.PrecioVenta"
                        step="0.01"
                      />
                    </TableCell>
                    <TableCell class="text-right font-mono"
                      >${{
                        (item.PrecioVenta * item.cantidad).toLocaleString()
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
      <Card>
        <CardHeader>
          <CardTitle>Gestión de Venta</CardTitle>
        </CardHeader>
        <CardContent class="space-y-6">
          <div class="space-y-2">
            <Label for="cliente">Cliente</Label>
            <div class="flex gap-2">
              <Input
                id="cliente"
                :value="clienteSeleccionado"
                readonly
                class="h-10"
              />
              <Button
                @click="isClienteModalOpen = true"
                variant="outline"
                size="icon"
                class="h-10 w-10 flex-shrink-0"
              >
                <UserSearch class="w-5 h-5" />
              </Button>
            </div>
          </div>
          <div class="space-y-2">
            <Label>Método de Pago</Label>
            <Select v-model="metodoPago">
              <SelectTrigger class="h-10">
                <SelectValue placeholder="Seleccione un método" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="efectivo">Efectivo</SelectItem>
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
              class="text-right h-10 text-lg font-mono"
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
              v-if="metodoPago === 'efectivo' && efectivoRecibido && cambio > 0"
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
            Finalizar Venta (F12)
          </Button>
        </CardFooter>
      </Card>
    </div>
  </div>
</template>
