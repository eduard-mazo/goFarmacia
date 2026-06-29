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
import { ArrowUpDown, ArrowUp, ArrowDown, ChevronDown, Search, ShieldCheck, User } from "lucide-vue-next";
import { Badge } from "@/components/ui/badge";
import { h, ref, watch, onMounted, computed } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import DropdownAction from "@/components/tables/DataTableVendedorDropDown.vue";
import { backend } from "@/../bridge/go/models";
import {
  ObtenerVendedoresPaginado,
  EliminarVendedor,
  ActualizarVendedor,
} from "@/../bridge/go/backend/Db";
import { toast } from "vue-sonner";
import { useAuthStore } from "@/stores/auth";
import { storeToRefs } from "pinia";

const authStore = useAuthStore();
const { user: authenticatedUser } = storeToRefs(authStore);

interface ObtenerVendedoresPaginadoResponse {
  Records: backend.Vendedor[];
  TotalRecords: number;
}

const listaVendedores = ref<backend.Vendedor[]>([]);
const totalVendedores = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

const cargarVendedores = async () => {
  try {
    const currentPage: number = pagination.value.pageIndex + 1;
    let sortBy = "";
    let sortOrder = "asc";
    if (sorting.value.length > 0) {
      sortBy = sorting.value[0]!.id;
      sortOrder = sorting.value[0]!.desc ? "desc" : "asc";
    }
    const response: ObtenerVendedoresPaginadoResponse =
      await ObtenerVendedoresPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value,
        sortBy,
        sortOrder
      );
    listaVendedores.value = response.Records || [];
    totalVendedores.value = response.TotalRecords || 0;
  } catch (error) {
    toast.error("Error al cargar vendedores", { description: `${error}` });
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

const columns: ColumnDef<backend.Vendedor>[] = [
  {
    accessorKey: "Nombre",
    header: sortHeader("Nombre"),
    cell: ({ row }) =>
      h("div", { class: "uppercase max-w-[300px] truncate font-medium" },
        `${row.original.Nombre} ${row.original.Apellido}`),
  },
  {
    accessorKey: "Cedula",
    header: sortHeader("Cédula"),
    cell: ({ row }) => h("div", { class: "font-mono text-muted-foreground" }, row.getValue("Cedula")),
  },
  {
    accessorKey: "Email",
    header: sortHeader("Email"),
    cell: ({ row }) =>
      h("div", { class: "lowercase max-w-[300px] truncate text-muted-foreground" }, row.getValue("Email")),
  },
  {
    accessorKey: "Role",
    header: "Rol",
    cell: ({ row }) => {
      const role = row.getValue("Role") as string;
      return h(
        Badge,
        {
          class: role === "admin"
            ? "bg-blue-100 text-blue-800 border-blue-200 border text-xs font-medium"
            : "bg-slate-100 text-slate-700 border-slate-200 border text-xs font-medium",
        },
        () => role === "admin" ? "Administrador" : "Cajero"
      );
    },
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      const isCurrentUser = row.original.UUID === authenticatedUser.value?.UUID;
      return h("div", { class: "relative" }, [
        h(DropdownAction, {
          vendedor: row.original,
          disabled: isCurrentUser,
          onEdit: (v: backend.Vendedor) => handleEdit(v),
          onDelete: (v: backend.Vendedor) => handleDelete(v),
        }),
      ]);
    },
  },
];

const table = useVueTable({
  get data() {
    return listaVendedores.value;
  },
  columns,
  manualPagination: true,
  manualSorting: true,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
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
  onPaginationChange: (updater) => valueUpdater(updater, pagination),
  onSortingChange: (updater) => valueUpdater(updater, sorting),
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => table.setPageIndex(newPage - 1),
});

async function handleEdit(vendedor: backend.Vendedor) {
  try {
    await ActualizarVendedor(vendedor);
    await cargarVendedores();
    toast.success("Vendedor editado con éxito", {
      description: `Nombre: ${vendedor.Nombre}, Cedula: ${vendedor.Cedula}`,
    });
  } catch (error) {
    toast.error("Error al actualizar el Vendedor", { description: `${error}` });
  }
}
async function handleDelete(vendedor: backend.Vendedor) {
  if (vendedor.UUID === authenticatedUser.value?.UUID) {
    toast.error("Acción no permitida", {
      description: "No puedes eliminar tu propio usuario.",
    });
    return;
  }
  try {
    await EliminarVendedor(vendedor.UUID);
    await cargarVendedores();
    toast.warning("Vendedor eliminado con éxito", {
      description: `Nombre: ${vendedor.Nombre}, Email: ${vendedor.Email}`,
    });
  } catch (error) {
    toast.error("Error al eliminar Vendedor", { description: `${error}` });
  }
}

onMounted(cargarVendedores);
watch(pagination, cargarVendedores, { deep: true });
watch(
  sorting,
  () => {
    pagination.value.pageIndex = 0;
    cargarVendedores();
  },
  { deep: true }
);
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    pagination.value.pageIndex = 0;
    cargarVendedores();
  }, 150);
});
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header bar -->
    <div class="border-b px-4 py-3 flex items-center justify-between shrink-0">
      <div>
        <h1 class="text-sm font-semibold">Vendedores</h1>
        <p class="text-xs text-muted-foreground">Administra el equipo de vendedores</p>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 bg-muted/10 flex items-center gap-2">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input
          class="h-7 text-xs pl-7 w-56"
          placeholder="Buscar por nombre, cédula..."
          v-model="busqueda"
        />
      </div>
      <template v-if="busqueda">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <Search class="h-2.5 w-2.5" />
          {{ totalVendedores }} resultado{{ totalVendedores !== 1 ? 's' : '' }} en todo el equipo
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
              {{ { Nombre: 'Nombre', Cedula: 'Cédula', Email: 'Email', Role: 'Rol' }[column.id] ?? column.id }}
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
                <User class="h-8 w-8 opacity-20" />
                <span class="text-xs">No se encontraron vendedores</span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer -->
    <div class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalVendedores }} vendedor(es)</span>
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
          :total="totalVendedores"
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
