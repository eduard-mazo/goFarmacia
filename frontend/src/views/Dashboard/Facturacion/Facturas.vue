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
import { Card } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { ArrowUpDown, Eye, Loader2, Search, Receipt } from "lucide-vue-next";
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

const columns: ColumnDef<backend.Factura>[] = [
  { accessorKey: "NumeroFactura", header: "N° Factura" },
  {
    accessorKey: "FechaEmision",
    header: ({ column }) =>
      h("button", {
        class: "flex items-center gap-1 text-[10px] font-medium text-muted-foreground hover:text-foreground transition-colors select-none",
        onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
      }, ["Fecha", h(ArrowUpDown, { class: "h-3 w-3 opacity-50" })]),
    cell: ({ row }) => formatDate(row.getValue("FechaEmision")),
  },
  {
    accessorFn: (row) => `${row.Cliente.Nombre} ${row.Cliente.Apellido}`,
    id: "Cliente",
    header: "Cliente",
  },
  {
    accessorFn: (row) => row.Vendedor.Nombre,
    id: "Vendedor",
    header: "Vendedor",
  },
  {
    accessorKey: "Total",
    header: ({ column }) =>
      h("button", {
        class: "flex items-center gap-1 text-[10px] font-medium text-muted-foreground hover:text-foreground transition-colors select-none",
        onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
      }, ["Total", h(ArrowUpDown, { class: "h-3 w-3 opacity-50" })]),
    cell: ({ row }) =>
      h(
        "div",
        { class: "text-right font-medium" },
        formatCurrency(row.getValue("Total"))
      ),
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
        <p class="text-sm font-semibold">Historial de Facturas</p>
        <p class="text-xs text-muted-foreground mt-0.5">Consulta y revisa todas las ventas registradas</p>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-56" placeholder="Buscar por N° Factura, cliente..." />
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
              No se encontraron facturas.
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
              <SelectItem v-for="size in [10, 20, 50]" :key="size" :value="`${size}`" class="text-xs">{{ size }}</SelectItem>
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
