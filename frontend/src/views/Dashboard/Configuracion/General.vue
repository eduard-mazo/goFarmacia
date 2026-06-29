<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import { toast } from "vue-sonner";
import {
  Settings, Save, Database, CheckCircle2, XCircle, Loader2,
  Eye, EyeOff, AlertTriangle, Mail, KeyRound, FileJson, FolderKey,
  ShieldCheck, Trash2, ReceiptText, Landmark, RefreshCcw, Cloud, Inbox,
} from "lucide-vue-next";
import { useDBStore } from "@/stores/dbStore";
import { useAuthStore } from "@/stores/auth";
import {
  EstadoAuth, ObtenerCredenciales, GuardarCredenciales,
} from "@/../bridge/go/backend/GmailService";
import { GetTablas, TruncarTabla } from "@/../bridge/go/backend/Db";
import { GetSyncStatus, SetSyncEnabled } from "@/../bridge/go/backend/SyncStatus";
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction,
} from "@/components/ui/alert-dialog";

// ── Stores ─────────────────────────────────────────────────────────────────
const dbStore = useDBStore();
const authStore = useAuthStore();

// ── DB config state ─────────────────────────────────────────────────────────
const dsn = ref(dbStore.dsnHint || "");
const showDSN = ref(false);
const testLoading = ref(false);
const saveLoading = ref(false);
const testResult = ref<{ ok: boolean; message: string } | null>(null);

const dbStatusBadge = computed(() => {
  if (dbStore.setupMode)
    return { label: "No configurada", class: "bg-amber-100 text-amber-700 border-amber-300" };
  if (dbStore.connected)
    return { label: "Conectada", class: "bg-green-100 text-green-700 border-green-300" };
  return { label: "Sin conexión", class: "bg-red-100 text-red-700 border-red-300" };
});

async function handleTestConnection() {
  testResult.value = null;
  testLoading.value = true;
  const err = await dbStore.testConnection(dsn.value.trim());
  testLoading.value = false;
  if (err === null) {
    testResult.value = { ok: true, message: "Conexión exitosa — servidor alcanzable." };
  } else {
    const isReachable = err.toLowerCase().includes("servidor alcanzable");
    testResult.value = { ok: isReachable, message: err };
  }
}

async function handleSaveDB() {
  testResult.value = null;
  saveLoading.value = true;
  const err = await dbStore.configureDB(dsn.value.trim());
  saveLoading.value = false;
  if (err === null) {
    toast.success("Base de datos configurada", {
      description: "La conexión se estableció y las migraciones se aplicaron.",
    });
    if (authStore.currentUser?.UUID === "setup-admin") {
      setTimeout(() => authStore.logout(), 1500);
    }
  } else {
    toast.error("Error al configurar base de datos", { description: err });
  }
}

// ── Gmail credentials ────────────────────────────────────────────────────────
interface CredsInfo { Exists: boolean; Path: string; ProjectID: string; ClientID: string; }

const creds = ref<CredsInfo>({ Exists: false, Path: "", ProjectID: "", ClientID: "" });
const credJSON = ref("");
const showCredEditor = ref(false);
const savingCreds = ref(false);

async function loadCreds() {
  creds.value = await ObtenerCredenciales();
}

async function handleSaveCreds() {
  if (!credJSON.value.trim()) return;
  savingCreds.value = true;
  try {
    await GuardarCredenciales(credJSON.value.trim());
    toast.success("credentials.json guardado");
    credJSON.value = "";
    showCredEditor.value = false;
    await loadCreds();
  } catch (e) {
    toast.error("Error al guardar credenciales", { description: `${e}` });
  } finally {
    savingCreds.value = false;
  }
}

// ── General settings ────────────────────────────────────────────────────────
interface AppSettings {
  nombreTienda: string;
  direccion: string;
  telefono: string;
  tasaIVA: number;
  umbralStockBajo: number;
}

