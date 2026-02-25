<script setup lang="ts">
import type { ColumnDef, PaginationState } from "@tanstack/vue-table";
import { FlexRender, getCoreRowModel, useVueTable } from "@tanstack/vue-table";
import { h, ref, onMounted, watch, computed } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
  Alert, AlertDescription, AlertTitle,
} from "@/components/ui/alert";
import {
  Pagination, PaginationContent, PaginationEllipsis,
  PaginationItem, PaginationNext, PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Mail, RefreshCw, ShieldCheck, ShieldOff, Eye,
  Search, Loader2, CheckCircle2, AlertCircle, FolderOpen,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { backend } from "@/../wailsjs/go/models";
import {
  EstadoAuth, IniciarOAuth2, RevocarAuth,
  SincronizarFacturas, ObtenerFacturasCompra, ObtenerDetalleFacturaCompra,
  ActualizarEstadoFacturaCompra,
} from "@/../wailsjs/go/backend/GmailService";

// ─── Types ──────────────────────────────────────────────────────────────────

interface AuthStatus {
  authenticated: boolean;
  credPresent: boolean;
  configDir: string;
}

interface SyncResult {
  total: number;
  nuevas: number;
  duplicadas: number;
  errores: string[];
}

// ─── State ──────────────────────────────────────────────────────────────────

const auth = ref<AuthStatus>({ authenticated: false, credPresent: false, configDir: "" });
const syncing = ref(false);
const syncResult = ref<SyncResult | null>(null);

const listaFacturas = ref<backend.FacturaCompra[]>([]);
const totalFacturas = ref(0);
const busqueda = ref("");
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });

const isDetailOpen = ref(false);
const detailFactura = ref<backend.FacturaCompra | null>(null);
const loadingDetailId = ref<string | null>(null);

// ─── Auth ────────────────────────────────────────────────────────────────────

const cargarEstadoAuth = async () => {
  auth.value = await EstadoAuth() as AuthStatus;
};

const conectarGmail = async () => {
  try {
    await IniciarOAuth2();
    toast.info("Navegador abierto", {
      description: "Completa la autenticación en el navegador. La app detectará el token automáticamente.",
    });
    // Poll until authenticated
    const poll = setInterval(async () => {
      await cargarEstadoAuth();
      if (auth.value.authenticated) {
        clearInterval(poll);
        toast.success("Gmail conectado correctamente");
        cargarFacturas();
      }
    }, 2000);
    setTimeout(() => clearInterval(poll), 120_000); // 2-minute timeout
  } catch (e) {
    toast.error("Error al iniciar OAuth2", { description: `${e}` });
  }
};

const desconectarGmail = async () => {
  await RevocarAuth();
  await cargarEstadoAuth();
  toast.info("Desconectado de Gmail");
};

// ─── Sync ────────────────────────────────────────────────────────────────────

const sincronizar = async () => {
  syncing.value = true;
  syncResult.value = null;
  try {
    const result = await SincronizarFacturas() as SyncResult;
    syncResult.value = result;
    if (result.nuevas > 0) {
      toast.success(`${result.nuevas} factura(s) nueva(s) importadas`);
    } else {
      toast.info("Sin facturas nuevas");
    }
    cargarFacturas();
  } catch (e) {
    toast.error("Error al sincronizar", { description: `${e}` });
  } finally {
    syncing.value = false;
  }
};

// ─── Table data ──────────────────────────────────────────────────────────────

const cargarFacturas = async () => {
  try {
    const page = pagination.value.pageIndex + 1;
    const resp = await ObtenerFacturasCompra(page, pagination.value.pageSize, busqueda.value);
    listaFacturas.value = resp.Records ?? [];
    totalFacturas.value = resp.TotalRecords ?? 0;
  } catch (e) {
    toast.error("Error al cargar facturas de compra", { description: `${e}` });
  }
};

