<script setup lang="ts">
import type {
  ColumnDef,
  PaginationState,
  SortingState,
} from "@tanstack/vue-table";
import { FlexRender, getCoreRowModel, useVueTable } from "@tanstack/vue-table";
import { h, ref, onMounted, watch, computed } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { ArrowUpDown, ArrowUp, ArrowDown, Eye, Loader2, Search, Receipt, TrendingUp } from "lucide-vue-next";
import { backend } from "@/../wailsjs/go/models";
import {
  ObtenerFacturasPaginado,
  ObtenerDetalleFactura,
} from "@/../wailsjs/go/backend/Db";
import { toast } from "vue-sonner";
import ReciboVentaModal from "@/components/modals/ReciboVentaModal.vue";

interface ObtenerFacturasPaginadoResponse {
  Records: backend.Factura[];
  TotalRecords: number;
}

const listaFacturas = ref<backend.Factura[]>([]);
const totalFacturas = ref(0);
const busqueda = ref("");
const sorting = ref<SortingState>([]);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });
const isModalOpen = ref(false);
const loadingFacturaId = ref<string | null>(null);
const facturaParaRecibo = ref<backend.Factura | null>(new backend.Factura());

const cargarFacturas = async () => {
  try {
    const currentPage = pagination.value.pageIndex + 1;
    let sortBy = "";
    let sortOrder = "asc";
    if (sorting.value.length > 0) {
      sortBy = sorting.value[0]!.id;
      sortOrder = sorting.value[0]!.desc ? "desc" : "asc";
    }
    const response: ObtenerFacturasPaginadoResponse =
      await ObtenerFacturasPaginado(
        currentPage,
        pagination.value.pageSize,
        busqueda.value,
        sortBy,
        sortOrder
      );
    listaFacturas.value = response.Records || [];
    totalFacturas.value = response.TotalRecords || 0;
  } catch (error) {
    toast.error("Error al cargar facturas", { description: `${error}` });
  }
};

const totalRevenue = computed(() =>
  listaFacturas.value.reduce((acc, f) => acc + (f.Total ?? 0), 0)
);

const formatCurrency = (value: number) =>
  new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    maximumFractionDigits: 0,
  }).format(value);
const formatDate = (dateString: string) => {
  if (!dateString) return "---";
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return "Fecha inválida";
  return date.toLocaleDateString("es-CO", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
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

const columns: ColumnDef<backend.Factura>[] = [
  {
    accessorKey: "NumeroFactura",
    header: sortHeader("N° Factura"),
    cell: ({ row }) => h("div", { class: "font-mono text-muted-foreground" }, row.getValue("NumeroFactura")),
  },
  {
    accessorKey: "FechaEmision",
    header: sortHeader("Fecha"),
    cell: ({ row }) => h("div", { class: "text-muted-foreground" }, formatDate(row.getValue("FechaEmision"))),
  },
  {
    accessorFn: (row) => `${row.Cliente.Nombre} ${row.Cliente.Apellido}`,
    id: "Cliente",
    header: "Cliente",
    cell: ({ row }) => h("div", { class: "uppercase font-medium truncate max-w-[200px]" },
      `${row.original.Cliente.Nombre} ${row.original.Cliente.Apellido}`),
  },
  {
    accessorFn: (row) => row.Vendedor.Nombre,
    id: "Vendedor",
    header: "Vendedor",
    cell: ({ row }) => h("div", { class: "text-muted-foreground" }, row.original.Vendedor.Nombre),
  },
  {
    accessorKey: "Total",
    header: sortHeader("Total"),
    cell: ({ row }) =>
      h("div", { class: "font-medium tabular-nums" }, formatCurrency(row.getValue("Total"))),
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const isLoading = loadingFacturaId.value === row.original.UUID;
      return h(
        Button,
        {
          variant: "ghost",
          class: "h-7 w-7 p-0",
          disabled: isLoading,
          title: "Ver factura",
          onClick: () => verDetalleFactura(row.original),
        },
        () =>
          isLoading
            ? h(Loader2, { class: "w-3.5 h-3.5 animate-spin" })
            : h(Eye, { class: "w-3.5 h-3.5" })
      );
    },
  },
];

const table = useVueTable({
  get data() {
    return listaFacturas.value;
  },
  columns,
  manualPagination: true,
  manualSorting: true,
  getCoreRowModel: getCoreRowModel(),
  get pageCount() {
    return Math.ceil(totalFacturas.value / pagination.value.pageSize);
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
async function verDetalleFactura(factura: backend.Factura) {
  loadingFacturaId.value = factura.UUID;
  try {
    const facturaCompleta = await ObtenerDetalleFactura(factura.UUID);
    facturaParaRecibo.value = facturaCompleta;
    isModalOpen.value = true;
  } catch (error) {
    toast.error("Error al cargar detalles", { description: `${error}` });
  } finally {
    loadingFacturaId.value = null;
  }
}

onMounted(cargarFacturas);
watch(pagination, cargarFacturas, { deep: true });
watch(
  sorting,
  () => {
    pagination.value.pageIndex = 0;
    cargarFacturas();
  },
  { deep: true }
);
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    pagination.value.pageIndex = 0;
    cargarFacturas();
  }, 300);
});
</script>