const STORAGE_KEY = "appSettings";
const settings = ref<AppSettings>({
  nombreTienda: "", direccion: "", telefono: "", tasaIVA: 19, umbralStockBajo: 10,
});

function loadSettings() {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored) {
    try { settings.value = { ...settings.value, ...JSON.parse(stored) }; }
    catch { /* ignore */ }
  }
}

function saveSettings() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings.value));
  toast.success("Configuración guardada", { description: "Los cambios se han aplicado correctamente." });
}

// ── Limpieza de datos de importación ────────────────────────────────────────
const conteoFacturas     = ref<number | null>(null);
const conteoTransfer     = ref<number | null>(null);
const limpiandoFacturas  = ref(false);
const limpiandoTransfer  = ref(false);

async function cargarConteos() {
  try {
    const tablas = await GetTablas();
    conteoFacturas.value = tablas.find((t: any) => t.Name === "facturas_compra")?.RowCount ?? 0;
    conteoTransfer.value = tablas.find((t: any) => t.Name === "transferencias_bancolombia")?.RowCount ?? 0;
  } catch { /* silencioso — no bloquear la página */ }
}

async function limpiarFacturas() {
  limpiandoFacturas.value = true;
  try {
    await TruncarTabla("facturas_compra");
    toast.success("Facturas electrónicas eliminadas", {
      description: "Las tablas facturas_compra y sus detalles han sido vaciadas.",
    });
    await cargarConteos();
  } catch (e) {
    toast.error("Error al limpiar facturas", { description: `${e}` });
  } finally {
    limpiandoFacturas.value = false;
  }
}

async function limpiarTransferencias() {
  limpiandoTransfer.value = true;
  try {
    await TruncarTabla("transferencias_bancolombia");
    toast.success("Transferencias eliminadas", {
      description: "La tabla transferencias_bancolombia ha sido vaciada.",
    });
    await cargarConteos();
  } catch (e) {
    toast.error("Error al limpiar transferencias", { description: `${e}` });
  } finally {
    limpiandoTransfer.value = false;
  }
}

// ── Sincronizaciones automáticas (daemons) ─────────────────────────────────
interface SyncRow {
  id: string;
  label: string;
  authenticated: boolean;
  credPresent: boolean;
  enabled: boolean;
  running: boolean;
  nextRun: string;
  lastRun: string;
}

const syncRows = ref<SyncRow[]>([]);
const syncToggling = ref<Record<string, boolean>>({});
let syncPollTimer: number | undefined;

const ICONS_BY_ID: Record<string, any> = {
  drive: Cloud,
  gmail: Mail,
  outlook: Inbox,
  bancolombia: Landmark,
};

function iconFor(id: string) { return ICONS_BY_ID[id] || RefreshCcw; }

function formatRelative(iso: string): string {
  if (!iso) return "—";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "—";
  const diffMs = t - Date.now();
  const abs = Math.abs(diffMs);
  const mins = Math.round(abs / 60000);
  if (mins < 1) return diffMs < 0 ? "hace <1 min" : "<1 min";
  if (mins < 60) return diffMs < 0 ? `hace ${mins} min` : `en ${mins} min`;
  const hrs = Math.round(mins / 60);
  if (hrs < 24) return diffMs < 0 ? `hace ${hrs} h` : `en ${hrs} h`;
  const days = Math.round(hrs / 24);
  return diffMs < 0 ? `hace ${days} d` : `en ${days} d`;
}

async function loadSyncStatus() {
  try {
    syncRows.value = await GetSyncStatus();
  } catch (e) {
    console.warn("[sync-status] fallo:", e);
  }
}

async function toggleSync(row: SyncRow) {
  if (!row.authenticated) {
    toast.warning("Sin autenticación", {
      description: "Conecta el proveedor antes de activar la sincronización automática.",
    });
    return;
  }
  const next = !row.enabled;
  syncToggling.value[row.id] = true;
  try {
    await SetSyncEnabled(row.id, next);
    toast.success(next ? "Sincronización activada" : "Sincronización desactivada", {
      description: row.label,
    });
    await loadSyncStatus();
  } catch (e) {
    toast.error("No se pudo cambiar el estado", { description: `${e}` });
  } finally {
    syncToggling.value[row.id] = false;
  }
}

