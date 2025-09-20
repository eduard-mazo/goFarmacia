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
import DropdownAction from "@/components/tables/DataTableClientDropDown.vue";
import CrearClienteModal from "@/components/modals/CrearClienteModal.vue";
import { backend } from "@/../wailsjs/go/models";
import {
  ObtenerClientesPaginado,
  EliminarCliente,
  ActualizarCliente,
} from "@/../wailsjs/go/backend/Db";
import { toast } from "vue-sonner";

interface ObtenerClientePaginadoResponse {
  Records: backend.Cliente[];
  TotalRecords: number;
}

// --- State Management for Data and Pagination ---
const listaClientes = ref<backend.Cliente[]>([]);
const totalClientes = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);
const isCreateModalOpen = ref(false); // Estado para el modal

const pagination = ref<PaginationState>({
  pageIndex: 0, // Corresponds to `paginaActual = 1`
  pageSize: 15,
});

// --- Data Fetching from Go Backend ---
const cargarClientes = async () => {
  try {
    // The backend expects page number starting from 1, TanStack uses 0-based index.
    const currentPage: number = pagination.value.pageIndex + 1;
    const response: ObtenerClientePaginadoResponse =
      await ObtenerClientesPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value
      );
    listaClientes.value = response.Records || [];
    totalClientes.value = response.TotalRecords || 0;
  } catch (error) {
    console.error(`Error al cargar clientes: ${error}`);
    toast.error("Error al cargar clientes", { description: `${error}` });
  }
};

// --- Column Definitions for Cliente ---
const columns: ColumnDef<backend.Cliente>[] = [
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
      const cliente = row.original;
      return h(
        "div",
        { class: "uppercase" },
        `${cliente.Nombre} ${cliente.Apellido}`
      );
    },
  },
  {
    accessorKey: "Documento",
    header: "Documento",
    cell: ({ row }) => {
      const cliente = row.original;
      return h(
        "div",
        { class: "uppercase" },
        `${cliente.TipoID} ${cliente.NumeroID}`
      );
    },
  },
  {
    accessorKey: "Email",
    header: "Email",
    cell: ({ row }) => h("div", { class: "lowercase" }, row.getValue("Email")),
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      const cliente = row.original;
      return h("div", { class: "relative" }, [
        h(DropdownAction, {
          cliente,
          onEdit: (v: backend.Cliente) => handleEdit(v),
          onDelete: (v: backend.Cliente) => handleDelete(v),
        }),
      ]);
    },
  },
];

// --- Table Instance with Server-Side Pagination ---
const table = useVueTable({
  get data() {
    return listaClientes.value;
  },
  columns,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  manualPagination: true,
  get pageCount() {
    return Math.ceil(totalClientes.value / pagination.value.pageSize);
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

const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => {
    table.setPageIndex(newPage - 1);
  },
});

// --- Action Handlers ---
async function handleEdit(cliente: backend.Cliente) {
  try {
    await ActualizarCliente(cliente);
    toast.success("Cliente actualizado", {
      description: `Los datos de ${cliente.Nombre} han sido guardados.`,
    });
    await cargarClientes();
  } catch (error) {
    console.error(`Error al actualizar el cliente: ${error}`);
    toast.error("Error al actualizar", { description: `${error}` });
  }
}

async function handleDelete(cliente: backend.Cliente) {
  if (confirm(`¿Estás seguro de que quieres eliminar a ${cliente.Nombre}?`)) {
    try {
      await EliminarCliente(cliente.id);
      await cargarClientes();
      toast.success("Cliente eliminado con éxito.");
    } catch (error) {
      console.error(`Error al eliminar cliente: ${error}`);
      toast.error("Error al eliminar", { description: `${error}` });
    }
  }
}

function handleClienteCreated() {
  cargarClientes();
}

// --- Lifecycle and Watchers ---
onMounted(cargarClientes);

watch(pagination, cargarClientes, { deep: true });

let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    if (pagination.value.pageIndex !== 0) {
      pagination.value.pageIndex = 0;
    } else {
      cargarClientes();
    }
  }, 300);
});
</script>

<template>
  <CrearClienteModal
    v-model:open="isCreateModalOpen"
    @client-created="handleClienteCreated"
  />
  <div class="w-full">
    <div class="flex items-center py-4 gap-2">
      <Input
        class="max-w-sm h-10"
        placeholder="Buscar por nombre, documento, email..."
        :model-value="busqueda"
        @update:model-value="busqueda = String($event)"
      />
      <Button @click="isCreateModalOpen = true" class="h-10">
        <PlusCircle class="w-4 h-4 mr-2" />
        Agregar Cliente
      </Button>
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
            @update:model-value="(value) => column.toggleVisibility(!!value)"
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
      <div class="space-x-2">
        <Pagination
          v-if="pageCount > 1"
          v-model:page="currentPage"
          :total="totalClientes"
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
