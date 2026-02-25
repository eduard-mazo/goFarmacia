<script setup lang="ts">
import { shallowRef, onMounted, watch, computed, ref } from "vue";
import {
  ObtenerDatosDashboard,
  ObtenerFechasConVentas,
} from "@/../wailsjs/go/backend/Db";
import { ObtenerResumenCompras } from "@/../wailsjs/go/backend/GmailService";
import { backend } from "@/../wailsjs/go/models";
import { CalendarDate, today, getLocalTimeZone } from "@internationalized/date";
import { format } from "date-fns";
import { es } from "date-fns/locale";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

import SalesTrendChart from "@/components/dashboard/SalesTrendChart.vue";
import PaymentMethodsChart from "@/components/dashboard/PaymentMethodsChart.vue";
import TopProductosChart from "@/components/dashboard/TopProductosChart.vue";
import TopVendedoresChart from "@/components/dashboard/TopVendedoresChart.vue";
import {
  DollarSign,
  ShoppingBag,
  BarChart2,
  UserCheck,
  PackageX,
  TrendingUp,
  AlertCircle,
  Wallet,
  Users,
  Calendar as CalendarIcon,
  Building2,
  Package,
  ArrowDownToLine,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type DashboardData = backend.DashboardData;
type ResumenCompras = backend.ResumenCompras;

const dashboardData = ref<DashboardData | null>(null);
const resumenCompras = ref<ResumenCompras | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);
const fechasConVentas = ref<Set<string>>(new Set());

const date = shallowRef<CalendarDate>(today(getLocalTimeZone()));

const formattedButtonDate = computed(() => {
  if (!date.value) return "Selecciona una fecha";
  return format(date.value.toDate(getLocalTimeZone()), "PPP", { locale: es });
});

const formattedTitleDate = computed(() => {
  const selectedDate = date.value.toDate(getLocalTimeZone());
  const todayDate = new Date();
  selectedDate.setHours(0, 0, 0, 0);
  todayDate.setHours(0, 0, 0, 0);
  if (selectedDate.getTime() === todayDate.getTime()) return "Hoy";
  return format(selectedDate, "d 'de' MMMM 'de' yyyy", { locale: es });
});

const formatCurrency = (value: number) =>
  new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(value);

async function loadDashboardData(fecha: CalendarDate) {
  isLoading.value = true;
  error.value = null;
  dashboardData.value = null;
  try {
    dashboardData.value = await ObtenerDatosDashboard(fecha.toString());
  } catch (err: any) {
    error.value = "No se pudieron cargar los datos. " + err;
  } finally {
    isLoading.value = false;
  }
}

async function loadFechasConVentas() {
  try {
    const dates = await ObtenerFechasConVentas();
    fechasConVentas.value = new Set(dates);
  } catch {}
}

async function loadResumenCompras() {
  try {
    // Last 30 days by default
    const hasta = new Date();
    const desde = new Date();
    desde.setDate(desde.getDate() - 30);
    const fmt = (d: Date) => d.toISOString().split("T")[0];
    resumenCompras.value = await ObtenerResumenCompras(fmt(desde), fmt(hasta)) as ResumenCompras;
  } catch { /* no purchase data — show nothing */ }
}

onMounted(() => {
  loadFechasConVentas();
  loadDashboardData(date.value);
  loadResumenCompras();
});

