<script setup lang="ts">
import { ref, onMounted } from "vue";
import { ObtenerDatosDashboard } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";
import SalesTrendChart from "@/components/dashboard/SalesTrendChart.vue";
import {
  DollarSign,
  ShoppingBag,
  BarChart2,
  UserCheck,
  PackageX,
  TrendingUp,
  AlertCircle,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type DashboardData = {
  totalVentasHoy: number;
  numeroVentasHoy: number;
  ticketPromedioHoy: number;
  ventasIndividuales: { timestamp: string; total: number }[];
  topProductos: { nombre: string; cantidad: number }[];
  productosSinStock: backend.Producto[];
  topVendedor: { nombreCompleto: string; totalVendido: number };
};

const dashboardData = ref<DashboardData | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

const formatCurrency = (value: number) => {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(value);
};

// Se mantiene la lógica de carga robusta
async function loadDashboardData() {
  isLoading.value = true;
  error.value = null;

  try {
    const response = await ObtenerDatosDashboard();
    dashboardData.value = response;
  } catch (err) {
    console.error("Error al cargar datos del dashboard:", err);
    error.value =
      "No se pudieron cargar los datos. Revisa la conexión o intenta más tarde.";
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadDashboardData);
</script>

<template>
  <div class="p-4 md:p-6 lg:p-8">
    <h1 class="text-3xl font-bold mb-6">Dashboard de Hoy</h1>

    <div v-if="isLoading" class="text-center py-10">Cargando métricas...</div>

    <div
      v-if="error"
      class="flex items-center gap-3 text-destructive-foreground bg-destructive p-4 rounded-lg"
    >
      <AlertCircle class="h-6 w-6 flex-shrink-0" />
      <span class="font-medium">{{ error }}</span>
    </div>

    <div
      v-if="dashboardData"
      class="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-4"
    >
      <Card>
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium"
            >Total Ventas del Día</CardTitle
          >
          <DollarSign class="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">
            {{ formatCurrency(dashboardData.totalVentasHoy) }}
          </div>
          <p class="text-xs text-muted-foreground">
            Ventas totales registradas hoy
          </p>
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
            {{ dashboardData.numeroVentasHoy }}
          </div>
          <p class="text-xs text-muted-foreground">
            Transacciones completadas hoy
          </p>
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
            {{ formatCurrency(dashboardData.ticketPromedioHoy) }}
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
          <CardTitle>Flujo de Ventas del Día</CardTitle>
        </CardHeader>
        <CardContent class="h-[300px]">
          <SalesTrendChart :chart-data="dashboardData.ventasIndividuales" />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2"
            ><TrendingUp class="h-5 w-5" /> Top Productos Vendidos</CardTitle
          >
        </CardHeader>
        <CardContent>
          <ul class="space-y-3 text-sm">
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
            <li
              v-if="!dashboardData.topProductos.length"
              class="text-muted-foreground text-xs"
            >
              Aún no hay ventas registradas hoy.
            </li>
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-destructive"
            ><PackageX class="h-5 w-5" /> Productos Sin Stock</CardTitle
          >
        </CardHeader>
        <CardContent>
          <ul class="space-y-3 text-sm">
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
            <li
              v-if="!dashboardData.productosSinStock.length"
              class="text-muted-foreground text-xs"
            >
              ¡Todo en orden! No hay faltantes.
            </li>
          </ul>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
