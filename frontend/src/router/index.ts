import { createRouter, createWebHashHistory } from "vue-router";
import { useAuthStore } from "../stores/auth";

// Importa las vistas
import Login from "../views/Login.vue";
import VentaPOS from "../views/VentaPOS.vue";
import Productos from "../views/Productos.vue";
import Clientes from "../views/Clientes.vue";
import Vendedores from "../views/Vendedores.vue";
import Facturas from "../views/Facturas.vue";
import Inventario from "../views/Inventario.vue";

const routes = [
  { path: "/login", name: "Login", component: Login, meta: { public: true } },
  { path: "/", name: "POS", component: VentaPOS },
  { path: "/productos", name: "Productos", component: Productos },
  { path: "/clientes", name: "Clientes", component: Clientes },
  { path: "/vendedores", name: "Vendedores", component: Vendedores },
  { path: "/facturas", name: "Facturas", component: Facturas },
  { path: "/inventario", name: "Inventario", component: Inventario },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

// Guardia de navegación
router.beforeEach((to, _, next) => {
  const authStore = useAuthStore();
  const isPublic = to.matched.some((record) => record.meta.public);

  if (!isPublic && !authStore.isAuthenticated) {
    next({ name: "Login" });
  } else {
    next();
  }
});

export default router;
