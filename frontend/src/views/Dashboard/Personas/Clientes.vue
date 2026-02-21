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
// --- NUEVAS IMPORTACIONES ---
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
import { ArrowUpDown, ChevronDown, PlusCircle, Search } from "lucide-vue-next";
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

const listaClientes = ref<backend.Cliente[]>([]);
const totalClientes = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);
const isCreateModalOpen = ref(false);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

const cargarClientes = async () => {
  try {
    const currentPage: number = pagination.value.pageIndex + 1;
    let sortBy = "";
    let sortOrder = "asc";
    if (sorting.value.length > 0) {
      sortBy = sorting.value[0]!.id;
      sortOrder = sorting.value[0]!.desc ? "desc" : "asc";
    }
    const response: ObtenerClientePaginadoResponse =
      await ObtenerClientesPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value,
        sortBy,
        sortOrder
      );
    listaClientes.value = response.Records || [];
    totalClientes.value = response.TotalRecords || 0;
  } catch (error) {
    toast.error("Error al cargar clientes", { description: `${error}` });
  }
};

const columns: ColumnDef<backend.Cliente>[] = [
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
        `${row.original.Nombre} ${row.original.Apellido}`
      ),
  },
  {
    accessorKey: "Documento",
    header: "Documento",
    cell: ({ row }) =>
      h(
        "div",
        { class: "uppercase" },
        `${row.original.TipoID} ${row.original.NumeroID}`
      ),
  },
  {
    accessorKey: "Email",
    header: "Email",
    cell: ({ row }) => h("div", { class: "lowercase" }, row.getValue("Email")),
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) =>
      h("div", { class: "relative" }, [
        h(DropdownAction, {
          cliente: row.original,
          onEdit: (c: backend.Cliente) => handleEdit(c),
          onDelete: (c: backend.Cliente) => handleDelete(c),
        }),
      ]),
  },
];

const table = useVueTable({
  get data() {
    return listaClientes.value;
  },
  columns,
  manualPagination: true,
  manualSorting: true,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
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
  onPaginationChange: (updater) => valueUpdater(updater, pagination),
  onSortingChange: (updater) => valueUpdater(updater, sorting),
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => table.setPageIndex(newPage - 1),
});

async function handleEdit(cliente: backend.Cliente) {
  try {
    await ActualizarCliente(cliente);
    await cargarClientes();
    toast.success("Cliente editado con éxito", {
      description: `Nombre: ${cliente.Nombre}, ID: ${cliente.NumeroID}`,
    });
  } catch (error) {
    toast.error("Error al actualizar", { description: `${error}` });
  }
}
async function handleDelete(cliente: backend.Cliente) {
  try {
    await EliminarCliente(cliente.UUID);
    await cargarClientes();
    toast.warning("Cliente eliminado con éxito", {
      description: `Nombre: ${cliente.Nombre}, ID: ${cliente.NumeroID}`,
    });
  } catch (error) {
    toast.error("Error al eliminar", { description: `${error}` });
  }
}
function handleClienteCreated() {
  cargarClientes();
}

onMounted(cargarClientes);
watch(pagination, cargarClientes, { deep: true });
watch(
  sorting,
  () => {
    pagination.value.pageIndex = 0;
    cargarClientes();
  },
  { deep: true }
);
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    pagination.value.pageIndex = 0;
    cargarClientes();
  }, 150);
});
</script>

<template>
  <CrearClienteModal v-model:open="isCreateModalOpen" @client-created="handleClienteCreated" />
  <div class="p-6 space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Clientes</h1>
        <p class="text-sm text-muted-foreground mt-0.5">Gestiona la base de clientes de la farmacia</p>
      </div>
      <Button @click="isCreateModalOpen = true" class="h-9 gap-2">
        <PlusCircle class="w-4 h-4" />Agregar Cliente
      </Button>
    </div>

    <!-- Toolbar -->
    <div class="flex items-center gap-3">
      <div class="relative flex-1 max-w-xs">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
        <Input class="pl-9 h-9" placeholder="Buscar por nombre, documento..." :model-value="busqueda"
          @update:model-value="busqueda = String($event)" />
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="outline" class="h-9 gap-2">
            Columnas <ChevronDown class="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuCheckboxItem v-for="column in table.getAllColumns().filter(c => c.getCanHide())"
            :key="column.id" class="capitalize" :model-value="column.getIsVisible()"
            @update:model-value="(v) => column.toggleVisibility(!!v)">
            {{ column.id }}
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Table -->
    <div class="rounded-lg border bg-card overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id">
            <TableHead v-for="header in headerGroup.headers" :key="header.id">
              <FlexRender v-if="!header.isPlaceholder" :render="header.column.columnDef.header"
                :props="header.getContext()" />
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="table.getRowModel().rows?.length">
            <TableRow v-for="row in table.getRowModel().rows" :key="row.id">
              <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
          </template>
          <TableRow v-else>
            <TableCell :colspan="columns.length" class="h-32 text-center text-muted-foreground">
              No se encontraron clientes.
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <!-- Pagination footer -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">{{ totalClientes }} cliente(s) en total</p>
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-2">
          <p class="text-sm text-muted-foreground">Filas</p>
          <Select :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(value) => table.setPageSize(Number(value))">
            <SelectTrigger class="h-8 w-[68px]">
              <SelectValue :placeholder="`${table.getState().pagination.pageSize}`" />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="size in [5, 10, 15, 20]" :key="size" :value="`${size}`">{{ size }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination v-if="pageCount > 1" v-model:page="currentPage" :total="totalClientes"
          :items-per-page="pagination.pageSize" :sibling-count="1" show-edges>
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious />
            <template v-for="(item, index) in items">
              <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                <Button class="w-9 h-9 p-0" :variant="item.value === currentPage ? 'default' : 'outline'">
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
