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
import AppSidebar from "@/components/AppSidebar.vue"
import { Toaster } from "vue-sonner";
</script>

<template>
  <Toaster richColors position="top-right" />
  <SidebarProvider>
    <AppSidebar />
    <SidebarInset>
      <main class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6 bg-white">
        <router-view />
      </main>
    </SidebarInset>
  </SidebarProvider>
</template>
