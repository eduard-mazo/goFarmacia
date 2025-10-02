<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AlertCircle } from "lucide-vue-next";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

import { useAuthStore } from "@/stores/auth";
import { backend } from "@/../wailsjs/go/models";

const registerPayload = ref<backend.Vendedor>(new backend.Vendedor());

const error = ref<string | null>(null);
const isLoading = ref(false);
const router = useRouter();
const authStore = useAuthStore();

const handleRegister = async () => {
  isLoading.value = true;
  error.value = "";
  try {
    await authStore.register(registerPayload.value);
    // Redirigir al login para que el usuario inicie sesión con su nueva cuenta.
    router.push({ path: "/login", query: { registered: "true" } });
  } catch (err: any) {
    error.value = err.message || "Ocurrió un error durante el registro.";
  } finally {
    isLoading.value = false;
  }
};
</script>

<template>
  <div class="flex flex-col gap-6">
    <Card class="overflow-hidden p-0">
      <CardContent class="grid p-0 md:grid-cols-2">
        <form @submit.prevent="handleRegister" class="p-6 md:p-8">
          <div class="grid gap-6 grid-cols-2">
            <div class="grid col-span-2 items-center text-center">
              <h1 class="text-2xl font-bold">Welcome back</h1>
              <p class="text-balance text-muted-foreground">
                Login to your Acme Inc account
              </p>
            </div>
            <div class="grid gap-2">
              <Label for="name">Nombres</Label>
              <Input
                id="name"
                v-model="registerPayload.Nombre"
                type="text"
                placeholder="John Doe"
                required
              />
            </div>
            <div class="grid gap-2">
              <Label for="name">Apellidos</Label>
              <Input
                id="name"
                v-model="registerPayload.Apellido"
                type="text"
                placeholder="Velasquez Suaza"
                required
              />
            </div>
            <div class="grid gap-2">
              <Label for="email">Correo Electrónico</Label>
              <Input
                id="email"
                v-model="registerPayload.Email"
                type="email"
                placeholder="m@example.com"
                required
              />
            </div>
            <div class="grid gap-2">
              <Label for="email">Cedula</Label>
              <Input
                id="email"
                v-model="registerPayload.Cedula"
                type="text"
                placeholder="1020430991"
                required
              />
            </div>
            <div class="col-span-2 grid gap-2">
              <Label for="password">Contraseña</Label>
              <Input
                id="password"
                v-model="registerPayload.Contrasena"
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
            <Button
              type="submit"
              class="col-span-2 w-full"
              :disabled="isLoading"
            >
              {{ isLoading ? "Creando cuenta..." : "Crear Cuenta" }}
            </Button>
            <div class="col-span-2 text-center text-sm">
              ¿Ya tienes una cuenta?
              <router-link to="/login" class="underline underline-offset-4">
                Inicia sesión
              </router-link>
            </div>
          </div>
        </form>
        <div class="relative hidden bg-muted md:block">
          <img
            src="@/assets/images/Register_luna.png"
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
