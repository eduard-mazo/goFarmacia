<script lang="ts">
export const iframeHeight = "800px"
export const description
  = "A simple sidebar with navigation grouped by section."
</script>

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
  <SidebarProvider>
    <AppSidebar />
    <SidebarInset>
      <main class="flex flex-1 flex-col overflow-auto bg-muted/20">
        <router-view />
      </main>
    </SidebarInset>
  </SidebarProvider>
</template>
