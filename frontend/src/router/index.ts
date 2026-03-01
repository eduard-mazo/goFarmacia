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
        path: "facturas",
        name: "Facturas",
        component: () => import("@/views/Dashboard/Facturacion/Facturas.vue"),
      },
      // ERP routes — admin only
      {
        path: "erp",
        name: "DashboardERP",
        component: () => import("@/views/Dashboard/DashboardERP.vue"),
        meta: { role: "admin" },
      },
      {
        path: "vendedores",
        name: "Vendedores",
        component: () =>
          import("@/views/Dashboard/Personas/Vendedores.vue"),
        meta: { role: "admin" },
      },
      {
        path: "productos",
        name: "Productos",
        component: () =>
          import("@/views/Dashboard/Catalogo/Productos.vue"),
        meta: { role: "admin" },
      },
      {
        path: "clientes",
        name: "Clientes",
        component: () =>
          import("@/views/Dashboard/Personas/Clientes.vue"),
        meta: { role: "admin" },
      },
      {
        path: "proveedores",
        name: "Proveedores",
        component: () =>
          import("@/views/Dashboard/Personas/Proveedores.vue"),
        meta: { role: "admin" },
      },
      {
        path: "controlStock",
        name: "ControlStock",
        component: () =>
          import("@/views/Dashboard/Inventario/ControlStock.vue"),
        meta: { role: "admin" },
      },
      {
        path: "reportes/ventas",
        name: "ReporteVentas",
        component: () =>
          import("@/views/Dashboard/Reportes/ReporteVentas.vue"),
        meta: { role: "admin" },
      },
      {
        path: "reportes/inventario",
        name: "ReporteInventario",
        component: () =>
          import("@/views/Dashboard/Reportes/ReporteInventario.vue"),
        meta: { role: "admin" },
      },
      {
        path: "configuracion",
        name: "Configuracion",
        component: () =>
          import("@/views/Dashboard/Configuracion/General.vue"),
        meta: { role: "admin" },
      },
      {
        path: "compras/facturas",
        name: "FacturasElectronicas",
        component: () =>
          import("@/views/Dashboard/Compras/FacturasElectronicas.vue"),
        meta: { role: "admin" },
      },
      {
        path: "tesoreria/bancolombia",
        name: "TransferenciasBancolombia",
        component: () =>
          import("@/views/Dashboard/Tesoreria/TransferenciasBancolombia.vue"),
        meta: { role: "admin" },
      },
      {
        path: "basedatos",
        name: "AdminBD",
        component: () =>
          import("@/views/Dashboard/BaseDatos/AdminBD.vue"),
        meta: { role: "admin" },
      },
      {
        path: "perfil",
        name: "MiPerfil",
        component: () =>
          import("@/views/Dashboard/Perfil/MiPerfil.vue"),
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
    // Role-based guard
    const requiredRole = to.meta.role as string | undefined;
    if (requiredRole && authStore.currentUser?.Role !== requiredRole) {
      next({ name: "DashboardHome" });
    } else {
      next();
    }
  }
});

export default router;
