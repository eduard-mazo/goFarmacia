<script setup lang="ts">
import type {
  ColumnDef,
  PaginationState,
  SortingState,
} from "@tanstack/vue-table";
import {
  FlexRender,
  getCoreRowModel,
  getSortedRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { h, ref, watch, onMounted, computed } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { ArrowUpDown, ArrowUp, ArrowDown, SlidersHorizontal, Search, Package } from "lucide-vue-next";
import { backend } from "@/../bridge/go/models";
import { ObtenerProductosPaginado } from "@/../bridge/go/backend/Db";
import { toast } from "vue-sonner";
import AjustarStockModal from "@/components/modals/AjustarStockModal.vue";

interface ObtenerProductosPaginadoResponse {
  Records: backend.Producto[];
  TotalRecords: number;
}

const listaProductos = ref<backend.Producto[]>([]);
const totalProductos = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);
const isAdjustModalOpen = ref(false);
const productoSeleccionado = ref<backend.Producto | null>(null);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

const cargarProductos = async () => {
  try {
    const currentPage = pagination.value.pageIndex + 1;
    let sortBy = "";
    let sortOrder = "asc";
    if (sorting.value.length > 0) {
      sortBy = sorting.value[0]!.id;
      sortOrder = sorting.value[0]!.desc ? "desc" : "asc";
    }
    const response: ObtenerProductosPaginadoResponse =
      await ObtenerProductosPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value,
        sortBy,
        sortOrder
      );
    listaProductos.value = response.Records || [];
    totalProductos.value = response.TotalRecords || 0;
  } catch (error) {
    toast.error("Error al cargar productos", { description: `${error}` });
  }
};

function sortHeader(label: string) {
  return ({ column }: { column: any }) =>
    h("button", {
      class: "flex items-center gap-1 group/sort text-[10px] font-medium text-muted-foreground hover:text-foreground transition-colors select-none",
      onClick: () => column.toggleSorting(),
    }, [
      label,
      h(column.getIsSorted() === "asc" ? ArrowUp
        : column.getIsSorted() === "desc" ? ArrowDown
        : ArrowUpDown, {
        class: `h-3 w-3 ${column.getIsSorted() ? "text-primary" : "opacity-30 group-hover/sort:opacity-60"}`,
      }),
    ]);
}

const columns: ColumnDef<backend.Producto>[] = [
  {
    accessorKey: "Codigo",
    header: sortHeader("Código"),
    cell: ({ row }) => h("div", { class: "font-mono text-muted-foreground" }, row.getValue("Codigo")),
  },
  {
    accessorKey: "Nombre",
    header: sortHeader("Nombre"),
    cell: ({ row }) =>
      h("div", { class: "uppercase max-w-[600px] truncate font-medium" }, row.getValue("Nombre")),
  },
  {
    accessorKey: "PrecioVenta",
    header: sortHeader("Precio Venta"),
    cell: ({ row }) =>
      h("div", { class: "font-medium tabular-nums" },
        new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 0 })
          .format(parseFloat(row.getValue("PrecioVenta")))
      ),
  },
  {
    accessorKey: "Stock",
    header: sortHeader("Stock"),
    cell: ({ row }) => {
      const stock = row.getValue("Stock") as number;
      return h("div", {
        class: `inline-flex items-center justify-center min-w-[2rem] px-1.5 py-0.5 rounded text-[10px] font-bold tabular-nums ${
          stock === 0 ? "bg-red-100 text-red-700"
          : stock <= 10 ? "bg-amber-100 text-amber-700"
          : "bg-emerald-50 text-emerald-700"
        }`,
      }, String(stock));
    },
  },
  {
    id: "actions",
    cell: ({ row }) => h(
      Button,
      {
        variant: "ghost",
        class: "h-7 w-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity",
        title: "Ajustar stock",
        onClick: () => handleOpenAdjustModal(row.original),
      },
      () => h(SlidersHorizontal, { class: "w-3.5 h-3.5" })
    )
  },
];

const table = useVueTable({
  get data() {
    return listaProductos.value;
  },
  columns,
  manualPagination: true,
  manualSorting: true,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  get pageCount() {
    return Math.ceil(totalProductos.value / pagination.value.pageSize);
  },
  state: {
    get sorting() {
      return sorting.value;
    },
    get pagination() {
      return pagination.value;
    },
  },
  onPaginationChange: (updater) => valueUpdater(updater, pagination),
  onSortingChange: (updater) => valueUpdater(updater, sorting),
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => table.setPageIndex(newPage - 1),
});