const verDetalle = async (factura: backend.FacturaCompra) => {
  loadingDetailId.value = factura.UUID;
  try {
    detailFactura.value = await ObtenerDetalleFacturaCompra(factura.UUID);
    isDetailOpen.value = true;
  } catch (e) {
    toast.error("Error al cargar detalle", { description: `${e}` });
  } finally {
    loadingDetailId.value = null;
  }
};

const cambiarEstado = async (factura: backend.FacturaCompra, estado: string) => {
  await ActualizarEstadoFacturaCompra(factura.UUID, estado);
  await cargarFacturas();
};

const formatCOP = (v: number) =>
  new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 0 }).format(v);

const formatDate = (s: string) => {
  if (!s) return "—";
  const d = new Date(s);
  return isNaN(d.getTime()) ? s : d.toLocaleDateString("es-CO", { year: "numeric", month: "short", day: "numeric" });
};

const estadoBadge: Record<string, string> = {
  PENDIENTE: "bg-yellow-100 text-yellow-800 border-yellow-200",
  PROCESADA: "bg-green-100 text-green-800 border-green-200",
  IGNORADA:  "bg-gray-100 text-gray-600 border-gray-200",
};

// ─── Table columns ───────────────────────────────────────────────────────────

const columns: ColumnDef<backend.FacturaCompra>[] = [
  { accessorKey: "NumeroFactura", header: "N° Factura" },
  {
    accessorKey: "FechaEmision",
    header: "Fecha",
    cell: ({ row }) => formatDate(row.getValue("FechaEmision")),
  },
  { accessorKey: "ProveedorNombre", header: "Proveedor" },
  { accessorKey: "ProveedorNIT", header: "NIT" },
  {
    accessorKey: "Total",
    header: "Total",
    cell: ({ row }) => h("div", { class: "text-right font-medium" }, formatCOP(row.getValue("Total"))),
  },
  {
    accessorKey: "Estado",
    header: "Estado",
    cell: ({ row }) => {
      const estado: string = row.getValue("Estado");
      return h("span", {
        class: `inline-flex items-center px-2 py-0.5 rounded text-[10px] font-medium border ${estadoBadge[estado] ?? "bg-gray-100 text-gray-600"}`,
      }, estado);
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const isLoading = loadingDetailId.value === row.original.UUID;
      return h("div", { class: "flex items-center gap-1" }, [
        h(Button, {
          variant: "ghost", class: "h-7 w-7 p-0", title: "Ver detalle",
          disabled: isLoading,
          onClick: () => verDetalle(row.original),
        }, () => isLoading ? h(Loader2, { class: "w-3.5 h-3.5 animate-spin" }) : h(Eye, { class: "w-3.5 h-3.5" })),
      ]);
    },
  },
];

const table = useVueTable({
  get data() { return listaFacturas.value; },
  columns,
  manualPagination: true,
  getCoreRowModel: getCoreRowModel(),
  get pageCount() { return Math.ceil(totalFacturas.value / pagination.value.pageSize); },
  state: {
    get pagination() { return pagination.value; },
  },
  onPaginationChange: (updater) => valueUpdater(updater, pagination),
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (p) => table.setPageIndex(p - 1),
});

// ─── Lifecycle ───────────────────────────────────────────────────────────────

onMounted(async () => {
  await cargarEstadoAuth();
  if (auth.value.authenticated) cargarFacturas();
});

watch(pagination, cargarFacturas, { deep: true });

let debounce: number;
watch(busqueda, () => {
  clearTimeout(debounce);
  debounce = setTimeout(() => {
    pagination.value.pageIndex = 0;
    cargarFacturas();
  }, 300);
});
</script>

