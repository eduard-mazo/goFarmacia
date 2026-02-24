<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import "vue-sonner/style.css";
import { toast } from "vue-sonner";
import {
  EventsOn,
  WindowFullscreen,
  WindowUnfullscreen,
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

const isFullscreen = ref(false);

function onKeydown(e: KeyboardEvent) {
  if (e.key === "F9") {
    e.preventDefault();
    if (isFullscreen.value) {
      WindowUnfullscreen();
    } else {
      WindowFullscreen();
    }
    isFullscreen.value = !isFullscreen.value;
  }
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <router-view />
</template>
