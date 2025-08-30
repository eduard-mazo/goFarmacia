import { createRouter, createWebHistory } from "vue-router";
import { useAuth } from "@/services/auth";

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
      // ... aquí puedes agregar más rutas para el dashboard
    ],
  },
  // Redirección por defecto
  {
    path: "/:pathMatch(.*)*",
    redirect: "/dashboard",
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

// Guarda de navegación global
router.beforeEach((to, from, next) => {
  const { isAuthenticated } = useAuth();
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth);

  if (requiresAuth && !isAuthenticated.value) {
    // Si la ruta requiere autenticación y el usuario no está logueado, redirige a login
    next({ name: "Login" });
  } else if (
    (to.name === "Login" || to.name === "Register") &&
    isAuthenticated.value
  ) {
    // Si el usuario ya está logueado, no puede acceder a login/register
    next({ name: "DashboardHome" });
  } else {
    next();
  }
});

export default router;
