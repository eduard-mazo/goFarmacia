<script setup lang="ts">
import type { ColumnDef, PaginationState } from "@tanstack/vue-table";
import { FlexRender, getCoreRowModel, useVueTable } from "@tanstack/vue-table";
import { h, ref, watch, onMounted, computed, nextTick } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription,
} from "@/components/ui/dialog";
import {
  Pagination, PaginationContent, PaginationEllipsis,
  PaginationItem, PaginationNext, PaginationPrevious,
} from "@/components/ui/pagination";
import {
  PlusCircle, Search, Building2, Package, TrendingUp, Eye, Loader2,
} from "lucide-vue-next";
import CrearProveedorModal from "@/components/modals/CrearProveedorModal.vue";
import DropdownAction from "@/components/tables/DataTableProveedorDropDown.vue";
import { backend } from "@/../wailsjs/go/models";
import {
  ObtenerProveedoresPaginado, EliminarProveedor, ActualizarProveedor,
} from "@/../wailsjs/go/backend/Db";
import {
  ObtenerProveedoresConEstadisticas, ObtenerTopProductosDeProveedor,
} from "@/../wailsjs/go/backend/GmailService";
import { toast } from "vue-sonner";

// ─── Types ────────────────────────────────────────────────────────────────────
type ProveedorStats = backend.ProveedorStats;
type ProductoComprado = backend.ProductoComprado;

// ─── State ────────────────────────────────────────────────────────────────────
const listaProveedores = ref<ProveedorStats[]>([]);
const totalProveedores = ref(0);
const busqueda = ref("");
const isCreateModalOpen = ref(false);
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

// Detail panel
const isDetailOpen = ref(false);
const selectedProveedor = ref<ProveedorStats | null>(null);
const topProductos = ref<ProductoComprado[]>([]);
const loadingDetailId = ref<string | null>(null);
const loadingDetail = ref(false);

// ─── Formatting ───────────────────────────────────────────────────────────────
const formatCOP = (v: number) =>
  new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 0 }).format(v);

const formatDate = (s: string) => {
  if (!s) return "—";
  const d = new Date(s);
  return isNaN(d.getTime()) ? s : d.toLocaleDateString("es-CO", { year: "numeric", month: "short", day: "numeric" });
};

// ─── Data ─────────────────────────────────────────────────────────────────────
const cargarProveedores = async () => {
  try {
    const resp = await ObtenerProveedoresConEstadisticas(
      pagination.value.pageIndex + 1,
      pagination.value.pageSize,
      busqueda.value,
    );
    listaProveedores.value = (resp as any).Records ?? [];
    totalProveedores.value = (resp as any).TotalRecords ?? 0;
  } catch {
    // Fallback: plain list (no purchase stats yet)
    try {
      const resp = await ObtenerProveedoresPaginado(
        pagination.value.pageIndex + 1,
        pagination.value.pageSize,
        busqueda.value,
      ) as any;
      listaProveedores.value = (resp.Records ?? []).map((p: any) => ({
        ...p, TotalFacturas: 0, TotalComprado: 0, UltimaCompra: "", TopProductos: [],
      }));
      totalProveedores.value = resp.TotalRecords ?? 0;
    } catch (e) {
      toast.error("Error al cargar proveedores", { description: `${e}` });
    }
  }
};

const verDetalle = async (prov: ProveedorStats) => {
  selectedProveedor.value = prov;
  isDetailOpen.value = true;
  if (!prov.NIT) { topProductos.value = []; return; }
  loadingDetail.value = true;
  try {
    topProductos.value = (await ObtenerTopProductosDeProveedor(prov.NIT, 10)) ?? [];
  } catch {
    topProductos.value = [];
  } finally {
    loadingDetail.value = false;
  }
};

