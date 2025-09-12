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

import { ArrowUpDown, ChevronDown } from "lucide-vue-next";
import { h, ref, watch, onMounted, computed } from "vue";
import { valueUpdater } from "../../utils";

import { Button } from "@/components/ui/button";
//import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import DropdownAction from "../../components/DataTableProductDropDown.vue";
import { backend } from "../../../wailsjs/go/models";
import {
  ObtenerProductosPaginado,
  EliminarProducto,
  ActualizarProducto,
} from "../../../wailsjs/go/backend/Db";

interface ObtenerProductosPaginadoResponse {
  Records: backend.Producto[];
  TotalRecords: number;
}

// --- State Management for Data and Pagination ---
const listaProductos = ref<backend.Producto[]>([]);
const totalProductos = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);

const pagination = ref<PaginationState>({
  pageIndex: 0, // 0-based index for TanStack Table
  pageSize: 15,
});

// --- Data Fetching from Go Backend ---
const cargarProductos = async () => {
  try {
    // El backend espera una página basada en 1, así que sumamos 1
    const currentPageBackend: number = pagination.value.pageIndex + 1;
    const response: ObtenerProductosPaginadoResponse =
      await ObtenerProductosPaginado(
        currentPageBackend,
        pagination.value.pageSize,
        busqueda.value
      );
    listaProductos.value = response.Records || [];
    totalProductos.value = response.TotalRecords || 0;
  } catch (error) {
    console.error(`Error al cargar Productos: ${error}`);
  }
};

// --- Column Definitions for Producto ---
const columns: ColumnDef<backend.Producto>[] = [
  /*  {
    id: "select",
    header: ({ table }) =>
      h(Checkbox, {
        checked: table.getIsAllPageRowsSelected(),
        "onUpdate:checked": (value: boolean) =>
          table.toggleAllPageRowsSelected(!!value),
        ariaLabel: "Select all",
      }),
    cell: ({ row }) =>
      h(Checkbox, {
        checked: row.getIsSelected(),
        "onUpdate:checked": (value: boolean) => row.toggleSelected(!!value),
        ariaLabel: "Select row",
      }),
    enableSorting: false,
    enableHiding: false,
  },*/
  {
    accessorKey: "Nombre",
    header: ({ column }) => {
      return h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Nombre", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      );
    },
    cell: ({ row }) => {
      return h("div", { class: "capitalize" }, row.getValue("Nombre"));
    },
  },
  {
    accessorKey: "Codigo",
    header: ({ column }) => {
      return h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Código", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      );
    },
    cell: ({ row }) => h("div", row.getValue("Codigo")),
  },
  {
    accessorKey: "PrecioVenta",
    header: ({ column }) => {
      return h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Precio Venta", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      );
    },
    cell: ({ row }) => {
      const amount = parseFloat(row.getValue("PrecioVenta"));
      const formatted = new Intl.NumberFormat("es-CO", {
        style: "currency",
        currency: "COP",
      }).format(amount);
      return h("div", { class: "text-left font-medium" }, formatted);
    },
  },
  {
    accessorKey: "Stock",
    header: ({ column }) => {
      return h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Stock", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      );
    },
    cell: ({ row }) => h("div", row.getValue("Stock")),
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      const producto = row.original;
      return h("div", { class: "relative" }, [
        h(DropdownAction, {
          producto,
          onEdit: (p: backend.Producto) => handleEdit(p),
          onDelete: (p: backend.Producto) => handleDelete(p),
        }),
      ]);
    },
  },
];

// --- Table Instance with Server-Side Pagination ---
const table = useVueTable({
  get data() {
    return listaProductos.value;
  },
  columns,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  manualPagination: true,
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
  onPaginationChange: (updater) => {
    if (typeof updater === "function") {
      pagination.value = updater(pagination.value);
    } else {
      pagination.value = updater;
    }
  },
  onSortingChange: (updaterOrValue) => valueUpdater(updaterOrValue, sorting),
});

// --- Computed properties for Pagination ---
const pageCount = computed(() => table.getPageCount());

// 🌉 Puente entre la página base 1 (UI) y el pageIndex base 0 (TanStack Table)
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => {
    table.setPageIndex(newPage - 1);
  },
});

// --- Action Handlers ---
async function handleEdit(producto: backend.Producto) {
  try {
    await ActualizarProducto(producto);
    await cargarProductos();
  } catch (error) {
    console.error(`Error al actualizar el producto: ${error}`);
  }
}

async function handleDelete(producto: backend.Producto) {
  try {
    await EliminarProducto(producto.id);
    await cargarProductos();
  } catch (error) {
    console.error(`Error al eliminar el producto: ${error}`);
  }
}

// --- Lifecycle and Watchers ---
onMounted(cargarProductos);

watch(pagination, cargarProductos, { deep: true });

let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    if (pagination.value.pageIndex !== 0) {
      table.setPageIndex(0);
    } else {
      cargarProductos();
    }
  }, 300);
});
</script>

<template>
  <div class="w-full">
    <div class="flex items-center py-4">
      <Input
        class="max-w-sm"
        placeholder="Buscar por nombre o código..."
        :model-value="busqueda"
        @update:model-value="busqueda = String($event)"
      />
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="outline" class="ml-auto">
            Columnas <ChevronDown class="ml-2 h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuCheckboxItem
            v-for="column in table
              .getAllColumns()
              .filter((column) => column.getCanHide())"
            :key="column.id"
            class="capitalize"
            :model-value="column.getIsVisible()"
            @update:model-value="
              (value) => {
                column.toggleVisibility(!!value);
              }
            "
          >
            {{ column.id }}
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <div class="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
          >
            <TableHead v-for="header in headerGroup.headers" :key="header.id">
              <FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="table.getRowModel().rows?.length">
            <TableRow v-for="row in table.getRowModel().rows" :key="row.id">
              <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id">
                <FlexRender
                  :render="cell.column.columnDef.cell"
                  :props="cell.getContext()"
                />
              </TableCell>
            </TableRow>
          </template>
          <TableRow v-else>
            <TableCell :colspan="columns.length" class="h-24 text-center">
              No se encontraron resultados.
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
    <div class="flex items-center justify-between space-x-2 py-4">
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
</template>
