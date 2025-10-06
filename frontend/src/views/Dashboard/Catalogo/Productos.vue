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
import { Card } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { ArrowUpDown, ChevronDown, PlusCircle } from "lucide-vue-next";
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
import DropdownAction from "@/components/tables/DataTableProductDropDown.vue";
import CrearProductoModal from "@/components/modals/CrearProductoModal.vue";
import { backend } from "@/../wailsjs/go/models";
import {
  ObtenerProductosPaginado,
  EliminarProducto,
  ActualizarProducto,
} from "@/../wailsjs/go/backend/Db";
import { toast } from "vue-sonner";

interface ObtenerProductosPaginadoResponse {
  Records: backend.Producto[];
  TotalRecords: number;
}

const listaProductos = ref<backend.Producto[]>([]);
const totalProductos = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);
const isCreateModalOpen = ref(false);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

const cargarProductos = async () => {
  try {
    const currentPage: number = pagination.value.pageIndex + 1;
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
    toast.error("Error al cargar Productos", { description: `${error}` });
  }
};

const columns: ColumnDef<backend.Producto>[] = [
  {
    accessorKey: "Nombre",
    header: ({ column }) =>
      h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Nombre", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
    cell: ({ row }) =>
      h(
        "div",
        { class: "uppercase max-w-[600px] truncate" },
        row.getValue("Nombre")
      ),
  },
  {
    accessorKey: "Codigo",
    header: ({ column }) =>
      h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Código", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
    cell: ({ row }) => h("div", row.getValue("Codigo")),
  },
  {
    accessorKey: "PrecioVenta",
    header: ({ column }) =>
      h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Precio Venta", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
    cell: ({ row }) =>
      h(
        "div",
        { class: "text-left font-medium" },
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
    cell: ({ row }) =>
      h("div", { class: "text-center" }, row.getValue("Stock")),
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) =>
      h("div", { class: "relative" }, [
        h(DropdownAction, {
          producto: row.original,
          onEdit: (p: backend.Producto) => handleEdit(p),
          onDelete: (p: backend.Producto) => handleDelete(p),
        }),
      ]),
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

async function handleEdit(producto: backend.Producto) {
  try {
    await ActualizarProducto(producto);
    await cargarProductos();
    toast.success("Producto editado con éxito", {
      description: `Nombre: ${producto.Nombre}, Código: ${producto.Codigo}`,
    });
  } catch (error) {
    toast.error("Error al actualizar el producto", { description: `${error}` });
  }
}
async function handleDelete(producto: backend.Producto) {
  try {
    await EliminarProducto(producto.id);
    await cargarProductos();
    toast.warning("Producto eliminado con éxito", {
      description: `Nombre: ${producto.Nombre}, Código: ${producto.Codigo}`,
    });
  } catch (error) {
    toast.error("Error al eliminar el producto", { description: `${error}` });
  }
}

function handleProductCreated() {
  cargarProductos();
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
  <CrearProductoModal
    v-model:open="isCreateModalOpen"
    @product-created="handleProductCreated"
  />
  <div class="w-full">
    <div class="flex items-center py-4 gap-2">
      <Input
        class="max-w-sm h-10"
        placeholder="Buscar por nombre o código..."
        :model-value="busqueda"
        @update:model-value="busqueda = String($event)"
      />
      <Button @click="isCreateModalOpen = true" class="h-10"
        ><PlusCircle class="w-4 h-4 mr-2" />Agregar Producto</Button
      >
      <DropdownMenu>
        <DropdownMenuTrigger as-child
          ><Button variant="outline" class="ml-auto h-10"
            >Columnas <ChevronDown class="ml-2 h-4 w-4" /></Button
        ></DropdownMenuTrigger>
        <DropdownMenuContent align="end"
          ><DropdownMenuCheckboxItem
            v-for="column in table
              .getAllColumns()
              .filter((column) => column.getCanHide())"
            :key="column.id"
            class="capitalize"
            :model-value="column.getIsVisible()"
            @update:model-value="(value) => column.toggleVisibility(!!value)"
            >{{ column.id }}</DropdownMenuCheckboxItem
          ></DropdownMenuContent
        >
      </DropdownMenu>
    </div>
    <Card class="py-0">
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
              >No se encontraron resultados.</TableCell
            ></TableRow
          >
        </TableBody>
      </Table>
    </Card>
    <div class="flex items-center justify-between space-x-2 py-4">
      <div class="flex-1 text-sm text-muted-foreground">
        Total, {{ totalProductos }} producto(s).
      </div>
      <div class="flex items-center space-x-4">
        <div class="flex items-center space-x-2">
          <p class="text-sm font-medium">Filas</p>
          <Select
            :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(value) => table.setPageSize(Number(value))"
          >
            <SelectTrigger class="h-8 w-[70px]">
              <SelectValue
                :placeholder="`${table.getState().pagination.pageSize}`"
              />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectItem
                v-for="size in [5, 10, 15, 20]"
                :key="size"
                :value="`${size}`"
              >
                {{ size }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination
          v-if="pageCount > 1"
          v-model:page="currentPage"
          :total="totalProductos"
          :items-per-page="pagination.pageSize"
          :sibling-count="1"
          show-edges
        >
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious />
            <template v-for="(item, index) in items">
              <PaginationItem
                v-if="item.type === 'page'"
                :key="index"
                :value="item.value"
                as-child
              >
                <Button
                  class="w-10 h-10 p-0"
                  :variant="item.value === currentPage ? 'default' : 'outline'"
                >
                  {{ item.value }}
                </Button>
              </PaginationItem>
              <PaginationEllipsis v-else :key="item.type" :index="index" />
            </template>
            <PaginationNext />
          </PaginationContent>
        </Pagination>
      </div>
    </div>
  </div>
</template>
