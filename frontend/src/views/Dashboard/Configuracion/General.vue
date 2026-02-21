<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { toast } from "vue-sonner";
import {
  Settings,
  Save,
  Database,
  CheckCircle2,
  XCircle,
  Loader2,
  Eye,
  EyeOff,
  AlertTriangle,
} from "lucide-vue-next";
import { useDBStore } from "@/stores/dbStore";
import { useAuthStore } from "@/stores/auth";

// ── Stores ─────────────────────────────────────────────────────────────────
const dbStore = useDBStore();
const authStore = useAuthStore();
const router = useRouter();

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
    // If server reachable but DB doesn't exist, that's OK — we'll create it
    const isReachable = err.toLowerCase().includes("servidor alcanzable");
    testResult.value = {
      ok: isReachable,
      message: err,
    };
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
    // If was in setup mode, log out so user registers/logs in with real DB
    if (authStore.currentUser?.UUID === "setup-admin") {
      setTimeout(() => {
        authStore.logout();
      }, 1500);
    }
  } else {
    toast.error("Error al configurar base de datos", { description: err });
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
  nombreTienda: "",
  direccion: "",
  telefono: "",
  tasaIVA: 19,
  umbralStockBajo: 10,
});

function loadSettings() {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored) {
    try {
      settings.value = { ...settings.value, ...JSON.parse(stored) };
    } catch (e) {
      console.error("Error al cargar configuración:", e);
    }
  }
}

function saveSettings() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings.value));
  toast.success("Configuración guardada", {
    description: "Los cambios se han aplicado correctamente.",
  });
}

onMounted(() => {
  loadSettings();
  // Sync DSN hint from store (populated by startPolling in DashboardLayout)
  if (!dsn.value && dbStore.dsnHint) {
    dsn.value = dbStore.dsnHint;
  }
});
</script>

