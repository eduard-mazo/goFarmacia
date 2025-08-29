<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/services/auth";

const name = ref("");
const email = ref("");
const password = ref("");
const error = ref<string | null>(null);
const isLoading = ref(false);

const router = useRouter();
const { register } = useAuth();

async function handleRegister() {
  error.value = null;
  isLoading.value = true;
  try {
    await register(name.value, email.value, password.value);
    router.push("/dashboard");
  } catch (err: any) {
    error.value = err.message || "Ocurrió un error inesperado.";
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="w-full lg:grid lg:min-h-screen lg:grid-cols-2">
    <div class="hidden bg-muted lg:block"></div>
    <div class="flex items-center justify-center py-12">
      <div class="mx-auto grid w-[350px] gap-6">
        <Card>
          <CardHeader>
            <CardTitle class="text-3xl"> Crear Cuenta </CardTitle>
            <CardDescription>
              Ingresa tus datos para crear una nueva cuenta.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form @submit.prevent="handleRegister" class="grid gap-4">
              <div class="grid gap-2">
                <Label for="name">Nombre Completo</Label>
                <Input
                  id="name"
                  v-model="name"
                  type="text"
                  placeholder="John Doe"
                  required
                />
              </div>
              <div class="grid gap-2">
                <Label for="email">Correo Electrónico</Label>
                <Input
                  id="email"
                  v-model="email"
                  type="email"
                  placeholder="m@example.com"
                  required
                />
              </div>
              <div class="grid gap-2">
                <Label for="password">Contraseña</Label>
                <Input
                  id="password"
                  v-model="password"
                  type="password"
                  required
                />
              </div>
              <p v-if="error" class="text-sm text-red-500">{{ error }}</p>
              <Button type="submit" class="w-full" :disabled="isLoading">
                {{ isLoading ? "Creando cuenta..." : "Crear Cuenta" }}
              </Button>
            </form>
            <div class="mt-4 text-center text-sm">
              ¿Ya tienes una cuenta?
              <router-link to="/login" class="underline">
                Inicia sesión
              </router-link>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>
