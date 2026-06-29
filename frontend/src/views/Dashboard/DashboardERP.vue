<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import {
  ObtenerResumenInventario,
  ObtenerDatosDashboard,
} from "@/../bridge/go/backend/Db";
import {
  Package,
  AlertTriangle,
  PackageX,
  DollarSign,
  TrendingUp,
  AlertCircle,
  ShieldCheck,
  ShieldAlert,
  ShieldX,
  RefreshCw,
  Layers,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

interface ProductoAlerta {
  uuid: string;
  nombre: string;
  codigo: string;
  stock: number;
  precioVenta: number;
}

interface ResumenInventario {
  totalProductos: number;
  productosStockBajo: number;
  productosSinStock: number;
  valorInventario: number;
  productosAlerta: ProductoAlerta[];
}

const resumen = ref<ResumenInventario | null>(null);
const ventasDia = ref<number>(0);
const isLoading = ref(true);
const isRefreshing = ref(false);
const error = ref<string | null>(null);

const formatCurrency = (value: number) =>
  new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(value);

const healthScore = computed(() => {
  if (!resumen.value || resumen.value.totalProductos === 0) return 100;
  const { totalProductos, productosStockBajo, productosSinStock } = resumen.value;
  const lowW = (productosStockBajo / totalProductos) * 30;
  const outW = (productosSinStock / totalProductos) * 60;
  return Math.max(0, Math.round(100 - lowW - outW));
});

const healthStatus = computed(() => {
  const s = healthScore.value;
  if (s >= 80) return { label: "Óptimo", colorClass: "text-emerald-600", borderClass: "border-l-emerald-500", bgClass: "bg-emerald-50 dark:bg-emerald-950/30", barClass: "bg-emerald-500" };
  if (s >= 60) return { label: "Precaución", colorClass: "text-amber-600", borderClass: "border-l-amber-500", bgClass: "bg-amber-50 dark:bg-amber-950/30", barClass: "bg-amber-500" };
  return { label: "Crítico", colorClass: "text-red-600", borderClass: "border-l-red-500", bgClass: "bg-red-50 dark:bg-red-950/30", barClass: "bg-red-500" };
});

const stockDistribution = computed(() => {
  if (!resumen.value || resumen.value.totalProductos === 0) return { ok: 100, low: 0, out: 0 };
  const { totalProductos, productosStockBajo, productosSinStock } = resumen.value;
  const ok = Math.round(((totalProductos - productosStockBajo - productosSinStock) / totalProductos) * 100);
  const low = Math.round((productosStockBajo / totalProductos) * 100);
  const out = 100 - ok - low;
  return { ok, low, out };
});

async function loadData(refresh = false) {
  if (refresh) isRefreshing.value = true;
  else isLoading.value = true;
  error.value = null;
  try {
    const [inv, dashboard] = await Promise.all([
      ObtenerResumenInventario(),
      ObtenerDatosDashboard(""),
    ]);
    resumen.value = inv;
    ventasDia.value = dashboard.totalVentasDia;
  } catch (err: any) {
    error.value = "No se pudieron cargar los datos. " + err;
  } finally {
    isLoading.value = false;
    isRefreshing.value = false;
  }
}

onMounted(() => loadData());
</script>

<template>
  <div class="p-6 space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold tracking-tight">Dashboard ERP</h1>
        <p class="text-xs text-muted-foreground mt-0.5 uppercase tracking-wide">
          Resumen operativo · Inventario &amp; Ventas
        </p>
      </div>
      <Button variant="ghost" size="sm" class="h-8 gap-1.5 text-xs" :disabled="isRefreshing" @click="loadData(true)">
        <RefreshCw class="h-3.5 w-3.5" :class="{ 'animate-spin': isRefreshing }" />
        Actualizar
      </Button>
    </div>

    <!-- Error -->
    <Alert v-if="error" variant="destructive">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error al cargar datos</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Loading skeletons -->
    <template v-if="isLoading">
      <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
        <div v-for="i in 4" :key="i" class="h-24 rounded-lg border bg-card animate-pulse" />
      </div>
      <div class="h-32 rounded-lg border bg-card animate-pulse" />
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-3">
        <div class="lg:col-span-2 h-64 rounded-lg border bg-card animate-pulse" />
        <div class="h-64 rounded-lg border bg-card animate-pulse" />
      </div>
    </template>

    <!-- Content -->
    <template v-if="!isLoading && resumen">
      <!-- KPI row -->
      <div class="grid gap-3 grid-cols-2 lg:grid-cols-4">
        <!-- Total Productos -->
        <Card class="border-l-4 border-l-blue-500 animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay:0ms">
          <CardContent class="p-4">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">Productos</p>
            <div class="text-2xl font-bold tabular-nums mt-1 text-blue-600">{{ resumen.totalProductos }}</div>
            <div class="flex items-center gap-1 mt-1.5">
              <Package class="h-3 w-3 text-muted-foreground" />
              <span class="text-[11px] text-muted-foreground">activos en catálogo</span>
            </div>
          </CardContent>
        </Card>

        <!-- Stock Bajo -->
        <Card class="border-l-4 border-l-amber-500 animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay:60ms">
          <CardContent class="p-4">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">Stock Bajo</p>
            <div class="text-2xl font-bold tabular-nums mt-1 text-amber-600">{{ resumen.productosStockBajo }}</div>
            <div class="flex items-center gap-1 mt-1.5">
              <AlertTriangle class="h-3 w-3 text-amber-500" />
              <span class="text-[11px] text-muted-foreground">≤ 10 unidades</span>
            </div>
          </CardContent>
        </Card>

        <!-- Sin Stock -->
        <Card class="border-l-4 border-l-red-500 animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay:120ms">
          <CardContent class="p-4">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">Sin Stock</p>
            <div class="text-2xl font-bold tabular-nums mt-1 text-red-600">{{ resumen.productosSinStock }}</div>
            <div class="flex items-center gap-1 mt-1.5">
              <PackageX class="h-3 w-3 text-red-500" />
              <span class="text-[11px] text-muted-foreground">agotados</span>
            </div>
          </CardContent>
        </Card>

        <!-- Valor Inventario -->
        <Card class="border-l-4 border-l-emerald-500 animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay:180ms">
          <CardContent class="p-4">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">Valor Inv.</p>
            <div class="text-xl font-bold tabular-nums mt-1 text-emerald-600 truncate">{{ formatCurrency(resumen.valorInventario) }}</div>
            <div class="flex items-center gap-1 mt-1.5">
              <DollarSign class="h-3 w-3 text-emerald-500" />
              <span class="text-[11px] text-muted-foreground">precio venta</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Health score + distribution bar -->
      <Card
        class="border-l-4 animate-in fade-in-0 slide-in-from-bottom-3 duration-500"
        :class="[healthStatus.borderClass, healthStatus.bgClass]"
        style="animation-delay:240ms"
      >
        <CardContent class="p-4">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <ShieldCheck v-if="healthScore >= 80" class="h-4 w-4 text-emerald-600" />
              <ShieldAlert v-else-if="healthScore >= 60" class="h-4 w-4 text-amber-600" />
              <ShieldX v-else class="h-4 w-4 text-red-600" />
              <span class="text-sm font-semibold" :class="healthStatus.colorClass">
                Salud del Inventario — {{ healthStatus.label }}
              </span>
            </div>
            <span class="text-2xl font-bold tabular-nums" :class="healthStatus.colorClass">
              {{ healthScore }}<span class="text-sm font-normal opacity-60">%</span>
            </span>
          </div>

          <!-- Stacked distribution bar -->
          <div class="flex rounded-full overflow-hidden h-2.5 bg-muted">
            <div
              class="bg-emerald-500 transition-all duration-700"
              :style="{ width: stockDistribution.ok + '%' }"
            />
            <div
              class="bg-amber-400 transition-all duration-700"
              :style="{ width: stockDistribution.low + '%' }"
            />
            <div
              class="bg-red-500 transition-all duration-700"
              :style="{ width: stockDistribution.out + '%' }"
            />
          </div>

          <!-- Legend -->
          <div class="flex items-center gap-5 mt-2.5 text-[11px] text-muted-foreground">
            <span class="flex items-center gap-1.5">
              <span class="inline-block h-2 w-2 rounded-full bg-emerald-500" />
              Normal {{ stockDistribution.ok }}%
            </span>
            <span class="flex items-center gap-1.5">
              <span class="inline-block h-2 w-2 rounded-full bg-amber-400" />
              Stock bajo {{ stockDistribution.low }}%
            </span>
            <span class="flex items-center gap-1.5">
              <span class="inline-block h-2 w-2 rounded-full bg-red-500" />
              Agotado {{ stockDistribution.out }}%
            </span>
          </div>
        </CardContent>
      </Card>

      <!-- Bottom: alert list + ventas -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-3">
        <!-- Productos que requieren atención -->
        <Card
          class="lg:col-span-2 animate-in fade-in-0 slide-in-from-bottom-3 duration-500"
          style="animation-delay:300ms"
        >
          <CardHeader class="pb-2 pt-4 px-4">
            <CardTitle class="text-sm font-semibold flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Layers class="h-4 w-4 text-muted-foreground" />
                <span>Productos en Alerta</span>
              </div>
              <Badge v-if="resumen.productosAlerta.length" variant="destructive" class="text-[10px] h-5 px-1.5">
                {{ resumen.productosAlerta.length }}
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent class="px-4 pb-4">
            <div v-if="!resumen.productosAlerta.length" class="py-8 text-center text-sm text-muted-foreground">
              Sin productos en alerta
            </div>
            <div v-else class="space-y-1 max-h-64 overflow-y-auto pr-1">
              <div
                v-for="p in resumen.productosAlerta"
                :key="p.uuid"
                class="flex items-center gap-3 py-1.5 px-2 rounded hover:bg-muted/50 transition-colors"
              >
                <!-- Stock badge -->
                <span
                  class="shrink-0 min-w-[2rem] text-center text-[11px] font-bold tabular-nums px-1.5 py-0.5 rounded"
                  :class="p.stock === 0
                    ? 'bg-red-100 text-red-700 dark:bg-red-950/50 dark:text-red-400'
                    : 'bg-amber-100 text-amber-700 dark:bg-amber-950/50 dark:text-amber-400'"
                >
                  {{ p.stock }}
                </span>
                <!-- Name + code -->
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-medium truncate leading-tight">{{ p.nombre }}</p>
                  <p class="text-[10px] text-muted-foreground font-mono">{{ p.codigo }}</p>
                </div>
                <!-- Price -->
                <span class="shrink-0 text-[11px] tabular-nums text-muted-foreground">
                  {{ formatCurrency(p.precioVenta) }}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Ventas del día -->
        <Card
          class="border-l-4 border-l-violet-500 animate-in fade-in-0 slide-in-from-bottom-3 duration-500"
          style="animation-delay:360ms"
        >
          <CardHeader class="pb-2 pt-4 px-4">
            <CardTitle class="text-sm font-semibold flex items-center gap-2">
              <TrendingUp class="h-4 w-4 text-violet-600" />
              Ventas del Día
            </CardTitle>
          </CardHeader>
          <CardContent class="px-4 pb-4">
            <div class="text-2xl font-bold tabular-nums text-violet-600 mt-1 truncate">
              {{ formatCurrency(ventasDia) }}
            </div>
            <p class="text-[11px] text-muted-foreground mt-1">Total registrado hoy</p>

            <!-- Spacer + ratio section -->
            <div class="mt-5 pt-4 border-t space-y-2">
              <div class="flex justify-between items-center text-xs">
                <span class="text-muted-foreground uppercase tracking-wide">Necesitan atención</span>
                <span class="font-semibold tabular-nums text-amber-600">
                  {{ resumen.productosStockBajo + resumen.productosSinStock }}
                </span>
              </div>
              <div class="flex justify-between items-center text-xs">
                <span class="text-muted-foreground uppercase tracking-wide">% del catálogo</span>
                <span class="font-semibold tabular-nums" :class="healthStatus.colorClass">
                  {{ 100 - stockDistribution.ok }}%
                </span>
              </div>
              <div class="flex justify-between items-center text-xs">
                <span class="text-muted-foreground uppercase tracking-wide">Estado</span>
                <Badge
                  class="text-[10px] h-5 px-1.5"
                  :class="healthScore >= 80
                    ? 'bg-emerald-100 text-emerald-700 border-emerald-200 dark:bg-emerald-950/50 dark:text-emerald-400'
                    : healthScore >= 60
                      ? 'bg-amber-100 text-amber-700 border-amber-200 dark:bg-amber-950/50 dark:text-amber-400'
                      : 'bg-red-100 text-red-700 border-red-200 dark:bg-red-950/50 dark:text-red-400'"
                  variant="outline"
                >
                  {{ healthStatus.label }}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
