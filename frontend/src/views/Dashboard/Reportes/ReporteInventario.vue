<script setup lang="ts">
import { ref, onMounted } from "vue";
import { ObtenerResumenInventario } from "@/../wailsjs/go/backend/Db";
import {
  Package,
  AlertTriangle,
  PackageX,
  DollarSign,
  AlertCircle,
} from "lucide-vue-next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";

interface ResumenInventario {
  totalProductos: number;
  productosStockBajo: number;
  productosSinStock: number;
  valorInventario: number;
  productosAlerta: {
    uuid: string;
    nombre: string;
    codigo: string;
    stock: number;
    precioVenta: number;
  }[];
}

const resumen = ref<ResumenInventario | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

const formatCurrency = (value: number) => {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(value);
};

async function loadData() {
  isLoading.value = true;
  error.value = null;
  try {
    resumen.value = await ObtenerResumenInventario();
  } catch (err: any) {
    error.value = "Error al cargar resumen de inventario: " + err;
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
      <h1 class="text-2xl font-semibold tracking-tight">Reporte de Inventario</h1>
      <p class="text-sm text-muted-foreground mt-0.5">Estado actual del inventario y productos en alerta</p>
    </div>

    <!-- Loading skeletons -->
    <div v-if="isLoading" class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
      <Card v-for="i in 4" :key="i" class="animate-pulse shadow-sm">
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
      <AlertTitle>Error al cargar reporte</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <template v-if="!isLoading && resumen">
      <!-- Stat cards -->
      <div class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
        <Card class="shadow-sm">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Total Productos</p>
                <div class="text-2xl font-bold tracking-tight mt-1">{{ resumen.totalProductos }}</div>
                <p class="text-xs text-muted-foreground mt-1">Activos en catálogo</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-blue-500/10 flex items-center justify-center shrink-0">
                <Package class="h-5 w-5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card class="shadow-sm">
          <CardContent class="p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-muted-foreground">Stock Bajo</p>
                <div class="text-2xl font-bold tracking-tight mt-1 text-yellow-600">
                  {{ resumen.productosStockBajo }}
                </div>
                <p class="text-xs text-muted-foreground mt-1">Stock ≤ 10 unidades</p>
              </div>
              <div class="h-10 w-10 rounded-xl bg-yellow-500/10 flex items-center justify-center shrink-0">
                <AlertTriangle class="h-5 w-5 text-yellow-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card class="shadow-sm">
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

        <Card class="shadow-sm bg-primary text-primary-foreground">
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

      <!-- Alert products -->
      <Card class="shadow-sm">
        <CardHeader class="pb-3">
          <CardTitle class="text-base font-semibold flex items-center gap-2">
            <AlertTriangle class="h-4 w-4 text-yellow-500" />
            Productos que Requieren Atención
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div v-if="resumen.productosAlerta?.length" class="space-y-2">
            <div v-for="p in resumen.productosAlerta" :key="p.uuid"
              class="flex items-center justify-between p-3 rounded-lg border bg-muted/30 hover:bg-muted/50 transition-colors">
              <div class="min-w-0">
                <p class="text-sm font-medium truncate">{{ p.nombre }}</p>
                <p class="text-xs text-muted-foreground font-mono">{{ p.codigo }}</p>
              </div>
              <div class="flex items-center gap-3 shrink-0 ml-3">
                <span class="text-sm text-muted-foreground hidden sm:block">
                  {{ formatCurrency(p.precioVenta) }}
                </span>
                <Badge :variant="p.stock === 0 ? 'destructive' : 'secondary'" class="min-w-[56px] justify-center">
                  {{ p.stock }} uds
                </Badge>
              </div>
            </div>
          </div>
          <p v-else class="text-sm text-muted-foreground py-4 text-center">
            Todos los productos tienen stock adecuado.
          </p>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
