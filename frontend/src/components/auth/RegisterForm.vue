<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle, Loader2, Eye, EyeOff, Building2, CheckCircle2 } from "lucide-vue-next";

import { useAuthStore } from "@/stores/auth";
import { backend } from "@/../wailsjs/go/models";

const registerPayload = ref<backend.Vendedor>(new backend.Vendedor());

const error = ref<string | null>(null);
const isLoading = ref(false);
const showPassword = ref<boolean>(false);
const router = useRouter();
const authStore = useAuthStore();

const handleRegister = async () => {
  isLoading.value = true;
  error.value = "";
  try {
    await authStore.register(registerPayload.value);
    router.push({ path: "/login", query: { registered: "true" } });
  } catch (err: any) {
    error.value = err.message || "Ocurrió un error durante el registro.";
  } finally {
    isLoading.value = false;
  }
};
</script>

<template>
  <div class="flex flex-col lg:flex-row h-screen">

    <!-- Left brand panel -->
    <div
      class="hidden lg:flex lg:w-2/5 bg-gradient-to-br from-slate-900 via-blue-950 to-indigo-900
             flex-col justify-center px-12 py-12 select-none overflow-hidden gap-12"
    >
      <!-- Icon + brand -->
      <div class="flex flex-col gap-5">
        <div class="w-14 h-14 rounded-2xl bg-white/10 flex items-center justify-center">
          <Building2 class="w-8 h-8 text-white" />
        </div>
        <div>
          <h1 class="text-2xl font-bold text-white tracking-tight">Droguería Luna</h1>
          <p class="text-blue-300 text-sm mt-1">Sistema integral de gestión farmacéutica</p>
        </div>
      </div>

      <!-- Feature list -->
      <ul class="flex flex-col gap-4">
        <li v-for="f in ['Control de inventario', 'Punto de venta integrado', 'Reportes en tiempo real']"
          :key="f" class="flex items-center gap-3 text-white/80 text-sm">
          <CheckCircle2 class="w-4 h-4 text-blue-300 shrink-0" />
          {{ f }}
        </li>
      </ul>

      <!-- Bottom tagline -->
      <p class="text-blue-400/60 text-xs">
        © {{ new Date().getFullYear() }} Droguería Luna · Todos los derechos reservados
      </p>
    </div>

    <!-- Right form panel -->
    <div class="w-full lg:w-3/5 flex flex-col justify-center items-center
                px-6 sm:px-12 lg:px-16 py-10 bg-background overflow-y-auto">

      <form @submit.prevent="handleRegister" class="w-full max-w-sm">

        <!-- Mobile brand header -->
        <div class="lg:hidden text-center mb-8">
          <h1 class="text-xl font-bold text-foreground">Droguería Luna</h1>
          <p class="text-muted-foreground text-sm">Sistema de gestión farmacéutica</p>
        </div>

        <div class="mb-8">
          <h2 class="text-2xl font-bold text-foreground mb-1">Crea tu cuenta</h2>
          <p class="text-muted-foreground text-sm">
            Completa el formulario para unirte al sistema
          </p>
        </div>

        <div class="flex flex-col gap-5">
          <!-- Nombre + Apellido -->
          <div class="grid grid-cols-2 gap-4">
            <div class="grid gap-2">
              <Label for="nombre">Nombres</Label>
              <Input
                id="nombre"
                v-model="registerPayload.Nombre"
                type="text"
                placeholder="Juan"
                required
                class="h-10"
              />
            </div>
            <div class="grid gap-2">
              <Label for="apellido">Apellidos</Label>
              <Input
                id="apellido"
                v-model="registerPayload.Apellido"
                type="text"
                placeholder="Rodríguez"
                required
                class="h-10"
              />
            </div>
          </div>

          <!-- Email + Cédula -->
          <div class="grid grid-cols-2 gap-4">
            <div class="grid gap-2">
              <Label for="email">Correo electrónico</Label>
              <Input
                id="email"
                v-model="registerPayload.Email"
                type="email"
                placeholder="usuario@ejemplo.com"
                required
                class="h-10"
              />
            </div>
            <div class="grid gap-2">
              <Label for="cedula">Cédula</Label>
              <Input
                id="cedula"
                v-model="registerPayload.Cedula"
                type="text"
                placeholder="1020430991"
                required
                class="h-10"
              />
            </div>
          </div>

          <!-- Contraseña -->
          <div class="grid gap-2">
            <Label for="password">Contraseña</Label>
            <div class="relative">
              <Input
                id="password"
                v-model="registerPayload.Contrasena"
                :type="showPassword ? 'text' : 'password'"
                placeholder="••••••••"
                required
                class="h-10 pr-10"
              />
              <button
                type="button"
                @click="showPassword = !showPassword"
                class="absolute inset-y-0 right-0 flex items-center pr-3
                       text-muted-foreground hover:text-foreground transition-colors"
                tabindex="-1"
              >
                <EyeOff v-if="showPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <Alert v-if="error" variant="destructive">
            <AlertCircle class="w-4 h-4" />
            <AlertTitle>Error de registro</AlertTitle>
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>

          <Button type="submit" class="w-full h-10" :disabled="isLoading">
            <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
            {{ isLoading ? "Creando cuenta..." : "Crear Cuenta" }}
          </Button>

          <p class="text-center text-sm text-muted-foreground">
            ¿Ya tienes una cuenta?
            <router-link
              to="/login"
              class="font-medium text-foreground underline underline-offset-4
                     hover:text-primary transition-colors"
            >
              Inicia sesión
            </router-link>
          </p>
        </div>
      </form>
    </div>

  </div>
</template>
