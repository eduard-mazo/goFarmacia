<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
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
import { useAuthStore } from '@/stores/auth';

const email = ref('');
const password = ref('');

const authStore = useAuthStore();
const router = useRouter();
const route = useRoute();

const isLoading = ref(false);
const error = ref('');
const registrationSuccess = ref(false);

onMounted(() => {
  // Mostrar mensaje de éxito si venimos del registro.
  if (route.query.registered === 'true') {
    registrationSuccess.value = true;
  }
});

const handleLogin = async () => {
  isLoading.value = true;
  error.value = '';
  try {
    await authStore.login(email.value, password.value);
    router.push('/'); 
  } catch (err: any) {
    error.value = err.message || 'Ocurrió un error al iniciar sesión.';
  } finally {
    isLoading.value = false;
  }
};
</script>

<template>
  <div class="w-full lg:grid lg:min-h-screen lg:grid-cols-2">
    <div class="flex items-center justify-center py-12">
      <div class="mx-auto grid w-[350px] gap-6">
        <Card>
          <CardHeader>
            <CardTitle class="text-3xl"> Iniciar Sesión </CardTitle>
            <CardDescription>
              Ingresa tu correo electrónico para acceder a tu cuenta.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form @submit.prevent="handleLogin" class="grid gap-4">
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
                <div class="flex items-center">
                  <Label for="password">Contraseña</Label>
                </div>
                <Input
                  id="password"
                  v-model="password"
                  type="password"
                  required
                />
              </div>
              <p v-if="error" class="text-sm text-red-500">{{ error }}</p>
              <Button type="submit" class="w-full" :disabled="isLoading">
                {{ isLoading ? "Ingresando..." : "Iniciar Sesión" }}
              </Button>
            </form>
            <div class="mt-4 text-center text-sm">
              ¿No tienes una cuenta?
              <router-link to="/register" class="underline">
                Regístrate
              </router-link>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
    <div class="hidden bg-muted lg:block"></div>
  </div>
</template>