onMounted(async () => {
  loadSettings();
  if (!dsn.value && dbStore.dsnHint) dsn.value = dbStore.dsnHint;
  await loadCreds();
  await cargarConteos();
  await loadSyncStatus();
  // Poll periodically so nextRun countdown + running state stay fresh.
  syncPollTimer = window.setInterval(loadSyncStatus, 15000);
});

import { onUnmounted } from "vue";
onUnmounted(() => { if (syncPollTimer) window.clearInterval(syncPollTimer); });
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- Page header -->
    <div>
      <h1 class="text-2xl font-semibold tracking-tight flex items-center gap-2">
        <Settings class="h-5 w-5 text-muted-foreground" />
        Configuración General
      </h1>
      <p class="text-sm text-muted-foreground mt-0.5">
        Conexión a base de datos, credenciales Gmail y parámetros del sistema
      </p>
    </div>

    <!-- ═══ Top row: DB + Gmail side by side ════════════════════════════════ -->
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-6">

      <!-- ── Database connection ─────────────────────────────────────────── -->
      <Card :class="{ 'border-amber-300': dbStore.setupMode }">
        <CardHeader class="pb-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Database class="h-4 w-4 text-muted-foreground" />
              <CardTitle class="text-base font-semibold">Conexión a PostgreSQL</CardTitle>
            </div>
            <Badge variant="outline" :class="dbStatusBadge.class" class="text-xs px-2 py-0.5">
              <span class="h-1.5 w-1.5 rounded-full mr-1.5 inline-block"
                :class="{
                  'bg-green-500': dbStore.connected,
                  'bg-amber-400': dbStore.setupMode,
                  'bg-red-500': !dbStore.connected && !dbStore.setupMode,
                }" />
              {{ dbStatusBadge.label }}
            </Badge>
          </div>
          <CardDescription v-if="dbStore.setupMode" class="text-amber-700 mt-1 text-sm">
            <AlertTriangle class="h-3.5 w-3.5 inline mr-1" />
            Modo configuración. Ingresa el DSN de tu base de datos PostgreSQL para comenzar.
          </CardDescription>
          <CardDescription v-else-if="dbStore.connected" class="mt-1 text-sm">
            Servidor conectado. Puedes actualizar el DSN y reconectar si es necesario.
          </CardDescription>
        </CardHeader>

        <CardContent class="space-y-4">
          <div class="grid gap-2">
            <Label for="dsn">Cadena de conexión (DSN)</Label>
            <div class="relative">
              <Input
                id="dsn"
                v-model="dsn"
                :type="showDSN ? 'text' : 'password'"
                class="h-9 pr-10 font-mono text-sm"
                placeholder="postgres://usuario:contraseña@host:5432/nombre_bd?sslmode=disable"
              />
              <button type="button" @click="showDSN = !showDSN"
                class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground transition-colors"
                tabindex="-1">
                <EyeOff v-if="showDSN" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
            <p class="text-xs text-muted-foreground">
              Formato:
              <code class="bg-muted px-1 rounded text-[11px]">postgres://user:pass@host:5432/dbname?sslmode=disable</code>
            </p>
          </div>

          <Alert v-if="testResult" :variant="testResult.ok ? 'default' : 'destructive'" class="py-2 px-3"
            :class="testResult.ok ? 'border-green-300 bg-green-50 text-green-800' : ''">
            <CheckCircle2 v-if="testResult.ok" class="h-4 w-4 text-green-600" />
            <XCircle v-else class="h-4 w-4" />
            <AlertDescription class="ml-6 text-xs">{{ testResult.message }}</AlertDescription>
          </Alert>

          <div class="flex gap-2 pt-1">
            <Button variant="outline" size="sm" class="h-9 gap-2"
              :disabled="testLoading || saveLoading || !dsn.trim()" @click="handleTestConnection">
              <Loader2 v-if="testLoading" class="h-4 w-4 animate-spin" />
              <CheckCircle2 v-else class="h-4 w-4" />
              Probar
            </Button>
            <Button size="sm" class="h-9 gap-2"
              :disabled="saveLoading || testLoading || !dsn.trim()" @click="handleSaveDB">
              <Loader2 v-if="saveLoading" class="h-4 w-4 animate-spin" />
              <Database v-else class="h-4 w-4" />
              {{ saveLoading ? "Conectando..." : "Guardar y Conectar" }}
            </Button>
          </div>

          <p v-if="dbStore.setupMode" class="text-xs text-muted-foreground border-t pt-3">
            Después de conectar, el sistema cerrará la sesión de configuración. Regístrate con tu cuenta real.
          </p>
        </CardContent>
      </Card>

      <!-- ── Gmail credentials ───────────────────────────────────────────── -->
      <Card>
        <CardHeader class="pb-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Mail class="h-4 w-4 text-muted-foreground" />
              <CardTitle class="text-base font-semibold">Credenciales Gmail OAuth2</CardTitle>
            </div>
            <Badge variant="outline" class="text-xs px-2 py-0.5"
              :class="creds.Exists
                ? 'bg-green-100 text-green-700 border-green-300'
                : 'bg-red-100 text-red-700 border-red-300'">
              <span class="h-1.5 w-1.5 rounded-full mr-1.5 inline-block"
                :class="creds.Exists ? 'bg-green-500' : 'bg-red-500'" />
              {{ creds.Exists ? "Configuradas" : "No configuradas" }}
            </Badge>
          </div>
          <CardDescription class="mt-1 text-sm">
            Necesarias para importar facturas DIAN desde Gmail.
          </CardDescription>
        </CardHeader>

        <CardContent class="space-y-4">
          <!-- Current credentials info -->
          <div class="rounded-lg border bg-muted/40 p-3 space-y-2 text-sm">
            <div class="flex items-center gap-2 text-muted-foreground">
              <FolderKey class="h-3.5 w-3.5 shrink-0" />
              <span class="font-mono text-xs break-all select-all">{{ creds.Path }}</span>
            </div>
            <Separator />
            <div v-if="creds.Exists" class="space-y-1.5">
              <div class="flex items-center gap-2">
                <FileJson class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <span class="text-xs text-muted-foreground">Proyecto:</span>
                <span class="font-mono text-xs">{{ creds.ProjectID || "—" }}</span>
              </div>
              <div class="flex items-center gap-2">
                <KeyRound class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <span class="text-xs text-muted-foreground">Client ID:</span>
                <span class="font-mono text-xs truncate max-w-[200px]">{{ creds.ClientID || "—" }}</span>
              </div>
              <div class="flex items-center gap-2 pt-0.5">
                <ShieldCheck class="h-3.5 w-3.5 text-green-600 shrink-0" />
                <span class="text-xs text-green-700">Archivo presente — listo para usar</span>
              </div>
            </div>
            <div v-else class="flex items-center gap-2 text-amber-700 text-xs">
              <AlertTriangle class="h-3.5 w-3.5 shrink-0" />
              Coloca el <code class="bg-muted px-1 rounded">credentials.json</code> de Google OAuth2 en esa ruta.
            </div>
          </div>

          <!-- Toggle editor -->
          <Button variant="outline" size="sm" class="h-8 text-xs gap-1.5 w-full"
            @click="showCredEditor = !showCredEditor">
            <FileJson class="h-3.5 w-3.5" />
            {{ showCredEditor ? "Ocultar editor" : (creds.Exists ? "Actualizar credentials.json" : "Pegar nuevo credentials.json") }}
          </Button>

          <!-- JSON editor -->
          <template v-if="showCredEditor">
            <div class="space-y-2">
              <Label class="text-xs">Pega aquí el contenido de credentials.json</Label>
              <Textarea
                v-model="credJSON"
                class="font-mono text-xs h-36 resize-none"
                placeholder='{"installed":{"client_id":"...","client_secret":"...","redirect_uris":["http://localhost:8094/gmail/oauth2/callback"]}}'
              />
              <p class="text-xs text-muted-foreground">
                Descarga desde
                <strong>Google Cloud Console → APIs → Credenciales → OAuth 2.0 → Aplicación de escritorio</strong>.
                Asegura que el URI de redirección sea
                <code class="bg-muted px-1 rounded">http://localhost:8094/gmail/oauth2/callback</code>.
              </p>
              <div class="flex gap-2">
                <Button size="sm" class="h-8 text-xs gap-1.5"
                  :disabled="savingCreds || !credJSON.trim()" @click="handleSaveCreds">
                  <Loader2 v-if="savingCreds" class="h-3.5 w-3.5 animate-spin" />
                  <Save v-else class="h-3.5 w-3.5" />
                  {{ savingCreds ? "Guardando..." : "Guardar credentials.json" }}
                </Button>
                <Button variant="ghost" size="sm" class="h-8 text-xs"
                  @click="showCredEditor = false; credJSON = ''">
                  Cancelar
                </Button>
              </div>
            </div>
          </template>
        </CardContent>
      </Card>
    </div>

    <!-- ═══ Sincronizaciones automáticas ═══════════════════════════════════ -->
    <template v-if="!dbStore.setupMode">
      <Card>
        <CardHeader class="pb-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <RefreshCcw class="h-4 w-4 text-muted-foreground" />
              <CardTitle class="text-base font-semibold">Sincronizaciones automáticas</CardTitle>
            </div>
            <Button variant="ghost" size="sm" class="h-7 text-xs gap-1.5" @click="loadSyncStatus">
              <RefreshCcw class="h-3.5 w-3.5" />Refrescar
            </Button>
          </div>
          <CardDescription class="mt-1 text-sm">
            Activa los daemons en segundo plano para cada proveedor.
            Se ejecutan periódicamente mientras la aplicación esté abierta.
          </CardDescription>
        </CardHeader>

        <CardContent class="pt-0">
          <div v-if="syncRows.length === 0" class="py-8 text-center text-sm text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin inline-block mr-2" />
            Cargando estado…
          </div>

          <ul v-else class="divide-y">
            <li
              v-for="row in syncRows"
              :key="row.id"
              class="flex items-center gap-4 py-3"
            >
              <!-- Icon -->
              <div class="rounded-md bg-muted p-2 shrink-0">
                <component :is="iconFor(row.id)" class="h-4 w-4 text-muted-foreground" />
              </div>

              <!-- Label + metadata -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium truncate">{{ row.label }}</span>

                  <!-- Auth dot -->
                  <span
                    class="h-1.5 w-1.5 rounded-full shrink-0"
                    :class="row.authenticated ? 'bg-emerald-500' : row.credPresent ? 'bg-amber-500' : 'bg-red-500'"
                    :title="row.authenticated ? 'Autenticado' : row.credPresent ? 'Credenciales sin token' : 'Sin credenciales'"
                  />

                  <!-- Running dot -->
                  <span
                    v-if="row.enabled && row.running"
                    class="inline-flex items-center gap-1 text-[10px] font-medium text-emerald-600"
                  >
                    <span class="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                    en ejecución
                  </span>
                </div>

                <div class="flex items-center gap-3 mt-0.5 text-xs text-muted-foreground">
                  <span v-if="row.lastRun" class="truncate">
                    Última: {{ formatRelative(row.lastRun) }}
                  </span>
                  <span v-if="row.enabled && row.nextRun" class="truncate">
                    Próxima: {{ formatRelative(row.nextRun) }}
                  </span>
                  <span v-if="!row.authenticated" class="text-amber-600 truncate">
                    Requiere autenticación
                  </span>
                </div>
              </div>

              <!-- Toggle switch -->
              <button
                type="button"
                role="switch"
                :aria-checked="row.enabled"
                :disabled="!row.authenticated || !!syncToggling[row.id]"
                class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                :class="row.enabled ? 'bg-emerald-500' : 'bg-muted-foreground/30'"
                @click="toggleSync(row)"
              >
                <span
                  class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow-md ring-0 transition-transform"
                  :class="row.enabled ? 'translate-x-4' : 'translate-x-0'"
                />
              </button>
            </li>
          </ul>
        </CardContent>
      </Card>
    </template>

    <!-- ═══ Bottom section (disabled in setup mode) ═════════════════════════ -->
    <template v-if="!dbStore.setupMode">
      <div class="grid grid-cols-1 xl:grid-cols-2 gap-6">

        <!-- ── Datos de la tienda ──────────────────────────────────────────── -->
        <Card>
          <CardHeader class="pb-4">
            <CardTitle class="text-base font-semibold">Datos de la Farmacia</CardTitle>
            <CardDescription class="text-sm">
              Información visible en tickets y reportes
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="grid gap-2">
              <Label for="nombreTienda">Nombre</Label>
              <Input id="nombreTienda" v-model="settings.nombreTienda" class="h-9"
                placeholder="Ej: Droguería Luna" />
            </div>
            <div class="grid gap-2">
              <Label for="direccion">Dirección</Label>
              <Input id="direccion" v-model="settings.direccion" class="h-9"
                placeholder="Dirección del establecimiento" />
            </div>
            <div class="grid gap-2">
              <Label for="telefono">Teléfono</Label>
              <Input id="telefono" v-model="settings.telefono" class="h-9"
                placeholder="Ej: 310 000 0000" />
            </div>
          </CardContent>
        </Card>

        <!-- ── Parámetros del sistema ──────────────────────────────────────── -->
        <Card>
          <CardHeader class="pb-4">
            <CardTitle class="text-base font-semibold">Parámetros del Sistema</CardTitle>
            <CardDescription class="text-sm">
              Valores por defecto aplicados en operaciones del POS
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <div class="grid gap-2">
                <Label for="tasaIVA">Tasa de IVA (%)</Label>
                <Input id="tasaIVA" v-model.number="settings.tasaIVA" type="number"
                  min="0" max="100" class="h-9" />
                <p class="text-xs text-muted-foreground">Porcentaje IVA sobre productos gravados</p>
              </div>
              <div class="grid gap-2">
                <Label for="umbralStock">Umbral Stock Bajo</Label>
                <Input id="umbralStock" v-model.number="settings.umbralStockBajo" type="number"
                  min="0" class="h-9" />
                <p class="text-xs text-muted-foreground">Unidades mínimas antes de alerta</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="flex justify-end pt-2">
        <Button @click="saveSettings" class="h-9 gap-2">
          <Save class="w-4 h-4" />Guardar Configuración
        </Button>
      </div>

    </template>

    <!-- ═══ Zona de riesgo (visible siempre que haya conexión activa) ══════════ -->
    <template v-if="dbStore.connected">
      <div>
        <div class="flex items-center gap-2 mb-3">
          <Trash2 class="h-4 w-4 text-destructive" />
          <h2 class="text-sm font-semibold text-destructive">Zona de riesgo</h2>
        </div>
        <p class="text-xs text-muted-foreground mb-4">
          Estas acciones eliminan datos de importación permanentemente y no se pueden deshacer.
          Úsalas para corregir inconsistencias antes de una re-sincronización completa.
        </p>

        <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">

          <!-- Facturas electrónicas -->
          <div class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 flex flex-col gap-3">
            <div class="flex items-start gap-3">
              <div class="rounded-md bg-destructive/10 p-2 shrink-0">
                <ReceiptText class="h-4 w-4 text-destructive" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium">Facturas Electrónicas</p>
                <p class="text-xs text-muted-foreground mt-0.5">
                  Vacía <code class="bg-muted px-1 rounded text-[11px]">facturas_compra</code>
                  y <code class="bg-muted px-1 rounded text-[11px]">facturas_compra_detalles</code>.
                </p>
                <p class="text-xs text-muted-foreground mt-1">
                  Registros actuales:
                  <span class="font-semibold text-foreground">
                    {{ conteoFacturas === null ? "—" : conteoFacturas.toLocaleString() }}
                  </span>
                </p>
              </div>
            </div>

            <AlertDialog>
              <template #trigger>
                <Button
                  variant="destructive" size="sm" class="h-8 text-xs w-full gap-1.5"
                  :disabled="limpiandoFacturas || conteoFacturas === 0"
                >
                  <Loader2 v-if="limpiandoFacturas" class="h-3.5 w-3.5 animate-spin" />
                  <Trash2 v-else class="h-3.5 w-3.5" />
                  {{ limpiandoFacturas ? "Eliminando..." : "Limpiar facturas electrónicas" }}
                </Button>
              </template>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle class="flex items-center gap-2 text-destructive">
                    <AlertTriangle class="h-4 w-4" />
                    ¿Eliminar todas las facturas electrónicas?
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    Se eliminarán <strong>{{ conteoFacturas?.toLocaleString() }} registros</strong>
                    de facturas_compra y todos sus detalles asociados. Esta acción
                    <strong>no se puede deshacer</strong>. Necesitarás re-sincronizar Gmail
                    para volver a importarlas.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancelar</AlertDialogCancel>
                  <AlertDialogAction
                    class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    @click="limpiarFacturas"
                  >
                    Sí, eliminar todo
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>

          <!-- Transferencias Bancolombia -->
          <div class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 flex flex-col gap-3">
            <div class="flex items-start gap-3">
              <div class="rounded-md bg-destructive/10 p-2 shrink-0">
                <Landmark class="h-4 w-4 text-destructive" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium">Transferencias Bancolombia</p>
                <p class="text-xs text-muted-foreground mt-0.5">
                  Vacía <code class="bg-muted px-1 rounded text-[11px]">transferencias_bancolombia</code>
                  por completo.
                </p>
                <p class="text-xs text-muted-foreground mt-1">
                  Registros actuales:
                  <span class="font-semibold text-foreground">
                    {{ conteoTransfer === null ? "—" : conteoTransfer.toLocaleString() }}
                  </span>
                </p>
              </div>
            </div>

            <AlertDialog>
              <template #trigger>
                <Button
                  variant="destructive" size="sm" class="h-8 text-xs w-full gap-1.5"
                  :disabled="limpiandoTransfer || conteoTransfer === 0"
                >
                  <Loader2 v-if="limpiandoTransfer" class="h-3.5 w-3.5 animate-spin" />
                  <Trash2 v-else class="h-3.5 w-3.5" />
                  {{ limpiandoTransfer ? "Eliminando..." : "Limpiar transferencias" }}
                </Button>
              </template>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle class="flex items-center gap-2 text-destructive">
                    <AlertTriangle class="h-4 w-4" />
                    ¿Eliminar todas las transferencias Bancolombia?
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    Se eliminarán <strong>{{ conteoTransfer?.toLocaleString() }} registros</strong>
                    de transferencias_bancolombia. Esta acción
                    <strong>no se puede deshacer</strong>. Necesitarás re-sincronizar el correo
                    de Bancolombia para volver a importarlas.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancelar</AlertDialogCancel>
                  <AlertDialogAction
                    class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    @click="limpiarTransferencias"
                  >
                    Sí, eliminar todo
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>

        </div>
      </div>
    </template>
  </div>
</template>
