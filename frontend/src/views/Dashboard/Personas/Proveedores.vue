<script setup lang="ts">
import type {
  ColumnDef,
  PaginationState,
} from "@tanstack/vue-table";
import {
  FlexRender,
  getCoreRowModel,
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
import { ChevronDown, PlusCircle, Search } from "lucide-vue-next";
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
import DropdownAction from "@/components/tables/DataTableProveedorDropDown.vue";
import CrearProveedorModal from "@/components/modals/CrearProveedorModal.vue";
import { backend } from "@/../wailsjs/go/models";
import {
  ObtenerProveedoresPaginado,
  EliminarProveedor,
  ActualizarProveedor,
} from "@/../wailsjs/go/backend/Db";
import { toast } from "vue-sonner";

interface ObtenerProveedorPaginadoResponse {
  Records: backend.Proveedor[];
  TotalRecords: number;
}

const listaProveedores = ref<backend.Proveedor[]>([]);
const totalProveedores = ref(0);
const busqueda = ref("");
const isCreateModalOpen = ref(false);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

const cargarProveedores = async () => {
  try {
    const currentPage: number = pagination.value.pageIndex + 1;
    const response: ObtenerProveedorPaginadoResponse =
      await ObtenerProveedoresPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value
      );
    listaProveedores.value = response.Records || [];
    totalProveedores.value = response.TotalRecords || 0;
  } catch (error) {
    toast.error("Error al cargar proveedores", { description: `${error}` });
  }
};

const columns: ColumnDef<backend.Proveedor>[] = [
  {
    accessorKey: "Nombre",
    header: "Nombre",
    cell: ({ row }) =>
      h(
        "div",
        { class: "uppercase max-w-[600px] truncate" },
        row.getValue("Nombre")
      ),
  },
  {
    accessorKey: "Telefono",
    header: "Teléfono",
    cell: ({ row }) => h("div", {}, row.getValue("Telefono")),
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
          proveedor: row.original,
          onEdit: (p: backend.Proveedor) => handleEdit(p),
          onDelete: (p: backend.Proveedor) => handleDelete(p),
        }),
      ]),
  },
];

const table = useVueTable({
  get data() {
    return listaProveedores.value;
  },
  columns,
  manualPagination: true,
  getCoreRowModel: getCoreRowModel(),
  get pageCount() {
    return Math.ceil(totalProveedores.value / pagination.value.pageSize);
  },
  state: {
    get pagination() {
      return pagination.value;
    },
  },
  onPaginationChange: (updater) => valueUpdater(updater, pagination),
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => table.setPageIndex(newPage - 1),
});

async function handleEdit(proveedor: backend.Proveedor) {
  try {
    await ActualizarProveedor(proveedor);
    await cargarProveedores();
    toast.success("Proveedor editado con éxito", {
      description: `Nombre: ${proveedor.Nombre}`,
    });
  } catch (error) {
    toast.error("Error al actualizar", { description: `${error}` });
  }
}

async function handleDelete(proveedor: backend.Proveedor) {
  try {
    await EliminarProveedor(proveedor.uuid);
    await cargarProveedores();
    toast.warning("Proveedor eliminado con éxito", {
      description: `Nombre: ${proveedor.Nombre}`,
    });
  } catch (error) {
    toast.error("Error al eliminar", { description: `${error}` });
  }
}

function handleProveedorCreated() {
  cargarProveedores();
}

onMounted(cargarProveedores);
watch(pagination, cargarProveedores, { deep: true });
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    pagination.value.pageIndex = 0;
    cargarProveedores();
  }, 150);
});
</script>

<template>
  <CrearProveedorModal v-model:open="isCreateModalOpen" @proveedor-created="handleProveedorCreated" />
  <div class="flex flex-col h-full">
    <!-- Header bar -->
    <div class="shrink-0 border-b px-4 py-3 flex items-center justify-between bg-background">
      <div>
        <p class="text-sm font-semibold">Proveedores</p>
        <p class="text-xs text-muted-foreground mt-0.5">Gestiona los proveedores de la farmacia</p>
      </div>
      <Button size="sm" class="h-7 gap-1.5 text-xs" @click="isCreateModalOpen = true">
        <PlusCircle class="h-3.5 w-3.5" />Agregar Proveedor
      </Button>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-56" placeholder="Buscar por nombre, email..." />
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="outline" class="h-7 gap-1.5 text-xs">
            Columnas <ChevronDown class="h-3.5 w-3.5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="text-xs">
          <DropdownMenuCheckboxItem
            v-for="column in table.getAllColumns().filter(c => c.getCanHide())"
            :key="column.id"
            class="capitalize text-xs"
            :model-value="column.getIsVisible()"
            @update:model-value="(v) => column.toggleVisibility(!!v)">
            {{ column.id }}
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Table area -->
    <div class="flex-1 overflow-auto">
      <table data-supabase-table class="w-full text-xs border-collapse">
        <thead class="sticky top-0 z-10">
          <tr class="bg-muted border-b">
            <th class="h-9 w-10 px-3 text-[10px] font-normal text-muted-foreground/40 text-center border-r">#</th>
            <th
              v-for="header in table.getHeaderGroups()[0]?.headers"
              :key="header.id"
              v-show="header.column.getIsVisible()"
              class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-left border-r last:border-r-0 whitespace-nowrap">
              <FlexRender v-if="!header.isPlaceholder" :render="header.column.columnDef.header" :props="header.getContext()" />
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-if="table.getRowModel().rows?.length">
            <tr
              v-for="(row, rowIndex) in table.getRowModel().rows"
              :key="row.id"
              class="border-b hover:bg-muted/40 transition-colors group">
              <td class="h-9 px-3 text-center text-[10px] font-mono text-muted-foreground/40 border-r w-10 select-none">
                {{ pagination.pageIndex * pagination.pageSize + rowIndex + 1 }}
              </td>
              <td
                v-for="cell in row.getVisibleCells()"
                :key="cell.id"
                class="h-9 px-3 border-r last:border-r-0">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </td>
            </tr>
          </template>
          <tr v-else>
            <td :colspan="columns.length + 1" class="h-32 text-center text-sm text-muted-foreground">
              No se encontraron proveedores.
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer -->
    <div class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalProveedores }} proveedor(es) en total</span>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5">
          <span class="text-xs text-muted-foreground">Filas</span>
          <Select
            :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(value) => table.setPageSize(Number(value))">
            <SelectTrigger class="h-7 w-16 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="size in [10, 20, 50]" :key="size" :value="`${size}`" class="text-xs">{{ size }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination
          v-if="pageCount > 1"
          v-model:page="currentPage"
          :total="totalProveedores"
          :items-per-page="pagination.pageSize"
          :sibling-count="1"
          show-edges>
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious class="h-7 w-7" />
            <template v-for="(item, index) in items">
              <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                <Button class="w-7 h-7 p-0 text-xs" :variant="item.value === currentPage ? 'default' : 'outline'">
                  {{ item.value }}
                </Button>
              </PaginationItem>
              <PaginationEllipsis v-else :key="item.type" :index="index" />
            </template>
            <PaginationNext class="h-7 w-7" />
          </PaginationContent>
        </Pagination>
      </div>
    </div>
  </div>
</template>
