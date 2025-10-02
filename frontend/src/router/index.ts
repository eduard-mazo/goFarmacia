import { createRouter, createWebHashHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { jwtDecode } from "jwt-decode";

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
    meta: { public: true },
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
  history: createWebHashHistory(),
  routes,
});

router.beforeEach((to, _, next) => {
  const authStore = useAuthStore();
  const token = authStore.token;

  if (token) {
    try {
      const decodedToken: { exp: number } = jwtDecode(token);
      if (decodedToken.exp * 1000 < Date.now()) {
        authStore.logout();
        return next({ name: "Login" });
      }
    } catch (error) {
      console.error("Token inválido:", error);
      authStore.logout();
      return next({ name: "Login" });
    }
  }

  if (!authStore.isAuthenticated) {
    authStore.tryAutoLogin();
  }

  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth);
  const isPublic = to.matched.some((record) => record.meta.public);

  if (requiresAuth && !authStore.isAuthenticated) {
    next({ name: "Login" });
  } else if (isPublic && authStore.isAuthenticated) {
    next({ name: "DashboardHome" });
  } else {
    next();
  }
});

export default router;