function handleOpenAdjustModal(producto: backend.Producto) {
  productoSeleccionado.value = producto;
  isAdjustModalOpen.value = true;
}
async function handleStockUpdated() {
  await cargarProductos();
  productoSeleccionado.value = null;
}

onMounted(cargarProductos);
watch(pagination, cargarProductos, { deep: true });
watch(
  sorting,
  () => {
    pagination.value.pageIndex = 0;
    cargarProductos();
  },
  { deep: true }
);
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    pagination.value.pageIndex = 0;
    cargarProductos();
  }, 150);
});
</script>

<template>
  <AjustarStockModal v-if="productoSeleccionado" v-model:open="isAdjustModalOpen" :producto="productoSeleccionado"
    @stock-updated="handleStockUpdated" />
  <div class="flex flex-col h-full">
    <!-- Header bar -->
    <div class="shrink-0 border-b px-4 py-3 flex items-center justify-between bg-background">
      <div>
        <p class="text-sm font-semibold">Control de Stock</p>
        <p class="text-xs text-muted-foreground mt-0.5">
          Supervisa y ajusta los niveles de inventario —
          Stock bajo <span class="text-yellow-600 font-medium">≤ 10</span> · Agotado <span class="text-red-600 font-medium">= 0</span>
        </p>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-56" placeholder="Buscar por nombre o código..." />
      </div>
      <template v-if="busqueda">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <Search class="h-2.5 w-2.5" />
          {{ totalProductos }} resultado{{ totalProductos !== 1 ? 's' : '' }} en todo el inventario
        </span>
      </template>
      <template v-if="sorting.length">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <component :is="sorting[0]?.desc ? ArrowDown : ArrowUp" class="h-2.5 w-2.5" />
          Ordenado por {{ sorting[0]?.id }} (todo el inventario)
        </span>
      </template>
    </div>

    <!-- Table area -->
    <div class="flex-1 overflow-auto">
      <table data-supabase-table class="w-full text-xs border-collapse">
        <thead class="sticky top-0 z-10">
          <tr class="bg-muted border-b">
            <th class="h-9 w-10 px-3 text-[10px] font-normal text-muted-foreground/40 text-center border-r">#</th>
            <th
              v-for="header in table.getHeaderGroups()[0]?.headers"
              :key="header.id"
              v-show="header.column.getIsVisible()"
              class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-left border-r last:border-r-0 whitespace-nowrap">
              <FlexRender v-if="!header.isPlaceholder" :render="header.column.columnDef.header" :props="header.getContext()" />
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-if="table.getRowModel().rows?.length">
            <tr
              v-for="(row, rowIndex) in table.getRowModel().rows"
              :key="row.id"
              class="border-b hover:bg-muted/40 transition-colors group">
              <td class="h-9 px-3 text-center text-[10px] font-mono text-muted-foreground/40 border-r w-10 select-none">
                {{ pagination.pageIndex * pagination.pageSize + rowIndex + 1 }}
              </td>
              <td
                v-for="cell in row.getVisibleCells()"
                :key="cell.id"
                class="h-9 px-3 border-r last:border-r-0">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </td>
            </tr>
          </template>
          <tr v-else>
            <td :colspan="columns.length + 1" class="h-40 text-center">
              <div class="flex flex-col items-center justify-center gap-2 text-muted-foreground">
                <Package class="h-8 w-8 opacity-20" />
                <span class="text-xs">No se encontraron productos</span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer -->
    <div class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalProductos }} producto(s) en total</span>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5">
          <span class="text-xs text-muted-foreground">Filas</span>
          <Select
            :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(value) => table.setPageSize(Number(value))">
            <SelectTrigger class="h-7 w-16 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="size in [10, 25, 50]" :key="size" :value="`${size}`" class="text-xs">{{ size }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination
          v-if="pageCount > 1"
          v-model:page="currentPage"
          :total="totalProductos"
          :items-per-page="pagination.pageSize"
          :sibling-count="1"
          show-edges>
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious class="h-7 w-7" />
            <template v-for="(item, index) in items">
              <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                <Button class="w-7 h-7 p-0 text-xs" :variant="item.value === currentPage ? 'default' : 'outline'">
                  {{ item.value }}
                </Button>
              </PaginationItem>
              <PaginationEllipsis v-else :key="item.type" :index="index" />
            </template>
            <PaginationNext class="h-7 w-7" />
          </PaginationContent>
        </Pagination>
      </div>
    </div>
  </div>
</template>
