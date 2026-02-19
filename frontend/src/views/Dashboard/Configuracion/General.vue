<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "vue-sonner";
import { Settings, Save } from "lucide-vue-next";

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

onMounted(loadSettings);
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
        Ajusta los datos de la tienda y parámetros del sistema
      </p>
    </div>

    <!-- Datos de la tienda -->
    <Card class="shadow-sm">
      <CardHeader class="pb-4">
        <CardTitle class="text-base font-semibold">Datos de la Tienda</CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="grid gap-2">
          <Label for="nombreTienda">Nombre de la farmacia</Label>
          <Input id="nombreTienda" v-model="settings.nombreTienda" class="h-9"
            placeholder="Ej: Droguería Luna" />
        </div>
        <div class="grid gap-2">
          <Label for="direccion">Dirección</Label>
          <Input id="direccion" v-model="settings.direccion" class="h-9"
            placeholder="Dirección del establecimiento" />
        </div>
        <div class="grid gap-2">
          <Label for="telefono">Teléfono de contacto</Label>
          <Input id="telefono" v-model="settings.telefono" class="h-9"
            placeholder="Ej: 310 000 0000" />
        </div>
      </CardContent>
    </Card>

    <!-- Parámetros del sistema -->
    <Card class="shadow-sm">
      <CardHeader class="pb-4">
        <CardTitle class="text-base font-semibold">Parámetros del Sistema</CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="grid gap-2">
          <Label for="tasaIVA">Tasa de IVA (%)</Label>
          <Input id="tasaIVA" v-model.number="settings.tasaIVA" type="number"
            min="0" max="100" class="h-9 max-w-[160px]" />
          <p class="text-xs text-muted-foreground">Porcentaje aplicado a los productos gravados</p>
        </div>
        <div class="grid gap-2">
          <Label for="umbralStock">Umbral de Stock Bajo</Label>
          <Input id="umbralStock" v-model.number="settings.umbralStockBajo" type="number"
            min="0" class="h-9 max-w-[160px]" />
          <p class="text-xs text-muted-foreground">Productos con stock igual o menor se marcarán en amarillo</p>
        </div>
      </CardContent>
    </Card>

    <div class="flex justify-end pt-2">
      <Button @click="saveSettings" class="h-9 gap-2">
        <Save class="w-4 h-4" />Guardar Configuración
      </Button>
    </div>
  </div>
</template>
