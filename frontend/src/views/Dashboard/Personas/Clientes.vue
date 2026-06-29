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
import { ArrowUpDown, ArrowUp, ArrowDown, ChevronDown, PlusCircle, Search, Users } from "lucide-vue-next";
import { h, ref, watch, onMounted, computed } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import DropdownAction from "@/components/tables/DataTableClientDropDown.vue";
import CrearClienteModal from "@/components/modals/CrearClienteModal.vue";
import { backend } from "@/../bridge/go/models";
import {
  ObtenerClientesPaginado,
  EliminarCliente,
  ActualizarCliente,
} from "@/../bridge/go/backend/Db";
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

const columns: ColumnDef<backend.Cliente>[] = [
  {
    accessorKey: "Nombre",
    header: sortHeader("Nombre"),
    cell: ({ row }) =>
      h("div", { class: "uppercase max-w-[600px] truncate font-medium" },
        `${row.original.Nombre} ${row.original.Apellido}`),
  },
  {
    accessorKey: "Documento",
    header: "Documento",
    cell: ({ row }) =>
      h("div", { class: "uppercase font-mono text-muted-foreground" },
        `${row.original.TipoID} ${row.original.NumeroID}`),
  },
  {
    accessorKey: "Email",
    header: "Email",
    cell: ({ row }) => h("div", { class: "lowercase text-muted-foreground" }, row.getValue("Email")),
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

  <div class="flex flex-col h-full">
    <!-- Header bar -->
    <div class="border-b px-4 py-3 flex items-center justify-between shrink-0">
      <div>
        <h1 class="text-sm font-semibold">Clientes</h1>
        <p class="text-xs text-muted-foreground">Gestiona la base de clientes de la farmacia</p>
      </div>
      <Button @click="isCreateModalOpen = true" class="h-7 gap-1.5 text-xs">
        <PlusCircle class="w-3.5 h-3.5" />
        Agregar Cliente
      </Button>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 bg-muted/10 flex items-center gap-2">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input
          class="h-7 text-xs pl-7 w-56"
          placeholder="Buscar por nombre, documento..."
          v-model="busqueda"
        />
      </div>
      <template v-if="busqueda">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <Search class="h-2.5 w-2.5" />
          {{ totalClientes }} resultado{{ totalClientes !== 1 ? 's' : '' }} en toda la base
        </span>
      </template>
      <template v-if="sorting.length">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <component :is="sorting[0]?.desc ? ArrowDown : ArrowUp" class="h-2.5 w-2.5" />
          Ordenado por {{ sorting[0]?.id }} (todos los registros)
        </span>
      </template>
      <div class="ml-auto">
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="outline" class="h-7 text-xs gap-1.5">
              Columnas <ChevronDown class="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="text-xs">
            <DropdownMenuCheckboxItem
              v-for="column in table.getAllColumns().filter(c => c.getCanHide())"
              :key="column.id"
              class="capitalize text-xs"
              :model-value="column.getIsVisible()"
              @update:model-value="(v) => column.toggleVisibility(!!v)"
            >
              {{ { Nombre: 'Nombre', Documento: 'Documento', Email: 'Email' }[column.id] ?? column.id }}
            </DropdownMenuCheckboxItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Table area -->
    <div class="flex-1 overflow-auto">
      <table data-supabase-table class="w-full text-xs border-collapse">
        <thead class="sticky top-0 z-10">
          <tr
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
            class="bg-muted border-b"
          >
            <th class="w-10 h-9 px-3 text-[10px] font-medium text-muted-foreground text-center border-r whitespace-nowrap">
              #
            </th>
            <th
              v-for="header in headerGroup.headers"
              :key="header.id"
              class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-left border-r last:border-r-0 whitespace-nowrap"
            >
              <FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-if="table.getRowModel().rows?.length">
            <tr
              v-for="(row, rowIndex) in table.getRowModel().rows"
              :key="row.id"
              class="border-b last:border-b-0 hover:bg-muted/40 transition-colors group"
            >
              <td class="h-9 px-3 font-mono text-[10px] text-muted-foreground/40 text-center border-r">
                {{ pagination.pageIndex * pagination.pageSize + rowIndex + 1 }}
              </td>
              <td
                v-for="cell in row.getVisibleCells()"
                :key="cell.id"
                class="h-9 px-3 border-r last:border-r-0"
              >
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </td>
            </tr>
          </template>
          <tr v-else>
            <td :colspan="columns.length + 1" class="h-40 text-center">
              <div class="flex flex-col items-center justify-center gap-2 text-muted-foreground">
                <Users class="h-8 w-8 opacity-20" />
                <span class="text-xs">No se encontraron clientes</span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer -->
    <div class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalClientes }} cliente(s)</span>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5">
          <span class="text-xs text-muted-foreground">Filas</span>
          <Select
            :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(value) => table.setPageSize(Number(value))"
          >
            <SelectTrigger class="h-7 w-16 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="size in [10, 25, 50]" :key="size" :value="`${size}`" class="text-xs">
                {{ size }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination
          v-if="pageCount > 1"
          v-model:page="currentPage"
          :total="totalClientes"
          :items-per-page="pagination.pageSize"
          :sibling-count="1"
          show-edges
        >
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious class="w-7 h-7" />
            <template v-for="(item, index) in items">
              <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                <Button
                  class="w-7 h-7 p-0 text-xs"
                  :variant="item.value === currentPage ? 'default' : 'outline'"
                >
                  {{ item.value }}
                </Button>
              </PaginationItem>
              <PaginationEllipsis v-else :key="item.type" :index="index" />
            </template>
            <PaginationNext class="w-7 h-7" />
          </PaginationContent>
        </Pagination>
      </div>
    </div>
  </div>
</template>
