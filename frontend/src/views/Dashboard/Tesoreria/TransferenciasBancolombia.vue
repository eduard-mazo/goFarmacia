<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card, CardContent, CardDescription, CardHeader, CardTitle,
} from "@/components/ui/card";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
  Alert, AlertDescription, AlertTitle,
} from "@/components/ui/alert";
import {
  Landmark, RefreshCw, ShieldCheck, ShieldOff, Loader2,
  CheckCircle2, AlertCircle, ArrowDownLeft, Eye, Clock,
  DollarSign, Mail,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { EventsOn, EventsOff } from "@/../wailsjs/runtime";
import { backend } from "@/../wailsjs/go/models";
import {
  EstadoAuth, IniciarOAuth2, RevocarAuth,
  ObtenerTransferencias, MarcarLeida, VerificarAhora,
} from "@/../wailsjs/go/backend/BancolombiaService";

// ─── Types ────────────────────────────────────────────────────────────────────

type AuthStatus = backend.BancolombiaAuthStatus;
type Transferencia = backend.TransferenciaBancolombia;
type CheckResult = backend.BancolombiaCheckResult;

// ─── State ────────────────────────────────────────────────────────────────────

const auth = ref<AuthStatus>({ authenticated: false, credPresent: false, configDir: "" });

const items = ref<Transferencia[]>([]);
const totalItems = ref(0);
const totalMonto = ref(0);
const page = ref(1);
const pageSize = 50;
const soloNoLeidas = ref(false);

const loading = ref(false);
const checking = ref(false);
const authenticating = ref(false);

const lastCheck = ref<CheckResult | null>(null);
const showAuthDialog = ref(false);
const authUrl = ref("");

const selectedItem = ref<Transferencia | null>(null);
const showDetail = ref(false);

// ─── Computed ─────────────────────────────────────────────────────────────────

const noLeidas = computed(() => items.value.filter(i => !i.leido).length);

const totalPages = computed(() => Math.ceil(totalItems.value / pageSize));

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatCOP(v: number): string {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(v);
}

function formatDate(d: string | Date): string {
  if (!d) return "—";
  return new Date(d).toLocaleString("es-CO", {
    day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

// ─── Load data ────────────────────────────────────────────────────────────────

async function loadAuth() {
  auth.value = await EstadoAuth();
}

async function fetchTransferencias() {
  loading.value = true;
  try {
    const res = await ObtenerTransferencias(page.value, pageSize, soloNoLeidas.value);
    items.value = res.items ?? [];
    totalItems.value = res.total;
    totalMonto.value = res.totalMonto;
  } catch (e: any) {
    toast.error("Error al cargar transferencias", { description: e?.toString() });
  } finally {
    loading.value = false;
  }
}

// ─── Actions ──────────────────────────────────────────────────────────────────

async function verificarAhora() {
  if (!auth.value.authenticated) {
    toast.warning("Configura la cuenta de Gmail de Bancolombia primero");
    return;
  }
  checking.value = true;
  try {
    const res = await VerificarAhora();
    lastCheck.value = res;
    if (res.nuevas > 0) {
      toast.success(`${res.nuevas} nueva(s) transferencia(s) encontrada(s)`);
      await fetchTransferencias();
    } else {
      toast.info(`${res.revisados} mensajes revisados — sin transferencias nuevas`);
    }
  } catch (e: any) {
    toast.error("Error al verificar", { description: e?.toString() });
  } finally {
    checking.value = false;
  }
}

async function iniciarAuth() {
  authenticating.value = true;
  authUrl.value = "";
  try {
    const url = await IniciarOAuth2();
    authUrl.value = url;
    showAuthDialog.value = true;
    // Poll for auth completion
    const interval = setInterval(async () => {
      const s = await EstadoAuth();
      if (s.authenticated) {
        clearInterval(interval);
        auth.value = s;
        showAuthDialog.value = false;
        toast.success("Cuenta Bancolombia conectada");
        await fetchTransferencias();
      }
    }, 2000);
    // Stop polling after 5 minutes
    setTimeout(() => clearInterval(interval), 300_000);
  } catch (e: any) {
    toast.error("Error al iniciar autenticación", { description: e?.toString() });
  } finally {
    authenticating.value = false;
  }
}

async function revocarAuth() {
  try {
    await RevocarAuth();
    auth.value = { ...auth.value, authenticated: false };
    toast.info("Cuenta Bancolombia desconectada");
  } catch (e: any) {
    toast.error("Error al revocar", { description: e?.toString() });
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

function toggleFiltro() {
  soloNoLeidas.value = !soloNoLeidas.value;
  page.value = 1;
  fetchTransferencias();
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────

onMounted(async () => {
  await loadAuth();
  if (auth.value.authenticated) {
    await fetchTransferencias();
  }

  // Live updates from background ticker
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
});

onUnmounted(() => {
  EventsOff("bancolombia:nueva");
  EventsOff("bancolombia:sync:result");
});
</script>

<template>
  <div class="flex flex-col h-full p-6 gap-6 overflow-auto">

    <!-- Header row -->
    <div class="flex items-center justify-between gap-4 flex-wrap">
      <div class="flex items-center gap-3">
        <div class="h-9 w-9 rounded-lg bg-amber-500/10 flex items-center justify-center">
          <Landmark class="h-5 w-5 text-amber-600" />
        </div>
        <div>
          <h1 class="text-xl font-semibold">Transferencias Bancolombia</h1>
          <p class="text-xs text-muted-foreground">
            Sincronización automática cada 2 min
            <span v-if="lastCheck" class="ml-2 text-emerald-600">
              · Última verificación {{ lastCheck.ts }}
            </span>
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2 flex-wrap">
        <!-- Auth status -->
        <div v-if="auth.authenticated" class="flex items-center gap-1.5 text-xs text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-full px-3 py-1">
          <ShieldCheck class="h-3.5 w-3.5" />
          Gmail conectado
        </div>
        <div v-else class="flex items-center gap-1.5 text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-full px-3 py-1">
          <ShieldOff class="h-3.5 w-3.5" />
          Sin conexión Gmail
        </div>

        <!-- Verificar ahora -->
        <Button
          size="sm"
          variant="outline"
          :disabled="checking || !auth.authenticated"
          @click="verificarAhora"
        >
          <Loader2 v-if="checking" class="h-4 w-4 mr-1.5 animate-spin" />
          <RefreshCw v-else class="h-4 w-4 mr-1.5" />
          Verificar ahora
        </Button>

        <!-- Connect / disconnect -->
        <Button v-if="!auth.authenticated" size="sm" @click="iniciarAuth" :disabled="authenticating">
          <Loader2 v-if="authenticating" class="h-4 w-4 mr-1.5 animate-spin" />
          <Mail v-else class="h-4 w-4 mr-1.5" />
          Conectar cuenta
        </Button>
        <Button v-else size="sm" variant="ghost" class="text-red-600 hover:text-red-700 hover:bg-red-50" @click="revocarAuth">
          <ShieldOff class="h-4 w-4 mr-1.5" />
          Desconectar
        </Button>
      </div>
    </div>

    <!-- Not authenticated message -->
    <Alert v-if="!auth.authenticated" variant="default" class="border-amber-200 bg-amber-50">
      <Landmark class="h-4 w-4 text-amber-600" />
      <AlertTitle class="text-amber-800">Configura la cuenta de Gmail</AlertTitle>
      <AlertDescription class="text-amber-700">
        Conecta la cuenta de Gmail donde Bancolombia envía las notificaciones de
        transferencia. Se usará la misma app de Google Cloud que el módulo de facturas.
        El ticker verificará automáticamente cada 2 minutos en segundo plano.
      </AlertDescription>
    </Alert>

    <!-- KPI cards -->
    <div v-if="auth.authenticated" class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Total registradas</CardDescription>
          <CardTitle class="text-2xl">{{ totalItems }}</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Sin leer</CardDescription>
          <CardTitle class="text-2xl text-amber-600">
            {{ items.filter(i => !i.leido).length }}
          </CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Total recibido</CardDescription>
          <CardTitle class="text-xl text-emerald-600">{{ formatCOP(totalMonto) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Última sync</CardDescription>
          <CardTitle class="text-sm font-medium text-muted-foreground flex items-center gap-1">
            <Clock class="h-3.5 w-3.5" />
            {{ lastCheck?.ts ?? "—" }}
            <span v-if="lastCheck" class="text-xs">({{ lastCheck.revisados }} revisados)</span>
          </CardTitle>
        </CardHeader>
      </Card>
    </div>

    <!-- Filters row -->
    <div v-if="auth.authenticated" class="flex items-center gap-3">
      <Button
        size="sm"
        :variant="soloNoLeidas ? 'default' : 'outline'"
        @click="toggleFiltro"
        class="h-8"
      >
        <Eye class="h-3.5 w-3.5 mr-1.5" />
        Solo no leídas
      </Button>
      <span class="text-xs text-muted-foreground">{{ totalItems }} notificaciones</span>
    </div>

    <!-- Table -->
    <div v-if="auth.authenticated" class="rounded-lg border bg-card overflow-hidden">
      <div v-if="loading" class="flex justify-center py-16">
        <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
      </div>

      <div v-else-if="items.length === 0" class="flex flex-col items-center py-16 gap-3 text-muted-foreground">
        <ArrowDownLeft class="h-8 w-8 opacity-40" />
        <p class="text-sm">
          {{ soloNoLeidas ? "Sin transferencias no leídas" : "Sin transferencias registradas" }}
        </p>
        <Button size="sm" variant="outline" @click="verificarAhora" :disabled="checking">
          <RefreshCw class="h-4 w-4 mr-1.5" />
          Verificar ahora
        </Button>
      </div>

      <table v-else class="w-full text-sm">
        <thead class="bg-muted/50 text-muted-foreground text-xs">
          <tr>
            <th class="px-4 py-3 text-left font-medium w-8"></th>
            <th class="px-4 py-3 text-left font-medium">Fecha</th>
            <th class="px-4 py-3 text-left font-medium">Remitente</th>
            <th class="px-4 py-3 text-left font-medium">Concepto</th>
            <th class="px-4 py-3 text-left font-medium">Referencia</th>
            <th class="px-4 py-3 text-right font-medium">Monto</th>
            <th class="px-4 py-3 w-12"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr
            v-for="item in items"
            :key="item.uuid"
            class="hover:bg-muted/30 transition-colors cursor-pointer"
            :class="{ 'bg-amber-50/60': !item.leido }"
            @click="openDetail(item)"
          >
            <!-- Unread dot -->
            <td class="px-4 py-3">
              <span v-if="!item.leido" class="h-2 w-2 rounded-full bg-amber-500 block mx-auto" />
            </td>

            <td class="px-4 py-3 whitespace-nowrap text-xs text-muted-foreground">
              {{ formatDate(item.fecha) }}
            </td>

            <td class="px-4 py-3">
              <span class="font-medium">{{ item.remitente || "—" }}</span>
            </td>

            <td class="px-4 py-3 max-w-[200px] truncate text-muted-foreground text-xs">
              {{ item.concepto || item.rawSubject || "—" }}
            </td>

            <td class="px-4 py-3 text-xs font-mono text-muted-foreground">
              {{ item.referencia || "—" }}
            </td>

            <td class="px-4 py-3 text-right font-semibold text-emerald-700">
              {{ formatCOP(item.monto) }}
            </td>

            <td class="px-4 py-3">
              <Eye class="h-3.5 w-3.5 text-muted-foreground mx-auto" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div v-if="auth.authenticated && totalPages > 1" class="flex justify-center gap-2">
      <Button size="sm" variant="outline" :disabled="page <= 1" @click="page--; fetchTransferencias()">
        Anterior
      </Button>
      <span class="text-xs text-muted-foreground self-center">
        Página {{ page }} / {{ totalPages }}
      </span>
      <Button size="sm" variant="outline" :disabled="page >= totalPages" @click="page++; fetchTransferencias()">
        Siguiente
      </Button>
    </div>

  </div>

  <!-- ── Detail dialog ─────────────────────────────────────────────────────── -->
  <Dialog v-model:open="showDetail">
    <DialogContent class="max-w-md">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <ArrowDownLeft class="h-5 w-5 text-emerald-600" />
          Transferencia recibida
        </DialogTitle>
      </DialogHeader>
      <div v-if="selectedItem" class="space-y-3 text-sm">
        <div class="rounded-lg border p-4 bg-emerald-50 text-center">
          <p class="text-3xl font-bold text-emerald-700">{{ formatCOP(selectedItem.monto) }}</p>
          <p class="text-xs text-emerald-600 mt-1">{{ formatDate(selectedItem.fecha) }}</p>
        </div>

        <dl class="space-y-2">
          <div v-if="selectedItem.remitente" class="flex justify-between">
            <dt class="text-muted-foreground">Remitente</dt>
            <dd class="font-medium text-right max-w-[200px] truncate">{{ selectedItem.remitente }}</dd>
          </div>
          <div v-if="selectedItem.referencia" class="flex justify-between">
            <dt class="text-muted-foreground">Referencia</dt>
            <dd class="font-mono text-xs">{{ selectedItem.referencia }}</dd>
          </div>
          <div v-if="selectedItem.cuentaDestino" class="flex justify-between">
            <dt class="text-muted-foreground">Cuenta destino</dt>
            <dd class="font-mono text-xs">{{ selectedItem.cuentaDestino }}</dd>
          </div>
          <div v-if="selectedItem.concepto" class="flex justify-between gap-4">
            <dt class="text-muted-foreground shrink-0">Concepto</dt>
            <dd class="text-right text-xs max-w-[200px]">{{ selectedItem.concepto }}</dd>
          </div>
        </dl>

        <div class="pt-2 border-t text-xs text-muted-foreground">
          <p class="truncate">Asunto: {{ selectedItem.rawSubject }}</p>
        </div>
      </div>
    </DialogContent>
  </Dialog>

  <!-- ── OAuth2 dialog ──────────────────────────────────────────────────────── -->
  <Dialog v-model:open="showAuthDialog">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Landmark class="h-5 w-5 text-amber-600" />
          Conectar cuenta Bancolombia
        </DialogTitle>
      </DialogHeader>
      <div class="space-y-4 text-sm">
        <p class="text-muted-foreground">
          Se abrió el navegador para autorizar el acceso de lectura a la cuenta de Gmail
          donde Bancolombia envía las notificaciones. Autoriza y vuelve aquí.
        </p>
        <Alert>
          <CheckCircle2 class="h-4 w-4" />
          <AlertDescription>
            Esperando autorización... esta ventana se cerrará automáticamente.
          </AlertDescription>
        </Alert>
        <div v-if="authUrl" class="text-xs">
          <p class="text-muted-foreground mb-1">O copia este enlace en el navegador:</p>
          <code class="block bg-muted rounded p-2 break-all">{{ authUrl }}</code>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
