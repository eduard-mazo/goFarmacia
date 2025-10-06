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

// Definición de tipos
type DashboardData = backend.DashboardData;

// --- ESTADO REACTIVO ---
// Los refs normales están bien para datos primitivos y objetos planos
const dashboardData = ref<DashboardData | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);
const fechasConVentas = ref<string[]>([]);

// Usamos shallowRef para mantener la instancia de CalendarDate intacta.
const date = shallowRef<CalendarDate>(today(getLocalTimeZone()));

// --- PROPIEDADES COMPUTADAS ---
// El resto del script no necesita cambios, ya que operan sobre la instancia correcta.
const formattedButtonDate = computed(() => {
  if (!date.value) return "Selecciona una fecha";
  return format(date.value.toDate(getLocalTimeZone()), "PPP", { locale: es });
});

const formattedTitleDate = computed(() => {
  const selectedDate = date.value.toDate(getLocalTimeZone());
  const todayDate = new Date();

  selectedDate.setHours(0, 0, 0, 0);
  todayDate.setHours(0, 0, 0, 0);

  if (selectedDate.getTime() === todayDate.getTime()) {
    return "Hoy";
  }
  return format(selectedDate, "d 'de' MMMM 'de' yyyy", { locale: es });
});

const formatCurrency = (value: number) => {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(value);
};

// --- LÓGICA DE CARGA DE DATOS ---
async function loadDashboardData(fecha: CalendarDate) {
  isLoading.value = true;
  error.value = null;
  dashboardData.value = null;

  try {
    const fechaStr = fecha.toString();
    const response = await ObtenerDatosDashboard(fechaStr);
    dashboardData.value = response;
  } catch (err: any) {
    console.error("Error al cargar datos del dashboard:", err);
    error.value = "No se pudieron cargar los datos. " + err;
  } finally {
    isLoading.value = false;
  }
}

async function loadFechasConVentas() {
  try {
    fechasConVentas.value = await ObtenerFechasConVentas();
  } catch (err) {
    console.error("Error al cargar fechas con ventas:", err);
  }
}

// --- HOOKS DEL CICLO DE VIDA ---
onMounted(() => {
  loadFechasConVentas();
  loadDashboardData(date.value);
});

watch(date, (newDate) => {
  if (newDate) {
    loadDashboardData(newDate);
  }
});
</script>

