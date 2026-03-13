<script setup lang="ts">
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/vue-table";
import { FlexRender, getCoreRowModel, useVueTable } from "@tanstack/vue-table";
import { h, ref, onMounted, onUnmounted, watch, computed, nextTick } from "vue";
import { valueUpdater } from "@/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
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
  CalendarDays, ChevronDown, FileSearch, ArrowUpDown, ArrowUp, ArrowDown, Terminal,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { EventsOn, EventsOff } from "@/../wailsjs/runtime";
import { backend } from "@/../wailsjs/go/models";
import {
  EstadoAuth, IniciarOAuth2, RevocarAuth,
  SincronizarConOpciones, EnriquecerDescripciones,
  ObtenerFacturasCompra, ObtenerDetalleFacturaCompra,
  ActualizarEstadoFacturaCompra,
  GetGmailSyncProgress,
} from "@/../wailsjs/go/backend/GmailService";

// ─── Types ───────────────────────────────────────────────────────────────────

interface AuthStatus { authenticated: boolean; credPresent: boolean; configDir: string; }
interface SyncLogEntry { nivel: string; mensaje: string; ts: string; }
interface SyncProgreso { running: boolean; fase: string; total: number; procesados: number; nuevas: number; duplicadas: number; errores: number; ultimoNro: string; }

type SyncModo = "hoy" | "semana" | "mes" | "rango" | "completo";

const MODO_LABELS: Record<SyncModo, string> = {
  hoy:      "Hoy",
  semana:   "Últimos 7 días",
  mes:      "Último mes",
  rango:    "Rango personalizado",
  completo: "Historial completo",
};

// ─── State ───────────────────────────────────────────────────────────────────

const auth = ref<AuthStatus>({ authenticated: false, credPresent: false, configDir: "" });

// Sync
const syncing = ref(false);
const enriqueciendo = ref(false);
const syncPopoverOpen = ref(false);
const syncModo = ref<SyncModo>("semana");
const syncDesde = ref("");
const syncHasta = ref("");
const syncLog = ref<SyncLogEntry[]>([]);
const syncProgreso = ref<SyncProgreso | null>(null);
const logRef = ref<HTMLElement | null>(null);

// Progress polling (recursive setTimeout — never accumulates concurrent calls)
let progressPollTimer: ReturnType<typeof setTimeout> | null = null;
let progressPollActive = false;

async function progressPollLoop() {
  if (!progressPollActive) return;
  try {
    const p = await GetGmailSyncProgress();
    if (progressPollActive) {
      syncProgreso.value = p as SyncProgreso;
    }
  } catch { /* ignore */ }
  if (progressPollActive) {
    progressPollTimer = setTimeout(progressPollLoop, 1000);
  }
}

function startProgressPoll() {
  progressPollActive = true;
  progressPollLoop();
}

function stopProgressPoll() {
  progressPollActive = false;
  if (progressPollTimer !== null) {
    clearTimeout(progressPollTimer);
    progressPollTimer = null;
  }
}

const progressPct = computed(() => {
  const p = syncProgreso.value;
  if (!p || p.total === 0) return 0;
  if (p.fase === "recolectando") return 5; // indeterminate-ish while collecting
  return Math.round((p.procesados / p.total) * 100);
});

// Table
const listaFacturas = ref<backend.FacturaCompra[]>([]);
const totalFacturas = ref(0);
const busqueda = ref("");
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize: 10 });
const sorting = ref<SortingState>([]);

// Console
const showConsole = ref(false);

// Detail dialog
const isDetailOpen = ref(false);
const detailFactura = ref<backend.FacturaCompra | null>(null);
const loadingDetailId = ref<string | null>(null);

// ─── Auth ─────────────────────────────────────────────────────────────────────

const cargarEstadoAuth = async () => {
  auth.value = await EstadoAuth() as AuthStatus;
};

const conectarGmail = async () => {
  try {
    await IniciarOAuth2();
    toast.info("Navegador abierto", {
      description: "Autentica en el navegador. La app detectará el token automáticamente.",
    });
    const poll = setInterval(async () => {
      await cargarEstadoAuth();
      if (auth.value.authenticated) {
        clearInterval(poll);
        toast.success("Gmail conectado correctamente");
        cargarFacturas();
      }
    }, 2000);
    setTimeout(() => clearInterval(poll), 120_000);
  } catch (e) {
    toast.error("Error al iniciar OAuth2", { description: `${e}` });
  }
};

