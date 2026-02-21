<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { ObtenerReporteVentasRango } from "@/../wailsjs/go/backend/Db";
import { format, subDays } from "date-fns";
import { es } from "date-fns/locale";
import {
  DollarSign,
  ShoppingBag,
  BarChart2,
  AlertCircle,
  Search,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import SalesTrendChart from "@/components/dashboard/SalesTrendChart.vue";
import PaymentMethodsChart from "@/components/dashboard/PaymentMethodsChart.vue";

interface ReporteVentas {
  totalVentas: number;
  numeroVentas: number;
  ticketPromedio: number;
  ventasIndividuales: { timestamp: string; total: number }[];
  topProductos: { nombre: string; cantidad: number }[];
  topVendedores: { nombreCompleto: string; totalVendido: number }[];
  metodosPago: Record<string, any>[];
}

const fechaInicio = ref(format(subDays(new Date(), 30), "yyyy-MM-dd"));
const fechaFin = ref(format(new Date(), "yyyy-MM-dd"));
const reporte = ref<ReporteVentas | null>(null);
const isLoading = ref(false);
const error = ref<string | null>(null);

const formatCurrency = (value: number) => {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(value);
};

const rangoLabel = computed(() => {
  const inicio = new Date(fechaInicio.value + "T00:00:00");
  const fin = new Date(fechaFin.value + "T00:00:00");
  return `${format(inicio, "d MMM yyyy", { locale: es })} - ${format(fin, "d MMM yyyy", { locale: es })}`;
});

async function cargarReporte() {
  isLoading.value = true;
  error.value = null;
  try {
    reporte.value = await ObtenerReporteVentasRango(
      fechaInicio.value,
      fechaFin.value
    );
  } catch (err: any) {
    error.value = "Error al cargar el reporte: " + err;
  } finally {
    isLoading.value = false;
  }
}

onMounted(cargarReporte);
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- Page header + filter bar -->
    <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Reporte de Ventas</h1>
        <p class="text-sm text-muted-foreground mt-0.5">{{ rangoLabel }}</p>
      </div>
      <div class="flex items-end gap-3 flex-wrap">
        <div class="grid gap-1">
          <Label class="text-xs text-muted-foreground">Desde</Label>
          <Input type="date" v-model="fechaInicio" class="h-9 w-[150px]" />
        </div>
        <div class="grid gap-1">
          <Label class="text-xs text-muted-foreground">Hasta</Label>
          <Input type="date" v-model="fechaFin" class="h-9 w-[150px]" />
        </div>
        <Button @click="cargarReporte" :disabled="isLoading" class="h-9 gap-2">
          <Search class="w-4 h-4" />Consultar
        </Button>
      </div>
    </div>

    <!-- Loading skeletons -->
    <div v-if="isLoading" class="grid gap-4 grid-cols-1 sm:grid-cols-3">
      <Card v-for="i in 3" :key="i" class="animate-pulse">
        <CardContent class="p-6">
          <div class="space-y-2">
            <div class="h-3.5 bg-muted rounded w-3/5"></div>
            <div class="h-7 bg-muted rounded w-4/5"></div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Error -->
    <Alert v-if="error" variant="destructive">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error al cargar reporte</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <template v-if="!isLoading && reporte">
      <!-- Stat cards -->
      <div class="grid gap-4 grid-cols-1 sm:grid-cols-3">
        <Card class="shadow-sm">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-muted-foreground">Total Ventas</p>
                <div class="text-2xl font-bold tracking-tight mt-1 truncate">
                  {{ formatCurrency(reporte.totalVentas) }}
                </div>
              </div>
              <div class="h-10 w-10 rounded-xl bg-emerald-500/10 flex items-center justify-center shrink-0">
                <DollarSign class="h-5 w-5 text-emerald-600" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card class="shadow-sm">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Nº de Ventas</p>
                <div class="text-2xl font-bold tracking-tight mt-1">{{ reporte.numeroVentas }}</div>
              </div>
              <div class="h-10 w-10 rounded-xl bg-blue-500/10 flex items-center justify-center shrink-0">
                <ShoppingBag class="h-5 w-5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card class="shadow-sm">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-muted-foreground">Ticket Promedio</p>
                <div class="text-2xl font-bold tracking-tight mt-1 truncate">
                  {{ formatCurrency(reporte.ticketPromedio) }}
                </div>
              </div>
              <div class="h-10 w-10 rounded-xl bg-violet-500/10 flex items-center justify-center shrink-0">
                <BarChart2 class="h-5 w-5 text-violet-600" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Charts -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-2">
        <Card class="shadow-sm">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold">Tendencia de Ventas</CardTitle>
          </CardHeader>
          <CardContent class="h-[280px]">
            <SalesTrendChart :chart-data="reporte.ventasIndividuales" />
          </CardContent>
        </Card>
        <Card class="shadow-sm">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold">Métodos de Pago</CardTitle>
          </CardHeader>
          <CardContent class="h-[280px] flex items-center justify-center">
            <PaymentMethodsChart v-if="reporte.metodosPago?.length" :chart-data="reporte.metodosPago" />
            <p v-else class="text-sm text-muted-foreground">Sin datos de pago para este período.</p>
          </CardContent>
        </Card>
      </div>

      <!-- Lists -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-2">
        <Card class="shadow-sm">
          <CardHeader class="pb-3">
            <CardTitle class="text-base font-semibold">Top Productos Vendidos</CardTitle>
          </CardHeader>
          <CardContent>
            <ul v-if="reporte.topProductos?.length" class="space-y-2">
              <li v-for="(p, i) in reporte.topProductos" :key="p.nombre"
                class="flex justify-between items-center py-2 border-b last:border-0">
                <div class="flex items-center gap-3 min-w-0">
                  <span class="text-xs font-bold text-muted-foreground w-5 shrink-0">#{{ i + 1 }}</span>
                  <span class="text-sm font-medium truncate">{{ p.nombre }}</span>
                </div>
                <span class="shrink-0 text-xs font-semibold bg-emerald-100 text-emerald-700 px-2 py-0.5 rounded-full ml-2">
                  {{ p.cantidad }} uds
                </span>
              </li>
            </ul>
            <p v-else class="text-sm text-muted-foreground py-4 text-center">Sin productos en este período.</p>
          </CardContent>
        </Card>
        <Card class="shadow-sm">
          <CardHeader class="pb-3">
            <CardTitle class="text-base font-semibold">Top Vendedores</CardTitle>
          </CardHeader>
          <CardContent>
            <ul v-if="reporte.topVendedores?.length" class="space-y-2">
              <li v-for="(v, i) in reporte.topVendedores" :key="v.nombreCompleto"
                class="flex justify-between items-center py-2 border-b last:border-0">
                <div class="flex items-center gap-3 min-w-0">
                  <span class="text-xs font-bold text-muted-foreground w-5 shrink-0">#{{ i + 1 }}</span>
                  <span class="text-sm font-medium truncate">{{ v.nombreCompleto }}</span>
                </div>
                <span class="shrink-0 text-xs font-semibold bg-blue-100 text-blue-700 px-2 py-0.5 rounded-full ml-2">
                  {{ formatCurrency(v.totalVendido) }}
                </span>
              </li>
            </ul>
            <p v-else class="text-sm text-muted-foreground py-4 text-center">Sin datos de vendedores.</p>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
