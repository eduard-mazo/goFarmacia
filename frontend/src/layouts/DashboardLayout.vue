<script setup lang="ts">
import { onMounted, onUnmounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import AppSidebar from "@/components/layout/AppSidebar.vue";
import { Toaster } from "vue-sonner";
import { useDBStore } from "@/stores/dbStore";
import { Button } from "@/components/ui/button";
import { AlertTriangle, Database } from "lucide-vue-next";

const dbStore = useDBStore();
const router = useRouter();
const route = useRoute();

onMounted(async () => {
  await dbStore.fetchStatus();
  dbStore.startPolling(15000);

  // Redirect to DB settings if in setup mode and not already there
  if (dbStore.setupMode && route.name !== "Configuracion") {
    router.replace("/dashboard/configuracion");
  }
});

onUnmounted(() => {
  dbStore.stopPolling();
});

// React when setup mode changes (e.g. after ConfigurarDB)
watch(
  () => dbStore.setupMode,
  (isSetup) => {
    if (isSetup && route.name !== "Configuracion") {
      router.replace("/dashboard/configuracion");
    }
  }
);
</script>

<template>
  <Toaster richColors position="top-right" />
  <SidebarProvider class="h-full min-h-0">
    <AppSidebar />
    <SidebarInset class="min-h-0 flex flex-col">

      <!-- Setup mode banner -->
      <div
        v-if="dbStore.setupMode"
        class="flex items-center gap-3 bg-amber-50 border-b border-amber-200 px-4 py-2.5 text-sm shrink-0"
      >
        <AlertTriangle class="h-4 w-4 shrink-0 text-amber-500" />
        <span class="flex-1 font-medium text-amber-800">
          Sistema en modo configuración — base de datos no conectada.
        </span>
        <Button
          variant="outline"
          size="sm"
          class="h-7 border-amber-300 text-amber-700 hover:bg-amber-100 shrink-0"
          @click="router.push('/dashboard/configuracion')"
        >
          <Database class="h-3.5 w-3.5 mr-1" />
          Configurar BD
        </Button>
      </div>

      <!-- Reconnect warning (connected to server but something changed) -->
      <div
        v-else-if="!dbStore.connected && dbStore.message"
        class="flex items-center gap-3 bg-red-50 border-b border-red-200 px-4 py-2.5 text-sm shrink-0"
      >
        <AlertTriangle class="h-4 w-4 shrink-0 text-red-500" />
        <span class="flex-1 font-medium text-red-800">
          Sin conexión a la base de datos — {{ dbStore.message }}
        </span>
        <Button
          variant="outline"
          size="sm"
          class="h-7 border-red-300 text-red-700 hover:bg-red-100 shrink-0"
          @click="router.push('/dashboard/configuracion')"
        >
          Revisar
        </Button>
      </div>

      <main class="flex flex-1 flex-col overflow-auto bg-muted/20">
        <router-view />
      </main>
    </SidebarInset>
  </SidebarProvider>
</template>
