<script setup lang="ts">
import {
  SidebarInset,
  SidebarProvider,
} from '@/components/ui/sidebar'
import AppSidebar from "@/components/layout/AppSidebar.vue";
import { Toaster } from "vue-sonner";
</script>

<template>
  <Toaster richColors position="top-right" />
  <SidebarProvider class="h-full min-h-0">
    <AppSidebar />
    <SidebarInset class="min-h-0">
      <main class="flex flex-1 flex-col h-full overflow-auto bg-muted/20">
        <router-view />
      </main>
    </SidebarInset>
  </SidebarProvider>
</template>