<template>
  <ReciboVentaModal :factura="facturaParaRecibo" @update:open="facturaParaRecibo = null" />
  <div class="flex flex-col h-full">
    <!-- Header bar -->
    <div class="shrink-0 border-b px-4 py-3 flex items-center justify-between bg-background">
      <div>
        <p class="text-sm font-semibold flex items-center gap-2">
          <Receipt class="h-4 w-4" />
          Historial de Facturas
        </p>
        <p class="text-xs text-muted-foreground mt-0.5">Consulta y revisa todas las ventas registradas</p>
      </div>
    </div>

    <!-- KPI strip (mirrors Bancolombia's stats bar) -->
    <div class="shrink-0 border-b px-4 py-2 grid grid-cols-3 gap-4 bg-muted/10">
      <div>
        <p class="text-[10px] text-muted-foreground font-medium uppercase tracking-wide">Total facturas</p>
        <p class="text-xl font-bold mt-0.5">{{ totalFacturas }}</p>
      </div>
      <div>
        <p class="text-[10px] text-muted-foreground font-medium uppercase tracking-wide">En esta página</p>
        <p class="text-xl font-bold mt-0.5">{{ listaFacturas.length }}</p>
      </div>
      <div>
        <p class="text-[10px] text-muted-foreground font-medium uppercase tracking-wide">Subtotal visible</p>
        <p class="text-lg font-bold mt-0.5 text-emerald-600">{{ formatCurrency(totalRevenue) }}</p>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-56" placeholder="Buscar por N° Factura, cliente..." />
      </div>
      <template v-if="busqueda">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <Search class="h-2.5 w-2.5" />
          {{ totalFacturas }} resultado{{ totalFacturas !== 1 ? 's' : '' }} en todo el historial
        </span>
      </template>
      <template v-if="sorting.length">
        <span class="text-[10px] text-muted-foreground border border-dashed rounded px-2 py-0.5 flex items-center gap-1">
          <component :is="sorting[0]?.desc ? ArrowDown : ArrowUp" class="h-2.5 w-2.5" />
          Ordenado por {{ sorting[0]?.id }} (todo el historial)
        </span>
      </template>
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
            <td :colspan="columns.length + 1" class="h-40 text-center">
              <div class="flex flex-col items-center justify-center gap-2 text-muted-foreground">
                <Receipt class="h-8 w-8 opacity-20" />
                <span class="text-xs">No se encontraron facturas</span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer -->
    <div class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalFacturas }} factura(s) en total</span>
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
              <SelectItem v-for="size in [10, 25, 50]" :key="size" :value="`${size}`" class="text-xs">{{ size }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination
          v-if="pageCount > 1"
          v-model:page="currentPage"
          :total="totalFacturas"
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
