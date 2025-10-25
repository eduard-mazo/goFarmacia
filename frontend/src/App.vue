<script setup lang="ts">
import "vue-sonner/style.css";
import { toast } from "vue-sonner";
import { EventsOn } from "@/../wailsjs/runtime";

EventsOn("sync:start", (mensaje: string) => {
  console.log(`Sincronizando Factura: ${mensaje}`);
  toast.loading(`Sincronizando Factura: ${mensaje}`, {
    id: "sync-toast",
  });
});

EventsOn("sync:finish", (mensaje: string) => {
  console.log(`FACTURA: ${mensaje} Sincronización completada`);
  toast.success("FACTURA:", {
    description: `${mensaje} Sincronización completada`,
    id: "sync-toast",
  });
});
</script>

<template>
  <router-view />
</template>
