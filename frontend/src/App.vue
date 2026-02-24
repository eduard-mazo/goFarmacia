<script setup lang="ts">
import { onMounted, onUnmounted } from "vue";
import "vue-sonner/style.css";
import { toast } from "vue-sonner";
import {
  EventsOn,
  WindowFullscreen,
  WindowUnfullscreen,
  WindowIsFullscreen,
} from "@/../wailsjs/runtime";

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

async function onKeydown(e: KeyboardEvent) {
  if (e.key === "F11") {
    e.preventDefault();
    const isFs = await WindowIsFullscreen();
    if (isFs) {
      WindowUnfullscreen();
    } else {
      WindowFullscreen();
    }
  }
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <router-view />
</template>
