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
import { AlertCircle, Loader2, ArrowLeft } from "lucide-vue-next";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

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
  <Card class="w-full max-w-sm">
    <CardHeader class="text-center">
      <CardTitle class="text-2xl">
        {{
          loginStep === 1 ? "¡Bienvenido de Nuevo!" : "Verificación Requerida"
        }}
      </CardTitle>
      <CardDescription>
        <span v-if="loginStep === 1">Ingresa a tu cuenta de Luna POS.</span>
        <span v-else
          >Ingresa el código de 6 dígitos de tu app de autenticación.</span
        >
      </CardDescription>
    </CardHeader>

    <CardContent>
      <form
        v-if="loginStep === 1"
        @submit.prevent="handleLogin"
        class="grid gap-4"
      >
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
          </div>
          <Input
            id="password"
            v-model="credentials.Contrasena"
            type="password"
            required
          />
        </div>
      </form>

      <form
        v-if="loginStep === 2"
        @submit.prevent="handleVerifyMFA"
        class="grid gap-2"
      >
        <Label for="mfa-code">Código de Autenticación</Label>
        <Input
          id="mfa-code"
          v-model="mfaCode"
          class="text-center text-lg tracking-[0.5em]"
          maxlength="6"
          autocomplete="one-time-code"
          required
        />
      </form>

      <Alert v-if="error" variant="destructive" class="mt-4">
        <AlertCircle class="w-4 h-4" />
        <AlertTitle>Error de Autenticación</AlertTitle>
        <AlertDescription>{{ error }}</AlertDescription>
      </Alert>
    </CardContent>

    <CardFooter class="flex flex-col gap-4">
      <Button
        v-if="loginStep === 1"
        type="submit"
        @click="handleLogin"
        class="w-full"
        :disabled="isLoading"
      >
        <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
        {{ isLoading ? "Ingresando..." : "Iniciar Sesión" }}
      </Button>
      <Button
        v-if="loginStep === 2"
        type="submit"
        @click="handleVerifyMFA"
        class="w-full"
        :disabled="isLoading"
      >
        <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
        {{ isLoading ? "Verificando..." : "Verificar Código" }}
      </Button>

      <div class="text-center text-sm">
        <span v-if="loginStep === 1">
          ¿No tienes una cuenta?
          <router-link to="/register" class="underline underline-offset-4"
            >Regístrate</router-link
          >
        </span>
        <Button
          v-if="loginStep === 2"
          variant="link"
          size="sm"
          @click="
            loginStep = 1;
            error = '';
          "
        >
          <ArrowLeft class="w-4 h-4 mr-1" /> Volver
        </Button>
      </div>
    </CardFooter>
  </Card>
</template>
