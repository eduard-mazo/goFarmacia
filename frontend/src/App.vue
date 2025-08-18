<template>
  <div class="sidebar">
    <h1>goFarmacia</h1>
    <nav>
      <button @click="vistaActual = 'POS'">Punto de Venta</button>
      <button @click="vistaActual = 'Productos'">Gestionar Productos</button>
      <button @click="vistaActual = 'Clientes'">Gestionar Clientes</button>
      <button @click="vistaActual = 'Vendedores'">Gestionar Vendedores</button>
      <button @click="vistaActual = 'Facturas'">Historial de Facturas</button>
    </nav>
  </div>
  <main>
    <VentaPOS v-if="vistaActual === 'POS'" />
    <ProductoManager v-if="vistaActual === 'Productos'" />
    <ClienteManager v-if="vistaActual === 'Clientes'" />
    <VendedorManager v-if="vistaActual === 'Vendedores'" />
  </main>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import VentaPOS from "./components/VentaPOS.vue";
import ProductoManager from "./components/ProductoManager.vue";
import ClienteManager from "./components/ClienteManager.vue";
import VendedorManager from "./components/VendedorManager.vue";

const vistaActual = ref("POS"); // Vista por defecto
</script>

<style scoped>
.sidebar {
  width: 250px;
  background-color: #2c3e50;
  color: white;
  padding: 1.5rem;
  height: 100vh;
  display: flex;
  flex-direction: column;
}

.sidebar h1 {
  color: #42b983;
  text-align: center;
  margin-bottom: 2rem;
}

nav {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

nav button {
  background-color: #34495e;
  color: white;
  border: none;
  text-align: left;
  padding: 12px 15px;
  border-radius: 5px;
  cursor: pointer;
  font-size: 1rem;
}

nav button:hover {
  background-color: #42b983;
}
</style>