const desconectarGmail = async () => {
  await RevocarAuth();
  await cargarEstadoAuth();
  toast.info("Desconectado de Gmail");
};

// ─── Sync ─────────────────────────────────────────────────────────────────────

const scrollLog = () => {
  nextTick(() => {
    if (logRef.value) logRef.value.scrollTop = logRef.value.scrollHeight;
  });
};

const iniciarSync = () => {
  if (syncing.value) return;
  syncPopoverOpen.value = false;
  syncing.value = true;
  showConsole.value = true;
  syncLog.value = [];
  syncProgreso.value = null;
  // Start polling in-memory progress before the goroutine begins.
  startProgressPoll();
  // Fire-and-forget: runs in goroutine on backend, result via "gmail:sync:result" event.
  SincronizarConOpciones({
    modo: syncModo.value,
    desde: syncModo.value === "rango" ? syncDesde.value : "",
    hasta: syncModo.value === "rango" ? syncHasta.value : "",
  });
};

const enriquecerDesdesPDF = async () => {
  if (enriqueciendo.value || syncing.value) return;
  enriqueciendo.value = true;
  syncLog.value = [];
  try {
    const res = await EnriquecerDescripciones();
    if (res.Enriquecidas > 0) {
      toast.success(`${res.Enriquecidas} descripción(es) actualizadas desde PDF`);
      cargarFacturas();
    } else {
      toast.info("Sin descripciones pendientes de enriquecimiento");
    }
  } catch (e) {
    toast.error("Error al enriquecer desde PDF", { description: `${e}` });
  } finally {
    enriqueciendo.value = false;
  }
};

// ─── Table ────────────────────────────────────────────────────────────────────

const cargarFacturas = async () => {
  try {
    const s = sorting.value[0];
    const sortField = s?.id ?? "";
    const sortDir = s?.desc === false ? "asc" : "desc";
    const resp = await ObtenerFacturasCompra(pagination.value.pageIndex + 1, pagination.value.pageSize, busqueda.value, sortField, sortDir);
    listaFacturas.value = resp.Records ?? [];
    totalFacturas.value = resp.TotalRecords ?? 0;
  } catch (e) {
    toast.error("Error al cargar facturas", { description: `${e}` });
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
  cargarFacturas();
};

// ─── Formatting ───────────────────────────────────────────────────────────────

const formatCOP = (v: number) =>
  new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 0 }).format(v);

const formatDate = (s: string) => {
  if (!s) return "—";
  const d = new Date(s);
  return isNaN(d.getTime()) ? s : d.toLocaleDateString("es-CO", { year: "numeric", month: "short", day: "numeric" });
};

const estadoClass: Record<string, string> = {
  PENDIENTE: "bg-yellow-100 text-yellow-800 border-yellow-200",
  PROCESADA: "bg-green-100 text-green-800 border-green-200",
  IGNORADA:  "bg-gray-100 text-gray-500 border-gray-200",
};

const logClass: Record<string, string> = {
  ok:    "text-green-400",
  info:  "text-slate-300",
  warn:  "text-yellow-400",
  error: "text-red-400",
};

// ─── Columns ──────────────────────────────────────────────────────────────────

// Document type metadata for DIAN UBL 2.1 classification
const tipoDocConfig: Record<string, { label: string; class: string }> = {
  "01": { label: "Factura",      class: "bg-blue-50 text-blue-700 border-blue-200" },
  "02": { label: "Fac. Export.", class: "bg-cyan-50 text-cyan-700 border-cyan-200" },
  "91": { label: "Nota Crédito", class: "bg-amber-50 text-amber-700 border-amber-200" },
  "92": { label: "Nota Débito",  class: "bg-orange-50 text-orange-700 border-orange-200" },
};