<template>
  <div class="p-6 space-y-6 max-w-2xl">
    <!-- Page header -->
    <div>
      <h1 class="text-2xl font-semibold tracking-tight flex items-center gap-2">
        <Settings class="h-5 w-5 text-muted-foreground" />
        Configuración General
      </h1>
      <p class="text-sm text-muted-foreground mt-0.5">
        Conexión a base de datos y parámetros del sistema
      </p>
    </div>

    <!-- ═══ Database connection card ═══════════════════════════════════════ -->
    <Card class="" :class="{ 'border-amber-300': dbStore.setupMode }">
      <CardHeader class="pb-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Database class="h-4 w-4 text-muted-foreground" />
            <CardTitle class="text-base font-semibold">Conexión a PostgreSQL</CardTitle>
          </div>
          <Badge
            variant="outline"
            :class="dbStatusBadge.class"
            class="text-xs px-2 py-0.5"
          >
            <span
              class="h-1.5 w-1.5 rounded-full mr-1.5 inline-block"
              :class="{
                'bg-green-500': dbStore.connected,
                'bg-amber-400': dbStore.setupMode,
                'bg-red-500': !dbStore.connected && !dbStore.setupMode,
              }"
            />
            {{ dbStatusBadge.label }}
          </Badge>
        </div>
        <CardDescription v-if="dbStore.setupMode" class="text-amber-700 mt-1 text-sm">
          <AlertTriangle class="h-3.5 w-3.5 inline mr-1" />
          El sistema está en modo configuración. Ingresa el DSN de tu base de datos PostgreSQL para comenzar.
        </CardDescription>
        <CardDescription v-else-if="dbStore.connected" class="mt-1 text-sm">
          Servidor conectado. Puedes actualizar el DSN y reconectar si es necesario.
        </CardDescription>
      </CardHeader>

      <CardContent class="space-y-4">
        <!-- DSN input -->
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
            <button
              type="button"
              @click="showDSN = !showDSN"
              class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground transition-colors"
              tabindex="-1"
            >
              <EyeOff v-if="showDSN" class="w-4 h-4" />
              <Eye v-else class="w-4 h-4" />
            </button>
          </div>
          <p class="text-xs text-muted-foreground">
            Formato URL:
            <code class="bg-muted px-1 rounded text-[11px]"
              >postgres://user:pass@host:5432/dbname?sslmode=disable</code
            >
            — Si la base de datos no existe, se creará automáticamente.
          </p>
        </div>

        <!-- Test result alert -->
        <Alert
          v-if="testResult"
          :variant="testResult.ok ? 'default' : 'destructive'"
          class="py-2 px-3"
          :class="testResult.ok ? 'border-green-300 bg-green-50 text-green-800' : ''"
        >
          <CheckCircle2 v-if="testResult.ok" class="h-4 w-4 text-green-600" />
          <XCircle v-else class="h-4 w-4" />
          <AlertDescription class="ml-6 text-xs">{{ testResult.message }}</AlertDescription>
        </Alert>

        <!-- Action buttons -->
        <div class="flex gap-2 pt-1">
          <Button
            variant="outline"
            size="sm"
            class="h-9 gap-2"
            :disabled="testLoading || saveLoading || !dsn.trim()"
            @click="handleTestConnection"
          >
            <Loader2 v-if="testLoading" class="h-4 w-4 animate-spin" />
            <CheckCircle2 v-else class="h-4 w-4" />
            Probar conexión
          </Button>

          <Button
            size="sm"
            class="h-9 gap-2"
            :disabled="saveLoading || testLoading || !dsn.trim()"
            @click="handleSaveDB"
          >
            <Loader2 v-if="saveLoading" class="h-4 w-4 animate-spin" />
            <Database v-else class="h-4 w-4" />
            {{ saveLoading ? "Conectando..." : "Guardar y Conectar" }}
          </Button>
        </div>

        <!-- Post-setup notice -->
        <p v-if="dbStore.setupMode" class="text-xs text-muted-foreground border-t pt-3">
          Después de conectar, el sistema cerrará la sesión de configuración. Regístrate o inicia
          sesión con tu cuenta real.
        </p>
      </CardContent>
    </Card>

    <!-- ═══ General settings (disabled in setup mode) ══════════════════════ -->
    <template v-if="!dbStore.setupMode">
      <!-- Datos de la tienda -->
      <Card class="">
        <CardHeader class="pb-4">
          <CardTitle class="text-base font-semibold">Datos de la Tienda</CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="grid gap-2">
            <Label for="nombreTienda">Nombre de la farmacia</Label>
            <Input
              id="nombreTienda"
              v-model="settings.nombreTienda"
              class="h-9"
              placeholder="Ej: Droguería Luna"
            />
          </div>
          <div class="grid gap-2">
            <Label for="direccion">Dirección</Label>
            <Input
              id="direccion"
              v-model="settings.direccion"
              class="h-9"
              placeholder="Dirección del establecimiento"
            />
          </div>
          <div class="grid gap-2">
            <Label for="telefono">Teléfono de contacto</Label>
            <Input
              id="telefono"
              v-model="settings.telefono"
              class="h-9"
              placeholder="Ej: 310 000 0000"
            />
          </div>
        </CardContent>
      </Card>

      <!-- Parámetros del sistema -->
      <Card class="">
        <CardHeader class="pb-4">
          <CardTitle class="text-base font-semibold">Parámetros del Sistema</CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="grid gap-2">
            <Label for="tasaIVA">Tasa de IVA (%)</Label>
            <Input
              id="tasaIVA"
              v-model.number="settings.tasaIVA"
              type="number"
              min="0"
              max="100"
              class="h-9 max-w-[160px]"
            />
            <p class="text-xs text-muted-foreground">Porcentaje aplicado a los productos gravados</p>
          </div>
          <div class="grid gap-2">
            <Label for="umbralStock">Umbral de Stock Bajo</Label>
            <Input
              id="umbralStock"
              v-model.number="settings.umbralStockBajo"
              type="number"
              min="0"
              class="h-9 max-w-[160px]"
            />
            <p class="text-xs text-muted-foreground">
              Productos con stock igual o menor se marcarán en amarillo
            </p>
          </div>
        </CardContent>
      </Card>

      <div class="flex justify-end pt-2">
        <Button @click="saveSettings" class="h-9 gap-2">
          <Save class="w-4 h-4" />Guardar Configuración
        </Button>
      </div>
    </template>
  </div>
</template>