// ─── Columns ─────────────────────────────────────────────────────────────────
const columns: ColumnDef<ProveedorStats>[] = [
  {
    accessorKey: "NIT",
    header: "NIT",
    cell: ({ row }) => h("div", { class: "font-mono text-[10px] text-muted-foreground" }, row.getValue("NIT") || "—"),
  },
  {
    accessorKey: "Nombre",
    header: "Nombre",
    cell: ({ row }) => h("div", { class: "uppercase font-medium truncate max-w-[220px]" }, row.getValue("Nombre")),
  },
  {
    accessorKey: "Telefono",
    header: "Teléfono",
    cell: ({ row }) => h("div", {}, row.getValue("Telefono") || "—"),
  },
  {
    accessorKey: "Email",
    header: "Email",
    cell: ({ row }) => h("div", { class: "lowercase text-muted-foreground truncate max-w-[180px]" }, row.getValue("Email") || "—"),
  },
  {
    accessorKey: "TotalFacturas",
    header: "Facturas",
    cell: ({ row }) => {
      const n: number = row.getValue("TotalFacturas") ?? 0;
      return h("div", { class: "text-center" }, [
        h("span", {
          class: n > 0 ? "inline-flex px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-50 text-blue-700 border border-blue-200" : "text-muted-foreground/40 text-[10px]",
        }, n > 0 ? `${n}` : "—"),
      ]);
    },
  },
  {
    accessorKey: "TotalComprado",
    header: "Total comprado",
    cell: ({ row }) => {
      const v: number = row.getValue("TotalComprado") ?? 0;
      return h("div", { class: "text-right font-medium" }, v > 0 ? formatCOP(v) : "—");
    },
  },
  {
    accessorKey: "UltimaCompra",
    header: "Última compra",
    cell: ({ row }) => h("div", { class: "text-muted-foreground" }, formatDate(row.getValue("UltimaCompra"))),
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => h("div", { class: "flex items-center gap-1" }, [
      h(Button, {
        variant: "ghost", class: "h-7 w-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity",
        title: "Ver productos comprados", onClick: () => verDetalle(row.original),
      }, () => h(Eye, { class: "w-3.5 h-3.5" })),
      h(DropdownAction, {
        proveedor: row.original as any,
        onEdit: (p: any) => handleEdit(p),
        onDelete: (p: any) => handleDelete(p),
      }),
    ]),
  },
];

const table = useVueTable({
  get data() { return listaProveedores.value; },
  columns,
  manualPagination: true,
  getCoreRowModel: getCoreRowModel(),
  get pageCount() { return Math.ceil(totalProveedores.value / pagination.value.pageSize); },
  state: { get pagination() { return pagination.value; } },
  onPaginationChange: (u) => valueUpdater(u, pagination),
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (p) => table.setPageIndex(p - 1),
});

// ─── Actions ─────────────────────────────────────────────────────────────────
async function handleEdit(proveedor: backend.Proveedor) {
  try {
    await ActualizarProveedor(proveedor);
    await cargarProveedores();
    toast.success("Proveedor actualizado", { description: proveedor.Nombre });
  } catch (e) {
    toast.error("Error al actualizar", { description: `${e}` });
  }
}

async function handleDelete(proveedor: backend.Proveedor) {
  try {
    await EliminarProveedor(proveedor.uuid);
    await cargarProveedores();
    toast.warning("Proveedor eliminado", { description: proveedor.Nombre });
  } catch (e) {
    toast.error("Error al eliminar", { description: `${e}` });
  }
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────
onMounted(cargarProveedores);
watch(pagination, cargarProveedores, { deep: true });
let debounceTimer: number;
watch(busqueda, () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => { pagination.value.pageIndex = 0; cargarProveedores(); }, 200);
});
</script>

