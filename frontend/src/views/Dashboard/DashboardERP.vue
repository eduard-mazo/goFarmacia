<script setup lang="ts">
import { ref, onMounted } from "vue";
import {
  ObtenerResumenInventario,
  ObtenerDatosDashboard,
} from "@/../wailsjs/go/backend/Db";
import {
  Package,
  AlertTriangle,
  PackageX,
  Users,
  DollarSign,
  TrendingUp,
  AlertCircle,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface ResumenInventario {
  totalProductos: number;
  productosStockBajo: number;
  productosSinStock: number;
  valorInventario: number;
}

const resumen = ref<ResumenInventario | null>(null);
const ventasMes = ref<number>(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

const formatCurrency = (value: number) =>
  new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(value);

async function loadData() {
  isLoading.value = true;
  error.value = null;
  try {
    const [inv, dashboard] = await Promise.all([
      ObtenerResumenInventario(),
      ObtenerDatosDashboard(""),
    ]);
    resumen.value = inv;
    ventasMes.value = dashboard.totalVentasDia;
  } catch (err: any) {
    error.value = "No se pudieron cargar los datos. " + err;
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadData);
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- Page header -->
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Dashboard ERP</h1>
      <p class="text-sm text-muted-foreground mt-0.5">
        Resumen operativo de inventario y ventas
      </p>
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

    <!-- Error -->
    <Alert v-if="error" variant="destructive">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error al cargar datos</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Content -->
    <template v-if="!isLoading && resumen">
      <!-- Top stat cards -->
      <div class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
        <!-- Total Productos -->
        <Card class="">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Total Productos</p>
                <div class="text-2xl font-bold tracking-tight mt-1">{{ resumen.totalProductos }}</div>
                <p class="text-xs text-muted-foreground mt-1">Productos activos en catálogo</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-blue-500/10 flex items-center justify-center shrink-0">
                <Package class="h-5 w-5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Stock Bajo -->
        <Card class="">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Stock Bajo</p>
                <div class="text-2xl font-bold tracking-tight mt-1 text-yellow-600">
                  {{ resumen.productosStockBajo }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Productos con stock ≤ 10</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-yellow-500/10 flex items-center justify-center shrink-0">
                <AlertTriangle class="h-5 w-5 text-yellow-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Sin Stock -->
        <Card class="">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Sin Stock</p>
                <div class="text-2xl font-bold tracking-tight mt-1 text-red-600">
                  {{ resumen.productosSinStock }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Productos agotados</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-red-500/10 flex items-center justify-center shrink-0">
                <PackageX class="h-5 w-5 text-red-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Valor Inventario (highlighted) -->
        <Card class="bg-primary text-primary-foreground">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-primary-foreground/70">Valor Inventario</p>
                <div class="text-xl font-bold tracking-tight mt-1 truncate">
                  {{ formatCurrency(resumen.valorInventario) }}
                </div>
                <p class="text-xs text-primary-foreground/70 mt-1">A precio de venta</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-white/15 flex items-center justify-center shrink-0">
                <DollarSign class="h-5 w-5 text-primary-foreground" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Bottom cards -->
      <div class="grid gap-4 grid-cols-1 lg:grid-cols-2">
        <!-- Ventas del Día -->
        <Card class="">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <TrendingUp class="h-4 w-4 text-emerald-600" />
              Ventas del Día
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div class="text-3xl font-bold tracking-tight">
              {{ formatCurrency(ventasMes) }}
            </div>
            <p class="text-sm text-muted-foreground mt-1">Total de ventas registradas hoy</p>
          </CardContent>
        </Card>

        <!-- Resumen Rápido -->
        <Card class="">
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-semibold flex items-center gap-2">
              <Users class="h-4 w-4 text-muted-foreground" />
              Resumen de Inventario
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div class="flex justify-between items-center text-sm py-2 border-b">
              <span class="text-muted-foreground">Productos activos</span>
              <span class="font-semibold">{{ resumen.totalProductos }}</span>
            </div>
            <div class="flex justify-between items-center text-sm py-2 border-b">
              <span class="text-muted-foreground">Con stock bajo</span>
              <span class="font-semibold text-yellow-600">{{ resumen.productosStockBajo }}</span>
            </div>
            <div class="flex justify-between items-center text-sm py-2">
              <span class="text-muted-foreground">Requieren atención</span>
              <span class="font-semibold text-red-600">
                {{ resumen.productosStockBajo + resumen.productosSinStock }}
              </span>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
