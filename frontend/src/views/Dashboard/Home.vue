<script setup lang="ts">
import { shallowRef, onMounted, onUnmounted, watch, computed, ref } from "vue";
import {
  ObtenerDatosDashboard,
  ObtenerFechasConVentas,
} from "@/../bridge/go/backend/Db";
import {
  ObtenerResumenCompras,
} from "@/../bridge/go/backend/GmailService";
import { backend } from "@/../bridge/go/models";
import { CalendarDate, today, getLocalTimeZone } from "@internationalized/date";
import { format } from "date-fns";
import { es } from "date-fns/locale";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

import SalesTrendChart from "@/components/dashboard/SalesTrendChart.vue";
import PaymentMethodsChart from "@/components/dashboard/PaymentMethodsChart.vue";
import TopProductosChart from "@/components/dashboard/TopProductosChart.vue";
import TopVendedoresChart from "@/components/dashboard/TopVendedoresChart.vue";
import {
  DollarSign, ShoppingBag, BarChart2, UserCheck, PackageX,
  TrendingUp, AlertCircle, Wallet, Users, Calendar as CalendarIcon,
  Building2, Package, ArrowDownToLine, Zap, Activity,
  Clock, AlertTriangle, ChevronRight, CreditCard,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "vue-sonner";

type DashboardData = backend.DashboardData;
type ResumenCompras = backend.ResumenCompras;

const dashboardData = ref<DashboardData | null>(null);
const resumenCompras = ref<ResumenCompras | null>(null);
const comprasLoading = ref(false);
const isLoading = ref(true);
const error = ref<string | null>(null);
const fechasConVentas = ref<Set<string>>(new Set());
const date = shallowRef<CalendarDate>(today(getLocalTimeZone()));
const now = ref(new Date());

let clockTimer: ReturnType<typeof setInterval> | null = null;

// ── Computed ──────────────────────────────────────────────────────────────────

const formattedButtonDate = computed(() => {
  if (!date.value) return "Selecciona una fecha";
  return format(date.value.toDate(getLocalTimeZone()), "PPP", { locale: es });
});

const formattedTitleDate = computed(() => {
  const sel = date.value.toDate(getLocalTimeZone());
  const tod = new Date();
  sel.setHours(0, 0, 0, 0); tod.setHours(0, 0, 0, 0);
  if (sel.getTime() === tod.getTime()) return "Hoy";
  return format(sel, "d 'de' MMMM 'de' yyyy", { locale: es });
});

// Business hours 7 am – 8 pm (13 h)
const dayProgress = computed(() => {
  const start = 7, end = 20;
  const curr = now.value.getHours() + now.value.getMinutes() / 60;
  if (curr <= start) return 0;
  if (curr >= end) return 100;
  return (curr - start) / (end - start) * 100;
});

const currentTimeStr = computed(() =>
  now.value.toLocaleTimeString("es-CO", { hour: "2-digit", minute: "2-digit" })
);

const topSellerPct = computed(() => {
  const total = dashboardData.value?.totalVentasDia ?? 0;
  const top   = dashboardData.value?.topVendedor?.totalVendido ?? 0;
  if (!total || !top) return 0;
  return Math.round((top / total) * 100);
});

const sinStockCount = computed(() =>
  dashboardData.value?.productosSinStock?.length ?? 0
);

const dominantPayment = computed(() => {
  const ms = dashboardData.value?.metodosPago;
  if (!ms?.length) return null;
  const sorted = [...ms].sort((a, b) => b.monto - a.monto);
  const top = sorted[0];
  const total = sorted.reduce((s, m) => s + m.monto, 0);
  return {
    nombre: top.metodo_pago,
    pct: total > 0 ? Math.round((top.monto / total) * 100) : 0,
  };
});

// Status badge for header
const dayStatus = computed(() => {
  if (sinStockCount.value > 5) return "critical";
  if (sinStockCount.value > 0) return "warn";
  return "ok";
});

const formatCurrency = (v: number) =>
  new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", minimumFractionDigits: 0 }).format(v);

// ── Data loading ──────────────────────────────────────────────────────────────

async function loadDashboardData(fecha: CalendarDate) {
  isLoading.value = true; error.value = null; dashboardData.value = null;
  try {
    dashboardData.value = await ObtenerDatosDashboard(fecha.toString());
  } catch (err: any) {
    error.value = "No se pudieron cargar los datos. " + err;
  } finally {
    isLoading.value = false;
  }
}

async function loadFechasConVentas() {
  try { fechasConVentas.value = new Set(await ObtenerFechasConVentas()); } catch {}
}

async function loadResumenCompras() {
  comprasLoading.value = true;
  try {
    resumenCompras.value = await ObtenerResumenCompras("", "") as ResumenCompras;
  } catch (err) {
    toast.error("Error al cargar resumen de compras", { description: `${err}` });
  } finally {
    comprasLoading.value = false;
  }
}

onMounted(() => {
  clockTimer = setInterval(() => { now.value = new Date(); }, 60_000);
  loadFechasConVentas();
  loadDashboardData(date.value);
  loadResumenCompras();
});

onUnmounted(() => { if (clockTimer) clearInterval(clockTimer); });

watch(date, (d) => { if (d) loadDashboardData(d); });
</script>

<template>
  <div class="p-6 space-y-5">

    <!-- ── Header ─────────────────────────────────────────────────────────── -->
    <div class="space-y-3">
      <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
        <div class="flex items-start gap-3">
          <!-- Status dot -->
          <div class="mt-1 shrink-0">
            <span
              class="inline-flex items-center justify-center w-2.5 h-2.5 rounded-full ring-4 ring-offset-1"
              :class="{
                'bg-emerald-500 ring-emerald-100': dayStatus === 'ok',
                'bg-amber-400  ring-amber-100':   dayStatus === 'warn',
                'bg-red-500    ring-red-100':      dayStatus === 'critical',
              }"
            />
          </div>
          <div>
            <h1 class="text-2xl font-bold tracking-tight leading-none">
              Dashboard —
              <span class="text-primary">{{ formattedTitleDate }}</span>
            </h1>
            <div class="flex items-center gap-2 mt-1">
              <Clock class="h-3 w-3 text-muted-foreground" />
              <span class="text-xs text-muted-foreground">{{ currentTimeStr }}</span>
              <span class="text-muted-foreground/30">·</span>
              <span
                class="text-xs font-medium"
                :class="{
                  'text-emerald-600': dayStatus === 'ok',
                  'text-amber-600':   dayStatus === 'warn',
                  'text-red-600':     dayStatus === 'critical',
                }"
              >
                <template v-if="dayStatus === 'ok'">Operación normal</template>
                <template v-else-if="dayStatus === 'warn'">{{ sinStockCount }} producto(s) agotado(s)</template>
                <template v-else>⚠ {{ sinStockCount }} productos sin stock</template>
              </span>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-2 shrink-0">
          <!-- Date picker -->
          <Popover>
            <PopoverTrigger as-child>
              <Button variant="outline" class="h-8 gap-2 w-[200px] justify-start font-normal text-xs">
                <CalendarIcon class="h-3.5 w-3.5 text-muted-foreground" />
                <span>{{ formattedButtonDate }}</span>
              </Button>
            </PopoverTrigger>
            <PopoverContent class="w-auto p-0">
              <Calendar v-model="date">
                <template #day-cell="{ date: day }">
                  <div class="relative">
                    {{ day.day }}
                    <span
                      v-if="fechasConVentas.has(day.toString())"
                      class="absolute bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full bg-emerald-500"
                    />
                  </div>
                </template>
              </Calendar>
            </PopoverContent>
          </Popover>
        </div>
      </div>

      <!-- Day progress bar -->
      <div class="space-y-1">
        <div class="flex items-center justify-between text-[10px] text-muted-foreground/60">
          <span>07:00</span>
          <span class="font-medium text-muted-foreground">{{ Math.round(dayProgress) }}% del día laboral</span>
          <span>20:00</span>
        </div>
        <div class="h-1 w-full bg-muted rounded-full overflow-hidden">
          <div
            class="h-full rounded-full transition-all duration-1000"
            :class="{
              'bg-emerald-500': dayProgress < 80,
              'bg-amber-400':   dayProgress >= 80 && dayProgress < 95,
              'bg-muted-foreground/30': dayProgress >= 95,
            }"
            :style="`width: ${dayProgress}%`"
          />
        </div>
      </div>
    </div>

    <!-- ── Loading skeletons ────────────────────────────────────────────── -->
    <div v-if="isLoading" class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
      <Card v-for="i in 4" :key="i" class="animate-pulse">
        <CardContent class="p-5">
          <div class="flex justify-between items-start">
            <div class="space-y-2 flex-1">
              <div class="h-3 bg-muted rounded w-3/5" />
              <div class="h-8 bg-muted rounded w-4/5" />
              <div class="h-2.5 bg-muted rounded w-full" />
            </div>
            <div class="h-9 w-9 bg-muted rounded-lg" />
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- ── Error ────────────────────────────────────────────────────────── -->
    <Alert v-if="error" variant="destructive">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error al cargar datos</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- ── Main content ─────────────────────────────────────────────────── -->
    <template v-if="!isLoading && dashboardData">

      <!-- Sin Stock critical banner (only when there are issues) -->
      <div
        v-if="sinStockCount > 0"
        class="flex items-center gap-3 px-4 py-3 rounded-lg border border-red-200 bg-red-50/80 animate-in fade-in-0 duration-300"
      >
        <PackageX class="h-4 w-4 text-red-600 shrink-0" />
        <div class="flex-1 min-w-0">
          <span class="text-sm font-semibold text-red-800">
            {{ sinStockCount }} producto{{ sinStockCount !== 1 ? 's' : '' }} agotado{{ sinStockCount !== 1 ? 's' : '' }}
          </span>
          <span class="text-xs text-red-700/70 ml-2">
            {{ dashboardData.productosSinStock.slice(0, 3).map(p => p.Nombre).join(', ') }}{{ sinStockCount > 3 ? ` y ${sinStockCount - 3} más` : '' }}
          </span>
        </div>
        <span class="shrink-0 text-[10px] font-bold uppercase tracking-wide bg-red-600 text-white px-2 py-0.5 rounded">
          Requiere acción
        </span>
      </div>

      <!-- KPI Cards -->
      <div class="grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">

        <!-- Total Ventas -->
        <div class="animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay: 0ms">
          <Card class="border-l-4 border-l-emerald-500 overflow-hidden">
            <CardContent class="p-5">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <p class="text-xs font-medium text-muted-foreground uppercase tracking-wide">Total Ventas</p>
                  <div class="text-2xl font-bold tracking-tight mt-1 tabular-nums truncate text-emerald-700">
                    {{ formatCurrency(dashboardData.totalVentasDia) }}
                  </div>
                  <p class="text-[11px] text-muted-foreground mt-1.5">Ingresos del día</p>
                </div>
                <div class="h-9 w-9 rounded-lg bg-emerald-500/10 flex items-center justify-center shrink-0">
                  <DollarSign class="h-4.5 w-4.5 text-emerald-600" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <!-- Nº Ventas -->
        <div class="animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay: 75ms">
          <Card class="border-l-4 border-l-blue-500 overflow-hidden">
            <CardContent class="p-5">
              <div class="flex items-start justify-between gap-3">
                <div class="flex-1">
                  <p class="text-xs font-medium text-muted-foreground uppercase tracking-wide">Transacciones</p>
                  <div class="text-2xl font-bold tracking-tight mt-1 tabular-nums text-blue-700">
                    {{ dashboardData.numeroVentasDia }}
                  </div>
                  <p class="text-[11px] text-muted-foreground mt-1.5">
                    Ventas completadas hoy
                  </p>
                </div>
                <div class="h-9 w-9 rounded-lg bg-blue-500/10 flex items-center justify-center shrink-0">
                  <ShoppingBag class="h-4.5 w-4.5 text-blue-600" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <!-- Ticket Promedio -->
        <div class="animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay: 150ms">
          <Card class="border-l-4 border-l-violet-500 overflow-hidden">
            <CardContent class="p-5">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <p class="text-xs font-medium text-muted-foreground uppercase tracking-wide">Ticket Promedio</p>
                  <div class="text-2xl font-bold tracking-tight mt-1 tabular-nums truncate text-violet-700">
                    {{ formatCurrency(dashboardData.ticketPromedioDia) }}
                  </div>
                  <div class="flex items-center gap-1 mt-1.5">
                    <span class="text-[11px] text-muted-foreground">Valor por venta</span>
                    <template v-if="dominantPayment">
                      <span class="text-muted-foreground/30">·</span>
                      <span class="text-[11px] text-violet-600 font-medium">{{ dominantPayment.nombre }} {{ dominantPayment.pct }}%</span>
                    </template>
                  </div>
                </div>
                <div class="h-9 w-9 rounded-lg bg-violet-500/10 flex items-center justify-center shrink-0">
                  <BarChart2 class="h-4.5 w-4.5 text-violet-600" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <!-- Vendedor del día (highlighted) -->
        <div class="animate-in fade-in-0 slide-in-from-bottom-3 duration-500" style="animation-delay: 225ms">
          <Card class="bg-primary text-primary-foreground overflow-hidden relative">
            <div class="absolute inset-0 opacity-5">
              <svg width="100%" height="100%" xmlns="http://www.w3.org/2000/svg">
                <defs><pattern id="dots" x="0" y="0" width="16" height="16" patternUnits="userSpaceOnUse">
                  <circle cx="2" cy="2" r="1.5" fill="white"/>
                </pattern></defs>
                <rect width="100%" height="100%" fill="url(#dots)"/>
              </svg>
            </div>
            <CardContent class="p-5 relative">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <p class="text-xs font-medium text-primary-foreground/70 uppercase tracking-wide">Vendedor del Día</p>
                  <div class="text-lg font-bold tracking-tight mt-1 truncate leading-tight">
                    {{ dashboardData.topVendedor?.nombreCompleto || "N/A" }}
                  </div>
                  <div class="flex items-center gap-1.5 mt-1.5">
                    <span class="text-[11px] text-primary-foreground/70 tabular-nums">
                      {{ formatCurrency(dashboardData.topVendedor?.totalVendido || 0) }}
                    </span>
                    <span v-if="topSellerPct > 0" class="text-[10px] font-bold bg-white/20 px-1.5 py-0.5 rounded-full">
                      {{ topSellerPct }}%
                    </span>
                  </div>
                </div>
                <div class="h-9 w-9 rounded-lg bg-white/15 flex items-center justify-center shrink-0">
                  <UserCheck class="h-4.5 w-4.5 text-primary-foreground" />
                </div>
              </div>
              <!-- Mini seller progress bar -->
              <div v-if="topSellerPct > 0" class="mt-3">
                <div class="h-1 w-full bg-white/20 rounded-full overflow-hidden">
                  <div class="h-full bg-white/70 rounded-full transition-all duration-700"
                       :style="`width: ${topSellerPct}%`" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <!-- Charts Row 1: Flujo (60%) + Top Productos (40%) -->
      <div class="grid gap-3 grid-cols-1 lg:grid-cols-5 animate-in fade-in-0 duration-500" style="animation-delay: 300ms">
        <Card class="lg:col-span-3">
          <CardHeader class="pb-2 pt-4 px-5">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold flex items-center gap-2">
                <Activity class="h-3.5 w-3.5 text-primary" />
                Flujo de Ventas por Hora
              </CardTitle>
              <span class="text-[10px] text-muted-foreground/60 bg-muted px-2 py-0.5 rounded-full">
                Ingresos · Transacciones
              </span>
            </div>
          </CardHeader>
          <CardContent class="h-[240px] px-5 pb-4">
            <SalesTrendChart :chart-data="dashboardData.ventasIndividuales" />
          </CardContent>
        </Card>

        <Card class="lg:col-span-2">
          <CardHeader class="pb-2 pt-4 px-5">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold flex items-center gap-2">
                <Zap class="h-3.5 w-3.5 text-emerald-600" />
                Top Productos
              </CardTitle>
              <span class="text-[10px] text-muted-foreground/60 bg-muted px-2 py-0.5 rounded-full">
                unidades
              </span>
            </div>
          </CardHeader>
          <CardContent class="h-[240px] px-5 pb-4">
            <TopProductosChart :chart-data="dashboardData.topProductos" />
          </CardContent>
        </Card>
      </div>

      <!-- Charts Row 2: Métodos de Pago + Vendedores -->
      <div class="grid gap-3 grid-cols-1 lg:grid-cols-2 animate-in fade-in-0 duration-500" style="animation-delay: 375ms">
        <Card>
          <CardHeader class="pb-2 pt-4 px-5">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold flex items-center gap-2">
                <CreditCard class="h-3.5 w-3.5 text-muted-foreground" />
                Métodos de Pago
              </CardTitle>
              <span v-if="dominantPayment" class="text-[10px] font-semibold text-violet-700 bg-violet-50 px-2 py-0.5 rounded-full border border-violet-200">
                {{ dominantPayment.nombre }} lidera
              </span>
            </div>
          </CardHeader>
          <CardContent class="h-[200px] px-5 pb-4">
            <PaymentMethodsChart :chart-data="dashboardData.metodosPago" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2 pt-4 px-5">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold flex items-center gap-2">
                <Users class="h-3.5 w-3.5 text-primary" />
                Rendimiento de Vendedores
              </CardTitle>
              <span class="text-[10px] text-muted-foreground/60 bg-muted px-2 py-0.5 rounded-full">
                Top 5
              </span>
            </div>
          </CardHeader>
          <CardContent class="h-[200px] px-5 pb-4">
            <TopVendedoresChart :chart-data="dashboardData.topVendedoresDia" />
          </CardContent>
        </Card>
      </div>

      <!-- Productos Sin Stock (full detail) -->
      <Card class="animate-in fade-in-0 duration-500" style="animation-delay: 450ms">
        <CardHeader class="pb-3 pt-4 px-5">
          <div class="flex items-center justify-between">
            <CardTitle class="text-sm font-semibold flex items-center gap-2"
              :class="sinStockCount > 0 ? 'text-destructive' : 'text-muted-foreground'">
              <PackageX class="h-3.5 w-3.5" />
              Productos Sin Stock
              <span v-if="sinStockCount > 0"
                class="text-[10px] font-bold bg-red-600 text-white px-1.5 py-0.5 rounded-full ml-1">
                {{ sinStockCount }}
              </span>
            </CardTitle>
          </div>
        </CardHeader>
        <CardContent class="px-5 pb-4">
          <div v-if="dashboardData.productosSinStock?.length"
            class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            <div
              v-for="p in dashboardData.productosSinStock"
              :key="p.UUID"
              class="flex items-center justify-between px-3 py-2.5 rounded-md border border-red-200 bg-red-50/60 group hover:bg-red-50 transition-colors"
            >
              <div class="min-w-0">
                <p class="text-xs font-semibold truncate text-red-900">{{ p.Nombre }}</p>
                <p class="font-mono text-[10px] text-red-700/60">{{ p.Codigo }}</p>
              </div>
              <span class="shrink-0 text-[10px] font-bold border border-red-300 text-red-700 px-2 py-0.5 rounded ml-3">
                0 uds
              </span>
            </div>
          </div>
          <div v-else class="flex items-center gap-2 py-3 text-emerald-700">
            <div class="h-6 w-6 rounded-full bg-emerald-100 flex items-center justify-center shrink-0">
              <svg class="h-3.5 w-3.5 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <span class="text-sm font-medium">¡Todo en orden! Sin faltantes de stock.</span>
          </div>
        </CardContent>
      </Card>
    </template>

    <!-- ── COMPRAS SECTION ──────────────────────────────────────────────────── -->
    <template v-if="resumenCompras || comprasLoading">
      <!-- Section divider -->
      <div class="flex items-center gap-3 pt-1">
        <div class="h-px flex-1 bg-border" />
        <div class="flex items-center gap-2 text-muted-foreground">
          <ArrowDownToLine class="h-3.5 w-3.5" />
          <span class="text-xs font-semibold uppercase tracking-wider">Historial de Compras</span>
        </div>
        <div class="h-px flex-1 bg-border" />
      </div>

      <!-- Loading skeleton -->
      <div v-if="comprasLoading" class="grid gap-3 grid-cols-1 sm:grid-cols-3">
        <Card v-for="i in 3" :key="i" class="animate-pulse">
          <CardContent class="p-5">
            <div class="space-y-2">
              <div class="h-3 bg-muted rounded w-3/5" />
              <div class="h-7 bg-muted rounded w-4/5" />
              <div class="h-2.5 bg-muted rounded w-full" />
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Empty state -->
      <div v-else-if="resumenCompras && resumenCompras.NumFacturas === 0"
        class="border rounded-lg px-6 py-8 text-center bg-muted/20">
        <ArrowDownToLine class="h-8 w-8 text-muted-foreground/40 mx-auto mb-2" />
        <p class="text-sm font-medium text-muted-foreground">Sin facturas de compra importadas</p>
        <p class="text-xs text-muted-foreground mt-1">
          Ve a <strong>Compras → Facturas Electrónicas</strong> y conecta Gmail para sincronizar.
        </p>
      </div>

      <!-- KPIs -->
      <div v-else-if="resumenCompras" class="grid gap-3 grid-cols-1 sm:grid-cols-3">
        <Card class="border-l-4 border-l-orange-500">
          <CardContent class="p-5">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <p class="text-xs font-medium text-muted-foreground uppercase tracking-wide">Total Comprado</p>
                <div class="text-2xl font-bold tracking-tight mt-1 tabular-nums truncate text-orange-700">
                  {{ formatCurrency(resumenCompras.TotalGastado) }}
                </div>
                <p class="text-[11px] text-muted-foreground mt-1.5">Neto: facturas − notas crédito</p>
              </div>
              <div class="h-9 w-9 rounded-lg bg-orange-500/10 flex items-center justify-center shrink-0">
                <DollarSign class="h-4.5 w-4.5 text-orange-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card class="border-l-4 border-l-blue-400">
          <CardContent class="p-5">
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1">
                <p class="text-xs font-medium text-muted-foreground uppercase tracking-wide">Documentos DIAN</p>
                <div class="text-2xl font-bold tracking-tight mt-1 tabular-nums text-blue-700">
                  {{ resumenCompras.NumFacturas }}
                </div>
                <div class="flex gap-1.5 mt-1.5 flex-wrap">
                  <span class="text-[10px] px-1.5 py-0.5 rounded-full border bg-blue-50 text-blue-700 border-blue-200 font-semibold">
                    {{ resumenCompras.NumFacturas }} fac.
                  </span>
                  <span v-if="(resumenCompras.NumNotasCredito ?? 0) > 0"
                    class="text-[10px] px-1.5 py-0.5 rounded-full border bg-amber-50 text-amber-700 border-amber-200 font-semibold">
                    {{ resumenCompras.NumNotasCredito }} NC
                  </span>
                  <span v-if="(resumenCompras.NumNotasDebito ?? 0) > 0"
                    class="text-[10px] px-1.5 py-0.5 rounded-full border bg-orange-50 text-orange-700 border-orange-200 font-semibold">
                    {{ resumenCompras.NumNotasDebito }} ND
                  </span>
                </div>
              </div>
              <div class="h-9 w-9 rounded-lg bg-blue-500/10 flex items-center justify-center shrink-0">
                <ShoppingBag class="h-4.5 w-4.5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card class="border-l-4 border-l-violet-500">
          <CardContent class="p-5">
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1">
                <p class="text-xs font-medium text-muted-foreground uppercase tracking-wide">Proveedores</p>
                <div class="text-2xl font-bold tracking-tight mt-1 tabular-nums text-violet-700">
                  {{ resumenCompras.NumProveedores }}
                </div>
                <p class="text-[11px] text-muted-foreground mt-1.5">Distintos en historial</p>
              </div>
              <div class="h-9 w-9 rounded-lg bg-violet-500/10 flex items-center justify-center shrink-0">
                <Building2 class="h-4.5 w-4.5 text-violet-600" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Top Products + Suppliers -->
      <div v-if="resumenCompras && resumenCompras.NumFacturas > 0" class="grid gap-3 grid-cols-1 lg:grid-cols-2">
        <!-- Top productos comprados -->
        <Card>
          <CardHeader class="pb-1 pt-4 px-5">
            <CardTitle class="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex items-center gap-2">
              <Package class="h-3.5 w-3.5 text-orange-500" />
              Top productos comprados
            </CardTitle>
          </CardHeader>
          <CardContent class="px-5 pb-4 pt-2">
            <div v-if="resumenCompras.TopProductos?.length" class="space-y-0 divide-y">
              <div v-for="(p, i) in resumenCompras.TopProductos.slice(0, 8)" :key="i"
                class="flex items-center gap-3 py-2.5 hover:bg-muted/30 -mx-2 px-2 rounded transition-colors">
                <!-- Rank circle -->
                <span class="w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold shrink-0"
                  :class="i === 0 ? 'bg-orange-100 text-orange-700' : i === 1 ? 'bg-muted text-muted-foreground' : 'text-muted-foreground/50'">
                  {{ i + 1 }}
                </span>
                <div class="flex-1 min-w-0">
                  <p class="text-xs font-semibold truncate">{{ p.Descripcion }}</p>
                  <p class="text-[10px] text-muted-foreground">
                    {{ p.TotalCantidad }} uds · {{ p.NumFacturas }} fac.
                  </p>
                </div>
                <span class="text-xs font-bold tabular-nums shrink-0 text-orange-700">
                  {{ formatCurrency(p.TotalComprado) }}
                </span>
              </div>
            </div>
            <p v-else class="text-xs text-muted-foreground py-4 text-center">Sin datos</p>
          </CardContent>
        </Card>

        <!-- Top proveedores -->
        <Card>
          <CardHeader class="pb-1 pt-4 px-5">
            <CardTitle class="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex items-center gap-2">
              <Building2 class="h-3.5 w-3.5 text-violet-500" />
              Top proveedores
            </CardTitle>
          </CardHeader>
          <CardContent class="px-5 pb-4 pt-2">
            <div v-if="resumenCompras.TopProveedores?.length" class="space-y-0 divide-y">
              <div v-for="(prov, i) in resumenCompras.TopProveedores" :key="i"
                class="flex items-center gap-3 py-2.5 hover:bg-muted/30 -mx-2 px-2 rounded transition-colors">
                <span class="w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold shrink-0"
                  :class="i === 0 ? 'bg-violet-100 text-violet-700' : i === 1 ? 'bg-muted text-muted-foreground' : 'text-muted-foreground/50'">
                  {{ i + 1 }}
                </span>
                <div class="flex-1 min-w-0">
                  <p class="text-xs font-semibold truncate">{{ prov.Nombre }}</p>
                  <p class="text-[10px] text-muted-foreground font-mono">
                    NIT {{ prov.NIT || "—" }} · {{ prov.TotalFacturas }} fac.
                  </p>
                </div>
                <span class="text-xs font-bold tabular-nums shrink-0 text-violet-700">
                  {{ formatCurrency(prov.TotalComprado) }}
                </span>
              </div>
            </div>
            <p v-else class="text-xs text-muted-foreground py-4 text-center">Sin datos</p>
          </CardContent>
        </Card>
      </div>
    </template>

  </div>
</template>
