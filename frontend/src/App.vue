<script setup lang="ts">
import { onMounted, onUnmounted } from "vue";
import "vue-sonner/style.css";
import { toast } from "vue-sonner";
import {
  EventsOn,
  WindowFullscreen,
  WindowUnfullscreen,
  WindowMaximise,
  WindowIsFullscreen,
} from "@/../bridge/runtime/runtime";

EventsOn("sync:start", (mensaje: string) => {
  console.log(`Sincronizando Factura: ${mensaje}`);
  toast.loading(`Sincronizando Factura: ${mensaje}`, { id: "sync-toast" });
});

EventsOn("sync:finish", (mensaje: string) => {
  console.log(`FACTURA: ${mensaje} Sincronización completada`);
  toast.success("FACTURA:", {
    description: `${mensaje} Sincronización completada`,
    id: "sync-toast",
  });
});

// Fullscreen toggle — always query actual window state to avoid ref desync.
// A 200ms gap between unfullscreen and re-maximize lets GTK process the transition.
let fsLocked = false;

async function onKeydown(e: KeyboardEvent) {
  if (e.key !== "F9" || fsLocked) return;
  e.preventDefault();
  fsLocked = true;
  try {
    const isFs = await WindowIsFullscreen();
    if (isFs) {
      WindowUnfullscreen();
      await new Promise<void>((r) => setTimeout(r, 200));
      WindowMaximise();
    } else {
      WindowFullscreen();
    }
  } finally {
    setTimeout(() => { fsLocked = false; }, 600);
  }
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <router-view />
</template>
