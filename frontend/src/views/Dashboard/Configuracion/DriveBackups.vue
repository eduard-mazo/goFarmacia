<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Separator } from "@/components/ui/separator";
import { toast } from "vue-sonner";
import { EventsOn, EventsOff } from "@/../bridge/runtime/runtime";
import {
  HardDrive, CloudUpload, Trash2, RefreshCw, Loader2,
  ShieldCheck, ShieldOff, FolderOpen, Clock, CheckCircle2, RotateCcw,
} from "lucide-vue-next";
import {
  EstadoAuthDrive, IniciarOAuth2Drive, RevocarAuthDrive,
  EjecutarBackupAhora, ListarBackups, EliminarBackup, RestaurarBackup,
  GetAutoBackup, SetAutoBackup,
} from "@/../bridge/go/backend/DriveBackupService";
import type { backend } from "@/../bridge/go/models";

type DriveAuthStatus = backend.DriveAuthStatus;
type DriveBackupResult = backend.DriveBackupResult;
type DriveBackupFile = backend.DriveBackupFile;
type DriveAutoBackupState = backend.DriveAutoBackupState;

// ── State ─────────────────────────────────────────────────────────────────────

const auth = ref<DriveAuthStatus>({ authenticated: false, credPresent: false, configDir: "" });
const autoState = ref<DriveAutoBackupState>({ enabled: true, nextBackup: "", lastBackup: "" });
const backups = ref<DriveBackupFile[]>([]);

const connecting = ref(false);
const backingUp = ref(false);
const loadingList = ref(false);
const deletingId = ref<string | null>(null);
const restoringId = ref<string | null>(null);
const restoreTarget = ref<DriveBackupFile | null>(null);
const restoreOpen = ref(false);

// ── Computed ──────────────────────────────────────────────────────────────────

