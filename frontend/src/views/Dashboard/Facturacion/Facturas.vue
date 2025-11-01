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
import { ArrowUpDown, Eye, Loader2 } from "lucide-vue-next";
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
      h(
        Button,
        {
          variant: "ghost",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Fecha", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
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
      h(
        Button,
        {
          variant: "ghost",
          class: "w-full flex justify-end",
          onClick: () => column.toggleSorting(column.getIsSorted() === "asc"),
        },
        () => ["Total", h(ArrowUpDown, { class: "ml-2 h-4 w-4" })]
      ),
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
          variant: "outline",
          size: "sm",
          disabled: isLoading,
          onClick: () => verDetalleFactura(row.original),
        },
        () =>
          isLoading
            ? [
              h(Loader2, { class: "w-4 h-4 mr-2 animate-spin" }),
              "Cargando...",
            ]
            : [h(Eye, { class: "w-4 h-4 mr-2" }), "Ver Factura"]
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
  <div class="w-full">
    <div class="flex items-center py-4">
      <Input class="max-w-sm h-10" placeholder="Buscar por N° Factura, cliente..." v-model="busqueda" />
    </div>
    <Card class="py-0">
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
            <TableCell :colspan="columns.length" class="h-24 text-center">No se encontraron facturas.</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </Card>
    <div class="flex items-center justify-between space-x-2 py-4">
      <div class="flex-1 text-sm text-muted-foreground">
        Total, {{ totalFacturas }} factura(s).
      </div>

      <div class="flex items-center space-x-4">
        <div class="flex items-center space-x-2">
          <p class="text-sm font-medium">Filas</p>
          <Select :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(value) => table.setPageSize(Number(value))">
            <SelectTrigger class="h-8 w-[70px]">
              <SelectValue :placeholder="`${table.getState().pagination.pageSize}`" />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="size in [5, 10, 15, 20]" :key="size" :value="`${size}`">
                {{ size }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Pagination v-if="pageCount > 1" v-model:page="currentPage" :total="totalFacturas"
          :items-per-page="pagination.pageSize" :sibling-count="1" show-edges>
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious />
            <template v-for="(item, index) in items">
              <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                <Button class="w-10 h-10 p-0" :variant="item.value === currentPage ? 'default' : 'outline'">
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
