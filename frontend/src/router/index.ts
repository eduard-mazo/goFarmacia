import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "../stores/auth";

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
        component: () => import("@/views/Dashboard/DashboardHome.vue"),
      },
      {
        path: "pos",
        name: "VentasPOS",
        component: () => import("../views/Dashboard/VentaPOS.vue"),
      },
      {
        path: "vendedores",
        name: "Vendedores",
        component: () => import("../views/Dashboard/Vendedores.vue"),
      },
      {
        path: "productos",
        name: "Productos",
        component: () => import("../views/Dashboard/Productos.vue"),
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

// Guardia de Navegación (Auth Guard)
router.beforeEach((to, _, next) => {
  const authStore = useAuthStore();

  // Intentar cargar la sesión desde localStorage en cada navegación
  // Esto asegura que el estado se mantenga al recargar la página
  if (!authStore.isAuthenticated) {
    authStore.tryAutoLogin();
  }

  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth);

  if (requiresAuth && !authStore.isAuthenticated) {
    // Si la ruta requiere autenticación y el usuario no está logueado,
    // redirigir a la página de login.
    next({ name: "Login" });
  } else if (
    (to.name === "Login" || to.name === "Register") &&
    authStore.isAuthenticated
  ) {
    // Si el usuario ya está logueado, no debería poder ver las páginas de login/registro.
    // Redirigirlo a la página de inicio.
    next({ name: "DashboardHome" });
  } else {
    // En cualquier otro caso, permitir la navegación.
    next();
  }
});

export default router;