const lastBackupLabel = computed(() => {
  if (!autoState.value.lastBackup) return "Nunca";
  return new Date(autoState.value.lastBackup).toLocaleString("es-CO", {
    day: "2-digit", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
});

const nextBackupLabel = computed(() => {
  if (!autoState.value.nextBackup || !autoState.value.enabled) return "—";
  return new Date(autoState.value.nextBackup).toLocaleString("es-CO", {
    day: "2-digit", month: "short",
    hour: "2-digit", minute: "2-digit",
  });
});

// ── Actions ───────────────────────────────────────────────────────────────────

async function cargarEstado() {
  auth.value = await EstadoAuthDrive();
  autoState.value = await GetAutoBackup();
}

async function cargarBackups() {
  if (!auth.value.authenticated) return;
  loadingList.value = true;
  try {
    backups.value = (await ListarBackups()) ?? [];
  } catch (e: any) {
    toast.error("Error al listar backups", { description: `${e}` });
  } finally {
    loadingList.value = false;
  }
}

async function conectar() {
  connecting.value = true;
  try {
    const url = await IniciarOAuth2Drive();
    if (url) window.open(url, "_blank");
    toast.info("Navegador abierto", {
      description: "Autentica con Google Drive. La app detectará el token automáticamente.",
    });
    // Poll until authenticated
    const poll = setInterval(async () => {
      await cargarEstado();
      if (auth.value.authenticated) {
        clearInterval(poll);
        toast.success("Google Drive conectado");
        cargarBackups();
      }
    }, 2000);
    setTimeout(() => clearInterval(poll), 120_000);
  } catch (e: any) {
    toast.error("Error al conectar Drive", { description: `${e}` });
  } finally {
    connecting.value = false;
  }
}

async function desconectar() {
  try {
    await RevocarAuthDrive();
    await cargarEstado();
    backups.value = [];
    toast.info("Google Drive desconectado");
  } catch (e: any) {
    toast.error("Error al desconectar", { description: `${e}` });
  }
}

async function hacerBackup() {
  backingUp.value = true;
  try {
    const result = await EjecutarBackupAhora();
    toast.success("Backup completado", {
      description: `${result.fileName} — ${formatSize(result.sizeBytes)}`,
    });
    autoState.value = await GetAutoBackup();
    await cargarBackups();
  } catch (e: any) {
    toast.error("Error al hacer backup", { description: `${e}` });
  } finally {
    backingUp.value = false;
  }
}

async function toggleAuto(enabled: boolean) {
  await SetAutoBackup(enabled);
  autoState.value = await GetAutoBackup();
}

function pedirRestaurar(backup: DriveBackupFile) {
  restoreTarget.value = backup;
  restoreOpen.value = true;
}

async function confirmarRestaurar() {
  const target = restoreTarget.value;
  if (!target) return;
  restoreOpen.value = false;
  restoringId.value = target.id;
  try {
    await RestaurarBackup(target.id);
    toast.success("Restauración completada", {
      description: `Base de datos restaurada desde ${target.name}`,
    });
  } catch (e: any) {
    toast.error("Error al restaurar backup", { description: `${e}` });
  } finally {
    restoringId.value = null;
    restoreTarget.value = null;
  }
}

async function eliminar(fileID: string) {
  deletingId.value = fileID;
  try {
    await EliminarBackup(fileID);
    backups.value = backups.value.filter((b) => b.id !== fileID);
    toast.success("Backup eliminado");
  } catch (e: any) {
    toast.error("Error al eliminar backup", { description: `${e}` });
  } finally {
    deletingId.value = null;
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

function formatDate(iso: string): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("es-CO", {
    day: "2-digit", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

onMounted(async () => {
  await cargarEstado();
  await cargarBackups();

  EventsOn("drive:backup:done", (result: DriveBackupResult) => {
    toast.success("Backup automático completado", {
      description: `${result.fileName} — ${formatSize(result.sizeBytes)}`,
    });
    cargarBackups();
    GetAutoBackup().then((s) => { autoState.value = s; });
  });

  EventsOn("drive:auth:ok", () => {
    cargarEstado();
    cargarBackups();
  });
});

import { onUnmounted } from "vue";
onUnmounted(() => {
  EventsOff("drive:backup:done");
  EventsOff("drive:auth:ok");
});
</script>

<template>
  <div class="max-w-3xl mx-auto p-6 space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-lg font-semibold flex items-center gap-2">
        <HardDrive class="h-5 w-5" />
        Backups Google Drive
      </h1>
      <p class="text-sm text-muted-foreground mt-0.5">
        Respaldo automático diario a las 7:00 PM en tu Google Drive personal (se conservan los últimos 5).
      </p>
    </div>

    <!-- Credentials missing -->
    <Alert v-if="!auth.credPresent">
      <FolderOpen class="h-4 w-4" />
      <AlertTitle>Credenciales no encontradas</AlertTitle>
      <AlertDescription class="text-xs space-y-1">
        <p>Coloca <code class="font-mono bg-muted px-1 rounded">credentials.json</code> en:</p>
        <p class="font-mono bg-muted px-2 py-1 rounded break-all select-all">{{ auth.configDir }}</p>
        <p class="text-muted-foreground mt-1">
          URI de redirección a agregar en Google Cloud Console:<br>
          <code class="font-mono">http://localhost:8096/drive/oauth2/callback</code>
        </p>
      </AlertDescription>
    </Alert>

    <!-- Auth + Auto Backup status card -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="text-sm font-medium">Estado de conexión</CardTitle>
        <CardDescription class="text-xs">Autenticación OAuth2 con Google Drive</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2.5">
            <component
              :is="auth.authenticated ? ShieldCheck : ShieldOff"
              class="h-5 w-5"
              :class="auth.authenticated ? 'text-green-600' : 'text-muted-foreground'"
            />
            <div>
              <p class="text-sm font-medium">
                {{ auth.authenticated ? "Google Drive conectado" : "Sin conexión" }}
              </p>
              <p class="text-xs text-muted-foreground">
                {{ auth.authenticated ? "Backups activos" : "Conecta para activar los backups automáticos" }}
              </p>
            </div>
          </div>
          <div class="flex gap-2">
            <Button
              v-if="!auth.authenticated"
              size="sm"
              class="gap-1.5"
              :disabled="!auth.credPresent || connecting"
              @click="conectar"
            >
              <Loader2 v-if="connecting" class="h-3.5 w-3.5 animate-spin" />
              <HardDrive v-else class="h-3.5 w-3.5" />
              {{ connecting ? "Abriendo…" : "Conectar Drive" }}
            </Button>
            <Button
              v-else
              size="sm"
              variant="outline"
              class="gap-1.5 text-red-600 border-red-200 hover:bg-red-50"
              @click="desconectar"
            >
              <ShieldOff class="h-3.5 w-3.5" />
              Desconectar
            </Button>
          </div>
        </div>

        <Separator v-if="auth.authenticated" />

        <!-- Auto backup toggle -->
        <div v-if="auth.authenticated" class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">Backup automático</p>
            <p class="text-xs text-muted-foreground">Se ejecuta diariamente a las 7:00 PM mientras la app está abierta</p>
          </div>
          <button
            @click="toggleAuto(!autoState.enabled)"
            class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-none"
            :class="autoState.enabled ? 'bg-primary' : 'bg-input'"
            type="button"
            :aria-checked="autoState.enabled"
            role="switch"
          >
            <span
              class="pointer-events-none block h-5 w-5 rounded-full bg-background shadow-lg ring-0 transition-transform"
              :class="autoState.enabled ? 'translate-x-5' : 'translate-x-0'"
            />
          </button>
        </div>

        <!-- Stats row -->
        <div v-if="auth.authenticated" class="grid grid-cols-2 gap-3">
          <div class="rounded-lg border bg-muted/20 px-3 py-2.5 text-xs">
            <p class="text-muted-foreground font-medium mb-1 flex items-center gap-1">
              <CheckCircle2 class="h-3 w-3" />
              Último backup
            </p>
            <p class="font-semibold">{{ lastBackupLabel }}</p>
          </div>
          <div class="rounded-lg border bg-muted/20 px-3 py-2.5 text-xs">
            <p class="text-muted-foreground font-medium mb-1 flex items-center gap-1">
              <Clock class="h-3 w-3" />
              Próximo backup
            </p>
            <p class="font-semibold">{{ nextBackupLabel }}</p>
          </div>
        </div>

        <!-- Manual backup button -->
        <Button
          v-if="auth.authenticated"
          class="w-full gap-2"
          variant="outline"
          :disabled="backingUp"
          @click="hacerBackup"
        >
          <Loader2 v-if="backingUp" class="h-4 w-4 animate-spin" />
          <CloudUpload v-else class="h-4 w-4" />
          {{ backingUp ? "Creando backup…" : "Hacer backup ahora" }}
        </Button>
      </CardContent>
    </Card>

    <!-- Backup list -->
    <Card v-if="auth.authenticated">
      <CardHeader class="pb-3 flex-row items-center justify-between">
        <div>
          <CardTitle class="text-sm font-medium">Historial de backups</CardTitle>
          <CardDescription class="text-xs">Archivos almacenados en la carpeta "goFarmacia Backups" de tu Drive</CardDescription>
        </div>
        <Button size="sm" variant="ghost" class="h-7 w-7 p-0" :disabled="loadingList" @click="cargarBackups">
          <Loader2 v-if="loadingList" class="h-3.5 w-3.5 animate-spin" />
          <RefreshCw v-else class="h-3.5 w-3.5" />
        </Button>
      </CardHeader>
      <CardContent class="p-0">
        <div v-if="loadingList" class="flex justify-center py-10">
          <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
        <div v-else-if="backups.length === 0" class="py-10 text-center">
          <HardDrive class="h-8 w-8 mx-auto mb-2 text-muted-foreground/40" />
          <p class="text-sm text-muted-foreground">Sin backups en Drive todavía</p>
          <p class="text-xs text-muted-foreground mt-0.5">Haz clic en "Hacer backup ahora" para crear el primero</p>
        </div>
        <div v-else class="divide-y">
          <div
            v-for="backup in backups"
            :key="backup.id"
            class="flex items-center justify-between px-4 py-3 hover:bg-muted/30 transition-colors"
          >
            <div class="min-w-0">
              <p class="text-xs font-mono font-medium truncate">{{ backup.name }}</p>
              <p class="text-[11px] text-muted-foreground mt-0.5">
                {{ formatDate(backup.createdAt) }}
                <span class="ml-2">
                  <Badge variant="secondary" class="text-[10px] h-4 px-1.5">{{ formatSize(backup.sizeBytes) }}</Badge>
                </span>
              </p>
            </div>
            <div class="flex items-center gap-1 shrink-0 ml-3">
              <Button
                size="sm"
                variant="ghost"
                class="h-7 w-7 p-0 text-amber-600 hover:bg-amber-50 hover:text-amber-700"
                title="Restaurar desde este backup"
                :disabled="restoringId === backup.id || deletingId === backup.id"
                @click="pedirRestaurar(backup)"
              >
                <Loader2 v-if="restoringId === backup.id" class="h-3.5 w-3.5 animate-spin" />
                <RotateCcw v-else class="h-3.5 w-3.5" />
              </Button>
              <Button
                size="sm"
                variant="ghost"
                class="h-7 w-7 p-0 text-red-500 hover:bg-red-50 hover:text-red-600"
                :disabled="deletingId === backup.id || restoringId === backup.id"
                @click="eliminar(backup.id)"
              >
                <Loader2 v-if="deletingId === backup.id" class="h-3.5 w-3.5 animate-spin" />
                <Trash2 v-else class="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Confirm restore dialog -->
    <AlertDialog v-model:open="restoreOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>¿Restaurar este backup?</AlertDialogTitle>
          <AlertDialogDescription>
            <span class="block">
              Esta acción <strong class="text-red-600">sobrescribirá completamente</strong> la base de datos actual
              con el contenido de:
            </span>
            <code v-if="restoreTarget" class="mt-2 block font-mono text-xs bg-muted px-2 py-1 rounded break-all">
              {{ restoreTarget.name }}
            </code>
            <span class="block mt-2 text-xs">
              Se borrará el esquema actual (<code class="font-mono">public</code>) y se aplicará el dump. No se puede deshacer.
            </span>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction
            class="bg-red-600 hover:bg-red-700 text-white"
            @click="confirmarRestaurar"
          >
            Sí, restaurar
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