function sortHeader(label: string) {
  return ({ column }: { column: any }) => h("button", {
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

const columns: ColumnDef<backend.FacturaCompra>[] = [
  {
    id: "TipoDocumento",
    accessorKey: "TipoDocumento",
    header: "Tipo",
    cell: ({ row }) => {
      const tipo: string = row.getValue("TipoDocumento") || "01";
      const cfg = tipoDocConfig[tipo] ?? { label: tipo, class: "bg-gray-100 text-gray-600 border-gray-200" };
      return h("span", {
        class: `inline-flex px-2 py-0.5 rounded text-[10px] font-medium border whitespace-nowrap ${cfg.class}`,
      }, cfg.label);
    },
  },
  { accessorKey: "NumeroFactura", header: sortHeader("N° Documento") },
  {
    accessorKey: "FechaEmision",
    header: sortHeader("Fecha"),
    cell: ({ row }) => formatDate(row.getValue("FechaEmision")),
  },
  { accessorKey: "ProveedorNombre", header: sortHeader("Proveedor") },
  { accessorKey: "ProveedorNIT", header: "NIT" },
  {
    accessorKey: "Total",
    header: sortHeader("Total"),
    cell: ({ row }) => {
      const tipo: string = row.original.TipoDocumento || "01";
      const isNote91 = tipo === "91";
      const val = row.getValue<number>("Total");
      return h("div", {
        class: `text-right font-medium ${isNote91 ? "text-amber-600" : ""}`,
        title: isNote91 ? "Nota crédito — reduce el total de compras" : "",
      }, isNote91 ? `(${formatCOP(val)})` : formatCOP(val));
    },
  },
  {
    accessorKey: "Estado",
    header: sortHeader("Estado"),
    cell: ({ row }) => {
      const e: string = row.getValue("Estado");
      return h("span", {
        class: `inline-flex px-2 py-0.5 rounded text-[10px] font-medium border ${estadoClass[e] ?? "bg-gray-100 text-gray-600"}`,
      }, e);
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const loading = loadingDetailId.value === row.original.UUID;
      return h(Button, {
        variant: "ghost", class: "h-7 w-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity",
        title: "Ver detalle", disabled: loading,
        onClick: () => verDetalle(row.original),
      }, () => loading
        ? h(Loader2, { class: "w-3.5 h-3.5 animate-spin" })
        : h(Eye, { class: "w-3.5 h-3.5" }));
    },
  },
];

const table = useVueTable({
  get data() { return listaFacturas.value; },
  columns,
  manualPagination: true,
  manualSorting: true,
  getCoreRowModel: getCoreRowModel(),
  get pageCount() { return Math.ceil(totalFacturas.value / pagination.value.pageSize); },
  state: {
    get pagination() { return pagination.value; },
    get sorting() { return sorting.value; },
  },
  onPaginationChange: (u) => valueUpdater(u, pagination),
  onSortingChange: (u) => { valueUpdater(u, sorting); pagination.value.pageIndex = 0; },
  enableMultiSort: false,
});

const pageCount = computed(() => table.getPageCount());
const currentPage = computed({
  get: () => pagination.value.pageIndex + 1,
  set: (p) => table.setPageIndex(p - 1),
});

// ─── Lifecycle ────────────────────────────────────────────────────────────────

onMounted(async () => {
  await cargarEstadoAuth();
  if (auth.value.authenticated) cargarFacturas();

  EventsOn("gmail:sync:result", async (res: { nuevas: number; total: number; duplicadas: number; errores: string[]; log: SyncLogEntry[] }) => {
    stopProgressPoll();
    syncing.value = false;
    if (res.log?.length) {
      syncLog.value.push(...res.log);
      scrollLog();
    }
    syncProgreso.value = {
      running: false,
      fase: "",
      total: res.total ?? 0,
      procesados: res.total ?? 0,
      nuevas: res.nuevas ?? 0,
      duplicadas: res.duplicadas ?? 0,
      errores: res.errores?.length ?? 0,
      ultimoNro: "",
    };
    if (res.nuevas > 0) {
      await cargarFacturas();
    }
  });
});

onUnmounted(() => {
  stopProgressPoll();
  EventsOff("gmail:sync:result");
});

watch(pagination, cargarFacturas, { deep: true });
watch(sorting, cargarFacturas, { deep: true });
let debounce: number;
watch(busqueda, () => {
  clearTimeout(debounce);
  debounce = setTimeout(() => { pagination.value.pageIndex = 0; cargarFacturas(); }, 300);
});
</script>

<template>
  <!-- Detail dialog -->
  <Dialog v-model:open="isDetailOpen">
    <DialogContent class="sm:max-w-[680px] max-h-[80vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Mail class="h-4 w-4" />
          {{ tipoDocConfig[detailFactura?.TipoDocumento || '01']?.label ?? 'Documento' }}
          {{ detailFactura?.NumeroFactura }}
        </DialogTitle>
      </DialogHeader>
      <template v-if="detailFactura">
        <!-- Document type banner for notes -->
        <div
          v-if="detailFactura.TipoDocumento === '91' || detailFactura.TipoDocumento === '92'"
          class="flex items-center gap-2 rounded-md px-3 py-2 text-xs"
          :class="detailFactura.TipoDocumento === '91'
            ? 'bg-amber-50 text-amber-700 border border-amber-200'
            : 'bg-orange-50 text-orange-700 border border-orange-200'"
        >
          <span class="font-semibold">
            {{ tipoDocConfig[detailFactura.TipoDocumento]?.label }}
          </span>
          <span v-if="detailFactura.ReferenciaDocumento">
            — referencia a factura: <span class="font-mono font-medium">{{ detailFactura.ReferenciaDocumento }}</span>
          </span>
        </div>

        <div class="grid grid-cols-2 gap-3 text-xs border rounded-lg p-3 bg-muted/30">
          <div>
            <p class="text-muted-foreground">Proveedor</p>
            <p class="font-medium">{{ detailFactura.ProveedorNombre }}</p>
            <p class="text-muted-foreground font-mono">NIT: {{ detailFactura.ProveedorNIT }}</p>
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
          <div class="col-span-2 border-t pt-2">
            <p class="text-muted-foreground text-[10px]">
              {{ detailFactura.TipoDocumento === '91' ? 'TOTAL NOTA CRÉDITO' : detailFactura.TipoDocumento === '92' ? 'TOTAL NOTA DÉBITO' : 'TOTAL FACTURA' }}
            </p>
            <p class="text-lg font-bold"
               :class="detailFactura.TipoDocumento === '91' ? 'text-amber-600' : ''">
              {{ detailFactura.TipoDocumento === '91' ? `(${formatCOP(detailFactura.Total)})` : formatCOP(detailFactura.Total) }}
            </p>
          </div>
        </div>

        <!-- Estado selector -->
        <div class="flex items-center gap-2 mt-1">
          <span class="text-xs text-muted-foreground">Estado:</span>
          <Select
            :model-value="detailFactura.Estado"
            @update:model-value="(v) => { cambiarEstado(detailFactura!, v); detailFactura!.Estado = v; }">
            <SelectTrigger class="h-7 w-36 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="PENDIENTE" class="text-xs">Pendiente</SelectItem>
              <SelectItem value="PROCESADA" class="text-xs">Procesada</SelectItem>
              <SelectItem value="IGNORADA"  class="text-xs">Ignorada</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Line items -->
        <div class="mt-2">
          <p class="text-xs font-medium text-muted-foreground mb-1.5">
            Productos ({{ detailFactura.Detalles?.length ?? 0 }})
          </p>
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
                <tr v-for="det in detailFactura.Detalles" :key="det.UUID"
                  class="border-b last:border-b-0 hover:bg-muted/30">
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
        <!-- Auth chip -->
        <span class="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border"
          :class="auth.authenticated
            ? 'bg-green-50 text-green-700 border-green-200'
            : 'bg-gray-50 text-gray-500 border-gray-200'">
          <component :is="auth.authenticated ? ShieldCheck : ShieldOff" class="h-3.5 w-3.5" />
          {{ auth.authenticated ? "Gmail conectado" : "Sin conexión" }}
        </span>

        <Button v-if="!auth.authenticated" size="sm" class="h-7 text-xs gap-1.5" @click="conectarGmail">
          <Mail class="h-3.5 w-3.5" />Conectar Gmail
        </Button>

        <Button v-else variant="ghost" size="sm" class="h-7 text-xs gap-1" @click="desconectarGmail">
          <ShieldOff class="h-3.5 w-3.5" />Desconectar
        </Button>

        <!-- Sync popover -->
        <Popover v-if="auth.authenticated" v-model:open="syncPopoverOpen">
          <PopoverTrigger as-child>
            <Button size="sm" class="h-7 text-xs gap-1.5" :disabled="syncing">
              <Loader2 v-if="syncing" class="h-3.5 w-3.5 animate-spin" />
              <RefreshCw v-else class="h-3.5 w-3.5" />
              {{ syncing ? "Sincronizando…" : "Sincronizar" }}
              <ChevronDown v-if="!syncing" class="h-3 w-3 opacity-60" />
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" class="w-72 p-0">
            <div class="border-b px-3 py-2">
              <p class="text-xs font-medium">Opciones de sincronización</p>
              <p class="text-[10px] text-muted-foreground mt-0.5">Elige el rango de correos a revisar</p>
            </div>
            <div class="p-3 space-y-2">
              <!-- Mode selector -->
              <div class="space-y-1">
                <label class="text-[10px] text-muted-foreground font-medium uppercase tracking-wide">Período</label>
                <div class="grid grid-cols-1 gap-1">
                  <button
                    v-for="(label, modo) in MODO_LABELS" :key="modo"
                    @click="syncModo = modo as SyncModo"
                    class="flex items-center gap-2 px-2.5 py-1.5 rounded text-xs transition-colors text-left"
                    :class="syncModo === modo
                      ? 'bg-primary text-primary-foreground'
                      : 'hover:bg-muted text-foreground'">
                    <CalendarDays class="h-3.5 w-3.5 shrink-0" />
                    {{ label }}
                  </button>
                </div>
              </div>

              <!-- Custom range inputs -->
              <template v-if="syncModo === 'rango'">
                <div class="grid grid-cols-2 gap-2">
                  <div class="space-y-1">
                    <label class="text-[10px] text-muted-foreground">Desde</label>
                    <Input v-model="syncDesde" type="date" class="h-7 text-xs" />
                  </div>
                  <div class="space-y-1">
                    <label class="text-[10px] text-muted-foreground">Hasta</label>
                    <Input v-model="syncHasta" type="date" class="h-7 text-xs" />
                  </div>
                </div>
              </template>

              <!-- Warning for completo -->
              <div v-if="syncModo === 'completo'"
                class="flex items-start gap-1.5 text-[10px] text-amber-700 bg-amber-50 border border-amber-200 rounded px-2 py-1.5">
                <AlertCircle class="h-3 w-3 shrink-0 mt-px" />
                Revisará todo el historial de Gmail. Puede tardar varios minutos.
              </div>

              <Button class="w-full h-7 text-xs" @click="iniciarSync">
                <RefreshCw class="h-3.5 w-3.5 mr-1.5" />
                Iniciar — {{ MODO_LABELS[syncModo] }}
              </Button>
            </div>
          </PopoverContent>
        </Popover>

        <!-- Enrich from PDF -->
        <Button
          v-if="auth.authenticated"
          size="sm"
          variant="outline"
          class="h-7 text-xs gap-1.5"
          :disabled="enriqueciendo || syncing"
          @click="enriquecerDesdesPDF"
        >
          <Loader2 v-if="enriqueciendo" class="h-3.5 w-3.5 animate-spin" />
          <FileSearch v-else class="h-3.5 w-3.5" />
          {{ enriqueciendo ? "Enriqueciendo…" : "Enriquecer PDF" }}
        </Button>

        <!-- Terminal / console toggle -->
        <Button
          v-if="auth.authenticated"
          size="sm"
          variant="outline"
          class="h-7 text-xs gap-1.5"
          :class="showConsole ? 'border-slate-700 bg-slate-900 text-slate-100 hover:bg-slate-800' : ''"
          @click="showConsole = !showConsole"
        >
          <Terminal class="h-3.5 w-3.5" />
          Consola
        </Button>
      </div>
    </div>

    <!-- Credentials alert -->
    <div v-if="!auth.credPresent" class="shrink-0 m-4">
      <Alert>
        <FolderOpen class="h-4 w-4" />
        <AlertTitle>Configura las credenciales de Gmail</AlertTitle>
        <AlertDescription class="text-xs space-y-1">
          <p>Coloca <code class="font-mono bg-muted px-1 rounded">credentials.json</code> en:</p>
          <p class="font-mono bg-muted px-2 py-1 rounded break-all select-all">{{ auth.configDir }}</p>
          <p class="text-muted-foreground">Tipo de app: <strong>Desktop</strong>. URI de redirección:
            <code class="font-mono">http://localhost:8094/gmail/oauth2/callback</code></p>
        </AlertDescription>
      </Alert>
    </div>

    <!-- Real-time sync console -->
    <div v-if="showConsole" class="shrink-0 border-b bg-slate-950 text-slate-200">

      <!-- Progress bar -->
      <div class="h-1 bg-slate-800 relative overflow-hidden">
        <div
          v-if="syncing"
          class="h-full bg-blue-500 transition-all duration-700"
          :class="syncProgreso?.fase === 'recolectando' ? 'animate-pulse' : ''"
          :style="{ width: progressPct + '%' }"
        />
        <div v-else-if="syncProgreso" class="h-full bg-green-500 w-full" />
      </div>

      <!-- Status row -->
      <div class="flex items-center gap-3 px-3 py-1.5 border-b border-slate-800 text-xs">
        <Loader2 v-if="syncing || enriqueciendo" class="h-3 w-3 animate-spin text-blue-400 shrink-0" />
        <CheckCircle2 v-else class="h-3 w-3 text-green-400 shrink-0" />

        <template v-if="syncing && syncProgreso">
          <!-- Phase: collecting IDs -->
          <template v-if="syncProgreso.fase === 'recolectando'">
            <span class="text-slate-400 font-mono">Recolectando mensajes…</span>
            <span class="font-mono text-blue-400">{{ syncProgreso.total }} encontrados</span>
          </template>
          <!-- Phase: processing -->
          <template v-else-if="syncProgreso.fase === 'procesando'">
            <span class="font-mono text-blue-300">
              {{ syncProgreso.procesados }}<span class="text-slate-600">/</span>{{ syncProgreso.total }}
            </span>
            <span class="font-mono text-green-400">✓ {{ syncProgreso.nuevas }} nuevas</span>
            <span class="font-mono text-slate-500">= {{ syncProgreso.duplicadas }} dup.</span>
            <span v-if="syncProgreso.errores" class="font-mono text-red-400">✗ {{ syncProgreso.errores }}</span>
            <span v-if="syncProgreso.ultimoNro" class="font-mono text-slate-500 truncate max-w-[160px]">
              {{ syncProgreso.ultimoNro }}
            </span>
          </template>
          <!-- Waiting for first poll result -->
          <template v-else>
            <span class="text-slate-400 font-mono">Iniciando sincronización…</span>
          </template>
        </template>

        <!-- Completed or enriching -->
        <template v-else-if="!syncing">
          <span class="text-slate-400 font-mono">
            {{ enriqueciendo ? "Enriqueciendo desde PDF…" : "Completado" }}
          </span>
          <template v-if="syncProgreso && !enriqueciendo">
            <span class="font-mono text-slate-300">{{ syncProgreso.total }} revisados</span>
            <span class="font-mono text-green-400">✓ {{ syncProgreso.nuevas }} nuevas</span>
            <span class="font-mono text-slate-500">= {{ syncProgreso.duplicadas }} dup.</span>
            <span v-if="syncProgreso.errores" class="font-mono text-red-400">✗ {{ syncProgreso.errores }} errores</span>
          </template>
        </template>

        <button v-if="!syncing && !enriqueciendo" class="ml-auto text-slate-500 hover:text-slate-300 text-[10px]"
          @click="syncLog = []; syncProgreso = null; showConsole = false">Cerrar</button>
      </div>

      <!-- Log lines (filled at end from gmail:sync:result) -->
      <div ref="logRef" class="overflow-y-auto max-h-36 px-3 py-1.5 font-mono text-[10px] leading-5 space-y-px">
        <div v-if="syncing && !syncLog.length" class="flex gap-2">
          <span class="text-slate-600 shrink-0 invisible">00:00:00</span>
          <span class="text-slate-500 animate-pulse">▋</span>
        </div>
        <div v-for="(entry, i) in syncLog" :key="i" class="flex gap-2">
          <span class="text-slate-600 shrink-0">{{ entry.ts }}</span>
          <span :class="logClass[entry.nivel] ?? 'text-slate-300'">{{ entry.mensaje }}</span>
        </div>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <div class="relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50 pointer-events-none" />
        <Input v-model="busqueda" class="pl-7 h-7 text-xs w-64"
          placeholder="Buscar por proveedor, NIT o N° factura…" />
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
            <td :colspan="columns.length + 1"
              class="h-32 text-center text-sm text-muted-foreground">
              {{ auth.authenticated
                ? "No se encontraron facturas de compra."
                : "Conecta Gmail para ver las facturas." }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer -->
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
