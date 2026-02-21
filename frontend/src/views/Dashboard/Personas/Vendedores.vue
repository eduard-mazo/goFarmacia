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
import { ArrowUpDown, ChevronDown, Search, ShieldCheck, User } from "lucide-vue-next";
import { Badge } from "@/components/ui/badge";
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
import DropdownAction from "@/components/tables/DataTableVendedorDropDown.vue";
import { backend } from "@/../wailsjs/go/models";
import {
  ObtenerVendedoresPaginado,
  EliminarVendedor,
  ActualizarVendedor,
} from "@/../wailsjs/go/backend/Db";
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

const columns: ColumnDef<backend.Vendedor>[] = [
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
        { class: "uppercase max-w-[300px] truncate" },
        `${row.original.Nombre} ${row.original.Apellido}`
      ),
  },
  {
    accessorKey: "Cedula",
    header: ({ column }) =>
      h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Cédula", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
    cell: ({ row }) => h("div", row.getValue("Cedula")),
  },
  {
    accessorKey: "Email",
    header: ({ column }) =>
      h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Email", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
    cell: ({ row }) =>
      h(
        "div",
        { class: "lowercase max-w-[300px] truncate" },
        row.getValue("Email")
      ),
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
  <div class="p-6 space-y-6">
    <!-- Page header -->
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Vendedores</h1>
      <p class="text-sm text-muted-foreground mt-0.5">Administra el equipo de vendedores</p>
    </div>

    <!-- Toolbar -->
    <div class="flex items-center gap-3">
      <div class="relative flex-1 max-w-xs">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
        <Input class="pl-9 h-9" placeholder="Buscar por nombre, cédula..." :model-value="busqueda"
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
              No se encontraron vendedores.
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <!-- Pagination footer -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">{{ totalVendedores }} vendedor(es) en total</p>
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
        <Pagination v-if="pageCount > 1" v-model:page="currentPage" :total="totalVendedores"
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
