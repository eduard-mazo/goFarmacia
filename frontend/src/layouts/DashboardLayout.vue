<script setup lang="ts">
import { onMounted, onUnmounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import AppSidebar from "@/components/layout/AppSidebar.vue";
import { Toaster } from "vue-sonner";
import { useDBStore } from "@/stores/dbStore";
import { Button } from "@/components/ui/button";
import { AlertTriangle, Database } from "lucide-vue-next";
import { EventsOn, EventsOff } from "@/../bridge/runtime/runtime";

const dbStore = useDBStore();
const router = useRouter();
const route = useRoute();

onMounted(async () => {
  // Listen for the backend "db:ready" event BEFORE the first fetchStatus call.
  // This resolves the startup race: if initDB() finishes after the frontend
  // mounts (common on Linux/WebKit2GTK), the event triggers an immediate
  // re-fetch so the banner disappears instantly instead of waiting 15 seconds.
  EventsOn("db:ready", () => {
    dbStore.fetchStatus();
  });

  await dbStore.fetchStatus();
  dbStore.startPolling(15000);
  if (dbStore.setupMode && route.name !== "Configuracion") {
    router.replace("/dashboard/configuracion");
  }
});

onUnmounted(() => {
  EventsOff("db:ready");
  dbStore.stopPolling();
});

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

      <!-- Mobile top bar — only visible below md breakpoint -->
      <header class="md:hidden shrink-0 sticky top-0 z-40 flex items-center gap-3 px-3 h-12 border-b border-border bg-background/95 backdrop-blur-sm">
        <SidebarTrigger class="h-8 w-8 shrink-0" />
        <div class="flex items-center gap-2 flex-1 min-w-0">
          <div class="flex items-center justify-center w-5 h-5 rounded bg-[#1e3a5f] shrink-0">
            <svg viewBox="0 0 16 16" fill="none" class="w-3 h-3">
              <rect x="6" y="1" width="4" height="14" rx="1" fill="white"/>
              <rect x="1" y="6" width="14" height="4" rx="1" fill="white"/>
            </svg>
          </div>
          <span class="text-sm font-semibold text-foreground truncate">Droguería Luna</span>
        </div>
        <div
          class="h-2 w-2 rounded-full shrink-0"
          :class="dbStore.connected ? 'bg-emerald-500' : dbStore.setupMode ? 'bg-amber-500' : 'bg-red-500'"
          :title="dbStore.connected ? 'Conectado' : dbStore.setupMode ? 'Sin configurar' : 'Sin conexión'"
        />
      </header>

      <!-- Setup mode banner -->
      <div
        v-if="dbStore.setupMode"
        class="shrink-0 flex items-center gap-3 bg-amber-50 border-b border-amber-200 px-4 py-2 text-sm"
      >
        <AlertTriangle class="h-3.5 w-3.5 shrink-0 text-amber-500" />
        <span class="flex-1 text-amber-800 font-medium">
          Modo configuración — base de datos no conectada.
        </span>
        <Button
          variant="outline"
          size="sm"
          class="h-7 border-amber-300 text-amber-700 hover:bg-amber-100 shrink-0"
          @click="router.push('/dashboard/configuracion')"
        >
          <Database class="h-3.5 w-3.5 mr-1" />Configurar
        </Button>
      </div>

      <!-- Connection error banner -->
      <div
        v-else-if="!dbStore.connected && dbStore.message"
        class="shrink-0 flex items-center gap-3 bg-red-50 border-b border-red-200 px-4 py-2 text-sm"
      >
        <AlertTriangle class="h-3.5 w-3.5 shrink-0 text-red-500" />
        <span class="flex-1 text-red-800 font-medium">
          Sin conexión — {{ dbStore.message }}
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

      <main class="flex flex-1 flex-col overflow-auto">
        <router-view />
      </main>
    </SidebarInset>
  </SidebarProvider>
</template>
