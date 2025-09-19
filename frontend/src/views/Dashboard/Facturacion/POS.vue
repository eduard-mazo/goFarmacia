<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/stores/auth";
import { useCartStore } from "@/stores/cart"; // [CORREGIDO]

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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
  Search,
  Trash2,
  UserSearch,
  PlusCircle,
  Save,
  RotateCcw,
  PackageOpen,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import CrearProductoModal from "@/components/modals/CrearProductoModal.vue";
import BuscarClienteModal from "@/components/modals/BuscarClienteModal.vue";
import ReciboVentaModal from "@/components/modals/ReciboVentaModal.vue";

// Funciones e interfaces del Backend
import {
  ObtenerClientesPaginado,
  ObtenerProductosPaginado,
  RegistrarVenta,
} from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";

// [STORE] - AUTENTICACIÓN Y CARRITO
const authStore = useAuthStore();
const { user: authenticatedUser } = storeToRefs(authStore);

const cartStore = useCartStore();
const { activeCart, savedCarts, activeCartTotal } = storeToRefs(cartStore);

// [COMPONENTE] - ESTADOS
const busqueda = ref("");
const productosEncontrados = ref<backend.Producto[]>([]);
const metodoPago = ref("efectivo");
const efectivoRecibido = ref<number | undefined>(undefined);
const clienteSeleccionado = ref("Cliente General");
const clienteID = ref(1);
const debounceTimer = ref<number | undefined>(undefined);
const isLoading = ref(false);

// [MODALS] - ESTADOS
const isCreateModalOpen = ref(false);
const isClienteModalOpen = ref(false);
const facturaParaRecibo = ref<backend.Factura | null>(null);

// [REF] - DOM
const searchInputRef = ref<{ $el: HTMLInputElement } | null>(null);
const searchResultsContainerRef = ref<HTMLElement | null>(null);
const searchResultItemsRef = ref<HTMLLIElement[]>([]);
const highlightedIndex = ref(-1);

