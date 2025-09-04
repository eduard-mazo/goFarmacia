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

import { ArrowUpDown, ChevronDown } from "lucide-vue-next";
import { h, ref, watch, onMounted } from "vue";
import { valueUpdater } from "../../utils";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import DropdownAction from "../../components/DataTableDropDown.vue"; // Adjusted path if needed
import { backend } from "../../../wailsjs/go/models";
import {
  ObtenerVendedoresPaginado,
  EliminarVendedor,
} from "../../../wailsjs/go/backend/Db";

interface ObtenerVendedoresPaginadoResponse {
  Records: backend.Vendedor[];
  TotalRecords: number;
}

// --- State Management for Data and Pagination ---
const listaVendedores = ref<backend.Vendedor[]>([]);
const totalVendedores = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);

const pagination = ref<PaginationState>({
  pageIndex: 0, // Corresponds to `paginaActual = 1`
  pageSize: 10,
});

// --- Data Fetching from Go Backend ---
const cargarVendedores = async () => {
  try {
    // The backend expects page number starting from 1, TanStack uses 0-based index.
    const currentPage: number = pagination.value.pageIndex + 1;
    const response: ObtenerVendedoresPaginadoResponse =
      await ObtenerVendedoresPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value
      );
    listaVendedores.value = response.Records || [];
    totalVendedores.value = response.TotalRecords || 0;
  } catch (error) {
    console.error(`Error al cargar vendedores: ${error}`);
    // Here you can add a user-facing notification
  }
};

// --- Column Definitions for Vendedor ---
const columns: ColumnDef<backend.Vendedor>[] = [
  {
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
  },
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
      const vendedor = row.original;
      return h(
        "div",
        { class: "uppercase" },
        `${vendedor.Nombre} ${vendedor.Apellido}`
      );
    },
  },
  {
    accessorKey: "Cedula",
    header: ({ column }) => {
      return h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Cedula", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      );
    },
    cell: ({ row }) => h("div", row.getValue("Cedula")),
  },
  {
    accessorKey: "Email",
    header: ({ column }) => {
      return h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Email", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      );
    },
    cell: ({ row }) => h("div", { class: "lowercase" }, row.getValue("Email")),
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      const vendedor = row.original;
      return h("div", { class: "relative" }, [
        h(DropdownAction, {
          vendedor,
          onEdit: (v: backend.Vendedor) => handleEdit(v),
          onDelete: (v: backend.Vendedor) => handleDelete(v),
        }),
      ]);
    },
  },
];

// --- Table Instance with Server-Side Pagination ---
const table = useVueTable({
  get data() {
    return listaVendedores.value;
  },
  columns,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  manualPagination: true,
  get pageCount() {
    return Math.ceil(totalVendedores.value / pagination.value.pageSize);
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

// --- Action Handlers ---
function handleEdit(vendedor: backend.Vendedor) {
  console.log("Edit:", vendedor);
  alert(`Editing Vendedor ID: ${vendedor.id}`);
}

async function handleDelete(vendedor: backend.Vendedor) {
  console.log("Delete:", vendedor);
  if (confirm(`Are you sure you want to delete ${vendedor.Nombre}?`)) {
    try {
      await EliminarVendedor(vendedor.id);
      await cargarVendedores();
      alert("Vendedor eliminado con éxito.");
    } catch (error) {
      console.error(`Error al eliminar vendedor: ${error}`);
      alert(`Failed to delete vendedor: ${error}`);
    }
  }
}

// --- Lifecycle and Watchers ---
onMounted(cargarVendedores);

// Watch for pagination changes and fetch data immediately
watch(pagination, cargarVendedores, { deep: true });

// Debounce search input to avoid excessive API calls
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    // When a new search is performed, go back to the first page.
    // The pagination watcher will be triggered automatically to fetch the data.
    if (pagination.value.pageIndex !== 0) {
      pagination.value.pageIndex = 0;
    }
    // If we are already on the first page, the pagination watcher won't fire,
    // so we need to trigger the fetch manually.
    else {
      cargarVendedores();
    }
  }, 300); // Wait for 300ms of inactivity before searching
});
</script>

<template>
  <div class="w-full">
    <div class="flex items-center py-4">
      <Input
        class="max-w-sm"
        placeholder="Buscar por nombre, apellido, cédula..."
        :model-value="busqueda"
        @update:model-value="busqueda = String($event)"
      />
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="outline" class="ml-auto">
            Columns <ChevronDown class="ml-2 h-4 w-4" />
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
            <TableRow
              v-for="row in table.getRowModel().rows"
              :key="row.id"
              :data-state="row.getIsSelected() && 'selected'"
            >
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
    <div class="flex items-center justify-end space-x-2 py-4">
      <div class="flex-1 text-sm text-muted-foreground">
        {{ table.getFilteredSelectedRowModel().rows.length }} of
        {{ table.getCoreRowModel().rows.length }} row(s) selected.
      </div>
      <div class="space-x-2">
        <Button
          variant="outline"
          size="sm"
          :disabled="!table.getCanPreviousPage()"
          @click="table.previousPage()"
        >
          Anterior
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="!table.getCanNextPage()"
          @click="table.nextPage()"
        >
          Siguiente
        </Button>
      </div>
    </div>
  </div>
</template>
