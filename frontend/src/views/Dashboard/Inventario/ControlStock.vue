<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-100">
    <div class="max-w-md w-full bg-white p-8 rounded-lg shadow-lg">
      <h2 class="text-2xl font-bold text-center text-gray-800 mb-6">
        Iniciar Sesión
      </h2>
      <form @submit.prevent="handleLogin">
        <div class="mb-4">
          <label for="cedula" class="block text-gray-700 text-sm font-bold mb-2"
            >Cédula</label
          >
          <input
            v-model="credenciales.Cedula"
            type="text"
            id="cedula"
            class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
            required
          />
        </div>
        <div class="mb-6">
          <label
            for="password"
            class="block text-gray-700 text-sm font-bold mb-2"
            >Contraseña</label
          >
          <input
            v-model="credenciales.Contrasena"
            type="password"
            id="password"
            class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 mb-3 leading-tight focus:outline-none focus:shadow-outline"
            required
          />
        </div>
        <p v-if="errorMsg" class="text-red-500 text-xs italic mb-4">
          {{ errorMsg }}
        </p>
        <div class="flex items-center justify-between">
          <button
            type="submit"
            class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline w-full"
          >
            Ingresar
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useAuthStore } from "../../../stores/auth";
import { backend } from "../../../../wailsjs/go/models";
import { RegistrarVendedor } from "../../../../wailsjs/go/backend/Db";

const vendedor = ref(new backend.Vendedor());
const authStore = useAuthStore();
const credenciales = ref(new backend.LoginRequest());
const errorMsg = ref("");
onMounted(() => {
  async () => {
    try {
      let resultado: string;
      resultado = await RegistrarVendedor(vendedor.value);
      alert(resultado);
    } catch (error) {
      alert(`Error al guardar vendedor: ${error}`);
    }
  };
});
const handleLogin = async () => {
  errorMsg.value = "";
  try {
    await authStore.login(credenciales.value);
  } catch (error) {
    errorMsg.value = String(error) || "Credenciales inválidas.";
  }
};
</script>
