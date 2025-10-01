<script setup>
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { LoginVendedor, VerificarLoginMFA } from "@/../wailsjs/go/backend/Db";
import { useAuthStore } from "@/stores/auth";
import { backend } from "@/../wailsjs/go/models";

const router = useRouter();
const authStore = useAuthStore();

const isLoading = ref(false);
const error = ref("");

const mfaRequired = ref(false);
const tempToken = ref("");
const mfaCode = ref("");

const vendedorInfo = ref(null);

const credentials = ref({
  Email: "",
  Contrasena: "",
});

const handleLogin = async () => {
  isLoading.value = true;
  error.value = "";
  try {
    const response = await LoginVendedor(credentials.value);

    vendedorInfo.value = response.vendedor;

    if (response.mfa_required) {
      mfaRequired.value = true;
      tempToken.value = response.token;
    } else {
      authStore.setAuthenticated(vendedorInfo.value, response.token);
      router.push("/");
    }
  } catch (err) {
    error.value = err;
    vendedorInfo.value = null;
  } finally {
    isLoading.value = false;
  }
};

const handleVerifyMFA = async () => {
  if (mfaCode.value.length !== 6) {
    error.value = "El código debe tener 6 dígitos.";
    return;
  }
  isLoading.value = true;
  error.value = "";
  try {
    const finalToken = await VerificarLoginMFA(tempToken.value, mfaCode.value);

    if (vendedorInfo.value) {
      authStore.setAuthenticated(vendedorInfo.value, finalToken);
      router.push("/");
    } else {
      throw new Error(
        "No se pudo recuperar la información del usuario. Intenta iniciar sesión de nuevo."
      );
    }
  } catch (err) {
    error.value = err;
  } finally {
    isLoading.value = false;
  }
};
</script>

<template>
  <div class="flex items-center justify-center min-h-screen bg-background">
    <Card class="w-full max-w-md">
      <CardHeader>
        <CardTitle class="text-2xl font-bold text-center">
          {{ !mfaRequired ? "Iniciar Sesión" : "Verificación de Dos Pasos" }}
        </CardTitle>
        <CardDescription class="text-center">
          {{
            !mfaRequired
              ? "Ingresa tus credenciales para acceder al sistema."
              : "Ingresa el código de tu aplicación de autenticación."
          }}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Alert v-if="error" variant="destructive" class="mb-4">
          <AlertTitle>Error de Autenticación</AlertTitle>
          <AlertDescription>
            {{ error }}
          </AlertDescription>
        </Alert>

        <form
          v-if="!mfaRequired"
          @submit.prevent="handleLogin"
          class="space-y-4"
        >
          <div class="space-y-2">
            <Label for="email">Email</Label>
            <Input
              id="email"
              v-model="credentials.Email"
              type="email"
              placeholder="tu@email.com"
              required
            />
          </div>
          <div class="space-y-2">
            <Label for="password">Contraseña</Label>
            <Input
              id="password"
              v-model="credentials.Contrasena"
              type="password"
              required
            />
          </div>
          <Button type="submit" class="w-full" :disabled="isLoading">
            {{ isLoading ? "Ingresando..." : "Ingresar" }}
          </Button>
        </form>

        <form
          v-if="mfaRequired"
          @submit.prevent="handleVerifyMFA"
          class="space-y-4"
        >
          <div class="space-y-2">
            <Label for="mfa-code">Código de 6 dígitos</Label>
            <Input
              id="mfa-code"
              v-model="mfaCode"
              type="text"
              placeholder="123456"
              maxlength="6"
              required
              autocomplete="one-time-code"
              class="text-center text-lg tracking-[0.5em]"
            />
          </div>
          <Button type="submit" class="w-full" :disabled="isLoading">
            {{ isLoading ? "Verificando..." : "Verificar Código" }}
          </Button>
        </form>
      </CardContent>
      <CardFooter class="flex justify-center">
        <p class="text-sm text-muted-foreground">
          ¿No tienes cuenta?
          <router-link to="/register" class="underline">Regístrate</router-link>
        </p>
      </CardFooter>
    </Card>
  </div>
</template>
