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
import { backend } from '../../wailsjs/go/models';
import { useAuthStore } from '../stores/auth';

const form = ref<backend.Vendedor>({
  Nombre: '',
  Apellido: '',
  Cedula: '',
  Email: '',
  Contrasena: '',
} as backend.Vendedor);

const error = ref<string | null>(null);
const isLoading = ref(false);
const router = useRouter();
const authStore = useAuthStore();

const handleRegister = async () => {
  isLoading.value = true;
  error.value = '';
  try {
    await authStore.register(form.value);
    // Redirigir al login para que el usuario inicie sesión con su nueva cuenta.
    router.push({ path: '/login', query: { registered: 'true' } });
  } catch (err: any) {
    error.value = err.message || 'Ocurrió un error durante el registro.';
  } finally {
    isLoading.value = false;
  }
};

</script>

<template>
  <div class="w-full lg:grid lg:min-h-screen lg:grid-cols-2">
    <div class="hidden bg-muted lg:block"></div>
    <div class="flex items-center justify-center py-12">
      <div class="mx-auto grid w-[600px] gap-6">
        <Card>
          <CardHeader>
            <CardTitle class="text-3xl"> Crear Cuenta </CardTitle>
            <CardDescription>
              Ingresa tus datos para crear una nueva cuenta.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form @submit.prevent="handleRegister" class="grid grid-cols-2 gap-4">
              <div class="grid gap-2">
                <Label for="name">Nombres</Label>
                <Input id="name" v-model="form.Nombre" type="text" placeholder="John Doe" required />
              </div>
              <div class="grid gap-2">
                <Label for="name">Apellidos</Label>
                <Input id="name" v-model="form.Apellido" type="text" placeholder="Velasquez Suaza" required />
              </div>
              <div class="grid gap-2">
                <Label for="email">Correo Electrónico</Label>
                <Input id="email" v-model="form.Email" type="email" placeholder="m@example.com" required />
              </div>
              <div class="grid gap-2">
                <Label for="email">Cedula</Label>
                <Input id="email" v-model="form.Cedula" type="text" placeholder="1020430991" required />
              </div>
              <div class="col-span-2 grid gap-2">
                <Label for="password">Contraseña</Label>
                <Input id="password" v-model="form.Contrasena" type="password" required />
              </div>
              <p v-if="error" class="text-sm text-red-500">{{ error }}</p>
              <Button type="submit" class="col-span-2 w-full" :disabled="isLoading">
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