<template>
  <!-- Detail dialog -->
  <Dialog v-model:open="isDetailOpen">
    <DialogContent class="sm:max-w-[680px] max-h-[80vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Mail class="h-4 w-4" />
          Factura {{ detailFactura?.NumeroFactura }}
        </DialogTitle>
      </DialogHeader>
      <template v-if="detailFactura">
        <!-- Header info -->
        <div class="grid grid-cols-2 gap-3 text-xs border rounded-lg p-3 bg-muted/30">
          <div>
            <p class="text-muted-foreground">Proveedor</p>
            <p class="font-medium">{{ detailFactura.ProveedorNombre }}</p>
            <p class="text-muted-foreground">NIT: {{ detailFactura.ProveedorNIT }}</p>
          </div>
          <div>
            <p class="text-muted-foreground">Fecha</p>
            <p class="font-medium">{{ formatDate(detailFactura.FechaEmision) }}</p>
            <p class="text-muted-foreground">Moneda: {{ detailFactura.Moneda }}</p>
          </div>
          <div>
            <p class="text-muted-foreground">Subtotal</p>
            <p class="font-medium">{{ formatCOP(detailFactura.Subtotal) }}</p>
          </div>
          <div>
            <p class="text-muted-foreground">IVA</p>
            <p class="font-medium">{{ formatCOP(detailFactura.IVA) }}</p>
          </div>
          <div class="col-span-2">
            <p class="text-muted-foreground">Total Factura</p>
            <p class="text-base font-bold">{{ formatCOP(detailFactura.Total) }}</p>
          </div>
        </div>

        <!-- Estado selector -->
        <div class="flex items-center gap-2 mt-1">
          <span class="text-xs text-muted-foreground">Estado:</span>
          <Select :model-value="detailFactura.Estado" @update:model-value="(v) => { cambiarEstado(detailFactura!, v); detailFactura!.Estado = v; }">
            <SelectTrigger class="h-7 w-32 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="PENDIENTE" class="text-xs">Pendiente</SelectItem>
              <SelectItem value="PROCESADA" class="text-xs">Procesada</SelectItem>
              <SelectItem value="IGNORADA" class="text-xs">Ignorada</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Line items -->
        <div class="mt-2">
          <p class="text-xs font-medium text-muted-foreground mb-1.5">Productos ({{ detailFactura.Detalles?.length ?? 0 }})</p>
          <div class="border rounded-lg overflow-hidden">
            <table class="w-full text-xs">
              <thead class="bg-muted border-b">
                <tr>
                  <th class="h-7 px-3 text-left font-medium text-muted-foreground">Descripción</th>
                  <th class="h-7 px-3 text-left font-medium text-muted-foreground">Código</th>
                  <th class="h-7 px-3 text-right font-medium text-muted-foreground">Cant.</th>
                  <th class="h-7 px-3 text-right font-medium text-muted-foreground">P. Unit.</th>
                  <th class="h-7 px-3 text-right font-medium text-muted-foreground">Total</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="det in detailFactura.Detalles" :key="det.UUID" class="border-b last:border-b-0 hover:bg-muted/30">
                  <td class="h-8 px-3 max-w-[220px] truncate uppercase">{{ det.Descripcion }}</td>
                  <td class="h-8 px-3 font-mono text-muted-foreground">{{ det.CodigoProducto || "—" }}</td>
                  <td class="h-8 px-3 text-right">{{ det.Cantidad }}</td>
                  <td class="h-8 px-3 text-right">{{ formatCOP(det.PrecioUnitario) }}</td>
                  <td class="h-8 px-3 text-right font-medium">{{ formatCOP(det.TotalLinea) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </DialogContent>
  </Dialog>

  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="shrink-0 border-b px-4 py-3 flex items-center justify-between bg-background">
      <div>
        <p class="text-sm font-semibold flex items-center gap-2">
          <Mail class="h-4 w-4" />
          Facturas Electrónicas (DIAN)
        </p>
        <p class="text-xs text-muted-foreground mt-0.5">Facturas de compra importadas desde Gmail</p>
      </div>
      <div class="flex items-center gap-2">
        <!-- Auth status chip -->
        <span class="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border"
          :class="auth.authenticated ? 'bg-green-50 text-green-700 border-green-200' : 'bg-gray-50 text-gray-500 border-gray-200'">
          <component :is="auth.authenticated ? ShieldCheck : ShieldOff" class="h-3.5 w-3.5" />
          {{ auth.authenticated ? "Gmail conectado" : "Sin conexión" }}
        </span>
        <!-- Connect / Disconnect -->
        <Button v-if="!auth.authenticated" size="sm" class="h-7 text-xs gap-1.5" @click="conectarGmail">
          <Mail class="h-3.5 w-3.5" /> Conectar Gmail
        </Button>
        <Button v-else variant="ghost" size="sm" class="h-7 text-xs gap-1" @click="desconectarGmail">
          <ShieldOff class="h-3.5 w-3.5" /> Desconectar
        </Button>
        <!-- Sync -->
        <Button v-if="auth.authenticated" size="sm" class="h-7 text-xs gap-1.5" :disabled="syncing" @click="sincronizar">
          <Loader2 v-if="syncing" class="h-3.5 w-3.5 animate-spin" />
          <RefreshCw v-else class="h-3.5 w-3.5" />
          {{ syncing ? "Sincronizando…" : "Sincronizar" }}
        </Button>
      </div>
    </div>

    <!-- Setup alert when no credentials -->
    <div v-if="!auth.credPresent" class="shrink-0 m-4">
      <Alert>
        <FolderOpen class="h-4 w-4" />
        <AlertTitle>Configura las credenciales de Gmail</AlertTitle>
        <AlertDescription class="text-xs space-y-1">
          <p>Coloca el archivo <code class="font-mono bg-muted px-1 rounded">credentials.json</code> de Google Cloud Console en:</p>
          <p class="font-mono bg-muted px-2 py-1 rounded break-all">{{ auth.configDir }}</p>
          <p class="text-muted-foreground">El tipo de aplicación debe ser <strong>Desktop</strong> con URI de redirección <code class="font-mono">http://localhost:8094/gmail/oauth2/callback</code>.</p>
        </AlertDescription>
      </Alert>
    </div>

    <!-- Sync result banner -->
    <div v-if="syncResult" class="shrink-0 border-b px-4 py-2 flex items-center gap-4 text-xs"
      :class="syncResult.errores?.length ? 'bg-red-50' : 'bg-green-50'">
      <CheckCircle2 v-if="!syncResult.errores?.length" class="h-3.5 w-3.5 text-green-600 shrink-0" />
      <AlertCircle v-else class="h-3.5 w-3.5 text-red-600 shrink-0" />
      <span>
        <strong>{{ syncResult.total }}</strong> mensajes procesados —
        <strong class="text-green-700">{{ syncResult.nuevas }}</strong> nuevas,
        <strong>{{ syncResult.duplicadas }}</strong> duplicadas
        <span v-if="syncResult.errores?.length" class="text-red-600">, {{ syncResult.errores.length }} error(es)</span>
      </span>
      <button class="ml-auto text-muted-foreground hover:text-foreground" @click="syncResult = null">✕</button>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-64" placeholder="Buscar por proveedor, NIT o N° factura…" />
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
              <FlexRender v-if="!header.isPlaceholder" :render="header.column.columnDef.header" :props="header.getContext()" />
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
              <td v-for="cell in row.getVisibleCells()" :key="cell.id" class="h-9 px-3 border-r last:border-r-0">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </td>
            </tr>
          </template>
          <tr v-else>
            <td :colspan="columns.length + 1" class="h-32 text-center text-sm text-muted-foreground">
              {{ auth.authenticated ? "No se encontraron facturas de compra." : "Conecta Gmail para ver las facturas." }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer / pagination -->
    <div class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalFacturas }} factura(s) de compra</span>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5">
          <span class="text-xs text-muted-foreground">Filas</span>
          <Select :model-value="`${table.getState().pagination.pageSize}`"
            @update:model-value="(v) => table.setPageSize(Number(v))">
            <SelectTrigger class="h-7 w-16 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="s in [10, 25, 50]" :key="s" :value="`${s}`" class="text-xs">{{ s }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination v-if="pageCount > 1" v-model:page="currentPage"
          :total="totalFacturas" :items-per-page="pagination.pageSize" :sibling-count="1" show-edges>
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
