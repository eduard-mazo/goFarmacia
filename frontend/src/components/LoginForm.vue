<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AlertCircle } from "lucide-vue-next"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

import { useAuthStore } from "@/stores/auth";

const email = ref("");
const password = ref("");

const authStore = useAuthStore();
const router = useRouter();
const route = useRoute();

const isLoading = ref(false);
const error = ref("");
const registrationSuccess = ref(false);

onMounted(() => {
  // Mostrar mensaje de éxito si venimos del registro.
  if (route.query.registered === "true") {
    registrationSuccess.value = true;
  }
});

const handleLogin = async () => {
  isLoading.value = true;
  error.value = "";
  try {
    await authStore.login(email.value, password.value);
    router.push("/");
  } catch (err: any) {
    error.value = err.message || "Ocurrió un error al iniciar sesión.";
  } finally {
    isLoading.value = false;
  }
};
</script>

<template>
  <div class="flex flex-col gap-6">
    <Card class="overflow-hidden p-0">
      <CardContent class="grid p-0 md:grid-cols-2">
        <form @submit.prevent="handleLogin" class="p-6 md:p-8">
          <div class="flex flex-col gap-6">
            <div class="flex flex-col items-center text-center">
              <h1 class="text-2xl font-bold">¡Bienvenido!</h1>
              <p class="text-balance text-muted-foreground">
                Accede a tu cuenta de Droguería Luna
              </p>
            </div>
            <div class="grid gap-2">
              <Label for="email">Correo</Label>
              <Input
                id="email"
                v-model="email"
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
                type="password"
                v-model="password"
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
        <div class="relative hidden bg-muted md:block">
          <img
            src="../assets/images/Login_luna.png"
            alt="Image"
            class="absolute inset-0 h-full w-full object-cover dark:brightness-[0.2] dark:grayscale"
          />
        </div>
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
