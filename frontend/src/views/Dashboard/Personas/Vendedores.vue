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
import { ArrowUpDown, ChevronDown } from "lucide-vue-next";
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
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      const isCurrentUser = row.original.id === authenticatedUser.value?.id;
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
  if (vendedor.id === authenticatedUser.value?.id) {
    toast.error("Acción no permitida", {
      description: "No puedes eliminar tu propio usuario.",
    });
    return;
  }
  try {
    await EliminarVendedor(vendedor.uuid);
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
  <div class="w-full">
    <div class="flex items-center py-4 gap-2">
      <Input
        class="max-w-sm h-10"
        placeholder="Buscar por nombre, cédula..."
        :model-value="busqueda"
        @update:model-value="busqueda = String($event)"
      />
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
              >No se encontraron resultados.</TableCell
            ></TableRow
          >
        </TableBody>
      </Table>
    </div>
    <div class="flex items-center justify-between space-x-2 py-4">
      <div class="flex-1 text-sm text-muted-foreground">
        Total, {{ totalVendedores }} vendedor(es).
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
