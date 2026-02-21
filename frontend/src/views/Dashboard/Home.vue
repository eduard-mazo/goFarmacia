<script setup lang="ts">
import { shallowRef, onMounted, watch, computed, ref } from "vue";
import {
  ObtenerDatosDashboard,
  ObtenerFechasConVentas,
} from "@/../wailsjs/go/backend/Db";
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
import {
  DollarSign,
  ShoppingBag,
  BarChart2,
  UserCheck,
  PackageX,
  TrendingUp,
  AlertCircle,
  Wallet,
  Calendar as CalendarIcon,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type DashboardData = backend.DashboardData;

const dashboardData = ref<DashboardData | null>(null);
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

onMounted(() => {
  loadFechasConVentas();
  loadDashboardData(date.value);
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
        <Card class="shadow-sm">
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
        <Card class="shadow-sm">
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
        <Card class="shadow-sm">
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
        <Card class="shadow-sm bg-primary text-primary-foreground">
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

      <!-- Charts row -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-2">
        <Card class="shadow-sm">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold">Flujo de Ventas por Hora</CardTitle>
          </CardHeader>
          <CardContent class="h-[280px]">
            <SalesTrendChart :chart-data="dashboardData.ventasIndividuales" />
          </CardContent>
        </Card>

        <Card class="shadow-sm">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <Wallet class="h-4 w-4 text-muted-foreground" />
              Métodos de Pago
            </CardTitle>
          </CardHeader>
          <CardContent class="h-[280px] flex items-center justify-center">
            <PaymentMethodsChart
              v-if="dashboardData.metodosPago?.length"
              :chart-data="dashboardData.metodosPago"
            />
            <p v-else class="text-sm text-muted-foreground">Sin datos de pago para este día.</p>
          </CardContent>
        </Card>
      </div>

      <!-- Bottom row -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-2">
        <!-- Top Productos -->
        <Card class="shadow-sm">
          <CardHeader class="pb-3">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <TrendingUp class="h-4 w-4 text-emerald-600" />
              Top Productos Vendidos
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ul v-if="dashboardData.topProductos?.length" class="space-y-2">
              <li
                v-for="(p, i) in dashboardData.topProductos"
                :key="p.nombre"
                class="flex justify-between items-center py-2 border-b last:border-0"
              >
                <div class="flex items-center gap-3 min-w-0">
                  <span class="text-xs font-bold text-muted-foreground w-5 shrink-0">#{{ i + 1 }}</span>
                  <span class="text-sm font-medium truncate">{{ p.nombre }}</span>
                </div>
                <span class="shrink-0 text-xs font-semibold bg-emerald-100 text-emerald-700 px-2 py-0.5 rounded-full ml-2">
                  {{ p.cantidad }} uds
                </span>
              </li>
            </ul>
            <p v-else class="text-sm text-muted-foreground py-4 text-center">
              No se vendieron productos este día.
            </p>
          </CardContent>
        </Card>

        <!-- Productos Sin Stock -->
        <Card class="shadow-sm">
          <CardHeader class="pb-3">
            <CardTitle class="text-base font-semibold flex items-center gap-2 text-destructive">
              <PackageX class="h-4 w-4" />
              Productos Sin Stock
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ul v-if="dashboardData.productosSinStock?.length" class="space-y-2">
              <li
                v-for="p in dashboardData.productosSinStock"
                :key="p.UUID"
                class="flex items-center justify-between py-2 border-b last:border-0"
              >
                <div class="min-w-0">
                  <p class="text-sm font-medium truncate">{{ p.Nombre }}</p>
                  <p class="font-mono text-xs text-muted-foreground">{{ p.Codigo }}</p>
                </div>
                <span class="shrink-0 text-xs font-semibold bg-red-100 text-red-700 px-2 py-0.5 rounded-full ml-2">
                  Agotado
                </span>
              </li>
            </ul>
            <p v-else class="text-sm text-muted-foreground py-4 text-center">
              ¡Todo en orden! Sin faltantes.
            </p>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
