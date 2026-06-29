<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/stores/auth";
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
import { ArrowUpDown, ArrowUp, ArrowDown, ChevronDown, PlusCircle, Search } from "lucide-vue-next";
import { h, ref, watch, onMounted, computed } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import DropdownAction from "@/components/tables/DataTableProductDropDown.vue";
import CrearProductoModal from "@/components/modals/CrearProductoModal.vue";
import { backend } from "@/../bridge/go/models";
import {
  ObtenerProductosPaginado,
  EliminarProducto,
  ActualizarProducto,
} from "@/../bridge/go/backend/Db";
import { toast } from "vue-sonner";

interface ObtenerProductosPaginadoResponse {
  Records: backend.Producto[];
  TotalRecords: number;
}

const listaProductos = ref<backend.Producto[]>([]);
const authStore = useAuthStore();
const { user: authenticatedUser } = storeToRefs(authStore);
const totalProductos = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);
const isCreateModalOpen = ref(false);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

const cargarProductos = async () => {
  try {
    const currentPage: number = pagination.value.pageIndex + 1;
    let sortBy = "";
    let sortOrder = "asc";
    if (sorting.value.length > 0) {
      sortBy = sorting.value[0]!.id;
      sortOrder = sorting.value[0]!.desc ? "desc" : "asc";
    }
    const response: ObtenerProductosPaginadoResponse =
      await ObtenerProductosPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value,
        sortBy,
        sortOrder
      );
    listaProductos.value = response.Records || [];
    totalProductos.value = response.TotalRecords || 0;
  } catch (error) {
    toast.error("Error al cargar Productos", { description: `${error}` });
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

const columns: ColumnDef<backend.Producto>[] = [
  {
    accessorKey: "Nombre",
    header: sortHeader("Nombre"),
    cell: ({ row }) =>
      h("div", { class: "uppercase max-w-[600px] truncate font-medium" }, row.getValue("Nombre")),
  },
  {
    accessorKey: "Codigo",
    header: sortHeader("Código"),
    cell: ({ row }) => h("div", { class: "font-mono text-muted-foreground" }, row.getValue("Codigo")),
  },
  {
    accessorKey: "PrecioVenta",
    header: sortHeader("Precio Venta"),
    cell: ({ row }) =>
      h("div", { class: "font-medium tabular-nums" },
        new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 0 })
          .format(parseFloat(row.getValue("PrecioVenta")))
      ),
  },
  {
    accessorKey: "Stock",
    header: sortHeader("Stock"),
    cell: ({ row }) => {
      const stock: number = row.getValue("Stock");
      return h("div", {
        class: `inline-flex items-center justify-center min-w-[2rem] px-1.5 py-0.5 rounded text-[10px] font-bold tabular-nums ${
          stock === 0 ? "bg-red-100 text-red-700"
          : stock <= 10 ? "bg-amber-100 text-amber-700"
          : "bg-emerald-50 text-emerald-700"
        }`,
      }, String(stock));
    },
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) =>
      h("div", { class: "relative" }, [
        h(DropdownAction, {
          producto: row.original,
          onEdit: (p: backend.Producto) => handleEdit(p),
          onDelete: (p: backend.Producto) => handleDelete(p),
        }),
      ]),
  },
];

const table = useVueTable({
  get data() {
    return listaProductos.value;
  },
  columns,
  manualPagination: true,
  manualSorting: true,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
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
  onPaginationChange: (updater) => valueUpdater(updater, pagination),
  onSortingChange: (updater) => valueUpdater(updater, sorting),
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (newPage) => table.setPageIndex(newPage - 1),
});

async function handleEdit(producto: backend.Producto) {
  try {
    if (!authenticatedUser.value?.UUID) {
      toast.error("Vendedor no identificado", {
        description: "Inicie sesión de nuevo.",
      });
      return;
    }
    const req = new backend.ProductoAjusteRequest({
      UUID: producto.UUID,
      Nombre: producto.Nombre,
      PrecioVenta: producto.PrecioVenta,
      Stock: producto.Stock,
      VendedorUUID: authenticatedUser.value.UUID
    })

    await ActualizarProducto(req);
    await cargarProductos();

    toast.success("Producto editado con éxito", {
      description: `Nombre: ${producto.Nombre}, Código: ${producto.Codigo}`,
    });
  } catch (error) {
    toast.error("Error al actualizar el producto", { description: `${error}` });
  }
}

async function handleDelete(producto: backend.Producto) {
  try {
    await EliminarProducto(producto.UUID);
    await cargarProductos();
    toast.warning("Producto eliminado con éxito", {
      description: `Nombre: ${producto.Nombre}, Código: ${producto.Codigo}`,
    });
  } catch (error) {
    toast.error("Error al eliminar el producto", { description: `${error}` });
  }
}

function handleProductCreated() {
  cargarProductos();
}

onMounted(cargarProductos);
watch(pagination, cargarProductos, { deep: true });
watch(
  sorting,
  () => {
    pagination.value.pageIndex = 0;
    cargarProductos();
  },
  { deep: true }
);
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    pagination.value.pageIndex = 0;
    cargarProductos();
  }, 150);
});
</script>

<template>
  <CrearProductoModal v-model:open="isCreateModalOpen" @product-created="handleProductCreated" />
  <div class="flex flex-col h-full">
    <!-- Header bar -->
    <div class="shrink-0 border-b px-4 py-3 flex items-center justify-between bg-background">
      <div>
        <p class="text-sm font-semibold">Productos</p>
        <p class="text-xs text-muted-foreground mt-0.5">Gestiona el catálogo de productos de la farmacia</p>
      </div>
      <Button size="sm" class="h-7 gap-1.5 text-xs" @click="isCreateModalOpen = true">
        <PlusCircle class="h-3.5 w-3.5" />Agregar Producto
      </Button>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-56" placeholder="Buscar por nombre o código..." />
      </div>
      <!-- Active filter/sort indicators — clarify these apply to ALL data -->
      <template v-if="busqueda">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <Search class="h-2.5 w-2.5" />
          {{ totalProductos }} resultado{{ totalProductos !== 1 ? 's' : '' }} en todo el catálogo
        </span>
      </template>
      <template v-if="sorting.length">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <component :is="sorting[0]?.desc ? ArrowDown : ArrowUp" class="h-2.5 w-2.5" />
          Ordenado por {{ sorting[0]?.id }} (todo el catálogo)
        </span>
      </template>
      <div class="ml-auto">
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
              No se encontraron productos.
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer -->
    <div class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalProductos }} producto(s) en total</span>
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
          :total="totalProductos"
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