<template>
  <div class="p-4 md:p-6 lg:p-8 space-y-6">
    <div
      class="flex flex-col md:flex-row md:items-center md:justify-between gap-4"
    >
      <h1 class="text-3xl font-bold">
        Dashboard de <span class="text-primary">{{ formattedTitleDate }}</span>
      </h1>

      <Popover>
        <PopoverTrigger as-child>
          <Button
            variant="outline"
            class="w-[280px] justify-start text-left font-normal"
          >
            <CalendarIcon class="mr-2 h-4 w-4" />
            <span>{{ formattedButtonDate }}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent class="w-auto p-0">
          <Calendar v-model="date">
            <template #day-cell="{ date: day }">
              <div class="relative">
                {{ day.day }}
                <span
                  v-if="fechasConVentas.includes(day.toString())"
                  class="absolute bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full bg-emerald-500"
                ></span>
              </div>
            </template>
          </Calendar>
        </PopoverContent>
      </Popover>
    </div>

    <div
      v-if="isLoading"
      class="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-4"
    >
      <Card v-for="i in 4" :key="i" class="animate-pulse">
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <div class="h-4 bg-muted rounded w-3/5"></div>
          <div class="h-4 w-4 bg-muted rounded-full"></div>
        </CardHeader>
        <CardContent>
          <div class="h-8 bg-muted rounded w-4/5 mb-2"></div>
          <div class="h-3 bg-muted rounded w-full"></div>
        </CardContent>
      </Card>
    </div>

    <div
      v-if="error"
      class="flex items-center gap-3 text-white bg-red-400 p-4 rounded-lg"
    >
      <AlertCircle class="h-6 w-6 flex-shrink-0" />
      <span class="font-medium">{{ error }}</span>
    </div>

    <div
      v-if="!isLoading && dashboardData"
      class="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-4"
    >
      <Card>
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium">Total Ventas</CardTitle>
          <DollarSign class="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">
            {{ formatCurrency(dashboardData.totalVentasDia) }}
          </div>
          <p class="text-xs text-muted-foreground">Ventas totales del día</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium">Nº de Ventas</CardTitle>
          <ShoppingBag class="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">
            {{ dashboardData.numeroVentasDia }}
          </div>
          <p class="text-xs text-muted-foreground">Transacciones completadas</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium">Ticket Promedio</CardTitle>
          <BarChart2 class="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">
            {{ formatCurrency(dashboardData.ticketPromedioDia) }}
          </div>
          <p class="text-xs text-muted-foreground">
            Valor promedio por transacción
          </p>
        </CardContent>
      </Card>

      <Card class="bg-primary text-primary-foreground">
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium">Vendedor del Día</CardTitle>
          <UserCheck class="h-4 w-4 text-primary-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-xl font-bold truncate">
            {{ dashboardData.topVendedor?.nombreCompleto || "N/A" }}
          </div>
          <p class="text-xs text-primary-foreground/80">
            {{ formatCurrency(dashboardData.topVendedor?.totalVendido || 0) }}
            en ventas
          </p>
        </CardContent>
      </Card>

      <Card class="md:col-span-2 lg:col-span-2">
        <CardHeader>
          <CardTitle>Flujo de Ventas por Hora</CardTitle>
        </CardHeader>
        <CardContent class="h-[300px]">
          <SalesTrendChart :chart-data="dashboardData.ventasIndividuales" />
        </CardContent>
      </Card>

      <Card class="md:col-span-2 lg:col-span-2">
        <CardHeader>
          <CardTitle class="flex items-center gap-2"
            ><Wallet class="h-5 w-5" /> Métodos de Pago</CardTitle
          >
        </CardHeader>
        <CardContent class="h-[300px] flex items-center justify-center">
          <PaymentMethodsChart
            v-if="
              dashboardData.metodosPago && dashboardData.metodosPago.length > 0
            "
            :chart-data="dashboardData.metodosPago"
          />
          <p v-else class="text-sm text-muted-foreground">
            No hay datos de pago para este día.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2"
            ><TrendingUp class="h-5 w-5" /> Top Productos Vendidos</CardTitle
          >
        </CardHeader>
        <CardContent>
          <ul
            class="space-y-3 text-sm"
            v-if="
              dashboardData.topProductos && dashboardData.topProductos.length
            "
          >
            <li
              v-for="p in dashboardData.topProductos"
              :key="p.nombre"
              class="flex justify-between items-center"
            >
              <span class="font-medium truncate pr-2">{{ p.nombre }}</span>
              <span
                class="flex-shrink-0 font-semibold bg-emerald-100 text-emerald-800 px-2 py-0.5 rounded"
                >{{ p.cantidad }} uds</span
              >
            </li>
          </ul>
          <p v-else class="text-sm text-muted-foreground">
            No se vendieron productos este día.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-destructive"
            ><PackageX class="h-5 w-5" /> Productos Sin Stock</CardTitle
          >
        </CardHeader>
        <CardContent>
          <ul
            class="space-y-3 text-sm"
            v-if="
              dashboardData.productosSinStock &&
              dashboardData.productosSinStock.length
            "
          >
            <li
              v-for="p in dashboardData.productosSinStock"
              :key="p.id"
              class="flex flex-col"
            >
              <span class="font-medium truncate">{{ p.Nombre }}</span>
              <span class="font-mono text-xs text-muted-foreground">{{
                p.Codigo
              }}</span>
            </li>
          </ul>
          <p v-else class="text-sm text-muted-foreground">
            ¡Todo en orden! No hay faltantes.
          </p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
