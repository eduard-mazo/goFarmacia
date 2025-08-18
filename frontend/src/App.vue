<template>
  <div class="flex flex-col h-screen">
    <header v-if="authStore.isAuthenticated" class="bg-white shadow-md">
      <nav
        class="container mx-auto px-6 py-3 flex justify-between items-center"
      >
        <div class="text-xl font-bold text-gray-800">goFarmacia</div>
        <div class="flex space-x-4">
          <RouterLink to="/" class="text-gray-600 hover:text-blue-500 transition">POS</RouterLink>
          <RouterLink to="/facturas" class="text-gray-600 hover:text-blue-500 transition">Facturas</RouterLink>
          <RouterLink to="/productos" class="text-gray-600 hover:text-blue-500 transition">Productos</RouterLink>
          <RouterLink to="/clientes" class="text-gray-600 hover:text-blue-500 transition">Clientes</RouterLink>
          <RouterLink to="/vendedores" class="text-gray-600 hover:text-blue-500 transition">Vendedores</RouterLink>
        </div>
        <div class="flex items-center space-x-4">
          <span class="text-gray-600"
            >Hola, {{ authStore.vendedorNombre }}</span
          >
          <button
            @click="authStore.logout()"
            class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition"
          >
            Salir
          </button>
        </div>
      </nav>
    </header>

    <main class="flex-grow p-6 overflow-auto">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from "./stores/auth";
const authStore = useAuthStore();
</script>

<style>
@reference "./style.css";
.nav-link {
  @apply text-gray-600 hover:text-blue-500 transition;
}
.router-link-active {
  @apply text-blue-600 font-semibold;
}
</style>
