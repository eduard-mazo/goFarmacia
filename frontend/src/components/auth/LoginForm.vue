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
import { AlertCircle, Loader2 } from "lucide-vue-next";
import { Card, CardContent } from "@/components/ui/card";

const authStore = useAuthStore();
const route = useRoute();

const credentials = ref<backend.LoginRequest>(new backend.LoginRequest());
const mfaCode = ref<string>("");

const isLoading = ref<boolean>(false);
const error = ref<string>("");
const loginStep = ref<number>(1); // 1 para credenciales, 2 para MFA

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
  <div class="flex flex-col gap-6">
    <Card class="overflow-hidden p-0">
      <CardContent class="grid p-0 md:grid-cols-2">
        <form
          v-if="loginStep === 1"
          @submit.prevent="handleLogin"
          class="p-6 md:p-8"
        >
          <div class="flex flex-col items-center text-center mb-7">
            <h1 class="text-2xl font-bold">¡Bienvenido!</h1>
            <p class="text-balance text-muted-foreground">
              Accede a tu cuenta de Droguería Luna
            </p>
          </div>
          <div class="flex flex-col gap-6">
            <div class="grid gap-2">
              <Label for="email">Correo</Label>
              <Input
                id="email"
                v-model="credentials.Email"
                type="email"
                placeholder="m@example.com"
                required
              />
            </div>
            <div class="grid gap-2">
              <div class="flex items-center">
                <Label for="password">Contraseña</Label>
                <a
                  href="#"
                  class="ml-auto text-sm underline-offset-2 hover:underline"
                >
                  Olvidaste tu contraseña?
                </a>
              </div>
              <Input
                id="password"
                v-model="credentials.Contrasena"
                type="password"
                required
              />
            </div>
            <Alert v-if="error" variant="destructive">
              <AlertCircle class="w-4 h-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>
                {{ error }}
              </AlertDescription>
            </Alert>
            <Button type="submit" class="w-full" :disabled="isLoading">
              <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
              {{ isLoading ? "Ingresando..." : "Iniciar Sesión" }}
            </Button>
            <div class="text-center text-sm">
              No tienes una cuenta?
              <router-link to="/register" class="underline underline-offset-4">
                Regístrate
              </router-link>
            </div>
          </div>
        </form>
        <form
          v-if="loginStep === 2"
          @submit.prevent="handleVerifyMFA"
          class="p-4 md:p-6"
        >
          <div class="flex flex-col items-center text-center mb-4">
            <h1 class="text-2xl font-bold">Verificación Requerida</h1>
            <p class="text-balance text-muted-foreground">
              Ingresa el código de 6 dígitos de tu app de autenticación.
            </p>
          </div>
          <Label for="mfa-code" class="mb-4">Código de Autenticación</Label>
          <Input
            id="mfa-code"
            v-model="mfaCode"
            class="text-center text-lg tracking-[0.5em] mb-2"
            maxlength="6"
            autocomplete="one-time-code"
            placeholder="123456"
            required
          />
          <Button type="submit" class="w-full mt-2" :disabled="isLoading">
            <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
            {{ isLoading ? "Verificando..." : "Verificar Código" }}
          </Button>
        </form>
        <div class="relative hidden bg-muted md:block">
          <img
            src="@/assets/images/Login_luna.png"
            alt="Image"
            class="absolute inset-0 h-full w-full object-cover dark:brightness-[0.2] dark:grayscale"
          />
        </div>
        <Alert v-if="error" variant="destructive">
          <AlertCircle class="w-4 h-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>
            {{ error }}
          </AlertDescription>
        </Alert>
      </CardContent>
    </Card>
    <div
      class="text-balance text-center text-xs text-muted-foreground [&_a]:underline [&_a]:underline-offset-4 hover:[&_a]:text-primary"
    >
      By clicking continue, you agree to our
      <a href="#">Terms of Service</a> and <a href="#">Privacy Policy</a>.
    </div>
  </div>
</template>
