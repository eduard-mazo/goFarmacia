import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const routes = [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { public: true },
  },
  {
    path: "/Register",
    name: "Register",
    component: () => import("@/views/Register.vue"),
  },
  {
    path: "/dashboard",
    component: () => import("@/layouts/DashboardLayout.vue"),
    meta: { requiresAuth: true },
    children: [
      {
        path: "",
        name: "DashboardHome",
        component: () => import("@/views/Dashboard/Home.vue"),
      },
      {
        path: "pos",
        name: "VentasPOS",
        component: () => import("@/views/Dashboard/Facturacion/POS.vue"),
      },
      {
        path: "vendedores",
        name: "Vendedores",
        component: () => import("@/views/Dashboard/Personas/Vendedores.vue"),
      },
      {
        path: "productos",
        name: "Productos",
        component: () => import("@/views/Dashboard/Catalogo/Productos.vue"),
      },
      {
        path: "clientes",
        name: "Clientes",
        component: () => import("@/views/Dashboard/Personas/Clientes.vue"),
      },
      {
        path: "facturas",
        name: "Facturas",
        component: () => import("@/views/Dashboard/Facturacion/Facturas.vue"),
      },
      {
        path: "controlStock",
        name: "ControlStock",
        component: () =>
          import("@/views/Dashboard/Inventario/ControlStock.vue"),
      },
    ],
  },
  {
    path: "/:pathMatch(.*)*",
    redirect: "/dashboard",
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, _, next) => {
  const authStore = useAuthStore();

  if (!authStore.isAuthenticated) {
    authStore.tryAutoLogin();
  }

  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth);

  if (requiresAuth && !authStore.isAuthenticated) {
    next({ name: "Login" });
  } else if (
    (to.name === "Login" || to.name === "Register") &&
    authStore.isAuthenticated
  ) {
    next({ name: "DashboardHome" });
  } else {
    next();
  }
});

export default router;
