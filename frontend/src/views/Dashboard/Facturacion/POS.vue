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
import FinalizarVentaModal from "@/components/modals/FinalizarVentaModal.vue";
import {
  ObtenerClientesPaginado,
  ObtenerProductosPaginado,
  RegistrarVenta,
} from "@/../bridge/go/backend/Db";
import { backend } from "@/../bridge/go/models";

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
const clienteUUID = ref<string>("");
const debounceTimer = ref<number | undefined>(undefined);
const isLoading = ref(false);
const pendingAutoAdd = ref(false);
const isCreateModalOpen = ref(false);
const isClienteModalOpen = ref(false);
const isFinalizarModalOpen = ref(false);
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
  if (trimmedValue.length < 1) {
    productosEncontrados.value = [];
    pendingAutoAdd.value = false;
    return;
  }
  isLoading.value = true;
  debounceTimer.value = setTimeout(async () => {
    try {
      const response: ObtenerProductosPaginadoResponse =
        await ObtenerProductosPaginado(1, 10, trimmedValue, "", "asc");
      productosEncontrados.value = response.Records || [];
      if (productosEncontrados.value.length > 0) {
        highlightedIndex.value = 0;
      }
      // Barcode auto-add: if Enter was pressed while loading and result is unique
      if (pendingAutoAdd.value && productosEncontrados.value.length === 1) {
        pendingAutoAdd.value = false;
        agregarAlCarrito(productosEncontrados.value[0]!);
        return;
      }
      pendingAutoAdd.value = false;
    } catch (error) {
      pendingAutoAdd.value = false;
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
  } else if (isLoading.value) {
    // Barcode scan: Enter came before the debounce resolved — flag for auto-add
    pendingAutoAdd.value = true;
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

function abrirModalFinalizar() {
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
  if (!clienteUUID.value) {
    toast.error("Cliente no seleccionado", {
      description: "Selecciona un cliente o espera a que cargue el cliente general.",
    });
    return;
  }
  isFinalizarModalOpen.value = true;
}

async function confirmarVenta(pago: { metodoPago: string; efectivoRecibido?: number }) {
  const ventaRequest = new backend.VentaRequest({
    ClienteUUID: clienteUUID.value,
    VendedorUUID: authenticatedUser.value!.UUID,
    MetodoPago: pago.metodoPago,
    Productos: activeCart.value.map((item) => ({
      ProductoUUID: item.UUID,
      Cantidad: item.cantidad,
      PrecioUnitario: item.PrecioVenta,
    })),
  });

  try {
    const facturaCreada = await RegistrarVenta(ventaRequest);
    toast.success("¡Venta registrada con éxito!", {
      description: `Factura N° ${facturaCreada.NumeroFactura} por un total de $${facturaCreada.Total.toLocaleString()}`,
    });
    facturaParaRecibo.value = facturaCreada;
    cartStore.clearActiveCart();
    efectivoRecibido.value = undefined;
    busqueda.value = "";
    isFinalizarModalOpen.value = false;
    await cargarClienteGeneralPorDefecto();
    nextTick(() => {
      searchInputRef.value?.$el?.focus();
    });
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
      clienteUUID.value = "";
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
    abrirModalFinalizar();
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
  <FinalizarVentaModal 
    v-model:open="isFinalizarModalOpen" 
    :total="activeCartTotal" 
    :cliente="clienteSeleccionado"
    @confirm="confirmarVenta"
  />

  <div class="flex flex-col h-[calc(100vh-3.5rem)]">

    <!-- ── TOP HEADER BAR (tabs + customer) ── -->
    <div class="shrink-0 flex items-center justify-between gap-4 px-6 py-3 border-b bg-card">
      <Tabs v-model="activeTab">
        <TabsList>
          <TabsTrigger value="venta-actual">Venta Actual</TabsTrigger>
          <TabsTrigger value="carritos-guardados">
            Carritos en Espera
            <Badge v-if="savedCarts.length > 0" variant="secondary" class="ml-2">{{ savedCarts.length }}</Badge>
          </TabsTrigger>
        </TabsList>
      </Tabs>

      <div v-show="activeTab === 'venta-actual'" class="flex items-center gap-3">
        <div class="flex items-center gap-2">
          <div class="flex flex-col items-end">
            <span class="text-[10px] uppercase font-bold text-muted-foreground leading-none">Cliente</span>
            <span class="text-sm font-semibold">{{ clienteSeleccionado }}</span>
          </div>
          <Button @click="isClienteModalOpen = true" variant="outline" size="sm" class="h-9 px-3 gap-2">
            <UserSearch class="w-4 h-4" />
            Cambiar
          </Button>
        </div>
      </div>
    </div>

    <!-- ── VENTA ACTUAL (v-show, NOT TabsContent — avoids overflow clipping) ── -->
    <div v-show="activeTab === 'venta-actual'" class="flex-1 min-h-0 flex flex-col gap-4 px-6 py-4">

      <!-- Search bar — parent is relative, no overflow anywhere above -->
      <div class="relative shrink-0">
        <Search class="absolute left-3.5 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground pointer-events-none" />
        <Input
          ref="searchInputRef"
          v-model="busqueda"
          placeholder="Buscar producto por nombre o código... (F10)"
          @keyup.enter="manejarBusquedaConEnter"
          @keydown.down.prevent="moverSeleccion('abajo')"
          @keydown.up.prevent="moverSeleccion('arriba')"
          class="pl-11 h-12 text-base"
        />

        <!-- Results dropdown: z-[200] ensures it's above cart table, sticky header, etc. -->
        <div
          v-if="productosEncontrados.length > 0"
          ref="searchResultsContainerRef"
          class="absolute left-0 right-0 top-[calc(100%+6px)] z-[200] rounded-lg border bg-background shadow-md max-h-72 overflow-y-auto"
        >
          <div
            v-for="(producto, index) in productosEncontrados"
            :key="producto.UUID"
            :ref="el => { if (el) searchResultItemsRef[index] = el as HTMLLIElement }"
            class="flex items-center justify-between px-4 py-3 cursor-pointer border-b last:border-0 transition-colors"
            :class="index === highlightedIndex
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-muted/60'"
            @click="agregarAlCarrito(producto)"
          >
            <div class="min-w-0">
              <p class="text-sm font-semibold">{{ producto.Nombre }}</p>
              <p class="text-xs font-mono mt-0.5"
                :class="index === highlightedIndex ? 'text-primary-foreground/70' : 'text-muted-foreground'">
                {{ producto.Codigo }} &middot; Stock: {{ producto.Stock }}
              </p>
            </div>
            <span class="ml-4 shrink-0 font-mono font-bold text-sm">
              ${{ producto.PrecioVenta.toLocaleString() }}
            </span>
          </div>
        </div>

        <!-- No results -->
        <div
          v-else-if="busqueda.length >= 1 && !isLoading && productosEncontrados.length === 0"
          class="absolute left-0 right-0 top-[calc(100%+6px)] z-[200] rounded-lg border bg-background shadow-md p-5 flex flex-col items-center gap-3"
        >
          <p class="text-sm text-muted-foreground">Sin resultados para "{{ busqueda }}"</p>
          <Button @click="isCreateModalOpen = true" variant="outline" size="sm">
            <PlusCircle class="w-4 h-4 mr-2" />Crear Producto
          </Button>
        </div>
      </div>

      <!-- Cart table — inner scroll, sticky header without backdrop-blur (no stacking context) -->
      <div class="flex-1 min-h-0 rounded-lg border bg-card overflow-hidden">
        <div class="h-full overflow-y-auto">
          <Table>
            <TableHeader class="sticky top-0 bg-card border-b z-10">
              <TableRow>
                <TableHead class="h-10 text-xs font-semibold w-28">Código</TableHead>
                <TableHead class="h-10 text-xs font-semibold">Producto</TableHead>
                <TableHead class="h-10 text-xs font-semibold w-28 text-center">Cant.</TableHead>
                <TableHead class="h-10 text-xs font-semibold w-36 text-right">Precio Unit.</TableHead>
                <TableHead class="h-10 text-xs font-semibold w-36 text-right">Subtotal</TableHead>
                <TableHead class="h-10 w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              <template v-if="activeCart.length > 0">
                <TableRow v-for="item in activeCart" :key="item.UUID" class="hover:bg-muted/30 transition-colors">
                  <TableCell class="py-2 font-mono text-xs text-muted-foreground">{{ item.Codigo }}</TableCell>
                  <TableCell class="py-2 font-medium text-sm">{{ item.Nombre }}</TableCell>
                  <TableCell class="py-2 text-center">
                    <Input
                      type="number"
                      class="w-20 h-8 text-center mx-auto"
                      :model-value="item.cantidad"
                      @update:model-value="cartStore.updateQuantity(item.UUID, Number($event))"
                      min="1"
                      :max="item.Stock"
                    />
                  </TableCell>
                  <TableCell class="py-2">
                    <Input
                      type="number"
                      class="w-32 h-8 text-right ml-auto font-mono"
                      v-model="item.PrecioVenta"
                      step="0.01"
                    />
                  </TableCell>
                  <TableCell class="py-2 text-right font-mono font-semibold text-sm">
                    ${{ (item.PrecioVenta * item.cantidad).toLocaleString() }}
                  </TableCell>
                  <TableCell class="py-2 text-center">
                    <Button size="icon" variant="ghost" class="h-8 w-8" @click="cartStore.removeFromCart(item.UUID)">
                      <Trash2 class="w-4 h-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              </template>
              <TableRow v-else>
                <TableCell colspan="6" class="h-48 text-center text-muted-foreground text-sm">
                  El carrito está vacío — busca un producto arriba o presiona F10
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </div>

      <!-- Footer: summary + actions -->
      <div class="shrink-0 rounded-lg border bg-card px-5 py-3 flex items-center gap-6">
        <div class="flex-1 text-sm text-muted-foreground">
          {{ activeCart.length }} artículo(s)
        </div>
        <div class="flex items-center gap-2">
          <span class="text-sm text-muted-foreground uppercase tracking-wider font-medium">Total</span>
          <span class="font-mono font-bold text-3xl text-primary">${{ activeCartTotal.toLocaleString() }}</span>
        </div>
        <div class="flex gap-2">
          <Button @click="cartStore.saveCurrentCart()" variant="outline" class="h-12 px-4 gap-2">
            <Save class="w-4 h-4" />En Espera
            <kbd class="ml-0.5 text-[10px] font-mono bg-muted border rounded px-1 text-muted-foreground">F11</kbd>
          </Button>
          <Button @click="abrirModalFinalizar" class="h-12 px-8 gap-2 font-bold text-lg">
            Pagar
            <kbd class="ml-0.5 text-[10px] font-mono rounded px-1 bg-primary/20 border border-primary/30">F12</kbd>
          </Button>
        </div>
      </div>
    </div>

    <!-- ── CARRITOS EN ESPERA ── -->
    <div v-show="activeTab === 'carritos-guardados'" class="flex-1 min-h-0 overflow-y-auto px-6 py-4">
      <div v-if="savedCarts.length > 0" class="space-y-2 max-w-2xl">
        <div
          v-for="cart in savedCarts"
          :key="cart.id"
          class="rounded-lg border bg-card px-4 py-3.5 flex items-center justify-between gap-4 hover:bg-muted/20 transition-colors"
        >
          <div class="min-w-0">
            <p class="font-semibold text-sm">{{ cart.nombre }}</p>
            <p class="text-xs text-muted-foreground mt-0.5">
              {{ cart.items.length }} producto(s) &middot;
              <span class="font-mono font-medium">${{ cart.total.toLocaleString() }}</span>
            </p>
          </div>
          <div class="flex gap-2 shrink-0">
            <Button variant="outline" size="sm" class="gap-1.5" @click="handleLoadCart(cart.id)">
              <RotateCcw class="w-3.5 h-3.5" />Cargar
            </Button>
            <Button variant="ghost" size="icon" class="h-8 w-8" @click="cartStore.deleteSavedCart(cart.id)">
              <Trash2 class="w-4 h-4 text-destructive" />
            </Button>
          </div>
        </div>
      </div>
      <div v-else class="h-full flex flex-col items-center justify-center text-muted-foreground gap-3">
        <PackageOpen class="w-12 h-12 opacity-30" />
        <div class="text-center">
          <p class="font-medium">No hay carritos en espera</p>
          <p class="text-sm mt-1 opacity-60">Guarda la venta actual con F11.</p>
        </div>
      </div>
    </div>

  </div>
</template>
