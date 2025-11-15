<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/stores/auth";
import { useCartStore } from "@/stores/cart";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
import {
  ObtenerClientesPaginado,
  ObtenerProductosPaginado,
  RegistrarVenta,
} from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";

interface ObtenerProductosPaginadoResponse {
  Records: backend.Producto[];
  TotalRecords: number;
}
interface ObtenerClientePaginadoResponse {
  Records: backend.Cliente[];
  TotalRecords: number;
}
const authStore = useAuthStore();
const { user: authenticatedUser } = storeToRefs(authStore);
const cartStore = useCartStore();
const { activeCart, savedCarts, activeCartTotal } = storeToRefs(cartStore);
const busqueda = ref("");
const productosEncontrados = ref<backend.Producto[]>([]);
const metodoPago = ref("efectivo");
const efectivoRecibido = ref<number | undefined>(undefined);
const clienteSeleccionado = ref("Cliente General");
const clienteUUID = ref<string>("SYSTEM-ADMIN");
const debounceTimer = ref<number | undefined>(undefined);
const isLoading = ref(false);
const isCreateModalOpen = ref(false);
const isClienteModalOpen = ref(false);
const facturaParaRecibo = ref<backend.Factura | null>(new backend.Factura());
const searchInputRef = ref<{ $el: HTMLInputElement } | null>(null);
const searchResultsContainerRef = ref<HTMLElement | null>(null);
const searchResultItemsRef = ref<HTMLLIElement[]>([]);
const highlightedIndex = ref(-1);

watch(busqueda, (nuevoValor) => {
  highlightedIndex.value = -1;
  searchResultItemsRef.value = [];
  clearTimeout(debounceTimer.value);
  const trimmedValue = nuevoValor.trim();
  if (trimmedValue.length < 2) {
    productosEncontrados.value = [];
    return;
  }
  isLoading.value = true;
  debounceTimer.value = setTimeout(async () => {
    try {
      const response: ObtenerProductosPaginadoResponse =
        await ObtenerProductosPaginado(1, 10, busqueda.value, "", "asc");
      if (
        response.Records.length === 1 &&
        response.Records[0]!.Codigo.toLowerCase() === trimmedValue.toLowerCase()
      ) {
        agregarAlCarrito(response.Records[0]!);
        productosEncontrados.value = [];
      } else {
        productosEncontrados.value = response.Records;
        if (response.Records.length > 0) {
          highlightedIndex.value = 0;
        }
      }
    } catch (error) {
      toast.error("Error de búsqueda", {
        description: "No se pudieron obtener los productos.",
      });
    } finally {
      isLoading.value = false;
    }
  }, 150);
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
  if (highlightedIndex.value >= 0 && productosEncontrados.value[highlightedIndex.value]) {
    agregarAlCarrito(productosEncontrados.value[highlightedIndex.value]!);
  } else if (productosEncontrados.value.length === 1) {
    agregarAlCarrito(productosEncontrados.value[0]!);
  }
}

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
  if (nuevoMetodo !== "efectivo") efectivoRecibido.value = undefined;
});

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
  if (!authenticatedUser.value?.UUID) {
    toast.error("Vendedor no identificado", {
      description: "Inicie sesión de nuevo.",
    });
    return;
  }
  const ventaRequest = new backend.VentaRequest({
    ClienteUUID: clienteUUID.value,
    VendedorUUID: authenticatedUser.value.UUID,
    MetodoPago: metodoPago.value,
    Productos: activeCart.value.map((item) => ({
      ProductoUUID: item.UUID,
      Cantidad: item.cantidad,
      PrecioUnitario: item.PrecioVenta,
    })),
  });
  try {
    const facturaCreada = await RegistrarVenta(ventaRequest);
    toast.success("¡Venta registrada con éxito!", {
      description: `Factura N° ${facturaCreada.NumeroFactura
        } por un total de $${facturaCreada.Total.toLocaleString()}`,
    });
    facturaParaRecibo.value = facturaCreada;
    cartStore.clearActiveCart();
    efectivoRecibido.value = undefined;
    busqueda.value = "";
    await cargarClienteGeneralPorDefecto();
  } catch (error) {
    toast.error("Error al registrar la venta", {
      description: `Hubo un problema: ${error}`,
    });
  }
}

async function cargarClienteGeneralPorDefecto() {
  try {
    const response: ObtenerClientePaginadoResponse =
      await ObtenerClientesPaginado(1, 1, "222222222", "", "asc");
    if (response && response.Records && response.Records.length > 0) {
      const clienteGeneral = response.Records[0] as backend.Cliente;
      clienteUUID.value = clienteGeneral.UUID;
      clienteSeleccionado.value = `${clienteGeneral.Nombre} ${clienteGeneral.Apellido}`;
    } else {
      clienteUUID.value = "SYSTEM-ADMIN";
      clienteSeleccionado.value = "Cliente General";
    }
  } catch (error) {
    clienteUUID.value = "SYSTEM-ADMIN";
    clienteSeleccionado.value = "Cliente General";
  }
}

