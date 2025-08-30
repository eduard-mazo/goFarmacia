// src/layouts/DashboardLayout.vue

<script setup lang="ts">
import {
  CircleUser,
  Home,
  Menu,
  Package,
  ShoppingCart,
} from "lucide-vue-next";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { useAuth } from "@/services/auth";
import { Toaster, toast } from "vue-sonner"; // Importamos Toaster y toast

const { user, logout } = useAuth();

// Función de ejemplo para notificaciones
function showComingSoonToast() {
  toast.info("Función no implementada", {
    description:
      "Esta característica estará disponible en futuras actualizaciones.",
  });
}
</script>

<template>
  <Toaster richColors position="top-right" />

  <div
    class="grid min-h-screen w-full md:grid-cols-[220px_1fr] lg:grid-cols-[280px_1fr]"
  >
    <div class="hidden border-r bg-muted/40 md:block">
      <div class="flex h-full max-h-screen flex-col gap-2">
        <div class="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
          <router-link
            to="/dashboard"
            class="flex items-center gap-2 font-semibold"
          >
            <Package class="h-6 w-6" />
            <span class="">Mi Negocio Inc</span>
          </router-link>
        </div>
        <div class="flex-1">
          <nav class="grid items-start px-2 text-sm font-medium lg:px-4">
            <router-link
              to="/dashboard"
              class="flex items-center gap-3 rounded-lg px-3 py-2 text-muted-foreground transition-all hover:text-primary"
              active-class="bg-muted text-primary"
            >
              <Home class="h-4 w-4" />
              Dashboard
            </router-link>
            <router-link
              to="/dashboard/pos"
              class="flex items-center gap-3 rounded-lg px-3 py-2 text-muted-foreground transition-all hover:text-primary"
              active-class="bg-muted text-primary"
            >
              <ShoppingCart class="h-4 w-4" />
              Punto de Venta
            </router-link>
          </nav>
        </div>
      </div>
    </div>

    <div class="flex flex-col">
      <header
        class="flex h-14 items-center gap-4 border-b bg-muted/40 px-4 lg:h-[60px] lg:px-6"
      >
        <Sheet>
          <SheetTrigger as-child>
            <Button variant="outline" size="icon" class="shrink-0 md:hidden">
              <Menu class="h-5 w-5" />
              <span class="sr-only">Toggle navigation menu</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="left" class="flex flex-col">
            <nav class="grid gap-2 text-lg font-medium">
              <router-link
                to="/dashboard"
                class="flex items-center gap-2 text-lg font-semibold mb-4"
              >
                <Package class="h-6 w-6" />
                <span>Mi Negocio Inc</span>
              </router-link>
              <router-link
                to="/dashboard"
                class="mx-[-0.65rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground"
                active-class="bg-muted text-foreground"
              >
                <Home class="h-5 w-5" />
                Dashboard
              </router-link>
              <router-link
                to="/dashboard/pos"
                class="mx-[-0.65rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground"
                active-class="bg-muted text-foreground"
              >
                <ShoppingCart class="h-5 w-5" />
                Punto de Venta
              </router-link>
            </nav>
          </SheetContent>
        </Sheet>

        <div class="w-full flex-1" />

        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="secondary" size="icon" class="rounded-full">
              <CircleUser class="h-5 w-5" />
              <span class="sr-only">Toggle user menu</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>{{ user?.name }}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem @click="showComingSoonToast">
              Configuración
            </DropdownMenuItem>
            <DropdownMenuItem @click="showComingSoonToast">
              Soporte
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem @click="logout"> Cerrar Sesión </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </header>

      <main class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6 bg-muted/20">
        <router-view />
      </main>
    </div>
  </div>
</template>
