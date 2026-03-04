<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from "vue";
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
  Landmark, RefreshCw, ShieldCheck, ShieldOff, Loader2,
  CheckCircle2, AlertCircle, ArrowDownLeft, Eye, Mail,
  ChevronDown, CalendarDays, Trash2, FolderOpen, Zap, Key,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { EventsOn, EventsOff, BrowserOpenURL } from "@/../wailsjs/runtime";
import { backend } from "@/../wailsjs/go/models";
import {
  EstadoAuth, IniciarOAuth2, RevocarAuth,
  ObtenerTransferencias, MarcarLeida, EliminarTransferencia,
  SincronizarConPeriodo, SetAutoPolling, GetAutoPolling,
  MarcarTodasLeidas,
} from "@/../wailsjs/go/backend/BancolombiaService";

// ─── Types ────────────────────────────────────────────────────────────────────

type AuthStatus = backend.BancolombiaAuthStatus;
type Transferencia = backend.TransferenciaBancolombia;
type CheckResult = backend.BancolombiaCheckResult;
type LogEntry = backend.BancolombiaLogEntry;

type SyncModo = "hoy" | "semana" | "mes" | "rango" | "completo";

const MODO_LABELS: Record<SyncModo, string> = {
  hoy:      "Hoy",
  semana:   "Últimos 7 días",
  mes:      "Último mes",
  rango:    "Rango personalizado",
  completo: "Historial completo",
};

const logClass: Record<string, string> = {
  ok:    "text-green-400",
  info:  "text-slate-300",
  warn:  "text-yellow-400",
  error: "text-red-400",
};

// ─── State ────────────────────────────────────────────────────────────────────

const auth = ref<AuthStatus>({ authenticated: false, credPresent: false, configDir: "" });

const items = ref<Transferencia[]>([]);
const totalItems = ref(0);
const totalMonto = ref(0);
const page = ref(1);
const pageSize = ref(25);
const soloNoLeidas = ref(false);

const loading = ref(false);
const syncing = ref(false);
const authenticating = ref(false);
const deletingUUID = ref<string | null>(null);
const markingAllRead = ref(false);

const lastCheck = ref<CheckResult | null>(null);
const syncPopoverOpen = ref(false);
const syncModo = ref<SyncModo>("semana");
const syncDesde = ref("");
const syncHasta = ref("");
const autoPolling = ref(true);

// Sync log (like FacturasElectronicas)
const syncLog = ref<LogEntry[]>([]);
const logRef = ref<HTMLElement | null>(null);

const selectedItem = ref<Transferencia | null>(null);
const showDetail = ref(false);
const confirmDeleteUUID = ref<string | null>(null);

// ─── Computed ─────────────────────────────────────────────────────────────────

const totalPages = computed(() => Math.ceil(totalItems.value / pageSize.value));

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatCOP(v: number): string {
  return new Intl.NumberFormat("es-CO", {
    style: "currency", currency: "COP",
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  }).format(v);
}

