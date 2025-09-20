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
import { valueUpdater } from "@/utils";

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
import DropdownAction from "@/components/tables/DataTableVendedorDropDown.vue"; // Adjusted path if needed
import { backend } from "@/../wailsjs/go/models";
import {
  ObtenerVendedoresPaginado,
  EliminarVendedor,
  ActualizarVendedor,
} from "@/../wailsjs/go/backend/Db";
import { toast } from "vue-sonner";

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
  pageSize: 15,
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
    toast.error("Error al cargar vendedores", { description: `${error}` });
  }
};

// --- Column Definitions for Vendedor ---
const columns: ColumnDef<backend.Vendedor>[] = [
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

// --- Computed properties for Pagination ---
const pageCount = computed(() => table.getPageCount());

//Puente entre la página base 1 (UI) y el pageIndex base 0 (TanStack Table)
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => {
    table.setPageIndex(newPage - 1);
  },
});

// --- Action Handlers ---
async function handleEdit(vendedor: backend.Vendedor) {
  try {
    await ActualizarVendedor(vendedor);
    await cargarVendedores();
    toast.success("Producto editado con éxito", {
      description: `Nombre: ${vendedor.Nombre}, Cedula: ${vendedor.Cedula}`,
    });
  } catch (error) {
    toast.error("Error al actualizar el VendeActualizarVendedor", {
      description: `${error}`,
    });
  }
}

async function handleDelete(vendedor: backend.Vendedor) {
  try {
    await EliminarVendedor(vendedor.id);
    await cargarVendedores();
    toast.warning("Vendedor eliminado con éxito", {
      description: `Nombre: ${vendedor.Nombre}, Email: ${vendedor.Email}`,
    });
  } catch (error) {
    toast.error("Error al eliminar vendedor", { description: `${error}` });
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
    if (pagination.value.pageIndex !== 0) {
      pagination.value.pageIndex = 0;
    } else {
      cargarVendedores();
    }
  }, 300);
});
</script>

<template>
  <div class="w-full">
    <div class="flex items-center py-4 gap-2">
      <Input
        class="max-w-sm h-10"
        placeholder="Buscar por nombre, apellido, cédula..."
        :model-value="busqueda"
        @update:model-value="busqueda = String($event)"
      />
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="outline" class="ml-auto h-10">
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
    <div class="flex items-center justify-end space-x-2 py-4">
      <div class="space-x-2">
        <Pagination
          v-if="pageCount > 1"
          v-model:page="currentPage"
          :total="totalVendedores"
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
