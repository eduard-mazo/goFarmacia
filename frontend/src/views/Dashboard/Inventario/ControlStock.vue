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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { ArrowUpDown, Edit } from "lucide-vue-next";
import { backend } from "@/../wailsjs/go/models";
import { ObtenerProductosPaginado } from "@/../wailsjs/go/backend/Db";
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
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 15 });

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

const columns: ColumnDef<backend.Producto>[] = [
  { accessorKey: "Codigo", header: "Código" },
  { accessorKey: "Nombre", header: "Nombre" },
  {
    accessorKey: "PrecioVenta",
    header: "Precio Venta",
    cell: ({ row }) =>
      h(
        "div",
        { class: "text-right font-medium" },
        new Intl.NumberFormat("es-CO", {
          style: "currency",
          currency: "COP",
        }).format(parseFloat(row.getValue("PrecioVenta")))
      ),
  },
  {
    accessorKey: "Stock",
    header: ({ column }) =>
      h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Stock", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
    cell: ({ row }) => {
      const stock = row.getValue("Stock") as number;
      let stockClass = "";
      if (stock <= 5) stockClass = "text-red-500 font-bold";
      else if (stock <= 10) stockClass = "text-yellow-500 font-bold";
      return h("div", { class: `text-center ${stockClass}` }, stock);
    },
  },
  {
    id: "actions",
    cell: ({ row }) =>
      h(
        Button,
        {
          variant: "outline",
          size: "sm",
          onClick: () => handleOpenAdjustModal(row.original),
        },
        [h(Edit, { class: "w-4 h-4 mr-2" }), "Ajustar"]
      ),
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
  <AjustarStockModal
    v-if="productoSeleccionado"
    v-model:open="isAdjustModalOpen"
    :producto="productoSeleccionado"
    @stock-updated="handleStockUpdated"
  />
  <div class="w-full">
    <h1 class="text-2xl font-semibold mb-4">Control de Stock</h1>
    <div class="flex items-center py-4">
      <Input
        class="max-w-sm h-10"
        placeholder="Buscar por nombre o código..."
        v-model="busqueda"
      />
    </div>
    <div class="rounded-md border">
      <Table>
        <TableHeader
          ><TableRow
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
            ><TableHead v-for="header in headerGroup.headers" :key="header.id"
              ><FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()" /></TableHead></TableRow
        ></TableHeader>
        <TableBody>
          <template v-if="table.getRowModel().rows?.length"
            ><TableRow v-for="row in table.getRowModel().rows" :key="row.id"
              ><TableCell v-for="cell in row.getVisibleCells()" :key="cell.id"
                ><FlexRender
                  :render="cell.column.columnDef.cell"
                  :props="cell.getContext()" /></TableCell></TableRow
          ></template>
          <TableRow v-else
            ><TableCell :colspan="columns.length" class="h-24 text-center"
              >No se encontraron productos.</TableCell
            ></TableRow
          >
        </TableBody>
      </Table>
    </div>
    <div class="flex items-center justify-end space-x-2 py-4">
      <Pagination
        v-if="pageCount > 1"
        v-model:page="currentPage"
        :total="totalProductos"
        :items-per-page="pagination.pageSize"
        :sibling-count="1"
        show-edges
      >
        <PaginationContent v-slot="{ items }">
          <PaginationPrevious /><template v-for="(item, index) in items">
            <PaginationItem
              v-if="item.type === 'page'"
              :key="index"
              :value="item.value"
              as-child
              ><Button
                class="w-10 h-10 p-0"
                :variant="item.value === currentPage ? 'default' : 'outline'"
                >{{ item.value }}</Button
              ></PaginationItem
            >
            <PaginationEllipsis
              v-else
              :key="item.type"
              :index="index" /></template
          ><PaginationNext />
        </PaginationContent>
      </Pagination>
    </div>
  </div>
</template>
