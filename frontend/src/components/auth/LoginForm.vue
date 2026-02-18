<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useAuthStore } from "@/stores/auth";
import { backend } from "@/../wailsjs/go/models";
import { toast } from "vue-sonner";
import { useRoute } from "vue-router";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  AlertCircle,
  Loader2,
  ShieldCheck,
  Eye,
  EyeOff,
} from "lucide-vue-next";
import loginIllustration from "@/assets/images/Login_luna.png";

const authStore = useAuthStore();
const route = useRoute();

const credentials = ref<backend.LoginRequest>(new backend.LoginRequest());
const mfaCode = ref<string>("");

const isLoading = ref<boolean>(false);
const error = ref<string>("");
const loginStep = ref<number>(1);
const showPassword = ref<boolean>(false);

onMounted(() => {
  if (route.query.registered === "true") {
    toast.success("¡Cuenta creada con éxito! Ahora puedes iniciar sesión.");
  }
});

async function handleLogin() {
  isLoading.value = true;
  error.value = "";
  try {
    const requiresMFA = await authStore.login(credentials.value);
    if (requiresMFA) {
      loginStep.value = 2;
    }
  } catch (err: any) {
    error.value = err.message || "Credenciales incorrectas.";
  } finally {
    isLoading.value = false;
  }
}

async function handleVerifyMFA() {
  if (mfaCode.value.length !== 6) {
    error.value = "El código debe tener 6 dígitos.";
    return;
  }
  isLoading.value = true;
  error.value = "";
  try {
    await authStore.verifyMfaAndFinishLogin(mfaCode.value);
  } catch (err: any) {
    error.value = err.message || "Código MFA incorrecto o expirado.";
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="flex flex-col lg:flex-row h-screen">

    <!-- Left brand panel -->
    <div
      class="hidden lg:flex lg:w-2/5 bg-gradient-to-br from-slate-900 via-blue-950 to-indigo-900
             flex-col justify-between px-10 py-10 select-none overflow-hidden"
    >
      <!-- Brand header -->
      <div>
        <h1 class="text-2xl font-bold text-white tracking-tight">Droguería Luna</h1>
        <p class="text-blue-300 text-sm mt-1">Sistema integral de gestión farmacéutica</p>
      </div>

      <!-- Illustration -->
      <div class="flex-1 flex items-center justify-center py-6">
        <div class="relative w-full max-w-sm">
          <!-- Glow halo behind image -->
          <div class="absolute inset-0 rounded-3xl bg-blue-500/20 blur-2xl scale-95"></div>
          <img
            :src="loginIllustration"
            alt="Droguería Luna — inicio de sesión"
            class="relative w-full rounded-2xl shadow-2xl shadow-black/50 ring-1 ring-white/10"
          />
        </div>
      </div>

      <!-- Bottom tagline -->
      <p class="text-blue-400 text-xs text-center">
        © {{ new Date().getFullYear() }} Droguería Luna · Todos los derechos reservados
      </p>
    </div>

    <!-- Right form panel -->
    <div class="w-full lg:w-3/5 flex flex-col justify-center items-center
                px-6 sm:px-12 lg:px-16 py-10 bg-background overflow-y-auto">

      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="opacity-0 translate-x-4"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition-all duration-200 ease-in"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 -translate-x-4"
        mode="out-in"
      >
        <!-- Step 1: Credentials -->
        <form
          v-if="loginStep === 1"
          key="credentials"
          @submit.prevent="handleLogin"
          class="w-full max-w-sm"
        >
          <!-- Mobile brand header (only visible when left panel is hidden) -->
          <div class="lg:hidden text-center mb-8">
            <h1 class="text-xl font-bold text-foreground">Droguería Luna</h1>
            <p class="text-muted-foreground text-sm">Sistema de gestión farmacéutica</p>
          </div>

          <div class="mb-8">
            <h2 class="text-2xl font-bold text-foreground mb-1">Bienvenido de nuevo</h2>
            <p class="text-muted-foreground text-sm">
              Ingresa tus credenciales para acceder al sistema
            </p>
          </div>

          <div class="flex flex-col gap-5">
            <div class="grid gap-2">
              <Label for="email">Correo electrónico</Label>
              <Input
                id="email"
                v-model="credentials.Email"
                type="email"
                placeholder="usuario@droguerialuna.com"
                required
                class="h-10"
              />
            </div>

            <div class="grid gap-2">
              <Label for="password">Contraseña</Label>
              <div class="relative">
                <Input
                  id="password"
                  v-model="credentials.Contrasena"
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
              <AlertTitle>Error de acceso</AlertTitle>
              <AlertDescription>{{ error }}</AlertDescription>
            </Alert>

            <Button type="submit" class="w-full h-10" :disabled="isLoading">
              <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
              {{ isLoading ? "Ingresando..." : "Iniciar Sesión" }}
            </Button>

            <p class="text-center text-sm text-muted-foreground">
              ¿No tienes una cuenta?
              <router-link
                to="/register"
                class="font-medium text-foreground underline underline-offset-4
                       hover:text-primary transition-colors"
              >
                Regístrate
              </router-link>
            </p>
          </div>
        </form>

        <!-- Step 2: MFA -->
        <form
          v-else
          key="mfa"
          @submit.prevent="handleVerifyMFA"
          class="w-full max-w-sm"
        >
          <div class="flex flex-col items-center text-center mb-8">
            <div class="inline-flex items-center justify-center w-14 h-14
                        rounded-2xl bg-primary/10 mb-4">
              <ShieldCheck class="w-7 h-7 text-primary" />
            </div>
            <h2 class="text-2xl font-bold text-foreground mb-1">
              Verificación en dos pasos
            </h2>
            <p class="text-muted-foreground text-sm max-w-xs">
              Ingresa el código de 6 dígitos de tu aplicación de autenticación.
            </p>
          </div>

          <div class="flex flex-col gap-5">
            <div class="grid gap-2">
              <Label for="mfa-code">Código de autenticación</Label>
              <Input
                id="mfa-code"
                v-model="mfaCode"
                class="text-center text-xl tracking-[0.6em] h-12 font-mono"
                maxlength="6"
                autocomplete="one-time-code"
                placeholder="000000"
                required
              />
            </div>

            <Alert v-if="error" variant="destructive">
              <AlertCircle class="w-4 h-4" />
              <AlertTitle>Código inválido</AlertTitle>
              <AlertDescription>{{ error }}</AlertDescription>
            </Alert>

            <Button type="submit" class="w-full h-10" :disabled="isLoading">
              <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
              {{ isLoading ? "Verificando..." : "Verificar Código" }}
            </Button>

            <button
              type="button"
              @click="loginStep = 1; error = ''"
              class="text-center text-sm text-muted-foreground hover:text-foreground
                     transition-colors underline underline-offset-4"
            >
              Volver al inicio de sesión
            </button>
          </div>
        </form>
      </Transition>
    </div>

  </div>
</template>