function formatDate(d: string | Date): string {
  if (!d) return "—";
  return new Date(d).toLocaleString("es-CO", {
    day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

/** Returns false if the rawSubject looks like an HTML artifact (URL, "Logo", etc.) */
function isCleanSubject(s: string): boolean {
  if (!s) return false;
  const lower = s.toLowerCase();
  return !lower.startsWith("logo") && !s.includes("http") && !s.includes("[https");
}

function transferTipoIcon(concepto: string) {
  const lower = (concepto ?? "").toLowerCase();
  if (lower.includes("llave")) return Key;
  if (lower.includes("qr")) return Zap;
  return ArrowDownLeft;
}

function scrollLog() {
  nextTick(() => {
    if (logRef.value) logRef.value.scrollTop = logRef.value.scrollHeight;
  });
}

// ─── Load data ────────────────────────────────────────────────────────────────

async function loadAuth() {
  auth.value = await EstadoAuth();
}

async function fetchTransferencias() {
  loading.value = true;
  try {
    const res = await ObtenerTransferencias(page.value, pageSize.value, soloNoLeidas.value);
    items.value = res.items ?? [];
    totalItems.value = res.total;
    totalMonto.value = res.totalMonto;
  } catch (e: any) {
    toast.error("Error al cargar transferencias", { description: e?.toString() });
  } finally {
    loading.value = false;
  }
}

async function loadPollingState() {
  try {
    const s = await GetAutoPolling();
    autoPolling.value = s.enabled;
  } catch (_) {}
}

// ─── Actions ──────────────────────────────────────────────────────────────────

async function iniciarSync() {
  if (syncing.value) return;
  syncPopoverOpen.value = false;
  syncing.value = true;
  syncLog.value = [];
  try {
    const res = await SincronizarConPeriodo({
      modo:  syncModo.value,
      desde: syncModo.value === "rango" ? syncDesde.value : "",
      hasta: syncModo.value === "rango" ? syncHasta.value : "",
    });
    lastCheck.value = res;
    await fetchTransferencias();
    if (res.nuevas > 0) {
      toast.success(`${res.nuevas} nueva(s) transferencia(s) importadas`);
    } else {
      toast.info(`${res.revisados} mensajes revisados — sin transferencias nuevas`);
    }
  } catch (e: any) {
    toast.error("Error al sincronizar", { description: e?.toString() });
    syncLog.value.push({ nivel: "error", mensaje: `Error fatal: ${e}`, ts: new Date().toLocaleTimeString() });
  } finally {
    syncing.value = false;
  }
}

async function toggleAutoPolling() {
  const next = !autoPolling.value;
  try {
    await SetAutoPolling(next);
    autoPolling.value = next;
    toast.info(next ? "Sincronización automática activada" : "Sincronización automática desactivada");
  } catch (e: any) {
    toast.error("Error al cambiar polling", { description: e?.toString() });
  }
}

async function iniciarAuth() {
  authenticating.value = true;
  try {
    const url = await IniciarOAuth2();
    BrowserOpenURL(url);
    toast.info("Navegador abierto", {
      description: "Autoriza el acceso en el navegador. La app detectará la conexión automáticamente.",
    });
    const interval = setInterval(async () => {
      const s = await EstadoAuth();
      if (s.authenticated) {
        clearInterval(interval);
        auth.value = s;
        toast.success("Cuenta Bancolombia conectada");
        await fetchTransferencias();
      }
    }, 2000);
    setTimeout(() => clearInterval(interval), 300_000);
  } catch (e: any) {
    toast.error("Error al iniciar autenticación", { description: e?.toString() });
  } finally {
    authenticating.value = false;
  }
}

async function desconectarGmail() {
  try {
    await RevocarAuth();
    auth.value = { ...auth.value, authenticated: false };
    toast.info("Cuenta Bancolombia desconectada");
  } catch (e: any) {
    toast.error("Error al revocar", { description: e?.toString() });
  }
}

async function marcarTodasLeidas() {
  if (markingAllRead.value) return;
  markingAllRead.value = true;
  try {
    await MarcarTodasLeidas();
    items.value.forEach(i => (i.leido = true));
    toast.success("Todas las transferencias marcadas como leídas");
  } catch (e: any) {
    toast.error("Error al marcar como leídas", { description: e?.toString() });
  } finally {
    markingAllRead.value = false;
  }
}

async function openDetail(item: Transferencia) {
  selectedItem.value = item;
  showDetail.value = true;
  if (!item.leido) {
    await MarcarLeida(item.uuid);
    const idx = items.value.findIndex(i => i.uuid === item.uuid);
    if (idx >= 0) items.value[idx].leido = true;
  }
}

async function eliminar(uuid: string) {
  if (deletingUUID.value) return;
  deletingUUID.value = uuid;
  try {
    await EliminarTransferencia(uuid);
    items.value = items.value.filter(i => i.uuid !== uuid);
    totalItems.value = Math.max(0, totalItems.value - 1);
    if (showDetail.value && selectedItem.value?.uuid === uuid) {
      showDetail.value = false;
    }
    confirmDeleteUUID.value = null;
    toast.success("Transferencia eliminada");
  } catch (e: any) {
    toast.error("Error al eliminar", { description: e?.toString() });
  } finally {
    deletingUUID.value = null;
  }
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────

onMounted(async () => {
  await loadAuth();
  if (auth.value.authenticated) {
    await fetchTransferencias();
    await loadPollingState();
  }

  EventsOn("bancolombia:nueva", async (t: Transferencia) => {
    items.value.unshift(t);
    totalItems.value++;
    toast.success(`Nueva transferencia: ${formatCOP(t.monto)}`, {
      description: t.remitente || "Bancolombia",
    });
  });

  EventsOn("bancolombia:sync:result", (res: CheckResult) => {
    lastCheck.value = res;
  });

  EventsOn("bancolombia:sync:log", (entry: LogEntry) => {
    syncLog.value.push(entry);
    scrollLog();
  });
});

onUnmounted(() => {
  EventsOff("bancolombia:nueva");
  EventsOff("bancolombia:sync:result");
  EventsOff("bancolombia:sync:log");
});

watch(page, fetchTransferencias);
watch(pageSize, () => { page.value = 1; fetchTransferencias(); });
watch(soloNoLeidas, () => { page.value = 1; fetchTransferencias(); });
</script>

<template>
  <!-- Detail dialog -->
  <Dialog v-model:open="showDetail">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2 text-sm">
          <component :is="transferTipoIcon(selectedItem?.concepto ?? '')" class="h-4 w-4 text-emerald-600" />
          Transferencia recibida
        </DialogTitle>
      </DialogHeader>
      <div v-if="selectedItem" class="space-y-3">
        <!-- Amount hero -->
        <div class="rounded-lg border p-4 bg-emerald-50 text-center">
          <p class="text-3xl font-bold text-emerald-700">{{ formatCOP(selectedItem.monto) }}</p>
          <p class="text-xs text-emerald-600 mt-1">{{ formatDate(selectedItem.fecha) }}</p>
        </div>

        <!-- Meta grid -->
        <div class="rounded-lg border bg-muted/20 divide-y text-xs">
          <div class="flex items-center justify-between px-3 py-2 gap-4">
            <span class="text-muted-foreground shrink-0">Remitente</span>
            <span class="font-semibold text-right truncate">{{ selectedItem.remitente || "—" }}</span>
          </div>
          <div v-if="selectedItem.concepto" class="flex items-center justify-between px-3 py-2 gap-4">
            <span class="text-muted-foreground shrink-0">Tipo</span>
            <span class="text-right">{{ selectedItem.concepto }}</span>
          </div>
          <div v-if="selectedItem.referencia" class="flex items-center justify-between px-3 py-2 gap-4">
            <span class="text-muted-foreground shrink-0">Referencia</span>
            <span class="font-mono text-right">{{ selectedItem.referencia }}</span>
          </div>
        </div>

        <!-- Delete -->
        <Button
          size="sm"
          variant="outline"
          class="w-full text-red-600 border-red-200 hover:bg-red-50 hover:border-red-300 text-xs"
          :disabled="!!deletingUUID"
          @click="confirmDeleteUUID = selectedItem!.uuid"
        >
          <Trash2 class="h-3.5 w-3.5 mr-1.5" />
          Eliminar
        </Button>
      </div>
    </DialogContent>
  </Dialog>

  <!-- Delete confirm dialog -->
  <Dialog :open="!!confirmDeleteUUID" @update:open="v => { if (!v) confirmDeleteUUID = null }">
    <DialogContent class="max-w-xs">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2 text-sm">
          <Trash2 class="h-4 w-4 text-red-600" />
          Eliminar transferencia
        </DialogTitle>
      </DialogHeader>
      <p class="text-sm text-muted-foreground">¿Eliminar esta transferencia del registro? Esta acción no se puede deshacer.</p>
      <div class="flex gap-2 pt-1">
        <Button variant="outline" size="sm" class="flex-1 text-xs" @click="confirmDeleteUUID = null">Cancelar</Button>
        <Button
          size="sm"
          class="flex-1 text-xs bg-red-600 hover:bg-red-700"
          :disabled="!!deletingUUID"
          @click="eliminar(confirmDeleteUUID!)"
        >
          <Loader2 v-if="deletingUUID" class="h-3.5 w-3.5 mr-1 animate-spin" />
          Eliminar
        </Button>
      </div>
    </DialogContent>
  </Dialog>

  <div class="flex flex-col h-full">
    <!-- Header — same compact style as FacturasElectronicas -->
    <div class="shrink-0 border-b px-4 py-3 flex items-center justify-between bg-background">
      <div>
        <p class="text-sm font-semibold flex items-center gap-2">
          <Landmark class="h-4 w-4" />
          Transferencias Bancolombia
        </p>
        <p class="text-xs text-muted-foreground mt-0.5">Notificaciones de transferencias por Gmail</p>
      </div>
      <div class="flex items-center gap-2">
        <!-- Auth chip — same style as FacturasElectronicas -->
        <span class="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border"
          :class="auth.authenticated
            ? 'bg-green-50 text-green-700 border-green-200'
            : 'bg-gray-50 text-gray-500 border-gray-200'">
          <component :is="auth.authenticated ? ShieldCheck : ShieldOff" class="h-3.5 w-3.5" />
          {{ auth.authenticated ? "Gmail conectado" : "Sin conexión" }}
        </span>

        <Button v-if="!auth.authenticated" size="sm" class="h-7 text-xs gap-1.5" :disabled="authenticating" @click="iniciarAuth">
          <Loader2 v-if="authenticating" class="h-3.5 w-3.5 animate-spin" />
          <Mail v-else class="h-3.5 w-3.5" />
          Conectar Gmail
        </Button>

        <Button v-else variant="ghost" size="sm" class="h-7 text-xs gap-1" @click="desconectarGmail">
          <ShieldOff class="h-3.5 w-3.5" />
          Desconectar
        </Button>

        <!-- Sync popover — same pattern as FacturasElectronicas -->
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
              <p class="text-[10px] text-muted-foreground mt-0.5">Elige el período de correos a revisar</p>
            </div>
            <div class="p-3 space-y-3">
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

              <div v-if="syncModo === 'completo'"
                class="flex items-start gap-1.5 text-[10px] text-amber-700 bg-amber-50 border border-amber-200 rounded px-2 py-1.5">
                <AlertCircle class="h-3 w-3 shrink-0 mt-px" />
                Revisará todo el historial de Gmail. Puede tardar varios minutos.
              </div>

              <Button class="w-full h-7 text-xs" @click="iniciarSync">
                <RefreshCw class="h-3.5 w-3.5 mr-1.5" />
                Iniciar — {{ MODO_LABELS[syncModo] }}
              </Button>

              <!-- Auto-polling toggle -->
              <div class="border-t pt-2.5 flex items-center justify-between">
                <div>
                  <p class="text-xs font-medium">Sincronización automática</p>
                  <p class="text-[10px] text-muted-foreground">Revisa cada 2 min en segundo plano</p>
                </div>
                <button
                  @click="toggleAutoPolling"
                  class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-none"
                  :class="autoPolling ? 'bg-primary' : 'bg-input'"
                  type="button"
                  :aria-checked="autoPolling"
                  role="switch"
                >
                  <span
                    class="pointer-events-none block h-4 w-4 rounded-full bg-background shadow-lg ring-0 transition-transform"
                    :class="autoPolling ? 'translate-x-4' : 'translate-x-0'"
                  />
                </button>
              </div>
            </div>
          </PopoverContent>
        </Popover>
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
          <p class="text-muted-foreground">URI de redirección:
            <code class="font-mono">http://localhost:8095/bancolombia/oauth2/callback</code></p>
        </AlertDescription>
      </Alert>
    </div>

    <div v-if="auth.credPresent && !auth.authenticated" class="shrink-0 m-4">
      <Alert class="border-amber-200 bg-amber-50">
        <Landmark class="h-4 w-4 text-amber-600" />
        <AlertTitle class="text-amber-800">Configura la cuenta de Gmail de Bancolombia</AlertTitle>
        <AlertDescription class="text-amber-700 text-xs">
          Conecta la cuenta de Gmail donde llegan las notificaciones de transferencia. Se usará la
          misma app de Google Cloud que el módulo de facturas.
        </AlertDescription>
      </Alert>
    </div>

    <!-- Real-time sync log — same dark terminal style as FacturasElectronicas -->
    <div v-if="syncing || syncLog.length > 0" class="shrink-0 border-b bg-slate-950 text-slate-200">
      <div class="flex items-center gap-4 px-3 py-1.5 border-b border-slate-800 text-xs">
        <Loader2 v-if="syncing" class="h-3 w-3 animate-spin text-blue-400 shrink-0" />
        <CheckCircle2 v-else class="h-3 w-3 text-green-400 shrink-0" />
        <span class="text-slate-400 font-mono">{{ syncing ? "Sincronizando…" : "Completado" }}</span>
        <template v-if="lastCheck">
          <span class="font-mono text-slate-300">↓ {{ lastCheck.revisados }} revisados</span>
          <span class="font-mono text-green-400">✓ {{ lastCheck.nuevas }} nuevas</span>
        </template>
        <button v-if="!syncing" class="ml-auto text-slate-500 hover:text-slate-300 text-[10px]"
          @click="syncLog = []">Cerrar</button>
      </div>
      <div ref="logRef" class="overflow-y-auto max-h-36 px-3 py-1.5 font-mono text-[10px] leading-5 space-y-px">
        <div v-for="(entry, i) in syncLog" :key="i" class="flex gap-2">
          <span class="text-slate-600 shrink-0">{{ entry.ts }}</span>
          <span :class="logClass[entry.nivel] ?? 'text-slate-300'">{{ entry.mensaje }}</span>
        </div>
        <div v-if="syncing" class="flex gap-2">
          <span class="text-slate-600 shrink-0 invisible">00:00:00</span>
          <span class="text-slate-500 animate-pulse">▋</span>
        </div>
      </div>
    </div>

    <!-- KPI strip -->
    <div v-if="auth.authenticated" class="shrink-0 border-b px-4 py-2 grid grid-cols-3 gap-4 bg-muted/10">
      <div>
        <p class="text-[10px] text-muted-foreground font-medium uppercase tracking-wide">Total</p>
        <p class="text-xl font-bold mt-0.5">{{ totalItems }}</p>
      </div>
      <div>
        <p class="text-[10px] text-muted-foreground font-medium uppercase tracking-wide">Sin leer</p>
        <p class="text-xl font-bold mt-0.5 text-amber-600">{{ items.filter(i => !i.leido).length }}</p>
      </div>
      <div>
        <p class="text-[10px] text-muted-foreground font-medium uppercase tracking-wide">Total recibido</p>
        <p class="text-lg font-bold mt-0.5 text-emerald-600">{{ formatCOP(totalMonto) }}</p>
      </div>
    </div>

    <!-- Toolbar -->
    <div v-if="auth.authenticated" class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
      <button
        @click="soloNoLeidas = !soloNoLeidas"
        class="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded border transition-colors"
        :class="soloNoLeidas
          ? 'bg-primary text-primary-foreground border-primary'
          : 'border-border text-muted-foreground hover:text-foreground hover:bg-muted'">
        <CheckCircle2 class="h-3.5 w-3.5" />
        Solo no leídas
      </button>
      <Button
        v-if="items.some(i => !i.leido)"
        size="sm"
        variant="outline"
        class="h-7 text-xs gap-1.5"
        :disabled="markingAllRead"
        @click="marcarTodasLeidas"
      >
        <Loader2 v-if="markingAllRead" class="h-3.5 w-3.5 animate-spin" />
        <CheckCircle2 v-else class="h-3.5 w-3.5" />
        Marcar todas como leídas
      </Button>
      <span class="text-xs text-muted-foreground ml-1">{{ totalItems }} transferencias
        <span v-if="lastCheck" class="text-emerald-600 ml-1">· Última sync {{ lastCheck.ts }}</span>
      </span>
    </div>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex justify-center py-16">
        <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
      </div>

      <div v-else-if="auth.authenticated && items.length === 0"
        class="flex flex-col items-center py-16 gap-3 text-muted-foreground">
        <ArrowDownLeft class="h-8 w-8 opacity-40" />
        <p class="text-sm">
          {{ soloNoLeidas ? "Sin transferencias no leídas" : "Sin transferencias registradas" }}
        </p>
        <Button size="sm" variant="outline" class="text-xs" @click="syncPopoverOpen = true">
          <RefreshCw class="h-3.5 w-3.5 mr-1.5" />
          Sincronizar
        </Button>
      </div>

      <table v-else-if="auth.authenticated" class="w-full text-xs border-collapse">
        <thead class="sticky top-0 z-10">
          <tr class="bg-muted border-b">
            <th class="h-9 w-8 px-3 text-[10px] font-normal text-muted-foreground/40 text-center border-r">#</th>
            <th class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-left border-r whitespace-nowrap">Fecha</th>
            <th class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-left border-r whitespace-nowrap">Remitente</th>
            <th class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-left border-r whitespace-nowrap">Tipo</th>
            <th class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-right border-r whitespace-nowrap">Monto</th>
            <th class="h-9 px-3 text-[10px] font-medium text-muted-foreground text-center whitespace-nowrap">Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(item, idx) in items"
            :key="item.uuid"
            class="border-b hover:bg-muted/40 transition-colors group"
            :class="{ 'bg-amber-50/60 dark:bg-amber-950/20': !item.leido }"
          >
            <td class="h-9 px-3 text-center border-r select-none">
              <span v-if="!item.leido" class="h-2 w-2 rounded-full bg-amber-500 block mx-auto" />
              <span v-else class="text-[10px] text-muted-foreground/40">
                {{ (page - 1) * pageSize + idx + 1 }}
              </span>
            </td>
            <td class="h-9 px-3 border-r whitespace-nowrap text-muted-foreground cursor-pointer" @click="openDetail(item)">
              {{ formatDate(item.fecha) }}
            </td>
            <td class="h-9 px-3 border-r font-medium cursor-pointer" @click="openDetail(item)">
              {{ item.remitente || "—" }}
            </td>
            <td class="h-9 px-3 border-r text-muted-foreground cursor-pointer whitespace-nowrap" @click="openDetail(item)">
              {{ item.concepto || "—" }}
            </td>
            <td class="h-9 px-3 border-r text-right font-semibold text-emerald-700 cursor-pointer" @click="openDetail(item)">
              {{ formatCOP(item.monto) }}
            </td>
            <td class="h-9 px-3 text-center">
              <div class="flex items-center justify-center gap-1">
                <button
                  class="h-6 w-6 flex items-center justify-center rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted"
                  title="Ver detalle"
                  @click="openDetail(item)"
                >
                  <Eye class="h-3.5 w-3.5 text-muted-foreground" />
                </button>
                <button
                  class="h-6 w-6 flex items-center justify-center rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-100"
                  title="Eliminar"
                  :disabled="deletingUUID === item.uuid"
                  @click.stop="confirmDeleteUUID = item.uuid"
                >
                  <Loader2 v-if="deletingUUID === item.uuid" class="h-3.5 w-3.5 animate-spin text-muted-foreground" />
                  <Trash2 v-else class="h-3.5 w-3.5 text-red-500" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer / Pagination -->
    <div v-if="auth.authenticated" class="shrink-0 border-t px-3 py-1.5 flex items-center justify-between bg-background">
      <span class="text-xs text-muted-foreground">{{ totalItems }} transferencia(s)</span>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5">
          <span class="text-xs text-muted-foreground">Filas</span>
          <Select :model-value="`${pageSize}`" @update:model-value="v => { pageSize = Number(v); page = 1; }">
            <SelectTrigger class="h-7 w-16 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent side="top">
              <SelectItem v-for="s in [10, 25, 50, 100]" :key="s" :value="`${s}`" class="text-xs">{{ s }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="totalPages > 1" class="flex items-center gap-1">
          <Button size="sm" variant="outline" class="h-7 w-7 p-0 text-xs" :disabled="page <= 1" @click="page--">‹</Button>
          <span class="text-xs text-muted-foreground px-1">{{ page }} / {{ totalPages }}</span>
          <Button size="sm" variant="outline" class="h-7 w-7 p-0 text-xs" :disabled="page >= totalPages" @click="page++">›</Button>
        </div>
      </div>
    </div>
  </div>
</template>