function handleClienteSeleccionado(cliente: backend.Cliente) {
  clienteUUID.value = cliente.UUID;
  clienteSeleccionado.value = `${cliente.Nombre} ${cliente.Apellido}`;
  isClienteModalOpen.value = false;
}

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

function handleLoadCart(cartId: number) {
  if (activeCart.value.length > 0) {
    cartStore.saveCurrentCart();
    toast.info("Carrito actual guardado en espera", {
      description:
        "Se ha generado un nuevo pendiente con los productos que tenías.",
    });
  }
  cartStore.loadCart(cartId);
  activeTab.value = "venta-actual";

  nextTick(() => {
    searchInputRef.value?.$el?.focus();
  });
}
</script>

<template>
  <CrearProductoModal v-model:open="isCreateModalOpen" :initial-codigo="busqueda"
    @product-created="handleProductCreated" />
  <BuscarClienteModal v-model:open="isClienteModalOpen" @cliente-seleccionado="handleClienteSeleccionado" />
  <ReciboVentaModal :factura="facturaParaRecibo" @update:open="facturaParaRecibo = null" />

  <div class="flex flex-col gap-4 h-[calc(100vh-2rem)]">
    <div class="flex-1 min-h-0">
      <Card class="h-full flex flex-col py-1 px-2">
        <CardContent class="p-2 h-full flex flex-col">
          <Tabs v-model="activeTab" class="h-full flex flex-col">
            <TabsList class="w-full flex-shrink-0">
              <TabsTrigger value="venta-actual" class="flex-1">
                Venta Actual
              </TabsTrigger>
              <TabsTrigger value="carritos-guardados" class="flex-1">
                Carritos en Espera
                <Badge v-if="savedCarts.length > 0" class="ml-2">{{
                  savedCarts.length
                }}</Badge>
              </TabsTrigger>
            </TabsList>

            <TabsContent value="venta-actual" class="flex-1 flex flex-col gap-4 mt-4 overflow-hidden">
              <div class="flex-shrink-0 flex flex-col gap-4">
                <div class="flex flex-wrap items-start gap-4">
                  <div class="flex-1 min-w-[250px] flex gap-2">
                    <Input id="cliente" :value="clienteSeleccionado" readonly class="h-10" />
                    <Button @click="isClienteModalOpen = true" variant="outline" size="icon"
                      class="h-10 w-10 flex-shrink-0">
                      <UserSearch class="w-5 h-5" />
                    </Button>
                  </div>
                  <div class="flex-1 min-w-[300px] flex items-center gap-2">
                    <Select v-model="metodoPago">
                      <SelectTrigger class="h-10">
                        <SelectValue placeholder="Método de Pago" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="efectivo">Efectivo</SelectItem>
                        <SelectItem value="transferencia">Transferencia</SelectItem>
                      </SelectContent>
                    </Select>
                    <Input v-if="metodoPago === 'efectivo'" id="efectivo" type="number" v-model="efectivoRecibido"
                      placeholder="Efectivo Recibido" class="text-right h-10 text-lg font-mono" />
                  </div>
                </div>

                <div class="relative">
                  <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                  <Input ref="searchInputRef" v-model="busqueda" placeholder="Buscar por nombre o código... (F10)"
                    @keyup.enter="manejarBusquedaConEnter" @keydown.down.prevent="moverSeleccion('abajo')"
                    @keydown.up.prevent="moverSeleccion('arriba')" class="pl-10 text-lg h-12" />
                  <div v-if="productosEncontrados.length > 0" ref="searchResultsContainerRef"
                    class="absolute z-10 w-full mt-2 border rounded-lg bg-card shadow-xl max-h-60 overflow-y-auto">
                    <ul>
                      <li v-for="(producto, index) in productosEncontrados" :key="producto.UUID"
                        :ref="el => { if (el) searchResultItemsRef[index] = el as HTMLLIElement }"
                        class="p-3 hover:bg-muted cursor-pointer flex justify-between items-center" :class="{
                          'bg-primary text-primary-foreground hover:bg-primary':
                            index === highlightedIndex,
                        }" @click="agregarAlCarrito(producto)">
                        <div>
                          <p class="font-semibold">{{ producto.Nombre }}</p>
                          <p class="text-sm" :class="index === highlightedIndex
                            ? 'text-primary-foreground/80'
                            : 'text-muted-foreground'
                            ">
                            Código: {{ producto.Codigo }} | Stock:
                            {{ producto.Stock }}
                          </p>
                        </div>
                        <span class="font-mono text-lg">${{ producto.PrecioVenta.toLocaleString() }}</span>
                      </li>
                    </ul>
                  </div>
                  <div v-else-if="
                    busqueda.length >= 2 &&
                    !isLoading &&
                    productosEncontrados.length === 0
                  "
                    class="absolute z-10 w-full mt-2 border rounded-lg bg-card shadow-xl p-4 text-center text-muted-foreground flex flex-col items-center gap-3">
                    <p>No se encontraron productos para "{{ busqueda }}"</p>
                    <Button @click="isCreateModalOpen = true" variant="outline">
                      <PlusCircle class="w-4 h-4 mr-2" />Crear Producto
                    </Button>
                  </div>
                </div>
              </div>

              <div class="flex-1 overflow-y-auto -mx-4 px-4">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead class="w-[120px]">Código</TableHead>
                      <TableHead>Producto</TableHead>
                      <TableHead class="w-[130px] text-center">Cantidad</TableHead>
                      <TableHead class="w-[170px] text-right">Precio Unit.</TableHead>
                      <TableHead class="w-[170px] text-right">Subtotal</TableHead>
                      <TableHead class="w-[80px] text-center">Acción</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <template v-if="activeCart.length > 0">
                      <TableRow v-for="item in activeCart" :key="item.UUID">
                        <TableCell class="font-mono">{{
                          item.Codigo
                        }}</TableCell>
                        <TableCell class="font-medium truncate whitespace-nowrap overflow-hidden">
                          {{ item.Nombre }}
                        </TableCell>
                        <TableCell class="text-center">
                          <Input type="number" class="w-20 text-center mx-auto h-10" :model-value="item.cantidad"
                            @update:model-value="
                              cartStore.updateQuantity(
                                item.UUID,
                                Number($event)
                              )
                              " min="1" :max="item.Stock" />
                        </TableCell>
                        <TableCell>
                          <Input type="number" class="w-full text-right h-10 font-mono" v-model="item.PrecioVenta"
                            step="0.01" />
                        </TableCell>
                        <TableCell class="text-right font-mono">
                          ${{
                            (item.PrecioVenta * item.cantidad).toLocaleString()
                          }}
                        </TableCell>
                        <TableCell class="text-center">
                          <Button size="icon" variant="ghost" @click="cartStore.removeFromCart(item.UUID)">
                            <Trash2 class="w-5 h-5 text-destructive" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    </template>
                    <TableRow v-else>
                      <TableCell colspan="6" class="text-center h-24 text-muted-foreground">
                        El carrito está vacío
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
              <div class="flex-shrink-0 border-t pt-4 space-y-4">
                <div class="flex justify-between items-center text-xl">
                  <span class="text-muted-foreground">Total</span>
                  <span class="font-bold font-mono text-3xl">${{ activeCartTotal.toLocaleString() }}</span>
                </div>
                <div v-if="
                  metodoPago === 'efectivo' && efectivoRecibido && cambio > 0
                " class="flex justify-between items-center text-xl">
                  <span class="text-muted-foreground">Cambio</span>
                  <span class="font-bold font-mono text-green-400 text-3xl">${{ cambio.toLocaleString() }}</span>
                </div>
                <div class="flex gap-2">
                  <Button @click="cartStore.saveCurrentCart()" variant="secondary" class="flex-1 h-12 text-base">
                    <Save class="w-5 h-5 mr-2" />Espera (F11)
                  </Button>
                  <Button @click="finalizarVenta" class="flex-1 h-12 text-base">
                    Finalizar (F12)
                  </Button>
                </div>
              </div>
            </TabsContent>
            <TabsContent value="carritos-guardados" class="flex-1 overflow-y-auto mt-4">
              <div v-if="savedCarts.length > 0" class="space-y-3">
                <Card v-for="cart in savedCarts" :key="cart.id">
                  <CardContent class="p-4 flex justify-between items-center">
                    <div class="flex-1">
                      <p class="font-semibold">{{ cart.nombre }}</p>
                      <p class="text-sm text-muted-foreground">
                        {{ cart.items.length }} productos por un total de
                        <span class="font-mono">${{ cart.total.toLocaleString() }}</span>
                      </p>
                    </div>
                    <div class="flex gap-2 items-center">
                      <Button class="py-1.5 px-3" variant="outline" size="sm" @click="handleLoadCart(cart.id)">
                        <RotateCcw class="w-4 h-4 mr-2" />Cargar
                      </Button>
                      <Button class="py-1.5 px-3" variant="ghost" size="icon"
                        @click="cartStore.deleteSavedCart(cart.id)">
                        <Trash2 class="w-4 h-4 text-destructive" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </div>
              <div v-else class="flex flex-col items-center justify-center h-full text-muted-foreground text-center">
                <PackageOpen class="w-16 h-16 mb-4" />
                <h3 class="text-lg font-semibold">No hay carritos en espera</h3>
                <p class="text-sm">
                  Puedes guardar una venta en curso usando (F11).
                </p>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
