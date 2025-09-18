<script setup lang="ts">
import { toast } from "vue-sonner";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

import { ResetearTodaLaData } from "@/../wailsjs/go/backend/Db";

async function resetDB() {
  toast.warning("BASE DE DATOS", {
    description: "Iniciando limpieza",
  });
  var res = await ResetearTodaLaData();
  toast.success("BASE DE DATOS", {
    description: res,
  });
}
</script>

<template>
  <Card>
    <CardHeader>
      <h3 class="text-lg font-medium">Bienvenido al Panel de Administración</h3>
    </CardHeader>
    <CardContent>
      <p class="mb-4">
        Desde aquí puedes gestionar productos, clientes, proveedores y
        facturación.
      </p>
    </CardContent>
  </Card>
</template>