// [SEARCH] - LÓGICA
watch(busqueda, (nuevoValor) => {
  highlightedIndex.value = -1;
  searchResultItemsRef.value = [];
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

function agregarAlCarrito(producto: backend.Producto) {
  if (!producto) return;
  cartStore.addToCart(producto);
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

// [KEYBOARD NAVIGATION & SCROLL]
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

watch(highlightedIndex, (newIndex) => {
  if (
    newIndex < 0 ||
    !searchResultsContainerRef.value ||
    !searchResultItemsRef.value[newIndex]
  )
    return;
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

// [COMPUTED & WATCHERS]
const cambio = computed(() => {
  if (
    metodoPago.value === "efectivo" &&
    efectivoRecibido.value &&
    efectivoRecibido.value > 0
  ) {
    const valor = efectivoRecibido.value - activeCartTotal.value;
    return valor >= 0 ? valor : 0;
  }
  return 0;
});

watch(metodoPago, (nuevoMetodo) => {
  if (nuevoMetodo !== "efectivo") {
    efectivoRecibido.value = undefined;
  }
});

// [LÓGICA DE VENTA]
async function finalizarVenta() {
  if (activeCart.value.length === 0) {
    toast.error("El carrito está vacío", {
      description: "Agrega productos antes de finalizar la venta.",
    });
    return;
  }
  const productoInvalido = activeCart.value.find(
    (item) => !item.PrecioVenta || item.PrecioVenta <= 0
  );
  if (productoInvalido) {
    toast.error("Precio de Venta Inválido", {
      description: `El producto "${productoInvalido.Nombre}" tiene un precio de venta no válido.`,
    });
    return;
  }
  if (!authenticatedUser.value?.id) {
    toast.error("Vendedor no identificado", {
      description: "Inicie sesión de nuevo.",
    });
    return;
  }

  const ventaRequest = new backend.VentaRequest({
    ClienteID: clienteID.value,
    VendedorID: authenticatedUser.value.id,
    MetodoPago: metodoPago.value,
    Productos: activeCart.value.map((item) => ({
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

    facturaParaRecibo.value = facturaCreada;

    cartStore.clearActiveCart();
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

// [INIT & CLIENTE]
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

// [MANEJO DE ATAJOS]
function handleKeyDown(event: KeyboardEvent) {
  if (event.key === "F12") {
    event.preventDefault();
    finalizarVenta();
  } else if (event.key === "F11") {
    event.preventDefault();
    cartStore.saveCurrentCart();
  } else if (event.key === "F10") {
    event.preventDefault();
    searchInputRef.value?.$el?.focus();
  }
}

// [LIFECYCLE]
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

const activeTab = ref("venta-actual");
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
  <ReciboVentaModal
    :factura="facturaParaRecibo"
    @update:open="facturaParaRecibo = null"
  />

  <div class="grid grid-cols-10 gap-6 h-[calc(100vh-8rem)]">
    <div class="col-span-10 lg:col-span-7 flex flex-col gap-6">
      <Card class="flex-1 overflow-hidden">
        <CardContent class="p-4 h-full flex flex-col">
          <Tabs v-model="activeTab" class="h-full flex flex-col">
            <TabsList class="w-full">
              <TabsTrigger value="venta-actual" class="flex-1"
                >Venta Actual</TabsTrigger
              >
              <TabsTrigger value="carritos-guardados" class="flex-1">
                Carritos en Espera
                <Badge v-if="savedCarts.length > 0" class="ml-2">{{
                  savedCarts.length
                }}</Badge>
              </TabsTrigger>
            </TabsList>

            <TabsContent
              value="venta-actual"
              class="flex-1 flex flex-col mt-4 overflow-hidden"
            >
              <div class="relative">
                <!-- Icono de búsqueda -->
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
                  class="pl-10 text-lg h-10"
                />

                <!-- Lista de resultados -->
                <div
                  v-if="productosEncontrados.length > 0"
                  ref="searchResultsContainerRef"
                  class="absolute z-10 w-full mt-2 border rounded-lg bg-card shadow-xl max-h-60 overflow-y-auto"
                >
                  <ul>
                    <li
                      v-for="(producto, index) in productosEncontrados"
                      :key="producto.id"
                      :ref="el => { if (el) searchResultItemsRef[index] = el as HTMLLIElement }"
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
                      <span class="font-mono text-lg">
                        ${{ producto.PrecioVenta.toLocaleString() }}
                      </span>
                    </li>
                  </ul>
                </div>

                <!-- Mensaje cuando no hay resultados -->
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
                      <TableHead class="text-right w-40"
                        >Precio Unit.</TableHead
                      >
                      <TableHead class="text-right w-40">Subtotal</TableHead>
                      <TableHead class="text-center w-20">Acción</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <template v-if="activeCart.length > 0">
                      <TableRow v-for="item in activeCart" :key="item.id">
                        <TableCell class="font-mono">{{
                          item.Codigo
                        }}</TableCell>
                        <TableCell class="font-medium">{{
                          item.Nombre
                        }}</TableCell>
                        <TableCell class="text-center">
                          <Input
                            type="number"
                            class="w-20 text-center mx-auto h-10"
                            :model-value="item.cantidad"
                            @update:model-value="
                              cartStore.updateQuantity(item.id, Number($event))
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
                            @click="cartStore.removeFromCart(item.id)"
                            ><Trash2 class="w-5 h-5 text-destructive"
                          /></Button>
                        </TableCell>
                      </TableRow>
                    </template>
                    <TableRow v-else>
                      <TableCell
                        colspan="6"
                        class="text-center h-24 text-muted-foreground"
                        >El carrito está vacío</TableCell
                      >
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </TabsContent>

            <TabsContent
              value="carritos-guardados"
              class="flex-1 overflow-y-auto mt-4"
            >
              <div v-if="savedCarts.length > 0" class="space-y-3">
                <Card v-for="cart in savedCarts" :key="cart.id">
                  <CardContent class="p-4 flex justify-between items-center">
                    <div class="flex-1">
                      <p class="font-semibold">{{ cart.nombre }}</p>
                      <p class="text-sm text-muted-foreground">
                        {{ cart.items.length }} productos por un total de
                        <span class="font-mono"
                          >${{ cart.total.toLocaleString() }}</span
                        >
                      </p>
                    </div>
                    <div class="flex gap-2 items-center">
                      <Button
                        class="py-1.5 px-3"
                        variant="outline"
                        size="sm"
                        @click="
                          cartStore.loadCart(cart.id);
                          activeTab = 'venta-actual';
                        "
                      >
                        <RotateCcw class="w-4 h-4" />
                        Cargar
                      </Button>
                      <Button
                        class="py-1.5 px-3"
                        variant="ghost"
                        size="icon"
                        @click="cartStore.deleteSavedCart(cart.id)"
                      >
                        <Trash2 class="w-4 h-4 text-destructive" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </div>
              <div
                v-else
                class="flex flex-col items-center justify-center h-full text-muted-foreground text-center"
              >
                <PackageOpen class="w-16 h-16 mb-4" />
                <h3 class="text-lg font-semibold">No hay carritos en espera</h3>
                <p class="text-sm">
                  Puedes guardar una venta en curso usando el botón (F11).
                </p>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>

    <div class="col-span-10 lg:col-span-3">
      <Card>
        <CardHeader><CardTitle>Gestión de Venta</CardTitle></CardHeader>
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
                ><UserSearch class="w-5 h-5"
              /></Button>
            </div>
          </div>
          <div class="space-y-2">
            <Label>Método de Pago</Label>
            <Select v-model="metodoPago">
              <SelectTrigger class="h-10"
                ><SelectValue placeholder="Seleccione un método"
              /></SelectTrigger>
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
                >${{ activeCartTotal.toLocaleString() }}</span
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
        <CardFooter class="flex flex-col gap-2">
          <Button
            @click="cartStore.saveCurrentCart()"
            variant="secondary"
            class="w-full h-10 text-lg"
          >
            <Save class="w-5 h-5 mr-2" />
            Guardar en Espera (F11)
          </Button>
          <Button @click="finalizarVenta" class="w-full h-10 text-lg">
            Finalizar Venta (F12)
          </Button>
        </CardFooter>
      </Card>
    </div>
  </div>
</template>