watch(date, (newDate) => {
  if (newDate) loadDashboardData(newDate);
});
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- Page header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">
          Dashboard —
          <span class="text-primary">{{ formattedTitleDate }}</span>
        </h1>
        <p class="text-sm text-muted-foreground mt-0.5">
          Resumen de ventas y actividad del día
        </p>
      </div>

      <Popover>
        <PopoverTrigger as-child>
          <Button variant="outline" class="h-9 gap-2 w-[220px] justify-start font-normal">
            <CalendarIcon class="h-4 w-4 text-muted-foreground" />
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
                ></span>
              </div>
            </template>
          </Calendar>
        </PopoverContent>
      </Popover>
    </div>

    <!-- Loading skeletons -->
    <div v-if="isLoading" class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
      <Card v-for="i in 4" :key="i" class="animate-pulse">
        <CardContent class="p-6">
          <div class="flex justify-between items-start">
            <div class="space-y-2 flex-1">
              <div class="h-3.5 bg-muted rounded w-3/5"></div>
              <div class="h-7 bg-muted rounded w-4/5"></div>
              <div class="h-3 bg-muted rounded w-full"></div>
            </div>
            <div class="h-10 w-10 bg-muted rounded-xl"></div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Error state -->
    <Alert v-if="error" variant="destructive">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error al cargar datos</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Content -->
    <template v-if="!isLoading && dashboardData">
      <!-- Stat cards -->
      <div class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
        <!-- Total Ventas -->
        <Card class="">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-muted-foreground">Total Ventas</p>
                <div class="text-2xl font-bold tracking-tight mt-1 truncate">
                  {{ formatCurrency(dashboardData.totalVentasDia) }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Ventas totales del día</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-emerald-500/10 flex items-center justify-center shrink-0">
                <DollarSign class="h-5 w-5 text-emerald-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Nº Ventas -->
        <Card class="">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Nº de Ventas</p>
                <div class="text-2xl font-bold tracking-tight mt-1">
                  {{ dashboardData.numeroVentasDia }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Transacciones completadas</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-blue-500/10 flex items-center justify-center shrink-0">
                <ShoppingBag class="h-5 w-5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Ticket Promedio -->
        <Card class="">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-muted-foreground">Ticket Promedio</p>
                <div class="text-2xl font-bold tracking-tight mt-1 truncate">
                  {{ formatCurrency(dashboardData.ticketPromedioDia) }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Valor promedio por venta</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-violet-500/10 flex items-center justify-center shrink-0">
                <BarChart2 class="h-5 w-5 text-violet-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Vendedor del día (highlighted) -->
        <Card class="bg-primary text-primary-foreground">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-primary-foreground/70">Vendedor del Día</p>
                <div class="text-lg font-bold tracking-tight mt-1 truncate">
                  {{ dashboardData.topVendedor?.nombreCompleto || "N/A" }}
                </div>
                <p class="text-xs text-primary-foreground/70 mt-1">
                  {{ formatCurrency(dashboardData.topVendedor?.totalVendido || 0) }} en ventas
                </p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-white/15 flex items-center justify-center shrink-0">
                <UserCheck class="h-5 w-5 text-primary-foreground" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Row 2: Flujo de ventas (60%) + Top Productos (40%) -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-5">
        <Card class="lg:col-span-3">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <TrendingUp class="h-4 w-4 text-primary" />
              Flujo de Ventas por Hora
            </CardTitle>
            <p class="text-xs text-muted-foreground">Ingresos (área) · Transacciones (línea punteada)</p>
          </CardHeader>
          <CardContent class="h-[260px]">
            <SalesTrendChart :chart-data="dashboardData.ventasIndividuales" />
          </CardContent>
        </Card>

        <Card class="lg:col-span-2">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <BarChart2 class="h-4 w-4 text-emerald-600" />
              Top Productos Vendidos
            </CardTitle>
            <p class="text-xs text-muted-foreground">Unidades despachadas en el día</p>
          </CardHeader>
          <CardContent class="h-[260px]">
            <TopProductosChart :chart-data="dashboardData.topProductos" />
          </CardContent>
        </Card>
      </div>

      <!-- Row 3: Métodos de Pago + Rendimiento Vendedores -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-2">
        <Card>
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <Wallet class="h-4 w-4 text-muted-foreground" />
              Métodos de Pago
            </CardTitle>
            <p class="text-xs text-muted-foreground">Monto total por forma de pago</p>
          </CardHeader>
          <CardContent class="h-[220px]">
            <PaymentMethodsChart :chart-data="dashboardData.metodosPago" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <Users class="h-4 w-4 text-primary" />
              Rendimiento de Vendedores
            </CardTitle>
            <p class="text-xs text-muted-foreground">Top 5 por monto vendido — hover para ver transacciones</p>
          </CardHeader>
          <CardContent class="h-[220px]">
            <TopVendedoresChart :chart-data="dashboardData.topVendedoresDia" />
          </CardContent>
        </Card>
      </div>

      <!-- Row 4: Productos Sin Stock -->
      <Card>
        <CardHeader class="pb-3">
          <CardTitle class="text-base font-semibold flex items-center gap-2 text-destructive">
            <PackageX class="h-4 w-4" />
            Productos Sin Stock
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div v-if="dashboardData.productosSinStock?.length"
            class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            <div
              v-for="p in dashboardData.productosSinStock"
              :key="p.UUID"
              class="flex items-center justify-between px-3 py-2 rounded-md border bg-red-50/50"
            >
              <div class="min-w-0">
                <p class="text-sm font-medium truncate">{{ p.Nombre }}</p>
                <p class="font-mono text-xs text-muted-foreground">{{ p.Codigo }}</p>
              </div>
              <span class="shrink-0 text-xs font-semibold bg-red-100 text-red-700 px-2 py-0.5 rounded-full ml-3">
                Agotado
              </span>
            </div>
          </div>
          <p v-else class="text-sm text-muted-foreground py-4 text-center">
            ¡Todo en orden! Sin faltantes.
          </p>
        </CardContent>
      </Card>
    </template>

    <!-- ───── COMPRAS SECTION ──────────────────────────────────────────────── -->
    <template v-if="resumenCompras && (resumenCompras.NumFacturas > 0)">
      <!-- Section header -->
      <div class="flex items-center gap-2 pt-2">
        <ArrowDownToLine class="h-4 w-4 text-primary" />
        <h2 class="text-base font-semibold tracking-tight">Compras — últimos 30 días</h2>
      </div>

      <!-- KPI cards -->
      <div class="grid gap-4 grid-cols-1 sm:grid-cols-3">
        <Card>
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-muted-foreground">Total comprado</p>
                <div class="text-2xl font-bold tracking-tight mt-1 truncate">
                  {{ formatCurrency(resumenCompras.TotalGastado) }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Monto total en facturas DIAN</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-orange-500/10 flex items-center justify-center shrink-0">
                <DollarSign class="h-5 w-5 text-orange-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Facturas recibidas</p>
                <div class="text-2xl font-bold tracking-tight mt-1">
                  {{ resumenCompras.NumFacturas }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Importadas desde Gmail</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-blue-500/10 flex items-center justify-center shrink-0">
                <ShoppingBag class="h-5 w-5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Proveedores activos</p>
                <div class="text-2xl font-bold tracking-tight mt-1">
                  {{ resumenCompras.NumProveedores }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Con compras en el período</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-violet-500/10 flex items-center justify-center shrink-0">
                <Building2 class="h-5 w-5 text-violet-600" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Top Products + Top Suppliers -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-2">
        <!-- Top productos comprados -->
        <Card>
          <CardHeader class="pb-2">
            <CardTitle class="text-sm font-semibold flex items-center gap-2">
              <Package class="h-4 w-4 text-orange-500" />
              Top productos comprados
            </CardTitle>
            <p class="text-xs text-muted-foreground">Por monto total facturado</p>
          </CardHeader>
          <CardContent class="pt-0">
            <div v-if="resumenCompras.TopProductos?.length" class="space-y-0 divide-y">
              <div v-for="(p, i) in resumenCompras.TopProductos.slice(0, 8)" :key="i"
                class="flex items-center gap-3 py-2">
                <span class="text-[10px] font-mono text-muted-foreground/50 w-4 shrink-0">
                  {{ i + 1 }}
                </span>
                <div class="flex-1 min-w-0">
                  <p class="text-xs font-medium truncate uppercase">{{ p.Descripcion }}</p>
                  <p class="text-[10px] text-muted-foreground">
                    {{ p.TotalCantidad }} unid · {{ p.NumFacturas }} factura(s)
                  </p>
                </div>
                <span class="text-xs font-semibold tabular-nums shrink-0">
                  {{ formatCurrency(p.TotalComprado) }}
                </span>
              </div>
            </div>
            <p v-else class="text-xs text-muted-foreground py-4 text-center">Sin datos</p>
          </CardContent>
        </Card>

        <!-- Top proveedores -->
        <Card>
          <CardHeader class="pb-2">
            <CardTitle class="text-sm font-semibold flex items-center gap-2">
              <Building2 class="h-4 w-4 text-violet-500" />
              Top proveedores
            </CardTitle>
            <p class="text-xs text-muted-foreground">Por monto total comprado</p>
          </CardHeader>
          <CardContent class="pt-0">
            <div v-if="resumenCompras.TopProveedores?.length" class="space-y-0 divide-y">
              <div v-for="(prov, i) in resumenCompras.TopProveedores" :key="i"
                class="flex items-center gap-3 py-2">
                <span class="text-[10px] font-mono text-muted-foreground/50 w-4 shrink-0">
                  {{ i + 1 }}
                </span>
                <div class="flex-1 min-w-0">
                  <p class="text-xs font-medium truncate uppercase">{{ prov.Nombre }}</p>
                  <p class="text-[10px] text-muted-foreground font-mono">
                    NIT {{ prov.NIT || "—" }} · {{ prov.TotalFacturas }} factura(s)
                  </p>
                </div>
                <span class="text-xs font-semibold tabular-nums shrink-0">
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