<template>
  <CrearProveedorModal v-model:open="isCreateModalOpen" @proveedor-created="cargarProveedores" />

  <!-- Detail Dialog -->
  <Dialog v-model:open="isDetailOpen">
    <DialogContent class="sm:max-w-[560px] max-h-[80vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2 text-sm">
          <Building2 class="h-4 w-4" />
          {{ selectedProveedor?.Nombre }}
        </DialogTitle>
        <DialogDescription class="text-xs text-muted-foreground font-mono">
          NIT: {{ selectedProveedor?.NIT || "—" }}
        </DialogDescription>
      </DialogHeader>

      <!-- Stats row -->
      <div class="grid grid-cols-3 gap-3 my-1">
        <div class="border rounded-lg p-3 text-center">
          <p class="text-[10px] text-muted-foreground">Facturas</p>
          <p class="text-xl font-bold">{{ selectedProveedor?.TotalFacturas ?? 0 }}</p>
        </div>
        <div class="border rounded-lg p-3 text-center col-span-2">
          <p class="text-[10px] text-muted-foreground">Total comprado</p>
          <p class="text-xl font-bold">{{ formatCOP(selectedProveedor?.TotalComprado ?? 0) }}</p>
        </div>
      </div>

      <!-- Top products -->
      <div>
        <p class="text-xs font-medium text-muted-foreground mb-1.5 flex items-center gap-1.5">
          <Package class="h-3.5 w-3.5" />
          Productos más comprados
        </p>
        <div v-if="loadingDetail" class="flex justify-center py-4">
          <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
        <div v-else-if="topProductos.length === 0" class="text-xs text-muted-foreground text-center py-4">
          Sin datos de compras registrados para este proveedor.
        </div>
        <div v-else class="border rounded-lg overflow-hidden">
          <table class="w-full text-xs">
            <thead class="bg-muted border-b">
              <tr>
                <th class="h-7 px-3 text-left font-medium text-muted-foreground">Producto</th>
                <th class="h-7 px-3 text-right font-medium text-muted-foreground">Cant.</th>
                <th class="h-7 px-3 text-right font-medium text-muted-foreground">Total</th>
                <th class="h-7 px-3 text-right font-medium text-muted-foreground"># Fact.</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(p, i) in topProductos" :key="i"
                class="border-b last:border-b-0 hover:bg-muted/30">
                <td class="h-8 px-3 uppercase truncate max-w-[260px]">{{ p.Descripcion }}</td>
                <td class="h-8 px-3 text-right tabular-nums">{{ p.TotalCantidad }}</td>
                <td class="h-8 px-3 text-right font-medium tabular-nums">{{ formatCOP(p.TotalComprado) }}</td>
                <td class="h-8 px-3 text-right text-muted-foreground">{{ p.NumFacturas }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </DialogContent>
  </Dialog>

  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="shrink-0 border-b px-4 py-3 flex items-center justify-between bg-background">
      <div>
        <p class="text-sm font-semibold flex items-center gap-2">
          <Building2 class="h-4 w-4" />Proveedores
        </p>
        <p class="text-xs text-muted-foreground mt-0.5">
          Gestiona proveedores — se actualizan automáticamente al sincronizar facturas electrónicas
        </p>
      </div>
      <Button size="sm" class="h-7 gap-1.5 text-xs" @click="isCreateModalOpen = true">
        <PlusCircle class="h-3.5 w-3.5" />Agregar Proveedor
      </Button>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-64" placeholder="Buscar por nombre, NIT o email..." />
      </div>
    </div>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <table data-supabase-table class="w-full text-xs border-collapse">
        <thead class="sticky top-0 z-10">
          <tr class="bg-muted border-b">
            <th class="h-9 w-10 px-3 text-[10px] font-normal text-muted-foreground/40 text-center border-r">#</th>
            <th v-for="header in table.getHeaderGroups()[0]?.headers" :key="header.id"
              class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-left border-r last:border-r-0 whitespace-nowrap">
              <FlexRender v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header" :props="header.getContext()" />
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-if="table.getRowModel().rows?.length">
            <tr v-for="(row, idx) in table.getRowModel().rows" :key="row.id"
              class="border-b hover:bg-muted/40 transition-colors group">
              <td class="h-9 px-3 text-center text-[10px] font-mono text-muted-foreground/40 border-r w-10 select-none">
                {{ pagination.pageIndex * pagination.pageSize + idx + 1 }}
              </td>
              <td v-for="cell in row.getVisibleCells()" :key="cell.id"
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
          <Select :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(v) => table.setPageSize(Number(v))">
            <SelectTrigger class="h-7 w-16 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="s in [10, 20, 50]" :key="s" :value="`${s}`" class="text-xs">{{ s }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination v-if="pageCount > 1" v-model:page="currentPage"
          :total="totalProveedores" :items-per-page="pagination.pageSize" :sibling-count="1" show-edges>
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious class="h-7 w-7" />
            <template v-for="(item, index) in items">
              <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                <Button class="w-7 h-7 p-0 text-xs"
                  :variant="item.value === currentPage ? 'default' : 'outline'">
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
